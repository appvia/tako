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

	"github.com/appvia/kev/pkg/kev/config"
	composego "github.com/compose-spec/compose-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Service Extensions", func() {

	Describe("parsing", func() {
		var (
			k8sCfg config.SvcK8sConfig
			err    error

			parsedK8sCfg config.SvcK8sConfig
			svc          composego.ServiceConfig
		)

		BeforeEach(func() {
			k8sCfg = config.SvcK8sConfig{}
		})

		JustBeforeEach(func() {
			m, err := k8sCfg.ToMap()
			Expect(err).NotTo(HaveOccurred())
			svc.Extensions = map[string]interface{}{
				config.K8SExtensionKey: m,
			}

			parsedK8sCfg, err = config.SvcK8sConfigFromCompose(&svc)
			Expect(err).NotTo(HaveOccurred())

		})

		Context("works with defaults", func() {
			BeforeEach(func() {
				k8sCfg.Workload.Type = "Deployment"
				k8sCfg.Workload.Replicas = 10
				k8sCfg.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()
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
					parsedK8sCfg, err = config.SvcK8sConfigFromCompose(&svc)
					Expect(err).ToNot(HaveOccurred())
					Expect(parsedK8sCfg).NotTo(BeNil())

					Expect(parsedK8sCfg.Workload.Replicas).To(Equal(config.DefaultReplicaNumber))
					Expect(parsedK8sCfg.Workload.LivenessProbe).To(BeEquivalentTo(config.DefaultLivenessProbe()))
					Expect(parsedK8sCfg.Workload.ReadinessProbe).To(BeEquivalentTo(config.DefaultReadinessProbe()))
				})
			})
		})

		Describe("validations", func() {
			Context("with missing workload", func() {
				JustBeforeEach(func() {
					svc.Extensions = map[string]interface{}{
						"x-k8s": map[string]interface{}{
							"bananas": 1,
						},
					}

					parsedK8sCfg, err = config.SvcK8sConfigFromCompose(&svc)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns defaults", func() {
					defaultWorkload := config.DefaultSvcK8sConfig().Workload
					Expect(parsedK8sCfg.Workload).To(BeEquivalentTo(defaultWorkload))
				})
			})

			Context("invalid/empty workload", func() {
				JustBeforeEach(func() {
					svc.Extensions = map[string]interface{}{
						"x-k8s": map[string]interface{}{
							"workload": map[string]interface{}{
								"bananas": 1,
							},
						},
					}

					parsedK8sCfg, err = config.SvcK8sConfigFromCompose(&svc)
					Expect(err).NotTo(HaveOccurred())
				})

				When("workload is invalid", func() {
					It("it is ignored and returns defaults", func() {
						Expect(parsedK8sCfg).To(BeEquivalentTo(config.DefaultSvcK8sConfig()))
					})
				})
			})

			Context("missing liveness probe type", func() {
				BeforeEach(func() {
					k8sCfg.Workload.Type = config.DefaultWorkload
					k8sCfg.Workload.Replicas = 10
				})

				When("liveness probe type not provided", func() {
					It("return default probe", func() {
						Expect(parsedK8sCfg.Workload.LivenessProbe).To(Equal(config.DefaultLivenessProbe()))
					})
				})
			})

			Context("missing replicas", func() {
				BeforeEach(func() {
					k8sCfg.Workload.Replicas = 0
				})

				It("returns in defaults", func() {
					defaultReplicas := config.DefaultSvcK8sConfig().Workload.Replicas
					Expect(parsedK8sCfg.Workload.Replicas).To(BeEquivalentTo(defaultReplicas))
				})
			})

			Context("missing service type", func() {
				It("returns error", func() {
					k8sconf := config.DefaultSvcK8sConfig()
					k8sconf.Service.Type = ""

					err = k8sconf.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("SvcK8sConfig.Service.Type is required"))
				})
			})

			Context("missing workload type", func() {
				It("returns error", func() {
					k8sconf := config.DefaultSvcK8sConfig()
					k8sconf.Workload.Type = ""

					err = k8sconf.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("SvcK8sConfig.Workload.Type is required"))
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
			var extensions map[string]interface{}
			var svc composego.ServiceConfig

			var parsedConf config.SvcK8sConfig
			var err error

			JustBeforeEach(func() {
				svc.Extensions = extensions
				parsedConf, err = config.SvcK8sConfigFromCompose(&svc)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("configs are empty", func() {
				BeforeEach(func() {
					extensions = make(map[string]interface{})
				})

				It("returns default when map is empty", func() {
					result, err := config.DefaultSvcK8sConfig().Merge(parsedConf)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeEquivalentTo(config.DefaultSvcK8sConfig()))
				})
			})

			Context("configs are invalid", func() {
				BeforeEach(func() {
					extensions = nil
				})

				It("returns default when map is nil", func() {
					result, err := config.DefaultSvcK8sConfig().Merge(parsedConf)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeEquivalentTo(config.DefaultSvcK8sConfig()))
				})
			})
		})
	})

})
