package context

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	clv1alpha1 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha1"
	clv1alpha2 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha2"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/context/mocks"
)

var _ = Describe("CrownLabs Context Objects", func() {
	var (
		ctx context.Context

		mockctrl *gomock.Controller
		mocklog  *mocks.MockLogger
		expected logr.Logger
		log      logr.Logger
	)

	BeforeEach(func() {
		mockctrl = gomock.NewController(GinkgoT())
		mocklog = mocks.NewMockLogger(mockctrl)
		expected = logr.Discard()
	})

	JustBeforeEach(func() {
		ctx = ctrl.LoggerInto(context.Background(), mocklog)
	})

	AfterEach(func() {
		mockctrl.Finish()
	})

	Describe("The context.InstanceInto/InstanceFrom methods", func() {
		When("storing an instance in the context and retrieving the logger", func() {
			var instance clv1alpha2.Instance

			BeforeEach(func() {
				instance = clv1alpha2.Instance{ObjectMeta: metav1.ObjectMeta{Name: "name", Namespace: "namespace"}}
				mocklog.EXPECT().WithValues(gomock.Eq(instanceKey), gomock.Eq(klog.KObj(&instance))).Return(expected)
			})

			JustBeforeEach(func() {
				ctx, log = InstanceInto(ctx, &instance)
			})

			It("InstanceInto should return and embed the correct logger", func() {
				Expect(log).To(Equal(expected))
				Expect(ctrl.LoggerFrom(ctx)).To(Equal(expected))
			})

			It("InstanceFrom should retrieve the same Instance object", func() {
				Expect(InstanceFrom(ctx)).To(BeIdenticalTo(&instance))
			})
		})
	})

	Describe("The context.TemplateInto/TemplateFrom methods", func() {
		When("storing a template in the context and retrieving the logger", func() {
			var template clv1alpha2.Template

			BeforeEach(func() {
				template = clv1alpha2.Template{ObjectMeta: metav1.ObjectMeta{Name: "name", Namespace: "namespace"}}
				mocklog.EXPECT().WithValues(gomock.Eq(templateKey), gomock.Eq(klog.KObj(&template))).Return(expected)
			})

			JustBeforeEach(func() {
				ctx, log = TemplateInto(ctx, &template)
			})

			It("TemplateInto should return and embed the correct logger", func() {
				Expect(log).To(Equal(expected))
				Expect(ctrl.LoggerFrom(ctx)).To(Equal(expected))
			})

			It("TemplateFrom should retrieve the same Template object", func() {
				Expect(TemplateFrom(ctx)).To(BeIdenticalTo(&template))
			})
		})
	})

	Describe("The context.TenantInto/TenantFrom methods", func() {
		When("storing a tenant in the context and retrieving the logger", func() {
			var tenant clv1alpha1.Tenant

			BeforeEach(func() {
				tenant = clv1alpha1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "name", Namespace: "namespace"}}
				mocklog.EXPECT().WithValues(gomock.Eq(tenantKey), gomock.Eq(klog.KObj(&tenant))).Return(expected)
			})

			JustBeforeEach(func() {
				ctx, log = TenantInto(ctx, &tenant)
			})

			It("TenantInto should return and embed the correct logger", func() {
				Expect(log).To(Equal(expected))
				Expect(ctrl.LoggerFrom(ctx)).To(Equal(expected))
			})

			It("TenantFrom should retrieve the same Template object", func() {
				Expect(TenantFrom(ctx)).To(BeIdenticalTo(&tenant))
			})
		})
	})

	Describe("The context.EnvironmentInto/EnvironmentFrom methods", func() {
		When("storing an environment in the context and retrieving the logger", func() {
			var environment clv1alpha2.Environment

			BeforeEach(func() {
				environment = clv1alpha2.Environment{Name: "name"}
				mocklog.EXPECT().WithValues(gomock.Eq(environmentKey), gomock.Eq(environment.Name)).Return(expected)
			})

			JustBeforeEach(func() {
				ctx, log = EnvironmentInto(ctx, &environment)
			})

			It("EnvironmentInto should return and embed the correct logger", func() {
				Expect(log).To(Equal(expected))
				Expect(ctrl.LoggerFrom(ctx)).To(Equal(expected))
			})

			It("EnvironmentFrom should retrieve the same Template object", func() {
				Expect(EnvironmentFrom(ctx)).To(BeIdenticalTo(&environment))
			})
		})
	})
})
