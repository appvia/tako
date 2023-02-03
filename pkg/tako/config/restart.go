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

type RestartPolicy string

const (
	// RestartPolicyAlways default value
	RestartPolicyAlways RestartPolicy = "Always"

	// RestartPolicyOnFailure restart policy
	RestartPolicyOnFailure RestartPolicy = "OnFailure"

	// RestartPolicyNever restart policy
	RestartPolicyNever RestartPolicy = "Never"
)

// String converts a restart policy to a string value
func (p RestartPolicy) String() string {
	return string(p)
}

// restartPolicies are the only restart policy settings
var restartPolicies = map[RestartPolicy]bool{
	RestartPolicyAlways:    true,
	RestartPolicyOnFailure: true,
	RestartPolicyNever:     true,
}

// RestartPoliciesFromValue returns a Restart Policy for a given case insensitive value.
// Returns a blank string and false for unknown values.
func RestartPoliciesFromValue(s string) (RestartPolicy, bool) {
	for k, v := range restartPolicies {
		if strings.ToLower(k.String()) == strings.ToLower(s) {
			return k, v
		}
	}
	return "", false
}

// validateRestartPolicy validator to validate a restart policy
func validateRestartPolicy(fl validator.FieldLevel) bool {
	_, valid := RestartPoliciesFromValue(fl.Field().String())
	return valid
}

// inferRestartPolicyFromComposeValue infers a Restart Policy for a compose value
func inferRestartPolicyFromComposeValue(v string) RestartPolicy {
	switch strings.ToLower(v) {
	case "", "always", "any":
		return RestartPolicyAlways
	case "no", "none", "never":
		return RestartPolicyNever
	case "on-failure", "onfailure":
		return RestartPolicyOnFailure
	case "unless-stopped":
		return RestartPolicyAlways
	default:
		return RestartPolicyAlways
	}
}
