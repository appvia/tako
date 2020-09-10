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
	"path/filepath"

	"github.com/appvia/kube-devx/pkg/kev/converter"
	"github.com/appvia/kube-devx/pkg/kev/log"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
)

const (
	// ManifestName main application manifest
	ManifestName = "kev.yaml"
	defaultEnv   = "dev"
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

// PrepareForSkaffold initialises a skaffold manifest for kev project.
// It'll also set the Skaffold field in kev manifest with skaffold file path passed as argument.
func PrepareForSkaffold(manifest *Manifest, skaffoldPath string, envs []string) (*SkaffoldManifest, error) {
	s, err := NewSkaffoldManifest(envs)
	if err != nil {
		return nil, err
	}

	manifest.Skaffold = skaffoldPath

	return s, nil
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

	filteredEnvs, err := manifest.GetEnvironments(envs)
	if err != nil {
		return errors.Wrap(err, "Unable to render")
	}

	rendered := map[string][]byte{}
	projects := map[string]*composego.Project{}
	files := map[string][]string{}
	sourcesFiles := manifest.GetSourcesFiles()

	for _, env := range filteredEnvs {
		p, err := manifest.MergeEnvIntoSources(env)
		if err != nil {
			return errors.Wrap(err, "Couldn't calculate compose project representation")
		}
		projects[env.Name] = p.Project
		files[env.Name] = append(sourcesFiles, env.File)
	}

	c := converter.Factory(format)
	outputPaths, err := c.Render(singleFile, dir, manifest.GetWorkingDir(), projects, files, rendered)
	if err != nil {
		log.Errorf("Couldn't render manifests")
		return err
	}

	if len(manifest.Skaffold) > 0 {
		if err := UpdateSkaffoldProfiles(filepath.Join(workDir, manifest.Skaffold), outputPaths); err != nil {
			log.Errorf("Couldn't update skaffold.yaml profiles")
			return err
		}
	}

	return nil
}
