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
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ServiceType string

const (
	// NoService default value
	NoService ServiceType = "None"

	// NodePortService svc type
	NodePortService ServiceType = "NodePort"

	// LoadBalancerService svc type
	LoadBalancerService ServiceType = "LoadBalancer"

	// ClusterIPService svc type
	ClusterIPService ServiceType = "ClusterIP"

	// HeadlessService svc type
	HeadlessService ServiceType = "Headless"
)

// String converts a service type to a string value
func (s ServiceType) String() string {
	return string(s)
}

// serviceTypes are the only service type settings
var serviceTypes = map[ServiceType]bool{
	NoService:           true,
	NodePortService:     true,
	LoadBalancerService: true,
	ClusterIPService:    true,
	HeadlessService:     true,
}

// ServiceTypeFromValue returns a Service Type for a given case insensitive value.
// Returns a blank string and false for unknown values.
func ServiceTypeFromValue(s string) (ServiceType, bool) {
	for k, v := range serviceTypes {
		if strings.ToLower(k.String()) == strings.ToLower(s) {
			return k, v
		}
	}
	return "", false
}

// ServiceTypesEqual checks if the supplied ServiceTypes are equal
func ServiceTypesEqual(s, t ServiceType) bool {
	return strings.ToLower(s.String()) == strings.ToLower(t.String())
}

// validateServiceType validator to validate a service type
func validateServiceType(fl validator.FieldLevel) bool {
	_, valid := ServiceTypeFromValue(fl.Field().String())
	return valid
}

// inferServiceTypeFromComposeValue returns service type based on passed string value
// @orig: https://github.com/kubernetes/kompose/blob/1f0a097836fb4e0ae4a802eb7ab543a4f9493727/pkg/loader/compose/utils.go#L108
// func inferServiceTypeFromComposeValue(v string) (string, error) {
func inferServiceTypeFromComposeValue(v string) (ServiceType, error) {
	switch strings.ToLower(v) {
	case "", "clusterip":
		return ClusterIPService, nil
	case "nodeport":
		return NodePortService, nil
	case "loadbalancer":
		return LoadBalancerService, nil
	case "headless":
		return HeadlessService, nil
	case "none":
		return NoService, nil
	default:
		return "", fmt.Errorf("unknown value %s, supported values are 'none, nodeport, clusterip, headless or loadbalancer'", v)
	}
}
