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

	"github.com/appvia/kev/pkg/kev/config"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

// MarshalYAML makes Services implement yaml.Marshaller.
func (s Services) MarshalYAML() (interface{}, error) {
	services := map[string]ServiceConfig{}
	for _, service := range s {
		services[service.Name] = service
	}
	return services, nil
}

// MarshalJSON makes Services implement json.Marshaler.
func (s Services) MarshalJSON() ([]byte, error) {
	data, err := s.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(data, "", "  ")
}

// Map converts services to a map.
func (s Services) Map() map[string]ServiceConfig {
	out := map[string]ServiceConfig{}
	for _, service := range s {
		out[service.Name] = service
	}
	return out
}

// Set converts services to a set.
func (s Services) Set() map[string]bool {
	out := map[string]bool{}
	for _, service := range s {
		out[service.Name] = true
	}
	return out
}

// GetLabels gets a service's labels
func (sc ServiceConfig) GetLabels() map[string]string {
	return sc.Labels
}

// minusEnvVars returns a copy of the ServiceConfig with blank env vars
func (sc ServiceConfig) minusEnvVars() ServiceConfig {
	return ServiceConfig{
		Name:        sc.Name,
		Labels:      sc.Labels,
		Environment: map[string]*string{},
	}
}

// toBaseLabels returns a copy of the ServiceConfig with only condensed base service labels
func (sc ServiceConfig) toBaseLabels() ServiceConfig {
	for key, _ := range sc.GetLabels() {
		if !contains(config.BaseServiceLabels, key) {
			delete(sc.Labels, key)
		}
	}

	return ServiceConfig{
		Name:        sc.Name,
		Labels:      sc.Labels,
		Environment: sc.Environment,
	}
}

// toBaseLabels returns a copy of the VolumeConfig with only condensed base volume labels
func (vc VolumeConfig) toBaseLabels() VolumeConfig {
	for key := range vc.Labels {
		if !contains(config.BaseVolumeLabels, key) {
			delete(vc.Labels, key)
		}
	}

	return VolumeConfig{
		Name:   vc.Name,
		Labels: vc.Labels,
	}
}

// diff detects changes between an overlay against another overlay.
func (o *composeOverlay) toBaseLabels() *composeOverlay {
	var services Services
	volumes := Volumes{}

	for _, svcConfig := range o.Services {
		services = append(services, svcConfig.toBaseLabels())
	}
	for key, volConfig := range o.Volumes {
		volumes[key] = volConfig.toBaseLabels()
	}

	return &composeOverlay{Version: o.Version, Services: services, Volumes: volumes}
}

// diff detects changes between an overlay against another overlay.
func (o *composeOverlay) diff(other *composeOverlay) changeset {
	return newChangeset(other, o)
}

// patch patches an overlay based on the supplied changeset patches.
func (o *composeOverlay) patch(cset changeset, reporter io.Writer) {
	cset.applyVersionPatchesIfAny(o, reporter)
	cset.applyServicesPatchesIfAny(o, reporter)
	cset.applyVolumesPatchesIfAny(o, reporter)
}

// mergeInto merges an overlay onto a compose project.
// For env vars, it enforces the expected docker-compose CLI behaviour.
func (o *composeOverlay) mergeInto(p *ComposeProject) error {
	if err := o.mergeServicesInto(p); err != nil {
		return errors.Wrap(err, "cannot merge services into project")
	}
	if err := o.mergeVolumesInto(p); err != nil {
		return errors.Wrap(err, "cannot merge volumes into project")
	}
	return nil
}

func (o *composeOverlay) mergeServicesInto(p *ComposeProject) error {
	var overridden composego.Services
	for _, override := range o.Services {
		base, err := p.GetService(override.Name)
		if err != nil {
			return err
		}

		envVarsFromNilToBlankInService(base)

		if err := mergo.Merge(&base.Labels, &override.Labels, mergo.WithOverride); err != nil {
			return errors.Wrapf(err, "cannot merge labels for service %s", override.Name)
		}
		if err := mergo.Merge(&base.Environment, &override.Environment, mergo.WithOverride); err != nil {
			return errors.Wrapf(err, "cannot merge env vars for service %s", override.Name)
		}
		overridden = append(overridden, base)
	}
	p.Services = overridden
	return nil
}

func (o *composeOverlay) mergeVolumesInto(p *ComposeProject) error {
	for name, override := range o.Volumes {
		base, ok := p.Volumes[name]
		if !ok {
			return fmt.Errorf("could not find volume %s", override.Name)
		}

		if err := mergo.Merge(&base.Labels, &override.Labels, mergo.WithOverwriteWithEmptyValue); err != nil {
			return errors.Wrapf(err, "cannot merge labels for volume %s", name)
		}
		p.Volumes[name] = base
	}
	return nil
}

// contains returns true of slice of strings contains a given string
func contains(strs []string, s string) bool {
	sort.Strings(strs)
	i := sort.SearchStrings(strs, s)
	return i < len(strs) && strs[i] == s
}
