package config

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

const K8SExtensionKey = "x-k8s"

// ExtensionRoot represents the root of the docker-compose extensions
type ExtensionRoot struct {
	K8S K8SConfiguration `yaml:"x-k8s"`
}

// K8SConfiguration represents the root of the k8s specific fields supported by kev.
type K8SConfiguration struct {
	Enabled  bool     `yaml:"enabled,omitempty"`
	Workload Workload `yaml:"workload" validate:"required,dive"`
	Service  Service  `yaml:"service,omitempty"`
}

func (k K8SConfiguration) ToMap() (map[string]interface{}, error) {
	bs, err := yaml.Marshal(k)
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

func (k K8SConfiguration) Merge(other K8SConfiguration) (K8SConfiguration, error) {
	k8s := k

	if err := mergo.Merge(&k8s, other, mergo.WithOverride); err != nil {
		return K8SConfiguration{}, err
	}

	return k8s, nil
}

func (k K8SConfiguration) Validate() error {
	err := validator.New().Struct(k.Workload)
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

// DefaultK8SConfig returns a K8SServiceConfig with all the defaults set into it.
func DefaultK8SConfig() K8SConfiguration {
	return K8SConfiguration{
		Enabled: DefaultServiceEnabled,
		Workload: Workload{
			Type:           DefaultWorkload,
			LivenessProbe:  DefaultLivenessProbe(),
			ReadinessProbe: DefaultReadinessProbe(),
			Replicas:       1,
		},
	}
}

type k8sConfigOptions struct {
	requireExtensions bool
}

// K8SCfgOption will modify parsing behaviour of the x-k8s extension.
type K8SCfgOption func(*k8sConfigOptions)

// RequireExtensions will ensure that x-k8s is present and that it is validated.
func RequireExtensions() K8SCfgOption {
	return func(kco *k8sConfigOptions) {
		kco.requireExtensions = true
	}
}

// K8SCfgFromMap handles the extraction of the k8s-specific extension values from the top level map.
func K8SCfgFromMap(m map[string]interface{}, opts ...K8SCfgOption) (K8SConfiguration, error) {
	var options k8sConfigOptions
	for _, o := range opts {
		o(&options)
	}

	if _, ok := m[K8SExtensionKey]; !ok && !options.requireExtensions {
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

	if err := extensions.K8S.Validate(); err != nil {
		return K8SConfiguration{}, err
	}

	k8s, err := DefaultK8SConfig().Merge(extensions.K8S)
	if err != nil {
		return K8SConfiguration{}, err
	}

	return k8s, nil
}

// Workload holds all the workload-related k8s configurations.
type Workload struct {
	Type           string         `yaml:"type,omitempty"`
	Replicas       int            `yaml:"replicas" validate:"required,gt=0"`
	LivenessProbe  LivenessProbe  `yaml:"livenessProbe" validate:"required"`
	ReadinessProbe ReadinessProbe `yaml:"readinessProbe,omitempty"`
}

// Service will hold the service specific extensions in the future.
// TODO: expand with new properties.
type Service struct {
}
