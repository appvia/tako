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

package transform

import (
	"fmt"

	"github.com/appvia/kube-devx/pkg/kev/defaults"
	"github.com/appvia/kube-devx/pkg/kev/utils"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/goccy/go-yaml"
)

// Transform is a transform func type.
// Documents how a transform func should be created.
// Useful as a function param for functions that accept transforms.
type Transform func(data []byte) ([]byte, error)

// DeployWithDefaults attaches a deploy block with presets to any service
// missing a deploy block.
func DeployWithDefaults(data []byte) ([]byte, error) {
	x, err := utils.UnmarshallComposeConfig(data)
	if err != nil {
		return []byte{}, err
	}

	var updated compose.Services

	err = x.WithServices(x.ServiceNames(), func(svc compose.ServiceConfig) error {
		if svc.Deploy == nil {
			svc.Deploy = defaults.Deploy()
		}
		updated = append(updated, svc)
		return nil
	})
	if err != nil {
		return []byte{}, err
	}

	x.Services = updated
	return yaml.Marshal(x)
}

// HealthCheckBase attaches a base healthcheck  block with placeholders to be updated by users
// to any service missing a healthcheck block.
func HealthCheckBase(data []byte) ([]byte, error) {
	x, err := utils.UnmarshallComposeConfig(data)
	if err != nil {
		return []byte{}, err
	}

	var updated compose.Services
	err = x.WithServices(x.ServiceNames(), func(svc compose.ServiceConfig) error {
		if svc.HealthCheck == nil {
			svc.HealthCheck = defaults.HealthCheck(svc.Name)
		}
		updated = append(updated, svc)
		return nil
	})
	if err != nil {
		return []byte{}, err
	}

	x.Services = updated
	return yaml.Marshal(x)
}

// ExternaliseSecrets ensures that all top level secrets are set to external
// to specify that the secrets have already been created.
func ExternaliseSecrets(data []byte) ([]byte, error) {
	x, err := utils.UnmarshallComposeConfig(data)
	if err != nil {
		return []byte{}, err
	}

	noSecrets := len(x.Secrets) < 1
	if noSecrets {
		return data, nil
	}

	updated := make(map[string]compose.SecretConfig)
	for key, config := range x.Secrets {
		config.File = ""
		config.External.External = true
		updated[key] = config
	}

	x.Secrets = updated
	return yaml.Marshal(x)
}

// ExternaliseConfigs ensures that all top level configs are set to external
// to specify that the configs have already been created.
func ExternaliseConfigs(data []byte) ([]byte, error) {
	x, err := utils.UnmarshallComposeConfig(data)
	if err != nil {
		return []byte{}, err
	}

	noConfigs := len(x.Configs) < 1
	if noConfigs {
		return data, nil
	}

	updated := make(map[string]compose.ConfigObjConfig)
	for key, config := range x.Configs {
		config.File = ""
		config.External.External = true
		updated[key] = config
	}

	x.Configs = updated
	return yaml.Marshal(x)
}

// Echo can be used to view data at different stages of
// a transform pipeline.
func Echo(data []byte) ([]byte, error) {
	x, err := utils.UnmarshallComposeConfig(data)
	if err != nil {
		return []byte{}, err
	}
	fmt.Println(string(data))
	return yaml.Marshal(x)
}
