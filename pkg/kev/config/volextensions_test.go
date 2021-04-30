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
	"github.com/appvia/kev/pkg/kev/config"
	composego "github.com/compose-spec/compose-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Volume Extensions", func() {
	var (
		composeVol    composego.VolumeConfig
		composeVolExt map[string]interface{}
		composeVolCfg map[string]interface{}
	)

	Context("load", func() {
		BeforeEach(func() {
			composeVolCfg = map[string]interface{}{
				"storageClass": "ssd",
				"size":         "10Gi",
				"selector":     "my-volume-selector-label",
			}
			composeVol.Extensions = map[string]interface{}{config.K8SExtensionKey: composeVolCfg}
			composeVolExt = composeVol.Extensions[config.K8SExtensionKey].(map[string]interface{})
		})

		It("loads the extension from a compose-go volume config", func() {
			cfg, err := config.K8sVolFromCompose(&composeVol)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Map()).To(Equal(composeVolCfg))
		})

		It("compensates from missing values with defaults", func() {
			delete(composeVolExt, "storageClass")
			delete(composeVolExt, "size")

			expected := map[string]interface{}{
				"storageClass": config.DefaultVolumeStorageClass,
				"size":         config.DefaultVolumeSize,
				"selector":     "my-volume-selector-label",
			}

			cfg, err := config.K8sVolFromCompose(&composeVol)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Map()).To(Equal(expected))
		})

		It("validates values", func() {
			composeVolExt["size"] = "10Gbs"
			_, err := config.K8sVolFromCompose(&composeVol)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid, use a resource quantity format"))
		})
	})
})
