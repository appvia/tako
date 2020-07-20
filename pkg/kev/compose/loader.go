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

package compose

import (
	"io/ioutil"

	"github.com/compose-spec/compose-go/cli"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/goccy/go-yaml"
)

// LoadAndPrepVersionedProject loads and parses a set of input compose files and returns a VersionedProject object
func LoadAndPrepVersionedProject(paths []string) (*VersionedProject, error) {
	composeProject, err := LoadProject(paths)
	if err != nil {
		return nil, err
	}
	version, err := getComposeVersion(paths[0])
	if err != nil {
		return nil, err
	}

	project := &VersionedProject{version, composeProject}

	transforms := []Transform{
		AugmentOrAddDeploy,
		HealthCheckBase,
		ExternaliseSecrets,
		ExternaliseConfigs,
	}
	for _, t := range transforms {
		if err := t(project); err != nil {
			return nil, err
		}
	}

	return project, nil
}

// LoadProject loads and parses a set of input compose files and returns a compose VersionedProject object
func LoadProject(paths []string) (*compose.Project, error) {
	projectOptions, err := cli.ProjectOptions{
		ConfigPaths: paths,
	}.
		WithOsEnv().
		WithDotEnv()

	if err != nil {
		return nil, err
	}

	project, err := cli.ProjectFromOptions(&projectOptions)
	if err != nil {
		return nil, err
	}

	for i := range project.Services {
		project.Services[i].EnvFile = nil
	}
	return project, nil
}

// getComposeVersion extracts version from compose file and returns a string
func getComposeVersion(file string) (string, error) {
	type ComposeVersion struct {
		Version string `json:"version"` // This affects YAML as well
	}

	version := ComposeVersion{}

	compose, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	if err = yaml.Unmarshal(compose, &version); err != nil {
		return "", err
	}

	return version.Version, nil
}
