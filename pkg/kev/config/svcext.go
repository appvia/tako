/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
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
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/appvia/kev/pkg/kev/log"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/go-playground/validator/v10"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const K8SExtensionKey = "x-k8s"

// ServiceExtension represents the root of the docker-compose extensions for a service
type ServiceExtension struct {
	K8S SvcK8sConfig `yaml:"x-k8s"`
}

// SvcK8sConfig represents the root of the k8s specific fields supported by kev.
type SvcK8sConfig struct {
	Disabled bool     `yaml:"disabled"`
	Workload Workload `yaml:"workload" validate:"required,dive"`
	Service  Service  `yaml:"service,omitempty"`
}

func (skc SvcK8sConfig) ToMap() (map[string]interface{}, error) {
	bs, err := yaml.Marshal(skc)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	err = yaml.Unmarshal(bs, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (skc SvcK8sConfig) Merge(other SvcK8sConfig) (SvcK8sConfig, error) {
	k8s := skc

	if err := mergo.Merge(&k8s, other, mergo.WithOverride); err != nil {
		return SvcK8sConfig{}, err
	}

	return k8s, nil
}

func (skc SvcK8sConfig) Validate() error {
	err := validator.New().Struct(skc)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, e := range validationErrors {
			if e.Tag() == "required" {
				return fmt.Errorf("%s is required", e.StructNamespace())
			}
		}

		return errors.New(validationErrors[0].Error())
	}

	return nil
}

// DefaultSvcK8sConfig returns a service's K8S Config with set defaults.
func DefaultSvcK8sConfig() SvcK8sConfig {
	return SvcK8sConfig{
		Disabled: false,
		Workload: Workload{
			Type:           DefaultWorkload,
			LivenessProbe:  DefaultLivenessProbe(),
			ReadinessProbe: DefaultReadinessProbe(),
			Replicas:       1,
			RestartPolicy:  RestartPolicyAlways,
			ImagePull: ImagePull{
				Policy: DefaultImagePullPolicy,
			},
			Autoscale: AutoscaleWithDefaults(),
		},
		Service: Service{
			Type: "None",
		},
	}
}

// SvcK8sConfigFromCompose creates a K8s service extension from a compose-go service.
// It extracts and infers values based on rules applied to the compose-go service.
func SvcK8sConfigFromCompose(svc *composego.ServiceConfig) (SvcK8sConfig, error) {
	var (
		cfg    SvcK8sConfig
		k8sExt SvcK8sConfig
	)

	cfg.Workload.Type = WorkloadTypeFromCompose(svc)
	cfg.Workload.Replicas = WorkloadReplicasFromCompose(svc)
	cfg.Workload.RestartPolicy = WorkloadRestartPolicyFromCompose(svc)
	svcType, err := ServiceTypeFromCompose(svc)
	if err != nil {
		return SvcK8sConfig{}, err
	}
	cfg.Service.Type = svcType

	cfg.Workload.LivenessProbe = LivenessProbeFromCompose(svc)
	cfg.Workload.ReadinessProbe = DefaultReadinessProbe()

	cfg.Workload.ImagePull = ImagePullWithDefaults()

	svcResource, err := ResourceFromCompose(svc)
	if err != nil {
		return SvcK8sConfig{}, err
	}

	cfg.Workload.Resource = svcResource
	cfg.Workload.Autoscale = AutoscaleWithDefaults()

	if _, ok := svc.Extensions[K8SExtensionKey]; ok {
		if k8sExt, err = ParseSvcK8sConfigFromMap(svc.Extensions, SkipValidation()); err != nil {
			return SvcK8sConfig{}, err
		}
	}

	cfg, err = cfg.Merge(k8sExt)
	if err != nil {
		return SvcK8sConfig{}, err
	}

	if err := cfg.Validate(); err != nil {
		return SvcK8sConfig{}, err
	}

	return cfg, nil
}

func ResourceFromCompose(svc *composego.ServiceConfig) (Resource, error) {
	var memLimit string
	var cpuLimit string
	if svc.Deploy != nil && svc.Deploy.Resources.Limits != nil {
		memLimit = getMemoryQuantity(int64(svc.Deploy.Resources.Limits.MemoryBytes))
		cpuLimit = svc.Deploy.Resources.Limits.NanoCPUs
	}

	var memRequest string
	var cpuRequest string
	if svc.Deploy != nil && svc.Deploy.Resources.Reservations != nil {
		memRequest = getMemoryQuantity(int64(svc.Deploy.Resources.Reservations.MemoryBytes))
		cpuRequest = svc.Deploy.Resources.Reservations.NanoCPUs
	}

	return Resource{
		MaxMemory: memLimit,
		Memory:    memRequest,
		CPU:       cpuRequest,
		MaxCPU:    cpuLimit,
	}, nil
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

func ImagePullWithDefaults() ImagePull {
	return ImagePull{
		Policy: DefaultImagePullPolicy,
		Secret: DefaultImagePullSecret,
	}
}

func AutoscaleWithDefaults() Autoscale {
	return Autoscale{
		MaxReplicas:     DefaultAutoscaleMaxReplicaNumber,
		CPUThreshold:    DefaultAutoscaleCPUThreshold,
		MemoryThreshold: DefaultAutoscaleMemoryThreshold,
	}
}

func ServiceTypeFromCompose(svc *composego.ServiceConfig) (string, error) {
	serviceType := NoService

	if len(svc.Ports) > 0 {
		serviceType = ClusterIPService
	}

	if svc.Deploy != nil && svc.Deploy.EndpointMode == "vip" {
		serviceType = NodePortService
	}

	serviceType, err := getServiceType(serviceType)
	if err != nil {
		log.ErrorWithFields(log.Fields{
			"service-name": svc.Name,
			"service-type": serviceType,
		}, "Unrecognised k8s service type. Compose project service will not have k8s service generated.")

		return "", fmt.Errorf("`%s` workload service type `%s` not supported", svc.Name, serviceType)
	}

	return serviceType, nil
}

// getServiceType returns service type based on passed string value
// @orig: https://github.com/kubernetes/kompose/blob/1f0a097836fb4e0ae4a802eb7ab543a4f9493727/pkg/loader/compose/utils.go#L108
func getServiceType(serviceType string) (string, error) {
	switch strings.ToLower(serviceType) {
	case "", "clusterip":
		return string(v1.ServiceTypeClusterIP), nil
	case "nodeport":
		return string(v1.ServiceTypeNodePort), nil
	case "loadbalancer":
		return string(v1.ServiceTypeLoadBalancer), nil
	case "headless":
		return HeadlessService, nil
	case "none":
		return NoService, nil
	default:
		return "", fmt.Errorf("Unknown value %s, supported values are 'none, nodeport, clusterip, headless or loadbalancer'", serviceType)
	}
}

// WorkloadRestartPolicyFromCompose infers a kev-valid restart policy from compose data.
func WorkloadRestartPolicyFromCompose(svc *composego.ServiceConfig) string {
	policy := RestartPolicyAlways

	if svc.Restart != "" {
		policy = svc.Restart
	}

	if svc.Deploy != nil && svc.Deploy.RestartPolicy != nil {
		policy = svc.Deploy.RestartPolicy.Condition
	}

	if policy == "unless-stopped" {
		log.WarnWithFields(log.Fields{
			"restart-policy": policy,
		}, "Restart policy 'unless-stopped' is not supported, converting it to 'always'")

		policy = "always"
	}

	return policy
}

func WorkloadReplicasFromCompose(svc *composego.ServiceConfig) int {
	if svc.Deploy == nil || svc.Deploy.Replicas == nil {
		return 1
	}

	return int(*svc.Deploy.Replicas)
}

// TODO: Turn these strings into enums
func WorkloadTypeFromCompose(svc *composego.ServiceConfig) string {
	if svc.Deploy != nil && svc.Deploy.Mode == "global" {
		return DaemonsetWorkload
	}

	if len(svc.Volumes) != 0 {
		return StatefulsetWorkload
	}

	return DeploymentWorkload
}

func LivenessProbeFromCompose(svc *composego.ServiceConfig) LivenessProbe {
	healthcheck := svc.HealthCheck
	var res LivenessProbe

	if healthcheck == nil {
		return DefaultLivenessProbe()
	}

	if healthcheck.Disable {
		res.Type = ProbeTypeNone.String()
		return res
	}

	res.Type = ProbeTypeExec.String()

	test := healthcheck.Test
	if len(test) > 0 && (strings.ToLower(test[0]) == "cmd" || strings.ToLower(test[0]) == "cmd-shell") {
		test = test[1:]
	}
	res.Exec.Command = test

	if healthcheck.Timeout != nil {
		res.Timeout = time.Duration(*healthcheck.Timeout)
	}

	if healthcheck.Retries != nil {
		res.FailureThreashold = int(*healthcheck.Retries)
	}

	if healthcheck.StartPeriod != nil {
		res.InitialDelay = time.Duration(*healthcheck.StartPeriod)
	}

	if healthcheck.Interval != nil {
		res.Period = time.Duration(*healthcheck.Interval)
	}

	return res
}

// ParseSvcK8sConfigFromMap handles the extraction of the k8s-specific extension values from the top level map.
func ParseSvcK8sConfigFromMap(m map[string]interface{}, opts ...K8sExtensionOption) (SvcK8sConfig, error) {
	var options extensionOptions
	for _, o := range opts {
		o(&options)
	}

	if _, ok := m[K8SExtensionKey]; !ok {
		return SvcK8sConfig{}, fmt.Errorf("missing %s service extension", K8SExtensionKey)
	}

	var extensions ServiceExtension

	var buf bytes.Buffer
	if err := yaml.NewEncoder(&buf).Encode(m); err != nil {
		return SvcK8sConfig{}, err
	}

	if err := yaml.NewDecoder(&buf).Decode(&extensions); err != nil {
		return SvcK8sConfig{}, err
	}

	if !options.skipValidation {
		if extensions.K8S.Workload.Type == "" {
			extensions.K8S.Workload.Type = DefaultWorkload
		}

		if err := extensions.K8S.Validate(); err != nil {
			return SvcK8sConfig{}, err
		}
	}

	return extensions.K8S, nil
}

// Workload holds all the workload-related k8s configurations.
type Workload struct {
	Type           string         `yaml:"type,omitempty" validate:"required,oneof=DaemonSet StatefulSet Deployment"`
	Replicas       int            `yaml:"replicas" validate:"required,gt=0"`
	LivenessProbe  LivenessProbe  `yaml:"livenessProbe" validate:"required"`
	ReadinessProbe ReadinessProbe `yaml:"readinessProbe,omitempty"`
	RestartPolicy  string         `yaml:"restartPolicy,omitempty"`
	ImagePull      ImagePull      `yaml:"imagePull,omitempty"`
	Resource       Resource       `yaml:"resource,omitempty"`
	Autoscale      Autoscale      `yaml:"autoscale,omitempty"`
}

type Resource struct {
	Memory    string `yaml:"memory,omitempty"`
	MaxMemory string `yaml:"maxMemory,omitempty"`
	CPU       string `yaml:"cpu,omitempty"`
	MaxCPU    string `yaml:"maxCpu,omitempty"`
}

type ImagePull struct {
	Policy string `yaml:"policy,omitempty" validate:"oneof='' IfNotPresent Never Always"`
	Secret string `yaml:"secret,omitempty"`
}

type Autoscale struct {
	MaxReplicas     int `yaml:"maxReplicas,omitempty"`
	CPUThreshold    int `yaml:"cpuThreshold,omitempty"`
	MemoryThreshold int `yaml:"memThreshold,omitempty"`
}

// Service will hold the service specific extensions in the future.
// TODO: expand with new properties.
type Service struct {
	Type string `yaml:"type" validate:"required,oneof=None NodePort ClusterIP Headless LoadBalancer"`
}
