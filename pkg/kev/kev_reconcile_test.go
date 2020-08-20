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
		reporter     bytes.Buffer
		workingDir   string
		source       *kev.ComposeProject
		overlayFiles []string
		overlay      *kev.ComposeProject
		manifest     *kev.Manifest
		env          *kev.Environment
		mErr         error
	)

	JustBeforeEach(func() {
		if len(overlayFiles) > 0 {
			overlay, _ = kev.NewComposeProject(overlayFiles)
		}
		manifest, mErr = kev.Reconcile(workingDir, &reporter)
		if mErr == nil {
			env, _ = manifest.GetEnvironment("dev")
		}
	})

	Describe("Reconciling changes from sources", func() {

		Context("when the compose version has been updated", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-version"
				source, _ = kev.NewComposeProject([]string{workingDir + "/docker-compose.yaml"})
				reporter = bytes.Buffer{}
			})

			It("should update all environments with the new version", func() {
				Expect(env.GetVersion()).To(Equal(source.GetVersion()))
			})

			It("should create a change summary", func() {
				Expect(reporter.String()).To(ContainSubstring(env.Name))
				Expect(reporter.String()).To(ContainSubstring("3.7"))
				Expect(reporter.String()).To(ContainSubstring(env.GetVersion()))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a compose service has been removed", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-service-removal"
				overlayFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				reporter = bytes.Buffer{}
			})

			It("confirms the number of services pre reconciliation", func() {
				Expect(overlay.Services).To(HaveLen(2))
				Expect(overlay.ServiceNames()).To(ContainElements("db", "wordpress"))
			})

			It("should remove the service from all environments", func() {
				Expect(env.GetServices()).To(HaveLen(1))
				Expect(env.GetServices()[0].Name).To(Equal("db"))
			})

			It("should create a change summary", func() {
				Expect(reporter.String()).To(ContainSubstring(env.Name))
				Expect(reporter.String()).To(ContainSubstring("wordpress"))
				Expect(reporter.String()).To(ContainSubstring("deleted"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when the compose service is edited", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-service-edit"
				overlayFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				reporter = bytes.Buffer{}
			})

			Context("and it changes port mode to host", func() {
				It("confirms the edit pre reconciliation", func() {
					s, _ := overlay.GetService("wordpress")
					Expect(s.Labels["kev.service.type"]).To(Equal("LoadBalancer"))
				})

				It("should update the label in all environments", func() {
					s, _ := env.GetService("wordpress")
					Expect(s.Labels["kev.service.type"]).To(Equal("NodePort"))
				})

				It("should create a change summary", func() {
					Expect(reporter.String()).To(ContainSubstring(env.Name))
					Expect(reporter.String()).To(ContainSubstring("wordpress"))
					Expect(reporter.String()).To(ContainSubstring("updated"))
					Expect(reporter.String()).To(ContainSubstring(config.LabelServiceType))
					Expect(reporter.String()).To(ContainSubstring("LoadBalancer"))
					Expect(reporter.String()).To(ContainSubstring("NodePort"))
				})

				It("should not error", func() {
					Expect(mErr).NotTo(HaveOccurred())
				})
			})
		})

		Context("when a new compose service has been added", func() {
			Context("and the service has no deploy or healthcheck config", func() {
				BeforeEach(func() {
					workingDir = "testdata/reconcile-service-basic"
					overlayFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
					reporter = bytes.Buffer{}
				})

				It("confirms the number of services pre reconciliation", func() {
					Expect(overlay.Services).To(HaveLen(1))
					Expect(overlay.ServiceNames()).To(ContainElements("db"))
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

				It("should create a change summary", func() {
					Expect(reporter.String()).To(ContainSubstring(env.Name))
					Expect(reporter.String()).To(ContainSubstring("added"))
					Expect(reporter.String()).To(ContainSubstring("wordpress"))
				})

				It("should not error", func() {
					Expect(mErr).NotTo(HaveOccurred())
				})
			})

			Context("and the service has deploy config", func() {
				BeforeEach(func() {
					workingDir = "testdata/reconcile-service-deploy"
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

		Context("when a new compose volume has been added", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-volume-add"
				overlayFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				reporter = bytes.Buffer{}
			})

			It("confirms the number of volumes pre reconciliation", func() {
				Expect(overlay.Volumes).To(HaveLen(0))
			})

			It("should add the new volume labels to all environments", func() {
				Expect(env.GetVolumes()).To(HaveLen(1))

				v, _ := env.GetVolume("db_data")
				Expect(v.Labels[config.LabelVolumeSize]).To(Equal("100Mi"))
				Expect(v.Labels[config.LabelVolumeStorageClass]).To(Equal("standard"))
			})

			It("should create a change summary", func() {
				Expect(reporter.String()).To(ContainSubstring(env.Name))
				Expect(reporter.String()).To(ContainSubstring("added"))
				Expect(reporter.String()).To(ContainSubstring("db_data"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a compose volume has been removed", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-volume-removal"
				overlayFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				reporter = bytes.Buffer{}
			})

			It("confirms the number of volumes pre reconciliation", func() {
				Expect(overlay.Volumes).To(HaveLen(1))
			})

			It("should remove the volume from all environments", func() {
				Expect(env.GetVolumes()).To(HaveLen(0))
			})

			It("should create a change summary", func() {
				Expect(reporter.String()).To(ContainSubstring(env.Name))
				Expect(reporter.String()).To(ContainSubstring("deleted"))
				Expect(reporter.String()).To(ContainSubstring("db_data"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a compose volume has been edited", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-volume-edit"
				overlayFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				reporter = bytes.Buffer{}
			})

			It("confirms the volume name pre reconciliation", func() {
				Expect(overlay.VolumeNames()).To(HaveLen(1))
				Expect(overlay.VolumeNames()).To(ContainElements("db_data"))
			})

			It("should update the volume in all environments", func() {
				Expect(env.VolumeNames()).To(HaveLen(1))
				Expect(env.VolumeNames()).To(ContainElements("mysql_data"))
			})

			It("should create a change summary", func() {
				Expect(reporter.String()).To(ContainSubstring(env.Name))
				Expect(reporter.String()).To(ContainSubstring("deleted"))
				Expect(reporter.String()).To(ContainSubstring("db_data"))
				Expect(reporter.String()).To(ContainSubstring("added"))
				Expect(reporter.String()).To(ContainSubstring("mysql_data"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when compose env vars have been removed", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-env-var-removal"
				overlayFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				reporter = bytes.Buffer{}
			})

			It("confirms the env vars pre reconciliation", func() {
				s, _ := overlay.GetService("wordpress")
				Expect(s.Environment).To(HaveLen(2))
				Expect(s.Environment).To(HaveKey("WORDPRESS_CACHE_USER"))
				Expect(s.Environment).To(HaveKey("WORDPRESS_CACHE_PASSWORD"))
			})

			It("should remove the env vars from all environments", func() {
				vars, _ := env.GetEnvVarsForService("wordpress")
				Expect(vars).To(HaveLen(0))
			})

			It("confirms environment labels post reconciliation", func() {
				s, _ := env.GetService("wordpress")
				Expect(s.GetLabels()).To(HaveLen(16))
			})

			It("should create a change summary", func() {
				Expect(reporter.String()).To(ContainSubstring(env.Name))
				Expect(reporter.String()).To(ContainSubstring("deleted"))
				Expect(reporter.String()).To(ContainSubstring("WORDPRESS_CACHE_USER"))
				Expect(reporter.String()).To(ContainSubstring("WORDPRESS_CACHE_PASSWORD"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when compose env var is overridden in an environment", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-env-var-override"
				overlayFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				reporter = bytes.Buffer{}
			})

			It("confirms the overridden env var pre reconciliation", func() {
				s, _ := overlay.GetService("db")
				Expect(s.Environment).To(HaveLen(1))
				Expect(s.Environment).To(HaveKey("TO_OVERRIDE"))
			})

			It("should keep the overridden env var in all environments", func() {
				vars, _ := env.GetEnvVarsForService("db")
				Expect(vars).To(HaveLen(1))
				Expect(vars).To(HaveKey("TO_OVERRIDE"))
			})

			It("should create a change summary", func() {
				Expect(reporter.String()).To(ContainSubstring(env.Name))
				Expect(reporter.String()).To(ContainSubstring("nothing to update"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when compose or overlay env var is not assigned a value", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-env-var-unassigned"
				overlayFiles = []string{workingDir + "/docker-compose.kev.dev.yaml"}
				reporter = bytes.Buffer{}
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
