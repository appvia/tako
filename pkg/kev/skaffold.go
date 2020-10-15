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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/analyze"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/build"
	initconfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/deploy"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/validation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/appvia/kev/pkg/kev/converter/kubernetes"
	"github.com/appvia/kev/pkg/kev/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SkaffoldManifest is a wrapper around latest SkaffoldConfig
type SkaffoldManifest latest.SkaffoldConfig

// Analysis holds the information about detected dockerfiles and images
type Analysis struct {
	Dockerfiles []string `json:"dockerfiles,omitempty"`
	Images      []string `json:"images,omitempty"`
}

const (
	// SkaffoldFileName is a file name of skaffold manifest
	SkaffoldFileName = "skaffold.yaml"

	// ProfileNamePrefix is a prefix to the added skaffold aprofile
	ProfileNamePrefix = "kev-"

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
func NewSkaffoldManifest(envs []string, project *ComposeProject) (*SkaffoldManifest, error) {

	analysis, err := analyzeProject()
	if err != nil {
		// just warn for now - potentially put project analysis behind a flag?
		log.Warn(err.Error())
	}

	manifest := BaseSkaffoldManifest()
	manifest.SetBuildArtifacts(analysis, project)
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
	if !fileExists(path) {
		return fmt.Errorf("skaffold config file (%s) doesn't exist", path)
	}

	skaffold, err := LoadSkaffoldManifest(path)
	if err != nil {
		return err
	}

	if changed := skaffold.UpdateProfiles(envToOutputPath); changed {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		if _, err := skaffold.WriteTo(file); err != nil {
			return err
		}
		return file.Close()
	}

	return nil
}

// UpdateProfiles updates profile for each environment with its K8s output path
// Note, currently the only supported format is native kubernetes manifests
func (s *SkaffoldManifest) UpdateProfiles(envToOutputPath map[string]string) bool {
	changed := false

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

			manifests := []string{
				manifestsPath,
			}

			// only update when necessary
			if !reflect.DeepEqual(p.Deploy.KubectlDeploy.Manifests, manifests) {
				p.Deploy.KubectlDeploy.Manifests = []string{
					manifestsPath,
				}
				changed = true
			}
		}
	}

	return changed
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
			Pipeline: latest.Pipeline{
				Deploy: latest.DeployConfig{
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
			Pipeline: latest.Pipeline{
				Deploy: latest.DeployConfig{
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

// SetBuildArtifacts detects build artifacts from the current project and adds `build` section to the manifest
func (s *SkaffoldManifest) SetBuildArtifacts(analysis *Analysis, project *ComposeProject) error {

	// don't set build artefacts if no analysis available
	if analysis == nil {
		return nil
	}

	artifacts := []*latest.Artifact{}

	for context, image := range collectBuildArtifacts(analysis, project) {
		artifacts = append(artifacts, &latest.Artifact{
			ImageName: image,
			Workspace: context,
		})
	}

	s.Build = latest.BuildConfig{
		Artifacts: artifacts,
	}

	return nil
}

// collectBuildArtfacts returns a map of build contexts to corresponding image names
func collectBuildArtifacts(analysis *Analysis, project *ComposeProject) map[string]string {
	buildArtifacts := map[string]string{}

	if len(analysis.Images) == 0 {
		// no images detected - usually the case when there are no kubernetes manifests
		// Extract referenced images and map them to their respective build contexts (if present) from Compose project
		// Note: It'll miss images without "build" context specified!

		if project.Project != nil && project.Project.Services != nil {
			for _, s := range project.Project.Services {
				if s.Build != nil && len(s.Build.Context) > 0 && len(s.Image) > 0 {
					buildArtifacts[s.Build.Context] = s.Image
				}
			}
		}
	}

	for _, d := range analysis.Dockerfiles {

		context := strings.ReplaceAll(d, "/Dockerfile", "")
		contextParts := strings.Split(context, "/")
		svcNameFromContext := contextParts[len(contextParts)-1]

		if d == "Dockerfile" {
			// Dockerfile detected in the root directory, use local dir as context
			// and current working directory name as service name
			context = "."
			wd, _ := os.Getwd()
			svcNameFromContext = filepath.Base(wd)
		}

		// Check whether images contain service name derived from context
		// as that's the best we can do in order to match a service to corresponding
		// docker registry image. If no docker registry image was detected
		// then we use service name as docker image name.
		re := regexp.MustCompile(fmt.Sprintf(`.*\/%s`, svcNameFromContext))

		for _, image := range analysis.Images {
			if found := re.FindAllStringSubmatchIndex(image, -1); found != nil {
				buildArtifacts[context] = image
				break
			} else {
				buildArtifacts[context] = svcNameFromContext
			}
		}
	}

	return buildArtifacts
}

// analyzeProject analyses the project and returns Analysis report object
func analyzeProject() (*Analysis, error) {
	c := initconfig.Config{
		Analyze:             true,
		EnableNewInitFormat: false,
	}

	a := analyze.NewAnalyzer(c)
	if err := a.Analyze("."); err != nil {
		return nil, err
	}

	deployInitializer := deploy.NewInitializer(a.Manifests(), a.KustomizeBases(), a.KustomizePaths(), c)
	images := deployInitializer.GetImages()

	buildInitializer := build.NewInitializer(a.Builders(), c)
	if err := buildInitializer.ProcessImages(images); err != nil {
		return nil, err
	}

	if c.Analyze {
		out := &bytes.Buffer{}
		buildInitializer.PrintAnalysis(out)

		a := &Analysis{}

		if err := json.Unmarshal(out.Bytes(), &a); err != nil {
			return nil, err
		}

		return a, nil
	}

	return nil, nil
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

// RunSkaffoldDev starts Skaffold pipeline in dev mode for given profiles, kubernetes context and namespace
func RunSkaffoldDev(ctx context.Context, out io.Writer, profiles []string, ns, kubeCtx, skaffoldFile string, pollInterval int) error {
	if pollInterval == 0 {
		pollInterval = 100 // 100 ms by default if interval not specified
	}

	logrus.SetLevel(logrus.WarnLevel)

	opts := config.SkaffoldOptions{
		ConfigurationFile:     skaffoldFile,
		ProfileAutoActivation: true,
		Trigger:               "polling",
		WatchPollInterval:     pollInterval,
		AutoBuild:             true,
		AutoSync:              true,
		AutoDeploy:            true,
		Profiles:              profiles,
		Namespace:             ns,
		KubeContext:           kubeCtx,
		Cleanup:               true,
		NoPrune:               false,
		NoPruneChildren:       false,
		CacheArtifacts:        false,
		PortForward: config.PortForwardOptions{
			Enabled:     true,
			ForwardPods: true,
		},
	}

	runCtx, cfg, err := runContext(opts)

	r, err := runner.NewForConfig(runCtx)

	if err != nil {
		return errors.Wrap(err, "Skaffold dev failed")
	}

	prune := func() {}
	if opts.Prune() {
		defer func() {
			prune()
		}()
	}

	cleanup := func() {}
	if opts.Cleanup {
		defer func() {
			cleanup()
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := func() error {

				err := r.Dev(ctx, out, cfg.Build.Artifacts)

				if r.HasDeployed() {
					cleanup = func() {
						if err := r.Cleanup(context.Background(), out); err != nil {
							log.Warnf("Skaffold deployer cleanup: %s", err)
						}
					}
				}

				if r.HasBuilt() {
					prune = func() {
						if err := r.Prune(context.Background(), out); err != nil {
							log.Warnf("Skaffold builder cleanup: %s", err)
						}
					}
				}

				return err
			}()

			if err != nil {
				if !errors.Is(err, runner.ErrorConfigurationChanged) {
					return errors.Wrap(err, "Something went wrong in the skaffold dev... ")
				}

				log.Error("Skaffold config has changed! Please restart `kev dev`.")
				return err
			}
		}
	}
}

// runContext returns runner context and config for Skaffold dev mode
func runContext(opts config.SkaffoldOptions) (*runcontext.RunContext, *latest.SkaffoldConfig, error) {
	parsed, err := schema.ParseConfigAndUpgrade(opts.ConfigurationFile, latest.Version)
	if err != nil {
		if os.IsNotExist(errors.Unwrap(err)) {
			return nil, nil, fmt.Errorf("skaffold config file %s not found - check your current working directory, or try running `skaffold init`", opts.ConfigurationFile)
		}

		// If the error is NOT that the file doesn't exist, then we warn the user
		// that maybe they are using an outdated version of Skaffold that's unable to read
		// the configuration.
		return nil, nil, fmt.Errorf("parsing skaffold config: %w", err)
	}

	config := parsed.(*latest.SkaffoldConfig)

	if err = schema.ApplyProfiles(config, opts); err != nil {
		return nil, nil, fmt.Errorf("applying profiles: %w", err)
	}

	kubectx.ConfigureKubeConfig(opts.KubeConfig, opts.KubeContext, config.Deploy.KubeContext)

	if err := defaults.Set(config); err != nil {
		return nil, nil, fmt.Errorf("setting default values: %w", err)
	}

	if err := validation.Process(config); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}

	runCtx, err := runcontext.GetRunContext(opts, config.Pipeline)
	if err != nil {
		return nil, nil, fmt.Errorf("getting run context: %w", err)
	}

	return runCtx, config, nil
}
