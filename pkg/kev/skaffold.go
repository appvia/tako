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
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/appvia/kev/pkg/kev/converter/kubernetes"
)

// SkaffoldManifest is a wrapper around latest SkaffoldConfig
type SkaffoldManifest latest.SkaffoldConfig

const (
	// SkaffoldFileName is a file name of skaffold manifest
	SkaffoldFileName = "skaffold.yaml"

	// ProfileNamePrefix is a prefix to the added skaffold aprofile
	ProfileNamePrefix = "zz-"

	// EnvProfileNameSuffix is a suffix added to environment specific profile name
	EnvProfileNameSuffix = "-env"

	// EnvProfileKubeContextSuffix is a suffix added to environment specific profile kube-context
	EnvProfileKubeContextSuffix = "-context"
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

// AddProfiles injects kev profiles to existing Skaffold manifest
// Note, if profile name already exists in the skaffold manifest then profile won't be added
func AddProfiles(path string, envs []string, includeAdditional bool) (*SkaffoldManifest, error) {
	skaffold, err := LoadSkaffoldManifest(path)
	if err != nil {
		return nil, err
	}

	skaffold.SetProfiles(envs)
	if includeAdditional {
		skaffold.AdditionalProfiles()
	}

	return skaffold, nil
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

		// envToOutputPath is keyed by canonical environment name, however
		// profile names in skaffold manifest might have additional suffix!
		// We must strip the profile suffix to check the path for that environment.
		envNameFromProfileName := strings.ReplaceAll(p.Name, EnvProfileNameSuffix, "")

		if outputPath, found := envToOutputPath[envNameFromProfileName]; found {
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

		if s.profileNameExist(e + EnvProfileNameSuffix) {
			continue
		}

		s.Profiles = append(s.Profiles, latest.Profile{
			Name: e + EnvProfileNameSuffix,
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
					KubeContext: e + EnvProfileKubeContextSuffix,
				},
				Test:        []*latest.TestCase{},
				PortForward: []*latest.PortForwardResource{},
			},
		})
	}
}

// AdditionalProfiles adds additional Skaffold profiles
func (s *SkaffoldManifest) AdditionalProfiles() {

	if !s.profileNameExist(ProfileNamePrefix + "minikube") {
		// Helper profile for developing in local minikube
		s.Profiles = append(s.Profiles, latest.Profile{
			Name: ProfileNamePrefix + "minikube",
			Activation: []latest.Activation{
				{
					KubeContext: "minikube",
				},
			},
		})
	}

	if !s.profileNameExist(ProfileNamePrefix + "docker-desktop") {
		// Helper profile for developing in local docker-desktop
		s.Profiles = append(s.Profiles, latest.Profile{
			Name: ProfileNamePrefix + "docker-desktop",
			Activation: []latest.Activation{
				{
					KubeContext: "docker-desktop",
				},
			},
		})
	}

	if !s.profileNameExist(ProfileNamePrefix + "ci-build-no-push") {
		// Helper profile for use in CI pipeline
		s.Profiles = append(s.Profiles, latest.Profile{
			Name: ProfileNamePrefix + "ci-build-no-push",
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
		})
	}

	if !s.profileNameExist(ProfileNamePrefix + "ci-build-and-push") {
		// Helper profile for use in CI pipeline
		s.Profiles = append(s.Profiles, latest.Profile{
			Name: ProfileNamePrefix + "ci-build-and-push",
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
		})
	}
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

// ProfilesNames returns sorted list of defined skaffold profile names
func (s *SkaffoldManifest) ProfilesNames() []string {
	profiles := []string{}
	for _, p := range s.Profiles {
		profiles = append(profiles, p.Name)
	}
	sort.Strings(profiles)
	return profiles
}

// profileNameExist returns true if skaffold contains profiles of given name
func (s *SkaffoldManifest) profileNameExist(profileName string) bool {
	profiles := s.ProfilesNames()
	i := sort.SearchStrings(profiles, profileName)
	return i < len(profiles) && profiles[i] == profileName
}

// sortProfiles sorts manifest's profiles by name
func (s *SkaffoldManifest) sortProfiles() {
	sort.Slice(s.Profiles, func(i, j int) bool {
		return s.Profiles[i].Name < s.Profiles[j].Name
	})
}
