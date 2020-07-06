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

	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/appvia/kube-devx/pkg/kev/converter"
	"github.com/spf13/cobra"
)

var renderLongDesc = `(render) render Kubernetes manifests in selected format.

  Examples:

	#### Render an app Kubernetes manifests (default) for all environments
	$ kev render

	#### Render an app Kubernetes manifests (default) for a specific environment(s)
	$ kev render -e <production> [-e <dev>]`

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render an application deployment artefacts according to the specified output format for a given environment (ALL environments by default).",
	Long:  renderLongDesc,
	RunE:  runRenderCmd,
}

func init() {
	flags := renderCmd.Flags()
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
		"", // default: will output kubernetes manifests in .kev/.build/<env>/k8s/...
		"Override default Kubernetes manifests output directory. Default: .kev/.build/<env>/k8s/",
	)

	flags.StringSliceP(
		"environment",
		"e",
		[]string{},
		"Target environment for which deployment files should be rendered",
	)

	rootCmd.AddCommand(renderCmd)
}

func runRenderCmd(cmd *cobra.Command, _ []string) error {
	format, err := cmd.Flags().GetString("format")
	singleFile, err := cmd.Flags().GetBool("single")
	dir, err := cmd.Flags().GetString("dir")
	envs, err := cmd.Flags().GetStringSlice("environment")

	fmt.Println("⚙️  Output format:", format)

	switch count := len(envs); {
	case count == 0:
		envs, err = app.GetEnvs()
		if err != nil {
			return fmt.Errorf("render failed, %s", err)
		}
	case count > 0:
		if err := app.ValidateHasEnvs(envs); err != nil {
			return fmt.Errorf("render failed, %s", err)
		}
	}

	appDef, err := app.LoadDefinition(envs)
	if err != nil {
		return err
	}

	c := converter.Factory(format)
	if err := c.Render(singleFile, dir, appDef); err != nil {
		return err
	}

	fmt.Println("🧰 App render complete!")

	return nil
}
