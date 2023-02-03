/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
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

package config_test

import (
	"bytes"

	"github.com/appvia/tako/pkg/tako/config"
	composego "github.com/compose-spec/compose-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Service Extension", func() {
	var (
		err          error
		parsedK8sCfg config.SvcK8sConfig
		svc          composego.ServiceConfig
	)

	JustBeforeEach(func() {
		parsedK8sCfg, err = config.SvcK8sConfigFromCompose(&svc)
	})

	AfterEach(func() {
		svc.Extensions = nil
		svc.Restart = ""
		svc.Deploy = nil
	})

	Describe("parsing", func() {
		Context("works with defaults", func() {
			BeforeEach(func() {
				svc.Extensions = map[string]interface{}{
					config.K8SExtensionKey: map[string]interface{}{
						"workload": map[string]interface{}{
							"type":     "Deployment",
							"replicas": 10,
							"livenessProbe": map[string]interface{}{
								"type": "none",
							},
						},
					},
				}
			})

			It("creates the config using defaults when the mandatory properties are present", func() {
				expectedLiveness := config.DefaultLivenessProbe()
				expectedLiveness.Type = config.ProbeTypeNone.String()

				Expect(parsedK8sCfg.Workload.Replicas).To(Equal(10))
				Expect(parsedK8sCfg.Workload.LivenessProbe).To(BeEquivalentTo(expectedLiveness))
				Expect(parsedK8sCfg.Workload.ReadinessProbe).To(BeEquivalentTo(config.DefaultReadinessProbe()))
			})
		})

		When("there is no k8s extension present", func() {
			Context("Without RequirePresent configuration", func() {
				It("does not fail validations", func() {
					Expect(parsedK8sCfg).NotTo(BeNil())

					Expect(parsedK8sCfg.Workload.Replicas).To(Equal(config.DefaultReplicaNumber))
					Expect(parsedK8sCfg.Workload.LivenessProbe).To(BeEquivalentTo(config.DefaultLivenessProbe()))
					Expect(parsedK8sCfg.Workload.ReadinessProbe).To(BeEquivalentTo(config.DefaultReadinessProbe()))
				})
			})
		})

		Describe("validations", func() {
			Context("from svc to k8s ext", func() {
				Context("with missing workload configuration", func() {
					BeforeEach(func() {
						svc.Extensions = map[string]interface{}{
							"x-k8s": map[string]interface{}{
								"bananas": 1,
							},
						}
					})

					It("returns defaults", func() {
						defaultWorkload := config.DefaultSvcK8sConfig().Workload
						Expect(parsedK8sCfg.Workload).To(BeEquivalentTo(defaultWorkload))
					})
				})

				Context("with invalid/empty workload configuration", func() {
					BeforeEach(func() {
						svc.Extensions = map[string]interface{}{
							"x-k8s": map[string]interface{}{
								"workload": map[string]interface{}{
									"bananas": 1,
								},
							},
						}
					})

					It("it is ignored and returns defaults", func() {
						Expect(parsedK8sCfg).To(BeEquivalentTo(config.DefaultSvcK8sConfig()))
					})
				})

				Context("missing liveness probe type in workload configuration", func() {
					BeforeEach(func() {
						svc.Extensions = map[string]interface{}{
							config.K8SExtensionKey: map[string]interface{}{
								"workload": map[string]interface{}{
									"type":     "Deployment",
									"replicas": 10,
								},
							},
						}
					})

					It("returns default probe", func() {
						Expect(parsedK8sCfg.Workload.LivenessProbe).To(Equal(config.DefaultLivenessProbe()))
					})
				})

				Context("missing replicas", func() {
					BeforeEach(func() {
						svc.Extensions = map[string]interface{}{
							config.K8SExtensionKey: map[string]interface{}{
								"workload": map[string]interface{}{
									"replicas": 0,
								},
							},
						}
					})

					It("returns in defaults", func() {
						defaultReplicas := config.DefaultSvcK8sConfig().Workload.Replicas
						Expect(parsedK8sCfg.Workload.Replicas).To(BeEquivalentTo(defaultReplicas))
					})
				})

				Context("restart policy", func() {
					When("invalid policy set in Restart Config", func() {
						BeforeEach(func() {
							svc.Restart = "invalid"
						})
						It("sets the default policy", func() {
							Expect(parsedK8sCfg.Workload.RestartPolicy).To(Equal(config.RestartPolicyAlways))
						})
					})

					When("unless-stopped policy set in Restart Config", func() {
						BeforeEach(func() {
							svc.Restart = "unless-stopped"
						})
						It("sets the default policy", func() {
							Expect(parsedK8sCfg.Workload.RestartPolicy).To(Equal(config.RestartPolicyAlways))
						})
					})

					When("invalid policy set in Deploy Config", func() {
						BeforeEach(func() {
							svc.Deploy = &composego.DeployConfig{
								RestartPolicy: &composego.RestartPolicy{
									Condition: "invalid",
								},
							}
						})

						It("sets the default policy", func() {
							Expect(parsedK8sCfg.Workload.RestartPolicy).To(Equal(config.RestartPolicyAlways))
						})
					})

					When("invalid policy set in extension", func() {
						BeforeEach(func() {
							svc.Extensions = map[string]interface{}{
								config.K8SExtensionKey: map[string]interface{}{
									"workload": map[string]interface{}{
										"restartPolicy": "invalid",
									},
								},
							}
						})

						It("should error", func() {
							Expect(err).To(HaveOccurred())
						})
					})

					When("policy is missing in extension", func() {
						BeforeEach(func() {
							svc.Extensions = map[string]interface{}{
								config.K8SExtensionKey: map[string]interface{}{
									"workload": map[string]interface{}{
										"restartPolicy": "",
									},
								},
							}
						})

						It("sets the default policy", func() {
							Expect(parsedK8sCfg.Workload.RestartPolicy).To(Equal(config.RestartPolicyAlways))
						})
					})
				})
			})

			Context("when running validate", func() {
				Context("with a missing service type", func() {
					It("returns error", func() {
						svcK8sConfig := config.DefaultSvcK8sConfig()
						svcK8sConfig.Service.Type = ""

						err = svcK8sConfig.Validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("SvcK8sConfig.Service.Type"))
					})
				})

				Context("with a missing workload type", func() {
					It("returns error", func() {
						svcK8sConfig := config.DefaultSvcK8sConfig()
						svcK8sConfig.Workload.Type = ""

						err = svcK8sConfig.Validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("SvcK8sConfig.Workload.Type"))
					})
				})
			})
		})
	})

	Describe("Marshalling", func() {
		It("doesn't lose information in serialization", func() {
			expected := config.DefaultLivenessProbe()

			var buf bytes.Buffer
			err := yaml.NewEncoder(&buf).Encode(expected)
			Expect(err).ToNot(HaveOccurred())

			var actual config.LivenessProbe
			err = yaml.NewDecoder(&buf).Decode(&actual)
			Expect(err).ToNot(HaveOccurred())

			Expect(expected).To(BeEquivalentTo(actual))
		})

		It("marshals invalid probetype as empty string", func() {
			expected := config.DefaultLivenessProbe()
			expected.Type = config.ProbeType("asd").String()

			var buf bytes.Buffer
			err := yaml.NewEncoder(&buf).Encode(expected)
			Expect(err).ToNot(HaveOccurred())

			var actual config.LivenessProbe
			err = yaml.NewDecoder(&buf).Decode(&actual)
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(BeEquivalentTo(actual))
			Expect(actual.Type).To(BeEmpty())
		})
	})

	Describe("Merge", func() {
		It("merges target into base", func() {
			k8sBase := config.DefaultSvcK8sConfig()
			var k8sTarget config.SvcK8sConfig
			k8sTarget.Workload.Replicas = 10
			k8sTarget.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()

			expected := k8sBase
			expected.Workload.Replicas = 10
			expected.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()

			result, err := k8sBase.Merge(k8sTarget)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEquivalentTo(expected))
		})

		Context("Fallback", func() {
			Context("configs are empty", func() {
				BeforeEach(func() {
					svc.Extensions = make(map[string]interface{})
				})

				It("returns default when map is empty", func() {
					result, err := config.DefaultSvcK8sConfig().Merge(parsedK8sCfg)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeEquivalentTo(config.DefaultSvcK8sConfig()))
				})
			})

			Context("configs are invalid", func() {
				It("returns default when map is nil", func() {
					result, err := config.DefaultSvcK8sConfig().Merge(parsedK8sCfg)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeEquivalentTo(config.DefaultSvcK8sConfig()))
				})
			})
		})
	})
})
