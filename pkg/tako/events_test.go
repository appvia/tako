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

package tako_test

import (
	"os"

	"github.com/appvia/tako/pkg/tako"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Events", func() {
	var (
		composePath = "init-default/compose-yml/compose.yml"
		wd          string
		err         error
	)

	BeforeEach(func() {
		wd, err = NewTempWorkingDir(composePath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err = os.RemoveAll(wd)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Init", func() {
		It("fires all the necessary events", func() {
			expectedEvents := []tako.RunnerEvent{
				tako.PreEnsureFirstInit,
				tako.PostEnsureFirstInit,
				tako.PreDetectSources,
				tako.PostDetectSources,
				tako.PreValidateSources,
				tako.SecretsDetected,
				tako.PostValidateSources,
				tako.PreCreateManifest,
				tako.PostCreateManifest,
				tako.PrePrintSummary,
				tako.PostPrintSummary,
			}
			var actualEvents []tako.RunnerEvent
			handler := func(event tako.RunnerEvent, _ tako.Runner) error {
				actualEvents = append(actualEvents, event)
				return nil
			}

			err = tako.InitProjectWithOptions(wd, tako.WithEventHandler(handler))
			Expect(err).NotTo(HaveOccurred())
			Expect(actualEvents).To(ConsistOf(expectedEvents))
		})
	})

	Context("Render", func() {
		It("fires all the necessary events", func() {
			expectedEvents := []tako.RunnerEvent{
				tako.PreLoadProject,
				tako.PostLoadProject,
				tako.PreValidateSources,
				tako.SecretsDetected,
				tako.PostValidateSources,
				tako.PreValidateEnvSources,
				tako.PostValidateEnvSources,
				tako.PreReconcileEnvs,
				tako.PostReconcileEnvs,
				tako.PreRenderFromComposeToK8sManifests,
				tako.PostRenderFromComposeToK8sManifests,
				tako.PrePrintSummary,
				tako.PostPrintSummary,
			}

			var actualEvents []tako.RunnerEvent
			handler := func(event tako.RunnerEvent, _ tako.Runner) error {
				actualEvents = append(actualEvents, event)
				return nil
			}

			err = tako.InitProjectWithOptions(wd)
			Expect(err).NotTo(HaveOccurred())

			err = tako.RenderProjectWithOptions(wd, tako.WithEventHandler(handler))
			Expect(err).NotTo(HaveOccurred())
			Expect(actualEvents).To(ConsistOf(expectedEvents))
		})
	})

	Context("Patch", func() {
		It("fires all the necessary events", func() {
			expectedEvents := []tako.RunnerEvent{
				tako.PrePatchManifest,
				tako.PostPatchManifest,
				tako.PrePrintSummary,
				tako.PostPrintSummary,
			}
			var actualEvents []tako.RunnerEvent
			handler := func(event tako.RunnerEvent, _ tako.Runner) error {
				actualEvents = append(actualEvents, event)
				return nil
			}

			err = tako.PatchWithOptions(wd, tako.WithEventHandler(handler))
			Expect(err).NotTo(HaveOccurred())
			Expect(actualEvents).To(ConsistOf(expectedEvents))
		})

	})
})
