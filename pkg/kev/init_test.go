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
	"io/ioutil"
	"path"

	"github.com/appvia/kev/pkg/kev"
	"github.com/appvia/kev/pkg/kev/config"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InitRunner", func() {
	var (
		workingDir string
		results    kev.WritableResults
		manifest   *kev.Manifest
		rErr       error
		envs       []string
		env        *kev.Environment
	)

	JustBeforeEach(func() {
		runner := kev.NewInitRunner(workingDir, kev.WithEnvs(envs))
		if results, rErr = runner.Run(); rErr == nil {
			manifest = runner.Manifest()
			env, _ = manifest.GetEnvironment("dev")
		}
	})

	Context("Run results", func() {
		BeforeEach(func() {
			workingDir = "./testdata/init-default/compose-yml"
		})

		It("should contain all required files", func() {
			Expect(results).To(HaveLen(2))
		})

		It("should contain a manifest file", func() {
			filename := path.Join(workingDir, kev.ManifestFilename)
			Expect(results).To(ContainElement(kev.WritableResult{WriterTo: manifest, FilePath: filename}))
		})

		It("should contain an override environment", func() {
			filename := path.Join(workingDir, "compose.kev.dev.yml")
			Expect(results).To(ContainElement(kev.WritableResult{WriterTo: env, FilePath: filename}))
		})
	})

	Context("Created manifest", func() {
		BeforeEach(func() {
			workingDir = "./testdata/init-default/compose-yml"
		})

		It("should contain an id attribute", func() {
			Expect(manifest.Id).ToNot(BeEmpty())
		})

		It("should write out a yaml file with manifest data", func() {
			var buffer bytes.Buffer
			_, err := manifest.WriteTo(&buffer)
			Expect(err).ToNot(HaveOccurred())

			Expect(buffer.String()).To(MatchRegexp(`id: [a-z0-9]+`))
			Expect(buffer.String()).To(ContainSubstring("compose:"))
			Expect(buffer.String()).To(MatchRegexp(`.*- .*compose.yml`))
			Expect(buffer.String()).To(ContainSubstring("environments:"))
			Expect(buffer.String()).To(MatchRegexp(`dev: .*compose.kev.dev.yml`))
		})
	})

	When("No alternate compose files supplied", func() {
		Context("and without any docker-compose file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata"
			})

			It("should error", func() {

				Expect(rErr).To(HaveOccurred())
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
				Expect(rErr).NotTo(HaveOccurred())
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
				Expect(rErr).NotTo(HaveOccurred())
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
				Expect(rErr).NotTo(HaveOccurred())
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
				Expect(rErr).NotTo(HaveOccurred())
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
				Expect(rErr).NotTo(HaveOccurred())
			})
		})
	})

	Context("sandbox dev environment", func() {
		When("dev is not supplied as an environment", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/compose-yaml"
				envs = []string{"stage"}
			})
			It("is created implicitly", func() {
				Expect(rErr).NotTo(HaveOccurred())
				Expect(manifest.GetEnvironmentsNames()).Should(ConsistOf("dev", "stage"))
			})
		})

		When("dev is supplied as an environment", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/compose-yaml"
				envs = []string{"stage", "dev"}
			})
			It("is only created once", func() {
				Expect(rErr).NotTo(HaveOccurred())
				Expect(manifest.GetEnvironmentsNames()).Should(ConsistOf("dev", "stage"))
			})
		})
	})

	Context("Created environment overrides", func() {
		BeforeEach(func() {
			workingDir = "./testdata/init-default/compose-yaml"
		})

		Context("marshalling", func() {
			It("should write out a yaml file with manifest data", func() {
				var buffer bytes.Buffer
				_, err := env.WriteTo(&buffer)
				Expect(err).ToNot(HaveOccurred())

				bs, err := ioutil.ReadFile("./testdata/init-default/compose-yaml/output.yaml")
				Expect(err).ToNot(HaveOccurred())

				expected := string(bs)
				Expect(cmp.Diff(buffer.String(), expected)).To(BeEmpty())
				Expect(buffer.String()).To(Equal(expected))
			})
		})

		Context("with services", func() {
			It("should include default config params", func() {
				svc, _ := env.GetService("db")

				k8sconf, err := config.ParseK8SCfgFromMap(svc.Extensions)
				Expect(err).NotTo(HaveOccurred())

				Expect(svc.GetLabels()).To(BeEmpty())
				Expect(k8sconf.Workload.LivenessProbe).To(Equal(config.DefaultLivenessProbe()))
				Expect(k8sconf.Workload.Replicas).NotTo(BeZero())
			})
		})
		Context("with volumes", func() {
			It("should include a subset of labels as config params", func() {
				vol, _ := env.GetVolume("db_data")
				Expect(vol.Labels).To(HaveLen(1))
				Expect(vol.Labels).To(HaveKey(config.LabelVolumeSize))
			})
		})
	})
})
