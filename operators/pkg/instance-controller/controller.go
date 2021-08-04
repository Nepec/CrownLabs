/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package instance_controller groups the functionalities related to the Instance controller.
package instance_controller

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	virtv1 "kubevirt.io/client-go/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	clv1alpha1 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha1"
	clv1alpha2 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha2"
	clctx "github.com/netgroup-polito/CrownLabs/operators/pkg/context"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/forge"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/utils"
)

// ContainerEnvOpts contains images name and tag for container environment.
type ContainerEnvOpts struct {
	ImagesTag         string
	VncImg            string
	WebsockifyImg     string
	NovncImg          string
	FileBrowserImg    string
	FileBrowserImgTag string
}

// InstanceReconciler reconciles a Instance object.
type InstanceReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	EventsRecorder     record.EventRecorder
	NamespaceWhitelist metav1.LabelSelector
	WebsiteBaseURL     string
	NextcloudBaseURL   string
	WebdavSecretName   string
	InstancesAuthURL   string
	Concurrency        int
	ContainerEnvOpts   ContainerEnvOpts

	// This function, if configured, is deferred at the beginning of the Reconcile.
	// Specifically, it is meant to be set to GinkgoRecover during the tests,
	// in order to lead to a controlled failure in case the Reconcile panics.
	ReconcileDeferHook func()
}

// Reconcile reconciles the state of an Instance resource.
func (r *InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	if r.ReconcileDeferHook != nil {
		defer r.ReconcileDeferHook()
	}

	log := ctrl.LoggerFrom(ctx, "instance", req.NamespacedName)

	// Get the instance object.
	var instance clv1alpha2.Instance
	if err = r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !kerrors.IsNotFound(err) {
			log.Error(err, "failed retrieving instance")
		}
		// Reconcile was triggered by a delete request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check the selector label, in order to know whether to perform or not reconciliation.
	if proceed, err := utils.CheckSelectorLabel(ctrl.LoggerInto(ctx, log), r.Client, instance.GetNamespace(), r.NamespaceWhitelist.MatchLabels); !proceed {
		// If there was an error while checking, show the error and try again.
		if err != nil {
			log.Error(err, "failed checking selector labels")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Add the retrieved instance as part of the context.
	ctx, _ = clctx.InstanceInto(ctx, &instance)

	// Retrieve the template associated with the current instance.
	templateName := types.NamespacedName{
		Namespace: instance.Spec.Template.Namespace,
		Name:      instance.Spec.Template.Name,
	}
	var template clv1alpha2.Template
	if err := r.Get(ctx, templateName, &template); err != nil {
		log.Error(err, "failed retrieving the instance template", "template", templateName)
		r.EventsRecorder.Eventf(&instance, v1.EventTypeWarning, EvTmplNotFound, EvTmplNotFoundMsg, template.Namespace, template.Name)
		return ctrl.Result{}, err
	}
	ctx, log = clctx.TemplateInto(ctx, &template)
	log.Info("successfully retrieved the instance template")

	// Retrieve the tenant associated with the current instance.
	tenantName := types.NamespacedName{Name: instance.Spec.Tenant.Name}
	var tenant clv1alpha1.Tenant
	if err := r.Get(ctx, tenantName, &tenant); err != nil {
		log.Error(err, "failed retrieving the instance tenant", "tenant", tenantName)
		r.EventsRecorder.Eventf(&instance, v1.EventTypeWarning, EvTntNotFound, EvTntNotFoundMsg, template.Name)
		return ctrl.Result{}, err
	}
	ctx, log = clctx.TenantInto(ctx, &tenant)
	log.Info("successfully retrieved the instance tenant")

	// Patch the instance labels to allow for easier categorization.
	labels, updated := forge.InstanceLabels(instance.GetLabels(), &template)
	if updated {
		original := instance.DeepCopy()
		instance.SetLabels(labels)
		if err := r.Patch(ctx, &instance, client.MergeFrom(original)); err != nil {
			log.Error(err, "failed to update the instance labels")
			return ctrl.Result{}, err
		}
		log.Info("instance labels correctly configured")
	}

	// Defer the function to patch the instance status depending on the modifications
	// performed while enforcing the desired environments.
	defer func(original, updated *clv1alpha2.Instance) {
		if !reflect.DeepEqual(original.Status, updated.Status) {
			if err2 := r.Status().Patch(ctx, updated, client.MergeFrom(original)); err2 != nil {
				log.Error(err2, "failed to update the instance status")
				err = err2
			} else {
				log.Info("instance status correctly updated")
			}
		}
	}(instance.DeepCopy(), &instance)

	// Iterate over and enforce the instance environments.
	if err := r.enforceEnvironments(ctx); err != nil {
		log.Error(err, "failed to enforce instance environments")

		// Do not set the CreationLoopBackOff phase in case of conflicts, to prevent transients.
		if !kerrors.IsConflict(err) {
			instance.Status.Phase = clv1alpha2.EnvironmentPhaseCreationLoopBackoff
		}

		return ctrl.Result{}, err
	}
	log.Info("instance environments correctly enforced")

	return ctrl.Result{}, nil
}

func (r *InstanceReconciler) enforceEnvironments(ctx context.Context) error {
	instance := clctx.InstanceFrom(ctx)
	template := clctx.TemplateFrom(ctx)

	for i := range template.Spec.EnvironmentList {
		environment := &template.Spec.EnvironmentList[i]
		ctx, log := clctx.EnvironmentInto(ctx, environment)

		// Currently, only instances composed of a single environment are supported.
		// Nonetheless, we return nil in the end, since it is useless to retry later.
		if i >= 1 {
			err := fmt.Errorf("instances composed of multiple environments are currently not supported")
			log.Error(err, "failed to process environment")
			return nil
		}

		switch template.Spec.EnvironmentList[i].EnvironmentType {
		case clv1alpha2.ClassVM:
			if err := r.EnforceVMEnvironment(ctx); err != nil {
				r.EventsRecorder.Eventf(instance, v1.EventTypeWarning, EvEnvironmentErr, EvEnvironmentErrMsg, environment.Name)
				return err
			}
		case clv1alpha2.ClassContainer:
			if err := r.EnforceContainerEnvironment(ctx); err != nil {
				r.EventsRecorder.Eventf(instance, v1.EventTypeWarning, EvEnvironmentErr, EvEnvironmentErrMsg, environment.Name)
				return err
			}
		}
	}
	return nil
}

// SetupWithManager registers a new controller for Instance resources.
func (r *InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetLogger().Info("setup manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&clv1alpha2.Instance{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.Deployment{}).
		Owns(&virtv1.VirtualMachine{}).
		Owns(&virtv1.VirtualMachineInstance{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.Concurrency,
		}).
		Complete(r)
}
