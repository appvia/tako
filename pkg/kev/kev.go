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

package kev

import (
	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/appvia/kube-devx/pkg/kev/compose"
	"github.com/appvia/kube-devx/pkg/kev/config"
)

func InitApp(composeFiles, envs []string) (*app.Definition, error) {
	versionedProject, err := compose.LoadAndPrepVersionedProject(composeFiles)
	if err != nil {
		return nil, err
	}

	inferred, err := config.Infer(versionedProject)
	if err != nil {
		return nil, err
	}

	return app.Init(inferred.ComposeWithPlaceholders, inferred.BaseConfig, envs)
}

func BuildApp(envs []string) (*app.Definition, error) {
	envs, err := app.ValidateEnvsOrGetAll(envs)
	if err != nil {
		return nil, err
	}

	def, err := app.LoadDefinition(envs)
	if err != nil {
		return nil, err
	}

	if err := def.DoBuild(); err != nil {
		return nil, err
	}

	return def, nil
}
