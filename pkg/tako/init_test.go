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

package tako_test

import (
	"bytes"
	"io/ioutil"
	"path/filepath"

	"github.com/appvia/tako/pkg/tako"
	"github.com/appvia/tako/pkg/tako/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InitRunner", func() {
	var (
		workingDir string
		results    tako.WritableResults
		manifest   *tako.Manifest
		rErr       error
		envs       []string
		env        *tako.Environment
	)

	JustBeforeEach(func() {
		runner := tako.NewInitRunner(workingDir, tako.WithEnvs(envs))
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
			filename := filepath.Join(workingDir, tako.ManifestFilename)
			Expect(results).To(ContainElement(tako.WritableResult{WriterTo: manifest, FilePath: filename}))
		})

		It("should contain an override environment", func() {
			filename := filepath.Join(workingDir, "compose.env.dev.yml")
			Expect(results).To(ContainElement(tako.WritableResult{WriterTo: env, FilePath: filename}))
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
			Expect(buffer.String()).To(MatchRegexp(`dev: .*compose.env.dev.yml`))
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

	Context("Sandbox dev environment", func() {
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
		When("no extensions in the sources", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/compose-yaml"
			})

			Context("marshalled and minified", func() {
				It("write out a yaml file with default minified extension data", func() {
					var buffer bytes.Buffer
					_, err := env.WriteTo(&buffer)
					Expect(err).ToNot(HaveOccurred())

					expected, err := ioutil.ReadFile("./testdata/init-default/compose-yaml/output.yaml")
					Expect(err).ToNot(HaveOccurred())

					Expect(buffer.String()).To(MatchYAML(expected))
				})
			})

			Context("services", func() {
				It("includes default config params in k8s extension", func() {
					svc, _ := env.GetService("db")

					svcK8sConfig, err := config.ParseSvcK8sConfigFromMap(svc.Extensions, config.SkipValidation())
					Expect(err).NotTo(HaveOccurred())
					Expect(svcK8sConfig.Workload.Replicas).NotTo(BeZero())
				})
			})

			Context("volumes ", func() {
				It("should include default config params in k8s extension", func() {
					vol, _ := env.GetVolume("db_data")
					k8sVol, err := config.ParseVolK8sConfigFromMap(vol.Extensions)

					Expect(err).NotTo(HaveOccurred())
					Expect(k8sVol.Size).To(Equal(config.DefaultVolumeSize))
				})
			})
		})
		When("there are extensions in the sources", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/compose-yaml-src-ext"
			})

			Context("marshalled and minified", func() {
				It("writes out a minified yaml file with sources values overriding extension defaults", func() {
					var buffer bytes.Buffer
					_, err := env.WriteTo(&buffer)
					Expect(err).ToNot(HaveOccurred())

					expected, err := ioutil.ReadFile("./testdata/init-default/compose-yaml-src-ext/output.yaml")
					Expect(err).ToNot(HaveOccurred())

					Expect(buffer.String()).To(MatchYAML(expected))
				})
			})

			Context("services", func() {
				It("includes replicas config value from sources in k8s extension", func() {
					svc, _ := env.GetService("db")

					svcK8sConfig, err := config.ParseSvcK8sConfigFromMap(svc.Extensions, config.SkipValidation())
					Expect(err).NotTo(HaveOccurred())
					Expect(svcK8sConfig.Workload.Replicas).To(Equal(10))
				})
			})

			Context("volumes ", func() {
				It("includes size config value from sources in k8s extension", func() {
					vol, _ := env.GetVolume("db_data")
					k8sVol, err := config.ParseVolK8sConfigFromMap(vol.Extensions)

					Expect(err).NotTo(HaveOccurred())
					Expect(k8sVol.Size).To(Equal("30Gi"))
				})
			})
		})
	})
})
