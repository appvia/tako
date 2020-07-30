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

	"github.com/compose-spec/compose-go/cli"
	composego "github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v2"
)

// newComposeProject loads and parses a set of input compose files and returns a composeProject object
func newComposeProject(paths []string) (*composeProject, error) {
	raw, err := RawProjectFromSources(paths)
	if err != nil {
		return nil, err
	}
	version, err := getComposeVersion(paths[0])
	if err != nil {
		return nil, err
	}

	return (&composeProject{version, raw}).transform()
}

func (p *composeProject) transform() (*composeProject, error) {
	transforms := []transform{
		augmentOrAddDeploy,
		healthCheckBase,
	}
	for _, t := range transforms {
		if err := t(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// RawProjectFromSources loads and parses a compose-go project from multiple docker-compose source files.
func RawProjectFromSources(paths []string) (*composego.Project, error) {
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
	version := struct {
		Version string `json:"version"` // This affects YAML as well
	}{}

	compose, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	if err = yaml.Unmarshal(compose, &version); err != nil {
		return "", err
	}
	return version.Version, nil
}
