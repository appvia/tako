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
	"path"

	"github.com/imdario/mergo"
)

// Compile calculates effective configuration with base configuration
// extended/overidden by environments specific configuration
func Compile(root, buildDir string, envs []string) ([]CompiledConfig, error) {
	baseConfig, err := GetBaseConfig(root)
	if err != nil {
		return nil, err
	}

	var compiled []CompiledConfig
	for _, env := range envs {
		envConfig, err := GetEnvConfig(root, env)
		if err != nil {
			return nil, err
		}

		compilePath := path.Join(root, buildDir)
		compiledConfig, err := compileEnvConfig(compilePath, env, envConfig, baseConfig)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, compiledConfig)
	}

	return compiled, nil
}

func compileEnvConfig(compilePath, env string, envConfig Config, baseConfig *Config) (CompiledConfig, error) {
	mergo.Merge(&envConfig, baseConfig)

	rawConfig, err := envConfig.Bytes()
	if err != nil {
		return CompiledConfig{}, err
	}

	compiledConfigPath := path.Join(compilePath, env, "config-compiled.yaml")
	compiledEnvConfig := CompiledConfig{
		Environment: env,
		Content:     rawConfig,
		File:        compiledConfigPath,
	}
	return compiledEnvConfig, nil
}
