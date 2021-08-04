package instance_controller_test

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	virtv1 "kubevirt.io/client-go/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clv1alpha1 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha1"
	clv1alpha2 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha2"
	clctx "github.com/netgroup-polito/CrownLabs/operators/pkg/context"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/forge"
	instance_controller "github.com/netgroup-polito/CrownLabs/operators/pkg/instance-controller"
)

var _ = Describe("Generation of the virtual machine and virtual machine instances", func() {
	var (
		ctx           context.Context
		clientBuilder fake.ClientBuilder
		reconciler    instance_controller.InstanceReconciler

		instance    clv1alpha2.Instance
		environment clv1alpha2.Environment

		objectName types.NamespacedName
		svc        corev1.Service
		secret     corev1.Secret
		vm         virtv1.VirtualMachine
		vmi        virtv1.VirtualMachineInstance

		ownerRef metav1.OwnerReference

		err error
	)

	const (
		instanceName      = "kubernetes-0000"
		instanceNamespace = "tenant-tester"
		templateName      = "kubernetes"
		templateNamespace = "workspace-netgroup"
		environmentName   = "control-plane"
		tenantName        = "tester"
		webdavCredentials = "webdav-credentials"

		image       = "internal/registry/image:v1.0"
		cpu         = 2
		cpuReserved = 25
		memory      = "1250M"
		disk        = "20Gi"
	)

	BeforeEach(func() {
		ctx = ctrl.LoggerInto(context.Background(), logr.Discard())
		clientBuilder = *fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			// These objects are required by the EnforceCloudInitSecret function.
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: webdavCredentials, Namespace: instanceNamespace},
				Data: map[string][]byte{
					instance_controller.WebdavSecretUsernameKey: []byte("username"),
					instance_controller.WebdavSecretPasswordKey: []byte("password"),
				},
			},
			&clv1alpha2.Template{ObjectMeta: metav1.ObjectMeta{Name: templateName, Namespace: templateNamespace}},
			&clv1alpha1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: tenantName}},
		)

		instance = clv1alpha2.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: instanceName, Namespace: instanceNamespace},
			Spec: clv1alpha2.InstanceSpec{
				Running:  true,
				Template: clv1alpha2.GenericRef{Name: templateName, Namespace: templateNamespace},
				Tenant:   clv1alpha2.GenericRef{Name: tenantName},
			},
		}
		environment = clv1alpha2.Environment{
			Name:            environmentName,
			EnvironmentType: clv1alpha2.ClassVM,
			Image:           image,
			Resources: clv1alpha2.EnvironmentResources{
				CPU:                   cpu,
				ReservedCPUPercentage: cpuReserved,
				Memory:                resource.MustParse(memory),
				Disk:                  resource.MustParse(disk),
			},
		}

		objectName = forge.NamespacedName(&instance)

		svc = corev1.Service{}
		secret = corev1.Secret{}
		vm = virtv1.VirtualMachine{}
		vmi = virtv1.VirtualMachineInstance{}

		ownerRef = metav1.OwnerReference{
			APIVersion:         clv1alpha2.GroupVersion.String(),
			Kind:               "Instance",
			Name:               instance.GetName(),
			UID:                instance.GetUID(),
			BlockOwnerDeletion: pointer.BoolPtr(true),
			Controller:         pointer.BoolPtr(true),
		}
	})

	JustBeforeEach(func() {
		client := FakeClientWrapped{Client: clientBuilder.Build()}
		reconciler = instance_controller.InstanceReconciler{Client: client, Scheme: scheme.Scheme, WebdavSecretName: webdavCredentials}

		ctx, _ = clctx.InstanceInto(ctx, &instance)
		ctx, _ = clctx.EnvironmentInto(ctx, &environment)
		err = reconciler.EnforceVMEnvironment(ctx)
	})

	It("Should enforce the cloud-init secret", func() {
		// Here, we only check the secret presence to assert the function execution, leaving the other assertions to the proper tests.
		Expect(reconciler.Get(ctx, objectName, &secret)).To(Succeed())
	})

	It("Should enforce the environment exposition objects", func() {
		// Here, we only check the service presence to assert the function execution, leaving the other assertions to the proper tests.
		Expect(reconciler.Get(ctx, objectName, &svc)).To(Succeed())
	})

	Context("The environment is not persistent", func() {
		BeforeEach(func() { environment.Persistent = false })

		When("the VMI it is not yet present", func() {
			When("the instance is running", func() {
				It("Should not return an error", func() { Expect(err).ToNot(HaveOccurred()) })

				It("The VMI should be present and have the common attributes", func() {
					Expect(reconciler.Get(ctx, objectName, &vmi)).To(Succeed())
					Expect(vmi.GetLabels()).To(Equal(forge.InstanceObjectLabels(nil, &instance)))
					Expect(vmi.GetOwnerReferences()).To(ContainElement(ownerRef))
				})

				It("The VMI should be present and have the expected specs", func() {
					Expect(reconciler.Get(ctx, objectName, &vmi)).To(Succeed())
					// Here we overwrite the VMI resources, since they would have a different representation due to the
					// marshaling/unmarshaling process. Still, the correctness of the value is already checked with the
					// appropriate test case.
					vmi.Spec.Domain.Resources = forge.VirtualMachineResources(&environment)
					Expect(vmi.Spec).To(Equal(forge.VirtualMachineInstanceSpec(&instance, &environment)))
				})

				It("Should leave the instance phase unset", func() {
					Expect(instance.Status.Phase).To(BeIdenticalTo(clv1alpha2.EnvironmentPhaseUnset))
				})
			})

			When("the instance is not running", func() {
				var notFoundError error

				BeforeEach(func() {
					instance.Spec.Running = false
					notFoundError = kerrors.NewNotFound(virtv1.Resource("virtualmachineinstances"), objectName.Name)
				})

				It("Should not return an error", func() { Expect(err).ToNot(HaveOccurred()) })

				It("The VMI should not be present", func() {
					Expect(reconciler.Get(ctx, objectName, &vmi)).To(MatchError(notFoundError))
				})

				It("Should set the instance phase to Off", func() {
					Expect(instance.Status.Phase).To(BeIdenticalTo(clv1alpha2.EnvironmentPhaseOff))
				})
			})
		})

		When("the VMI is already present", func() {
			var existing virtv1.VirtualMachineInstance

			BeforeEach(func() {
				existing = virtv1.VirtualMachineInstance{
					ObjectMeta: forge.NamespacedNameToObjectMeta(objectName),
					Status:     virtv1.VirtualMachineInstanceStatus{Phase: virtv1.Running},
				}
				existing.SetCreationTimestamp(metav1.NewTime(time.Now()))
				clientBuilder.WithObjects(&existing)
			})

			When("the instance is running", func() {
				It("Should not return an error", func() { Expect(err).ToNot(HaveOccurred()) })

				It("The VMI should still be present and have the common attributes", func() {
					Expect(reconciler.Get(ctx, objectName, &vmi)).To(Succeed())
					Expect(vmi.GetLabels()).To(Equal(forge.InstanceObjectLabels(nil, &instance)))
					Expect(vmi.GetOwnerReferences()).To(ContainElement(ownerRef))
				})

				It("The VMI should still be present and have unmodified specs", func() {
					Expect(reconciler.Get(ctx, objectName, &vmi)).To(Succeed())
					Expect(vmi.Spec).To(Equal(existing.Spec))
				})

				It("Should set the correct instance phase", func() {
					Expect(instance.Status.Phase).To(BeIdenticalTo(clv1alpha2.EnvironmentPhaseRunning))
				})
			})

			When("the instance is not running", func() {
				BeforeEach(func() { instance.Spec.Running = false })

				It("Should not return an error", func() { Expect(err).ToNot(HaveOccurred()) })

				It("The VMI should still be present but unmodified", func() {
					Expect(reconciler.Get(ctx, objectName, &vmi)).To(Succeed())
					Expect(vmi.ObjectMeta.Labels).To(Equal(existing.ObjectMeta.Labels))
					Expect(vmi.Spec).To(Equal(existing.Spec))
					Expect(vmi.Status).To(Equal(existing.Status))
				})

				It("Should set the instance phase to Off", func() {
					Expect(instance.Status.Phase).To(BeIdenticalTo(clv1alpha2.EnvironmentPhaseOff))
				})
			})
		})
	})

	Context("The environment is persistent", func() {
		BeforeEach(func() { environment.Persistent = true })

		When("the VM is not yet present", func() {
			It("Should not return an error", func() { Expect(err).ToNot(HaveOccurred()) })

			It("The VM should be present and have the common attributes", func() {
				Expect(reconciler.Get(ctx, objectName, &vm)).To(Succeed())
				Expect(vm.GetLabels()).To(Equal(forge.InstanceObjectLabels(nil, &instance)))
				Expect(vm.GetOwnerReferences()).To(ContainElement(ownerRef))
			})

			It("The VM should be present and have the expected specs", func() {
				Expect(reconciler.Get(ctx, objectName, &vm)).To(Succeed())
				// Here we overwrite the VM resources, since they would have a different representation due to the
				// marshaling/unmarshaling process. Still, the correctness of the value is already checked with the
				// appropriate test case. Additionally, we also overwrite the running value, which is checked in a
				// different It clause.
				vm.Spec.Template.Spec.Domain.Resources = forge.VirtualMachineResources(&environment)
				vm.Spec.Running = nil
				Expect(vm.Spec).To(Equal(forge.VirtualMachineSpec(&instance, &environment)))
			})

			It("The VM should be present and with the running flag set", func() {
				Expect(reconciler.Get(ctx, objectName, &vm)).To(Succeed())
				Expect(*vm.Spec.Running).To(BeTrue())
			})

			It("Should leave the instance phase unset", func() {
				Expect(instance.Status.Phase).To(BeIdenticalTo(clv1alpha2.EnvironmentPhaseUnset))
			})
		})

		WhenVMAlreadyPresentCase := func(running bool) {
			BeforeEach(func() {
				existing := virtv1.VirtualMachine{
					ObjectMeta: forge.NamespacedNameToObjectMeta(objectName),
					Spec:       virtv1.VirtualMachineSpec{Running: pointer.Bool(running)},
					Status:     virtv1.VirtualMachineStatus{PrintableStatus: virtv1.VirtualMachineStatusRunning},
				}
				existing.SetCreationTimestamp(metav1.NewTime(time.Now()))
				clientBuilder.WithObjects(&existing)
			})

			It("Should not return an error", func() { Expect(err).ToNot(HaveOccurred()) })

			It("The VM should still be present and have the common attributes", func() {
				Expect(reconciler.Get(ctx, objectName, &vm)).To(Succeed())
				Expect(vm.GetLabels()).To(Equal(forge.InstanceObjectLabels(nil, &instance)))
				Expect(vm.GetOwnerReferences()).To(ContainElement(ownerRef))
			})

			It("The VM should still be present and have unmodified specs", func() {
				Expect(reconciler.Get(ctx, objectName, &vm)).To(Succeed())
				// Here we overwrite the running value, as it is checked in a different It clause.
				vm.Spec.Running = nil
				Expect(vmi.Spec).To(Equal(virtv1.VirtualMachineInstanceSpec{}))
			})

			It("Should set the correct instance phase", func() {
				Expect(instance.Status.Phase).To(BeIdenticalTo(clv1alpha2.EnvironmentPhaseRunning))
			})

			Context("The instance is running", func() {
				BeforeEach(func() { instance.Spec.Running = true })

				It("The VM should be present and with the running flag set", func() {
					Expect(reconciler.Get(ctx, objectName, &vm)).To(Succeed())
					Expect(*vm.Spec.Running).To(BeTrue())
				})
			})

			Context("The instance is not running", func() {
				BeforeEach(func() { instance.Spec.Running = false })

				It("The VM should be present and with the running flag not set", func() {
					Expect(reconciler.Get(ctx, objectName, &vm)).To(Succeed())
					Expect(*vm.Spec.Running).To(BeFalse())
				})
			})
		}

		When("the VM is already present and it is running", func() { WhenVMAlreadyPresentCase(true) })
		When("the VM is already present and it is not running", func() { WhenVMAlreadyPresentCase(false) })
	})
})
