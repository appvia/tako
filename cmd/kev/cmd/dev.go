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
	"strings"

	"github.com/appvia/kev/pkg/kev"
	"github.com/appvia/kev/pkg/kev/log"
	"github.com/spf13/cobra"
)

var devLongDesc = `(dev) Continuously watches and reconciles changes to the source compose files and re-renders K8s manifests.

  Examples:

   ### Run Kev in dev mode
   $ kev dev

   ### Run Kev in dev mode for a particular environment only
   $ kev dev -e dev -e prod
 `

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Watches changes to the source Compose files and re-renders K8s manifests.",
	Long:  devLongDesc,
	RunE:  runDevCmd,
}

func init() {
	flags := devCmd.Flags()
	flags.SortFlags = false

	flags.StringP(
		"format",
		"f",
		"kubernetes", // default: native kubernetes manifests
		"Deployment files format. Default: Kubernetes manifests.",
	)

	flags.BoolP(
		"single",
		"s",
		false, // default: produce multiple files. If true then a single file will be produced.
		"Controls whether to produce individual manifests or a single file output. Default: false",
	)

	flags.StringP(
		"dir",
		"d",
		"", // default: will output kubernetes manifests in k8s/<env>...
		"Override default Kubernetes manifests output directory. Default: k8s/<env>",
	)

	flags.StringSliceP(
		"environment",
		"e",
		[]string{"dev"},
		"Specify an environment\n(default: dev)",
	)

	rootCmd.AddCommand(devCmd)
}

func runDevCmd(cmd *cobra.Command, args []string) error {
	envs, err := cmd.Flags().GetStringSlice("environment")
	if err != nil {
		return err
	}

	log.Infof(`Running Kev in development mode... Watched environments: %s`, strings.Join(envs, ", "))

	change := make(chan string, 1)
	defer close(change)

	go kev.Dev(envs, change)

	for {
		if len(<-change) > 0 {
			fmt.Print("\n♻️  Re-rendering manifests...\n\n")

			if err := runReconcileCmd(cmd, args); err != nil {
				return err
			}

			if err := runRenderCmd(cmd, args); err != nil {
				return err
			}

			// <-change
		}
	}
}
