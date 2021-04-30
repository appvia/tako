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
	"strconv"

	"github.com/appvia/kev/pkg/kev/config"

	composego "github.com/compose-spec/compose-go/types"
)

// setDefaultLabels sets sensible workload defaults as labels.
func setDefaultLabels(target *ServiceConfig) {
	target.Labels.Add(config.LabelWorkloadServiceAccountName, config.DefaultServiceAccountName)
}

// TODO: Remove this whole thing when ready
// extractVolumesLabels extracts volume labels into a label's Volumes attribute.
func extractVolumesLabels(c *ComposeProject, out *composeOverride) {
	vols := make(map[string]VolumeConfig)

	for _, v := range c.VolumeNames() {
		labels := map[string]string{}

		if storageClass, ok := c.Volumes[v].Labels[config.LabelVolumeStorageClass]; ok {
			labels[config.LabelVolumeStorageClass] = storageClass
		} else {
			labels[config.LabelVolumeStorageClass] = config.DefaultVolumeStorageClass
		}

		if volSize, ok := c.Volumes[v].Labels[config.LabelVolumeSize]; ok {
			labels[config.LabelVolumeSize] = volSize
		} else {
			labels[config.LabelVolumeSize] = config.DefaultVolumeSize
		}

		vols[v] = VolumeConfig{Labels: labels}
	}
	out.Volumes = vols
}

//TODO: Remove once all functions have been moved over.
// extractDeploymentLabels extracts deployment related into a label's Service.
func extractDeploymentLabels(source composego.ServiceConfig, target *ServiceConfig) {
	extractWorkloadResourceRequests(source, target)
	extractWorkloadResourceLimits(source, target)
	extractWorkloadRollingUpdatePolicy(source, target)
}

// extractWorkloadRollingUpdatePolicy extracts deployment's rolling update policy.
func extractWorkloadRollingUpdatePolicy(source composego.ServiceConfig, target *ServiceConfig) {
	if source.Deploy != nil && source.Deploy.UpdateConfig != nil {
		value := strconv.FormatUint(*source.Deploy.UpdateConfig.Parallelism, 10)
		target.Labels.Add(config.LabelWorkloadRollingUpdateMaxSurge, value)
	}
}

// extractWorkloadResourceLimits extracts deployment's resource limits.
func extractWorkloadResourceLimits(source composego.ServiceConfig, target *ServiceConfig) {
	if source.Deploy != nil && source.Deploy.Resources.Limits != nil {
		target.Labels.Add(config.LabelWorkloadMaxCPU, source.Deploy.Resources.Limits.NanoCPUs)
	}
}

// extractWorkloadResourceRequests extracts deployment's resource requests.
func extractWorkloadResourceRequests(source composego.ServiceConfig, target *ServiceConfig) {
	if source.Deploy != nil && source.Deploy.Resources.Reservations != nil {
		target.Labels.Add(config.LabelWorkloadCPU, source.Deploy.Resources.Reservations.NanoCPUs)
	}
}
