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
	"io"
	"os"

	"github.com/appvia/kube-devx/pkg/kev/converter"
	"github.com/appvia/kube-devx/pkg/kev/log"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
)

const (
	// ManifestName main application manifest
	ManifestName = "kev.yaml"

	defaultEnv         = "dev"
	configFileTemplate = "docker-compose.kev.%s.yaml"
)

// Init initialises a kev manifest including source compose files and environments.
// A default environment will be allocated if no environments were provided.
func Init(composeSources, envs []string, workingDir string) (*Manifest, error) {
	m, err := NewManifest(composeSources, workingDir)
	if err != nil {
		return nil, err
	}

	if _, err := m.CalculateSourcesBaseOverlay(); err != nil {
		return nil, err
	}
	return m.MintEnvironments(envs), nil
}

// Reconcile reconciles changes with docker-compose sources against deployment environments.
func Reconcile(workingDir string, reporter io.Writer) (*Manifest, error) {
	m, err := LoadManifest(workingDir)
	if err != nil {
		return nil, err
	}
	if _, err := m.ReconcileConfig(reporter); err != nil {
		return nil, errors.Wrap(err, "Could not reconcile project latest")
	}
	return m, err
}

// Render renders k8s manifests for a kev app. It returns an app definition with rendered manifest info
func Render(format string, singleFile bool, dir string, envs []string) error {
	// @todo filter specified envs, or all if none provided
	workDir, err := os.Getwd()
	if err != nil {
		log.Error("Couldn't get working directory")
		return err
	}

	manifest, err := LoadManifest(workDir)
	if err != nil {
		log.Error("Unable to load app manifest")
		return err
	}

	filteredEnvs, err := manifest.EnvironmentsAsMap(envs)
	if err != nil {
		return errors.Wrap(err, "Unable to render")
	}

	rendered := map[string][]byte{}
	projects := map[string]*composego.Project{}
	files := map[string][]string{}

	for env, file := range filteredEnvs {
		sourcesFiles := manifest.GetSourcesFiles()
		inputFiles := append(sourcesFiles, file)
		p, err := rawProjectFromSources(inputFiles)
		if err != nil {
			return errors.Wrap(err, "Couldn't calculate compose project representation")
		}
		projects[env] = p
		files[env] = inputFiles
	}

	c := converter.Factory(format)
	if err := c.Render(singleFile, dir, manifest.GetWorkingDir(), projects, files, rendered); err != nil {
		log.Errorf("Couldn't render manifests")
		return err
	}

	return nil
}
