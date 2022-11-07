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

package converter

import (
	"github.com/appvia/kev/pkg/kev/converter/dummy"
	kubernetes "github.com/appvia/kev/pkg/kev/converter/kubernetes"
	kustomize "github.com/appvia/kev/pkg/kev/converter/kustomize"
	kmd "github.com/appvia/komando"
	composego "github.com/compose-spec/compose-go/types"
)

// Converter is an interface implemented by each converter kind
type Converter interface {
	// Render builds an output for an app
	Render(singleFile bool,
		dir, workDir string,
		projects map[string]*composego.Project,
		files map[string][]string,
		rendered map[string][]byte,
		excluded map[string][]string) (map[string]string, error)
}

// Factory returns a converter
func Factory(name string, ui kmd.UI) Converter {
	switch name {
	case "dummy":
		// Dummy converter example
		return dummy.New()
	case "kustomize":
		// Kubernetes manifests converter by default
		if ui == nil {
			return kustomize.New()
		}
		return kustomize.NewWithUI(ui)

	default:
		// Kubernetes manifests converter by default
		if ui == nil {
			return kubernetes.New()
		}
		return kubernetes.NewWithUI(ui)
	}
}
