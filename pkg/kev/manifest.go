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
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// NewManifest returns a new Manifest struct.
func NewManifest(files []string, workingDir string) (*Manifest, error) {
	s, err := newSources(files, workingDir)
	if err != nil {
		return nil, err
	}
	return &Manifest{Sources: s}, nil
}

// LoadManifest returns application manifests.
func LoadManifest(workingDir string) (*Manifest, error) {
	data, err := ioutil.ReadFile(path.Join(workingDir, ManifestName))
	if err != nil {
		return nil, err
	}
	var m *Manifest
	return m, yaml.Unmarshal(data, &m)
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

// CalculateSourcesBaseOverride extracts the base override from the manifest's docker-compose source files.
func (m *Manifest) CalculateSourcesBaseOverride(opts ...BaseOverrideOpts) (*Manifest, error) {
	if err := m.Sources.CalculateBaseOverride(opts...); err != nil {
		return nil, err
	}
	return m, nil
}

// MintEnvironments create new environments based on candidate environments and manifest base labels.
// If no environments are provided, a default environment will be created.
func (m *Manifest) MintEnvironments(candidates []string) *Manifest {
	fileNameTemplate := m.GetEnvironmentFileNameTemplate()

	m.Environments = Environments{}
	if len(candidates) == 0 {
		candidates = append(candidates, defaultEnv)
	}

	override := m.getSourcesOverride().toBaseLabels()
	for _, env := range candidates {
		m.Environments = append(m.Environments, &Environment{
			Name:     env,
			override: override,
			File:     path.Join(m.getWorkingDir(), fmt.Sprintf(fileNameTemplate, env)),
		})
	}
	return m
}

// GetEnvironmentFileNameTemplate returns environment file name template to match
// the naming convention of the first compose source file
func (m *Manifest) GetEnvironmentFileNameTemplate() string {
	firstSrc := filepath.Base(m.Sources.Files[0])
	parts := strings.Split(firstSrc, ".")
	ext := parts[len(parts)-1]
	return strings.ReplaceAll(firstSrc, ext, "kev.%s."+ext)
}

// ReconcileConfig reconciles config changes with docker-compose sources against deployment environments.
func (m *Manifest) ReconcileConfig(reporter io.Writer) (*Manifest, error) {
	if _, err := m.CalculateSourcesBaseOverride(withEnvVars); err != nil {
		return nil, err
	}

	sourcesOverride := m.getSourcesOverride()
	for _, e := range m.Environments {
		if err := e.reconcile(sourcesOverride, reporter); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// MergeEnvIntoSources merges an environment into a parsed instance of the tracked docker-compose sources.
// It returns the merged ComposeProject.
func (m *Manifest) MergeEnvIntoSources(e *Environment) (*ComposeProject, error) {
	e.prepareForMergeUsing(m.getSourcesOverride())

	p, err := m.sourcesToComposeProject()
	if err != nil {
		return nil, err
	}
	if err := e.mergeInto(p); err != nil {
		return nil, err
	}
	return p, nil
}

// DetectSecretsInSources detects any potential secrets setup as environment variables in a manifests sources.
func (m *Manifest) DetectSecretsInSources(matchers []map[string]string, reporter io.Writer) error {
	sourcesFiles := m.GetSourcesFiles()
	p, err := NewComposeProject(sourcesFiles)
	if err != nil {
		return err
	}

	candidates := Services{}
	for _, s := range p.Services {
		candidates = append(candidates, ServiceConfig{Name: s.Name, Environment: s.Environment})
	}

	_, _ = reporter.Write([]byte(fmt.Sprintf("\n✓ Detecting secret leaks in sources [%s]\n", sourcesFiles)))
	return candidates.detectSecrets(matchers, reporter)
}

// DetectSecretsInEnvs detects any potential secrets setup as environment variables
// in a manifests deployment environments config.
func (m *Manifest) DetectSecretsInEnvs(matchers []map[string]string, reporter io.Writer) error {
	var filter []string
	envs, err := m.GetEnvironments(filter)
	if err != nil {
		return err
	}

	for _, env := range envs {
		_, _ = reporter.Write([]byte(fmt.Sprintf("\n✓ Detecting secret leaks in env [%s]\n", env.Name)))
		err := env.GetServices().detectSecrets(matchers, reporter)
		if err != nil {
			return err
		}
	}
	return nil
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
	return m.Sources.override
}

// sourcesToComposeProject returns the manifests compose sources as a ComposeProject.
func (m *Manifest) sourcesToComposeProject() (*ComposeProject, error) {
	return m.Sources.toComposeProject()
}
