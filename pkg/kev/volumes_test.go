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
	composego "github.com/compose-spec/compose-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("VolumeConfig", func() {
	Context("validation", func() {

		When("base labels", func() {
			It("fails when base labels are not present", func() {
				err := VolumeConfig{Labels: composego.Labels{}}.validate()
				Expect(err).Should(MatchError(ContainSubstring(config.BaseVolumeLabels[0])))
			})

			It("fails when volume size label is not a memory unit", func() {
				volumeConfig := VolumeConfig{Labels: composego.Labels{config.LabelVolumeSize: "value"}}
				Expect(volumeConfig.validate()).Should(HaveOccurred())
			})
		})

		When("other labels", func() {
			var volumeConfig VolumeConfig

			BeforeEach(func() {
				volumeConfig = VolumeConfig{
					Labels: composego.Labels{config.LabelVolumeSize: "10Gi"}}
			})

			It("fails when unknown labels are present", func() {
				volumeConfig.Labels["kev.volume.unknown"] = "value"
				Expect(volumeConfig.validate()).Should(HaveOccurred())
			})

			It("fails when storage class label has the incorrect pattern", func() {
				volumeConfig.Labels[config.LabelVolumeStorageClass] = " wrong-pattern"
				Expect(volumeConfig.validate()).Should(HaveOccurred())
			})

			It("fails when volume selector label has the incorrect pattern", func() {
				volumeConfig.Labels[config.LabelVolumeSelector] = " wrong-pattern"
				Expect(volumeConfig.validate()).Should(HaveOccurred())
			})
		})
	})
})
