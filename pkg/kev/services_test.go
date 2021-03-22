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

package kev

import (
	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/converter/kubernetes"

	composego "github.com/compose-spec/compose-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceConfig", func() {
	Context("validation", func() {

		When("base labels", func() {

			// TODO: This needs a better test in a future iteration, the base labels are no longer just the required ones.
			It("fails when base labels are not present", func() {
				err := ServiceConfig{Labels: composego.Labels{
					config.LabelWorkloadReplicas: "1",
				}}.validate()
				Expect(err).Should(MatchError(ContainSubstring(config.LabelWorkloadLivenessProbeType)))

				err = ServiceConfig{Labels: composego.Labels{
					config.LabelWorkloadLivenessProbeType:    kubernetes.ProbeTypeCommand.String(),
					config.LabelWorkloadLivenessProbeCommand: "echo i'm a useless probe",
				}}.validate()
				Expect(err).Should(MatchError(ContainSubstring(config.LabelWorkloadReplicas)))

				err = ServiceConfig{Labels: composego.Labels{}}.validate()
				Expect(err).Should(HaveOccurred())
			})

			It("success if the necessary labels are present", func() {
				err := ServiceConfig{Labels: composego.Labels{
					config.LabelWorkloadLivenessProbeType:    kubernetes.ProbeTypeCommand.String(),
					config.LabelWorkloadLivenessProbeCommand: "echo i'm a useless probe",
					config.LabelWorkloadReplicas:             "1",
				}}.validate()
				Expect(err).NotTo(HaveOccurred())
			})

			It("fails when replicas label is not a number", func() {
				serviceConfig := ServiceConfig{Labels: composego.Labels{config.LabelWorkloadLivenessProbeCommand: "value"}}
				serviceConfig.Labels[config.LabelWorkloadReplicas] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})
		})

		When("other labels", func() {
			var serviceConfig ServiceConfig

			BeforeEach(func() {
				serviceConfig = ServiceConfig{
					Labels: composego.Labels{
						config.LabelWorkloadLivenessProbeType: kubernetes.ProbeTypeNone.String(),
						config.LabelWorkloadReplicas:          "1",
					}}
			})

			It("fails when unknown labels are present", func() {
				serviceConfig.Labels["kev.workload.unknown"] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("ignores non-kev labels and doesn't validate them", func() {
				serviceConfig.Labels["kubernetes.io/ingress.class"] = "external"
				Expect(serviceConfig.validate()).Should(Not(HaveOccurred()))
			})

			It("fails when component enabled label has incorrect value", func() {
				serviceConfig.Labels[config.LabelComponentEnabled] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when image pull policy label has wrong value", func() {
				serviceConfig.Labels[config.LabelWorkloadImagePullPolicy] = "wrong-value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when liveness probe disabled label has wrong value", func() {
				serviceConfig.Labels[config.LabelWorkloadImagePullPolicy] = "wrong-value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when liveness probe initial delay label is not a duration", func() {
				serviceConfig.Labels[config.LabelWorkloadLivenessProbeInitialDelay] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when liveness probe initial interval label is not a duration", func() {
				serviceConfig.Labels[config.LabelWorkloadLivenessProbeInterval] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when liveness probe retries label is not a number", func() {
				serviceConfig.Labels[config.LabelWorkloadLivenessProbeRetries] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when liveness probe timeout label is not a duration", func() {
				serviceConfig.Labels[config.LabelWorkloadLivenessProbeTimeout] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when max memory label is not a memory unit", func() {
				serviceConfig.Labels[config.LabelWorkloadMaxMemory] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when the memory label is not a memory unit", func() {
				serviceConfig.Labels[config.LabelWorkloadMemory] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when rolling update max surge label is not a number", func() {
				serviceConfig.Labels[config.LabelWorkloadRollingUpdateMaxSurge] = "value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when service account name label has the incorrect pattern", func() {
				serviceConfig.Labels[config.LabelWorkloadServiceAccountName] = " wrong-pattern"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			It("fails when workload type label has the wrong value", func() {
				serviceConfig.Labels[config.LabelWorkloadType] = "wrong-value"
				Expect(serviceConfig.validate()).Should(HaveOccurred())
			})

			Context("http probe validation", func() {
				Context("readiness", func() {
					BeforeEach(func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeType] = kubernetes.ProbeTypeNone.String()
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeType] = kubernetes.ProbeTypeHTTP.String()
					})

					It("fails when http probe has missing path", func() {
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeHTTPPort] = "8080"

						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("required"))
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadReadinessProbeHTTPPath))
					})

					It("fails when http probe has missing port", func() {
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeHTTPPath] = "/status"

						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("required"))
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadReadinessProbeHTTPPort))
					})

					It("succeeds when all values are provided", func() {
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeHTTPPath] = "/status"
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeHTTPPort] = "8080"

						err := serviceConfig.validate()
						Expect(err).To(Succeed())
					})
				})

				Context("liveness", func() {
					BeforeEach(func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeType] = kubernetes.ProbeTypeHTTP.String()
					})

					It("fails when http probe has missing path", func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeHTTPPort] = "8080"

						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("required"))
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadLivenessProbeHTTPPath))
					})

					It("fails when http probe has missing port", func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeHTTPPath] = "/status"

						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("required"))
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadLivenessProbeHTTPPort))
					})

					It("succeeds when all values are provided", func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeHTTPPort] = "/status"
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeHTTPPath] = "8080"

						err := serviceConfig.validate()
						Expect(err).To(Succeed())
					})
				})
			})

			Context("tcp probe validation", func() {
				Context("readiness", func() {
					BeforeEach(func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeType] = kubernetes.ProbeTypeNone.String()
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeType] = kubernetes.ProbeTypeTCP.String()
					})

					It("fails when tcp probe has missing port", func() {
						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("required"))
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadReadinessProbeTCPPort))
					})

					It("succeeds when port is provided", func() {
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeTCPPort] = "8080"

						err := serviceConfig.validate()
						Expect(err).To(Succeed())
					})
				})
				Context("liveness", func() {
					BeforeEach(func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeType] = kubernetes.ProbeTypeTCP.String()
					})

					It("fails when tcp probe has missing port", func() {
						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("required"))
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadLivenessProbeTCPPort))
					})

					It("succeeds when port is provided", func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeTCPPort] = "8080"

						err := serviceConfig.validate()
						Expect(err).To(Succeed())
					})
				})
			})

			Context("command probe validation", func() {
				Context("rediness probes", func() {
					BeforeEach(func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeType] = kubernetes.ProbeTypeNone.String()
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeType] = kubernetes.ProbeTypeCommand.String()
					})

					It("fails when command probe has missing properties", func() {
						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("required"))
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadReadinessProbeCommand))
					})
				})

				Context("liveness probe", func() {
					BeforeEach(func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeType] = kubernetes.ProbeTypeCommand.String()
					})

					It("fails when command probe has missing properties", func() {
						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("required"))
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadLivenessProbeCommand))
					})
				})
			})

			Context("probe type enum validation", func() {
				Context("readiness", func() {
					BeforeEach(func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeType] = kubernetes.ProbeTypeNone.String()
					})

					It("fails when type is misspelled", func() {
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeType] = "wrong-value"

						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadReadinessProbeType))
						Expect(err.Error()).To(ContainSubstring("none"))
					})

					It("succeeds if set to none", func() {
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeType] = kubernetes.ProbeTypeNone.String()
						serviceConfig.Labels[config.LabelWorkloadReadinessProbeCommand] = "i'm just a leftover"

						err := serviceConfig.validate()
						Expect(err).To(Succeed())
					})
				})

				Context("liveness", func() {
					It("fails when type is misspelled", func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeType] = "wrong-value"

						err := serviceConfig.validate()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring(config.LabelWorkloadLivenessProbeType))
						Expect(err.Error()).To(ContainSubstring("none"))
					})

					It("succeeds if set to none", func() {
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeType] = kubernetes.ProbeTypeNone.String()
						serviceConfig.Labels[config.LabelWorkloadLivenessProbeCommand] = "i'm just a leftover"

						err := serviceConfig.validate()
						Expect(err).To(Succeed())
					})
				})
			})

		})
	})
})
