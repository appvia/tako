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
	"path"
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

// NewManifest returns a new Manifest struct.
func NewManifest(sources *Sources) *Manifest {
	return &Manifest{
		Id:      uuid.New().String(),
		Sources: sources,
	}
}

// LoadManifest returns application manifests.
func LoadManifest(workingDir string) (*Manifest, error) {
	data, err := ioutil.ReadFile(path.Join(workingDir, ManifestFilename))
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

// GetManifestName returns base manifest file name (without extension)
func GetManifestName() string {
	return strings.TrimSuffix(ManifestFilename, filepath.Ext(ManifestFilename))
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

// MintEnvironments create new environments based on candidate environments and manifest base labels.
// This includes an implicit sandbox environment that will always be created.
func (m *Manifest) MintEnvironments(candidates []string) *Manifest {
	m.UI.Header("Creating deployment environments...")
	sg := m.UI.StepGroup()
	defer sg.Done()

	fileNameTemplate := m.GetEnvironmentFileNameTemplate()

	m.Environments = Environments{}
	if !contains(candidates, SandboxEnv) {
		candidates = append([]string{SandboxEnv}, candidates...)
	}

	override := m.getSourcesOverride().toBaseLabels()
	for _, env := range candidates {
		envFilename := path.Join(m.getWorkingDir(), fmt.Sprintf(fileNameTemplate, GetManifestName(), env))
		var step kmd.Step
		if env == SandboxEnv {
			step = sg.Add(fmt.Sprintf("Creating the %s sandbox env file: %s", SandboxEnv, envFilename))
		} else {
			step = sg.Add(fmt.Sprintf("Creating the %s env file: %s", env, envFilename))
		}

		m.Environments = append(m.Environments, &Environment{
			Name:     env,
			override: override,
			File:     envFilename,
		})
		step.Success()
	}
	return m
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
		renderStepError(m.UI, sg.Add(""), renderStepReconcile, err)
		return nil, err
	}

	sourcesOverride := m.getSourcesOverride()
	filteredEnvs, err := m.GetEnvironments(envs)
	if err != nil {
		sg := m.UI.StepGroup()
		defer sg.Done()
		renderStepError(m.UI, sg.Add(""), renderStepReconcile, err)
		return nil, err
	}

	for _, e := range filteredEnvs {
		if err := validateExtensions(e.override.Services); err != nil {
			sg := m.UI.StepGroup()
			defer sg.Done()
			renderStepError(m.UI, sg.Add(""), renderStepReconcile, err)
			return nil, err
		}

		log.DebugTitlef("Reconciling environment [%s]", e.Name)

		m.UI.Output(fmt.Sprintf("%s: %s", e.Name, e.File))

		sourcesOverride.
			toLabelsMatching(e.override).
			diffAndPatch(e.override)
	}

	return m, nil
}

func validateExtensions(services Services) error {
	for _, s := range services {
		_, err := config.ParseSvcK8sConfigFromMap(s.Extensions, config.RequireExtensions())
		if err != nil {
			return errors.Wrapf(err, "%s extensions not valid for service %s", config.K8SExtensionKey, s.Name)
		}
	}

	return nil
}

// MergeEnvIntoSources merges an environment into a parsed instance of the tracked docker-compose sources.
// It returns the merged ComposeProject.
func (m *Manifest) MergeEnvIntoSources(e *Environment) (*ComposeProject, error) {
	e.prepareForMergeUsing(m.getSourcesOverride())

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
func (m *Manifest) RenderWithConvertor(c converter.Converter, outputDir string, singleFile bool, envs []string, excluded map[string][]string) (map[string]string, error) {
	errSg := m.UI.StepGroup()
	defer errSg.Done()

	if _, err := m.CalculateSourcesBaseOverride(); err != nil {
		renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, err)
		return nil, err
	}

	filteredEnvs, err := m.GetEnvironments(envs)
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

	outputPaths, err := c.Render(singleFile, outputDir, m.getWorkingDir(), projects, files, rendered, excluded)
	if err != nil {
		log.Errorf("Couldn't render manifests")
		renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, err)
		return nil, err
	}

	if len(m.Skaffold) > 0 {
		// Update skaffold profiles upon render - this ensures profiles stay up to date
		if err := UpdateSkaffoldProfiles(m.Skaffold, outputPaths); err != nil {
			log.Errorf("Couldn't update skaffold.yaml profiles")
			decoratedErr := errors.Errorf("Couldn't update skaffold.yaml profiles, details:\n%s", err)
			renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, decoratedErr)
			return nil, err
		}

		// Update skaffold build artifacts - these may change over time, usually by manual update in base docker compose
		composeProject, err := m.SourcesToComposeProject()
		if err != nil {
			log.Errorf("Couldn't build Docker Compose Project from tracked source files")
			decoratedErr := errors.Errorf("Couldn't build Docker Compose Project from tracked source files, details:\n%s", err)
			renderStepError(m.UI, errSg.Add(""), renderStepRenderGeneral, decoratedErr)
			return nil, err
		}

		if err = UpdateSkaffoldBuildArtifacts(m.Skaffold, composeProject); err != nil {
			log.Errorf("Couldn't update skaffold.yaml build artifacts")
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

// SourcesToComposeProject returns the manifests compose sources as a ComposeProject.
func (m *Manifest) SourcesToComposeProject() (*ComposeProject, error) {
	return m.Sources.toComposeProject()
}

func ManifestExistsForPath(manifestPath string) bool {
	_, err := os.Stat(manifestPath)
	return err == nil
}
