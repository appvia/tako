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

// Note: Types below have been extracted from Kompose project and updated accordingly
// to meet new dependencies and our requirements.
// Original code ref: https://github.com/kubernetes/kompose/blob/78908c94e5168984791ed57a0dd30651d4e70fc1/pkg/kobject/kobject.go

package kubernetes

import (
	composego "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
)

// ConvertOptions holds all options that controls transformation process
type ConvertOptions struct {
	ToStdout     bool     // Display output to STDOUT
	CreateChart  bool     // Create K8s manifests as Chart
	GenerateJSON bool     // Ganerate outcome as JSON. By defaults YAML gets generated.
	EmptyVols    bool     // Treat all referenced volumes as Empty volumes
	Volumes      string   // Volumes to be generated ("persistentVolumeClaim"|"emptyDir"|"hostPath"|"configMap") (default "persistentVolumeClaim")
	InputFiles   []string // Compose files to be processed
	OutFile      string   // If Directory output will be split into individual files
	YAMLIndent   int      // YAML Indentation in resultant K8s manifests
}

// Volumes holds the container volume struct
type Volumes struct {
	SvcName       string // Service name to which volume is linked
	MountPath     string // Mountpath extracted from docker-compose file
	VFrom         string // denotes service name from which volume is coming
	VolumeName    string // name of volume if provided explicitly
	Host          string // host machine address
	Container     string // Mountpath
	Mode          string // access mode for volume
	PVCName       string // name of PVC
	PVCSize       string // PVC size
	StorageClass  string // PVC storage class
	SelectorValue string // Value of the label selector
}

// ProjectService is a wrapper type around composego.ServiceConfig
type ProjectService composego.ServiceConfig

// ErrUnsupportedProbeType should be returned when an unsupported probe type is provided.
var ErrUnsupportedProbeType = errors.New("unsupported probe type")

// ProbeType defines all possible types of kubernetes probes
type ProbeType int

// Valid checks if a ProbeType contains an expected value.
func (p ProbeType) Valid() bool {
	_, ok := probeString[p]

	return ok
}

// String returns the string representation of a ProbeType.
func (p ProbeType) String() string {
	s, ok := probeString[p]
	if !ok {
		return ""
	}

	return s
}

var probeString map[ProbeType]string = map[ProbeType]string{
	ProbeTypeNone:    "none",
	ProbeTypeCommand: "command",
	ProbeTypeHTTP:    "http",
	ProbeTypeTCP:     "tcp",
}

const (
	// ProbeTypeNone disables probe checks.
	ProbeTypeNone ProbeType = iota
	// ProbeTypeCommand uses a shell command for probe checks.
	ProbeTypeCommand
	// ProbeTypeHTTP defines an http request which is used by probe checks.
	ProbeTypeHTTP
	// ProbeTypeTCP defines a tcp port which is used by probe checks.
	ProbeTypeTCP
)

// ProbeTypeFromString finds the ProbeType from it's string representation or returns Disabled as a default.
func ProbeTypeFromString(s string) (ProbeType, bool) {
	for k, v := range probeString {
		if s == v {
			return k, true
		}
	}

	return ProbeTypeNone, false
}
