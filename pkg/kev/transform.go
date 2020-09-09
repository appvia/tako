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
	"time"

	"github.com/appvia/kev/pkg/kev/config"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/imdario/mergo"
	"k8s.io/apimachinery/pkg/api/resource"
)

type transform func(*ComposeProject) error

// augmentOrAddDeploy augments a service's existing deploy block or attaches a new one with default presets.
func augmentOrAddDeploy(x *ComposeProject) error {
	var updated composego.Services
	err := x.WithServices(x.ServiceNames(), func(svc composego.ServiceConfig) error {
		deploy := createDeploy()

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

// augmentOrAddHealthCheck augments a service's existing healthcheck block or attaches a new one with default presets.
func augmentOrAddHealthCheck(x *ComposeProject) error {
	var updated composego.Services
	err := x.WithServices(x.ServiceNames(), func(svc composego.ServiceConfig) error {
		check := createHealthCheck(svc.Name)

		if svc.HealthCheck != nil {
			if err := mergo.Merge(&check, svc.HealthCheck, mergo.WithOverride); err != nil {
				return err
			}
		}

		svc.HealthCheck = &check
		updated = append(updated, svc)
		return nil
	})
	if err != nil {
		return err
	}

	x.Services = updated
	return nil
}

// createDeploy returns a deploy block with configured presets.
func createDeploy() composego.DeployConfig {
	replica := uint64(config.DefaultReplicaNumber)
	parallelism := uint64(config.DefaultRollingUpdateMaxSurge)

	lm, _ := resource.ParseQuantity(config.DefaultResourceLimitMem)
	rm, _ := resource.ParseQuantity(config.DefaultResourceRequestMem)

	defaultMemLimit, _ := lm.AsInt64()
	defaultMemReq, _ := rm.AsInt64()

	return composego.DeployConfig{
		Replicas: &replica,
		Mode:     "replicated",
		Resources: composego.Resources{
			Limits: &composego.Resource{
				NanoCPUs:    config.DefaultResourceLimitCPU,
				MemoryBytes: composego.UnitBytes(defaultMemLimit),
			},
			Reservations: &composego.Resource{
				NanoCPUs:    config.DefaultResourceRequestCPU,
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
	testMsg := fmt.Sprintf(config.DefaultLivenessProbeCommand, svcName)
	to, _ := time.ParseDuration(config.DefaultLivenessProbeTimeout)
	iv, _ := time.ParseDuration(config.DefaultLivenessProbeInterval)
	sp, _ := time.ParseDuration(config.DefaultLivenessProbeInitialDelay)
	timeout := composego.Duration(to)
	interval := composego.Duration(iv)
	startPeriod := composego.Duration(sp)
	retries := uint64(config.DefaultLivenessProbeRetries)

	return composego.HealthCheckConfig{
		Test:        []string{"CMD", "echo", testMsg},
		Timeout:     &timeout,
		Interval:    &interval,
		Retries:     &retries,
		StartPeriod: &startPeriod,
		Disable:     config.DefaultLivenessProbeDisable,
	}
}
