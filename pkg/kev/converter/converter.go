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
	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/appvia/kube-devx/pkg/kev/converter/dummy"
	"github.com/appvia/kube-devx/pkg/kev/converter/kubernetes"
)

// Converter is an interface implemented by each converter kind
type Converter interface {
	// Render builds an output for an app
	Render(singleFile bool, dir string, app *app.Definition) error
}

// Factory returns a converter
func Factory(name string) Converter {
	switch name {
	case "dummy":
		// Dummy converter example
		return dummy.New()
	default:
		// Kubernetes manifests converter by default
		return kubernetes.New()
	}
}
