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
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/appvia/kube-devx/pkg/kev/converter"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	// ManifestName main application manifest
	ManifestName = "kev.yaml"

	defaultEnv         = "dev"
	configFileTemplate = "docker-compose.kev.%s.yaml"
)

// Init initialises a kev manifest including source compose files and environments.
// A default environment will be allocated if no environments were provided.
func Init(composeSources, envs []string) (*Manifest, error) {
	m, err := NewManifest(composeSources).
		ExtractLabels()
	if err != nil {
		return nil, err
	}

	return m.MintEnvironments(envs), nil
}

// Render renders k8s manifests for a kev app. It returns an app definition with rendered manifest info
func Render(format string, singleFile bool, dir string, envs []string) error {
	// @todo filter specified envs, or all if none provided
	workDir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "Couldn't get working directory")
	}

	manifest, err := LoadManifest(workDir)
	if err != nil {
		return errors.Wrap(err, "Unable to load app manifest")
	}

	rendered := map[string][]byte{}
	projects := map[string]*composego.Project{}
	files := map[string][]string{}

	for _, env := range manifest.Environments {
		inputFiles := append(manifest.Sources, env.File)
		if p, err := RawProjectFromSources(inputFiles); err != nil {
			return errors.Wrap(err, "Couldn't calculate compose project representation")
		} else {
			projects[env.Name] = p
			files[env.Name] = inputFiles
		}
	}

	c := converter.Factory(format)
	if err := c.Render(singleFile, dir, manifest.GetWorkingDir(), projects, files, rendered); err != nil {
		return err
	}

	return nil
}

// NewManifest returns a new Manifest struct
func NewManifest(sources []string) *Manifest {
	return &Manifest{
		Sources: sources,
	}
}

// LoadManifest returns application manifests
func LoadManifest(workingDir string) (*Manifest, error) {
	fmt.Println("working dir:", workingDir)
	data, err := ioutil.ReadFile(path.Join(workingDir, ManifestName))
	if err != nil {
		return nil, err
	}
	var m *Manifest
	return m, yaml.Unmarshal(data, &m)
}
