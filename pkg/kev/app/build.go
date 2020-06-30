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

// GetBuildInfo get the latest info as map of overrides and config pairs.
// The build's base config is added under the key "base".
func (def *Definition) GetBuildInfo() map[string]ConfigPair {
	out := map[string]ConfigPair{}
	out["base"] = def.Build.Base
	for override, pair := range def.Build.Overrides {
		out[override] = pair
	}
	return out
}

// DoBuild builds an app definition manifest
func (def *Definition) DoBuild(buildDir string) error {
	def.Build = BuildConfig{}

	if err := def.buildBase(buildDir); err != nil {
		return err
	}

	if err := def.buildOverrides(buildDir); err != nil {
		return err
	}

	return nil
}

func (def *Definition) buildBase(buildDir string) error {
	target := interpolate.
		NewTarget().
		Content(def.Base.Compose.Content).
		Resolver(interpolate.NewJsonPathResolver())

	source, err := yaml.YAMLToJSON(def.Base.Config.Content)
	if err != nil {
		return err
	}

	interpolated, err := target.Interpolate(source)
	if err != nil {
		return err
	}

	def.Build.Base = ConfigPair{
		Compose: FileConfig{
			Content: interpolated,
			File:    path.Join(buildDir, ComposeBuildFile),
		},
		Config: FileConfig{
			Content: def.Base.Config.Content,
			File:    path.Join(buildDir, ConfigBuildFile),
		},
	}

	return nil
}

func (def *Definition) buildOverrides(buildDir string) error {
	def.Build.Overrides = map[string]ConfigPair{}

	for override, _ := range def.Overrides {
		compiledConfig, err := CompileConfig(buildDir, override, def.Overrides[override], def.Base.Config)
		if err != nil {
			return err
		}

		interpolatedCompose, err := InterpolateComposeOverride(buildDir, override, compiledConfig, def.Base.Compose)
		if err != nil {
			return err
		}

		def.Build.Overrides[override] = ConfigPair{
			Compose: interpolatedCompose,
			Config:  compiledConfig,
		}
	}

	return nil
}

// CompileConfig calculates effective configuration for given environment.
// i.e. a base configuration extended/overridden by environment specific configuration.
func CompileConfig(buildRoot, override string, overrideConfig, base FileConfig) (FileConfig, error) {
	baseConfig, err := config.Unmarshal(base.Content)
	if err != nil {
		return FileConfig{}, err
	}

	envConfig, err := config.Unmarshal(overrideConfig.Content)
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
		Content: envConfigContent,
		File:    path.Join(buildRoot, override, ConfigBuildFile),
	}, nil
}

// InterpolateComposeOverride interpolates the base compose.yaml with compiled config.yaml for given environment.
func InterpolateComposeOverride(buildDir, override string, overrideConfig, baseCompose FileConfig) (FileConfig, error) {
	target := interpolate.
		NewTarget().
		Content(baseCompose.Content).
		Resolver(interpolate.NewJsonPathResolver())

	source, err := yaml.YAMLToJSON(overrideConfig.Content)
	if err != nil {
		return FileConfig{}, err
	}

	envComposeContent, err := target.Interpolate(source)
	if err != nil {
		return FileConfig{}, err
	}

	return FileConfig{
		Content: envComposeContent,
		File:    path.Join(buildDir, override, ComposeBuildFile),
	}, nil
}
