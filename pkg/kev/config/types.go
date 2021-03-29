package config

import (
	"errors"
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
