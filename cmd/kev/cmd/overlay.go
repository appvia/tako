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
	"log"

	"github.com/appvia/kube-devx/pkg/kev"
	"github.com/spf13/cobra"
)

var overlayCmd = &cobra.Command{
	Use:   "overlay",
	Short: "Overlay a kev compose config onto a set of compose files.",
	RunE:  runOverlayCmd,
}

func init() {
	flags := overlayCmd.Flags()
	flags.SortFlags = false

	flags.StringSliceP(
		"docker-compose-file",
		"f",
		[]string{},
		"docker-compose file to use as application base - use multiple flags for additional files",
	)
	if err := overlayCmd.MarkFlagRequired("docker-compose-file"); err != nil {
		log.Fatal(err)
	}

	flags.StringP(
		"config",
		"c",
		"",
		"docker-compose file with labels to use as config",
	)
	if err := overlayCmd.MarkFlagRequired("config"); err != nil {
		log.Fatal(err)
	}

	flags.StringP(
		"output",
		"o",
		"",
		"Output file with result of overlaying config over docker-compose file(s)",
	)
	if err := overlayCmd.MarkFlagRequired("output"); err != nil {
		log.Fatal(err)
	}

	rootCmd.AddCommand(overlayCmd)
}

func runOverlayCmd(cmd *cobra.Command, _ []string) error {
	composeFiles, _ := cmd.Flags().GetStringSlice("docker-compose-file")
	config, _ := cmd.Flags().GetString("config")
	out, _ := cmd.Flags().GetString("output")

	return kev.Overlay(composeFiles, config, out)
}
