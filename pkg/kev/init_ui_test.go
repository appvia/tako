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
	"github.com/appvia/kev/pkg/kev"
	"github.com/appvia/kev/pkg/kev/config"
	kmd "github.com/appvia/komando"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InitRunner UI", func() {
	var (
		workingDir string
		runner     *kev.InitRunner
		ui         kmd.UI
		log        kmd.UILog
		envs       []string
	)

	JustBeforeEach(func() {
		workingDir = "./testdata/init-default/compose-yml"
		ui, log = kmd.FakeUIAndLog()
		runner = kev.NewInitRunner(workingDir, kev.WithEnvs(envs), kev.WithUI(ui))
	})

	AfterEach(func() {
		log.Reset()
	})

	Context("Verifying the project", func() {
		It("displays intended ui reporting", func() {
			err := runner.EnsureFirstInit()
			Expect(err).NotTo(HaveOccurred())

			Expect(log.NextHeader()).To(HaveKeyWithValue("Verifying project...", []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue("Ensuring this project has not already been initialised", []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue(kmd.LogStepSuccess, []string{}))
		})
	})

	Context("Detecting compose sources", func() {
		It("displays intended ui reporting", func() {
			_, err := runner.DetectSources()
			Expect(err).NotTo(HaveOccurred())

			Expect(log.NextHeader()).To(HaveKeyWithValue("Detecting compose sources...", []string{}))

			Expect(log.NextStep()).To(HaveKeyWithValue("Scanning for compose configuration", []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue(kmd.LogStepSuccess, []string{}))

			Expect(log.NextStep()).To(HaveKeyWithValue("Using: testdata/init-default/compose-yml/compose.yml", []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue(kmd.LogStepSuccess, []string{}))
		})
	})

	Context("Validating sources", func() {
		It("displays intended ui reporting", func() {
			sources, err := runner.DetectSources()
			Expect(err).NotTo(HaveOccurred())
			log.Reset()

			err = runner.ValidateSources(sources, config.SecretMatchers)
			Expect(err).NotTo(HaveOccurred())

			Expect(log.NextHeader()).To(HaveKeyWithValue("Validating compose sources...", []string{}))

			Expect(log.NextOutput()).To(HaveKeyWithValue("Detecting secrets in: testdata/init-default/compose-yml/compose.yml", []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue("Analysing service: db", []string{}))

			// Compose-go stores environment vars in a map. Therefore, the detected secrets order cannot be guaranteed.
			// So, we just ensure the UI shows the correct messaging - regardless of order.
			var detectedSecrets []map[string][]string
			for i := 0; i < 3; i++ {
				detectedSecrets = append(detectedSecrets, log.NextOutput())
			}
			Expect(detectedSecrets).To(ContainElement(HaveKeyWithValue("env var [MYSQL_USER] - Contains word: user", []string{"3", "|", "log"})))
			Expect(detectedSecrets).To(ContainElement(HaveKeyWithValue("env var [MYSQL_PASSWORD] - Contains word: password", []string{"3", "|", "log"})))
			Expect(detectedSecrets).To(ContainElement(HaveKeyWithValue("env var [MYSQL_ROOT_PASSWORD] - Contains word: password", []string{"3", "|", "log"})))
		})
	})

	Context("Creating deployment environments", func() {
		It("displays intended ui reporting", func() {
			sources, err := runner.DetectSources()
			Expect(err).NotTo(HaveOccurred())
			log.Reset()

			err = runner.CreateManifestAndEnvironmentOverrides(sources)
			Expect(err).NotTo(HaveOccurred())

			Expect(log.NextHeader()).To(HaveKeyWithValue("Creating deployment environments...", []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue("Creating the dev sandbox env file: testdata/init-default/compose-yml/compose.env.dev.yml", []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue("Success", []string{}))
		})
	})

	Context("Detecting Skaffold settings", func() {
		It("displays intended ui reporting", func() {
			_, err := runner.Run()
			Expect(err).NotTo(HaveOccurred())

			Expect(log.LastHeader()).To(HaveKeyWithValue("Detecting Skaffold settings...", []string{}))
			Expect(log.LastOutput()).To(HaveKeyWithValue("Skipping - no Skaffold options detected", []string{}))
		})
	})
})
