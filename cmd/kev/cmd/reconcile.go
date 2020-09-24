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
		return displayError(cmdName, err)
	}

	reporter := getReporter(verbose)
	manifest, err := kev.Reconcile(workingDir, reporter)
	if err != nil {
		return displayError(cmdName, err)
	}

	for _, environment := range manifest.Environments {
		filePath := path.Join(workingDir, environment.File)
		if err := writeTo(filePath, environment); err != nil {
			return displayError(cmdName, err)
		}
	}

	return nil
}
