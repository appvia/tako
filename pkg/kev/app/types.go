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

	"gopkg.in/yaml.v3"
)

// Definition provides details for the app's base compose and config files.
type Definition struct {
	Name        string
	BaseCompose FileConfig
	Config      FileConfig
	Envs        []FileConfig
}

// FileConfig details an app definition FileConfig, including its Content and recommended file path.
type FileConfig struct {
	Content []byte
	File    string
}

func (c FileConfig) Dir() string {
	return path.Dir(c.File)
}

// EnvConfig to ensure ordering of params in an environment's config.yaml
type EnvConfig struct {
	// Defines app default Kubernetes workload parameters.
	Workload *yaml.Node `yaml:",omitempty" json:"workload,omitempty"`
	// Defines app default component K8s service parameters.
	Service *yaml.Node `yaml:",omitempty" json:"service,omitempty"`
	// Control volumes defined in compose file by specifing storage class and size.
	Volumes *yaml.Node `yaml:",omitempty" json:"volumes,omitempty"`
	// Map of defined compose services
	Components map[string]*yaml.Node `yaml:",omitempty,inline" json:"components,omitempty,inline"`
}
