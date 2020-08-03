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
	"bytes"

	"github.com/appvia/kube-devx/pkg/kev/log"
	yaml3 "gopkg.in/yaml.v3"
)

// MarshalIndent marshals arbitrary struct and applies Indent to format the output
func MarshalIndent(v interface{}, indent int) ([]byte, error) {
	var out bytes.Buffer
	encoder := yaml3.NewEncoder(&out)
	defer func() {
		if err := encoder.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	encoder.SetIndent(indent)
	if err := encoder.Encode(&v); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
