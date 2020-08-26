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
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/appvia/kube-devx/pkg/kev"
	"github.com/appvia/kube-devx/pkg/kev/converter/kubernetes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

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

			It("falls back to default `dev` environment onlys", func() {
				Expect(manifest.Profiles).ToNot(BeEmpty())
				Expect(manifest.Profiles).To(HaveLen(1))
				Expect(manifest.Profiles[0].Name).To(Equal("dev"))
			})
		})
	})

	Describe("AdditionalProfiles", func() {

		manifest := kev.BaseSkaffoldManifest()
		manifest.AdditionalProfiles()

		It("adds all additional profiles", func() {
			Expect(manifest.Profiles).To(HaveLen(4))
		})

		It("adds additional profiles to the skaffold manifest", func() {
			Expect(manifest.Profiles).To(ContainElement(latest.Profile{
				Name: "minikube",
				Activation: []latest.Activation{
					{
						KubeContext: "minikube",
					},
				},
			}))
		})
	})
})
