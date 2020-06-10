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

package cmd

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

var longDescription = `(init) reuses one or more docker-compose files to initialise a cloud native app.

Examples:

  # Initialise an app definition with a single docker-compose file
  $ kev init -n <myapp> -e <production> -c docker-compose.yaml

  # Initialise an app definition with multiple docker-compose files.
  # These will be interpreted as one file.
  $ kev init -n <myapp> -e <production> -c docker-compose.yaml -c docker-compose.other.yaml`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Reuses project docker-compose file(s) to initialise an app definition.",
	Long:  longDescription,
	RunE:  runInitCmd,
}

func init() {
	flags := initCmd.Flags()
	flags.SortFlags = false

	flags.StringP(
		"name",
		"n",
		"",
		"Application name",
	)
	initCmd.MarkFlagRequired("name")

	flags.StringSliceP(
		"compose-file",
		"c",
		[]string{},
		"Compose file to use as application base - use multiple flags for additional files",
	)
	initCmd.MarkFlagRequired("compose-file")

	flags.StringP(
		"environment",
		"e",
		"",
		"Target environment in addition to application base (optional) ",
	)

	rootCmd.AddCommand(initCmd)
}

func runInitCmd(cmd *cobra.Command, args []string) error {
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
