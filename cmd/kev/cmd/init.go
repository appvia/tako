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
	"io"
	"log"
	"os"
	"path"

	"github.com/appvia/kube-devx/pkg/kev"
	"github.com/spf13/cobra"
)

var initLongDesc = `(init) reuses your docker-compose files to initialise cloud native app rendering.

Examples:

  #### Initialise project rendering with a single docker-compose file
  $ kev init -f docker-compose.yaml

  #### Initialise project rendering with multiple docker-compose files. These will be interpreted as one file.
  $ kev init -f docker-compose.yaml -f docker-compose.other.yaml

  #### Initialise project rendering with a default deployment environment.
  $ kev init -f docker-compose.yaml

  #### Initialise project rendering with a deployment environment.
  $ kev init -e staging -f docker-compose.yaml

  #### Initialise project rendering with multiple deployment environments.
  $ kev init -e staging -e dev -e prod -f docker-compose.yaml`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Reuses project docker-compose file(s) to initialise project rendering.",
	Long:  initLongDesc,
	RunE:  runInitCmd,
}

func init() {
	flags := initCmd.Flags()
	flags.SortFlags = false

	// TODO: add defaults behaviour similar to docker-compose.
	//  Defaults: docker-compose.yml & docker-compose.override.yml
	flags.StringSliceP(
		"file",
		"f",
		[]string{},
		"Specify a compose file",
	)
	if err := initCmd.MarkFlagRequired("file"); err != nil {
		log.Fatal(err)
	}

	flags.StringSliceP(
		"environment",
		"e",
		[]string{"dev"},
		"Specify a deployment environment (default: dev)",
	)

	rootCmd.AddCommand(initCmd)
}

func runInitCmd(cmd *cobra.Command, _ []string) error {
	files, _ := cmd.Flags().GetStringSlice("file")
	envs, _ := cmd.Flags().GetStringSlice("environment")

	manifest, err := kev.Init(files, envs)
	if err != nil {
		return err
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	filePath := path.Join(workingDir, kev.ManifestName)
	if err := writeTo(filePath, manifest); err != nil {
		return err
	}

	for _, environment := range manifest.Environments {
		filePath := path.Join(workingDir, environment.File)
		if err := writeTo(filePath, environment); err != nil {
			return err
		}
	}

	return nil
}

func writeTo(filePath string, w io.WriterTo) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	if _, err := w.WriteTo(file); err != nil {
		return err
	}
	return file.Close()
}
