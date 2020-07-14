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
	"strings"

	"gopkg.in/yaml.v3"
)

// Definition is the app definition including base compose, config files.
// And, environment config files. Also, build config related to the most recent build.
type Definition struct {
	Base      ConfigTuple
	Overrides map[string]FileConfig // maps environment name to its configuration
	Build     BuildConfig
}

func (def *Definition) RootDir() string {
	return baseDir
}

func (def *Definition) WorkDir() string {
	return workDir
}

func (def *Definition) WorkPath() string {
	return path.Join(def.RootDir(), def.WorkDir())
}

func (def *Definition) BuildDir() string {
	return buildDir
}

func (def *Definition) BuildPath() string {
	return path.Join(def.RootDir(), def.WorkDir(), def.BuildDir())
}

// BuildConfig is an app definition's build config.
// It contains base config along with interpolated compose.
// And, compiled config files and interpolated compose files per overriding environment.
type BuildConfig struct {
	Base      ConfigTuple
	Overrides map[string]ConfigTuple
}

// FileConfig details an app definition FileConfig, including its Content and recommended file path.
type FileConfig struct {
	Content []byte
	File    string
}

// ConfigTuple is a configuration spec containing information about compose and config file pair
type ConfigTuple struct {
	Compose FileConfig
	Config  FileConfig
}

// Dir returns the application config's immediate parent directory
func (c FileConfig) Dir() string {
	parts := strings.Split(c.Path(), "/")
	return parts[len(parts)-1]
}

// Path returns the application config's directory path
func (c FileConfig) Path() string {
	return path.Dir(c.File)
}

// OverrideConfig to ensure ordering of params in an environment's config.yaml
type OverrideConfig struct {
	// Defines app default Kubernetes workload parameters.
	Workload *yaml.Node `yaml:",omitempty" json:"workload,omitempty"`
	// Defines app default component K8s service parameters.
	Service *yaml.Node `yaml:",omitempty" json:"service,omitempty"`
	// Control volumes defined in compose file by specifing storage class and size.
	Volumes *yaml.Node `yaml:",omitempty" json:"volumes,omitempty"`
	// Map of defined compose services
	Components map[string]*yaml.Node `yaml:",omitempty,inline" json:"components,omitempty,inline"`
}
