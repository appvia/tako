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

package kev_test

import (
	"bytes"

	"github.com/appvia/kev/pkg/kev"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Detect", func() {
	var (
		workingDir string
		reporter   bytes.Buffer
		dErr       error
	)

	JustBeforeEach(func() {
		dErr = kev.DetectSecrets(workingDir, &reporter)
	})

	BeforeEach(func() {
		workingDir = "testdata/detect-secrets"
		reporter = bytes.Buffer{}
	})

	It("should not error", func() {
		Expect(dErr).ShouldNot(HaveOccurred())
	})

	When("secrets leaked as environment variables in sources", func() {
		It("should create a detected leaks summary", func() {
			Expect(reporter.String()).Should(ContainSubstring("MYSQL_ROOT_PASSWORD"))
			Expect(reporter.String()).Should(ContainSubstring("MYSQL_USER"))
			Expect(reporter.String()).Should(ContainSubstring("MYSQL_PASSWORD"))
			Expect(reporter.String()).Should(ContainSubstring("WORDPRESS_DB_USER"))
			Expect(reporter.String()).Should(ContainSubstring("WORDPRESS_DB_PASSWORD"))
		})
	})

	When("secrets leaked as environment variables in overridden environments", func() {
		It("should create a detected leaks summary", func() {
			Expect(reporter.String()).Should(ContainSubstring("AWS_ACCESS_KEY_ID"))
			Expect(reporter.String()).Should(ContainSubstring("AWS_SECRET_ACCESS_KEY"))
			Expect(reporter.String()).ShouldNot(ContainSubstring("CACHE_SWITCH"))
		})
	})
})
