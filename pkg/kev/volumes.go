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

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/xeipuuv/gojsonschema"
)

func newVolumeConfig(name string, p *ComposeProject) (VolumeConfig, error) {
	cfg := VolumeConfig{
		Labels: p.Volumes[name].Labels,
	}
	return cfg, cfg.validate()
}

func (vc VolumeConfig) validate() error {
	ls := gojsonschema.NewGoLoader(config.VolumesSchema)
	ld := gojsonschema.NewGoLoader(vc.Labels)

	result, err := gojsonschema.Validate(ls, ld)
	if err != nil {
		return err
	}

	if !result.Valid() {
		return errors.New(result.Errors()[0].Description())
	}
	return nil
}

// condenseLabels returns a copy of the VolumeConfig with only condensed base volume labels
func (vc VolumeConfig) condenseLabels(labels []string) VolumeConfig {
	for key := range vc.Labels {
		if !contains(labels, key) {
			delete(vc.Labels, key)
		}
	}

	return VolumeConfig{
		Name:       vc.Name,
		Labels:     vc.Labels,
		Extensions: vc.Extensions,
	}
}
