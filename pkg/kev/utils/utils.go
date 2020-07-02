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

package utils

import (
	"bytes"
	"io/ioutil"
	"log"

	"github.com/compose-spec/compose-go/cli"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/goccy/go-yaml"
	yaml3 "gopkg.in/yaml.v3"
)

// UnmarshallGeneral deserializes a []byte into an map[string]interface{}
func UnmarshallGeneral(data []byte) (map[string]interface{}, error) {
	var out map[string]interface{}
	err := yaml.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MarshallAndFormat marshals arbitrary struct
func MarshallAndFormat(v interface{}, spaces int) ([]byte, error) {
	var out bytes.Buffer
	encoder := yaml3.NewEncoder(&out)
	defer func() {
		if err := encoder.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	encoder.SetIndent(spaces)
	if err := encoder.Encode(&v); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// LoadAndParse loads and parses a set of input compose files and returns compose Project object
func LoadAndParse(paths []string) (*compose.Project, error) {
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

// GetComposeVersion extracts version from compose file and returns a string
func GetComposeVersion(file string) (string, error) {
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
