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
	"github.com/appvia/kev/pkg/kev"
	"github.com/spf13/cobra"
)

var initLongDesc = `Tracks compose sources & creates deployment environments.

Examples:

  ### Initialise kev.yaml with root docker-compose.yml and override file tracking. Adds a sandbox dev deployment environment.
  $ kev init

  ### Use an alternate docker-compose.yml file.
  $ kev init -f docker-compose.dev.yaml

  ### Use multiple alternate docker-compose.yml files.
  $ kev init -f docker-compose.alternate.yaml -f docker-compose.other.yaml

  ### Use a specified environment - in addition to a sandbox dev deployment environment.
  $ kev init -e staging

  ### Use multiple specified environments - in addition to the sandbox dev deployment environment.
  $ kev init -e staging -e production

  ### Prepare project for use with Skaffold.
  $ kev init -e staging --skaffold`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Tracks compose sources & creates deployment environments.",
	Long:  initLongDesc,
	RunE:  runInitCmd,
	PostRunE: func(cmd *cobra.Command, args []string) error {
		// os.Stdout.Write([]byte("\n"))
		// return runDetectSecretsCmd(cmd, args)
		return nil
	},
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

	flags.BoolP("skaffold", "s", false, "prepare the project for Skaffold")

	rootCmd.AddCommand(initCmd)
}

func runInitCmd(cmd *cobra.Command, _ []string) error {
	files, _ := cmd.Flags().GetStringSlice("file")
	envs, _ := cmd.Flags().GetStringSlice("environment")
	skaffold, _ := cmd.Flags().GetBool("skaffold")

	// The working directory is always the current directory.
	// This ensures created manifest yaml entries are portable between users and require no path fixing.
	wd := "."
	return kev.InitProjectWithOptions(wd,
		kev.WithComposeSources(files),
		kev.WithEnvs(envs),
		kev.WithSkaffold(skaffold))
}
