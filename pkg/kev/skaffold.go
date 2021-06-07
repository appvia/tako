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
	"sort"
	"strings"
	"time"

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

	// EnvProfileNameSuffix is a suffix added to environment specific profile name
	EnvProfileNameSuffix = "-env"

	// EnvProfileKubeContextSuffix is a suffix added to environment specific profile kube-context
	EnvProfileKubeContextSuffix = "-context"

	// DefaultSkaffoldNamespace is a default namespace to which Skaffold will deploy manifests
	DefaultSkaffoldNamespace = "default"
)

var (
	disabled = false
	enabled  = true
)

// NewSkaffoldManifest returns a new SkaffoldManifest struct.
func NewSkaffoldManifest(envs []string, project *ComposeProject) *SkaffoldManifest {

	// it's OK to pass nil analysis so no error handling necessary here
	analysis, _ := analyzeProject()

	manifest := BaseSkaffoldManifest()
	manifest.SetBuildArtifacts(analysis, project)
	manifest.SetProfiles(envs)
	manifest.SetAdditionalProfiles()

	return manifest
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

// InjectProfiles injects kev profiles to existing Skaffold manifest
// Note, if profile name already exists in the skaffold manifest then profile won't be added
func InjectProfiles(path string, envs []string, includeAdditional bool) (*SkaffoldManifest, error) {
	skaffold, err := LoadSkaffoldManifest(path)
	if err != nil {
		return nil, err
	}

	skaffold.SetProfiles(envs)
	if includeAdditional {
		skaffold.SetAdditionalProfiles()
	}

	return skaffold, nil
}

// UpdateSkaffoldBuildArtifacts updates skaffold build artefacts with freshly discovered list of images and contexts.
// Note, it'll persist updated build artefacts in the skaffold.yaml file only when change in build artefacts was detected.
// Important: The last discovered images and contexts will be persisted (if changed)!
func UpdateSkaffoldBuildArtifacts(path string, project *ComposeProject) error {
	if !fileExists(path) {
		return fmt.Errorf("skaffold config file (%s) doesn't exist", path)
	}

	skaffold, err := LoadSkaffoldManifest(path)
	if err != nil {
		return err
	}

	// ignore analysis errors as it's OK to pass nil analysis
	analysis, _ := analyzeProject()

	changed := skaffold.UpdateBuildArtifacts(analysis, project)

	// only persist when the list of artifacts changed
	if changed {
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

// UpdateBuildArtifacts sets build artefacts in Skaffold manifest and returns change status
// true - when list of artefacts was updated, false - otherwise
func (s *SkaffoldManifest) UpdateBuildArtifacts(analysis *Analysis, project *ComposeProject) bool {
	prevArts := s.Build.Artifacts

	sort.SliceStable(prevArts, func(i, j int) bool {
		return prevArts[i].ImageName < prevArts[j].ImageName
	})

	s.SetBuildArtifacts(analysis, project)

	currArts := s.Build.Artifacts
	sort.SliceStable(currArts, func(i, j int) bool {
		return currArts[i].ImageName < currArts[j].ImageName
	})

	return !reflect.DeepEqual(currArts, prevArts)
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
			Name: "App",
		},
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				BuildType: latest.BuildType{
					// Local build is a default build strategy!
					// When "local" kubecontext is in use the built images won't be pushed to a registry.
					// If "Push" option isn't specified (which is our default), then images are pushed only if
					// the current Kubernetes context connects to a remote cluster.
					LocalBuild: &latest.LocalBuild{},
				},
				TagPolicy: latest.TagPolicy{
					GitTagger: &latest.GitTagger{
						Variant: "Tags",
					},
				},
			},
		},
	}
}

