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
	"time"

	"github.com/appvia/kev/pkg/kev/converter"
	"github.com/appvia/kev/pkg/kev/log"
	"github.com/appvia/kev/pkg/kev/terminal"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// NewManifest returns a new Manifest struct.
// func NewManifest(sources *Sources, workingDir string) (*Manifest, error) {
func NewNewManifest(sources *Sources) *Manifest {
	return &Manifest{
		Id:      uuid.New().String(),
		Sources: sources,
	}
}

func NewManifest(files []string, workingDir string) (*Manifest, error) {
	s, err := newSources(files, workingDir)
	if err != nil {
		return nil, err
	}
	return &Manifest{
		Id:      uuid.New().String(),
		Sources: s,
	}, nil
}

// LoadManifest returns application manifests.
func LoadManifest(workingDir string) (*Manifest, error) {
	data, err := ioutil.ReadFile(path.Join(workingDir, ManifestFilename))
	if err != nil {
		return nil, err
	}
	var m *Manifest
	return m, yaml.Unmarshal(data, &m)
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
		var step terminal.Step
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
		step.Success(time.Second * 2)
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
func (m *Manifest) ReconcileConfig() (*Manifest, error) {
	if _, err := m.CalculateSourcesBaseOverride(withEnvVars); err != nil {
		return nil, err
	}
	sourcesOverride := m.getSourcesOverride()
	for _, e := range m.Environments {
		if err := e.reconcile(sourcesOverride); err != nil {
			return nil, err
		}
	}

	return m, nil
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
func (m *Manifest) RenderWithConvertor(c converter.Converter, outputDir string, singleFile bool, envs []string, excluded map[string][]string) (
	*Manifest,
	error,
) {
	if _, err := m.CalculateSourcesBaseOverride(); err != nil {
		return nil, err
	}

	filteredEnvs, err := m.GetEnvironments(envs)
	if err != nil {
		return nil, err
	}

	rendered := map[string][]byte{}
	projects := map[string]*composego.Project{}
	files := map[string][]string{}
	sourcesFiles := m.GetSourcesFiles()

	for _, env := range filteredEnvs {
		p, err := m.MergeEnvIntoSources(env)
		if err != nil {
			return nil, errors.Wrap(err, "Couldn't calculate compose project representation")
		}
		projects[env.Name] = p.Project
		files[env.Name] = append(sourcesFiles, env.File)
	}

	outputPaths, err := c.Render(singleFile, outputDir, m.getWorkingDir(), projects, files, rendered, excluded)
	if err != nil {
		log.Errorf("Couldn't render manifests")
		return nil, err
	}

	if len(m.Skaffold) > 0 {
		if err := UpdateSkaffoldProfiles(m.Skaffold, outputPaths); err != nil {
			log.Errorf("Couldn't update skaffold.yaml profiles")
			return nil, err
		}
	}
	return m, nil
}

// DetectSecretsInSources detects any potential secrets setup as environment variables in a manifests sources.
func (m *Manifest) DetectSecretsInSources(matchers []map[string]string) error {
	sourcesFiles := m.GetSourcesFiles()

	sg := m.UI.StepGroup()

	for _, source := range sourcesFiles {
		m.UI.Output(fmt.Sprintf("Detecting secrets in: %s", source))
		p, err := NewComposeProject([]string{source})
		if err != nil {
			return err
		}

		for _, s := range p.Services {
			step := sg.Add(fmt.Sprintf("Analysing service: %s", s.Name))
			serviceConfig := ServiceConfig{Name: s.Name, Environment: s.Environment}

			hits := serviceConfig.detectSecretsInEnvVars(matchers)
			if len(hits) == 0 {
				step.Success(time.Second*5, "Non detected in service: ", s.Name)
				continue
			}

			step.Warning(time.Second*5, "Detected in service: ", s.Name)
			for _, hit := range hits {
				m.UI.Output(
					fmt.Sprintf("env var [%s] - %s", hit.envVar, hit.description),
					terminal.WithStyle(terminal.LogStyle),
					terminal.WithIndentChar(terminal.LogIndentChar),
					terminal.WithIndent(3),
				)
			}
		}
	}

	// p, err := NewComposeProject(sourcesFiles)
	// if err != nil {
	// 	return err
	// }
	//
	// for _, f := range sourcesFiles {
	// 	m.UI.Output(fmt.Sprintf("Detecting secrets in: %s", f))
	// }

	// var matches []secretHit
	// for _, svc := range s {
	// 	matches = append(matches, svc.detectSecretsInEnvVars(matchers)...)
	// }
	//
	// if len(matches) == 0 {
	// 	return false
	// }
	//
	// detectedFn()
	// for _, m := range matches {
	// 	log.Warnf("Service [%s], env var [%s] looks like a secret", m.svcName, m.envVar)
	// }
	// return true

	// sg := m.UI.StepGroup()
	// candidates := Services{}
	// for _, s := range p.Services {
	// 	step := sg.Add(fmt.Sprintf("Analysing service: %s", s.Name))
	// 	serviceConfig := ServiceConfig{Name: s.Name, Environment: s.Environment}
	//
	// 	hits := serviceConfig.detectSecretsInEnvVars(matchers)
	// 	if len(hits) == 0 {
	// 		step.Success(time.Second*5, "Non detected in service: ", s.Name)
	// 		continue
	// 	}
	//
	// 	step.Warning(time.Second*5, "Detected in service: ", s.Name)
	// 	for _, hit := range hits {
	// 		m.UI.Output(
	// 			fmt.Sprintf("env var [%s] - %s", hit.envVar, hit.description),
	// 			terminal.WithStyle(terminal.LogStyle),
	// 			terminal.WithIndentChar(terminal.LogIndentChar),
	// 			terminal.WithIndent(3),
	// 		)
	// 	}
	// 	// candidates = append(candidates, serviceConfig)
	// }

	// detected := candidates.detectSecrets(matchers, func() {
	// 	log.Warnf("Detected potential secrets in sources %s", sourcesFiles)
	// })

	// candidates := Services{}
	// for _, s := range p.Services {
	// 	candidates = append(candidates, ServiceConfig{Name: s.Name, Environment: s.Environment})
	// }
	//
	// detected := candidates.detectSecrets(matchers, func() {
	// 	log.Warnf("Detected potential secrets in sources %s", sourcesFiles)
	// })

	// if !detected {
	// 	log.Debug("No secrets detected in project sources")
	// }

	return nil
}

// DetectSecretsInEnvs detects any potential secrets setup as environment variables
// in a manifests deployment environments config.
func (m *Manifest) DetectSecretsInEnvs(matchers []map[string]string) error {
	var filter []string
	envs, err := m.GetEnvironments(filter)
	if err != nil {
		return err
	}

	for _, env := range envs {
		detected := env.GetServices().detectSecrets(matchers, func() {
			log.Warnf("Detected potential secrets in env [%s]", env.Name)
		})
		if !detected {
			log.Debugf("No secrets detected in env [%s]", env.Name)
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

// SourcesToComposeProject returns the manifests compose sources as a ComposeProject.
func (m *Manifest) SourcesToComposeProject() (*ComposeProject, error) {
	return m.Sources.toComposeProject()
}

func ManifestExistsForPath(manifestPath string) bool {
	_, err := os.Stat(manifestPath)
	return err == nil
}

func EnsureFirstInit(wd string) error {
	manifestPath := path.Join(wd, ManifestFilename)
	if ManifestExistsForPath(manifestPath) {
		absWd, _ := filepath.Abs(wd)
		err := fmt.Errorf("%s already exists at: %s", ManifestFilename, absWd)
		return err
	}
	return nil
}
