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
)

const (
	// ComposeFile base compose file name
	ComposeFile = "compose.yaml"

	// ComposeBuildFile build time compose file name
	ComposeBuildFile = "compose.build.yaml"

	// ConfigFile config file name
	ConfigFile = "config.yaml"

	// ConfigBuildFile build time config file name
	ConfigBuildFile = "config.build.yaml"
)

// LoadDefinition returns the current app definition manifest
func LoadDefinition(root, buildDir string, envs []string) (*Definition, error) {
	var def = &Definition{}

	if err := loadBase(root, def); err != nil {
		return nil, err
	}

	if err := loadOverrides(root, envs, def); err != nil {
		return nil, err
	}

	if err := loadBuildIfAvailable(path.Join(root, buildDir), envs, def); err != nil {
		return nil, err
	}

	return def, nil
}

func loadBase(dir string, def *Definition) error {
	if err := loadBaseConfigPair(dir, ComposeFile, ConfigFile, &def.Base); err != nil {
		return err
	}
	return nil
}

func loadBaseConfigPair(dir, composeFile, configFile string, pair *ConfigPair) error {
	composePath := path.Join(dir, composeFile)
	compose, err := ioutil.ReadFile(composePath)
	if err != nil {
		return err
	}

	configPath := path.Join(dir, configFile)
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

func loadOverrides(dir string, envs []string, def *Definition) error {
	overrides := make(map[string]FileConfig)
	for _, env := range envs {
		config, err := loadOverrideConfig(dir, env)
		if err != nil {
			return err
		}
		overrides[env] = config
	}
	def.Overrides = overrides
	return nil
}

func loadBuildIfAvailable(dir string, envs []string, def *Definition) error {
	def.Build = BuildConfig{}
	if !WasBuilt(dir) {
		return nil
	}

	var base ConfigPair
	if err := loadBaseConfigPair(dir, ComposeBuildFile, ConfigBuildFile, &base); err != nil {
		return err
	}
	def.Build.Base = base

	overrides := map[string]ConfigPair{}
	for _, env := range envs {
		var pair ConfigPair
		if err := loadBaseConfigPair(path.Join(dir, env), ComposeBuildFile, ConfigBuildFile, &pair); err != nil {
			return err
		}
		overrides[env] = pair
	}
	def.Build.Overrides = overrides

	return nil
}

// loadOverrideConfig retrieves the config.yaml for a specified environment.
func loadOverrideConfig(root, env string) (FileConfig, error) {
	configPath := path.Join(root, env, ConfigFile)
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
func WasBuilt(buildDir string) bool {
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
func ValidateHasEnvs(root string, candidates []string) error {
	envs, err := GetEnvs(root)
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
func GetEnvs(root string) ([]string, error) {
	var envs []string

	files, err := ioutil.ReadDir(root)
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
