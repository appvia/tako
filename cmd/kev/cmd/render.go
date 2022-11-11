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

var renderLongDesc = `(render) render Kubernetes manifests in selected format.

Examples:

  ### Render an app Kubernetes manifests (default) for all environments
  $ kev render

  ### Render an app Kubernetes manifests (default) for a specific environment(s)
  $ kev render -e staging [-e production ...]`

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Generates application's deployment artefacts according to the specified output format for a given environment (ALL environments by default).",
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
		"", // default: will output kubernetes manifests in k8s/<env>...
		"Override default Kubernetes manifests output directory. Default: k8s/<env>",
	)

	flags.StringSliceP(
		"environment",
		"e",
		[]string{},
		"Target environment for which deployment files should be rendered",
	)

	flags.StringSliceP(
		"additional-manifests",
		"a",
		[]string{},
		"Additional Kubernetes manifests to be included in the output",
	)

	rootCmd.AddCommand(renderCmd)
}

func runRenderCmd(cmd *cobra.Command, _ []string) error {
	format, _ := cmd.Flags().GetString("format")
	singleFile, _ := cmd.Flags().GetBool("single")
	dir, _ := cmd.Flags().GetString("dir")
	envs, _ := cmd.Flags().GetStringSlice("environment")
	verbose, _ := cmd.Root().Flags().GetBool("verbose")
	additionalManifests, _ := cmd.Flags().GetStringSlice("additional-manifests")

	// The working directory is always the current directory.
	// This ensures created manifest yaml entries are portable between users and require no path fixing.
	wd := "."

	return kev.RenderProjectWithOptions(wd,
		kev.WithAppName(rootCmd.Use),
		kev.WithManifestFormat(format),
		kev.WithManifestsAsSingleFile(singleFile),
		kev.WithAdditionalManifests(additionalManifests),
		kev.WithOutputDir(dir),
		kev.WithEnvs(envs),
		kev.WithLogVerbose(verbose),
	)
}
