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
	LabelPrefix = "kev."

	// LabelComponentEnabled toggles the project service component
	LabelComponentEnabled = LabelPrefix + "component.enabled"

	// LabelWorkloadType defines type of workload controller
	LabelWorkloadType = LabelPrefix + "workload.type"

	// LabelWorkloadReplicas represents number of replicas for given workload. Takes presedence if defined.
	LabelWorkloadReplicas = LabelPrefix + "workload.replicas"

	// LabelWorkloadAutoscaleMaxReplicas represents maximum number of replicas for given workload. Used in Horizontal Pod Autoscaler.
	LabelWorkloadAutoscaleMaxReplicas = LabelPrefix + "workload.autoscale-max-replicas"

	// LabelWorkloadAutoscaleCPUUtilizationThreshold represents a maximum CPU utilization for given workload that instructs horizontal pod autoscaler.
	LabelWorkloadAutoscaleCPUUtilizationThreshold = LabelPrefix + "workload.autoscale-cpu-threshold"

	// LabelWorkloadAutoscaleMemoryUtilizationThreshold represents a maximum Memory utilization for given workload that instructs horizontal pod autoscaler.
	LabelWorkloadAutoscaleMemoryUtilizationThreshold = LabelPrefix + "workload.autoscale-mem-threshold"

	// LabelWorkloadRollingUpdateMaxSurge max number of nodes updated at once
	LabelWorkloadRollingUpdateMaxSurge = LabelPrefix + "workload.rolling-update-max-surge"

	// LabelWorkloadMemory defines Memory request for workload
	LabelWorkloadMemory = LabelPrefix + "workload.memory"

	// LabelWorkloadCPU defines CPU request for workload
	LabelWorkloadCPU = LabelPrefix + "workload.cpu"

	// LabelWorkloadMaxMemory defines max Memory limit for workload
	LabelWorkloadMaxMemory = LabelPrefix + "workload.max-memory"

	// LabelWorkloadMaxCPU defines max CPU limit for workload
	LabelWorkloadMaxCPU = LabelPrefix + "workload.max-cpu"

	// LabelWorkloadSecurityContextRunAsUser sets pod security context RunAsUser attribute
	LabelWorkloadSecurityContextRunAsUser = LabelPrefix + "workload.pod-security-run-as-user"

	// LabelWorkloadSecurityContextRunAsGroup sets pod security context RunAsGroup attribute
	LabelWorkloadSecurityContextRunAsGroup = LabelPrefix + "workload.pod-security-run-as-group"

	// LabelWorkloadSecurityContextFsGroup sets pod security context FsGroup attribute
	LabelWorkloadSecurityContextFsGroup = LabelPrefix + "workload.pod-security-fs-group"

	// LabelWorkloadImagePullPolicy defines when to pull images from registry
	LabelWorkloadImagePullPolicy = LabelPrefix + "workload.image-pull-policy"

	// LabelWorkloadImagePullSecret defines docker registry image pull secret
	LabelWorkloadImagePullSecret = LabelPrefix + "workload.image-pull-secret"

	// LabelWorkloadRestartPolicy defines when to restart a pod
	LabelWorkloadRestartPolicy = LabelPrefix + "workload.restart-policy"

	// LabelWorkloadServiceAccountName defines service account name to be used by the workload
	LabelWorkloadServiceAccountName = LabelPrefix + "workload.service-account-name"

	// LabelWorkloadLivenessProbeType defines the command for workload liveness probe.
	LabelWorkloadLivenessProbeType = LabelPrefix + "workload.liveness-probe-type"

	// LabelWorkloadLivenessProbeCommand defines the command for workload liveness probe.
	LabelWorkloadLivenessProbeCommand = LabelPrefix + "workload.liveness-probe-command"

	// LabelWorkloadLivenessProbeHTTPPort defines the port for the http liveness probes.
	LabelWorkloadLivenessProbeHTTPPort = LabelPrefix + "workload.liveness-probe.http-port"

	// LabelWorkloadLivenessProbeHTTPPath defines the path for the http liveness probe.
	LabelWorkloadLivenessProbeHTTPPath = LabelPrefix + "workload.liveness-probe.http-path"

	// LabelWorkloadLivenessProbeTCPPort defines the path for the http liveness probe.
	LabelWorkloadLivenessProbeTCPPort = LabelPrefix + "workload.liveness-probe.tcp-port"

	// LabelWorkloadLivenessProbeInterval defines the interval for workload liveness probe
	LabelWorkloadLivenessProbeInterval = LabelPrefix + "workload.liveness-probe-interval"

	// LabelWorkloadLivenessProbeTimeout defines the timeout for workload liveness probe
	LabelWorkloadLivenessProbeTimeout = LabelPrefix + "workload.liveness-probe-timeout"

	// LabelWorkloadLivenessProbeInitialDelay defines the initial delay for workload liveness probe
	LabelWorkloadLivenessProbeInitialDelay = LabelPrefix + "workload.liveness-probe-initial-delay"

	// LabelWorkloadLivenessProbeRetries defines number of times workload liveness probe will retry
	LabelWorkloadLivenessProbeRetries = LabelPrefix + "workload.liveness-probe-retries"

	// LabelWorkloadReadinessProbeCommand defines the command for workload liveness probe
	LabelWorkloadReadinessProbeCommand = LabelPrefix + "workload.readiness-probe-command"

	// LabelWorkloadReadinessProbeInterval defines the interval for workload liveness probe
	LabelWorkloadReadinessProbeInterval = LabelPrefix + "workload.readiness-probe-interval"

	// LabelWorkloadReadinessProbeTimeout defines the timeout for workload liveness probe
	LabelWorkloadReadinessProbeTimeout = LabelPrefix + "workload.readiness-probe-timeout"

	// LabelWorkloadReadinessProbeInitialDelay defines the initial delay for workload liveness probe
	LabelWorkloadReadinessProbeInitialDelay = LabelPrefix + "workload.readiness-probe-initial-delay"

	// LabelWorkloadReadinessProbeRetries defines number of times workload liveness probe will retry
	LabelWorkloadReadinessProbeRetries = LabelPrefix + "workload.readiness-probe-retries"

	// LabelWorkloadReadinessProbeDisabled disables workload liveness probe
	LabelWorkloadReadinessProbeDisabled = LabelPrefix + "workload.readiness-probe-disabled"

	// LabelServiceType defines the type of service to be created
	LabelServiceType = LabelPrefix + "service.type"

	// LabelServiceNodePortPort defines port number for NodePort k8s service kind
	LabelServiceNodePortPort = LabelPrefix + "service.nodeport.port"

	// LabelServiceExpose informs whether K8s service should be exposed externally. To enable set as "true" or "domain.com,otherdomain.com".
	LabelServiceExpose = LabelPrefix + "service.expose"

	// LabelServiceExposeTLSSecret  provides the name of the TLS secret to use with the Kubernetes ingress controller
	LabelServiceExposeTLSSecret = LabelPrefix + "service.expose.tls-secret"

	// LabelVolumeSize defines persistent volume size
	LabelVolumeSize = LabelPrefix + "volume.size"

	// LabelVolumeSelector defines persistent volume selector
	LabelVolumeSelector = LabelPrefix + "volume.selector"

	// LabelVolumeStorageClass defines persistent volume storage class
	LabelVolumeStorageClass = LabelPrefix + "volume.storage-class"
)

var BaseServiceLabels = []string{
	LabelWorkloadReplicas,
	LabelWorkloadLivenessProbeType,
	LabelWorkloadLivenessProbeCommand,
}

var BaseVolumeLabels = []string{
	LabelVolumeSize,
}
