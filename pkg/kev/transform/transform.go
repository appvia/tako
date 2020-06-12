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

package transform

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Transform is a transform func type.
// Documents how a transform func should be created.
// Useful as a function param for functions that accept transforms.
type Transform func(data []byte) ([]byte, error)

// unmarshall unmarshalls a data byte into an map[string]interface{}
func unmarshall(data []byte) (map[string]interface{}, error) {
	var out map[string]interface{}
	err := yaml.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EchoTransform can be used to view data at different stages of
// a transform pipeline.
func EchoTransform(data []byte) ([]byte, error) {
	x, err := unmarshall(data)
	if err != nil {
		return []byte{}, err
	}
	fmt.Println(string(data))
	return yaml.Marshal(x)
}
