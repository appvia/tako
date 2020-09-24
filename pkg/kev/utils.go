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

import "sort"

// contains returns true of slice of strings contains a given string
func contains(src []string, s string) bool {
	sort.Strings(src)
	i := sort.SearchStrings(src, s)
	return i < len(src) && src[i] == s
}

// keys returns keys for a given map[string]string
func keys(src map[string]string) []string {
	out := make([]string, 0, len(src))
	for k := range src {
		out = append(out, k)
	}
	return out
}
