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
	"strconv"
	"strings"

	composego "github.com/compose-spec/compose-go/types"
	"k8s.io/apimachinery/pkg/api/resource"
)

// setDefaultLabels sets sensible workload defaults as labels.
func setDefaultLabels(target *composego.ServiceConfig) {
	target.Labels.Add("kev.workload.image-pull-policy", DefaultImagePullPolicy)
	target.Labels.Add("kev.workload.service-account-name", DefaultServiceAccountName)
}

// extractVolumesLabels extracts volume labels into a label's Volumes attribute.
func extractVolumesLabels(c *composeProject, out *labels) {
	// Volumes map
	vols := make(map[string]composego.VolumeConfig)

	for _, v := range c.VolumeNames() {
		vols[v] = composego.VolumeConfig{
			Labels: map[string]string{
				"kev.volumes.class": DefaultVolumeClass,
				"kev.volumes.size":  DefaultVolumeSize,
			},
		}
	}
	out.Volumes = vols
}

// extractServiceTypeLabels extracts service type labels into a label's Service.
func extractServiceTypeLabels(source composego.ServiceConfig, target *composego.ServiceConfig) {
	if source.Ports == nil {
		target.Labels.Add("kev.service.type", NoService)
	} else {
		for _, p := range source.Ports {
			if p.Published != 0 && p.Mode == "host" {
				target.Labels.Add("kev.service.type", NodePortService)
			} else if p.Published != 0 && p.Mode == "ingress" {
				target.Labels.Add("kev.service.type", LoadBalancerService)
			} else if p.Published != 0 || (p.Published == 0 && p.Target != 0) {
				target.Labels.Add("kev.service.type", ClusterIPService)
			} else if p.Published == 0 {
				target.Labels.Add("kev.service.type", HeadlessService)
			}
			// @todo: Processing just the first port for now!
			break
		}
	}
}

// extractDeploymentLabels extracts deployment related into a label's Service.
func extractDeploymentLabels(source composego.ServiceConfig, target *composego.ServiceConfig) {
	extractWorkloadType(source, target)
	extractWorkloadReplicas(source, target)
	extractWorkloadRestartPolicy(source, target)
	extractWorkloadResourceRequests(source, target)
	extractWorkloadResourceLimits(source, target)
	extractWorkloadRollingUpdatePolicy(source, target)
}

// extractWorkloadRollingUpdatePolicy extracts deployment's rolling update policy.
func extractWorkloadRollingUpdatePolicy(source composego.ServiceConfig, target *composego.ServiceConfig) {
	if source.Deploy != nil && source.Deploy.UpdateConfig != nil {
		value := strconv.FormatUint(*source.Deploy.UpdateConfig.Parallelism, 10)
		target.Labels.Add("kev.workload.rolling-update-max-surge", value)
	}
}

// extractWorkloadResourceLimits extracts deployment's resource limits.
func extractWorkloadResourceLimits(source composego.ServiceConfig, target *composego.ServiceConfig) {
	if source.Deploy != nil && source.Deploy.Resources.Limits != nil {
		target.Labels.Add("kev.workload.max-cpu", source.Deploy.Resources.Limits.NanoCPUs)

		value := getMemoryQuantity(int64(source.Deploy.Resources.Limits.MemoryBytes))
		target.Labels.Add("kev.workload.max-memory", value)
	}
}

// extractWorkloadResourceRequests extracts deployment's resource requests.
func extractWorkloadResourceRequests(source composego.ServiceConfig, target *composego.ServiceConfig) {
	if source.Deploy != nil && source.Deploy.Resources.Reservations != nil {
		target.Labels.Add("kev.workload.cpu", source.Deploy.Resources.Reservations.NanoCPUs)

		value := getMemoryQuantity(int64(source.Deploy.Resources.Reservations.MemoryBytes))
		target.Labels.Add("kev.workload.memory", value)
	}
}

// extractWorkloadRestartPolicy extracts deployment's restart policy.
func extractWorkloadRestartPolicy(source composego.ServiceConfig, target *composego.ServiceConfig) {
	if source.Deploy != nil && source.Deploy.RestartPolicy != nil {
		if source.Deploy.RestartPolicy.Condition == "on-failure" {
			target.Labels.Add("kev.workload.restart", RestartPolicyOnFailure)
		} else if source.Deploy.RestartPolicy.Condition == "none" {
			target.Labels.Add("kev.workload.restart", RestartPolicyNever)
		} else {
			// Always restart by default
			target.Labels.Add("kev.workload.restart", RestartPolicyAlways)
		}
	}
}

// extractWorkloadReplicas extracts deployment's restart policy.
func extractWorkloadReplicas(source composego.ServiceConfig, target *composego.ServiceConfig) {
	if source.Deploy != nil {
		value := strconv.FormatUint(*source.Deploy.Replicas, 10)
		target.Labels.Add("kev.workload.replicas", value)
	}
}

// extractWorkloadType extracts deployment's workload type.
func extractWorkloadType(source composego.ServiceConfig, target *composego.ServiceConfig) {
	if source.Deploy != nil && source.Deploy.Mode == "global" {
		target.Labels.Add("kev.workload.type", DaemonsetWorkload)
	} else {
		// replicated
		if source.Volumes != nil {
			// Volumes in use so likely a Statefulset
			target.Labels.Add("kev.workload.type", StatefulsetWorkload)
		} else {
			// default to deployment
			target.Labels.Add("kev.workload.type", DeploymentWorkload)
		}
	}
}

// extractHealthcheckLabels extracts health check labels into a label's Service.
func extractHealthcheckLabels(source composego.ServiceConfig, target *composego.ServiceConfig) {
	target.Labels.Add("kev.workload.liveness-probe-disable", strconv.FormatBool(source.HealthCheck.Disable))
	target.Labels.Add("kev.workload.liveness-probe-interval", source.HealthCheck.Interval.String())

	retries := strconv.FormatUint(*source.HealthCheck.Retries, 10)
	target.Labels.Add("kev.workload.liveness-probe-retries", retries)

	target.Labels.Add("kev.workload.liveness-probe-initial-delay", source.HealthCheck.StartPeriod.String())
	target.Labels.Add("kev.workload.liveness-probe-command", formatSlice(source.HealthCheck.Test))
	target.Labels.Add("kev.workload.liveness-probe-timeout", source.HealthCheck.Timeout.String())
}

// formatSlice formats a string slice as '["value1", "value2", "value3"]'
func formatSlice(test []string) string {
	quoted := fmt.Sprintf("[%q]", strings.Join(test, `", "`))
	return strings.ReplaceAll(quoted, "\\", "")
}

// GetMemoryQuantity returns memory amount as string in Kubernetes notation
// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory
// Example: 100Mi, 20Gi
func getMemoryQuantity(b int64) string {
	const unit int64 = 1024

	q := resource.NewQuantity(b, resource.BinarySI)

	quantity, _ := q.AsInt64()
	if quantity%unit == 0 {
		return q.String()
	}

	// Kubernetes resource quantity computation doesn't do well with values containing decimal points
	// Example: 10.6Mi would translate to 11114905 (bytes)
	// Let's keep consistent with kubernetes resource amount notation (below).

	if b < unit {
		return fmt.Sprintf("%d", b)
	}

	div, exp := unit, 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%ci", float64(b)/float64(div), "KMGTPE"[exp])
}
