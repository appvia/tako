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
	"github.com/goccy/go-yaml"
)

// Inferred struct holds compose with parameter placeholders
// and derived application configuration
type Inferred struct {
	ComposeWithPlaceholders []byte
	BaseConfig              *Config
}

// Infer looks at resultant compose.yaml and extracts elements useful to
// deployment in Kubernetes, replaces values of those attributes with placeholders
// and places actual values in config.yaml for further tweaking.
func Infer(composeVersion string, composeConfig *compose.Project) (Inferred, error) {
	baseConfig := New()

	inferVolumesInfo(composeConfig, baseConfig)
	setSensibleDefaults(baseConfig)

	for _, s := range composeConfig.Services {
		c := &Component{}
		inferEnvironment(&s, c)
		inferService(&s, c)
		inferDeploymentInfo(&s, c)
		inferHealthcheckInfo(&s, c)
		baseConfig.Components[s.Name] = *c
	}

	withPlaceholders, err := injectPlaceholders(composeVersion, composeConfig)
	if err != nil {
		return Inferred{}, err
	}

	return Inferred{
		ComposeWithPlaceholders: withPlaceholders,
		BaseConfig:              baseConfig,
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
func inferVolumesInfo(composeConfig *compose.Project, appConfig *Config) {
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
func inferEnvironment(s *compose.ServiceConfig, cmp *Component) {
	serviceEnvs := make(map[string]string)
	for k, v := range s.Environment {
		if v == nil {
			temp := "" // *string cannot be initialized
			v = &temp  // in one statement
		}

		serviceEnvs[k] = *v
	}
	// set service environment
	cmp.Environment = serviceEnvs
}

// Extracts information about K8s service requirements
func inferService(s *compose.ServiceConfig, cmp *Component) {
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
func inferDeploymentInfo(s *compose.ServiceConfig, cmp *Component) {
	// get workload object
	w := cmp.Workload

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
		w.Replicas = *s.Deploy.Replicas

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
			w.Memory = GetMemoryQuantity(int64(s.Deploy.Resources.Reservations.MemoryBytes))
		}
		// Resources Limits
		if s.Deploy.Resources.Limits != nil {
			w.MaxCPU = s.Deploy.Resources.Limits.NanoCPUs
			w.MaxMemory = GetMemoryQuantity(int64(s.Deploy.Resources.Limits.MemoryBytes))
		}
		// Rolling update policy
		if s.Deploy.UpdateConfig != nil {
			w.RollingUpdateMaxSurge = *s.Deploy.UpdateConfig.Parallelism
		}
	}

	cmp.Workload = w
}

// Extracts service healthcheck information
func inferHealthcheckInfo(s *compose.ServiceConfig, cmp *Component) {
	w := cmp.Workload
	w.LivenessProbeDisable = &s.HealthCheck.Disable
	w.LivenessProbeCommand = s.HealthCheck.Test
	w.LivenessProbeInterval = s.HealthCheck.Interval.String()
	w.LivenessProbeInitialDelay = s.HealthCheck.StartPeriod.String()
	w.LivenessProbeTimeout = s.HealthCheck.Timeout.String()
	w.LivenessProbeRetries = *s.HealthCheck.Retries

	cmp.Workload = w
}

// injectPlaceholders substitutes key attribute values with placeholders
func injectPlaceholders(composeVersion string, composeConfig *compose.Project) ([]byte, error) {
	data, err := yaml.Marshal(composeConfig)
	if err != nil {
		return nil, err
	}

	// Unmarshal compose config to map[string]interface{}
	opaqueConfig, err := utils.UnmarshallGeneral(data)
	if err != nil {
		return nil, err
	}

	// Service specific placeholders
	for name, svc := range opaqueConfig["services"].(map[string]interface{}) {
		// placeholder prefix
		prefix := fmt.Sprintf("%s.workload", name)

		//== Deploy ==
		deploy := svc.(map[string]interface{})["deploy"]
		//- Replicas
		deploy.(map[string]interface{})["replicas"] = placeholder(prefix, "replicas")
		//- Rolling update max surge
		updateConfig := deploy.(map[string]interface{})["update_config"]
		updateConfig.(map[string]interface{})["parallelism"] = placeholder(prefix, "rolling-update-max-surge")
		//- Resource Requests & Limits
		resources := deploy.(map[string]interface{})["resources"]
		reservations := resources.(map[string]interface{})["reservations"]
		limits := resources.(map[string]interface{})["limits"]
		reservations.(map[string]interface{})["cpus"] = placeholder(prefix, "cpu")
		reservations.(map[string]interface{})["memory"] = placeholder(prefix, "memory")
		limits.(map[string]interface{})["cpus"] = placeholder(prefix, "max-cpu")
		limits.(map[string]interface{})["memory"] = placeholder(prefix, "max-memory")

		//== Healthcheck ==
		hc := svc.(map[string]interface{})["healthcheck"]
		hc.(map[string]interface{})["disable"] = placeholder(prefix, "liveness-probe-disable")
		hc.(map[string]interface{})["interval"] = placeholder(prefix, "liveness-probe-interval")
		hc.(map[string]interface{})["retries"] = placeholder(prefix, "liveness-probe-retries")
		hc.(map[string]interface{})["start_period"] = placeholder(prefix, "liveness-probe-initial-delay")
		hc.(map[string]interface{})["test"] = placeholder(prefix, "liveness-probe-command")
		hc.(map[string]interface{})["timeout"] = placeholder(prefix, "liveness-probe-timeout")

		//== Environment ==
		prefix = fmt.Sprintf("%s.environment", name)
		environment := svc.(map[string]interface{})["environment"]
		if environment != nil {
			for e := range environment.(map[string]interface{}) {
				environment.(map[string]interface{})[e] = placeholder(prefix, e)
			}
		}
	}

	out := ShallowComposeConfig{
		Version:  composeVersion,
		Services: opaqueConfig["services"],
		Networks: opaqueConfig["networks"],
		Volumes:  opaqueConfig["volumes"],
		Secrets:  opaqueConfig["secrets"],
		Configs:  opaqueConfig["configs"],
	}
	return utils.MarshallAndFormat(out, 2)
}

// Builds config attribute value placeholder
func placeholder(prefix, key string) string {
	return fmt.Sprintf("${%s.%s}", prefix, key)
}
