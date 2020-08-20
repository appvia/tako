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

	composego "github.com/compose-spec/compose-go/types"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// MarshalYAML makes Services implement yaml.Marshaller
func (s Services) MarshalYAML() (interface{}, error) {
	services := map[string]ServiceConfig{}
	for _, service := range s {
		services[service.Name] = service
	}
	return services, nil
}

// MarshalJSON makes Services implement json.Marshaler
func (s Services) MarshalJSON() ([]byte, error) {
	data, err := s.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(data, "", "  ")
}

// GetLabels gets a service's labels
func (sc ServiceConfig) GetLabels() map[string]string {
	return sc.Labels
}

func (o *composeOverlay) diff(other *composeOverlay) (changeset, error) {
	d, _ := diff.NewDiffer()
	clog, err := d.Diff(other, o)
	if err != nil {
		return changeset{}, err
	}
	return newChangeset(clog)
}

func (o *composeOverlay) patch(cset changeset, reporter io.Writer) {
	cset.applyVersionPatchesIfAny(o, reporter)
	cset.applyServicesPatchesIfAny(o, reporter)
	cset.applyVolumesPatchesIfAny(o, reporter)
}

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

		zeroValueUnassignedEnvVarsInService(base)

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
		if err := mergo.Merge(&base.Labels, &override.Labels, mergo.WithOverride); err != nil {
			return errors.Wrapf(err, "cannot merge labels for volume %s", name)
		}
		p.Volumes[name] = base
	}
	return nil
}
