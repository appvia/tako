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

	appDef.Build = make(map[string]BuildConfig)

	// iterate through app defined environments
	for name, envConfig := range appDef.Envs {
		// get compiled config for current environment
		compiledEnvConfig, err := CompileConfig(path, envConfig, appDef.BaseConfig)
		if err != nil {
			return nil, err
		}

		// interpolate base compose with compiled env config params
		interpolatedCompose, err := InterpolateCompose(path, compiledEnvConfig, appDef.BaseCompose)
		if err != nil {
			return nil, err
		}

		bc := BuildConfig{
			ConfigFile:  compiledEnvConfig,
			ComposeFile: interpolatedCompose,
		}
		// app definition build information is keyed by environment name
		appDef.Build[name] = bc
	}

	return appDef, nil
}

// CompileConfig calculates effective configuration for given environment.
// i.e. a base configuration extended/overridden by environment specific configuration.
func CompileConfig(buildRoot string, env, base FileConfig) (FileConfig, error) {
	baseConfig, err := config.Marshal(base.Content)
	if err != nil {
		return FileConfig{}, err
	}

	envConfig, err := config.Marshal(env.Content)
	if err != nil {
		return FileConfig{}, err
	}

	err = mergo.Merge(envConfig, *baseConfig)
	if err != nil {
		return FileConfig{}, err
	}

	envConfigContent, err := envConfig.Bytes()
	if err != nil {
		return FileConfig{}, err
	}

	return FileConfig{
		Environment: env.Environment,
		Content:     envConfigContent,
		File:        path.Join(buildRoot, env.Environment, ConfigBuildFile),
	}, nil
}

// InterpolateCompose interpolates the base compose.yaml with compiled config.yaml for given environment.
func InterpolateCompose(buildRoot string, envConfig, baseCompose FileConfig) (FileConfig, error) {
	target := interpolate.
		NewTarget().
		Content(baseCompose.Content).
		Resolver(interpolate.NewJsonPathResolver())

	source, err := yaml.YAMLToJSON(envConfig.Content)
	if err != nil {
		return FileConfig{}, err
	}

	envComposeContent, err := target.Interpolate(source)
	if err != nil {
		return FileConfig{}, err
	}

	return FileConfig{
		Environment: envConfig.Environment,
		Content:     envComposeContent,
		File:        path.Join(buildRoot, envConfig.Environment, ComposeBuildFile),
	}, nil
}
