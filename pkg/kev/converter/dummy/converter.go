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
	"github.com/appvia/kev/pkg/kev/log"
	composego "github.com/compose-spec/compose-go/types"
)

// Name of the converter
const Name = "dummy"

// Dummy is a dummy converter adapter
type Dummy struct{}

// New return a dummy converter
func New() *Dummy {
	return &Dummy{}
}

// Render generates outcome
func (c *Dummy) Render(singleFile bool,
	dir, workDir string,
	projects map[string]*composego.Project,
	files map[string][]string,
	additionalManifests []string,
	rendered map[string][]byte,
	excluded map[string][]string) (map[string]string, error) {

	log.Debugf("Hello from %s adapter Render()", Name)
	return nil, nil

}
