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
	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/appvia/kube-devx/pkg/kev/transform"
	"github.com/compose-spec/compose-go/cli"
	compose "github.com/compose-spec/compose-go/types"
)

// NewApp creates a new Definition using app root and docker compose files
func NewApp(root string, composeFiles, envs []string) (*app.Definition, error) {
	baseCompose, err := loadAndParse(composeFiles)
	if err != nil {
		return nil, err
	}

	transforms := []transform.Transform{
		transform.AugmentOrAddDeploy,
		transform.HealthCheckBase,
		transform.ExternaliseSecrets,
		transform.ExternaliseConfigs,
	}

	for _, t := range transforms {
		if err := t(baseCompose); err != nil {
			return nil, err
		}
	}

	inferred, err := config.Infer(baseCompose)
	if err != nil {
		return nil, err
	}

	return app.Init(root, inferred.ComposeWithPlaceholders, inferred.BaseConfig, envs)
}

func loadAndParse(paths []string) (*compose.Project, error) {
	projectOptions, err := cli.ProjectOptions{
		ConfigPaths: paths,
	}.
		WithOsEnv().
		WithDotEnv()

	if err != nil {
		return nil, err
	}

	return cli.ProjectFromOptions(&projectOptions)
}
