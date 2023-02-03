/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
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

package tako

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/appvia/tako/pkg/tako/log"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
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

// toWritableResults returns the environments as WritableResults.
func (e Environments) toWritableResults() WritableResults {
	var out []WritableResult
	for _, environment := range e {
		out = append(out, WritableResult{
			WriterTo: environment,
			FilePath: environment.File,
		})
	}
	return out
}

// Write is a convenience method to write out the environment's overrides to disk
func (e Environments) Write() error {
	return e.toWritableResults().Write()
}

// GetVersion gets the environment's override version.
func (e *Environment) GetVersion() string {
	return e.override.Version
}

// GetServices gets the environment's override services.
func (e *Environment) GetServices() Services {
	var out = make([]ServiceConfig, len(e.override.Services))
	copy(out, e.override.Services)
	return out
}

// GetService retrieves the specific service by name from the environment's override.
func (e *Environment) GetService(name string) (ServiceConfig, error) {
	return e.override.getService(name)
}

// UpdateExtensions updates a service's extensions. Any new extensions included will be created.
func (e *Environment) UpdateExtensions(svcName string, ext map[string]interface{}) error {
	if _, err := e.GetService(svcName); err != nil {
		return err
	}

	var services Services
	for _, svc := range e.GetServices() {
		if svc.Name == svcName {
			if err := mergo.Merge(&svc.Extensions, ext, mergo.WithOverride); err != nil {
				return err
			}
		}
		services = append(services, svc)
	}
	e.override.Services = services
	return nil
}

// RemoveExtension removes an extension from a service's extensions using its key.
func (e *Environment) RemoveExtension(svcName string, key string) error {
	if _, err := e.GetService(svcName); err != nil {
		return err
	}

	var services Services
	for _, svc := range e.GetServices() {
		if svc.Name == svcName {
			delete(svc.Extensions, key)
		}
		services = append(services, svc)
	}
	e.override.Services = services
	return nil
}

// GetEnvVarsForService retrieves the env vars for a specific service from the environment's override.
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

// GetVolumes gets the environment's override volumes.
func (e *Environment) GetVolumes() Volumes {
	out := make(Volumes)
	for k, v := range e.override.Volumes {
		out[k] = v
	}
	return out
}

// VolumeNames return names for all volumes from the environment's override volumes.
func (e *Environment) VolumeNames() []string {
	var out []string
	for k := range e.GetVolumes() {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// GetVolume retrieves a specific volume by name from the environment's override volumes.
func (e *Environment) GetVolume(name string) (VolumeConfig, error) {
	return e.override.getVolume(name)
}

// ToSources converts an environment to a sources object.
func (e *Environment) ToSources() *Sources {
	return &Sources{
		Files:    []string{e.File},
		override: e.override,
	}
}

// WriteTo writes out an environment to a writer.
// The Environment struct implements the io.WriterTo interface.
func (e *Environment) WriteTo(w io.Writer) (n int64, err error) {
	data, err := MarshalIndent(e.override, 2)
	if err != nil {
		return int64(0), err
	}
	written, err := w.Write(data)
	return int64(written), err
}

func (e *Environment) loadOverride() (*Environment, error) {
	p, err := NewComposeProject([]string{e.File})
	if err != nil {
		return nil, errors.Errorf("%s\nsee compose file: %s", err.Error(), e.File)
	}

	var services Services
	for _, name := range p.ServiceNames() {
		s, err := p.GetService(name)
		if err != nil {
			return nil, err
		}
		envVarsFromNilToBlankInService(s)
		serviceConfig, err := newServiceConfig(s)
		if err != nil {
			log.Debugf("cannot load environment [%s], service [%s]: err %s", e.Name, name, err.Error())
		}
		services = append(services, serviceConfig)
	}
	volumes := Volumes{}
	for _, volName := range p.VolumeNames() {
		volumeConfig, err := newVolumeConfig(volName, p)
		if err != nil {
			return nil, errors.Wrapf(err, "Cannot load environment [%s], volume [%s]", e.Name, volName)
		}
		volumes[volName] = volumeConfig
	}
	e.override = &composeOverride{
		Version:  p.GetVersion(),
		Services: services,
		Volumes:  volumes,
	}
	return e, nil
}

func (e *Environment) mergeInto(p *ComposeProject) error {
	return e.override.mergeInto(p)
}

func loadEnvironment(name, file string) (*Environment, error) {
	e := &Environment{
		Name: name,
		File: file,
	}
	return e.loadOverride()
}
