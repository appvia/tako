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

	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/appvia/kube-devx/pkg/kev/utils"
	yaml3 "gopkg.in/yaml.v3"
)

const (
	ComposeFile      = "compose.yaml"
	ComposeBuildFile = "compose.build.yaml"
	ConfigFile       = "config.yaml"
	ConfigBuildFile  = "config-compiled.yaml"
)

// Init creates a new app definition manifest
// based on a compose.yaml, inferred app config and required environments.
func Init(root string, compose []byte, baseConfig *config.Config, envs []string) (*Definition, error) {
	composePath := path.Join(root, ComposeFile)
	configPath := path.Join(root, ConfigFile)

	envConfigs, err := createEnvData(envs, root, baseConfig)
	if err != nil {
		return nil, err
	}

	configData, err := baseConfig.Bytes()
	if err != nil {
		return nil, err
	}

	return &Definition{
		BaseCompose: FileConfig{Content: compose, File: composePath},
		Config:      FileConfig{Content: configData, File: configPath},
		Envs:        envConfigs,
	}, nil
}

// GetDefinition returns the current app definition manifest
func GetDefinition(root string, envs []string) (*Definition, error) {
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
		Envs: envConfigs,
	}, nil
}

// GetEnvConfig, retrieves the config.yaml for a specified environment.
func GetEnvConfig(root string, env string) (FileConfig, error) {
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

// ValidateHasEnvs, checks whether supplied environments exist or not.
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

// GetEnvs, returns a string slice of all app environments
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

func createEnvData(envs []string, appDir string, baseConfig *config.Config) ([]FileConfig, error) {
	envConfig := &EnvConfig{
		Workload: &yaml3.Node{
			Kind:        yaml3.MappingNode,
			LineComment: "Override global workload settings here.",
		},
		Service: &yaml3.Node{
			Kind:        yaml3.MappingNode,
			LineComment: "Override global service settings here.",
		},
		Volumes: &yaml3.Node{
			Kind:        yaml3.MappingNode,
			LineComment: "Override global volumes settings here.",
		},
		Components: make(map[string]*yaml3.Node),
	}

	for key := range baseConfig.Components {
		envConfig.Components[key] = &yaml3.Node{
			Kind:        yaml3.MappingNode,
			LineComment: fmt.Sprintf("Override the %s service settings here.", key),
		}
	}

	out, err := utils.MarshallAndFormat(&envConfig, 2)
	if err != nil {
		return nil, err
	}

	var envConfigs []FileConfig
	for _, env := range envs {
		envConfigs = append(envConfigs, FileConfig{
			Environment: env,
			Content:     out,
			File:        path.Join(appDir, env, "config.yaml"),
		})
	}

	return envConfigs, nil
}
