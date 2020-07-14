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
	// DefaultVolumeClass default PV size
	DefaultVolumeClass = "standard"
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
)
