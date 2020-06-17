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

	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/spf13/cobra"
)

//  @todo: change this when build does more than just config compilation!
var buildLongDesc = `(build) builds configuration.

 Examples:

   # Builds an app configuration for a given environment
   $ kev build -n <myapp> -e [<production>]`

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

func runBuildCmd(cmd *cobra.Command, args []string) error {
	appEnvironments, _ := cmd.Flags().GetStringSlice("environment")

	// No environment supplied - discovering all env subdirs
	if len(appEnvironments) == 0 {
		files, err := ioutil.ReadDir(BaseDir)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.IsDir() {
				appEnvironments = append(appEnvironments, file.Name())
			}
		}
	}

	compiledConfigs, err := config.Compile(BaseDir, appEnvironments)
	if err != nil {
		return err
	}

	for _, compiledConfig := range compiledConfigs {
		if err = ioutil.WriteFile(compiledConfig.FilePath, compiledConfig.Content, os.ModePerm); err != nil {
			return err
		}
		fmt.Printf("🔩 App configuration built for `%s` environment\n", compiledConfig.Environment)
	}

	return nil
}
