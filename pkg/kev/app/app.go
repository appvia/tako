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
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/appvia/kube-devx/pkg/kev"
)

const (
	// ComposeFile base compose file name
	ComposeFile = "compose.yaml"

	// ComposeBuildFile build time compose file name
	ComposeBuildFile = "compose.build.yaml"

	// ConfigFile config file name (base)
	ConfigFile = "config.yaml"

	// ConfigBuildFile build time config file name
	ConfigBuildFile = "config.build.yaml"

	// Base labels the app's base Compose and Config files during init and build.
	// These files are the basis for user defined overrides that map to app environments.
	// This is a reserved name.
	Base = "kev-base"
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
// Note: If kev.WorkDir is defined base compose file will be placed
// under this subdirectory to declutter config space.
func loadBase(pair *ConfigTuple) error {
	composePath := path.Join(kev.BaseDir, kev.WorkDir, ComposeFile)
	configPath := path.Join(kev.BaseDir, ConfigFile)
	if err := configPair(composePath, configPath, pair); err != nil {
		return err
	}
	return nil
}

// loadBuildConfigPair load built config pair for a given environment.
// To get the base build config pair, pass env as an empty string!
func loadBuildConfigPair(env string, pair *ConfigTuple) error {
	envDir := path.Join(kev.BaseDir, kev.WorkDir, kev.BuildDir, env)
	composePath := path.Join(envDir, ComposeBuildFile)
	configPath := path.Join(envDir, ConfigBuildFile)
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
	if !WasBuilt() {
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
	configPath := path.Join(kev.BaseDir, env, ConfigFile)
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return FileConfig{}, err
	}

	return FileConfig{
		Content: content,
		File:    configPath,
	}, nil
}

// WasBuilt checks whether the app has previously been built
func WasBuilt() bool {
	buildDir := path.Join(kev.BaseDir, kev.WorkDir, kev.BuildDir)
	composeBuildExists := fileExists(path.Join(buildDir, ComposeBuildFile))
	configBuildExists := fileExists(path.Join(buildDir, ConfigBuildFile))
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

// ValidateHasEnvs checks whether supplied environments exist or not.
func ValidateHasEnvs(candidates []string) error {
	envs, err := GetEnvs()
	if err != nil {
		return err
	}

	sort.Strings(envs)
	var invalid []string

	for _, c := range candidates {
		i := sort.SearchStrings(envs, c)
		valid := i < len(envs) && envs[i] == c
		if !valid {
			invalid = append(invalid, c)
		}
	}

	if len(invalid) > 0 {
		return fmt.Errorf("cannot find environment(s): %s", strings.Join(invalid, ", "))
	}

	return nil
}

// GetEnvs returns a string slice of all app environments
func GetEnvs() ([]string, error) {
	var envs []string

	files, err := ioutil.ReadDir(kev.BaseDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			envs = append(envs, file.Name())
		}
	}

	return envs, nil
}
