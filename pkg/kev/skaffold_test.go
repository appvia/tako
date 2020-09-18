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
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/appvia/kev/pkg/kev"
	"github.com/appvia/kev/pkg/kev/converter/kubernetes"
	"github.com/appvia/kev/pkg/kev/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus/hooks/test"
)

var hook *test.Hook

func init() {
	// Use mem buffer in test instead of Stdout
	logBuffer := &bytes.Buffer{}
	log.SetOutput(logBuffer)
	hook = test.NewLocal(log.GetLogger())
}

var _ = Describe("Skaffold", func() {

	Describe("BaseSkaffoldManifest", func() {
		It("returns base skaffold", func() {
			Expect(kev.BaseSkaffoldManifest()).To(Equal(
				&kev.SkaffoldManifest{
					APIVersion: latest.Version,
					Kind:       "Config",
					Metadata: latest.Metadata{
						Name: "KevApp",
					},
				},
			))
		})
	})

	Describe("SetProfiles", func() {

		When("environment names have been specified", func() {

			envs := []string{"dev", "uat", "prod"}
			manifest := kev.BaseSkaffoldManifest()
			manifest.SetProfiles(envs)

			It("returns skaffold profiles as expected", func() {
				Expect(manifest.Profiles).ToNot(BeEmpty())
				Expect(manifest.Profiles).To(HaveLen(3))
			})

			It("generates correct pipeline Deploy section for each environment", func() {
				for i, p := range manifest.Profiles {
					Expect(p.Deploy).To(Equal(latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: &latest.KubectlDeploy{
								Manifests: []string{
									filepath.Join(kubernetes.MultiFileSubDir, envs[i], "*"),
								},
							},
						},
						KubeContext: envs[i] + "-context",
					}))
				}
			})

			It("generates correct pipeline Build section for each environment", func() {
				disabled := false

				for _, p := range manifest.Profiles {
					Expect(p.Build).To(Equal(latest.BuildConfig{
						BuildType: latest.BuildType{
							LocalBuild: &latest.LocalBuild{
								Push: &disabled,
							},
						},
						TagPolicy: latest.TagPolicy{
							GitTagger: &latest.GitTagger{
								Variant: "Tags",
							},
						},
					}))
				}
			})

			It("generates correct pipeline Test section for each environment", func() {
				for _, p := range manifest.Profiles {
					Expect(p.Test).To(Equal([]*latest.TestCase{}))
				}
			})

			It("generates correct pipeline PortForward section for each environment", func() {
				for _, p := range manifest.Profiles {
					Expect(p.PortForward).To(Equal([]*latest.PortForwardResource{}))
				}
			})
		})

		When("there are no environments", func() {

			envs := []string{}
			manifest := kev.BaseSkaffoldManifest()
			manifest.SetProfiles(envs)

			It("falls back to default `dev` environment only", func() {
				Expect(manifest.Profiles).ToNot(BeEmpty())
				Expect(manifest.Profiles).To(HaveLen(1))
				Expect(manifest.Profiles[0].Name).To(Equal("dev-env"))
			})
		})

		When("profiles for specified environment already exists in skaffold profiles", func() {

			envs := []string{"dev", "uat", "prod"}
			manifest := kev.BaseSkaffoldManifest()
			manifest.SetProfiles(envs)

			BeforeEach(func() {
				// explicitly triggering another SetProfiles(envs)
				manifest.SetProfiles(envs)
			})

			It("doesn't add existing environment profile again", func() {
				Expect(manifest.Profiles).To(HaveLen(3))
			})
		})

	})

	Describe("AdditionalProfiles", func() {

		manifest := kev.BaseSkaffoldManifest()
		manifest.AdditionalProfiles()

		It("adds all additional profiles", func() {
			Expect(manifest.Profiles).To(HaveLen(4))
		})

		It("adds additional profiles to the skaffold manifest with name containing kev defined prefix", func() {
			Expect(manifest.Profiles).To(ContainElement(latest.Profile{
				Name: "zz-minikube",
				Activation: []latest.Activation{
					{
						KubeContext: "minikube",
					},
				},
			}))
		})

		When("profile of the same name already exists in skaffold profiles", func() {

			BeforeEach(func() {
				// explicitly triggering another AdditionalProfiles
				manifest.AdditionalProfiles()
			})

			It("doesn't add existing additional profiles again", func() {
				Expect(manifest.Profiles).To(HaveLen(4))
			})
		})
	})

	Describe("UpdateProfiles", func() {
		var manifest *kev.SkaffoldManifest

		envName := "test"

		BeforeEach(func() {
			envs := []string{envName}
			manifest = kev.BaseSkaffoldManifest()
			manifest.SetProfiles(envs)
		})

		Context("for skaffold profile names matching rendereded environment", func() {

			When("rendered manifests output path is a directory", func() {
				outputPath := "testdata" // point at any existing directory for test!

				envToOutputPath := map[string]string{
					envName: outputPath,
				}

				It("updates the matching profile with new manifests path selecting all the files in that directory", func() {
					manifest.UpdateProfiles(envToOutputPath)
					Expect(manifest.Profiles[0].Deploy.KubectlDeploy.Manifests).To(ContainElement(filepath.Join(outputPath, "*")))
				})
			})

			When("rendered manifests output path is a single file", func() {
				outputPath := "testdata/init-default/skaffold/skaffold.yaml" // point at any existing file for test!

				envToOutputPath := map[string]string{
					envName: outputPath,
				}

				It("updates the matching profile with new manifests path pointing at specific file", func() {
					manifest.UpdateProfiles(envToOutputPath)
					Expect(manifest.Profiles[0].Deploy.KubectlDeploy.Manifests).To(ContainElement(outputPath))
				})
			})

		})

		Context("when skaffold profile names don't match rendered enviornment", func() {
			envToOutputPath := map[string]string{
				"anotherEnv": "a/new/manifests/path",
			}

			It("profile manifests path should remain unchanged", func() {
				manifest.UpdateProfiles(envToOutputPath)
				Expect(manifest.Profiles[0].Deploy.KubectlDeploy.Manifests).To(ContainElement("k8s/test/*"))
			})
		})
	})

	Describe("AddProfiles", func() {
		var (
			skaffoldManifest          *kev.SkaffoldManifest
			existingSkaffoldPath      string
			err                       error
			includeAdditionalProfiles bool
		)

		BeforeEach(func() {
			existingSkaffoldPath = "testdata/init-default/skaffold/skaffold.yaml"
			includeAdditionalProfiles = false
		})

		When("skaffold profile doesn't already exist in the manifest", func() {
			// Note, example skaffold already contains dev environment profile
			BeforeEach(func() {
				envs := []string{"prod"}
				skaffoldManifest, err = kev.AddProfiles(existingSkaffoldPath, envs, includeAdditionalProfiles)
			})

			It("adds that profile to skaffold manifest", func() {
				Expect(skaffoldManifest.ProfilesNames()).To(ContainElement("dev-env"))
				Expect(skaffoldManifest.ProfilesNames()).To(ContainElement("prod-env"))
				Expect(skaffoldManifest.Profiles).To(HaveLen(2))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("skaffold profile of given name already exists in the manifest", func() {
			// Note, example skaffold already contains dev environment profile
			BeforeEach(func() {
				envs := []string{"dev"}
				skaffoldManifest, err = kev.AddProfiles(existingSkaffoldPath, envs, includeAdditionalProfiles)
			})

			It("doesn't add it to the skaffold manifest", func() {
				Expect(skaffoldManifest.ProfilesNames()).To(ContainElement("dev-env"))
				Expect(skaffoldManifest.Profiles).To(HaveLen(1))
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})

	Describe("SetBuildArtifacts", func() {

		var skaffoldManifest *kev.SkaffoldManifest

		Context("with detected service Dockerfiles", func() {

			BeforeEach(func() {
				skaffoldManifest = &kev.SkaffoldManifest{}
			})

			// Note, service name is derived from the Dockerfile location path
			// example: src/myservice/Dockerfile will result in `myservice` service name

			Context("and detected remote registry image names matching service name", func() {
				analysis := &kev.Analysis{
					Dockerfiles: []string{"src/myservice/Dockerfile"},
					Images:      []string{"quay.io/myorg/myservice", "myservice"},
				}

				It("picks remote registry image path and sets correct Build configuration", func() {
					skaffoldManifest.SetBuildArtifacts(analysis)
					Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(1))
					Expect(skaffoldManifest.Build.Artifacts[0].ImageName).To(Equal("quay.io/myorg/myservice"))
					Expect(skaffoldManifest.Build.Artifacts[0].Workspace).To(Equal("src/myservice"))
				})
			})

			Context("and no remote registry image names detected matching service name", func() {
				analysis := &kev.Analysis{
					Dockerfiles: []string{"src/myservice/Dockerfile"},
					Images:      []string{"quay.io/myorg/someotherserviceregistry"},
				}

				It("sets image name to be the same as service name and sets correct Build configuration", func() {
					skaffoldManifest.SetBuildArtifacts(analysis)
					Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(1))
					Expect(skaffoldManifest.Build.Artifacts[0].ImageName).To(Equal("myservice"))
					Expect(skaffoldManifest.Build.Artifacts[0].Workspace).To(Equal("src/myservice"))
				})
			})

		})

	})
})
