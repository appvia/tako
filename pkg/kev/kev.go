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
	"io/ioutil"
	"os"
	"path"

	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/appvia/kube-devx/pkg/kev/compose"
	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/appvia/kube-devx/pkg/kev/converter"
	"github.com/appvia/kube-devx/pkg/kev/yaml"
)

// Init inits a kev app returning app definition
func Init(composeFiles, envs []string) (*app.Definition, error) {
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

// Build builds a kev app. It returns an app definition with the build info
func Build(envs []string) (*app.Definition, error) {
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

// BuildFromDefinition is like Build, but builds a kev app from an already loaded app definition
func BuildFromDefinition(def *app.Definition, envs []string) error {
	envs, err := app.ValidateDefEnvsOrGetAll(def, envs)
	if err != nil {
		return err
	}

	def.ExcludeOtherOverrides(envs)

	if err := def.DoBuild(); err != nil {
		return err
	}

	return nil
}

// Render renders k8s manifests for a kev app. It returns an app definition with rendered manifest info
func Render(format string, singleFile bool, dir string, envs []string) (*app.Definition, error) {
	envs, err := app.ValidateEnvsOrGetAll(envs)
	if err != nil {
		return nil, err
	}

	def, err := app.LoadDefinition(envs)
	if err != nil {
		return nil, err
	}

	c := converter.Factory(format)
	if err := c.Render(singleFile, dir, def); err != nil {
		return nil, err
	}

	return def, nil
}

// RenderFromDefinition is like Render, but renders a kev app from an already loaded app definition
func RenderFromDefinition(def *app.Definition, format string, singleFile bool, dir string, envs []string) error {
	envs, err := app.ValidateDefEnvsOrGetAll(def, envs)
	if err != nil {
		return err
	}

	def.ExcludeOtherOverrides(envs)
	c := converter.Factory(format)
	if err := c.Render(singleFile, dir, def); err != nil {
		return err
	}

	return nil
}

func Overlay(composeFiles []string, configFile string, outFile string) error {
	files := append(composeFiles, configFile)
	project, err := compose.LoadProject(files)
	if err != nil {
		return err
	}

	data, err := yaml.MarshalIndent(project, 2)
	return ioutil.WriteFile(path.Join(outFile), data, os.ModePerm)
}
