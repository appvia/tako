/**
 * Copyright 2023 Appvia Ltd <info@appvia.io>
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
	"github.com/appvia/tako/pkg/tako"
	"github.com/spf13/cobra"
)

var patchLongDesc = `(patch) Patch previously rendered K8s manifests.

 Examples:

	### Patch images for specified services
	$ tako patch --dir /path/to/k8s/manifests --image myservice=myimage:v1.0.1 --image myservice2=myimage2:v2.0.2

	### Patch images for specified services and store patched manifests in a different directory
	$ tako patch --dir /path/to/k8s/manifests --image myservice=myimage:v1.0.1 --image myservice2=myimage2:v2.0.2 --output-dir /path/to/patched/manifests

 `

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: "Patches K8s manifests by setting images for specified services.",
	Long:  patchLongDesc,
	RunE:  runPatchCmd,
}

func init() {
	flags := patchCmd.Flags()
	flags.SortFlags = false

	flags.StringP(
		"dir",
		"d",
		".", // default: local directory
		"Path to the kubernetes manifests that require patching.",
	)

	flags.StringSliceP(
		"image",
		"i",
		[]string{},
		"Image to be patched in deployment files for a given service name. Format: <service>=<image>.",
	)

	flags.StringP(
		"output-dir",
		"o",
		"",
		`Path to directory where patched manifests should be stored.
⌙ Output directory structure will reflect that of the source directory tree.
⌙ Note that only manifests that were patched will be stored in the output directory!
⌙ Manifests will be overriden in-place if output directory is not specified.`,
	)

	rootCmd.AddCommand(patchCmd)
}

func runPatchCmd(cmd *cobra.Command, _ []string) error {

	dir, _ := cmd.Flags().GetString("dir")
	images, _ := cmd.Flags().GetStringSlice("image")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	verbose, _ := cmd.Root().Flags().GetBool("verbose")

	if dir == "" {
		dir = "."
	}

	// The working directory is always the current directory.
	// This ensures created manifest yaml entries are portable between users and require no path fixing.
	wd := "."

	return tako.PatchWithOptions(wd,
		tako.WithAppName(rootCmd.Use),
		tako.WithPatchManifestsDir(dir),
		tako.WithPatchImages(images),
		tako.WithPatchOutputDir(outputDir),
		tako.WithLogVerbose(verbose),
	)

}
