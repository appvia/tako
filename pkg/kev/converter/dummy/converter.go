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

package dummy

import (
	"fmt"

	"github.com/appvia/kube-devx/pkg/kev/app"
)

// Name of the converter
const Name = "dummy"

// Converter is a dummy adapter
type Converter struct{}

// New return a dummy converter
func New() *Converter {
	return &Converter{}
}

// Render generates outcome
func (c *Converter) Render(singleFile bool, dir string, app *app.Definition) error {
	fmt.Printf("Hello from %s adapter Render()\n", Name)
	return nil
}
