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
	"path"
	"sort"
)

// GetLastRenderInfo gets a list of the app's last rendered manifests .
func (def *Definition) GetLastRenderInfo() []FileConfig {
	var out []FileConfig
	for _, fc := range def.Rendered {
		out = append(out, fc)
	}
	return out
}

// RenderedFilenames gets the last rendered filenames
func (def *Definition) RenderedFilenames() []string {
	var out []string
	for _, r := range def.Rendered {
		out = append(out, path.Base(r.File))
	}
	sort.Strings(out)
	return out
}
