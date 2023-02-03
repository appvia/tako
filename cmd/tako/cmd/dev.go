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

	"github.com/appvia/tako/pkg/tako"
	"github.com/spf13/cobra"
)

var devLongDesc = `(dev) Continuous reconcile and re-render of K8s manifests with optional project build, push and deploy (using --skaffold).

Examples:

   ### Run Tako in dev mode
   $ tako dev

   ### Use a custom directory to render manifests
   $ tako dev -d my-manifests

   ### Activate the Skaffold dev loop to build, push and deploy your project
   $ tako dev --skaffold

   ### Activate the Skaffold dev loop to build, push and deploy your project to a particular namespace
   $ tako dev --skaffold --namespace myspace

   ### Activate the Skaffold dev loop to build, push and deploy your project using a specific kubecontext
   $ tako dev --skaffold --namespace myspace --kubecontext mycontext

   ### Activate the Skaffold dev loop to build, push and deploy your project and tail deployed app logs
   $ tako dev --skaffold --tail

   ### Activate the Skaffold dev loop to build, push and deploy your project "staging" configuration
   $ tako dev --skaffold --tako-env staging

   ### Activate the Skaffold dev loop and manually trigger build, push and deploy of your project (useful for stacking up code changes before deployment)
   $ tako dev --skaffold --manual-trigger
`

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Continuous reconcile and re-render of K8s manifests with optional project build, push and deploy (using --skaffold).",
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

	flags.StringSlice("environment", []string{}, "")
	_ = flags.MarkHidden("environment")

	flags.BoolP("skaffold", "", false, "[Experimental] Activates Skaffold dev loop.")

	flags.StringP(
		"namespace",
		"n",
		tako.DefaultSkaffoldNamespace, // default: will be default kubernetes namespace...
		"[Experimental] Kubernetes namespaces to which Skaffold dev deploys the application.",
	)

	flags.StringP(
		"kubecontext",
		"k",
		"", // default: it'll use currently set kubecontext...
		"[Experimental] Kubernetes context to be used by Skaffold dev.",
	)

	flags.StringP(
		"tako-env",
		"",
		tako.SandboxEnv,
		fmt.Sprintf("[Experimental] Tako environment that will be deployed by Skaffold. If not specified it'll use the sandbox %s env.", tako.SandboxEnv),
	)

	flags.BoolP(
		"tail",
		"t",
		false,
		"[Experimental] Enable Skaffold deployed application log tailing.",
	)

	flags.BoolP(
		"manual-trigger",
		"m",
		false,
		"[Experimental] Expect user to manually trigger Skaffold's build/push/deploy. Useful for batching source code changes before release.",
	)

	rootCmd.AddCommand(devCmd)
}

func runDevCmd(cmd *cobra.Command, _ []string) error {
	skaffold, _ := cmd.Flags().GetBool("skaffold")
	namespace, _ := cmd.Flags().GetString("namespace")
	kubecontext, _ := cmd.Flags().GetString("kubecontext")
	takoenv, _ := cmd.Flags().GetString("tako-env")
	tail, _ := cmd.Flags().GetBool("tail")
	manualTrigger, _ := cmd.Flags().GetBool("manual-trigger")
	verbose, _ := cmd.Root().Flags().GetBool("verbose")

	eventHandler := func(e tako.RunnerEvent, r tako.Runner) error { return nil }

	var envs []string
	if len(takoenv) > 0 && skaffold {
		// when in --skaffold mode - only watch, render and deploy a specified environment
		envs = append(envs, takoenv)
	}

	// The working directory is always the current directory.
	// This ensures created manifest yaml entries are portable between users and require no path fixing.
	wd := "."

	return tako.DevWithOptions(wd,
		tako.WithAppName(rootCmd.Use),
		tako.WithEventHandler(eventHandler),
		tako.WithSkaffold(skaffold),
		tako.WithK8sNamespace(namespace),
		tako.WithKubecontext(kubecontext),
		tako.WithSkaffoldTailEnabled(tail),
		tako.WithSkaffoldManualTriggerEnabled(manualTrigger),
		tako.WithSkaffoldVerboseEnabled(verbose),
		tako.WithEnvs(envs),
		tako.WithLogVerbose(verbose),
	)
}