// SetProfiles adds Skaffold profiles for all Kev project environments
// when list of environments is empty it will add profile for defaultEnvs
func (s *SkaffoldManifest) SetProfiles(envs []string) {

	if len(envs) == 0 {
		envs = []string{SandboxEnv}
	}

	for _, e := range envs {

		if s.profileNameExist(e + EnvProfileNameSuffix) {
			continue
		}

		s.Profiles = append(s.Profiles, latest.Profile{
			Name: e + EnvProfileNameSuffix,
			Pipeline: latest.Pipeline{
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						// @todo(mc) strategy will depend on the output format so deploy
						// type might mutate as well when iterating with Kev
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

// SetAdditionalProfiles adds additional Skaffold profiles
func (s *SkaffoldManifest) SetAdditionalProfiles() {

	s.AddProfileIfNotPresent(latest.Profile{
		Name: "ci-local-build-no-push",
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

	s.AddProfileIfNotPresent(latest.Profile{
		Name: "ci-local-build-and-push",
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

// AddProfileIfNotPresent adds Skaffold profile unless profile with that name already exists
func (s *SkaffoldManifest) AddProfileIfNotPresent(p latest.Profile) {
	if !s.profileNameExist(p.Name) {
		s.Profiles = append(s.Profiles, p)
	}
}

// SetBuildArtifacts detects build artifacts from the current project and adds `build` section to the manifest
func (s *SkaffoldManifest) SetBuildArtifacts(analysis *Analysis, project *ComposeProject) {
	artifacts := []*latest.Artifact{}

	for context, image := range collectBuildArtifacts(analysis, project) {
		artifact := &latest.Artifact{
			ImageName: image,
			Workspace: context,
		}

		if analysis == nil || analysis.Dockerfiles == nil || len(analysis.Dockerfiles) == 0 {
			// no Dockerfiles detected, set `buildpacks` as build strategy for the artifact
			artifact.ArtifactType = latest.ArtifactType{
				BuildpackArtifact: &latest.BuildpackArtifact{
					Builder: "paketobuildpacks/builder:base",
				},
			}
		}
		artifacts = append(artifacts, artifact)
	}

	s.Build.Artifacts = artifacts
}

// collectBuildArtfacts returns a map of build contexts to corresponding image names
func collectBuildArtifacts(analysis *Analysis, project *ComposeProject) map[string]string {
	buildArtifacts := map[string]string{}

	// There are at least 3 cases we should handle:
	//
	// 1) Dockerfile detected in analysis and/or docker images referenced in docker-compose file
	// 	  * local docker build -> build artifact as docker image with build context
	// 2) No Dockerfile detected in analysis and docker-compose references image (without context!)
	//    * no build required -> referenced image looks like pre-built image
	// 3) No Dockerfile detected in analysis and docker-compose references image (with context!)
	//    * buildpacks -> context is present

	// Skaffold analysis is present and Dockerfiles have been detected
	if analysis != nil && analysis.Dockerfiles != nil {
		for _, d := range analysis.Dockerfiles {

			var context, svcImageNameFromContext string

			if d == "Dockerfile" {
				// Dockerfile detected in the root directory (i.e. no prefix path), use local dir
				// as context and current working directory name as service image name
				context = "."
				wd, _ := os.Getwd()
				svcImageNameFromContext = filepath.Base(wd)
			} else {
				// Dockerfile detected in subdirectory, use subdirectory as context
				// and immediate parent directory name in which Dockerfile reside as service image name
				context = strings.ReplaceAll(d, "/Dockerfile", "")
				contextParts := strings.Split(context, "/")
				svcImageNameFromContext = contextParts[len(contextParts)-1]
			}

			// NOTE: This may not be always accurate!
			buildArtifacts[context] = svcImageNameFromContext

			// Check whether images detected by Analysis contain service image name derived from the
			// context as that's the best we can do in order to match a service to a corresponding
			// docker registry image.
			//
			// NOTE: When *NO* Images are detected by analysis this is usually due to the absence of K8s
			// manifests which Analysis uses to determine which images are in use.
			if analysis != nil && analysis.Images != nil {
				for _, image := range analysis.Images {
					if len(image) > 0 && strings.HasSuffix(image, svcImageNameFromContext) {
						buildArtifacts[context] = image
						break
					}
				}
			}
		}
	}

	// Extract images referenced by a Docker Compose project and map them to their respective build contexts (if present)
	// Note: Images that don't specify build "context" will be ignored! Such images are deemed pre-built / external and not
	// requiring to be built. `docker-compose build` itself skips images that don't specify `build.context`!
	if project != nil && project.Project != nil && project.Project.Services != nil {
		for _, s := range project.Project.Services {
			if s.Build != nil && len(s.Build.Context) > 0 && len(s.Image) > 0 {
				buildArtifacts[s.Build.Context] = s.Image
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
func RunSkaffoldDev(ctx context.Context, out io.Writer, skaffoldFile string, profiles []string, runCfg *runConfig) error {
	var mutedPhases []string
	var trigger string
	var pollInterval int

	if runCfg.SkaffoldManualTrigger {
		trigger = "manual"
		pollInterval = 0
	} else {
		trigger = "polling"
		pollInterval = 100 // 100ms by default
	}

	if runCfg.SkaffoldVerbose {
		mutedPhases = []string{}
	} else {
		mutedPhases = []string{"build"} // possible options "build", "deploy", "status-check"
	}

	logrus.SetLevel(logrus.WarnLevel)

	// Port-forward options
	pfopts := config.PortForwardOptions{}
	pfopts.Set("user,debug,pods,services")

	skaffoldOpts := config.SkaffoldOptions{
		ConfigurationFile:     skaffoldFile,
		ProfileAutoActivation: true,
		Trigger:               trigger,
		WatchPollInterval:     pollInterval,
		AutoBuild:             true,
		AutoSync:              true,
		AutoDeploy:            true,
		Profiles:              profiles,
		Namespace:             runCfg.K8sNamespace,
		KubeContext:           runCfg.Kubecontext,
		Cleanup:               true,
		NoPrune:               false,
		NoPruneChildren:       false,
		CacheArtifacts:        false,
		StatusCheck:           true,
		Tail:                  runCfg.SkaffoldTail,
		PortForward:           pfopts,
		Muted: config.Muted{
			Phases: mutedPhases,
		},
		WaitForDeletions: config.WaitForDeletions{
			Max:     60 * time.Second,
			Delay:   2 * time.Second,
			Enabled: true,
		},
		CustomLabels: []string{
			"kev.dev/profile=" + profiles[0],
			"kev.dev/kubecontext=" + runCfg.Kubecontext,
			"kev.dev/namespace=" + runCfg.K8sNamespace,
			fmt.Sprintf("kev.dev/pollinterval=%d", pollInterval),
		},
	}

	runCtx, cfg, err := runContext(skaffoldOpts, profiles, out)
	if err != nil {
		return errors.Wrap(err, "Skaffold dev failed")
	}

	r, err := runner.NewForConfig(runCtx)
	if err != nil {
		return errors.Wrap(err, "Skaffold dev failed")
	}

	prune := func() {}
	if skaffoldOpts.Prune() {
		defer func() {
			prune()
		}()
	}

	cleanup := func() {}
	if skaffoldOpts.Cleanup {
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
func runContext(opts config.SkaffoldOptions, profiles []string, out io.Writer) (*runcontext.RunContext, *latest.SkaffoldConfig, error) {
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

	configs := []*latest.SkaffoldConfig{}

	for _, p := range parsed {
		configs = append(configs, p.(*latest.SkaffoldConfig))
	}

	config := configs[0]

	appliedProfiles, err := schema.ApplyProfiles(config, opts, profiles)
	if err != nil {
		return nil, nil, fmt.Errorf("applying profiles: %w", err)
	}
	_, _ = fmt.Fprintln(out, "Applied profiles:", appliedProfiles)

	kubectx.ConfigureKubeConfig(opts.KubeConfig, opts.KubeContext, config.Deploy.KubeContext)

	if err := defaults.Set(configs[0]); err != nil {
		return nil, nil, fmt.Errorf("setting default values: %w", err)
	}

	if err := validation.Process(configs); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}

	runCtx, err := runcontext.GetRunContext(opts, []latest.Pipeline{config.Pipeline})
	if err != nil {
		return nil, nil, fmt.Errorf("getting run context: %w", err)
	}

	return runCtx, config, nil
}

// ActivateSkaffoldDevLoop checks whether skaffold dev loop can be activated, and returns an error if not.
// It'll also attempt to reconcile Skaffold profiles before starting dev loop - this is done
// so that necessary profiles are added to the Skaffold config. It's necessary as environment
// specific profile is supplied to Skaffold so it knows what manifests to deploy and to which k8s cluster.
func ActivateSkaffoldDevLoop(workDir string) (string, *SkaffoldManifest, error) {
	manifest, err := LoadManifest(workDir)
	if err != nil {
		return "", nil, errors.Wrap(err, "Unable to load app manifest")
	}

	msg := fmt.Sprintf(`
If you don't currently have skaffold.yaml in your project you may bootstrap a new one with "skaffold init" command.
Once you have skaffold.yaml in your project, make sure that Kev references it by adding "skaffold: skaffold.yaml" in %s!`, ManifestFilename)

	if len(manifest.Skaffold) == 0 {
		return "", nil, errors.New("Can't activate Skaffold dev loop. Kev wasn't initialized with --skaffold." + msg)
	}

	configPath := filepath.Join(workDir, manifest.Skaffold)

	if !fileExists(configPath) {
		return "", nil, errors.New("Can't find Skaffold config file referenced by Kev manifest. Have you initialized Kev with --skaffold?" + msg)
	}

	// Reconcile skaffold config and add potentially missing profiles before starting dev loop
	reconciledSkaffoldConfig, err := InjectProfiles(configPath, manifest.GetEnvironmentsNames(), true)
	if err != nil {
		return "", nil, errors.Wrap(err, "Couldn't reconcile Skaffold config - required profiles haven't been added.")
	}

	return configPath, reconciledSkaffoldConfig, nil
}
