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

package config

import (
	"io/ioutil"
	"path"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

// CompiledConfig holds compiled configuration content and suggested file path
type CompiledConfig struct {
	Content  []byte
	FilePath string
}

// Compile calculates effective configuration with base configuration
// extended/overidden by environment specific configuration
func Compile(root, name, env string) (*CompiledConfig, error) {
	appDir := path.Join(root, name)
	appBaseConfigPath := path.Join(appDir, "config.yaml")
	appEnvConfigPath := path.Join(appDir, env, "config.yaml")
	appCompiledConfigPath := path.Join(appDir, env, "config-compiled.yaml")

	// Read and unmarshal base configuration
	baseConfigContent, err := ioutil.ReadFile(appBaseConfigPath)
	if err != nil {
		return &CompiledConfig{}, err
	}
	baseConfig := Config{}
	_ = yaml.Unmarshal([]byte(baseConfigContent), &baseConfig)

	// Read and unmarshal env configuration
	envConfigContent, err := ioutil.ReadFile(appEnvConfigPath)
	if err != nil {
		return &CompiledConfig{}, err
	}
	envConfig := Config{}
	_ = yaml.Unmarshal([]byte(envConfigContent), &envConfig)

	mergo.Merge(&envConfig, baseConfig)

	compiledConfigBytes, err := envConfig.Bytes()
	if err != nil {
		return &CompiledConfig{}, err
	}

	return &CompiledConfig{
		Content:  compiledConfigBytes,
		FilePath: appCompiledConfigPath,
	}, nil
}
