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
	"errors"
	"os"

	"github.com/appvia/tako/pkg/tako/config"
	"github.com/spf13/cobra"
)

var silentErr = errors.New("silentErr")
var rootCmd = &cobra.Command{
	Short:            "Develop Kubernetes apps iteratively using Docker-Compose.",
	Use:              config.AppName,
	TraverseChildren: true,
	SilenceErrors:    true,
	SilenceUsage:     true,
}

// NewRootCmd returns root command
func NewRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	flags := rootCmd.Flags()
	flags.SortFlags = false

	flags.BoolP(
		"verbose",
		"v",
		false,
		"Show more output",
	)

	// This is required to help with error handling from RunE , https://github.com/spf13/cobra/issues/914#issuecomment-548411337
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println(cmd.UsageString())
		return silentErr
	})
}

// Execute command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
