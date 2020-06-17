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

// CompiledConfig holds compiled environment configuration content and suggested file path
type CompiledConfig struct {
	Environment string
	Content     []byte
	FilePath    string
}

// Compile calculates effective configuration with base configuration
// extended/overidden by environments specific configuration
func Compile(root string, envs []string) ([]CompiledConfig, error) {
	appBaseConfigPath := path.Join(root, "config.yaml")

	// Read and unmarshal base configuration
	baseConfigContent, err := ioutil.ReadFile(appBaseConfigPath)
	if err != nil {
		return nil, err
	}
	baseConfig := Config{}
	if err = yaml.Unmarshal([]byte(baseConfigContent), &baseConfig); err != nil {
		return nil, err
	}

	var compiledConfigs []CompiledConfig

	for _, env := range envs {
		appEnvConfigPath := path.Join(root, env, "config.yaml")
		appCompiledConfigPath := path.Join(root, env, "config-compiled.yaml")

		// Read and unmarshal env configuration
		envConfigContent, err := ioutil.ReadFile(appEnvConfigPath)
		if err != nil {
			return nil, err
		}
		envConfig := Config{}
		if err = yaml.Unmarshal([]byte(envConfigContent), &envConfig); err != nil {
			return nil, err
		}

		mergo.Merge(&envConfig, baseConfig)

		compiledConfigBytes, err := envConfig.Bytes()
		if err != nil {
			return nil, err
		}

		compiledConfigs = append(compiledConfigs, CompiledConfig{
			Environment: env,
			Content:     compiledConfigBytes,
			FilePath:    appCompiledConfigPath,
		})
	}

	return compiledConfigs, nil
}
