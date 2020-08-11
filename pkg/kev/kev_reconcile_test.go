package kev_test

import (
	"bytes"

	"github.com/appvia/kube-devx/pkg/kev"
	"github.com/appvia/kube-devx/pkg/kev/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconcile", func() {
	var (
		buffer       bytes.Buffer
		workingDir   string
		composeFiles []string
		project      *kev.ComposeProject
		manifest     *kev.Manifest
		env          *kev.Environment
		mErr         error
	)

	JustBeforeEach(func() {
		project, _ = kev.NewComposeProject(composeFiles)
		manifest, mErr = kev.Reconcile(workingDir, &buffer)
		env, _ = manifest.GetEnvironment("dev")
	})

	Describe("Reconciling changes from sources", func() {
		// Adding environment vars??

		Context("when the version has been updated", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-version"
				composeFiles = []string{workingDir + "/docker-compose.yaml"}
			})

			It("should update all environments with the new version", func() {
				Expect(env.GetVersion()).To(Equal(project.GetVersion()))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a service has been removed", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-service-removal"
				composeFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
			})

			It("confirms the number of services pre reconciliation", func() {
				Expect(project.Services).To(HaveLen(2))
				Expect(project.ServiceNames()).To(ContainElements("db", "wordpress"))
			})

			It("should remove the service from all environments", func() {
				Expect(env.GetServices()).To(HaveLen(1))
				Expect(env.GetServices()[0].Name).To(Equal("wordpress"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a service has edited attributes", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-service-edit"
				composeFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
			})

			It("confirms the edit pre reconciliation", func() {
				s, _ := project.GetService("wordpress")
				Expect(s.Labels["kev.service.type"]).To(Equal("LoadBalancer"))
			})

			It("should update the related label in all environments", func() {
				s, _ := env.GetService("wordpress")
				Expect(s.Labels["kev.service.type"]).To(Equal("NodePort"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a new service has been added", func() {

			Context("and the service has no deploy or healthcheck config", func() {
				BeforeEach(func() {
					workingDir = "testdata/reconcile-service-basic"
					composeFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				})

				It("confirms the number of services pre reconciliation", func() {
					Expect(project.Services).To(HaveLen(1))
					Expect(project.ServiceNames()).To(ContainElements("db"))
				})

				It("should add the new service labels to all environments", func() {
					Expect(env.GetServices()).To(HaveLen(2))
					Expect(env.GetServices()[0].Name).To(Equal("db"))
					Expect(env.GetServices()[1].Name).To(Equal("wordpress"))
				})

				It("should configure the added service labels with defaults", func() {
					expected := newDefaultServiceLabels("wordpress", "LoadBalancer")
					Expect(env.GetServices()[1].GetLabels()).To(Equal(expected))
				})

				It("should not error", func() {
					Expect(mErr).NotTo(HaveOccurred())
				})
			})

			Context("and the service has deploy config", func() {
				BeforeEach(func() {
					workingDir = "testdata/reconcile-service-deploy"
					composeFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				})

				It("should configure the added service labels from deploy config", func() {
					expected := newDefaultServiceLabels("wordpress", "LoadBalancer")
					expected[config.LabelWorkloadReplicas] = "3"
					expected[config.LabelWorkloadCPU] = "0.25"
					expected[config.LabelWorkloadMaxCPU] = "0.25"
					expected[config.LabelWorkloadMemory] = "20Mi"
					expected[config.LabelWorkloadMaxMemory] = "50Mi"
					Expect(env.GetServices()[1].GetLabels()).To(Equal(expected))
				})
			})

			Context("and the service has healthcheck config", func() {
				BeforeEach(func() {
					workingDir = "testdata/reconcile-service-healthcheck"
					composeFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				})

				It("should configure the added service labels from healthcheck config", func() {
					expected := newDefaultServiceLabels("wordpress", "LoadBalancer")
					expected[config.LabelWorkloadLivenessProbeCommand] = "[\"CMD\", \"curl\", \"localhost:80/healthy\"]"
					expected[config.LabelWorkloadLivenessProbeDisabled] = "true"
					expected[config.LabelWorkloadLivenessProbeInitialDelay] = "2m0s"
					expected[config.LabelWorkloadLivenessProbeInterval] = "5m0s"
					expected[config.LabelWorkloadLivenessProbeRetries] = "10"
					expected[config.LabelWorkloadLivenessProbeTimeout] = "30s"
					Expect(env.GetServices()[1].GetLabels()).To(Equal(expected))
				})
			})
		})

		Context("when a new volume has been added", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-volume-add"
				composeFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
			})

			It("confirms the number of volumes pre reconciliation", func() {
				Expect(project.Volumes).To(HaveLen(0))
			})

			It("should add the new volume labels to all environments", func() {
				Expect(env.GetVolumes()).To(HaveLen(1))

				v, _ := env.GetVolume("db_data")
				Expect(v.Labels["kev.volume.size"]).To(Equal("100Mi"))
				Expect(v.Labels["kev.volume.storage-class"]).To(Equal("standard"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a volume has been removed", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-volume-removal"
				composeFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
			})

			It("confirms the number of volumes pre reconciliation", func() {
				Expect(project.Volumes).To(HaveLen(1))
			})

			It("should remove the volume from all environments", func() {
				Expect(env.GetVolumes()).To(HaveLen(0))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a volume has been edited", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-volume-edit"
				composeFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
			})

			It("confirms the volume name pre reconciliation", func() {
				Expect(project.VolumeNames()).To(HaveLen(1))
				Expect(project.VolumeNames()).To(ContainElements("db_data"))
			})

			It("should update the volume in all environments", func() {
				Expect(env.VolumeNames()).To(HaveLen(1))
				Expect(env.VolumeNames()).To(ContainElements("mysql_data"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when env vars have been removed", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-env-var-removal"
				composeFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
			})

			It("confirms the env vars pre reconciliation", func() {
				s, _ := project.GetService("wordpress")
				Expect(s.Environment).To(HaveLen(2))
				Expect(s.Environment).To(HaveKey("WORDPRESS_CACHE_USER"))
				Expect(s.Environment).To(HaveKey("WORDPRESS_CACHE_PASSWORD"))
			})

			It("should remove the env vars from all environments", func() {
				vars, _ := env.GetEnvVars("wordpress")
				Expect(vars).To(HaveLen(0))
			})

			It("confirms environment labels post reconciliation", func() {
				s, _ := env.GetService("wordpress")
				Expect(s.GetLabels()).To(HaveLen(16))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})
	})
})

func newDefaultServiceLabels(name string, svcType string) map[string]string {
	return map[string]string{
		config.LabelServiceType:                       svcType,
		config.LabelWorkloadCPU:                       "0.1",
		config.LabelWorkloadImagePullPolicy:           "IfNotPresent",
		config.LabelWorkloadLivenessProbeCommand:      "[\"CMD\", \"echo\", \"Define healthcheck command for service " + name + "\"]",
		config.LabelWorkloadLivenessProbeDisabled:     "false",
		config.LabelWorkloadLivenessProbeInitialDelay: "1m0s",
		config.LabelWorkloadLivenessProbeInterval:     "1m0s",
		config.LabelWorkloadLivenessProbeRetries:      "3",
		config.LabelWorkloadLivenessProbeTimeout:      "10s",
		config.LabelWorkloadMaxCPU:                    "0.5",
		config.LabelWorkloadMaxMemory:                 "500Mi",
		config.LabelWorkloadMemory:                    "10Mi",
		config.LabelWorkloadReplicas:                  "1",
		config.LabelWorkloadRollingUpdateMaxSurge:     "1",
		config.LabelWorkloadServiceAccountName:        "default",
		config.LabelWorkloadType:                      "Deployment",
	}
}
