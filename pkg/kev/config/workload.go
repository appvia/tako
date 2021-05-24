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
	"strings"

	"github.com/go-playground/validator/v10"
)

type WorkloadType string

const (
	// DeploymentWorkload workload type
	DeploymentWorkload WorkloadType = "Deployment"

	// DaemonSetWorkload workload type
	DaemonSetWorkload WorkloadType = "DaemonSet"

	// StatefulSetWorkload workload type
	StatefulSetWorkload WorkloadType = "StatefulSet"
)

// String converts a workload type to a string value
func (w WorkloadType) String() string {
	return string(w)
}

// workloadTypes are the only workload type settings
var workloadTypes = map[WorkloadType]bool{
	DeploymentWorkload:  true,
	DaemonSetWorkload:   true,
	StatefulSetWorkload: true,
}

// WorkloadTypeFromValue returns a Workload Type for a given case insensitive value.
// Returns a blank string and false for unknown values.
func WorkloadTypeFromValue(s string) (WorkloadType, bool) {
	for k, v := range workloadTypes {
		if strings.ToLower(k.String()) == strings.ToLower(s) {
			return k, v
		}
	}
	return "", false
}

// WorkloadTypesEqual checks if the supplied WorkloadTypes are equal
func WorkloadTypesEqual(s, t WorkloadType) bool {
	return strings.ToLower(s.String()) == strings.ToLower(t.String())
}

// validateWorkloadType validator to validate a workload type
func validateWorkloadType(fl validator.FieldLevel) bool {
	_, valid := WorkloadTypeFromValue(fl.Field().String())
	return valid
}
