package config

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

// ExtensionRoot represents the root of the docker-compose extensions
type ExtensionRoot struct {
	K8S K8SConfiguration `yaml:"x-k8s"`
}

// K8SConfiguration represents the root of the k8s specific fields supported by kev.
type K8SConfiguration struct {
	Enabled  bool     `yaml:"enabled,omitempty"`
	Workload Workload `yaml:"workload,omitempty" validate:"required"`
	Service  Service  `yaml:"service,omitempty"`
}

// DefaultK8SConfig returns a K8SServiceConfig with all the defaults set into it.
func DefaultK8SConfig() K8SConfiguration {
	return K8SConfiguration{
		Enabled: DefaultServiceEnabled,
		Workload: Workload{
			Type:           DefaultWorkload,
			LivenessProbe:  DefaultLivenessProbe(),
			ReadinessProbe: DefaultReadinessProbe(),
		},
	}
}

// K8SCfgFromMap handles the extraction of the k8s-specific extension values from the top level map.
func K8SCfgFromMap(m map[string]interface{}) (K8SConfiguration, error) {
	if _, ok := m["x-k8s"]; !ok {
		c := DefaultK8SConfig()
		c.Workload.Replicas = DefaultReplicaNumber
		return c, nil
	}

	var extensions ExtensionRoot

	var buf bytes.Buffer
	if err := yaml.NewEncoder(&buf).Encode(m); err != nil {
		return K8SConfiguration{}, err
	}

	if err := yaml.NewDecoder(&buf).Decode(&extensions); err != nil {
		return K8SConfiguration{}, err
	}

	err := validator.New().Struct(extensions.K8S.Workload)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, e := range validationErrors {
			if e.Tag() == "required" {
				return K8SConfiguration{}, fmt.Errorf("%s is required", e.StructNamespace())
			}
		}

		return K8SConfiguration{}, errors.New(validationErrors[0].Error())
	}

	k8s := DefaultK8SConfig()

	if err := mergo.Merge(&extensions.K8S, k8s); err != nil {
		return K8SConfiguration{}, err
	}

	return extensions.K8S, nil
}

// Workload holds all the workload-related k8s configurations.
type Workload struct {
	Type           string         `yaml:"type,omitempty"`
	Replicas       int            `yaml:"replicas,omitempty" validate:"required"`
	LivenessProbe  LivenessProbe  `yaml:"livenessProbe,omitempty"`
	ReadinessProbe ReadinessProbe `yaml:"readinessProbe,omitempty"`
}

// Service will hold the service specific extensions in the future.
// TODO: expand with new properties.
type Service struct {
}
