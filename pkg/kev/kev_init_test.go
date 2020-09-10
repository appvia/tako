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

var _ = Describe("Init", func() {
	var (
		workingDir string
		manifest   *kev.Manifest
		mErr       error
	)

	JustBeforeEach(func() {
		manifest, mErr = kev.Init([]string{}, []string{}, workingDir)
	})

	Context("with no alternate compose files supplied", func() {
		Context("and without any docker-compose file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata"
			})

			It("should error", func() {
				Expect(mErr).To(HaveOccurred())
			})
		})

		Context("and with a compose.yml file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/compose-yml"
			})

			It("should initialise the manifest using the file", func() {
				Expect(manifest.GetSourcesFiles()).To(Equal([]string{"testdata/init-default/compose-yml/compose.yml"}))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("and with a compose.yaml file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/compose-yaml"
			})

			It("should initialise the manifest using the file", func() {
				Expect(manifest.GetSourcesFiles()).To(Equal([]string{"testdata/init-default/compose-yaml/compose.yaml"}))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("and with a docker-compose.yaml file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/docker-compose-yaml"
			})

			It("should initialise the manifest using the file", func() {
				Expect(manifest.GetSourcesFiles()).To(Equal([]string{"testdata/init-default/docker-compose-yaml/docker-compose.yaml"}))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("and with a docker-compose.yaml file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/docker-compose-yml"
			})

			It("should initialise the manifest using the file", func() {
				Expect(manifest.GetSourcesFiles()).To(Equal([]string{"testdata/init-default/docker-compose-yml/docker-compose.yml"}))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("and with a docker-compose.yml file & optional override file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/docker-compose-override"
			})

			It("should initialise the manifest using both files", func() {
				expected := []string{"" +
					"testdata/init-default/docker-compose-override/docker-compose.yaml",
					"testdata/init-default/docker-compose-override/docker-compose.override.yaml",
				}
				Expect(manifest.GetSourcesFiles()).To(Equal(expected))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})
	})
})
