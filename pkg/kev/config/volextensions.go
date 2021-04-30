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
	K8S K8sVol `yaml:"x-k8s"`
}

// K8sVol represents the root of the k8s specific fields supported by kev.
type K8sVol struct {
	Size         string `yaml:"size,omitempty" validate:"required,quantity"`
	StorageClass string `yaml:"storageClass,omitempty"`
	Selector     string `yaml:"selector,omitempty"`
}

// Merge merges a src K8s volume config
func (k K8sVol) Merge(src K8sVol) (K8sVol, error) {
	if err := mergo.Merge(&k, src, mergo.WithOverride); err != nil {
		return K8sVol{}, err
	}
	return k, nil
}

// Map converts a K8sVol config into a map
func (k K8sVol) Map() (map[string]interface{}, error) {
	bs, err := yaml.Marshal(k)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	return m, yaml.Unmarshal(bs, &m)
}

func (k K8sVol) Validate() error {
	validate := validator.New()

	if err := validate.RegisterValidation("quantity", validateResourceQuantity); err != nil {
		return err
	}

	if err := validate.Struct(k); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, e := range validationErrors {
			if e.Tag() == "required" {
				return fmt.Errorf("%s is required", e.StructNamespace())
			}

			if e.Tag() == "quantity" {
				return fmt.Errorf(
					"%s is invalid, use a resource quantity format, e.g. 129M, 10Gi, 123Mi",
					e.StructNamespace(),
				)
			}
		}
		return errors.New(validationErrors[0].Error())
	}

	return nil
}

// DefaultK8sVol returns a K8s Volume config with all the defaults set into it.
func DefaultK8sVol() K8sVol {
	return K8sVol{
		Size:         DefaultVolumeSize,
		StorageClass: DefaultVolumeStorageClass,
	}
}

// K8sVolFromCompose returns a K8sVol from a compose-go VolumeConfig
func K8sVolFromCompose(vol *composego.VolumeConfig) (K8sVol, error) {
	cfg := DefaultK8sVol()
	volFromMap, err := ParseK8sVolFromMap(vol.Extensions)
	if err != nil {
		return K8sVol{}, err
	}

	cfg, err = cfg.Merge(volFromMap)
	if err != nil {
		return K8sVol{}, err
	}

	if err := cfg.Validate(); err != nil {
		return K8sVol{}, err
	}

	return cfg, nil
}

// ParseK8sVolFromMap parses a volume extension from the related map
func ParseK8sVolFromMap(m map[string]interface{}) (K8sVol, error) {
	if _, ok := m[K8SExtensionKey]; !ok {
		return K8sVol{}, nil
	}

	var ext VolumeExtension

	var buf bytes.Buffer
	if err := yaml.NewEncoder(&buf).Encode(m); err != nil {
		return K8sVol{}, err
	}

	if err := yaml.NewDecoder(&buf).Decode(&ext); err != nil {
		return K8sVol{}, err
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
