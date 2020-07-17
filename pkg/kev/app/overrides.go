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

package app

import (
	"fmt"
	"sort"
	"strings"
)

// OverrideNames gets override names for a definition
func (def *Definition) OverrideNames() []string {
	var out []string
	for key, _ := range def.Overrides {
		out = append(out, key)
	}
	return out
}

// ValidateHasOverrides validates if a definition has a set of overrides
func (def *Definition) ValidateHasOverrides(candidates []string) error {
	overrides := def.OverrideNames()

	sort.Strings(overrides)
	var invalid []string

	for _, c := range candidates {
		i := sort.SearchStrings(overrides, c)
		valid := i < len(overrides) && overrides[i] == c
		if !valid {
			invalid = append(invalid, c)
		}
	}

	if len(invalid) > 0 {
		return fmt.Errorf("cannot find environment(s): %s", strings.Join(invalid, ", "))
	}

	return nil
}

// ExcludeOtherOverrides removes app and build overrides from a loaded definition not passed in as a param.
// This does not remove the overrides from the actual app.
func (def *Definition) ExcludeOtherOverrides(overrides []string) {
	flattened := strings.Join(overrides, " ")
	for o, _ := range def.Overrides {
		if !strings.Contains(flattened, o) {
			delete(def.Overrides, o)
		}
	}
	for o, _ := range def.Build.Overrides {
		if !strings.Contains(flattened, o) {
			delete(def.Build.Overrides, o)
		}
	}
}
