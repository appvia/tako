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
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/log"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/spf13/cast"
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func NewProjectService(svc composego.ServiceConfig) (ProjectService, error) {
	cfg, err := config.SvcK8sConfigFromCompose(&svc)
	if err != nil {
		return ProjectService{}, err
	}

	return ProjectService{
		ServiceConfig: svc,
		SvcK8sConfig:  cfg,
	}, nil
}

// enabled returns Bool telling Kev whether app component is enabled/disabled
func (p *ProjectService) enabled() bool {
	return !p.SvcK8sConfig.Disabled
}

// replicas returns number of replicas for given project service
func (p *ProjectService) replicas() int32 {
	return int32(p.SvcK8sConfig.Workload.Replicas)
}

// autoscaleMaxReplicas returns maximum number of replicas for autoscaler
func (p *ProjectService) autoscaleMaxReplicas() int32 {
	return int32(p.SvcK8sConfig.Workload.Autoscale.MaxReplicas)
}

// autoscaleTargetCPUUtilization returns target CPU utilization percentage for autoscaler
func (p *ProjectService) autoscaleTargetCPUUtilization() int32 {
	return int32(p.SvcK8sConfig.Workload.Autoscale.CPUThreshold)
}

// autoscaleTargetMemoryUtilization returns target memory utilization percentage for autoscaler
func (p *ProjectService) autoscaleTargetMemoryUtilization() int32 {
	return int32(p.SvcK8sConfig.Workload.Autoscale.MemoryThreshold)
}

// workloadType returns workload type for the project service
func (p *ProjectService) workloadType() string {
	workloadType := p.SvcK8sConfig.Workload.Type

	if p.Deploy != nil && p.Deploy.Mode == "global" && !strings.EqualFold(workloadType, config.DaemonsetWorkload) {
		log.WarnfWithFields(log.Fields{
			"project-service": p.Name,
			"workload-type":   workloadType,
		}, "Compose service defined as 'global' should map to K8s DaemonSet. Current configuration forces conversion to %s",
			workloadType)
	}

	return workloadType
}

// serviceType returns service type for project service workload
func (p *ProjectService) serviceType() (string, error) {
	serviceType := p.SvcK8sConfig.Service.Type

	// @step validate whether service type is set properly when node port is specified
	if !strings.EqualFold(serviceType, string(v1.ServiceTypeNodePort)) && p.nodePort() != 0 {
		log.ErrorfWithFields(log.Fields{
			"project-service": p.Name,
			"service-type":    serviceType,
			"nodeport":        p.nodePort(),
		}, "%s label value must be set as `NodePort` when assiging node port value", serviceType)

		return "", fmt.Errorf("`%s` workload service type must be set as `NodePort` when assiging node port value", p.Name)
	}

	if len(p.ports()) > 1 && p.nodePort() != 0 {
		log.ErrorfWithFields(log.Fields{
			"project-service": p.Name,
		}, "Cannot set %s label value when service has multiple ports specified.", config.LabelServiceNodePortPort)

		return "", fmt.Errorf("`%s` cannot set NodePort service port when project service has multiple ports defined", p.Name)
	}

	return serviceType, nil
}

// nodePort returns the port for NodePort service type
func (p *ProjectService) nodePort() int32 {
	if val, ok := p.Labels[config.LabelServiceNodePortPort]; ok {
		nodePort, _ := strconv.Atoi(val)
		return int32(nodePort)
	}

	return 0
}

// exposeService tells whether service for project component should be exposed
func (p *ProjectService) exposeService() (string, error) {
	if val, ok := p.Labels[config.LabelServiceExpose]; ok {
		if val == "" && p.tlsSecretName() != "" {
			log.ErrorfWithFields(log.Fields{
				"project-service": p.Name,
				"tls-secret-name": p.tlsSecretName(),
			}, "TLS secret name specified via %s label but project service not exposed!",
				config.LabelServiceExposeTLSSecret)

			return "", fmt.Errorf("Service can't have TLS secret name when it hasn't been exposed")
		}
		return val, nil
	}

	return "", nil
}

// tlsSecretName returns TLS secret name for exposed service (to be used in the ingress configuration)
func (p *ProjectService) tlsSecretName() string {
	if val, ok := p.Labels[config.LabelServiceExposeTLSSecret]; ok {
		return val
	}

	return ""
}

