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
					config.LabelWorkloadLivenessProbeType: kubernetes.ProbeTypeCommand.String(),
				}}.validate()
				Expect(err).Should(MatchError(ContainSubstring(config.LabelWorkloadReplicas)))

				err = ServiceConfig{Labels: composego.Labels{}}.validate()
				Expect(err).Should(HaveOccurred())

				err = ServiceConfig{Labels: composego.Labels{
					config.LabelWorkloadLivenessProbeType: kubernetes.ProbeTypeCommand.String(),
					config.LabelWorkloadReplicas:          "1",
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
		})
	})
})
