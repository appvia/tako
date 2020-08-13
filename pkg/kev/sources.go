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
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CalculateBaseOverlay calculates the base set of labels deduced from a group of compose sources.
func (s *Sources) CalculateBaseOverlay() error {
	ready, err := NewComposeProject(s.Files, WithTransforms)
	if err != nil {
		return err
	}
	s.overlay = extractLabels(ready)
	return nil
}

// MarshalYAML makes Sources implement yaml.Marshaler.
func (s *Sources) MarshalYAML() (interface{}, error) {
	var out []string
	for _, f := range s.Files {
		out = append(out, f)
	}
	return out, nil
}

// UnmarshalYAML makes Sources implement yaml.UnmarshalYAML.
func (s *Sources) UnmarshalYAML(value *yaml.Node) error {
	for i := 0; i < len(value.Content); i += 1 {
		s.Files = append(s.Files, value.Content[i].Value)
	}
	return nil
}

// MarshalJSON makes Sources implement json.Marshaler.
func (s *Sources) MarshalJSON() ([]byte, error) {
	data, err := s.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(data, "", "  ")
}

func (s *Sources) GetWorkingDir() string {
	if len(s.Files) < 1 {
		return ""
	}
	return filepath.Dir(s.Files[0])
}
