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

	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/appvia/kube-devx/pkg/kev/transform"
	"github.com/compose-spec/compose-go/loader"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/goccy/go-yaml"
)

// NewApp creates a new Definition using
// provided name, docker compose files and app root
func NewApp(root, name string, composeFiles, envs []string) (*app.Definition, error) {
	baseCompose, err := loadAndParse(composeFiles)
	if err != nil {
		return nil, err
	}

	composeData, err := yaml.Marshal(baseCompose)
	if err != nil {
		return nil, err
	}

	composeData, err = transform.DeployWithDefaults(composeData)
	if err != nil {
		return nil, err
	}

	composeData, err = transform.HealthCheckBase(composeData)
	if err != nil {
		return nil, err
	}

	composeData, err = transform.ExternaliseSecrets(composeData)
	if err != nil {
		return nil, err
	}

	composeData, err = transform.ExternaliseConfigs(composeData)
	if err != nil {
		return nil, err
	}

	inferred, err := config.Infer(composeData)
	if err != nil {
		return nil, err
	}

	return app.NewDefinition(root, name, inferred.ComposeWithPlaceholders, inferred.AppConfig, envs)
}

func loadAndParse(paths []string) (*compose.Config, error) {
	var configFiles []compose.ConfigFile

	for _, p := range paths {
		b, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, err
		}

		parsed, err := loader.ParseYAML(b)
		if err != nil {
			return nil, err
		}

		configFiles = append(configFiles, compose.ConfigFile{Filename: p, Config: parsed})
	}

	return loader.Load(compose.ConfigDetails{
		WorkingDir:  path.Dir(paths[0]),
		ConfigFiles: configFiles,
	})
}
