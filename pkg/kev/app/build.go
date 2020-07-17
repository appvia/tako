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
	"fmt"
	"path"
	"strings"

	"github.com/appvia/kube-devx/pkg/interpolate"
	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
)

// GetBuildInfo gets a flat list for a build's config pairs (both app and overrides).
func (def *Definition) GetBuildInfo() []ConfigTuple {
	var out []ConfigTuple
	out = append(out, def.Build.Base)
	for _, tuple := range def.Build.Overrides {
		out = append(out, tuple)
	}
	return out
}

// GetMappedBuildInfo gets a map for a build's config pairs.
// If overrides have been built then the map will only include overrides.
// If no overrides have been built then the map will only include app config pairs.
func (def *Definition) GetMappedBuildInfo() map[string]ConfigTuple {
	out := make(map[string]ConfigTuple)
	if def.HasBuiltOverrides() {
		out = def.GetOverridesBuildInfo()
	} else {
		out[""] = def.GetAppBuildInfo()
	}
	return out
}

// HasBuiltOverrides informs whether the Build contains overrides.
func (def *Definition) HasBuiltOverrides() bool {
	return len(def.Build.Overrides) > 0
}

// GetAppBuildInfo returns the base app build info, used for build display and manifest render of app only.
func (def *Definition) GetAppBuildInfo() ConfigTuple {
	return def.Build.Base
}

// GetOverridesBuildInfo returns the overrides build info, used for build display and manifest render of envs only.
func (def *Definition) GetOverridesBuildInfo() map[string]ConfigTuple {
	return def.Build.Overrides
}

// DoBuild builds an app definition manifest
func (def *Definition) DoBuild() error {
	def.Build = BuildConfig{}

	if err := def.buildBase(); err != nil {
		return err
	}

	if err := def.buildOverrides(); err != nil {
		return err
	}

	return nil
}

func (def *Definition) buildBase() error {
	target := interpolate.
		NewTarget().
		Content(def.Base.Compose.Content).
		Resolver(interpolate.NewJsonPathResolver())

	source, err := yaml.YAMLToJSON(def.Base.Config.Content)
	if err != nil {
		return err
	}

	interpolated, err := target.Interpolate(source, quoteCPU)
	if err != nil {
		return err
	}

	def.Build.Base = ConfigTuple{
		Compose: FileConfig{
			Content: interpolated,
			File:    path.Join(baseDir, workDir, buildDir, composeBuildFile),
		},
		Config: FileConfig{
			Content: def.Base.Config.Content,
			File:    path.Join(baseDir, workDir, buildDir, configBuildFile),
		},
	}

	return nil
}

func (def *Definition) buildOverrides() error {
	def.Build.Overrides = map[string]ConfigTuple{}

	for override, _ := range def.Overrides {
		compiledConfig, err := compileConfig(override, def.Overrides[override], def.Base.Config)
		if err != nil {
			return err
		}

		interpolatedCompose, err := interpolateComposeOverride(override, def.Base.Compose, compiledConfig)
		if err != nil {
			return err
		}

		def.Build.Overrides[override] = ConfigTuple{
			Compose: interpolatedCompose,
			Config:  compiledConfig,
		}
	}

	return nil
}

// compileConfig calculates effective configuration for given environment.
// i.e. a base configuration extended/overridden by environment specific configuration.
func compileConfig(override string, overrideConfig, base FileConfig) (FileConfig, error) {
	baseConfig, err := config.Unmarshal(base.Content)
	if err != nil {
		return FileConfig{}, err
	}

	config, err := config.Unmarshal(overrideConfig.Content)
	if err != nil {
		return FileConfig{}, err
	}

	err = mergo.Merge(config, *baseConfig)
	if err != nil {
		return FileConfig{}, err
	}

	tempMergePatchToBeRemovedAfterMergoRelease0_3_10(config, *baseConfig)

	configContent, err := config.Bytes()
	if err != nil {
		return FileConfig{}, err
	}

	return FileConfig{
		Content: configContent,
		File:    path.Join(baseDir, workDir, buildDir, override, configBuildFile),
	}, nil
}

// @todo remove this - See https://github.com/imdario/mergo/issues/131
// Addresses (workload, service.workload).LivenessProbeDisable
func tempMergePatchToBeRemovedAfterMergoRelease0_3_10(dst *config.Config, src config.Config) {
	dst.Workload.LivenessProbeDisable = src.Workload.LivenessProbeDisable
	for key, c := range dst.Components {
		c.Workload.LivenessProbeDisable = src.Components[key].Workload.LivenessProbeDisable
		dst.Components[key] = c
	}
}

// interpolateComposeOverride interpolates the base compose.yaml with compiled config.yaml for given environment.
func interpolateComposeOverride(override string, baseCompose, overrideConfig FileConfig) (FileConfig, error) {
	target := interpolate.
		NewTarget().
		Content(baseCompose.Content).
		Resolver(interpolate.NewJsonPathResolver())

	source, err := yaml.YAMLToJSON(overrideConfig.Content)
	if err != nil {
		return FileConfig{}, err
	}

	envComposeContent, err := target.Interpolate(source, quoteCPU)
	if err != nil {
		return FileConfig{}, err
	}

	return FileConfig{
		Content: envComposeContent,
		File:    path.Join(baseDir, workDir, buildDir, override, composeBuildFile),
	}, nil
}

// quoteCPU formats cpu values to quoted values. E.g. "<CPU_VALUE>"
var quoteCPU interpolate.Formatter = func(key string, value []byte) []byte {
	hasCpu := strings.Contains(strings.ToLower(key), "cpu")
	if hasCpu {
		return []byte(fmt.Sprintf("\"%s\"", value))
	}
	return value
}
