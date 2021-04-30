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
			k8s config.K8sSvc
			err error

			parsedCfg config.K8sSvc
			svc       composego.ServiceConfig
		)

		BeforeEach(func() {
			k8s = config.K8sSvc{}
		})

		JustBeforeEach(func() {
			m, err := k8s.ToMap()
			Expect(err).NotTo(HaveOccurred())
			svc.Extensions = map[string]interface{}{
				config.K8SExtensionKey: m,
			}

			parsedCfg, err = config.K8sSvcFromCompose(&svc)
			Expect(err).NotTo(HaveOccurred())

		})

		Context("works with defaults", func() {
			BeforeEach(func() {
				k8s.Workload.Type = "Deployment"
				k8s.Workload.Replicas = 10
				k8s.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()
			})

			It("creates the config using defaults when the mandatory properties are present", func() {
				expectedLiveness := config.DefaultLivenessProbe()
				expectedLiveness.Type = config.ProbeTypeNone.String()

				Expect(parsedCfg.Workload.Replicas).To(Equal(10))
				Expect(parsedCfg.Workload.LivenessProbe).To(BeEquivalentTo(expectedLiveness))
				Expect(parsedCfg.Workload.ReadinessProbe).To(BeEquivalentTo(config.DefaultReadinessProbe()))
			})
		})

		When("there is no k8s extension present", func() {
			Context("Without RequirePresent configuration", func() {
				It("does not fail validations", func() {
					parsedCfg, err = config.K8sSvcFromCompose(&svc)
					Expect(err).ToNot(HaveOccurred())
					Expect(parsedCfg).NotTo(BeNil())

					Expect(parsedCfg.Workload.Replicas).To(Equal(config.DefaultReplicaNumber))
					Expect(parsedCfg.Workload.LivenessProbe).To(BeEquivalentTo(config.DefaultLivenessProbe()))
					Expect(parsedCfg.Workload.ReadinessProbe).To(BeEquivalentTo(config.DefaultReadinessProbe()))
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

					parsedCfg, err = config.K8sSvcFromCompose(&svc)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns defaults", func() {
					Expect(parsedCfg).To(BeEquivalentTo(config.DefaultK8sSvc()))
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

					parsedCfg, err = config.K8sSvcFromCompose(&svc)
					Expect(err).NotTo(HaveOccurred())
				})

				When("workload is invalid", func() {
					It("it is ignored and returns defaults", func() {
						Expect(parsedCfg).To(BeEquivalentTo(config.DefaultK8sSvc()))
					})
				})
			})

			Context("missing liveness probe type", func() {
				BeforeEach(func() {
					k8s.Workload.Type = config.DefaultWorkload
					k8s.Workload.Replicas = 10
				})

				When("liveness probe type not provided", func() {
					It("return default probe", func() {
						Expect(parsedCfg.Workload.LivenessProbe).To(Equal(config.DefaultLivenessProbe()))
					})
				})
			})

			Context("missing replicas", func() {
				BeforeEach(func() {
					k8s.Workload.Type = config.DefaultWorkload
					k8s.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()
				})

				It("returns in defaults", func() {
					k8sconf := config.DefaultK8sSvc()
					k8sconf.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()

					Expect(parsedCfg).To(BeEquivalentTo(k8sconf))
				})
			})

			Context("missing service type", func() {
				It("returns error", func() {
					k8sconf := config.DefaultK8sSvc()
					k8sconf.Service.Type = ""

					err = k8sconf.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8sSvc.Service.Type is required"))
				})
			})

			Context("missing workload type", func() {
				It("returns error", func() {
					k8sconf := config.DefaultK8sSvc()
					k8sconf.Workload.Type = ""

					err = k8sconf.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8sSvc.Workload.Type is required"))
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
			k8sBase := config.DefaultK8sSvc()
			var k8sTarget config.K8sSvc
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

			var parsedConf config.K8sSvc
			var err error

			JustBeforeEach(func() {
				svc.Extensions = extensions
				parsedConf, err = config.K8sSvcFromCompose(&svc)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("configs are empty", func() {
				BeforeEach(func() {
					extensions = make(map[string]interface{})
				})

				It("returns default when map is empty", func() {
					result, err := config.DefaultK8sSvc().Merge(parsedConf)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeEquivalentTo(config.DefaultK8sSvc()))
				})
			})

			Context("configs are invalid", func() {
				BeforeEach(func() {
					extensions = nil
				})

				It("returns default when map is nil", func() {
					result, err := config.DefaultK8sSvc().Merge(parsedConf)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeEquivalentTo(config.DefaultK8sSvc()))
				})
			})
		})
	})

})
