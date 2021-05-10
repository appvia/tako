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

package config

import (
	"bytes"
	"fmt"
	"regexp"

	composego "github.com/compose-spec/compose-go/types"
	"github.com/go-playground/validator/v10"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const resourceQuantityPattern = `^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`

var resourceQuantityRegex = regexp.MustCompile(resourceQuantityPattern)

// VolumeExtension represents the root of the docker-compose extensions for a volume
type VolumeExtension struct {
	K8S VolK8sConfig `yaml:"x-k8s"`
}

// VolK8sConfig represents the root of the k8s specific fields supported by kev.
type VolK8sConfig struct {
	Size         string `yaml:"size,omitempty" validate:"required,quantity"`
	StorageClass string `yaml:"storageClass,omitempty"`
	Selector     string `yaml:"selector,omitempty"`
}

// Merge merges in a src volume's K8s config
func (vkc VolK8sConfig) Merge(src VolK8sConfig) (VolK8sConfig, error) {
	if err := mergo.Merge(&vkc, src, mergo.WithOverride); err != nil {
		return VolK8sConfig{}, err
	}
	return vkc, nil
}

// Map converts a VolK8sConfig config into a map
func (vkc VolK8sConfig) Map() (map[string]interface{}, error) {
	bs, err := yaml.Marshal(vkc)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	return m, yaml.Unmarshal(bs, &m)
}

// Validate validates a volumes K8s config
func (vkc VolK8sConfig) Validate() error {
	validate := validator.New()

	if err := validate.RegisterValidation("quantity", validateResourceQuantity); err != nil {
		return err
	}

	if err := validate.Struct(vkc); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, e := range validationErrors {
			if e.Tag() == "required" {
				return fmt.Errorf("%s is required", e.StructNamespace())
			}

			if e.Tag() == "quantity" {
				return fmt.Errorf(
					"%s is invalid, use a resource quantity format, e.g. 10M, 10Gi, 10Mi",
					e.StructNamespace(),
				)
			}
		}
		return errors.New(validationErrors[0].Error())
	}

	return nil
}

// DefaultVolK8sConfig returns a volume's K8s config with set defaults.
func DefaultVolK8sConfig() VolK8sConfig {
	return VolK8sConfig{
		Size:         DefaultVolumeSize,
		StorageClass: DefaultVolumeStorageClass,
	}
}

// VolK8sConfigFromCompose returns a VolK8sConfig from a compose-go VolumeConfig
func VolK8sConfigFromCompose(vol *composego.VolumeConfig) (VolK8sConfig, error) {
	cfg := DefaultVolK8sConfig()
	volFromMap, err := ParseVolK8sConfigFromMap(vol.Extensions)
	if err != nil {
		return VolK8sConfig{}, err
	}

	cfg, err = cfg.Merge(volFromMap)
	if err != nil {
		return VolK8sConfig{}, err
	}

	if err := cfg.Validate(); err != nil {
		return VolK8sConfig{}, err
	}

	return cfg, nil
}

// ParseVolK8sConfigFromMap parses a volume extension from the related map
func ParseVolK8sConfigFromMap(m map[string]interface{}) (VolK8sConfig, error) {
	if _, ok := m[K8SExtensionKey]; !ok {
		return VolK8sConfig{}, nil
	}

	var ext VolumeExtension

	var buf bytes.Buffer
	if err := yaml.NewEncoder(&buf).Encode(m); err != nil {
		return VolK8sConfig{}, err
	}

	if err := yaml.NewDecoder(&buf).Decode(&ext); err != nil {
		return VolK8sConfig{}, err
	}

	return ext.K8S, nil
}

// validateResourceQuantity validates a value conforms to a quantity
// e.g. 40Mi, 128Gi, 129M
// See:
// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/resources.md#resource-quantities
func validateResourceQuantity(fl validator.FieldLevel) bool {
	quantity := fl.Field().String()
	return resourceQuantityRegex.MatchString(quantity)
}
