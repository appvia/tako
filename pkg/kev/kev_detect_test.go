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
	"github.com/appvia/kev/pkg/kev"
	"github.com/appvia/kev/pkg/kev/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

var _ = Describe("Detect", func() {
	var (
		hook       *test.Hook
		workingDir string
		dErr       error
	)

	JustBeforeEach(func() {
		hook = testutil.NewLogger(logrus.InfoLevel)
		dErr = kev.DetectSecrets(workingDir)
	})

	JustAfterEach(func() {
		hook.Reset()
	})

	BeforeEach(func() {
		workingDir = "testdata/detect-secrets"
	})

	It("should not error", func() {
		Expect(dErr).ShouldNot(HaveOccurred())
	})

	It("should log the detected secrets summary using the warning level", func() {
		Expect(testutil.GetLoggedLevel(hook)).To(Equal("warning"))
	})

	When("secrets leaked as environment variables in sources", func() {
		It("should create a detected leaks summary", func() {
			Expect(testutil.GetLoggedMsgs(hook)).Should(ContainSubstring("MYSQL_ROOT_PASSWORD"))
			Expect(testutil.GetLoggedMsgs(hook)).Should(ContainSubstring("MYSQL_USER"))
			Expect(testutil.GetLoggedMsgs(hook)).Should(ContainSubstring("MYSQL_PASSWORD"))
			Expect(testutil.GetLoggedMsgs(hook)).Should(ContainSubstring("WORDPRESS_DB_USER"))
			Expect(testutil.GetLoggedMsgs(hook)).Should(ContainSubstring("WORDPRESS_DB_PASSWORD"))
		})
	})

	When("secrets leaked as environment variables in overridden environments", func() {
		It("should create a detected leaks summary", func() {
			Expect(testutil.GetLoggedMsgs(hook)).Should(ContainSubstring("AWS_ACCESS_KEY_ID"))
			Expect(testutil.GetLoggedMsgs(hook)).Should(ContainSubstring("AWS_SECRET_ACCESS_KEY"))
			Expect(testutil.GetLoggedMsgs(hook)).ShouldNot(ContainSubstring("CACHE_SWITCH"))
		})
	})
})
