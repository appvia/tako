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

package config

const (
	// DefaultVolumeSize default value PV class
	DefaultVolumeSize = "1Gi"

	// DefaultVolumeStorageClass default PV storage class
	DefaultVolumeStorageClass = ""

	// DefaultService is a default service
	DefaultService = NoService

	// DefaultRestartPolicy is a default restart policy
	DefaultRestartPolicy = RestartPolicyAlways

	// DefaultWorkload is a default workload type
	DefaultWorkload = DeploymentWorkload

	// DefaultServiceAccountName is a default SA to be used
	DefaultServiceAccountName = "default"

	// DefaultImagePullPolicy default image pull policy
	DefaultImagePullPolicy = "IfNotPresent"

	// DefaultImagePullSecret default image pull credentials secret name
	DefaultImagePullSecret = ""

	// JobWorkload workload type
	JobWorkload = "Job"

	// DefaultReplicaNumber default number of replicas per workload
	DefaultReplicaNumber = 1

	// DefaultAutoscaleMaxReplicaNumber default maximum number of replicas per workload (used in Horizontal Pod Autoscaler)
	DefaultAutoscaleMaxReplicaNumber = 0

	// DefaultAutoscaleCPUThreshold default CPU utilization threshold (percentage) for the workload's Horizontal Pod Autoscaler
	DefaultAutoscaleCPUThreshold = 70

	// DefaultAutoscaleMemoryThreshold default Memory utilization threshold (percentage) for the workload's Horizontal Pod Autoscaler
	DefaultAutoscaleMemoryThreshold = 70

	// DefaultRollingUpdateMaxSurge default number of containers to be updated at a time
	DefaultRollingUpdateMaxSurge = 1

	// DefaultResourceLimitMem default Memory resource limit
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory
	DefaultResourceLimitMem = "500Mi"

	// DefaultResourceRequestMem default Memory resource request
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory
	DefaultResourceRequestMem = "10Mi"

	// DefaultResourceLimitCPU default CPU Limit
	// Kubernetes notation details: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu
	// Default: 0.5, which is equivalent to 50% of CPU
	DefaultResourceLimitCPU = "0.5"

	// DefaultResourceRequestCPU default CPU resource request
	// This value follows docker compose resource notation
	// https://docs.docker.com/compose/compose-file/#resources
	// Kubernetes notation details: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu
	// Default: 0.1, which is equivalent to 10% of CPU
	DefaultResourceRequestCPU = "0.1"

	// DefaultProbeTimeout default 10s
	DefaultProbeTimeout = "10s"

	// DefaultProbeInterval default 1m (1 minute)
	DefaultProbeInterval = "1m"

	// DefaultProbeInitialDelay default 1m (1 minute)
	DefaultProbeInitialDelay = "1m"

	// DefaultProbeFailureThreshold default 3. Defines the failure threshold (number of retries) for the workload before giving up.
	DefaultProbeFailureThreshold = 3

	// DefaultProbeSuccessThreshold default 1. Minimum consecutive successes for the probe to be considered successful
	DefaultProbeSuccessThreshold = 1

	// DefaultProbeDisable default false. Enabled by default
	DefaultProbeDisable = false
)

var (
	// DefaultSecurityContextRunAsUser default UID for pod security context
	DefaultSecurityContextRunAsUser *int64 = nil

	// DefaultSecurityContextRunAsGroup default GID for pod security context
	DefaultSecurityContextRunAsGroup *int64 = nil

	// DefaultSecurityContextFsGroup default fs Group for pod security context
	DefaultSecurityContextFsGroup *int64 = nil

	// DefaultLivenessProbeCommand default command
	DefaultLivenessProbeCommand = []string{"echo", "Define healthcheck command for service"}
)
