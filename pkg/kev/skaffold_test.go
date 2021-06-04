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
	composego "github.com/compose-spec/compose-go/types"
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

	Describe("NewSkaffoldManifest", func() {
		var (
			skaffoldManifest *kev.SkaffoldManifest
		)

		JustBeforeEach(func() {
			skaffoldManifest = kev.NewSkaffoldManifest([]string{}, &kev.ComposeProject{})
		})

		It("generates skaffold config for the project", func() {
			Expect(skaffoldManifest).ToNot(BeNil())
		})
	})

	Describe("BaseSkaffoldManifest", func() {
		It("returns base skaffold with global pipeline build configuration", func() {
			Expect(kev.BaseSkaffoldManifest()).To(Equal(
				&kev.SkaffoldManifest{
					APIVersion: latest.Version,
					Kind:       "Config",
					Metadata: latest.Metadata{
						Name: "App",
					},
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							BuildType: latest.BuildType{
								LocalBuild: &latest.LocalBuild{},
							},
							TagPolicy: latest.TagPolicy{
								GitTagger: &latest.GitTagger{
									Variant: "Tags",
								},
							},
						},
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
					}))
				}
			})

			It("generates correct pipeline Build section for each environment", func() {
				for _, p := range manifest.Profiles {
					Expect(p.Build).To(Equal(latest.BuildConfig{}))
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

	Describe("SetAdditionalProfiles", func() {

		manifest := kev.BaseSkaffoldManifest()
		manifest.SetAdditionalProfiles()

		It("adds all additional profiles", func() {
			Expect(manifest.Profiles).To(HaveLen(2))
		})

		Context("ci-local-build-no-push", func() {
			enabled := false

			It("adds additional profiles to the skaffold manifest", func() {
				Expect(manifest.Profiles).To(ContainElement(latest.Profile{
					Name: "ci-local-build-no-push",
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							BuildType: latest.BuildType{
								LocalBuild: &latest.LocalBuild{
									Push: &enabled,
								},
							},
						},
					},
				}))
			})
		})

		Context("ci-local-build-and-push", func() {
			enabled := true

			It("adds additional profiles to the skaffold manifest", func() {
				Expect(manifest.Profiles).To(ContainElement(latest.Profile{
					Name: "ci-local-build-and-push",
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							BuildType: latest.BuildType{
								LocalBuild: &latest.LocalBuild{
									Push: &enabled,
								},
							},
						},
					},
				}))
			})
		})

		When("profile of the same name already exists in skaffold profiles", func() {

			BeforeEach(func() {
				// explicitly triggering another SetAdditionalProfiles
				manifest.SetAdditionalProfiles()
			})

			It("doesn't add existing additional profiles again", func() {
				Expect(manifest.Profiles).To(HaveLen(2))
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

	Describe("UpdateBuildArtifacts", func() {
		var (
			skaffoldManifest *kev.SkaffoldManifest
			project          *kev.ComposeProject
			analysis         *kev.Analysis
			changed          bool
		)

		BeforeEach(func() {
			analysis = &kev.Analysis{
				Dockerfiles: []string{"src/svc1/Dockerfile"},
				Images:      []string{"quay.io/myorg/svc1"},
			}

			project = &kev.ComposeProject{
				Project: &composego.Project{
					Services: composego.Services(
						[]composego.ServiceConfig{
							{
								Name:  "svc2",
								Image: "quay.io/myorg/svc2",
								Build: &composego.BuildConfig{
									Context: "src/svc2",
								},
							},
						},
					),
				},
			}

			skaffoldManifest = &kev.SkaffoldManifest{}
			skaffoldManifest.Build.Artifacts = []*latest.Artifact{
				{
					ImageName: "quay.io/myorg/svc1",
					Workspace: "src/svc1",
				},
				{
					ImageName: "quay.io/myorg/svc2",
					Workspace: "src/svc2",
				},
			}
		})

		JustBeforeEach(func() {
			changed = skaffoldManifest.UpdateBuildArtifacts(analysis, project)
		})

		When("list of detected build artefacts had not changed", func() {
			It("doesn't update build artefacts in skaffold manifest", func() {
				Expect(changed).To(BeFalse())
				Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(2))

				images := []string{}
				for _, a := range skaffoldManifest.Build.Artifacts {
					images = append(images, a.ImageName)
				}
				Expect(images).To(ContainElements("quay.io/myorg/svc1", "quay.io/myorg/svc2"))
				Expect(images).ToNot(ContainElement("quay.io/myorg/svc99"))
			})
		})

		When("list of detected build artefacts has changed", func() {
			BeforeEach(func() {
				project.Project.Services[0].Image = "quay.io/myorg/svc99"
			})

			It("updates build artefacts in skaffold manifest", func() {
				Expect(changed).To(BeTrue())
				Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(2))

				images := []string{}
				for _, a := range skaffoldManifest.Build.Artifacts {
					images = append(images, a.ImageName)
				}
				Expect(images).To(ContainElements("quay.io/myorg/svc1", "quay.io/myorg/svc99"))
				Expect(images).ToNot(ContainElement("quay.io/myorg/svc2"))
			})
		})
	})

	Describe("InjectProfiles", func() {
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
				skaffoldManifest, err = kev.InjectProfiles(existingSkaffoldPath, envs, includeAdditionalProfiles)
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
				skaffoldManifest, err = kev.InjectProfiles(existingSkaffoldPath, envs, includeAdditionalProfiles)
			})

			It("doesn't add it to the skaffold manifest", func() {
				Expect(skaffoldManifest.ProfilesNames()).To(ContainElement("dev-env"))
				Expect(skaffoldManifest.Profiles).To(HaveLen(1))
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})

	Describe("SetBuildArtifacts", func() {

		var (
			skaffoldManifest *kev.SkaffoldManifest
			project          *kev.ComposeProject
			analysis         *kev.Analysis
		)

		BeforeEach(func() {
			skaffoldManifest = &kev.SkaffoldManifest{}
		})

		JustBeforeEach(func() {
			skaffoldManifest.SetBuildArtifacts(analysis, project)
		})

		Context("with detected service Dockerfiles", func() {
			// Note, service image name is derived from the Dockerfile location path
			// example: src/myservice/Dockerfile will result in `myservice` service image name

			Context("and detected remote registry image names matching service name", func() {
				BeforeEach(func() {
					analysis = &kev.Analysis{
						Dockerfiles: []string{"src/myservice/Dockerfile"},
						Images:      []string{"quay.io/myorg/myservice", "myservice"},
					}
					project = &kev.ComposeProject{}
				})

				It("picks remote registry image path and sets correct Build configuration", func() {
					Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(1))
					Expect(skaffoldManifest.Build.Artifacts[0].ImageName).To(Equal("quay.io/myorg/myservice"))
					Expect(skaffoldManifest.Build.Artifacts[0].Workspace).To(Equal("src/myservice"))
				})
			})

			Context("and no remote registry image names detected matching service name", func() {
				BeforeEach(func() {
					analysis = &kev.Analysis{
						Dockerfiles: []string{"src/myservice/Dockerfile"},
						Images:      []string{"quay.io/myorg/someotherserviceregistry"},
					}
				})

				It("sets image name to be the same as service name and sets correct Build configuration", func() {
					Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(1))
					Expect(skaffoldManifest.Build.Artifacts[0].ImageName).To(Equal("myservice"))
					Expect(skaffoldManifest.Build.Artifacts[0].Workspace).To(Equal("src/myservice"))
				})
			})

			When("Docker Compose defines image name with identical context", func() {
				BeforeEach(func() {
					analysis = &kev.Analysis{
						Dockerfiles: []string{"src/myservice/Dockerfile"},
						Images:      []string{},
					}
					project = &kev.ComposeProject{
						Project: &composego.Project{
							Services: composego.Services(
								[]composego.ServiceConfig{
									{
										Name:  "svc1",
										Image: "quay.io/myorg/svc1",
										Build: &composego.BuildConfig{
											Context: "src/myservice",
										},
									},
								},
							),
						},
					}
				})

				It("overrides Skaffold detected build artifact image name with Docker Compose extracted one", func() {
					Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(1))
					Expect(skaffoldManifest.Build.Artifacts[0].ImageName).To(Equal("quay.io/myorg/svc1"))
					Expect(skaffoldManifest.Build.Artifacts[0].Workspace).To(Equal("src/myservice"))
				})
			})
		})

		Context("with or without images detected by Skaffold analysis", func() {
			BeforeEach(func() {
				analysis = &kev.Analysis{
					Images: []string{}, // this can be either empty slice or a list of images
				}
			})

			Context("fallback to Docker Compose source files for extraction of images and build contexts", func() {

				When("Docker Compose project has services referencing images with build contexts", func() {
					image := "quay.io/org/myimage:latest"
					context := "my/context"

					BeforeEach(func() {
						project = &kev.ComposeProject{
							Project: &composego.Project{
								Services: composego.Services(
									[]composego.ServiceConfig{
										{
											Name:  "svc1",
											Image: image,
											Build: &composego.BuildConfig{
												Context: context,
											},
										},
									},
								),
							},
						}
					})

					Context("and Dockerfiles detected by Skaffold analysis", func() {
						BeforeEach(func() {
							analysis.Dockerfiles = []string{"src/myservice/Dockerfile"}
						})

						It("generates skaffold build artefacts with extracted Docker Compose images and their respective contexts", func() {
							// Note: there are two build artifacts: 1) from Skaffold analysis, 2) from Docker Compose
							Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(2))
							Expect(skaffoldManifest.Build.Artifacts).To(ContainElements(
								&latest.Artifact{
									ImageName: "myservice",
									Workspace: "src/myservice",
								},
								&latest.Artifact{
									ImageName: image,
									Workspace: context,
								},
							))
						})

						It("uses default `docker` build strategy for artifact", func() {
							// ensure artifact type is not set to `buildpack`
							Expect(skaffoldManifest.Build.Artifacts[0].ArtifactType.BuildpackArtifact).To(BeNil())
							Expect(skaffoldManifest.Build.Artifacts[1].ArtifactType.BuildpackArtifact).To(BeNil())
						})
					})

					Context("and Dockerfiles NOT detected by Skaffold analysis", func() {
						BeforeEach(func() {
							analysis.Dockerfiles = []string{}
						})

						It("generates skaffold build artefacts with extracted Docker Compose images and their respective contexts", func() {
							// Note: there is one build artifact: 1) from Docker Compose (no Skaffold analysis dockefiles detected!)
							Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(1))
							Expect(skaffoldManifest.Build.Artifacts[0].ImageName).To(Equal(image))
							Expect(skaffoldManifest.Build.Artifacts[0].Workspace).To(Equal(context))
						})

						It("uses `buildpack` build strategy for artifact as Skaffold analysis didn't detect any Dockerfiles", func() {
							// ensure artifact type is set to `buildpack`
							Expect(skaffoldManifest.Build.Artifacts[0].ArtifactType).To(Equal(latest.ArtifactType{
								BuildpackArtifact: &latest.BuildpackArtifact{
									Builder: "paketobuildpacks/builder:base",
								},
							}))
						})
					})

					Context("with nil analysis", func() {
						BeforeEach(func() {
							analysis = nil
						})

						It("generates skaffold build artefacts with extracted Docker Compose images and their respective contexts", func() {
							// Note: there is one build artifact: 1) from Docker Compose (no Skaffold analysis dockefiles detected!)
							Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(1))
							Expect(skaffoldManifest.Build.Artifacts[0].ImageName).To(Equal(image))
							Expect(skaffoldManifest.Build.Artifacts[0].Workspace).To(Equal(context))
						})

						It("uses `buildpack` build strategy for artifact as Skaffold analysis didn't detect any Dockerfiles", func() {
							// ensure artifact type is set to `buildpack`
							Expect(skaffoldManifest.Build.Artifacts[0].ArtifactType).To(Equal(latest.ArtifactType{
								BuildpackArtifact: &latest.BuildpackArtifact{
									Builder: "paketobuildpacks/builder:base",
								},
							}))
						})
					})

				})

				When("Docker Compose project doesn't have services referencing images with build contexts", func() {
					image := "quay.io/org/myimage:latest"

					BeforeEach(func() {
						project = &kev.ComposeProject{
							Project: &composego.Project{
								Services: composego.Services(
									[]composego.ServiceConfig{
										{
											Name:  "svc1",
											Image: image,
										},
									},
								),
							},
						}
					})

					Context("and Dockerfiles detected by Skaffold analysis", func() {
						BeforeEach(func() {
							analysis.Dockerfiles = []string{"src/myservice/Dockerfile"}
						})

						It("skips Docker Compose images without build context defined", func() {
							// NOTE: Skaffold analysis detected Dockerfile so there will be only one build artifact
							Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(1))
						})
					})

					Context("and Dockerfiles NOT detected by Skaffold analysis", func() {
						BeforeEach(func() {
							analysis.Dockerfiles = []string{}
						})

						It("skips Docker Compose images without build context defined", func() {
							// NOTE: Skaffold analysis didn't detect Dockerfile and Docker Compose service didn't have build.context
							Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(0))
						})

					})

					Context("with nil analysis", func() {
						BeforeEach(func() {
							analysis = nil
						})

						It("skips Docker Compose images without build context defined", func() {
							// NOTE: Skaffold analysis didn't detect Dockerfile and Docker Compose service didn't have build.context
							Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(0))
						})
					})
				})

				When("Docker Compose project doesn't have any services", func() {
					BeforeEach(func() {
						project = &kev.ComposeProject{
							Project: &composego.Project{
								Services: composego.Services{},
							},
						}
					})

					Context("and Dockerfiles detected by Skaffold analysis", func() {
						BeforeEach(func() {
							analysis.Dockerfiles = []string{"src/myservice/Dockerfile"}
						})

						It("doesn't add additional build artefacts", func() {
							// NOTE: Skaffold analysis detected Dockerfile so there will be only one build artifact
							Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(1))
						})
					})

					Context("and Dockerfiles NOT detected by Skaffold analysis", func() {
						BeforeEach(func() {
							analysis.Dockerfiles = []string{}
						})

						It("doesn't add additional build artefacts", func() {
							// NOTE: Skaffold analysis didn't detect Dockerfile and Docker Compose doesn't have services
							Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(0))
						})
					})

					Context("with nil analysis", func() {
						BeforeEach(func() {
							analysis = nil
						})

						It("doesn't add additional build artefacts", func() {
							// NOTE: Skaffold analysis didn't detect Dockerfile and Docker Compose doesn't have services
							Expect(skaffoldManifest.Build.Artifacts).To(HaveLen(0))
						})
					})
				})
			})
		})
	})
})
