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

	"github.com/appvia/kube-devx/pkg/kev/bootstrap"
	"github.com/disiqueira/gotree"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initLongDesc = `(init) reuses one or more docker-compose files to initialise a cloud native app.

Examples:

  # Initialise an app definition with a single docker-compose file
  $ kev init -n <myapp> -e <production> -c docker-compose.yaml

  # Initialise an app definition with multiple docker-compose files.
  # These will be interpreted as one file.
  $ kev init -n <myapp> -e <production> -c docker-compose.yaml -c docker-compose.other.yaml`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Reuses project docker-compose file(s) to initialise an app definition.",
	Long:  initLongDesc,
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

	defSource := gotree.New("\n\nSource compose file(s)")
	for _, e := range composeFiles {
		defSource.Add(e)
	}
	fmt.Println(defSource.Print())

	def, err := bootstrap.NewApp(BaseDir, appName, composeFiles)
	if err != nil {
		return err
	}

	appDir := path.Join(BaseDir, appName)
	if err := os.MkdirAll(appDir, os.ModePerm); err != nil {
		return err
	}

	ioutil.WriteFile(def.BaseCompose.FilePath, def.BaseCompose.Content, os.ModePerm)
	if err != nil {
		return err
	}

	enc := yaml.NewEncoder(outFile)
	enc.SetIndent(2)

	ioutil.WriteFile(def.Config.FilePath, def.Config.Content, os.ModePerm)
	if err != nil {
		return err
	}

	fmt.Println("🚀 App initialised")
	defTree := gotree.New(BaseDir)
	node2 := defTree.Add(appName)
	node2.Add(def.BaseCompose.FilePath)
	node2.Add(def.Config.FilePath)
	fmt.Println(defTree.Print())

	return nil
}
