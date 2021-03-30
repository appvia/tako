package config

import (
	"errors"
	"time"
)

// ErrUnsupportedProbeType should be returned when an unsupported probe type is provided.
var ErrUnsupportedProbeType = errors.New("unsupported probe type")

// ProbeType defines all possible types of kubernetes probes
type ProbeType string

// Valid checks if a ProbeType contains an expected value.
func (p ProbeType) Valid() bool {
	return probeString[p]
}

// String returns the string representation of a ProbeType.
func (p ProbeType) String() string {
	if !p.Valid() {
		return ""
	}

	return string(p)
}

var probeString map[ProbeType]bool = map[ProbeType]bool{
	ProbeTypeNone: true,
	ProbeTypeExec: true,
	ProbeTypeHTTP: true,
	ProbeTypeTCP:  true,
}

var (
	// ProbeTypeNone disables probe checks.
	ProbeTypeNone ProbeType = "none"
	// ProbeTypeExec uses a shell command for probe checks.
	ProbeTypeExec ProbeType = "exec"
	// ProbeTypeHTTP defines an http request which is used by probe checks.
	ProbeTypeHTTP ProbeType = "http"
	// ProbeTypeTCP defines a tcp port which is used by probe checks.
	ProbeTypeTCP ProbeType = "tcp"
)

// ProbeTypeFromString finds the ProbeType from it's string representation or returns Disabled as a default.
func ProbeTypeFromString(s string) (ProbeType, bool) {
	if probeString[ProbeType(s)] {
		return ProbeType(s), true
	}

	return ProbeTypeNone, false
}

// LivenessProbe holds all the settings for the same k8s probe.
type LivenessProbe struct {
	// TODO: find a decent way of using ProbeType here that validates the content of the string
	Type        string `yaml:"type" validate:"required"`
	ProbeConfig `yaml:",inline"`
}

// DefaultLivenessProbe creates a default liveness probe. Defaults to exec.
func DefaultLivenessProbe() LivenessProbe {
	delay, _ := time.ParseDuration(DefaultProbeInitialDelay)
	interval, _ := time.ParseDuration(DefaultProbeInterval)
	timeout, _ := time.ParseDuration(DefaultProbeTimeout)

	return LivenessProbe{
		Type: ProbeTypeExec.String(),
		ProbeConfig: ProbeConfig{
			Exec: ExecProbe{
				Command: DefaultLivenessProbeCommand,
			},
			InitialDelay:      delay,
			Period:            interval,
			FailureThreashold: DefaultProbeRetries,
			Timeout:           timeout,
		},
	}
}

// ReadinessProbe holds all the settings for the same k8s probe.
type ReadinessProbe struct {
	// TODO: find a decent way of using ProbeType here that validates the content of the string
	Type        string `yaml:"type"`
	ProbeConfig `yaml:",inline"`
}

// DefaultReadinessProbe defines the default readiness probe. Defaults to none.
func DefaultReadinessProbe() ReadinessProbe {
	delay, _ := time.ParseDuration(DefaultProbeInitialDelay)
	interval, _ := time.ParseDuration(DefaultProbeInterval)
	timeout, _ := time.ParseDuration(DefaultProbeTimeout)

	return ReadinessProbe{
		Type: ProbeTypeNone.String(),
		ProbeConfig: ProbeConfig{
			InitialDelay:      delay,
			Period:            interval,
			FailureThreashold: DefaultProbeRetries,
			Timeout:           timeout,
		},
	}
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
