/**
 * Copyright 2020 Appvia Ltd <info@appvia.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kev_test

import (
	"github.com/appvia/kev/pkg/kev"
	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/testutil"
	kmd "github.com/appvia/komando"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

var _ = Describe("Reconcile", func() {
	var (
		hook           *test.Hook
		loggedMessages string
		workingDir     string
		source         *kev.ComposeProject
		overrideFiles  []string
		override       *kev.ComposeProject
		manifest       *kev.Manifest
		env            *kev.Environment
		mErr           error
	)

	JustBeforeEach(func() {
		var err error
		if len(overrideFiles) > 0 {
			override, err = kev.NewComposeProject(overrideFiles)
			Expect(err).NotTo(HaveOccurred())
		}
		hook = testutil.NewLogger(logrus.DebugLevel)

		r := kev.NewRenderRunner(workingDir, kev.WithUI(kmd.NoOpUI()))
		err = r.LoadProject()
		Expect(err).NotTo(HaveOccurred(), workingDir)

		manifest, mErr = r.Manifest().ReconcileConfig()
		Expect(mErr).NotTo(HaveOccurred(), workingDir)

		env, err = manifest.GetEnvironment("dev")
		Expect(err).NotTo(HaveOccurred())

		loggedMessages = testutil.GetLoggedMsgs(hook)
	})

	JustAfterEach(func() {
		hook.Reset()
	})

	Describe("Reconcile changes from overrides", func() {
		When("the override version has been updated", func() {
			BeforeEach(func() {
				var err error
				workingDir = "testdata/reconcile-override-rollback"
				source, err = kev.NewComposeProject([]string{workingDir + "/docker-compose.yaml"})
				Expect(err).NotTo(HaveOccurred())
				overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
			})

			It("confirms the version pre reconciliation", func() {
				Expect(override.GetVersion()).NotTo(Equal(source.GetVersion()))
			})

			It("should roll back the change", func() {
				Expect(env.GetVersion()).To(Equal(source.GetVersion()))
			})
		})

		Context("for services and volumes", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-override-keep"
				overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
			})

			When("the override service extensions have been updated", func() {
				It("confirms the values pre reconciliation", func() {
					s, err := override.GetService("db")
					Expect(err).NotTo(HaveOccurred())

					svcK8sConfig, err := config.ParseSvcK8sConfigFromMap(s.Extensions)
					Expect(err).NotTo(HaveOccurred())

					Expect(svcK8sConfig.Workload.Resource.CPU).To(Equal("0.5"))
					Expect(svcK8sConfig.Workload.Resource.MaxCPU).To(Equal("0.75"))
					Expect(svcK8sConfig.Workload.Resource.Memory).To(Equal("50Mi"))
					Expect(svcK8sConfig.Workload.Replicas).To(Equal(5))
					Expect(svcK8sConfig.Workload.ServiceAccountName).To(Equal("overridden-service-account-name"))
				})

				It("keeps overridden override values", func() {
					s, err := env.GetService("db")
					Expect(err).NotTo(HaveOccurred())

					svcK8sConfig, err := config.ParseSvcK8sConfigFromMap(s.Extensions)
					Expect(err).NotTo(HaveOccurred())

					Expect(svcK8sConfig.Workload.Resource.CPU).To(Equal("0.5"))
					Expect(svcK8sConfig.Workload.Resource.MaxCPU).To(Equal("0.75"))
					Expect(svcK8sConfig.Workload.Resource.Memory).To(Equal("50Mi"))
					Expect(svcK8sConfig.Workload.Replicas).To(Equal(5))
					Expect(svcK8sConfig.Workload.ServiceAccountName).To(Equal("overridden-service-account-name"))
				})
			})

			When("the override volume extensions have been updated", func() {
				It("confirms the values pre reconciliation", func() {
					v := override.Volumes["db_data"]
					volK8sConfig, err := config.ParseVolK8sConfigFromMap(v.Extensions)

					Expect(err).NotTo(HaveOccurred())
					Expect(volK8sConfig.Size).To(Equal("200Mi"))
				})

				It("keeps overridden override values", func() {
					v, _ := env.GetVolume("db_data")
					volK8sConfig, err := config.ParseVolK8sConfigFromMap(v.Extensions)

					Expect(err).NotTo(HaveOccurred())
					Expect(volK8sConfig.Size).To(Equal("200Mi"))
				})
			})
		})
	})

	Describe("Reconciling changes from sources", func() {

		Context("when the compose version has been updated", func() {
			BeforeEach(func() {
				var err error
				workingDir = "testdata/reconcile-version"
				source, err = kev.NewComposeProject([]string{workingDir + "/docker-compose.yaml"})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should update all environments with the new version", func() {
				Expect(env.GetVersion()).To(Equal(source.GetVersion()))
			})

			It("should log the change summary using the debug level", func() {
				Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
			})

			It("should create a change summary", func() {
				Expect(loggedMessages).To(ContainSubstring(env.Name))
				Expect(loggedMessages).To(ContainSubstring(env.Name))
				Expect(loggedMessages).To(ContainSubstring("3.7"))
				Expect(loggedMessages).To(ContainSubstring(env.GetVersion()))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a compose service has been removed", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-service-removal"
				overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
			})

			It("confirms the number of services pre reconciliation", func() {
				Expect(override.Services).To(HaveLen(2))
				Expect(override.ServiceNames()).To(ContainElements("db", "wordpress"))
			})

			It("should remove the service from all environments", func() {
				Expect(env.GetServices()).To(HaveLen(1))
				Expect(env.GetServices()[0].Name).To(Equal("db"))
			})

			It("should log the change summary using the debug level", func() {
				Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
			})

			It("should create a change summary", func() {
				Expect(loggedMessages).To(ContainSubstring(env.Name))
				Expect(loggedMessages).To(ContainSubstring("wordpress"))
				Expect(loggedMessages).To(ContainSubstring("removed"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when the compose service is edited", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-service-edit"
				overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
			})

			Context("and it changes port mode to host", func() {
				It("confirms the edit pre reconciliation", func() {
					s, _ := override.GetService("wordpress")

					svcK8sConfig, err := config.ParseSvcK8sConfigFromMap(s.Extensions)
					Expect(err).NotTo(HaveOccurred())

					Expect(svcK8sConfig.Service.Type).To(Equal(config.LoadBalancerService))
				})

				It("should not update any of the environments", func() {
					s, _ := env.GetService("wordpress")

					svcK8sConfig, err := config.ParseSvcK8sConfigFromMap(s.Extensions)
					Expect(err).NotTo(HaveOccurred())

					Expect(svcK8sConfig.Service.Type).To(Equal(config.LoadBalancerService))
				})

				It("should log the change summary using the debug level", func() {
					Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
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
					overrideFiles = []string{
						workingDir + "/docker-compose.env.dev.yaml",
						workingDir + "/docker-compose.env.stage.yaml",
					}
				})

				It("confirms the number of services pre reconciliation", func() {
					Expect(override.Services).To(HaveLen(1))
					Expect(override.ServiceNames()).To(ContainElements("db"))
				})

				It("should add the new service to all environments", func() {
					envs, err := manifest.GetEnvironments([]string{"dev", "stage"})
					Expect(err).ToNot(HaveOccurred())
					for _, env := range envs {
						Expect(env.GetServices()).To(HaveLen(2), "failed for env: %s", env.Name)
						Expect(env.GetServices()[0].Name).To(Equal("db"), "failed for env: %s", env.Name)
						Expect(env.GetServices()[1].Name).To(Equal("wordpress"), "failed for env: %s", env.Name)
					}
				})

				It("should configure the added service extension value defaults in all environments", func() {
					expected, err := newMinifiedServiceExtensions("wordpress", false)
					Expect(err).NotTo(HaveOccurred())

					envs, err := manifest.GetEnvironments([]string{"dev", "stage"})
					Expect(err).ToNot(HaveOccurred())
					for _, env := range envs {
						Expect(env.GetServices()[1].Extensions).To(Equal(expected), "failed for env: %s", env.Name)
					}
				})

				It("should not include any env vars in any environments", func() {
					envs, err := manifest.GetEnvironments([]string{"dev", "stage"})
					Expect(err).ToNot(HaveOccurred())
					for _, env := range envs {
						Expect(env.GetServices()[1].Environment).To(HaveLen(0), "failed for env: %s", env.Name)
					}
				})

				It("should log the change summary using the debug level", func() {
					Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
				})

				It("should create a change summary", func() {
					Expect(loggedMessages).To(ContainSubstring(env.Name))
					Expect(loggedMessages).To(ContainSubstring("added"))
					Expect(loggedMessages).To(ContainSubstring("wordpress"))
				})

				It("should not error", func() {
					Expect(mErr).NotTo(HaveOccurred())
				})
			})

			Context("and the service has deploy config", func() {
				BeforeEach(func() {
					workingDir = "testdata/reconcile-service-deploy"
				})

				It("should configure parse config into extensions", func() {
					expected, err := newMinifiedServiceExtensions("wordpress", false, config.SvcK8sConfig{
						Workload: config.Workload{
							Replicas: 3,
						},
					})
					Expect(err).NotTo(HaveOccurred())

					k8s, err := config.ParseSvcK8sConfigFromMap(env.GetServices()[1].Extensions, config.SkipValidation())
					Expect(err).NotTo(HaveOccurred())

					Expect(k8s.Workload.Replicas).To(Equal(3))
					Expect(cmp.Diff(env.GetServices()[1].Extensions, expected)).To(BeEmpty())
				})
			})

			Context("and the service has healthcheck config", func() {
				BeforeEach(func() {
					workingDir = "testdata/reconcile-service-healthcheck"
				})

				It("should configure the added service extensions from healthcheck config", func() {
					expected, err := newMinifiedServiceExtensions("wordpress", true)
					Expect(err).NotTo(HaveOccurred())
					expected["x-k8s"].(map[string]interface{})["workload"].(map[string]interface{})["livenessProbe"] = map[string]interface{}{
						"type": config.ProbeTypeNone.String(),
					}

					Expect(env.GetServices()[1].Extensions).To(Equal(expected))
				})
			})
		})

		Context("when a new compose volume has been added", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-volume-add"
				overrideFiles = []string{
					workingDir + "/docker-compose.env.dev.yaml",
					workingDir + "/docker-compose.env.stage.yaml",
				}
			})

			It("confirms the number of volumes pre reconciliation", func() {
				Expect(override.Volumes).To(HaveLen(0))
			})

			It("should add the new volumes to all environments", func() {
				envs, err := manifest.GetEnvironments([]string{"dev", "stage"})
				Expect(err).ToNot(HaveOccurred())

				for _, env := range envs {
					Expect(env.GetVolumes()).To(HaveLen(1))

					v, _ := env.GetVolume("db_data")
					volExt := v.Extensions[config.K8SExtensionKey].(map[string]interface{})

					Expect(v.Extensions).To(HaveLen(1), "failed for env: %s", env.Name)
					Expect(volExt["size"]).To(Equal("1Gi"), "failed for env: %s", env.Name)
				}

			})

			It("should log the change summary using the debug level", func() {
				Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
			})

			It("should create a change summary", func() {
				Expect(loggedMessages).To(ContainSubstring(env.Name))
				Expect(loggedMessages).To(ContainSubstring("added"))
				Expect(loggedMessages).To(ContainSubstring("db_data"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a compose volume has been removed", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-volume-removal"
				overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
			})

			It("confirms the number of volumes pre reconciliation", func() {
				Expect(override.Volumes).To(HaveLen(1))
			})

			It("should remove the volume from all environments", func() {
				Expect(env.GetVolumes()).To(HaveLen(0))
			})

			It("should log the change summary using the debug level", func() {
				Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
			})

			It("should create a change summary", func() {
				Expect(loggedMessages).To(ContainSubstring(env.Name))
				Expect(loggedMessages).To(ContainSubstring("removed"))
				Expect(loggedMessages).To(ContainSubstring("db_data"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when a compose volume has been edited", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-volume-edit"
				overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
			})

			It("confirms the volume name pre reconciliation", func() {
				Expect(override.VolumeNames()).To(HaveLen(1))
				Expect(override.VolumeNames()).To(ContainElements("db_data"))
			})

			It("should update the volume in all environments", func() {
				Expect(env.VolumeNames()).To(HaveLen(1))
				Expect(env.VolumeNames()).To(ContainElements("mysql_data"))
			})

			It("should log the change summary using the debug level", func() {
				Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
			})

			It("should create a change summary", func() {
				Expect(loggedMessages).To(ContainSubstring(env.Name))
				Expect(loggedMessages).To(ContainSubstring("removed"))
				Expect(loggedMessages).To(ContainSubstring("db_data"))
				Expect(loggedMessages).To(ContainSubstring("added"))
				Expect(loggedMessages).To(ContainSubstring("mysql_data"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when compose env vars have been removed", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-env-var-removal"
				overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
			})

			It("confirms the env vars pre reconciliation", func() {
				s, _ := override.GetService("wordpress")
				Expect(s.Environment).To(HaveLen(2))
				Expect(s.Environment).To(HaveKey("WORDPRESS_CACHE_USER"))
				Expect(s.Environment).To(HaveKey("WORDPRESS_CACHE_PASSWORD"))
			})

			It("should remove the env vars from all environments", func() {
				vars, _ := env.GetEnvVarsForService("wordpress")
				Expect(vars).To(HaveLen(0))
			})

			It("should log the change summary using the debug level", func() {
				Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
			})

			It("should create a change summary", func() {
				Expect(loggedMessages).To(ContainSubstring(env.Name))
				Expect(loggedMessages).To(ContainSubstring("removed"))
				Expect(loggedMessages).To(ContainSubstring("WORDPRESS_CACHE_USER"))
				Expect(loggedMessages).To(ContainSubstring("WORDPRESS_CACHE_PASSWORD"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when compose env var is overridden in an environment", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-env-var-override"
				overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
			})

			It("confirms the overridden env var pre reconciliation", func() {
				s, _ := override.GetService("db")
				Expect(s.Environment).To(HaveLen(1))
				Expect(s.Environment).To(HaveKey("TO_OVERRIDE"))
			})

			It("should keep the overridden env var in all environments", func() {
				vars, _ := env.GetEnvVarsForService("db")
				Expect(vars).To(HaveLen(1))
				Expect(vars).To(HaveKey("TO_OVERRIDE"))
			})

			It("should log the change summary using the debug level", func() {
				Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when compose or override env var is not assigned a value", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-env-var-unassigned"
				overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when compose image name is overridden in an environment", func() {
			BeforeEach(func() {
				workingDir = "testdata/reconcile-image-name-override"
				overrideFiles = []string{
					workingDir + "/docker-compose.env.dev.yaml",
					workingDir + "/docker-compose.env.stage.yaml",
				}
			})

			It("confirms the overridden image name pre-reconciliation", func() {
				s, err := override.GetService("db")
				Expect(err).NotTo(HaveOccurred())
				Expect(s.Image).To(Equal("mysql:latest"))
			})

			It("should keep the appropriate image in the environment override", func() {
				testCases := []struct {
					env           string
					expectedImage string
				}{
					{env: "dev", expectedImage: "mysql:latest"},
					{env: "stage", expectedImage: ""},
				}
				for _, tc := range testCases {
					env, err := manifest.GetEnvironment(tc.env)
					Expect(err).NotTo(HaveOccurred())

					svc, err := env.GetService("db")
					Expect(err).NotTo(HaveOccurred())
					Expect(svc.Image).To(Equal(tc.expectedImage))
				}
			})

			It("should log the change summary using the debug level", func() {
				Expect(testutil.GetLoggedLevel(hook)).To(Equal("debug"))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("when healthcheck is overridden by overlay", func() {
			Context("liveness tcp", func() {
				BeforeEach(func() {
					workingDir = "testdata/reconcile-healthcheck-tcp"
					overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
				})

				It("should have a valid tcp", func() {
					svcK8sConfig, err := config.ParseSvcK8sConfigFromMap(env.GetServices()[0].Extensions)
					Expect(err).NotTo(HaveOccurred())

					Expect(svcK8sConfig.Workload.LivenessProbe.Type).To(Equal(config.ProbeTypeTCP.String()))
					Expect(svcK8sConfig.Workload.LivenessProbe.TCP.Port).To(Equal(8080))
					Expect(svcK8sConfig.Workload.LivenessProbe.Exec.Command).To(BeEmpty())
				})
			})
			Context("liveness and readiness http", func() {
				BeforeEach(func() {
					workingDir = "testdata/reconcile-healthcheck-http"
					overrideFiles = []string{workingDir + "/docker-compose.env.dev.yaml"}
				})

				It("should have a valid http liveness probe", func() {
					svcCfg, err := env.GetService("db")
					Expect(err).NotTo(HaveOccurred())

					svcK8sConfig, err := config.ParseSvcK8sConfigFromMap(svcCfg.Extensions)
					Expect(err).NotTo(HaveOccurred())

					Expect(svcK8sConfig.Workload.LivenessProbe.Type).To(Equal(config.ProbeTypeHTTP.String()))
					Expect(svcK8sConfig.Workload.LivenessProbe.HTTP.Port).To(Equal(8080))
					Expect(svcK8sConfig.Workload.LivenessProbe.HTTP.Path).To(Equal("/status"))
					Expect(svcK8sConfig.Workload.LivenessProbe.Exec.Command).To(BeEmpty())
				})

				It("should have a valid http readiness probe", func() {
					svcCfg, err := env.GetService("wordpress")
					Expect(err).NotTo(HaveOccurred())

					svcK8sConfig, err := config.ParseSvcK8sConfigFromMap(svcCfg.Extensions)
					Expect(err).NotTo(HaveOccurred())

					Expect(svcK8sConfig.Workload.ReadinessProbe.Type).To(Equal(config.ProbeTypeHTTP.String()))
					Expect(svcK8sConfig.Workload.ReadinessProbe.HTTP.Port).To(Equal(8080))
				})
			})
		})
	})
})

func newMinifiedServiceExtensions(_ string, includeLivenessProbe bool, svcK8sConfigs ...config.SvcK8sConfig) (map[string]interface{}, error) {
	livenessProbe := config.DefaultLivenessProbe()
	k8s := config.SvcK8sConfig{
		Workload: config.Workload{
			Replicas: config.DefaultReplicaNumber,
		},
	}

	if includeLivenessProbe {
		k8s.Workload.LivenessProbe = config.LivenessProbe{
			Type: livenessProbe.Type,
			ProbeConfig: config.ProbeConfig{
				Exec: config.ExecProbe{
					Command: livenessProbe.Exec.Command,
				},
			},
		}
	}

	for _, conf := range svcK8sConfigs {
		c, err := k8s.Merge(conf)
		if err != nil {
			return nil, err
		}

		k8s = c
	}

	m, err := k8s.Map()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{config.K8SExtensionKey: m}, nil
}
