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

	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/appvia/kube-devx/pkg/kev/utils"
	yaml3 "gopkg.in/yaml.v3"
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
