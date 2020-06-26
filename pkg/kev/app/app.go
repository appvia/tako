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

// GetDefinition returns the current app definition manifest
func GetDefinition(root, buildDir string, envs []string) (*Definition, error) {
	composePath := path.Join(root, ComposeFile)
	baseCompose, err := ioutil.ReadFile(composePath)
	if err != nil {
		return nil, err
	}

	configPath := path.Join(root, ConfigFile)
	baseConfig, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var envConfigs []FileConfig
	for _, env := range envs {
		envConfig, err := GetEnvConfig(root, env)
		if err != nil {
			return nil, err
		}
		envConfigs = append(envConfigs, envConfig)
	}

	var buildConfig BuildConfig
	for _, env := range envs {
		// get compiled configuration and interpolated compose for a given environment
		compiled, interpolated, err := GetBuildConfig(root, buildDir, env)
		if err != nil {
			return nil, err
		}
		buildConfig.Compiled = append(buildConfig.Compiled, compiled)
		buildConfig.Interpolated = append(buildConfig.Interpolated, interpolated)
	}

	return &Definition{
		BaseCompose: FileConfig{
			Environment: "base",
			Content:     baseCompose,
			File:        composePath,
		},
		Config: FileConfig{
			Environment: "base",
			Content:     baseConfig,
			File:        configPath,
		},
		Envs:  envConfigs,
		Build: buildConfig,
	}, nil
}

// GetEnvConfig retrieves the config.yaml for a specified environment.
func GetEnvConfig(root, env string) (FileConfig, error) {
	configPath := path.Join(root, env, ConfigFile)
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return FileConfig{}, err
	}
	return FileConfig{
		Environment: env,
		Content:     content,
		File:        configPath,
	}, nil
}

// GetBuildConfig retrieves the compiled config.build.yaml & interpolated compose.build.yaml for a specified environment.
func GetBuildConfig(root, buildDir, env string) (FileConfig, FileConfig, error) {
	configPath := path.Join(root, buildDir, env, ConfigBuildFile)
	configContent, err := ioutil.ReadFile(configPath)
	if err != nil {
		return FileConfig{}, FileConfig{}, err
	}

	composePath := path.Join(root, buildDir, env, ComposeBuildFile)
	composeContent, err := ioutil.ReadFile(composePath)
	if err != nil {
		return FileConfig{}, FileConfig{}, err
	}

	return FileConfig{
			Environment: env,
			Content:     configContent,
			File:        configPath,
		}, FileConfig{
			Environment: env,
			Content:     composeContent,
			File:        composePath,
		}, nil
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
