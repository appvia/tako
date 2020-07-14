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
	// defaultVolumeClass default PV size
	defaultVolumeClass = "standard"
	// defaultService is a default service
	defaultService = noService
	// defaultRestartPolicy is a default restart policy
	defaultRestartPolicy = RestartPolicyAlways
	// defaultWorkload is a defauld workload type
	defaultWorkload = DeploymentWorkload
	// defaultServiceAccountName is a default SA to be used
	defaultServiceAccountName = "default"
	// defaultImagePullPolicy default image pull policy
	defaultImagePullPolicy = "IfNotPresent"
	// defaultImagePullSecret default image pull credentials secret name
	defaultImagePullSecret = ""
	// defaultSecurityContextRunAsUser default UID for pod security context
	defaultSecurityContextRunAsUser = ""
	// defaultSecurityContextRunAsGroup default GID for pod security context
	defaultSecurityContextRunAsGroup = ""
	// defaultSecurityContextFsGroup default fs Group for pod security context
	defaultSecurityContextFsGroup = ""

	// noService default value
	noService = "None"
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
