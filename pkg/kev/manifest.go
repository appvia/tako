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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/converter"
	"github.com/appvia/kev/pkg/kev/log"
	kmd "github.com/appvia/komando"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Infix component of generated environment override filenames
// (e.g. results in docker-compose.env.dev.yaml)
const envOverrideFileInfix = "env"

// NewManifest returns a new Manifest struct.
func NewManifest(sources *Sources) *Manifest {
	return &Manifest{
		Id:      uuid.New().String(),
		Sources: sources,
	}
}

// LoadManifest returns application manifests.
func LoadManifest(workingDir string) (*Manifest, error) {
	data, err := ioutil.ReadFile(filepath.Join(workingDir, ManifestFilename))
	if err != nil {
		return nil, err
	}

	var m *Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	m.UI = kmd.NoOpUI()

	return m, nil
}

// WriteTo writes out a manifest to a writer.
// The Manifest struct implements the io.WriterTo interface.
func (m *Manifest) WriteTo(w io.Writer) (n int64, err error) {
	data, err := MarshalIndent(m, 2)
	if err != nil {
		return int64(0), err
	}

	written, err := w.Write(data)
	return int64(written), err
}

// GetEnvironment gets a specific environment.
func (m *Manifest) GetEnvironment(name string) (*Environment, error) {
	for _, env := range m.Environments {
		if env.Name == name {
			return env, nil
		}
	}
	return nil, fmt.Errorf("no such environment: %s", name)
}

