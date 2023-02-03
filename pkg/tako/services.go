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

package tako

import (
	"encoding/json"
	"regexp"

	"github.com/appvia/tako/pkg/tako/config"
	composego "github.com/compose-spec/compose-go/types"
)

func newServiceConfig(s composego.ServiceConfig) (ServiceConfig, error) {
	config := ServiceConfig{
		Name:        s.Name,
		Image:       s.Image,
		Environment: s.Environment,
		Extensions:  s.Extensions,
	}
	return config, nil
}

// MarshalYAML makes Services implement yaml.Marshaller.
func (s Services) MarshalYAML() (interface{}, error) {
	services := map[string]ServiceConfig{}
	for _, service := range s {
		services[service.Name] = service
	}
	return services, nil
}

// MarshalJSON makes Services implement json.Marshaler.
func (s Services) MarshalJSON() ([]byte, error) {
	data, err := s.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(data, "", "  ")
}

// Map converts services to a map.
func (s Services) Map() map[string]ServiceConfig {
	out := map[string]ServiceConfig{}
	for _, service := range s {
		out[service.Name] = service
	}
	return out
}

// Set converts services to a set.
func (s Services) Set() map[string]bool {
	out := map[string]bool{}
	for _, service := range s {
		out[service.Name] = true
	}
	return out
}

func (sc ServiceConfig) detectSecretsInEnvVars(matchers []map[string]string) []secretHit {
	var matches []secretHit

	for key, val := range sc.Environment {
		for _, matcher := range matchers {
			var candidate string

			if matcher["part"] == config.PartIdentifier {
				candidate = key
			}

			if matcher["part"] == matcher[config.PartIdentifier] {
				candidate = *val
			}

			if found := regexp.MustCompile(matcher["match"]).MatchString(candidate); found {
				matches = append(matches, secretHit{sc.Name, key, matcher["description"]})
				break
			}
		}
	}

	return matches
}
