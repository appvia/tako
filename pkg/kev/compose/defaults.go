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

const (
	// DefaultReplicaNumber default number of replicas per workload
	DefaultReplicaNumber = 1

	// defaultRollingUpdateMaxSurge default number of containers to be updated at a time
	defaultRollingUpdateMaxSurge = 1

	// defaultResourceLimitMem default Memory resource limit
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory
	defaultResourceLimitMem = "500Mi"

	// defaultResourceRequestMem default Memory resource request
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory
	defaultResourceRequestMem = "10Mi"

	// Kubernetes notation details: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu
	// Default: 0.5, which is equivalent to 50% of CPU
	defaultResourceLimitCPU = "0.5"

	// defaultResourceRequestCPU default CPU resource request
	// This value follows docker compose resource notation
	// https://docs.docker.com/compose/compose-file/#resources
	// Kubernetes notation details: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu
	// Default: 0.1, which is equivalent to 10% of CPU
	defaultResourceRequestCPU = "0.1"

	// defaultLivenessProbeCommand default command
	defaultLivenessProbeCommand = "Define healthcheck command for service %s"

	// defaultLivenessProbeTimeout default 10s
	defaultLivenessProbeTimeout = "10s"

	// defaultLivenessProbeInterval default 1m (1 minute)
	defaultLivenessProbeInterval = "1m"

	// defaultLivenessProbeInitialDelay default 1m (1 minute)
	defaultLivenessProbeInitialDelay = "1m"

	// defaultLivenessProbeRetries default 3. Number of retries for liveness probe command
	defaultLivenessProbeRetries = 3

	// defaultLivenessProbeDisable default false. Enabled by default
	defaultLivenessProbeDisable = false
)
