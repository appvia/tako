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

	"github.com/appvia/kev/pkg/kev"
	"github.com/spf13/cobra"
)

func runDetectSecretsCmd(cmd *cobra.Command, _ []string) error {
	cmdName := "Detect secrets"
	verbose, _ := cmd.Root().Flags().GetBool("verbose")

	workingDir, err := os.Getwd()
	if err != nil {
		return displayError(cmdName, err)
	}

	reporter := getReporter(verbose)
	if err := kev.DetectSecrets(workingDir, reporter); err != nil {
		return displayError(cmdName, err)
	}
	_, _ = reporter.Write([]byte("\n"))

	return nil
}
