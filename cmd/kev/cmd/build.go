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

	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/spf13/cobra"
)

//  @todo: change this when build does more than just config compilation!
var buildLongDesc = `(build) builds configuration.

 Examples:

   #### Builds an app configuration for all environments
   $ kev build

   #### Builds an app configuration for a specific environment(s)
   $ kev build -e <production> [-e <dev>]`

var buildCmd = &cobra.Command{
	Use: "build",
	// @todo: change short description!
	Short: "Builds an application configuration for given environment (ALL environments by default).",
	Long:  buildLongDesc,
	RunE:  runBuildCmd,
}

func init() {
	flags := buildCmd.Flags()
	flags.SortFlags = false

	flags.StringSliceP(
		"environment",
		"e",
		[]string{},
		"Target environment for which configuration should be compiled",
	)

	rootCmd.AddCommand(buildCmd)
}

func runBuildCmd(cmd *cobra.Command, _ []string) error {
	envs, err := cmd.Flags().GetStringSlice("environment")

	switch count := len(envs); {
	case count == 0:
		envs, err = app.GetEnvs()
		if err != nil {
			return fmt.Errorf("builds failed, %s", err)
		}
	case count > 0:
		if err := app.ValidateHasEnvs(envs); err != nil {
			return fmt.Errorf("builds failed, %s", err)
		}
	}

	def, err := app.LoadDefinition(envs)
	if err != nil {
		return err
	}

	if err := def.DoBuild(); err != nil {
		return err
	}

	for _, build := range def.GetInternalBuildInfo() {
		if err := os.MkdirAll(build.Config.Path(), os.ModePerm); err != nil {
			return err
		}

		if err = ioutil.WriteFile(build.Config.File, build.Config.Content, os.ModePerm); err != nil {
			return err
		}
		if err = ioutil.WriteFile(build.Compose.File, build.Compose.Content, os.ModePerm); err != nil {
			return err
		}
	}

	displayBuild(def)
	return nil
}

func displayBuild(def *app.Definition) {
	if def.HasBuiltOverrides() {
		displayEnvsBuilt(def.GetOverridesBuildInfo())
	} else {
		displayAppBuilt()
	}
}

func displayEnvsBuilt(builtEnvsSet map[string]app.ConfigTuple) {
	for k := range builtEnvsSet {
		fmt.Printf("🔧 [%s] environment ready.\n", k)
	}
	displayAppBuilt()
}

func displayAppBuilt() {
	fmt.Println("🔩 App build complete!")
}
