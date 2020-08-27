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

package kev

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/appvia/kube-devx/pkg/kev/converter/kubernetes"
)

// SkaffoldManifest is a wrapper around latest SkaffoldConfig
type SkaffoldManifest latest.SkaffoldConfig

const (
	// SkaffoldFileName is a file name of skaffold manifest
	SkaffoldFileName = "skaffold.yaml"
)

var (
	disabled = false
	enabled  = true
)

// NewSkaffoldManifest returns a new SkaffoldManifest struct.
func NewSkaffoldManifest(envs []string) (*SkaffoldManifest, error) {

	manifest := BaseSkaffoldManifest()
	manifest.SetProfiles(envs)
	manifest.AdditionalProfiles()

	return manifest, nil
}

// LoadSkaffoldManifest returns skaffold manifest.
func LoadSkaffoldManifest(path string) (*SkaffoldManifest, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s *SkaffoldManifest
	return s, yaml.Unmarshal(data, &s)
}

// UpdateSkaffoldProfiles updates skaffold profiles with appropriate kubernetes files output paths.
// Note, it'll persist updated profiles in the skaffold.yaml file.
// Important: This will always persist the last rendered directory as Deploy manifests source!
func UpdateSkaffoldProfiles(path string, envToOutputPath map[string]string) error {
	skaffold, err := LoadSkaffoldManifest(path)
	if err != nil {
		return err
	}

	skaffold.UpdateProfiles(envToOutputPath)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := skaffold.WriteTo(file); err != nil {
		return err
	}
	return file.Close()
}

// UpdateProfiles updates profile for each environment with its K8s output path
// Note, currently the only supported format is native kubernetes manifests
func (s *SkaffoldManifest) UpdateProfiles(envToOutputPath map[string]string) {
	for _, p := range s.Profiles {
		if outputPath, found := envToOutputPath[p.Name]; found {

			manifestsPath := ""
			if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
				manifestsPath = filepath.Join(outputPath, "*")
			} else if err == nil && info.Mode().IsRegular() {
				manifestsPath = outputPath
			}

			p.Deploy.KubectlDeploy.Manifests = []string{
				manifestsPath,
			}
		}
	}
}

// BaseSkaffoldManifest returns base Skaffold manifest
func BaseSkaffoldManifest() *SkaffoldManifest {
	return &SkaffoldManifest{
		APIVersion: latest.Version,
		Kind:       "Config",
		Metadata: latest.Metadata{
			Name: "KevApp",
		},
		// @todo figure out top level pipeline elements
		// Pipeline: latest.Pipeline{}
	}
}

// SetProfiles adds Skaffold profiles for all Kev project environments
// when list of environments is empty it will add profile for defaultEnvs
func (s *SkaffoldManifest) SetProfiles(envs []string) {

	if len(envs) == 0 {
		envs = []string{defaultEnv}
	}

	for _, e := range envs {
		s.Profiles = append(s.Profiles, latest.Profile{
			Name: e,
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
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
					// @todo set artifacts appropriately or leave it for user to fill in
					// Artifacts: []*latest.Artifact{},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						// @todo strategy will depend on the output format so this might
						// need to mutate when iterating with Kev
						KubectlDeploy: &latest.KubectlDeploy{
							Manifests: []string{
								filepath.Join(kubernetes.MultiFileSubDir, e, "*"),
							},
						},
					},
					// @todo define convention on how kubernetes context are named.
					// for now simply user environment name with `-context` suffix.
					KubeContext: e + "-context",
				},
				Test:        []*latest.TestCase{},
				PortForward: []*latest.PortForwardResource{},
			},
		})
	}
}

// AdditionalProfiles adds additional Skaffold profiles
func (s *SkaffoldManifest) AdditionalProfiles() {
	s.Profiles = append(s.Profiles, []latest.Profile{
		// Helper profile for developing in local minikube
		{
			Name: "minikube",
			Activation: []latest.Activation{
				{
					KubeContext: "minikube",
				},
			},
		},
		// Helper profile for developing in local docker-desktop
		{
			Name: "docker-desktop",
			Activation: []latest.Activation{
				{
					KubeContext: "docker-desktop",
				},
			},
		},
		// Helper profile for use in CI pipeline
		{
			Name: "ci-build-no-push",
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{
							Push: &disabled,
						},
					},
				},
				// deploy is a no-op intentionally
				Deploy: latest.DeployConfig{},
			},
		},
		// Helper profile for use in CI pipeline
		{
			Name: "ci-build-and-push",
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{
							Push: &enabled,
						},
					},
				},
				// deploy is a no-op intentionally
				Deploy: latest.DeployConfig{},
			},
		},
	}...)
}

// WriteTo writes out a skaffold manifest to a writer.
// The SkaffoldManifest struct implements the io.WriterTo interface.
func (s *SkaffoldManifest) WriteTo(w io.Writer) (n int64, err error) {
	data, err := yaml.Marshal(s)
	if err != nil {
		return int64(0), err
	}

	written, err := w.Write(data)
	return int64(written), err
}