// getKubernetesUpdateStrategy gets update strategy for compose project service
// Note: it only supports `parallelism` and `order`
// @todo add label support for update strategy!
func (p *ProjectService) getKubernetesUpdateStrategy() *v1apps.RollingUpdateDeployment {
	if p.Deploy == nil || p.Deploy.UpdateConfig == nil {
		return nil
	}

	cfg := p.Deploy.UpdateConfig
	r := v1apps.RollingUpdateDeployment{}

	if cfg.Order == "stop-first" {
		if cfg.Parallelism != nil {
			maxUnavailable := intstr.FromInt(cast.ToInt(*cfg.Parallelism))
			r.MaxUnavailable = &maxUnavailable
		}

		maxSurge := intstr.FromInt(0)
		r.MaxSurge = &maxSurge
		return &r
	}

	if cfg.Order == "start-first" {
		if cfg.Parallelism != nil {
			maxSurge := intstr.FromInt(cast.ToInt(*cfg.Parallelism))
			r.MaxSurge = &maxSurge
		}
		maxUnavailable := intstr.FromInt(0)
		r.MaxUnavailable = &maxUnavailable
		return &r
	}

	return nil
}

// volumes gets volumes for compose project service, respecting volume lables if specified.
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L535
func (p *ProjectService) volumes(project *composego.Project) ([]Volumes, error) {
	vols, err := retrieveVolume(p.Name, project)
	if err != nil {
		log.Error("Could not retrieve volume")
		return nil, err
	}

	for i, vol := range vols {
		composeVol := volumeByNameAndFormat(vol.VolumeName, rfc1123, project.Volumes)
		k8sVol, err := config.VolK8sConfigFromCompose(&composeVol)
		if err != nil {
			return nil, err
		}

		// We can't assign value to struct field in map while iterating over it, so temporary variable `temp` is used here
		var temp = vols[i]
		temp.PVCSize = k8sVol.Size
		temp.SelectorValue = k8sVol.Selector
		temp.StorageClass = k8sVol.StorageClass
		vols[i] = temp
	}

	return vols, nil
}

// placement returns information regarding pod affinity
// @todo Add placement support via labels!
func (p *ProjectService) placement() map[string]string {
	if p.Deploy != nil && p.Deploy.Placement.Constraints != nil {
		return loadPlacement(p.Deploy.Placement.Constraints)
	}

	return nil
}

// resourceRequests returns workload resource requests (memory & cpu)
// It parses CPU & Memory as k8s resource.Quantity regardless
// of how values are supplied (via deploy block or labels).
// It supports resource notations:
// - CPU: 0.1, 100m (which is the same as 0.1), 1
// - Memory: 1, 1M, 1m, 1G, 1Gi
func (p *ProjectService) resourceRequests() (*int64, *int64) {
	var memRequest int64
	var cpuRequest int64

	// @step extract requests from deploy block if present
	if p.Deploy != nil && p.Deploy.Resources.Reservations != nil {
		memRequest = int64(p.Deploy.Resources.Reservations.MemoryBytes)
		cpu, _ := resource.ParseQuantity(p.Deploy.Resources.Reservations.NanoCPUs)
		cpuRequest = cpu.ToDec().MilliValue()
	}

	if val := p.SvcK8sConfig.Workload.Resource.Memory; val != "" {
		v, _ := resource.ParseQuantity(val)
		memRequest, _ = v.AsInt64()
	}

	if val := p.SvcK8sConfig.Workload.Resource.CPU; val != "" {
		v, _ := resource.ParseQuantity(val)
		cpuRequest = v.ToDec().MilliValue()
	}

	return &memRequest, &cpuRequest
}

// resourceLimits returns workload resource limits (memory & cpu)
// It parses CPU & Memory as k8s resource.Quantity regardless
// of how values are supplied (via deploy block or labels).
// It supports resource notations:
// - CPU: 0.1, 100m (which is the same as 0.1), 1
// - Memory: 1, 1M, 1m, 1G, 1Gi
func (p *ProjectService) resourceLimits() (*int64, *int64) {
	var memLimit int64
	var cpuLimit int64

	// @step extract limits from deploy block if present
	if p.Deploy != nil && p.Deploy.Resources.Limits != nil {
		cpu, _ := resource.ParseQuantity(p.Deploy.Resources.Limits.NanoCPUs)
		cpuLimit = cpu.ToDec().MilliValue()
	}

	if val := p.SvcK8sConfig.Workload.Resource.MaxMemory; val != "" {
		v, _ := resource.ParseQuantity(val)
		memLimit, _ = v.AsInt64()
	}

	if val := p.SvcK8sConfig.Workload.Resource.MaxCPU; val != "" {
		v, _ := resource.ParseQuantity(val)
		cpuLimit = v.ToDec().MilliValue()
	}

	return &memLimit, &cpuLimit
}

