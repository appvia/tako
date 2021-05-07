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
	"github.com/appvia/kev/pkg/kev/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	var workingDir = "testdata/merge"

	Describe("MergeEnvIntoSources", func() {
		source, _ := kev.NewComposeProject([]string{workingDir + "/docker-compose.yaml"})

		Context("pre merge", func() {
			It("confirms there are no service labels", func() {
				sourceSvc, _ := source.GetService("db")
				Expect(sourceSvc.Labels).To(HaveLen(0))
			})

			It("confirms env var overrides", func() {
				sourceSvc, _ := source.GetService("db")
				overrideMeWithVal := "value"
				Expect(sourceSvc.Environment["OVERRIDE_ME_EMPTY"]).To(BeNil())
				Expect(sourceSvc.Environment["OVERRIDE_ME_WITH_VAL"]).To(Equal(&overrideMeWithVal))
			})

			It("confirms there are no volume labels", func() {
				sourceVol := source.Volumes["db_data"]
				Expect(sourceVol.Labels).To(HaveLen(0))
			})
		})

		Context("post merge", func() {
			var (
				merged   *kev.ComposeProject
				mergeErr error
				env      *kev.Environment
				manifest *kev.Manifest
			)

			BeforeEach(func() {
				var err error
				manifest, err = kev.LoadManifest(workingDir)
				Expect(err).NotTo(HaveOccurred())

				_, err = manifest.CalculateSourcesBaseOverride()
				Expect(err).NotTo(HaveOccurred())

				env, err = manifest.GetEnvironment("dev")
				Expect(err).NotTo(HaveOccurred())

				merged, mergeErr = manifest.MergeEnvIntoSources(env)
				Expect(mergeErr).NotTo(HaveOccurred())
			})

			It("merged the environment labels into sources", func() {
				var err error
				mergedSvc, err := merged.GetService("db")
				Expect(err).NotTo(HaveOccurred())
				envSvc, err := env.GetService("db")
				Expect(err).NotTo(HaveOccurred())
				Expect(mergedSvc.Labels).To(Equal(envSvc.Labels))
			})

			It("merged the environment extensions into sources", func() {
				sources, err := manifest.SourcesToComposeProject()
				Expect(err).NotTo(HaveOccurred())

				srcSvc, err := sources.GetService("db")
				Expect(err).NotTo(HaveOccurred())

				mergedSvc, err := merged.GetService("db")
				Expect(err).NotTo(HaveOccurred())

				envSvc, err := env.GetService("db")
				Expect(err).NotTo(HaveOccurred())

				Expect(srcSvc.Extensions).To(HaveLen(2))
				Expect(mergedSvc.Extensions).To(HaveLen(3))
				Expect(mergedSvc.Extensions["x-other-extension"]).To(Equal(envSvc.Extensions["x-other-extension"]))

				mergedSvcAnExt := mergedSvc.Extensions["x-an-extension"].(map[string]interface{})
				envSvcAnExt := envSvc.Extensions["x-an-extension"].(map[string]interface{})

				Expect(mergedSvcAnExt["key"]).To(Equal("value"))
				Expect(mergedSvcAnExt["override-key"]).To(Equal(envSvcAnExt["override-key"]))

				k8sconf, err := config.ParseSvcK8sConfigFromMap(mergedSvc.Extensions)
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sconf.Workload.LivenessProbe.Type).To(Equal(config.ProbeTypeExec.String()))
				Expect(k8sconf.Workload.LivenessProbe.Exec.Command).To(Equal([]string{"echo", "I'm a useless check"}))
			})

			It("merged the environment env var overrides into sources", func() {
				mergedSvc, _ := merged.GetService("db")
				envSvc, _ := env.GetService("db")
				Expect(mergedSvc.Environment["OVERRIDE_ME_EMPTY"]).To(Equal(envSvc.Environment["OVERRIDE_ME_EMPTY"]))
				Expect(mergedSvc.Environment["OVERRIDE_ME_WITH_VAL"]).To(Equal(envSvc.Environment["OVERRIDE_ME_WITH_VAL"]))
			})

			It("merged the environment volume labels into sources", func() {
				mergedVol := merged.Volumes["db_data"]
				envVol, _ := env.GetVolume("db_data")
				Expect(mergedVol.Labels).To(Equal(envVol.Labels))
			})

			It("should not error", func() {
				Expect(mergeErr).NotTo(HaveOccurred())
			})
		})
	})

	Describe("GetEnvironmentFileNameTemplate", func() {

		var (
			m     *kev.Manifest
			files []string
		)

		JustBeforeEach(func() {
			m = &kev.Manifest{
				Sources: &kev.Sources{
					Files: files,
				},
			}
		})

		Context("with mutiple source compose files", func() {
			BeforeEach(func() {
				files = []string{
					"my-custom-docker-compose.yaml",
					"my-custom-docker-compose.override.yaml",
				}
			})

			It("returns environment file name template as expected", func() {
				Expect(m.GetEnvironmentFileNameTemplate()).To(Equal("my-custom-docker-compose.%s.%s.yaml"))
			})
		})

		Context("with a single source compoe file", func() {
			BeforeEach(func() {
				files = []string{
					"compose.yml",
				}
			})

			It("returns environment file name template as expected", func() {
				Expect(m.GetEnvironmentFileNameTemplate()).To(Equal("compose.%s.%s.yml"))
			})
		})
	})

	Describe("LoadManifest", func() {
		Context("validation", func() {
			It("fails for invalid loaded environment", func() {
				workingDir := "testdata/validation"
				_, err := kev.LoadManifest(workingDir)
				Expect(err).Should(HaveOccurred())
			})
		})
	})
})
