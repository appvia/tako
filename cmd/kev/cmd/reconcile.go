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
	"path"

	"github.com/appvia/kev/pkg/kev"
	"github.com/spf13/cobra"
)

func runReconcileCmd(cmd *cobra.Command, _ []string) error {
	cmdName := "Reconcile"
	verbose, _ := cmd.Root().Flags().GetBool("verbose")

	workingDir, err := os.Getwd()
	if err != nil {
		return displayError(err)
	}

	setReporting(verbose)
	displayCmdStarted(cmdName)
	displayReconcileRules(verbose)

	manifest, err := kev.Reconcile(workingDir)
	if err != nil {
		return displayError(err)
	}

	for _, environment := range manifest.Environments {
		filePath := path.Join(workingDir, environment.File)
		if err := writeTo(filePath, environment); err != nil {
			return displayError(err)
		}
	}

	return nil
}

func displayReconcileRules(verbose bool) {
	if verbose {
		_, _ = fmt.Fprintf(os.Stdout, "\033[2mHOW DOES RECONCILE WORK?\n")
		_, _ = fmt.Fprintf(os.Stdout, "\033[2m᛫ New services & volumes in a project will be added to all environments.\n")
		_, _ = fmt.Fprintf(os.Stdout, "\033[2m᛫ Removed services & volumes from a project will be removed from all environments.\n")
		_, _ = fmt.Fprintf(os.Stdout, "\033[2m᛫ Generally, updates to a project's settings DON'T affect environments.\n")
		_, _ = fmt.Fprintf(os.Stdout, "\033[2m᛫ Updates to a service's ports in a project DO affect environments.\n")
		_, _ = fmt.Fprintf(os.Stdout, "\033[2m᛫ ENV VARS will be removed if they are found in an environment file but not in the project.\n")
	}
}
