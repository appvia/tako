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

var _ = Describe("RenderRunner UI", func() {
	var (
		workingDir string
		runner     *kev.RenderRunner
		ui         kmd.UI
		log        kmd.UILog
		envs       []string
	)

	JustBeforeEach(func() {
		workingDir = "./testdata/detect-secrets"
		ui, log = kmd.FakeUIAndLog()
		runner = kev.NewRenderRunner(workingDir, kev.WithEnvs(envs), kev.WithUI(ui))
	})

	AfterEach(func() {
		log.Reset()
	})

	Context("Validating sources", func() {
		It("displays intended ui reporting", func() {
			err := runner.LoadProject()
			Expect(err).NotTo(HaveOccurred())
			log.Reset()

			err = runner.ValidateSources(runner.Manifest().Sources, config.SecretMatchers)
			Expect(err).NotTo(HaveOccurred())

			Expect(log.NextHeader()).To(HaveKeyWithValue("Validating compose sources...", []string{}))

			Expect(log.NextOutput()).To(HaveKeyWithValue("Detecting secrets in: testdata/detect-secrets/docker-compose.yaml", []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue(`Analysing service: db`, []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue("Warning", []string{`[Detected in service:  db]`}))

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

	Context("Validating environment overrides", func() {
		It("displays intended ui reporting", func() {
			err := runner.LoadProject()
			Expect(err).NotTo(HaveOccurred())
			log.Reset()

			err = runner.ValidateEnvSources(config.SecretMatchers)
			Expect(err).NotTo(HaveOccurred())

			Expect(log.NextHeader()).To(HaveKeyWithValue("Validating compose environment overrides...", []string{}))

			Expect(log.NextOutput()).To(HaveKeyWithValue("Detecting secrets in: testdata/detect-secrets/docker-compose.env.dev.yaml", []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue(`Analysing service: db`, []string{}))
			Expect(log.NextStep()).To(HaveKeyWithValue("Warning", []string{`[Detected in service:  db]`}))

			// Compose-go stores environment vars in a map. Therefore, the detected secrets order cannot be guaranteed.
			// So, we just ensure the UI shows the correct messaging - regardless of order.
			var detectedSecrets []map[string][]string
			for i := 0; i < 3; i++ {
				detectedSecrets = append(detectedSecrets, log.NextOutput())
			}
			Expect(detectedSecrets).To(ContainElement(HaveKeyWithValue(MatchRegexp(`AWS_ACCESS_KEY_ID`), []string{"3", "|", "log"})))
			Expect(detectedSecrets).To(ContainElement(HaveKeyWithValue(MatchRegexp(`AWS_SECRET_ACCESS_KEY`), []string{"3", "|", "log"})))
			Expect(detectedSecrets).ToNot(ContainElement(HaveKey(MatchRegexp(`CACHE_SWITCH`))))
		})
	})
})
