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

import (
	"fmt"

	"github.com/appvia/kube-devx/pkg/kev/utils"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/dustin/go-humanize"
	"github.com/goccy/go-yaml"
)

// Inferred struct holds compose with parameter placeholders
// and derived application configuration
type Inferred struct {
	ComposeWithPlaceholders []byte
	AppConfig               *Config
}

// Infer looks at resultant compose.yaml and extracts elements useful to
// deployment in Kubernetes, replaces values of those attributes with placeholders
// and places actual values in config.yaml for further tweaking.
func Infer(data []byte) (Inferred, error) {
	// Application Configuration
	appConfig := New()

	composeConfig, err := utils.UnmarshallComposeConfig(data)
	if err != nil {
		return Inferred{}, err
	}

	// Extract volumes information
	extractVolumesInfo(composeConfig, appConfig)
	// Set common app level settings
	setSensibleDefaults(appConfig)

	// Service level parameters
	for _, s := range composeConfig.Services {
		// Initiate config component (i.e. composeConfig service)
		c := &Component{}
		// Environment information
		extractEnvironment(&s, c)
		// Derive service type
		extractService(&s, c)
		// Deployment details
		extractDeploymentInfo(&s, c)
		// Add component to the app Config
		appConfig.Components[s.Name] = *c
	}

	composeBytes, err := yaml.Marshal(composeConfig)
	if err != nil {
		return Inferred{}, err
	}

	return Inferred{
		ComposeWithPlaceholders: composeBytes,
		AppConfig:               appConfig,
	}, nil
}

// setSensibleDefaults set common app level parameters with sensible defaults
func setSensibleDefaults(appConfig *Config) {
	appConfig.Workload = Workload{
		ImagePullPolicy:    DefaultImagePullPolicy,
		ServiceAccountName: DefaultServiceAccountName,
	}
}

// Extracts volumes information
func extractVolumesInfo(composeConfig *compose.Config, appConfig *Config) {
	// Volumes map
	vols := make(map[string]Volume)

	for _, v := range composeConfig.VolumeNames() {
		vols[v] = Volume{
			Size:  DefaultVolumeSize,
			Class: DefaultVolumeClass,
		}
	}

	// set Volumes information in app config
	appConfig.Volumes = vols
}

// Extracts environment variables for each compose service and parametrises them
func extractEnvironment(s *compose.ServiceConfig, cmp *Component) {
	// Environmet
	placeholders := make(compose.MappingWithEquals)
	serviceEnvs := make(map[string]string)
	for k, v := range s.Environment {
		if v == nil {
			temp := "" // *string cannot be initialized
			v = &temp  // in one statement
		}

		// prepare env variable placeholder
		p := fmt.Sprint("$${", s.Name, ".environment.", k, "}")
		placeholders[k] = &p

		serviceEnvs[k] = *v
	}
	// set service environment
	cmp.Environment = serviceEnvs

	// override values of key attributes with parametrised placeholders
	s.Environment.OverrideBy(placeholders)
}

// Extracts information about K8s service requirements
func extractService(s *compose.ServiceConfig, cmp *Component) {
	if s.Ports == nil {
		cmp.Service.Type = NoService
	} else {
		for _, p := range s.Ports {
			if p.Published != 0 && p.Mode == "host" {
				// Service published as NodePort
				cmp.Service.Type = NodePortService
				cmp.Service.Nodeport = p.Target
			} else if p.Published != 0 && p.Mode == "ingress" {
				// Service published as LoadBalancer
				// @todo: we might need to supress that and create ClusterIP kind by default!
				cmp.Service.Type = LoadBalancerService
			} else if p.Published != 0 {
				// Service published as ClusterIP
				cmp.Service.Type = ClusterIPService
			} else if p.Published == 0 {
				// Service unpublished i.e. Headless
				cmp.Service.Type = HeadlessService
			}
			// @todo: Process just the first one for now!
			break
		}
	}
}

// Extracts deployment information
func extractDeploymentInfo(s *compose.ServiceConfig, cmp *Component) {
	// Initiate workload object
	w := Workload{}

	// Workload type
	if s.Deploy != nil && s.Deploy.Mode == "global" {
		// service is a DaemonSet
		w.Type = DaemonsetWorkload
	} else {
		// replicated
		if s.Volumes != nil {
			// Volumes in use so likely a Statefulset
			w.Type = StatefulsetWorkload
		} else {
			// default to deployment
			w.Type = DeploymentWorkload
		}
	}

	if s.Deploy != nil {
		// Replicas
		w.Replicas = s.Deploy.Replicas
		// RestartPolicy
		if s.Deploy.RestartPolicy != nil {
			if s.Deploy.RestartPolicy.Condition == "on-failure" {
				w.Restart = RestartPolicyOnFailure
			} else if s.Deploy.RestartPolicy.Condition == "none" {
				w.Restart = RestartPolicyNever
			} else {
				// Always restart by default
				w.Restart = RestartPolicyAlways
			}
		}
		// Resources Requests
		if s.Deploy.Resources.Reservations != nil {
			w.CPU = s.Deploy.Resources.Reservations.NanoCPUs
			w.Memory = humanize.Bytes(uint64(s.Deploy.Resources.Reservations.MemoryBytes))
		}
		// Resources Limits
		if s.Deploy.Resources.Limits != nil {
			w.MaxCPU = s.Deploy.Resources.Limits.NanoCPUs
			w.MaxMemory = humanize.Bytes(uint64(s.Deploy.Resources.Limits.MemoryBytes))
		}
		// Rolling update policy
		if s.Deploy.UpdateConfig != nil {
			w.RollingUpdateMaxSurge = s.Deploy.UpdateConfig.Parallelism
		}
	}

	cmp.Workload = w
}
