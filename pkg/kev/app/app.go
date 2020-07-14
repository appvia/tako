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

package app

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

const (
	// baseDir is a top level directory for app files
	baseDir = ".kev"

	// workDir is app workspace directory and contains interim artifacts and compiled configuration
	// If defined it must start with "." to differentiate from environment directories
	workDir = ".workspace"

	// buildDir is the app build directory.
	// It must start with "." if workDir is not specified
	buildDir = "build"

	// composeFile base compose file name
	composeFile = "compose.yaml"

	// composeBuildFile build time compose file name
	composeBuildFile = "compose.build.yaml"

	// configFile config file name (base)
	configFile = "config.yaml"

	// configBuildFile build time config file name
	configBuildFile = "config.build.yaml"
)

// ValidateHasEnvs checks whether supplied environments exist or not.
func ValidateHasEnvs(candidates []string) error {
	envs, err := GetEnvs()
	if err != nil {
		return err
	}

	sort.Strings(envs)
	var invalid []string

	for _, c := range candidates {
		i := sort.SearchStrings(envs, c)
		valid := i < len(envs) && envs[i] == c
		if !valid {
			invalid = append(invalid, c)
		}
	}

	if len(invalid) > 0 {
		return fmt.Errorf("cannot find environment(s): %s", strings.Join(invalid, ", "))
	}

	return nil
}

// GetEnvs returns a string slice of all app environments
func GetEnvs() ([]string, error) {
	var envs []string

	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			envs = append(envs, file.Name())
		}
	}

	return envs, nil
}

// ValidateEnvsOrGetAll ensures the supplied non empty list of envs is valid.
// If env list is empty return all available envs.
func ValidateEnvsOrGetAll(envs []string) ([]string, error) {
	switch count := len(envs); {
	case count == 0:
		var err error
		envs, err = GetEnvs()
		if err != nil {
			return nil, fmt.Errorf("builds failed, %s", err)
		}
	case count > 0:
		if err := ValidateHasEnvs(envs); err != nil {
			return nil, fmt.Errorf("builds failed, %s", err)
		}
	}

	return envs, nil
}
