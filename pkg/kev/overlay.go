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
	"sort"

	"github.com/appvia/kev/pkg/kev/config"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

// getService retrieves the specific service by name from the overlay's services.
func (o *composeOverlay) getService(name string) (ServiceConfig, error) {
	for _, s := range o.Services {
		if s.Name == name {
			return s, nil
		}
	}
	return ServiceConfig{}, fmt.Errorf("no such service: %s", name)
}

// getVolume retrieves a specific volume by name from the overlay's volumes.
func (o *composeOverlay) getVolume(name string) (VolumeConfig, error) {
	for k, v := range o.Volumes {
		if k == name {
			return v, nil
		}
	}
	return VolumeConfig{}, fmt.Errorf("no such volume: %s", name)
}

// toBaseLabels returns a copy of the composeOverlay with condensed base labels for services and volumes.
func (o *composeOverlay) toBaseLabels() *composeOverlay {
	var services Services
	volumes := Volumes{}

	for _, svcConfig := range o.Services {
		services = append(services, svcConfig.condenseLabels(config.BaseServiceLabels))
	}
	for key, volConfig := range o.Volumes {
		volumes[key] = volConfig.condenseLabels(config.BaseVolumeLabels)
	}

	return &composeOverlay{Version: o.Version, Services: services, Volumes: volumes}
}

// toLabelsMatching condenses an overlay's labels to the same label keys found in the provided overlay.
func (o *composeOverlay) toLabelsMatching(other *composeOverlay) *composeOverlay {
	services := o.servicesWithLabelsMatching(other)
	volumes := o.volumesWithLabelsMatching(other)
	return &composeOverlay{Version: o.Version, Services: services, Volumes: volumes}
}

func (o *composeOverlay) servicesWithLabelsMatching(other *composeOverlay) Services {
	var services Services
	for _, svc := range o.Services {
		otherSvc, err := other.getService(svc.Name)
		if err != nil {
			services = append(services, svc)
			continue
		}
		services = append(services, svc.condenseLabels(keys(otherSvc.Labels)))
	}
	return services
}

func (o *composeOverlay) volumesWithLabelsMatching(other *composeOverlay) Volumes {
	volumes := Volumes{}
	for volKey, volConfig := range o.Volumes {
		otherVol, err := other.getVolume(volKey)
		if err != nil {
			volumes[volKey] = volConfig
			continue
		}
		volumes[volKey] = volConfig.condenseLabels(keys(otherVol.Labels))
	}
	return volumes
}

// expandLabelsFrom returns a copy of the compose overlay
// filling in gaps in services and volumes labels (keys and values) using the provided overlay.
func (o *composeOverlay) expandLabelsFrom(other *composeOverlay) *composeOverlay {
	services := o.servicesLabelsExpandedFrom(other)
	volumes := o.volumesLabelsExpandedFrom(other)
	return &composeOverlay{Version: o.Version, Services: services, Volumes: volumes}
}

func (o *composeOverlay) servicesLabelsExpandedFrom(other *composeOverlay) Services {
	var out Services
	for _, otherSvc := range other.Services {
		dstSvc, err := o.getService(otherSvc.Name)
		if err != nil {
			continue
		}
		for key, value := range otherSvc.GetLabels() {
			if _, ok := dstSvc.Labels[key]; !ok {
				dstSvc.Labels[key] = value
			}
		}
		out = append(out, dstSvc)
	}
	return out
}

func (o *composeOverlay) volumesLabelsExpandedFrom(other *composeOverlay) Volumes {
	out := Volumes{}
	for otherVolKey, otherVolConfig := range other.Volumes {
		dstVol, err := o.getVolume(otherVolKey)
		if err != nil {
			continue
		}

		for key, value := range otherVolConfig.Labels {
			if _, ok := dstVol.Labels[key]; !ok {
				dstVol.Labels[key] = value
			}
		}
		out[otherVolKey] = dstVol
	}
	return out
}

func (o *composeOverlay) overrideServiceTypeFrom(other *composeOverlay) {
	var services Services
	for _, otherSvc := range other.Services {
		dstSvc, err := o.getService(otherSvc.Name)
		if err != nil {
			continue
		}
		dstSvc.Labels[config.LabelServiceType] = otherSvc.Labels[config.LabelServiceType]
		services = append(services, dstSvc)
	}
	o.Services = services
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
func contains(src []string, s string) bool {
	sort.Strings(src)
	i := sort.SearchStrings(src, s)
	return i < len(src) && src[i] == s
}

// contains returns true of slice of strings contains a given string
func keys(src map[string]string) []string {
	out := make([]string, 0, len(src))
	for k := range src {
		out = append(out, k)
	}
	return out
}
