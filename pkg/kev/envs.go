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
	"sort"

	"gopkg.in/yaml.v3"
)

// MarshalYAML makes Environments implement yaml.Marshaler.
func (e Environments) MarshalYAML() (interface{}, error) {
	out := map[string]string{}
	for _, env := range e {
		out[env.Name] = env.File
	}
	return out, nil
}

// UnmarshalYAML makes Environments implement yaml.UnmarshalYAML.
func (e *Environments) UnmarshalYAML(value *yaml.Node) error {
	for i := 0; i < len(value.Content); i += 2 {
		env, err := loadEnvironment(value.Content[i].Value, value.Content[i+1].Value)
		if err != nil {
			return err
		}
		*e = append(*e, env)
	}
	return nil
}

// MarshalJSON makes Environments implement json.Marshaler.
func (e Environments) MarshalJSON() ([]byte, error) {
	data, err := e.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(data, "", "  ")
}

// GetVersion gets the environment's overlay version.
func (e *Environment) GetVersion() string {
	return e.overlay.Version
}

// GetServices gets the environment's overlay services.
func (e *Environment) GetServices() Services {
	var out = make([]ServiceConfig, len(e.overlay.Services))
	copy(out, e.overlay.Services)
	return out
}

// GetService retrieves the specific service by name from the environment's overlay.
func (e *Environment) GetService(name string) (ServiceConfig, error) {
	return e.overlay.getService(name)
	// for _, s := range e.GetServices() {
	// 	if s.Name == name {
	// 		return s, nil
	// 	}
	// }
	// return ServiceConfig{}, fmt.Errorf("no such service: %s", name)
}

// GetEnvVarsForService retrieves the env vars for a specific service from the environment's overlay.
func (e *Environment) GetEnvVarsForService(name string) (map[string]*string, error) {
	s, err := e.GetService(name)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*string)
	for k, v := range s.Environment {
		out[k] = v
	}
	return out, nil
}

// GetVolumes gets the environment's overlay volumes.
func (e *Environment) GetVolumes() Volumes {
	out := make(Volumes)
	for k, v := range e.overlay.Volumes {
		out[k] = v
	}
	return out
}

// VolumeNames return names for all volumes from the environment's overlay volumes.
func (e *Environment) VolumeNames() []string {
	var out []string
	for k := range e.GetVolumes() {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// GetVolume retrieves a specific volume by name from the environment's overlay volumes.
func (e *Environment) GetVolume(name string) (VolumeConfig, error) {
	return e.overlay.getVolume(name)
	// for k, v := range e.GetVolumes() {
	// 	if k == name {
	// 		return v, nil
	// 	}
	// }
	// return VolumeConfig{}, fmt.Errorf("no such volume: %s", name)
}

// WriteTo writes out an environment to a writer.
// The Environment struct implements the io.WriterTo interface.
func (e *Environment) WriteTo(w io.Writer) (n int64, err error) {
	data, err := MarshalIndent(e.overlay, 2)
	if err != nil {
		return int64(0), err
	}
	written, err := w.Write(data)
	return int64(written), err
}

func (e *Environment) loadOverlay() (*Environment, error) {
	p, err := NewComposeProject([]string{e.File})
	if err != nil {
		return nil, err
	}

	var services Services
	for _, name := range p.ServiceNames() {
		s, err := p.GetService(name)
		if err != nil {
			return nil, err
		}
		envVarsFromNilToBlankInService(s)
		services = append(services, ServiceConfig{Name: s.Name, Labels: s.Labels, Environment: s.Environment})
	}
	e.overlay = &composeOverlay{
		Version:  p.GetVersion(),
		Services: services,
	}
	extractVolumesLabels(p, e.overlay)
	return e, nil
}

func (e *Environment) reconcile(overlay *composeOverlay, reporter io.Writer) error {
	_, _ = reporter.Write([]byte(fmt.Sprintf("✓ Reconciling environment [%s]\n", e.Name)))

	labelsMatching := overlay.toBaseLabelsMatching(e.overlay)
	cset := labelsMatching.diff(e.overlay)
	if cset.HasNoPatches() {
		_, _ = reporter.Write([]byte(fmt.Sprint(" → nothing to update\n")))
		return nil
	}

	e.patch(cset, reporter)
	return nil
}

func (e *Environment) patch(cset changeset, reporter io.Writer) {
	e.overlay.patch(cset, reporter)
}

func (e *Environment) mergeInto(p *ComposeProject) error {
	return e.overlay.mergeInto(p)
}

func loadEnvironment(name, file string) (*Environment, error) {
	e := &Environment{
		Name: name,
		File: file,
	}
	return e.loadOverlay()
}
