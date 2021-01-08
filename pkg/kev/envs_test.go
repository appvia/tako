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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment", func() {
	var (
		env *kev.Environment
	)

	Describe("Extensions", func() {
		JustBeforeEach(func() {
			manifest, err := kev.LoadManifest("testdata/in-cluster-wordpress")
			Expect(err).ToNot(HaveOccurred())

			env, err = manifest.GetEnvironment("dev")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("update extensions", func() {
			It("adds new extensions", func() {
				env.UpdateExtensions("db", map[string]interface{}{
					"x-ext-one": "test-one",
					"x-ext-two": "test-two",
				})
				envSvc, _ := env.GetService("db")
				Expect(envSvc.Extensions).To(HaveKeyWithValue("x-ext-one", "test-one"))
				Expect(envSvc.Extensions).To(HaveKeyWithValue("x-ext-two", "test-two"))
			})

			It("overwrites existing extensions", func() {
				env.UpdateExtensions("db", map[string]interface{}{
					"x-ext-name": "test",
				})
				env.UpdateExtensions("db", map[string]interface{}{
					"x-ext-name": "changed",
				})
				envSvc, _ := env.GetService("db")
				Expect(envSvc.Extensions).To(HaveKeyWithValue("x-ext-name", "changed"))
			})

			It("errors for unknown services", func() {
				err := env.UpdateExtensions("unknown", map[string]interface{}{
					"x-ext-one": "test-one",
				})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("remove extensions", func() {
			It("deletes an existing extension", func() {
				env.UpdateExtensions("db", map[string]interface{}{
					"x-ext-one": "test-one",
					"x-ext-two": "test-two",
				})
				env.RemoveExtension("db", "x-ext-one")
				envSvc, _ := env.GetService("db")
				Expect(envSvc.Extensions).ToNot(HaveKeyWithValue("x-ext-one", "test-one"))
				Expect(envSvc.Extensions).To(HaveKeyWithValue("x-ext-two", "test-two"))
			})

			It("errors for unknown services", func() {
				err := env.RemoveExtension("unknown", "key")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
