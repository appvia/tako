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
	"io/ioutil"
	"os"
	"path"
)

// LoadDefinition returns the current app definition manifest
func LoadDefinition(envs []string) (*Definition, error) {
	var def = &Definition{}

	if err := loadBase(&def.Base); err != nil {
		return nil, err
	}

	if err := loadOverrides(envs, def); err != nil {
		return nil, err
	}

	if err := loadBuildIfAvailable(envs, def); err != nil {
		return nil, err
	}

	return def, nil
}

// loadBase loads base config pair
// Note: If kev.workDir is defined base compose file will be placed
// under this subdirectory to declutter config space.
func loadBase(pair *ConfigTuple) error {
	composePath := path.Join(baseDir, workDir, composeFile)
	configPath := path.Join(baseDir, configFile)
	if err := configPair(composePath, configPath, pair); err != nil {
		return err
	}
	return nil
}

// loadBuildConfigPair load built config pair for a given environment.
// To get the base build config pair, pass env as an empty string!
func loadBuildConfigPair(env string, pair *ConfigTuple) error {
	envDir := path.Join(baseDir, workDir, buildDir, env)
	composePath := path.Join(envDir, composeBuildFile)
	configPath := path.Join(envDir, configBuildFile)
	if err := configPair(composePath, configPath, pair); err != nil {
		return err
	}
	return nil
}

// configPair returns a ConfigTuple for provided compose and config paths
func configPair(composePath, configPath string, pair *ConfigTuple) error {
	compose, err := ioutil.ReadFile(composePath)
	if err != nil {
		return err
	}

	config, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	pair.Compose = FileConfig{
		Content: compose,
		File:    composePath,
	}
	pair.Config = FileConfig{
		Content: config,
		File:    configPath,
	}
	return nil
}

func loadOverrides(envs []string, def *Definition) error {
	overrides := make(map[string]FileConfig)
	for _, env := range envs {
		config, err := loadOverrideConfig(env)
		if err != nil {
			return err
		}
		overrides[env] = config
	}
	def.Overrides = overrides
	return nil
}

func loadBuildIfAvailable(envs []string, def *Definition) error {
	def.Build = BuildConfig{}
	if !built() {
		return nil
	}

	var base ConfigTuple
	if err := loadBuildConfigPair("", &base); err != nil {
		return err
	}
	def.Build.Base = base

	overrides := map[string]ConfigTuple{}
	for _, env := range envs {
		var pair ConfigTuple
		if err := loadBuildConfigPair(env, &pair); err != nil {
			return err
		}
		overrides[env] = pair
	}
	def.Build.Overrides = overrides

	return nil
}

// loadOverrideConfig retrieves the config.yaml for a specified environment.
func loadOverrideConfig(env string) (FileConfig, error) {
	configPath := path.Join(baseDir, env, configFile)
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return FileConfig{}, err
	}

	return FileConfig{
		Content: content,
		File:    configPath,
	}, nil
}

// built checks whether the app has previously been built
func built() bool {
	buildDir := path.Join(baseDir, workDir, buildDir)
	composeBuildExists := fileExists(path.Join(buildDir, composeBuildFile))
	configBuildExists := fileExists(path.Join(buildDir, configBuildFile))
	return composeBuildExists && configBuildExists
}

func fileExists(filePath string) bool {
	if _, err := os.Stat(filePath); err == nil {
		// exists
		return true
	} else if os.IsNotExist(err) {
		// does not exist
		return false
	} else {
		return false
	}
}