// GetEnvironments returns filtered app environments.
// If no filter is provided all app environments will be returned.
func (m *Manifest) GetEnvironments(filter []string) (Environments, error) {
	if len(filter) == 0 {
		var allOut = make([]*Environment, len(m.Environments))
		copy(allOut, m.Environments)
		return allOut, nil
	}

	var out Environments
	for _, f := range filter {
		e, err := m.GetEnvironment(f)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// GetEnvironmentsNames returns a slice of all defined environment names
func (m *Manifest) GetEnvironmentsNames() []string {
	out := []string{}
	for _, e := range m.Environments {
		out = append(out, e.Name)
	}
	return out
}

// CalculateSourcesBaseOverride extracts the base override from the manifest's docker-compose source files.
func (m *Manifest) CalculateSourcesBaseOverride(opts ...BaseOverrideOpts) (*Manifest, error) {
	if err := m.Sources.CalculateBaseOverride(opts...); err != nil {
		return nil, err
	}
	return m, nil
}

// MintEnvironments create new environments based on candidate environments.
// This includes an implicit sandbox environment that will always be created.
func (m *Manifest) MintEnvironments(candidates []string) error {
	m.UI.Header("Creating deployment environments...")
	sg := m.UI.StepGroup()
	defer sg.Done()

	fileNameTemplate := m.GetEnvironmentFileNameTemplate()

	m.Environments = Environments{}
	if !contains(candidates, SandboxEnv) {
		candidates = append([]string{SandboxEnv}, candidates...)
	}

	overrideTemplate := m.getSourcesOverride()
	if err := minifyK8sExtensionsToBaseAttributes(overrideTemplate); err != nil {
		return err
	}

	for _, env := range candidates {
		envFilename := filepath.Join(m.getWorkingDir(), fmt.Sprintf(fileNameTemplate, envOverrideFileInfix, env))
		var step kmd.Step
		if env == SandboxEnv {
			step = sg.Add(fmt.Sprintf("Creating the %s sandbox env file: %s", SandboxEnv, envFilename))
		} else {
			step = sg.Add(fmt.Sprintf("Creating the %s env file: %s", env, envFilename))
		}

		candidate := &Environment{
			Name:     env,
			override: overrideTemplate,
			File:     envFilename,
		}

		m.Environments = append(m.Environments, candidate)
		step.Success()
	}

	return nil
}

// GetEnvironmentFileNameTemplate returns environment file name template to match
// the naming convention of the first compose source file
func (m *Manifest) GetEnvironmentFileNameTemplate() string {
	firstSrc := filepath.Base(m.Sources.Files[0])
	parts := strings.Split(firstSrc, ".")
	ext := parts[len(parts)-1]
	return strings.ReplaceAll(firstSrc, ext, "%s.%s."+ext)
}

// ReconcileConfig reconciles config changes with docker-compose sources against deployment environments.
func (m *Manifest) ReconcileConfig(envs ...string) (*Manifest, error) {
	if _, err := m.CalculateSourcesBaseOverride(withEnvVars); err != nil {
		sg := m.UI.StepGroup()
		defer sg.Done()
		renderStepError(m.UI, sg.Add(""), renderStepReconcileDetect, err)
		return nil, err
	}

	sourcesOverride := m.getSourcesOverride()
	filteredEnvs, err := m.GetEnvironments(envs)
	if err != nil {
		sg := m.UI.StepGroup()
		defer sg.Done()
		renderStepError(m.UI, sg.Add(""), renderStepReconcileDetect, err)
		return nil, err
	}

	for _, e := range filteredEnvs {
		if err := validateEnvExtensions(e, sourcesOverride); err != nil {
			sg := m.UI.StepGroup()
			renderStepError(m.UI, sg.Add(""), renderStepReconcileDetect, err)
			sg.Done()
			return nil, err
		}

		log.Debugf("Reconciling environment [%s]", e.Name)

		m.UI.Output(fmt.Sprintf("%s: %s", e.Name, e.File))

		if err := sourcesOverride.diffAndPatch(e.override); err != nil {
			sg := m.UI.StepGroup()
			renderStepError(m.UI, sg.Add(""), renderStepReconcileApply, err)
			sg.Done()
			return nil, err
		}
	}

	return m, nil
}

func validateEnvExtensions(e *Environment, base *composeOverride) error {
	for _, s := range e.GetServices() {
		baseSvc, missingSvcErr := base.getService(s.Name)
		if missingSvcErr != nil {
			continue
		}

		baseSvcK8sCfg, err := config.ParseSvcK8sConfigFromMap(baseSvc.Extensions, config.SkipValidation())
		if err != nil {
			return errors.Wrapf(missingSvcErr, "when parsing service %s extensions in base compose file", baseSvc.Name)
		}

		envSvcK8sCfg, err := config.ParseSvcK8sConfigFromMap(s.Extensions, config.SkipValidation())
		if err != nil {
			return errors.Wrapf(missingSvcErr, "when parsing service %s extensions", s.Name)
		}

		mergedK8sSvcCfg, err := baseSvcK8sCfg.Merge(envSvcK8sCfg)
		if err != nil {
			return missingSvcErr
		}

		if err := mergedK8sSvcCfg.Validate(); err != nil {
			return err
		}
	}

	for name, vol := range e.GetVolumes() {
		baseVol, missingVolError := base.getVolume(name)
		if missingVolError != nil {
			continue
		}

		baseVolK8sCfg, err := config.ParseVolK8sConfigFromMap(baseVol.Extensions)
		if err != nil {
			return errors.Wrapf(missingVolError, "when parsing vol %s extensions in base compose file", name)
		}

		envVolK8sCfg, err := config.ParseVolK8sConfigFromMap(vol.Extensions)
		if err != nil {
			return errors.Wrapf(missingVolError, "when parsing vol %s extensions", name)
		}

		mergedVolK8sCfg, err := baseVolK8sCfg.Merge(envVolK8sCfg)
		if err != nil {
			return missingVolError
		}

		if err := mergedVolK8sCfg.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// MergeEnvIntoSources merges an environment into a parsed instance of the tracked docker-compose sources.
// It returns the merged ComposeProject.
func (m *Manifest) MergeEnvIntoSources(e *Environment) (*ComposeProject, error) {
	p, err := m.SourcesToComposeProject()
	if err != nil {
		return nil, err
	}
	if err := e.mergeInto(p); err != nil {
		return nil, err
	}
	return p, nil
}

// RenderWithConvertor renders K8s manifests with specific converter
func (m *Manifest) RenderWithConvertor(c converter.Converter, runc *runConfig) (map[string]string, error) {
	errSg := m.UI.StepGroup()
	defer errSg.Done()

	if _, err := m.CalculateSourcesBaseOverride(); err != nil {
		renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, err)
		return nil, err
	}

	filteredEnvs, err := m.GetEnvironments(runc.Envs)
	if err != nil {
		renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, err)
		return nil, err
	}

	rendered := map[string][]byte{}
	projects := map[string]*composego.Project{}
	files := map[string][]string{}
	sourcesFiles := m.GetSourcesFiles()

	for _, env := range filteredEnvs {
		p, err := m.MergeEnvIntoSources(env)
		if err != nil {
			wrappedErr := errors.Wrapf(err, "environment %s, details:\n", env.Name)
			renderStepError(m.UI, errSg.Add(""), renderStepRenderOverlay, wrappedErr)
			return nil, wrappedErr
		}
		projects[env.Name] = p.Project
		files[env.Name] = append(sourcesFiles, env.File)
	}

	outputPaths, err := c.Render(runc.ManifestsAsSingleFile, runc.OutputDir, m.getWorkingDir(),
		projects, files, runc.AdditionalManifests, rendered, runc.ExcludeServicesByEnv)
	if err != nil {
		renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, err)
		return nil, err
	}

	if len(m.Skaffold) > 0 {
		// Update skaffold profiles upon render - this ensures profiles stay up to date
		if err := UpdateSkaffoldProfiles(m.Skaffold, outputPaths); err != nil {
			decoratedErr := errors.Errorf("Couldn't update skaffold.yaml profiles, details:\n%s", err)
			renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, decoratedErr)
			return nil, err
		}

		// Update skaffold build artifacts - these may change over time, usually by manual update in base docker compose
		composeProject, err := m.SourcesToComposeProject()
		if err != nil {
			decoratedErr := errors.Errorf("Couldn't build Docker Compose Project from tracked source files, details:\n%s", err)
			renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, decoratedErr)
			return nil, err
		}

		if err = UpdateSkaffoldBuildArtifacts(m.Skaffold, composeProject); err != nil {
			decoratedErr := errors.Errorf("Couldn't update skaffold.yaml build artifacts, details:\n%s", err)
			renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, decoratedErr)
			return nil, err
		}
	}

	return outputPaths, nil
}

