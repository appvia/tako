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

	"github.com/appvia/kube-devx/pkg/interpolate"
	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
)

// Build builds the app based on an app definition manifest
func Build(path string, appDef *Definition) (*Definition, error) {
	appDef.Build = BuildConfig{}

	compiled, err := CompileConfig(path, appDef)
	if err != nil {
		return nil, err
	}

	appDef.Build.Compiled = append(appDef.Build.Compiled, compiled...)

	interpolated, err := InterpolateUsingConfig(appDef)
	if err != nil {
		return nil, err
	}

	appDef.Build.Interpolated = append(appDef.Build.Interpolated, interpolated...)
	return appDef, nil
}

// CompileConfig calculates effective configuration with base configuration
// extended/overridden by environments specific configuration
func CompileConfig(buildRoot string, appDef *Definition) ([]FileConfig, error) {
	var compiled []FileConfig

	baseConfig, err := config.Marshal(appDef.Config.Content)
	if err != nil {
		return nil, err
	}

	rawBaseConfig, err := baseConfig.Bytes()
	if err != nil {
		return nil, err
	}

	compiled = append(compiled, FileConfig{
		Environment: "base",
		Content:     rawBaseConfig,
		File:        path.Join(buildRoot, ConfigBuildFile),
	})

	for _, env := range appDef.Envs {
		envConfig, err := config.Marshal(env.Content)
		if err != nil {
			return nil, err
		}

		compiledConfig, err := compileEnvConfig(buildRoot, env.Environment, *envConfig, baseConfig)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, compiledConfig)
	}

	return compiled, nil
}

func compileEnvConfig(compilePath, env string, envConfig config.Config, baseConfig *config.Config) (FileConfig, error) {
	err := mergo.Merge(&envConfig, baseConfig)
	if err != nil {
		return FileConfig{}, err
	}

	rawConfig, err := envConfig.Bytes()
	if err != nil {
		return FileConfig{}, err
	}

	compiledConfigPath := path.Join(compilePath, env, ConfigBuildFile)
	compiledEnvConfig := FileConfig{
		Environment: env,
		Content:     rawConfig,
		File:        compiledConfigPath,
	}
	return compiledEnvConfig, nil
}

// InterpolateUsingConfig interpolates the base compose.yaml creating different
// variation for every compiled config.yaml per environment.
func InterpolateUsingConfig(appDef *Definition) ([]FileConfig, error) {
	target := interpolate.
		NewTarget().
		Content(appDef.BaseCompose.Content).
		Resolver(interpolate.NewJsonPathResolver())

	var interpolated []FileConfig
	for _, compiled := range appDef.Build.Compiled {
		source, err := yaml.YAMLToJSON(compiled.Content)
		if err != nil {
			return nil, err
		}

		content, err := target.Interpolate(source)
		if err != nil {
			return nil, err
		}

		interpolated = append(interpolated, FileConfig{
			Environment: compiled.Environment,
			Content:     content,
			File:        path.Join(path.Dir(compiled.File), ComposeBuildFile),
		})
	}
	return interpolated, nil
}
