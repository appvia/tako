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
	DefaultVolumeSize = "100Mi"

	// DefaultVolumeStorageClass default PV storage class
	DefaultVolumeStorageClass = ""

	// DefaultService is a default service
	DefaultService = NoService

	// DefaultRestartPolicy is a default restart policy
	DefaultRestartPolicy = RestartPolicyAlways

	// DefaultWorkload is a defauld workload type
	DefaultWorkload = DeploymentWorkload

	// DefaultServiceAccountName is a default SA to be used
	DefaultServiceAccountName = "default"

	// DefaultImagePullPolicy default image pull policy
	DefaultImagePullPolicy = "IfNotPresent"

	// DefaultImagePullSecret default image pull credentials secret name
	DefaultImagePullSecret = ""

	// DefaultSecurityContextRunAsUser default UID for pod security context
	DefaultSecurityContextRunAsUser = ""

	// DefaultSecurityContextRunAsGroup default GID for pod security context
	DefaultSecurityContextRunAsGroup = ""

	// DefaultSecurityContextFsGroup default fs Group for pod security context
	DefaultSecurityContextFsGroup = ""

	// NoService default value
	NoService = "None"

	// NodePortService svc type
	NodePortService = "NodePort"

	// LoadBalancerService svc type
	LoadBalancerService = "LoadBalancer"

	// ClusterIPService svc type
	ClusterIPService = "ClusterIP"

	// HeadlessService svc type
	HeadlessService = "Headless"

	// RestartPolicyAlways default value
	RestartPolicyAlways = "Always"

	// RestartPolicyOnFailure restart policy
	RestartPolicyOnFailure = "OnFailure"

	// RestartPolicyNever restart policy
	RestartPolicyNever = "Never"

	// DeploymentWorkload workload type
	DeploymentWorkload = "Deployment"

	// DaemonsetWorkload workload type
	DaemonsetWorkload = "DaemonSet"

	// StatefulsetWorkload workload type
	StatefulsetWorkload = "StatefulSet"

	// JobWorkload workload type
	JobWorkload = "Job"

	// DefaultReplicaNumber default number of replicas per workload
	DefaultReplicaNumber = 1

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

	// DefaultLivenessProbeCommand default command
	DefaultLivenessProbeCommand = "Define healthcheck command for service %s"

	// DefaultLivenessProbeTimeout default 10s
	DefaultLivenessProbeTimeout = "10s"

	// DefaultLivenessProbeInterval default 1m (1 minute)
	DefaultLivenessProbeInterval = "1m"

	// DefaultLivenessProbeInitialDelay default 1m (1 minute)
	DefaultLivenessProbeInitialDelay = "1m"

	// DefaultLivenessProbeRetries default 3. Number of retries for liveness probe command
	DefaultLivenessProbeRetries = 3

	// DefaultLivenessProbeDisable default false. Enabled by default
	DefaultLivenessProbeDisable = false
)
