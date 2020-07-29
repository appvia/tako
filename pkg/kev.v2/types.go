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
	"encoding/json"
	"fmt"
	"io"
	"path"
	"path/filepath"

	composego "github.com/compose-spec/compose-go/types"
)

type composeProject struct {
	version string
	*composego.Project
}

type labels struct {
	Version  string `yaml:"version,omitempty" json:"version,omitempty"`
	Services composego.Services
	Volumes  composego.Volumes
}

type Manifest struct {
	Sources      []string     `yaml:"compose,omitempty" json:"compose,omitempty"`
	Environments Environments `yaml:"environments,omitempty" json:"environments,omitempty"`
	labels       labels
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
func (m *Manifest) GetEnvironment(name string) (Environment, error) {
	for _, env := range m.Environments {
		if env.Name == name {
			return env, nil
		}
	}
	return Environment{}, fmt.Errorf("no such environment: %s", name)
}

// ExtractLabels extracts the base set of labels from the manifest's docker-compose source files.
func (m *Manifest) ExtractLabels() (*Manifest, error) {
	ready, err := newComposeProject(m.Sources)
	if err != nil {
		return nil, err
	}
	m.labels = extractLabels(ready)
	return m, nil
}

// extractLabels same as ExtractLabels but works on a compose project object.
func extractLabels(c *composeProject) labels {
	out := labels{
		Version: c.version,
	}
	extractVolumesLabels(c, &out)

	for _, s := range c.Services {
		target := composego.ServiceConfig{
			Name:   s.Name,
			Labels: map[string]string{},
		}
		setDefaultLabels(&target)
		extractServiceTypeLabels(s, &target)
		extractDeploymentLabels(s, &target)
		extractHealthcheckLabels(s, &target)
		out.Services = append(out.Services, target)
	}
	return out
}

// MintEnvironments create new environments based on candidate environments and manifest base labels.
// If no environments are provided, a default environment will be created.
func (m *Manifest) MintEnvironments(candidates []string) *Manifest {
	m.Environments = Environments{}
	if len(candidates) == 0 {
		candidates = append(candidates, defaultEnv)
	}
	for _, env := range candidates {
		m.Environments = append(m.Environments, Environment{
			Name:    env,
			content: m.labels,
			File:    path.Join(m.GetWorkingDir(), fmt.Sprintf(configFileTemplate, env)),
		})
	}
	return m
}

func (m *Manifest) GetWorkingDir() string {
	if len(m.Sources) < 1 {
		return ""
	}
	return filepath.Dir(m.Sources[0])
}

type Environments []Environment

// MarshalYAML makes Environments implement yaml.Marshaler.
func (e Environments) MarshalYAML() (interface{}, error) {
	out := map[string]string{}
	for _, env := range e {
		out[env.Name] = env.File
	}
	return out, nil
}

// MarshalJSON makes Environments implement json.Marshaler.
func (e Environments) MarshalJSON() ([]byte, error) {
	data, err := e.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(data, "", "  ")
}

type Environment struct {
	Name    string `yaml:"-" json:"-"`
	File    string `yaml:"-" json:"-"`
	content labels
}

// WriteTo writes out an environment to a writer.
// The Environment struct implements the io.WriterTo interface.
func (e Environment) WriteTo(w io.Writer) (n int64, err error) {
	data, err := MarshalIndent(e.content, 2)
	if err != nil {
		return int64(0), err
	}
	written, err := w.Write(data)
	return int64(written), err
}
