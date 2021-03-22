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

package kev

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/log"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/xeipuuv/gojsonschema"
)

func newServiceConfig(s composego.ServiceConfig) (ServiceConfig, error) {
	config := ServiceConfig{Name: s.Name, Labels: s.Labels, Environment: s.Environment, Extensions: s.Extensions}
	return config, config.validate()
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

func (s Services) detectSecrets(matchers []map[string]string, detectedFn func()) bool {
	var matches []secretHit
	for _, svc := range s {
		matches = append(matches, svc.detectSecretsInEnvVars(matchers)...)
	}

	if len(matches) == 0 {
		return false
	}

	detectedFn()
	for _, m := range matches {
		log.Warnf("Service [%s], env var [%s] looks like a secret", m.svcName, m.envVar)
	}
	return true
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

// validate runs validation of kev labels against config schema validation.
// IMPORTANT: All labels prefixed with `kev.` are always validated!
//            Other labels are left as is and won't be validated!
// NOTE: non kev labels are normally converted to annotations which is handy
// with things like ingress objects accepting a number of custom annotations.
// @todo(mc): post migrating labels to extensions this could be addressed by
// dedicated `annotations` section.
func (sc ServiceConfig) validate() error {
	svcLabelsToValidate := composego.Labels{}
	for slk, slv := range sc.Labels {
		if strings.HasPrefix(slk, config.LabelPrefix) {
			svcLabelsToValidate[slk] = slv
		}
	}

	ls := gojsonschema.NewGoLoader(config.ServicesSchema)
	ld := gojsonschema.NewGoLoader(svcLabelsToValidate)

	result, err := gojsonschema.Validate(ls, ld)
	if err != nil {
		return err
	}

	if result.Valid() {
		return nil
	}

	// Prioritise clear error messages.
	if e := findError(result, withType("required")); e != nil {
		return errors.New(e.Description())
	}

	// Exclude errors that are very cryptic and hurt usability.
	if e := findError(result, excludeTypes("number_one_of", "number_any_of", "number_all_of")); e != nil {
		return errors.New(e.Description())
	}

	// If we don't find anything useful just go with whatever is available.
	return errors.New(result.Errors()[0].Description())
}

func excludeTypes(ts ...string) func(gojsonschema.ResultError) bool {
	return func(re gojsonschema.ResultError) bool {
		for _, t := range ts {
			if t == re.Type() {
				return false
			}
		}
		return true
	}
}

func withType(t string) func(gojsonschema.ResultError) bool {
	return func(re gojsonschema.ResultError) bool {
		return t == re.Type()
	}
}

func findError(result *gojsonschema.Result, predicate func(re gojsonschema.ResultError) bool) gojsonschema.ResultError {
	if result.Valid() {
		return nil
	}

	for _, e := range result.Errors() {
		if predicate(e) {
			return e
		}
	}

	return nil
}

// GetLabels gets a service's labels
func (sc ServiceConfig) GetLabels() map[string]string {
	return sc.Labels
}

// minusEnvVars returns a copy of the ServiceConfig with blank env vars
func (sc ServiceConfig) minusEnvVars() ServiceConfig {
	return ServiceConfig{
		Name:        sc.Name,
		Labels:      sc.Labels,
		Environment: map[string]*string{},
	}
}

// condenseLabels returns a copy of the ServiceConfig with only condensed base service labels
func (sc ServiceConfig) condenseLabels(labels []string) ServiceConfig {
	for key := range sc.GetLabels() {
		if !contains(labels, key) {
			delete(sc.Labels, key)
		}
	}

	return ServiceConfig{
		Name:        sc.Name,
		Labels:      sc.Labels,
		Environment: sc.Environment,
	}
}
