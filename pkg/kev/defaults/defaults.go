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

package defaults

import (
	"fmt"
	"time"

	"github.com/appvia/kube-devx/pkg/kev/config"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/dustin/go-humanize"
)

// Deploy returns a deploy block with configured presets.
func Deploy() compose.DeployConfig {
	replica := uint64(config.DefaultReplicaNumber)
	parallelism := uint64(config.DefaultRollingUpdateMaxSurge)

	defaultMemLimit, _ := humanize.ParseBytes(config.DefaultResourceLimitMem)
	defaultMemReq, _ := humanize.ParseBytes(config.DefaultResourceRequestMem)

	return compose.DeployConfig{
		Replicas: &replica,
		Mode:     "replicated",
		Resources: compose.Resources{
			Limits: &compose.Resource{
				NanoCPUs:    config.DefaultResourceLimitCPU,
				MemoryBytes: compose.UnitBytes(defaultMemLimit),
			},
			Reservations: &compose.Resource{
				NanoCPUs:    config.DefaultResourceRequestCPU,
				MemoryBytes: compose.UnitBytes(defaultMemReq),
			},
		},
		UpdateConfig: &compose.UpdateConfig{
			Parallelism: &parallelism,
		},
	}
}

// HealthCheck returns a healthcheck block with configured placeholders.
func HealthCheck(svcName string) compose.HealthCheckConfig {
	testMsg := fmt.Sprintf(config.DefaultLivenessProbeCommand, svcName)
	to, _ := time.ParseDuration(config.DefaultLivenessProbeTimeout)
	iv, _ := time.ParseDuration(config.DefaultLivenessProbeInterval)
	sp, _ := time.ParseDuration(config.DefaultLivenessProbeInitialDelay)
	timeout := compose.Duration(to)
	interval := compose.Duration(iv)
	startPeriod := compose.Duration(sp)
	retries := uint64(config.DefaultLivenessProbeRetries)

	return compose.HealthCheckConfig{
		Test:        []string{"\"CMD\"", "\"echo\"", testMsg},
		Timeout:     &timeout,
		Interval:    &interval,
		Retries:     &retries,
		StartPeriod: &startPeriod,
		Disable:     config.DefaultLivenessProbeDisable,
	}
}