// GetSourcesFiles gets the sources tracked docker-compose files.
func (m *Manifest) GetSourcesFiles() []string {
	return m.Sources.Files
}

// SourcesToComposeProject returns the manifests compose sources as a ComposeProject.
func (m *Manifest) SourcesToComposeProject() (*ComposeProject, error) {
	return m.Sources.toComposeProject()
}

func ManifestExistsForPath(manifestPath string) bool {
	_, err := os.Stat(manifestPath)
	return err == nil
}

// getWorkingDir gets the sources working directory.
func (m *Manifest) getWorkingDir() string {
	return m.Sources.getWorkingDir()
}

// getSourcesOverride gets the sources calculated override.
func (m *Manifest) getSourcesOverride() *composeOverride {
	override := m.Sources.override
	override.UI = m.UI
	return override
}

// minifyK8sExtensionsToBaseAttributes removes all attributes except a selected few deemed
// most useful for users to configure immediately.
func minifyK8sExtensionsToBaseAttributes(override *composeOverride) error {
	for _, svc := range override.Services {
		minifiedSvcExt, err := config.MinifySvcK8sExtension(svc.Extensions)
		if err != nil {
			return err
		}
		svc.Extensions[config.K8SExtensionKey] = minifiedSvcExt
	}

	for _, vol := range override.Volumes {
		minifiedVolExt, err := config.MinifyVolK8sExtension(vol.Extensions)
		if err != nil {
			return err
		}
		vol.Extensions[config.K8SExtensionKey] = minifiedVolExt
	}

	return nil
}
