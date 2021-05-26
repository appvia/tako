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

package kev_test

import (
	"os"

	"github.com/appvia/kev/pkg/kev"
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
			expectedEvents := []kev.RunnerEvent{
				kev.PreEnsureFirstInit,
				kev.PostEnsureFirstInit,
				kev.PreDetectSources,
				kev.PostDetectSources,
				kev.PreValidateSources,
				kev.SecretsDetected,
				kev.PostValidateSources,
				kev.PreCreateManifest,
				kev.PostCreateManifest,
				kev.PrePrintSummary,
				kev.PostPrintSummary,
			}
			var actualEvents []kev.RunnerEvent
			handler := func(event kev.RunnerEvent, _ kev.Runner) error {
				actualEvents = append(actualEvents, event)
				return nil
			}

			err = kev.InitProjectWithOptions(wd, kev.WithEventHandler(handler))
			Expect(err).NotTo(HaveOccurred())
			Expect(actualEvents).To(ConsistOf(expectedEvents))
		})
	})

	Context("Render", func() {
		It("fires all the necessary events", func() {
			expectedEvents := []kev.RunnerEvent{
				kev.PreLoadProject,
				kev.PostLoadProject,
				kev.PreValidateSources,
				kev.SecretsDetected,
				kev.PostValidateSources,
				kev.PreValidateEnvSources,
				kev.PostValidateEnvSources,
				kev.PreReconcileEnvs,
				kev.PostReconcileEnvs,
				kev.PreRenderFromComposeToK8sManifests,
				kev.PostRenderFromComposeToK8sManifests,
				kev.PrePrintSummary,
				kev.PostPrintSummary,
			}

			var actualEvents []kev.RunnerEvent
			handler := func(event kev.RunnerEvent, _ kev.Runner) error {
				actualEvents = append(actualEvents, event)
				return nil
			}

			err = kev.InitProjectWithOptions(wd)
			Expect(err).NotTo(HaveOccurred())

			err = kev.RenderProjectWithOptions(wd, kev.WithEventHandler(handler))
			Expect(err).NotTo(HaveOccurred())
			Expect(actualEvents).To(ConsistOf(expectedEvents))
		})
	})
})
