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

	"github.com/appvia/kev/pkg/kev"
	"github.com/spf13/cobra"
)

var devLongDesc = `(dev) Continuous reconcile and re-render of K8s manifests with optional project build, push and deploy (using --skaffold).

Examples:

   ### Run Kev in dev mode
   $ kev dev

   ### Use a custom directory to render manifests
   $ kev dev -d my-manifests

   ### Activate the Skaffold dev loop to build, push and deploy your project
   $ kev dev --skaffold

   ### Activate the Skaffold dev loop to build, push and deploy your project to a particular namespace
   $ kev dev --skaffold --namespace myspace

   ### Activate the Skaffold dev loop to build, push and deploy your project using a specific kubecontext
   $ kev dev --skaffold --namespace myspace --kubecontext mycontext

   ### Activate the Skaffold dev loop to build, push and deploy your project and tail deployed app logs
   $ kev dev --skaffold --tail

   ### Activate the Skaffold dev loop to build, push and deploy your project "staging" configuration
   $ kev dev --skaffold --kev-env staging

   ### Activate the Skaffold dev loop and manually trigger build, push and deploy of your project (useful for stacking up code changes before deployment)
   $ kev dev --skaffold --manual-trigger
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
		kev.DefaultSkaffoldNamespace, // default: will be default kubernetes namespace...
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
		kev.SandboxEnv,
		fmt.Sprintf("[Experimental] Kev environment that will be deployed by Skaffold. If not specified it'll use the sandbox %s env.", kev.SandboxEnv),
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
	skaffold, err := cmd.Flags().GetBool("skaffold")
	namespace, err := cmd.Flags().GetString("namespace")
	kubecontext, err := cmd.Flags().GetString("kubecontext")
	kevenv, err := cmd.Flags().GetString("kev-env")
	tail, _ := cmd.Flags().GetBool("tail")
	manualTrigger, _ := cmd.Flags().GetBool("manual-trigger")
	verbose, _ := cmd.Root().Flags().GetBool("verbose")

	if err != nil {
		return err
	}

	eventHandler := func(e kev.RunnerEvent, r kev.Runner) error { return nil }

	var envs []string
	if len(kevenv) > 0 {
		envs = append(envs, kevenv)
	}

	// The working directory is always the current directory.
	// This ensures created manifest yaml entries are portable between users and require no path fixing.
	wd := "."

	return kev.DevWithOptions(wd,
		kev.WithAppName(rootCmd.Use),
		kev.WithEventHandler(eventHandler),
		kev.WithSkaffold(skaffold),
		kev.WithK8sNamespace(namespace),
		kev.WithKubecontext(kubecontext),
		kev.WithSkaffoldTailEnabled(tail),
		kev.WithSkaffoldManualTriggerEnabled(manualTrigger),
		kev.WithSkaffoldVerboseEnabled(verbose),
		kev.WithEnvs(envs),
		kev.WithLogVerbose(verbose),
	)
}
