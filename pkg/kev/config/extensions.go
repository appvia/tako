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
	K8S K8SServiceConfig `yaml:"x-k8s"`
}

// K8SServiceConfig represents the root of the k8s specific fields supported by kev.
type K8SServiceConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Workload Workload `yaml:"workload" validate:"required"`
	Service  Service  `yaml:"service"`
}

// DefaultK8SServiceConfig returns a K8SServiceConfig with all the defaults set into it.
func DefaultK8SServiceConfig() K8SServiceConfig {
	return K8SServiceConfig{
		Enabled: DefaultServiceEnabled,
		Workload: Workload{
			Type:           DefaultWorkload,
			LivenessProbe:  DefaultLivenessProbe(),
			ReadinessProbe: DefaultReadinessProbe(),
		},
	}
}

// K8SServiceCfgFromMap handles the extraction of the k8s-specific extension values from the top level map.
func K8SServiceCfgFromMap(m map[string]interface{}) (K8SServiceConfig, error) {
	var extensions ExtensionRoot

	var buf bytes.Buffer
	if err := yaml.NewEncoder(&buf).Encode(m); err != nil {
		return K8SServiceConfig{}, err
	}

	if err := yaml.NewDecoder(&buf).Decode(&extensions); err != nil {
		return K8SServiceConfig{}, err
	}

	err := validator.New().Struct(extensions.K8S.Workload)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, e := range validationErrors {
			if e.Tag() == "required" {
				return K8SServiceConfig{}, fmt.Errorf("%s is required", e.StructNamespace())
			}
		}

		return K8SServiceConfig{}, errors.New(validationErrors[0].Error())
	}

	k8s := DefaultK8SServiceConfig()

	if err := mergo.Merge(&extensions.K8S, k8s); err != nil {
		return K8SServiceConfig{}, err
	}

	return extensions.K8S, nil
}

// Workload holds all the workload-related k8s configurations.
type Workload struct {
	Type           string         `yaml:"type"`
	Replicas       int            `yaml:"replicas" validate:"required"`
	LivenessProbe  LivenessProbe  `yaml:"livenessProbe"`
	ReadinessProbe ReadinessProbe `yaml:"readinessProbe"`
}

// Service will hold the service specific extensions in the future.
// TODO: expand with new properties.
type Service struct {
}
