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
	"github.com/appvia/kube-devx/pkg/kev/bootstrap"
	"github.com/spf13/cobra"
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
	return bootstrap.FromCompose(cmd, args)
}
