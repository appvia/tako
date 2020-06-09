package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/compose-spec/compose-go/loader"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/spf13/cobra"
)

var longDescription = `(init) reuses one or more docker-compose files to initialise a cloud native app.

Examples:

  # Initialise an app definition with a single docker-compose file
  $ kev init -n <myapp> -e <production> -c docker-compose.yaml

  # Initialise an app definition with multiple docker-compose files.
  # These will be interpreted as one file.
  $ kev init -n <myapp> -e <production> -c docker-compose.yaml -c docker-compose.other.yaml`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Reuses project docker-compose file(s) to initialise an app definition.",
	Long:  longDescription,
	RunE: runInitCmd,
}

func init() {
	flags := initCmd.Flags()
	flags.SortFlags = false

	flags.StringP(
		"name",
		"n",
		"",
		"Application name",
	)
	initCmd.MarkFlagRequired("name")

	flags.StringSliceP(
		"compose-file",
		"c",
		[]string{},
		"Compose file to use as application base - use multiple flags for additional files",
	)
	initCmd.MarkFlagRequired("compose-file")

	flags.StringP(
		"environment",
		"e",
		"",
		"Target environment in addition to application base (optional) ",
	)

	rootCmd.AddCommand(initCmd)
}

func runInitCmd(cmd *cobra.Command, args []string) error {
	composeFiles, _ := cmd.Flags().GetStringSlice("compose-file")
	config, err := load(composeFiles)

	if err != nil {
		return err
	}

	fmt.Println("Loaded ...")
	prettyPrint(composeFiles)
	fmt.Println("\nFound ...")
	fmt.Println("\nServices:")
	prettyPrint(config.ServiceNames())

	fmt.Println("\nVolumes:")
	prettyPrint(config.VolumeNames())

	fmt.Println("\nNetworks:")
	prettyPrint(config.NetworkNames())

	return nil
}

func load(paths []string) (*compose.Config, error) {
	var configFiles []compose.ConfigFile

	for _, path := range paths {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		config, err := loader.ParseYAML(b)
		if err != nil {
			return nil, err
		}

		configFiles = append(configFiles, compose.ConfigFile{Filename: path, Config: config})
	}

	return loader.Load(compose.ConfigDetails{
		WorkingDir:  path.Dir(paths[0]),
		ConfigFiles: configFiles,
	})
}

func prettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Printf("%s\n", string(b))
	}
	return
}
