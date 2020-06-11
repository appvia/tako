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

package bootstrap

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/compose-spec/compose-go/loader"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/disiqueira/gotree"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

// BaseDir is a top level directory for Kev files
const BaseDir = ".kev"

// FromCompose initiate an App from docker compose files
func FromCompose(cmd *cobra.Command, args []string) error {
	appName, _ := cmd.Flags().GetString("name")
	composeFiles, _ := cmd.Flags().GetStringSlice("compose-file")

	config, err := load(composeFiles)
	if err != nil {
		return err
	}

	defSource := gotree.New("\n\nSource compose file(s)")
	for _, e := range composeFiles {
		defSource.Add(e)
	}
	fmt.Println(defSource.Print())

	appDir := path.Join(BaseDir, appName)
	if err := os.MkdirAll(appDir, os.ModePerm); err != nil {
		return err
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	appBaseComposeFile := "compose.yaml"
	appBaseComposePath := path.Join(appDir, appBaseComposeFile)
	ioutil.WriteFile(appBaseComposePath, bytes, os.ModePerm)
	if err != nil {
		return err
	}

	appBaseConfigFile := "config.yaml"
	appBaseConfigPath := path.Join(appDir, appBaseConfigFile)
	var appTempConfigContent = fmt.Sprintf(`app:
   name: %s
   description: new app.
 `, appName)
	ioutil.WriteFile(appBaseConfigPath, []byte(appTempConfigContent), os.ModePerm)
	if err != nil {
		return err
	}

	fmt.Println("🚀 App initialised")
	defTree := gotree.New(BaseDir)
	node2 := defTree.Add(appName)
	node2.Add(appBaseComposeFile)
	node2.Add(appBaseConfigFile)
	fmt.Println(defTree.Print())

	return nil
}

func load(paths []string) (*compose.Config, error) {
	var configFiles []compose.ConfigFile

	for _, path := range paths {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		config, err := loader.ParseYAML(b)
		if err != nil {
			return nil, err
		}

		configFiles = append(configFiles, compose.ConfigFile{Filename: path, Config: config})
	}

	return loader.Load(compose.ConfigDetails{
		WorkingDir:  path.Dir(paths[0]),
		ConfigFiles: configFiles,
	})
}
