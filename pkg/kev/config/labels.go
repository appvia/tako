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
	// LabelComponentEnabled toggles the project service component
	LabelComponentEnabled = "kev.component.enabled"

	// LabelWorkloadType defines type of workload controller
	LabelWorkloadType = "kev.workload.type"

	// LabelWorkloadReplicas represents number of replicas for given workload. Takes presedence if defined.
	LabelWorkloadReplicas = "kev.workload.replicas"

	// LabelWorkloadAutoscaleMaxReplicas represents maximum number of replicas for given workload. Used in Horizontal Pod Autoscaler.
	LabelWorkloadAutoscaleMaxReplicas = "kev.workload.autoscale-max-replicas"

	// LabelWorkloadAutoscaleCPUUtilizationThreshold represents a maximum CPU utilization for given workload that instructs horizontal pod autoscaler.
	LabelWorkloadAutoscaleCPUUtilizationThreshold = "kev.workload.autoscale-cpu-threshold"

	// LabelWorkloadAutoscaleMemoryUtilizationThreshold represents a maximum Memory utilization for given workload that instructs horizontal pod autoscaler.
	LabelWorkloadAutoscaleMemoryUtilizationThreshold = "kev.workload.autoscale-mem-threshold"

	// LabelWorkloadRollingUpdateMaxSurge max number of nodes updated at once
	LabelWorkloadRollingUpdateMaxSurge = "kev.workload.rolling-update-max-surge"

	// LabelWorkloadMemory defines Memory request for workload
	LabelWorkloadMemory = "kev.workload.memory"

	// LabelWorkloadCPU defines CPU request for workload
	LabelWorkloadCPU = "kev.workload.cpu"

	// LabelWorkloadMaxMemory defines max Memory limit for workload
	LabelWorkloadMaxMemory = "kev.workload.max-memory"

	// LabelWorkloadMaxCPU defines max CPU limit for workload
	LabelWorkloadMaxCPU = "kev.workload.max-cpu"

	// LabelWorkloadSecurityContextRunAsUser sets pod security context RunAsUser attribute
	LabelWorkloadSecurityContextRunAsUser = "kev.workload.pod-security-run-as-user"

	// LabelWorkloadSecurityContextRunAsGroup sets pod security context RunAsGroup attribute
	LabelWorkloadSecurityContextRunAsGroup = "kev.workload.pod-security-run-as-group"

	// LabelWorkloadSecurityContextFsGroup sets pod security context FsGroup attribute
	LabelWorkloadSecurityContextFsGroup = "kev.workload.pod-security-fs-group"

	// LabelWorkloadImagePullPolicy defines when to pull images from registry
	LabelWorkloadImagePullPolicy = "kev.workload.image-pull-policy"

	// LabelWorkloadImagePullSecret defines docker registry image pull secret
	LabelWorkloadImagePullSecret = "kev.workload.image-pull-secret"

	// LabelWorkloadRestartPolicy defines when to restart a pod
	LabelWorkloadRestartPolicy = "kev.workload.restart-policy"

	// LabelWorkloadServiceAccountName defines service account name to be used by the workload
	LabelWorkloadServiceAccountName = "kev.workload.service-account-name"

	// LabelWorkloadLivenessProbeCommand defines the command for workload liveness probe
	LabelWorkloadLivenessProbeCommand = "kev.workload.liveness-probe-command"

	// LabelWorkloadLivenessProbeInterval defines the interval for workload liveness probe
	LabelWorkloadLivenessProbeInterval = "kev.workload.liveness-probe-interval"

	// LabelWorkloadLivenessProbeTimeout defines the timeout for workload liveness probe
	LabelWorkloadLivenessProbeTimeout = "kev.workload.liveness-probe-timeout"

	// LabelWorkloadLivenessProbeInitialDelay defines the initial delay for workload liveness probe
	LabelWorkloadLivenessProbeInitialDelay = "kev.workload.liveness-probe-initial-delay"

	// LabelWorkloadLivenessProbeRetries defines number of times workload liveness probe will retry
	LabelWorkloadLivenessProbeRetries = "kev.workload.liveness-probe-retries"

	// LabelWorkloadLivenessProbeDisabled disables workload liveness probe
	LabelWorkloadLivenessProbeDisabled = "kev.workload.liveness-probe-disabled"

	// LabelWorkloadReadinessProbeCommand defines the command for workload liveness probe
	LabelWorkloadReadinessProbeCommand = "kev.workload.readiness-probe-command"

	// LabelWorkloadReadinessProbeInterval defines the interval for workload liveness probe
	LabelWorkloadReadinessProbeInterval = "kev.workload.readiness-probe-interval"

	// LabelWorkloadReadinessProbeTimeout defines the timeout for workload liveness probe
	LabelWorkloadReadinessProbeTimeout = "kev.workload.readiness-probe-timeout"

	// LabelWorkloadReadinessProbeInitialDelay defines the initial delay for workload liveness probe
	LabelWorkloadReadinessProbeInitialDelay = "kev.workload.readiness-probe-initial-delay"

	// LabelWorkloadReadinessProbeRetries defines number of times workload liveness probe will retry
	LabelWorkloadReadinessProbeRetries = "kev.workload.readiness-probe-retries"

	// LabelWorkloadReadinessProbeDisabled disables workload liveness probe
	LabelWorkloadReadinessProbeDisabled = "kev.workload.readiness-probe-disabled"

	// LabelServiceType defines the type of service to be created
	LabelServiceType = "kev.service.type"

	// LabelServiceNodePortPort defines port number for NodePort k8s service kind
	LabelServiceNodePortPort = "kev.service.nodeport.port"

	// LabelServiceExpose informs whether K8s service should be exposed externally. To enable set as "true" or "domain.com,otherdomain.com".
	LabelServiceExpose = "kev.service.expose"

	// LabelServiceExposeTLSSecret  provides the name of the TLS secret to use with the Kubernetes ingress controller
	LabelServiceExposeTLSSecret = "kev.service.expose.tls-secret"

	// LabelVolumeSize defines persistent volume size
	LabelVolumeSize = "kev.volume.size"

	// LabelVolumeSelector defines persistent volume selector
	LabelVolumeSelector = "kev.volume.selector"

	// LabelVolumeStorageClass defines persistent volume storage class
	LabelVolumeStorageClass = "kev.volume.storage-class"
)

var BaseServiceLabels = []string{
	LabelWorkloadLivenessProbeCommand,
	LabelWorkloadReplicas,
}

var BaseVolumeLabels = []string{
	LabelVolumeSize,
}
