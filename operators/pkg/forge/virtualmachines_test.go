package forge_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	virtv1 "kubevirt.io/client-go/api/v1"

	clv1alpha2 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha2"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/forge"
)

var _ = Describe("VirtualMachines and VirtualMachineInstances forging", func() {
	var (
		instance    clv1alpha2.Instance
		environment clv1alpha2.Environment
	)

	const (
		instanceName      = "kubernetes-0000"
		instanceNamespace = "tenant-tester"
		image             = "internal/registry/image:v1.0"
		cpu               = 2
		cpuReserved       = 25
		memory            = "1250M"
		disk              = "20Gi"
	)

	BeforeEach(func() {
		instance = clv1alpha2.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: instanceName, Namespace: instanceNamespace},
		}
		environment = clv1alpha2.Environment{
			Image: image,
			Resources: clv1alpha2.EnvironmentResources{
				CPU:                   cpu,
				ReservedCPUPercentage: cpuReserved,
				Memory:                resource.MustParse(memory),
				Disk:                  resource.MustParse(disk),
			},
		}
	})

	Describe("The forge.VirtualMachineSpec function", func() {
		var spec virtv1.VirtualMachineSpec

		JustBeforeEach(func() {
			spec = forge.VirtualMachineSpec(&instance, &environment)
		})

		It("Should set the correct template labels", func() {
			Expect(spec.Template.ObjectMeta.GetLabels()).To(Equal(forge.InstanceSelectorLabels(&instance)))
		})
		It("Should set the correct template spec", func() {
			Expect(spec.Template.Spec).To(Equal(forge.VirtualMachineInstanceSpec(&instance, &environment)))
		})
		It("Should set the correct datavolume template", func() {
			Expect(spec.DataVolumeTemplates).To(ContainElement(
				forge.DataVolumeTemplate(forge.NamespacedName(&instance).Name, &environment)))
		})
	})

	Describe("The forge.VirtualMachineInstanceSpec function", func() {
		var spec virtv1.VirtualMachineInstanceSpec

		JustBeforeEach(func() {
			spec = forge.VirtualMachineInstanceSpec(&instance, &environment)
		})

		It("Should set the correct domain", func() {
			Expect(spec.Domain).To(Equal(forge.VirtualMachineDomain(&environment)))
		})
		It("Should set the cloud-init volumes", func() {
			Expect(spec.Volumes).To(ContainElement(forge.VolumeCloudInit(forge.NamespacedName(&instance).Name)))
		})
		It("Should set the correct readiness probe", func() {
			Expect(spec.ReadinessProbe).To(Equal(forge.VirtualMachineReadinessProbe(&environment)))
		})
		It("Should set the correct networks", func() {
			Expect(spec.Networks).To(ContainElement(*virtv1.DefaultPodNetwork()))
		})
		It("Should set the correct termination grace period", func() {
			Expect(*spec.TerminationGracePeriodSeconds).To(BeNumerically("==", 60))
		})

		When("the environment is not persistent", func() {
			BeforeEach(func() { environment.Persistent = false })
			It("Should set the container-disk volume", func() {
				Expect(spec.Volumes).To(ContainElement(forge.VolumeContainerDisk(image)))
			})
		})

		When("the environment is persistent", func() {
			BeforeEach(func() { environment.Persistent = true })
			It("Should set the persistent-disk volume", func() {
				Expect(spec.Volumes).To(ContainElement(forge.VolumePersistentDisk(forge.NamespacedName(&instance).Name)))
			})
		})
	})

	Describe("The forge.VirtualMachineDomain function", func() {
		var domain virtv1.DomainSpec

		JustBeforeEach(func() {
			domain = forge.VirtualMachineDomain(&environment)
		})

		It("Should set the correct CPU value", func() {
			Expect(domain.CPU.Cores).To(BeNumerically("==", cpu))
		})
		It("Should set the correct memory value", func() {
			Expect(*domain.Memory.Guest).To(Equal(resource.MustParse(memory)))
		})
		It("Should set the correct resources", func() {
			Expect(domain.Resources).To(Equal(forge.VirtualMachineResources(&environment)))
		})
		It("Should set the correct devices", func() {
			Expect(domain.Devices.Disks).To(ContainElement(forge.VolumeDiskTarget("root")))
			Expect(domain.Devices.Disks).To(ContainElement(forge.VolumeDiskTarget("cloud-init")))
			Expect(domain.Devices.Interfaces).To(ContainElement(*virtv1.DefaultBridgeNetworkInterface()))
		})
	})

	Describe("The forge.VolumeRootDisk function", func() {
		var volume virtv1.Volume

		JustBeforeEach(func() {
			volume = forge.VolumeRootDisk(&instance, &environment)
		})

		When("the environment is not persistent", func() {
			BeforeEach(func() { environment.Persistent = false })
			It("Should forge the container-disk volume", func() {
				Expect(volume).To(Equal(forge.VolumeContainerDisk(image)))
			})
		})

		When("the environment is persistent", func() {
			BeforeEach(func() { environment.Persistent = true })
			It("Should forge the persistent-disk volume", func() {
				Expect(volume).To(Equal(forge.VolumePersistentDisk(forge.NamespacedName(&instance).Name)))
			})
		})
	})

	Describe("The forge.VolumePersistentDisk function", func() {
		var volume virtv1.Volume
		const name = "data-volume-name"

		JustBeforeEach(func() {
			volume = forge.VolumePersistentDisk(name)
		})

		It("Should set the correct volume name", func() { Expect(volume.Name).To(Equal("root")) })
		It("Should set the correct volume type", func() { Expect(volume.DataVolume).ToNot(BeNil()) })
		It("Should set the correct volume image", func() { Expect(volume.DataVolume.Name).To(BeIdenticalTo(name)) })
	})

	Describe("The forge.VolumeContainerDisk function", func() {
		var volume virtv1.Volume

		JustBeforeEach(func() {
			volume = forge.VolumeContainerDisk(image)
		})

		It("Should set the correct volume name", func() { Expect(volume.Name).To(BeIdenticalTo("root")) })
		It("Should set the correct volume type", func() { Expect(volume.ContainerDisk).ToNot(BeNil()) })
		It("Should set the correct volume image", func() { Expect(volume.ContainerDisk.Image).To(BeIdenticalTo(image)) })
		It("Should set the correct volume image pull secret", func() { Expect(volume.ContainerDisk.ImagePullSecret).To(BeIdenticalTo("registry-credentials")) })
		It("Should set the correct volume image pull policy", func() { Expect(volume.ContainerDisk.ImagePullPolicy).To(BeIdenticalTo(corev1.PullIfNotPresent)) })
	})

	Describe("The forge.VolumeCloudInit function", func() {
		var volume virtv1.Volume
		const name = "cloud-init-secret"

		JustBeforeEach(func() {
			volume = forge.VolumeCloudInit(name)
		})

		It("Should set the correct volume name", func() { Expect(volume.Name).To(BeIdenticalTo("cloud-init")) })
		It("Should set the correct volume type", func() { Expect(volume.CloudInitNoCloud).ToNot(BeNil()) })
		It("Should set the correct volume secret reference", func() {
			Expect(volume.CloudInitNoCloud.UserDataSecretRef.Name).To(BeIdenticalTo(name))
		})
	})

	Describe("The forge.VolumeDiskTarget function", func() {
		var disk virtv1.Disk
		const name = "disk-name"

		JustBeforeEach(func() {
			disk = forge.VolumeDiskTarget(name)
		})

		It("Should set the correct disk name", func() { Expect(disk.Name).To(BeIdenticalTo(name)) })
		It("Should set the correct disk type", func() { Expect(disk.DiskDevice).ToNot(BeNil()) })
		It("Should set the correct disk target", func() { Expect(disk.DiskDevice.Disk.Bus).To(BeIdenticalTo("virtio")) })
	})

	Describe("The forge.VirtualMachineResources functions", func() {
		Describe("The accessory functions", func() {
			It("VirtualMachineCPURequests should correctly compute CPU requests", func() {
				Expect(forge.VirtualMachineCPURequests(&environment)).To(
					Equal(*resource.NewScaledQuantity(500, resource.Milli)))
			})

			It("VirtualMachineCPULimits should correctly compute CPU limits", func() {
				Expect(forge.VirtualMachineCPULimits(&environment)).To(
					Equal(*resource.NewScaledQuantity(2500, resource.Milli)))
			})

			It("VirtualMachineMemoryRequirements should correctly compute the memory requirements", func() {
				Expect(forge.VirtualMachineMemoryRequirements(&environment)).To(
					Equal(*resource.NewScaledQuantity(1750, resource.Mega)))
			})
		})

		Describe("The VirtualMachineResources function", func() {
			var requirements virtv1.ResourceRequirements

			JustBeforeEach(func() {
				requirements = forge.VirtualMachineResources(&environment)
			})

			It("Should set the CPU requests", func() {
				Expect(*requirements.Requests.Cpu()).To(Equal(forge.VirtualMachineCPURequests(&environment)))
			})
			It("Should set the CPU limits", func() {
				Expect(*requirements.Limits.Cpu()).To(Equal(forge.VirtualMachineCPULimits(&environment)))
			})
			It("Should set the memory requests", func() {
				Expect(*requirements.Requests.Memory()).To(Equal(forge.VirtualMachineMemoryRequirements(&environment)))
			})
			It("Should set the memory limits", func() {
				Expect(*requirements.Limits.Memory()).To(Equal(forge.VirtualMachineMemoryRequirements(&environment)))
			})
		})
	})

	Describe("The forge.VirtualMachineReadinessProbe function", func() {
		type VMReadinessProbeCase struct {
			Environment clv1alpha2.Environment
			Port        int
		}

		DescribeTable("Correctly returns the expected readiness probe",
			func(c VMReadinessProbeCase) {
				output := forge.VirtualMachineReadinessProbe(&c.Environment)

				Expect(output.Handler).To(Equal(virtv1.Handler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(c.Port),
					},
				}))

			},
			Entry("When the environment has the GUI enabled", VMReadinessProbeCase{
				Environment: clv1alpha2.Environment{GuiEnabled: true},
				Port:        forge.GUIPortNumber,
			}),
			Entry("When the environment has not the GUI enabled", VMReadinessProbeCase{
				Environment: clv1alpha2.Environment{GuiEnabled: false},
				Port:        forge.SSHPortNumber,
			}),
		)
	})

	Describe("The forge.DataVolumeTemplate function", func() {
		var dataVolumeTemplate virtv1.DataVolumeTemplateSpec
		const name = "kubernetes-volume"

		JustBeforeEach(func() {
			dataVolumeTemplate = forge.DataVolumeTemplate(name, &environment)
		})

		Context("The DataVolumeTemplate is forged", func() {
			It("Should have the correct name", func() {
				Expect(dataVolumeTemplate.GetName()).To(BeIdenticalTo(name))
			})

			It("Should target the correct image registry", func() {
				Expect(dataVolumeTemplate.Spec.Source.Registry.URL).To(
					BeIdenticalTo("docker://" + image))
			})

			It("Should request the correct disk size", func() {
				Expect(dataVolumeTemplate.Spec.PVC.Resources.Requests).To(Equal(
					corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(disk)}))
			})
		})
	})

})
