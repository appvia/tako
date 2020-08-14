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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"sort"

	"gopkg.in/yaml.v3"
)

// NewManifest returns a new Manifest struct
func NewManifest(files []string, workingDir string) (*Manifest, error) {
	s, err := newSources(files, workingDir)
	if err != nil {
		return nil, err
	}
	return &Manifest{Sources: s}, nil
}

// LoadManifest returns application manifests
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

// EnvironmentsAsMap returns filtered app environments
// If no filter is provided all app environments will be returned
func (m *Manifest) EnvironmentsAsMap(filter []string) (map[string]string, error) {
	sort.Strings(filter)
	environments := map[string]string{}

	if len(filter) == 0 {
		for _, e := range m.Environments {
			environments[e.Name] = e.File
		}
	} else {
		for _, e := range m.Environments {
			i := sort.SearchStrings(filter, e.Name)
			if i < len(filter) && filter[i] == e.Name {
				environments[e.Name] = e.File
			}
		}
	}

	if len(environments) == 0 {
		return nil, errors.New("No environments found")
	}

	return environments, nil
}

// CalculateSourcesBaseOverlay extracts the base overlay from the manifest's docker-compose source files.
func (m *Manifest) CalculateSourcesBaseOverlay(opts ...BaseOverlayOpts) (*Manifest, error) {
	if err := m.Sources.CalculateBaseOverlay(opts...); err != nil {
		return nil, err
	}
	return m, nil
}

// MintEnvironments create new environments based on candidate environments and manifest base labels.
// If no environments are provided, a default environment will be created.
func (m *Manifest) MintEnvironments(candidates []string) *Manifest {
	m.Environments = Environments{}
	if len(candidates) == 0 {
		candidates = append(candidates, defaultEnv)
	}
	for _, env := range candidates {
		m.Environments = append(m.Environments, &Environment{
			Name:    env,
			overlay: m.GetSourcesLabels(),
			File:    path.Join(m.GetWorkingDir(), fmt.Sprintf(configFileTemplate, env)),
		})
	}
	return m
}

func (m *Manifest) GetWorkingDir() string {
	return m.Sources.GetWorkingDir()
}

func (m *Manifest) GetSourcesLabels() *composeOverlay {
	return m.Sources.overlay
}
func (m *Manifest) GetSourcesFiles() []string {
	return m.Sources.Files
}

func (m *Manifest) ReconcileConfig(reporter io.Writer) (*Manifest, error) {
	if _, err := m.CalculateSourcesBaseOverlay(withEnvVars); err != nil {
		return nil, err
	}

	// spew.Dump(m.Sources.overlay)

	for _, e := range m.Environments {
		if err := e.reconcile(m.GetSourcesLabels(), reporter); err != nil {
			return nil, err
		}
	}

	return m, nil
}
