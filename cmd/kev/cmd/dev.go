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
	"os"
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

   ### Run Kev in dev mode and render manifests to custom directory
   $ kev dev -d my-manifests

   ### Run Kev in dev mode with Skaffold dev loop activated
   $ kev dev --skaffold
 `

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Watches changes to the source Compose files and re-renders K8s manifests.",
	Long:  devLongDesc,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return verifySkaffoldExpectedFlags(cmd)
	},
	RunE: runDevCmd,
}

const skaffoldNamespace = "default"

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
		"Specify an environment",
	)

	flags.BoolP("skaffold", "", false, "[Experimental] Activates Skaffold dev loop.")

	flags.StringP(
		"namespace",
		"n",
		skaffoldNamespace, // default: will be default kubernetes namespace...
		"[Experimental] Kubernetes namespaces to which Skaffold dev deploys the application.",
	)

	flags.StringP(
		"kubecontext",
		"k",
		"", // default: it'll use currently set kubecontext...
		"[Experimental] Kubernetes context to be used by Skaffold dev.",
	)

	flags.StringP(
		"kev-env",
		"",
		"", // default: it'll use the first element from `environment` slice...
		"[Experimental] Kev environment that will be deployed by Skaffold. If not specified it'll use the first env name passed in '--environment' flag.",
	)

	rootCmd.AddCommand(devCmd)
}

func verifySkaffoldExpectedFlags(cmd *cobra.Command) error {
	envs, _ := cmd.Flags().GetStringSlice("environment")
	skaffold, _ := cmd.Flags().GetBool("skaffold")
	namespace, _ := cmd.Flags().GetString("namespace")
	kubecontext, _ := cmd.Flags().GetString("kubecontext")
	kevenv, _ := cmd.Flags().GetString("kev-env")

	if skaffold {
		if len(namespace) == 0 {
			log.Warnf("Skaffold `namespace` not specified - will use `%s`", skaffoldNamespace)
			cmd.Flag("namespace").Value.Set(skaffoldNamespace)
		} else {
			log.Infof("Skaffold dev loop will deploy to `%s` namespace", namespace)
		}

		if len(kubecontext) == 0 {
			log.Warn("Skaffold `kubecontext` not specified - will use current kubectl context")
		} else {
			log.Infof("Skaffold dev loop will use `%s` kube context", kubecontext)
		}

		if len(kevenv) == 0 {
			log.Warnf("Kev environment not specified. Skaffold will use profile pointing at default `%s` environment", envs[0])
			cmd.Flag("kev-env").Value.Set(envs[0])
		} else {
			log.Infof("Skaffold will use profile pointing at Kev `%s` environment", kevenv)
		}
	}

	return nil
}

func runDevCmd(cmd *cobra.Command, args []string) error {
	envs, err := cmd.Flags().GetStringSlice("environment")
	skaffold, err := cmd.Flags().GetBool("skaffold")
	namespace, err := cmd.Flags().GetString("namespace")
	kubecontext, err := cmd.Flags().GetString("kubecontext")
	kevenv, err := cmd.Flags().GetString("kev-env")

	if err != nil {
		return err
	}

	log.Infof(`Running Kev in development mode... Watched environments: %s`, strings.Join(envs, ", "))

	change := make(chan string, 50)
	defer close(change)

	workDir, err := os.Getwd()
	if err != nil {
		log.Error("Couldn't get working directory")
		return err
	}

	if skaffold {
		profileName := kevenv + kev.EnvProfileNameSuffix
		go kev.RunSkaffoldDev(cmd.Context(), cmd.OutOrStdout(), []string{profileName}, namespace, kubecontext, 1000)
	}

	go kev.Watch(workDir, envs, change)

	for {
		ch := <-change
		if len(ch) > 0 {
			fmt.Printf("\n♻️  %s changed! Re-rendering manifests...\n\n", ch)

			if err := runReconcileCmd(cmd, args); err != nil {
				return err
			}

			if err := runDetectSecretsCmd(cmd, args); err != nil {
				return err
			}

			// re-render manifests for specified environments only
			if err := runRenderCmd(cmd, args); err != nil {
				return err
			}

			// empty the buffer as we only ever do one re-render cycle per a batch of changes
			if len(change) > 0 {
				for range change {
					if len(change) == 0 {
						break
					}
				}
			}
		}
	}
}