// runAsUser returns pod security context runAsUser value
func (p *ProjectService) runAsUser() int64 {
	return int64(p.SvcK8sConfig.Workload.PodSecurity.RunAsUser)
}

// runAsGroup returns pod security context runAsGroup value
func (p *ProjectService) runAsGroup() int64 {
	return int64(p.SvcK8sConfig.Workload.PodSecurity.RunAsGroup)
}

// fsGroup returns pod security context fsGroup value
func (p *ProjectService) fsGroup() int64 {
	return int64(p.SvcK8sConfig.Workload.PodSecurity.FsGroup)
}

// imagePullPolicy returns image PullPolicy for project service
func (p *ProjectService) imagePullPolicy() v1.PullPolicy {
	return v1.PullPolicy(p.SvcK8sConfig.Workload.ImagePull.Policy)
}

// imagePullSecret returns image pull secret (for private registries)
func (p *ProjectService) imagePullSecret() string {
	return p.SvcK8sConfig.Workload.ImagePull.Secret
}

// serviceAccountName returns service account name to be used by the pod
func (p *ProjectService) serviceAccountName() string {
	if val, ok := p.Labels[config.LabelWorkloadServiceAccountName]; ok {
		return val
	}

	return config.DefaultServiceAccountName
}

// restartPolicy return workload restart policy. Supports both docker-compose and Kubernetes notations.
func (p *ProjectService) restartPolicy() v1.RestartPolicy {
	policy := config.RestartPolicyAlways
	if p.SvcK8sConfig.Workload.RestartPolicy != "" {
		policy = p.SvcK8sConfig.Workload.RestartPolicy
	}

	restartPolicy, err := getRestartPolicy(p.Name, policy)
	if err != nil {
		log.WarnWithFields(log.Fields{
			"project-service": p.Name,
			"restart-policy":  policy,
		}, "Restart policy is not supported, defaulting to 'Always'")

		return v1.RestartPolicy(config.RestartPolicyAlways)
	}

	return restartPolicy
}

// environment returns composego project service environment variables, and evaluates ENV from OS
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L465
func (p *ProjectService) environment() composego.MappingWithEquals {
	// Note: empty value ENV variables will be also interpolated with ENV value defined in the OS environment
	envs := composego.MappingWithEquals{}

	for name, value := range p.Environment {
		if value != nil {
			envs[name] = value
		} else {
			result, _ := os.LookupEnv(name)
			if result != "" {
				envs[name] = &result
			} else {
				log.WarnWithFields(log.Fields{
					"project-service": p.Name,
					"env-var":         name,
				}, "Env Var has no value and will be ignored")

				continue
			}
		}
	}

	return envs
}

// ports returns combined list of ports from both project service `Ports` and `Expose`. Docker Expose ports are treated as TCP ports.
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L185
func (p *ProjectService) ports() []composego.ServicePortConfig {
	prts := []composego.ServicePortConfig{}
	exist := map[string]bool{}

	for _, port := range p.Ports {
		prts = append(prts, port)
		exist[cast.ToString(port.Target)+strings.ToUpper(port.Protocol)] = true
	}

	// Compose Expose ports aren't published to the host - they are meant to be accessed only by linked services.
	// We simply map them onto the list of existing ports, see above.
	// https://docs.docker.com/compose/compose-file/#expose
	if p.Expose != nil {
		for _, port := range p.Expose {
			portValue := port
			protocol := v1.ProtocolTCP

			// @todo - this seem invalid as expose can only specify individual ports
			// if strings.Contains(portValue, "/") {
			// 	splits := strings.Split(port, "/")
			// 	portValue = splits[0]
			// 	protocol = v1.Protocol(strings.ToUpper(splits[1]))
			// }

			if exist[portValue+string(protocol)] {
				continue
			}

			prts = append(prts, composego.ServicePortConfig{
				Target:    cast.ToUint32(portValue),
				Published: cast.ToUint32(portValue),
				Protocol:  string(protocol),
			})
		}
	}

	return prts
}

func (p *ProjectService) LivenessProbe() (*v1.Probe, error) {
	p1 := composego.ServiceConfig(p.ServiceConfig)
	k8sconf, err := config.SvcK8sConfigFromCompose(&p1)
	if err != nil {
		return nil, err
	}

	return LivenessProbeToV1Probe(k8sconf.Workload.LivenessProbe)
}

func (p *ProjectService) ReadinessProbe() (*v1.Probe, error) {
	p1 := composego.ServiceConfig(p.ServiceConfig)
	k8sconf, err := config.SvcK8sConfigFromCompose(&p1)
	if err != nil {
		return nil, err
	}

	return ReadinessProbeToV1Probe(k8sconf.Workload.ReadinessProbe)
}
