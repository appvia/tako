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
	"github.com/appvia/kube-devx/pkg/kev/defaults"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/imdario/mergo"
)

type Transform func(*compose.Project) error

// AugmentOrAddDeploy augments a service's existing deploy block or attaches a new one with default presets.
func AugmentOrAddDeploy(x *compose.Project) error {
	var updated compose.Services
	err := x.WithServices(x.ServiceNames(), func(svc compose.ServiceConfig) error {
		var deploy = defaults.Deploy()

		if svc.Deploy != nil {
			if err := mergo.Merge(&deploy, svc.Deploy, mergo.WithOverride); err != nil {
				return err
			}
		}

		svc.Deploy = &deploy
		updated = append(updated, svc)
		return nil
	})
	if err != nil {
		return err
	}

	x.Services = updated

	return nil
}

// HealthCheckBase attaches a base healthcheck  block with placeholders to be updated by users
// to any service missing a healthcheck block.
func HealthCheckBase(x *compose.Project) error {
	var updated compose.Services
	err := x.WithServices(x.ServiceNames(), func(svc compose.ServiceConfig) error {
		if svc.HealthCheck == nil {
			check := defaults.HealthCheck(svc.Name)
			svc.HealthCheck = &check
		}
		updated = append(updated, svc)
		return nil
	})
	if err != nil {
		return err
	}

	x.Services = updated
	return nil
}

// ExternaliseSecrets ensures that all top level secrets are set to external
// to specify that the secrets have already been created.
func ExternaliseSecrets(x *compose.Project) error {
	noSecrets := len(x.Secrets) < 1
	if noSecrets {
		return nil
	}

	updated := make(map[string]compose.SecretConfig)
	for key, config := range x.Secrets {
		config.File = ""
		config.External.External = true
		updated[key] = config
	}

	x.Secrets = updated
	return nil
}

// ExternaliseConfigs ensures that all top level configs are set to external
// to specify that the configs have already been created.
func ExternaliseConfigs(x *compose.Project) error {
	noConfigs := len(x.Configs) < 1
	if noConfigs {
		return nil
	}

	updated := make(map[string]compose.ConfigObjConfig)
	for key, config := range x.Configs {
		config.File = ""
		config.External.External = true
		updated[key] = config
	}

	x.Configs = updated
	return nil
}
