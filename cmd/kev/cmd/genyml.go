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

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var genymlCmd = &cobra.Command{
	Use:   "genyml",
	Short: "Spikes generating yml with comments & more.",
	RunE:  runGenYamlCmd,
}

// App struct
type App struct {
	Name string `yaml:"name"`
	// Services []yaml.Node `yaml:"services"`
	Services *yaml.Node `yaml:"services,omitempty"`
}

// Doc struct
type Doc struct {
	App *App `yaml:"app"`
}

func runGenYamlCmd(cmd *cobra.Command, _ []string) error {
	out, _ := cmd.Flags().GetString("out")
	doc := &Doc{
		App: &App{
			Name: "hello-world",
			Services: &yaml.Node{
				HeadComment: "Start, expected app services",
				Kind:        yaml.SequenceNode,
				Content: []*yaml.Node{
					{
						Kind:        yaml.ScalarNode,
						Value:       "[placeholder]",
						LineComment: "add service name",
					},
					{
						Kind:        yaml.ScalarNode,
						Value:       "[placeholder]",
						LineComment: "add service name",
					},
				},
			},
		},
	}

	outFile, err := os.Create(out)
	if err != nil {
		return err
	}
	defer outFile.Close()

	enc := yaml.NewEncoder(outFile)
	enc.SetIndent(2)
	return enc.Encode(doc)
}

func init() {
	flags := genymlCmd.Flags()
	flags.SortFlags = false

	flags.StringP(
		"out",
		"o",
		"",
		"Output destination file",
	)
	_ = genymlCmd.MarkFlagRequired("out")

	rootCmd.AddCommand(genymlCmd)
}
