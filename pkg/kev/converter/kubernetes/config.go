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

package kubernetes

import (
	"strings"

	"github.com/appvia/kube-devx/pkg/kev/config"
)

// @todo Refactor when we combine all bits of configuration in one place

// CombinedConfig represents combined kev + kompose + conversion configuration options
type CombinedConfig struct {
	kevConfig      *config.Config
	komposeObject  KomposeObject
	convertOptions ConvertOptions
}

// getKevComponent get Kev component configuration
func (c *CombinedConfig) getKevComponent(name string) config.Component {
	return c.kevConfig.Components[name]
}

// getKomposeComponent get Kompose component configuration
func (c *CombinedConfig) getKomposeComponent(name string) ServiceConfig {
	return c.komposeObject.ServiceConfigs[name]
}

// imagePullPolicy gets image pull policy for given service name
func (c *CombinedConfig) imagePullPolicy(name string) string {
	if c.getKevComponent(name).Workload.ImagePullPolicy != "" {
		return c.getKevComponent(name).Workload.ImagePullPolicy
	} else if c.kevConfig.Workload.ImagePullPolicy != "" {
		return c.kevConfig.Workload.ImagePullPolicy
	} else {
		policy, err := GetImagePullPolicy(name, c.getKomposeComponent(name).ImagePullPolicy)
		if err != nil {
			// Value derived by kompose is invalid. Default to "IfNotPresent".
			return config.DefaultImagePullPolicy
		}
		return string(policy)
	}
}

// imagePullSecret returns image pull secret for a service name
func (c *CombinedConfig) imagePullSecret(name string) string {
	if c.getKevComponent(name).Workload.ImagePullSecret != "" {
		return c.getKevComponent(name).Workload.ImagePullSecret
	} else if c.kevConfig.Workload.ImagePullSecret != "" {
		return c.kevConfig.Workload.ImagePullSecret
	} else if c.getKomposeComponent(name).ImagePullSecret != "" {
		return c.getKomposeComponent(name).ImagePullSecret
	}
	return config.DefaultImagePullSecret
}

// restartPolicy gets restart policy for given service name
func (c *CombinedConfig) restartPolicy(name string) string {
	if c.getKevComponent(name).Workload.Restart != "" {
		return c.getKevComponent(name).Workload.Restart
	} else if c.kevConfig.Workload.Restart != "" {
		return c.kevConfig.Workload.Restart
	} else {
		restart, err := GetRestartPolicy(name, c.getKomposeComponent(name).Restart)
		if err != nil {
			// Value derived by kompose is invalid. Default to "Always".
			return config.RestartPolicyAlways
		}
		return string(restart)
	}
}

// replicas returns number of replicas for service name
func (c *CombinedConfig) replicas(name string) int {
	if c.getKevComponent(name).Workload.Replicas != 0 {
		return int(c.getKevComponent(name).Workload.Replicas)
	} else if c.kevConfig.Workload.Replicas != 0 {
		return int(c.kevConfig.Workload.Replicas)
	} else if c.convertOptions.Replicas != 0 {
		return c.convertOptions.Replicas
	}
	return config.DefaultReplicaNumber
}

// workloadType returns type of workload for a given service name
func (c *CombinedConfig) workloadType(name string) string {
	if c.getKevComponent(name).Workload.Type != "" {
		return c.getKevComponent(name).Workload.Type
	} else if c.kevConfig.Workload.Type != "" {
		return c.kevConfig.Workload.Type
	} else if c.convertOptions.Controller != "" {
		return c.convertOptions.Controller
	}
	return config.DefaultWorkload
}

// isDaemonSet tells whether a component workload type is DaemonSet
func (c *CombinedConfig) isDaemonSet(name string) bool {
	return strings.ToLower(c.workloadType(name)) == strings.ToLower(config.DaemonsetWorkload)
}

// serviceType returns type of K8s service for a given component name
func (c *CombinedConfig) serviceType(name string) string {
	if c.getKevComponent(name).Service.Type != "" {
		return c.getKevComponent(name).Service.Type
	} else if c.kevConfig.Service.Type != "" {
		return c.kevConfig.Service.Type
	} else if c.getKomposeComponent(name).ServiceType != "" {
		return c.getKomposeComponent(name).ServiceType
	}
	return config.DefaultService
}

// serviceAccount returns service account for given component
func (c *CombinedConfig) serviceAccount(name string) string {
	if c.getKevComponent(name).Workload.ServiceAccountName != "" {
		return c.getKevComponent(name).Workload.ServiceAccountName
	} else if c.kevConfig.Workload.ServiceAccountName != "" {
		return c.kevConfig.Workload.ServiceAccountName
	}
	return config.DefaultServiceAccountName
}
