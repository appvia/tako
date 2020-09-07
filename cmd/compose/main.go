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

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/compose-spec/compose-go/cli"
	composegoLoader "github.com/compose-spec/compose-go/loader"
	composego "github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

func main() {
	files := []string{
		"docker-compose.yaml",
		"docker-compose.kev.dev.yaml",
	}
	fmt.Println("Parsing ./docker-compose.yaml")

	p, err := loadFromSources(files)
	// p, err := rawProjectFromSources(files)
	if err != nil {
		panic(err)
	}
	fmt.Println("Marshalling ./docker-compose.yaml")
	data, err := yaml.Marshal(p)
	if err != nil {
		panic(err)
	}
	if err := write("docker-compose.parsed.yaml", data); err != nil {
		panic(err)
	}
}

func write(filePath string, data []byte) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Close()
}

func loadFromSources(paths []string) (*composego.Project, error) {
	var configs []composego.ConfigFile

	for _, file := range paths {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		config, err := composegoLoader.ParseYAML(b)
		if err != nil {
			return nil, err
		}

		configs = append(configs, composego.ConfigFile{Filename: file, Config: config})
	}

	return composegoLoader.Load(composego.ConfigDetails{
		ConfigFiles: configs,
		WorkingDir:  ".",
	})
}

func rawProjectFromSources(paths []string) (*composego.Project, error) {
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
