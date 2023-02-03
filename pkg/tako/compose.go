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

package tako

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/appvia/tako/pkg/tako/log"
	"github.com/compose-spec/compose-go/cli"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// defaultComposeFileNames defines the Compose file names for auto-discovery (in order of preference)
var defaultComposeFileNames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yml",
	"docker-compose.yaml",
}

// defaultComposeOverrideFileNames defines the Compose file names for auto-discovery (in order of preference)
var defaultComposeOverrideFileNames = []string{
	"compose.override.yaml",
	"compose.override.yml",
	"docker-compose.override.yml",
	"docker-compose.override.yaml",
}

type ComposeOpts func(project *ComposeProject) (*ComposeProject, error)

// NewComposeProject loads and parses a set of input compose files and returns a ComposeProject object
func NewComposeProject(paths []string, opts ...ComposeOpts) (*ComposeProject, error) {
	raw, err := rawProjectFromSources(paths)
	if err != nil {
		return nil, err
	}
	version, err := getComposeVersion(paths[0])
	if err != nil {
		return nil, err
	}

	p := &ComposeProject{version: version, Project: raw}
	for _, opt := range opts {
		_, err := opt(p)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

// WithTransforms ensures project attributes are augmented beyond the base compose-go values
func WithTransforms(p *ComposeProject) (*ComposeProject, error) {
	return p.transform()
}

func (p *ComposeProject) transform() (*ComposeProject, error) {
	transforms := []transform{
		augmentOrAddDeploy,
		augmentOrAddHealthCheck,
	}
	for _, t := range transforms {
		if err := t(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// GetVersion gets a project's version
func (p *ComposeProject) GetVersion() string {
	return p.version
}

// rawProjectFromSources loads and parses a compose-go project from multiple docker-compose source files.
func rawProjectFromSources(paths []string) (*composego.Project, error) {
	projectOptions, err := cli.NewProjectOptions(paths, cli.WithOsEnv, cli.WithDotEnv, cli.WithDiscardEnvFile)
	if err != nil {
		return nil, err
	}

	return cli.ProjectFromOptions(projectOptions)
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

// findDefaultComposeFiles scans the workingDir to find a root docker-compose file
// and its optional override file.
func findDefaultComposeFiles(workingDir string) ([]string, error) {
	var defaults []string

	composeFile, err := findDefaultComposeIn(workingDir)
	if err != nil {
		return nil, err
	}
	defaults = append(defaults, composeFile)

	if overrideFile := findOptionalOverrideComposeIn(filepath.Dir(composeFile)); len(overrideFile) > 0 {
		defaults = append(defaults, overrideFile)
	}

	return defaults, nil
}

func findDefaultComposeIn(workingDir string) (string, error) {
	pwd := workingDir
	if pwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		pwd = wd
	}

	found := findFirstFileFromFilesInDir(defaultComposeFileNames, pwd)
	if len(found) > 0 {
		return found, nil
	}

	return "", errors.New("can't find any docker compose file in this directory")
}

func findOptionalOverrideComposeIn(composeFileDir string) string {
	return findFirstFileFromFilesInDir(defaultComposeOverrideFileNames, composeFileDir)
}

func findFirstFileFromFilesInDir(files []string, dir string) string {
	var candidates []string

	for _, n := range files {
		f := filepath.Join(dir, n)
		if _, err := os.Stat(f); err == nil {
			candidates = append(candidates, f)
		}
	}

	if len(candidates) > 0 {
		winner := candidates[0]
		if len(candidates) > 1 {
			log.Debugf("Found multiple override config files with supported names: %s", strings.Join(candidates, ", "))
			log.Debugf("Using %s", winner)
		}
		return winner
	}

	return ""
}

// envVarsFromNilToBlankInService updates nil pointer env vars to empty strings.
// It works around a quirk in the compose-go lib where env var values are defined as *string rather string.
// It's handy for avoiding unknown type errors when diffing env vars.
// And, if required during a merge, it ensures env vars with nil pointer values are overridden.
func envVarsFromNilToBlankInService(svc composego.ServiceConfig) {
	emptyVal := ""
	for key, val := range svc.Environment {
		if val == nil {
			svc.Environment[key] = &emptyVal
		}
	}
}
