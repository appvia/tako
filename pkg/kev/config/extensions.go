package config

import (
	"bytes"
	"errors"
	"fmt"
	"time"

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

// LivenessProbe holds all the settings for the same k8s probe.
type LivenessProbe struct {
	// TODO: find a decent way of using ProbeType here that validates the content of the string
	Type        string `yaml:"type" validate:"required"`
	ProbeConfig `yaml:",inline"`
}

// DefaultLivenessProbe creates a default liveness probe. Defaults to exec.
func DefaultLivenessProbe() LivenessProbe {
	id, _ := time.ParseDuration(DefaultProbeInitialDelay)
	p, _ := time.ParseDuration(DefaultProbeInterval)
	t, _ := time.ParseDuration(DefaultProbeTimeout)

	return LivenessProbe{
		Type: ProbeTypeExec.String(),
		ProbeConfig: ProbeConfig{
			Exec: ExecProbe{
				Command: DefaultLivenessProbeCommand,
			},
			InitialDelay:      id,
			Period:            p,
			FailureThreashold: DefaultProbeRetries,
			Timeout:           t,
		},
	}
}

// DefaultReadinessProbe defines the default readiness probe. Defaults to none.
func DefaultReadinessProbe() ReadinessProbe {
	id, _ := time.ParseDuration(DefaultProbeInitialDelay)
	p, _ := time.ParseDuration(DefaultProbeInterval)
	t, _ := time.ParseDuration(DefaultProbeTimeout)

	return ReadinessProbe{
		Type: ProbeTypeNone.String(),
		ProbeConfig: ProbeConfig{
			InitialDelay:      id,
			Period:            p,
			FailureThreashold: DefaultProbeRetries,
			Timeout:           t,
		},
	}
}

// ReadinessProbe holds all the settings for the same k8s probe.
type ReadinessProbe struct {
	// TODO: find a decent way of using ProbeType here that validates the content of the string
	Type        string `yaml:"type"`
	ProbeConfig `yaml:",inline"`
}

// ProbeConfig holds all the shared properties between liveness and readiness probe.
type ProbeConfig struct {
	HTTP HTTPProbe `yaml:"http"`
	TCP  TCPProbe  `yaml:"tcp"`
	Exec ExecProbe `yaml:"exec"`

	InitialDelay      time.Duration `yaml:"initialDelay"`
	Period            time.Duration `yaml:"period"`
	FailureThreashold int           `yaml:"failureThreashold"`
	Timeout           time.Duration `yaml:"timeout"`
}

// HTTPProbe holds the necessary properties to define the http check on the k8s probe.
type HTTPProbe struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

// TCPProbe holds the necessary properties to define the tcp check on the k8s probe.
type TCPProbe struct {
	Port int `yaml:"port"`
}

// ExecProbe holds the necessary properties to define the exec check on the k8s probe.
type ExecProbe struct {
	Command string `yaml:"command"`
}

// Service will hold the service specific extensions in the future.
// TODO: expand with new properties.
type Service struct {
}
