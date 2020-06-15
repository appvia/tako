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

package bootstrap

import (
	"io/ioutil"
	"path"

	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/appvia/kube-devx/pkg/kev/transform"
	"github.com/compose-spec/compose-go/loader"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/goccy/go-yaml"
)

// AppDefinition provides details for the app's base compose and config files.
type AppDefinition struct {
	Name        string
	BaseCompose Payload
	Config      Payload
}

// Payload details an app definition Payload, including its Content and recommended file path.
type Payload struct {
	Content  []byte
	FilePath string
}

// NewApp creates a new AppDefinition using
// provided name, docker compose files and app root
func NewApp(root, name string, composeFiles []string) (*AppDefinition, error) {
	compose, err := loadAndParse(composeFiles)
	if err != nil {
		return nil, err
	}

	bytes, err := yaml.Marshal(compose)
	if err != nil {
		return nil, err
	}

	bytes, err = transform.DeployWithDefaults(bytes)
	if err != nil {
		return nil, err
	}

	bytes, err = transform.HealthCheckBase(bytes)
	if err != nil {
		return nil, err
	}

	bytes, err = transform.ExternaliseSecrets(bytes)
	if err != nil {
		return nil, err
	}

	bytes, err = transform.ExternaliseConfigs(bytes)
	if err != nil {
		return nil, err
	}

	// Application Configuration
	appConfig := config.New()

	// Infer configuration parameters from transformed compose
	bytes, err = config.Infer(bytes, appConfig)
	if err != nil {
		return nil, err
	}

	bytes, err = transform.Echo(bytes)
	if err != nil {
		return nil, err
	}

	appDir := path.Join(root, name)
	appBaseComposePath := path.Join(appDir, "compose.yaml")
	appBaseConfigPath := path.Join(appDir, "config.yaml")

	configBytes, err := appConfig.Bytes()
	if err != nil {
		return nil, err
	}

	return &AppDefinition{
		BaseCompose: Payload{
			Content:  bytes,
			FilePath: appBaseComposePath,
		},
		Config: Payload{
			Content:  configBytes,
			FilePath: appBaseConfigPath,
		},
	}, nil

}

func loadAndParse(paths []string) (*compose.Config, error) {
	var configFiles []compose.ConfigFile

	for _, path := range paths {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		config, err := loader.ParseYAML(b)
		if err != nil {
			return nil, err
		}

		configFiles = append(configFiles, compose.ConfigFile{Filename: path, Config: config})
	}

	return loader.Load(compose.ConfigDetails{
		WorkingDir:  path.Dir(paths[0]),
		ConfigFiles: configFiles,
	})
}
