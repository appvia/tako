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
	"os"
	"path"

	"github.com/appvia/kube-devx/pkg/kev"
	"github.com/spf13/cobra"
)

var initLongDesc = `Tracks compose sources & creates deployment environments.

Examples:

  # Initialise kev.yaml with root docker-compose.yml and override file tracking.
  # Adds the default dev deployment environment.
  $ kev init

  # Use an alternate docker-compose.yml file.
  $ kev init -f docker-compose.dev.yaml
  
  # Use multiple alternate docker-compose.yml files.
  $ kev init -f docker-compose.alternate.yaml -f docker-compose.other.yaml

  # Use a specified environment.
  $ kev init -e staging

  # Use multiple specified environments.
  $ kev init -e staging`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Tracks compose sources & creates deployment environments.",
	Long:  initLongDesc,
	RunE:  runInitCmd,
}

func init() {
	flags := initCmd.Flags()
	flags.SortFlags = false

	flags.StringSliceP(
		"file",
		"f",
		[]string{},
		"Specify an alternate compose file\n(default: docker-compose.yml or docker-compose.yaml)",
	)

	flags.StringSliceP(
		"environment",
		"e",
		[]string{},
		"Specify a deployment environment\n(default: dev)",
	)

	rootCmd.AddCommand(initCmd)
}

func runInitCmd(cmd *cobra.Command, _ []string) error {
	files, _ := cmd.Flags().GetStringSlice("file")
	envs, _ := cmd.Flags().GetStringSlice("environment")

	manifest, err := kev.Init(files, envs, ".")
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
