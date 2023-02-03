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

package tako

import (
	"encoding/json"
	"path/filepath"

	"github.com/appvia/tako/pkg/tako/config"
	"github.com/appvia/tako/pkg/tako/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type BaseOverrideOpts func(s *Sources, c *ComposeProject) error

// CalculateBaseOverride calculates the extensions deduced from a group of compose sources.
func (s *Sources) CalculateBaseOverride(opts ...BaseOverrideOpts) error {
	ready, err := NewComposeProject(s.Files, WithTransforms)
	if err != nil {
		return errors.Errorf("%s\nsee compose files: %v", err.Error(), s.Files)
	}

	s.override = &composeOverride{
		Version: ready.version,
		Volumes: map[string]VolumeConfig{},
	}

	if err := extractVolumesExtensions(ready, s.override); err != nil {
		return err
	}

	for _, svc := range ready.Services {
		target := ServiceConfig{
			Name:       svc.Name,
			Extensions: svc.Extensions,
		}

		k8sConf, err := config.SvcK8sConfigFromCompose(&svc)
		if err != nil {
			return err
		}

		m, err := k8sConf.Map()
		if err != nil {
			return err
		}

		if target.Extensions == nil {
			target.Extensions = make(map[string]interface{})
		}
		target.Extensions[config.K8SExtensionKey] = m

		s.override.Services = append(s.override.Services, target)
	}

	for _, opt := range opts {
		if err := opt(s, ready); err != nil {
			log.Debug(err.Error())
			return err
		}
	}

	return nil
}

// extractVolumesExtensions adds a k8s extension to each defined volume.
// Every extension contains default k8s settings.
func extractVolumesExtensions(c *ComposeProject, out *composeOverride) error {
	for _, v := range c.VolumeNames() {
		vol := c.Volumes[v]

		k8sVol, err := config.VolK8sConfigFromCompose(&vol)
		if err != nil {
			return nil
		}

		target := VolumeConfig{
			Extensions: vol.Extensions,
		}

		if target.Extensions == nil {
			target.Extensions = make(map[string]interface{})
		}

		m, err := k8sVol.Map()
		if err != nil {
			return err
		}

		target.Extensions[config.K8SExtensionKey] = m

		out.Volumes[v] = target
	}

	return nil
}

// withEnvVars attaches the sources env vars to the base override
func withEnvVars(s *Sources, origin *ComposeProject) error {
	var services Services
	for _, svc := range s.override.Services {
		originSvc, err := origin.GetService(svc.Name)
		if err != nil {
			return err
		}

		envVarsFromNilToBlankInService(originSvc)

		services = append(services, ServiceConfig{
			Name:        svc.Name,
			Environment: originSvc.Environment,
			Extensions:  svc.Extensions,
		})
	}
	s.override.Services = services
	return nil
}

// MarshalYAML makes Sources implement yaml.Marshaler.
func (s *Sources) MarshalYAML() (interface{}, error) {
	var out []string
	out = append(out, s.Files...)
	return out, nil
}

// UnmarshalYAML makes Sources implement yaml.UnmarshalYAML.
func (s *Sources) UnmarshalYAML(value *yaml.Node) error {
	for i := 0; i < len(value.Content); i += 1 {
		s.Files = append(s.Files, value.Content[i].Value)
	}
	return nil
}

// MarshalJSON makes Sources implement json.Marshaler.
func (s *Sources) MarshalJSON() ([]byte, error) {
	data, err := s.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(data, "", "  ")
}

func (s *Sources) getWorkingDir() string {
	if len(s.Files) < 1 {
		return ""
	}
	return filepath.Dir(s.Files[0])
}

func (s *Sources) toComposeProject() (*ComposeProject, error) {
	return NewComposeProject(s.Files)
}
