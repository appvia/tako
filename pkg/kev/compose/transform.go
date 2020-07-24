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

package compose

import (
	"fmt"
	"time"

	composego "github.com/compose-spec/compose-go/types"
	"github.com/imdario/mergo"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Transform func(*VersionedProject) error

// AugmentOrAddDeploy augments a service's existing deploy block or attaches a new one with default presets.
func AugmentOrAddDeploy(x *VersionedProject) error {
	var updated composego.Services
	err := x.WithServices(x.ServiceNames(), func(svc composego.ServiceConfig) error {
		var deploy = createDeploy()

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
func HealthCheckBase(x *VersionedProject) error {
	var updated composego.Services
	err := x.WithServices(x.ServiceNames(), func(svc composego.ServiceConfig) error {
		if svc.HealthCheck == nil {
			check := createHealthCheck(svc.Name)
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
func ExternaliseSecrets(x *VersionedProject) error {
	noSecrets := len(x.Secrets) < 1
	if noSecrets {
		return nil
	}

	updated := make(map[string]composego.SecretConfig)
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
func ExternaliseConfigs(x *VersionedProject) error {
	noConfigs := len(x.Configs) < 1
	if noConfigs {
		return nil
	}

	updated := make(map[string]composego.ConfigObjConfig)
	for key, config := range x.Configs {
		config.File = ""
		config.External.External = true
		updated[key] = config
	}

	x.Configs = updated
	return nil
}

// createDeploy returns a deploy block with configured presets.
func createDeploy() composego.DeployConfig {
	replica := uint64(DefaultReplicaNumber)
	parallelism := uint64(defaultRollingUpdateMaxSurge)

	lm, _ := resource.ParseQuantity(defaultResourceLimitMem)
	rm, _ := resource.ParseQuantity(defaultResourceRequestMem)

	defaultMemLimit, _ := lm.AsInt64()
	defaultMemReq, _ := rm.AsInt64()

	return composego.DeployConfig{
		Replicas: &replica,
		Mode:     "replicated",
		Resources: composego.Resources{
			Limits: &composego.Resource{
				NanoCPUs:    defaultResourceLimitCPU,
				MemoryBytes: composego.UnitBytes(defaultMemLimit),
			},
			Reservations: &composego.Resource{
				NanoCPUs:    defaultResourceRequestCPU,
				MemoryBytes: composego.UnitBytes(defaultMemReq),
			},
		},
		UpdateConfig: &composego.UpdateConfig{
			Parallelism: &parallelism,
		},
	}
}

// createHealthCheck returns a healthcheck block with configured placeholders.
func createHealthCheck(svcName string) composego.HealthCheckConfig {
	testMsg := fmt.Sprintf(defaultLivenessProbeCommand, svcName)
	to, _ := time.ParseDuration(defaultLivenessProbeTimeout)
	iv, _ := time.ParseDuration(defaultLivenessProbeInterval)
	sp, _ := time.ParseDuration(defaultLivenessProbeInitialDelay)
	timeout := composego.Duration(to)
	interval := composego.Duration(iv)
	startPeriod := composego.Duration(sp)
	retries := uint64(defaultLivenessProbeRetries)

	return composego.HealthCheckConfig{
		Test:        []string{"\"CMD\"", "\"echo\"", testMsg},
		Timeout:     &timeout,
		Interval:    &interval,
		Retries:     &retries,
		StartPeriod: &startPeriod,
		Disable:     defaultLivenessProbeDisable,
	}
}
