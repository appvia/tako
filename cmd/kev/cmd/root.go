package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/compose-spec/compose-go/loader"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/spf13/cobra"
)

const banner = `
o              
| /            
OO   o-o o   o 
| \  |-'  \ /  
o  o o-o   o  

`

var silentErr = errors.New("silentErr")
var rootCmd = &cobra.Command{
	Use:   "kev",
		Short: "Reuse and run your Docker Compose applications on Kubernetes",
	Long: `Kev helps you transform your Docker Compose applications 
   into Cloud Native applications you can run on Kubernetes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		base := path.Join( "hack", "sample-dc-app", "docker-compose.yml")
		override := path.Join("hack", "sample-dc-app", "docker-compose.override.yml")
		workingDir := path.Dir(base)

		config, err := load(
			workingDir,
			[]string{
				base,
				override,
			})

		if err != nil {
			return err
		}

		fmt.Printf("Loaded ...\n\n- base: [%s]\n- override: [%s]\n", base, override)
		fmt.Println("\nFound ...")
		fmt.Println("\nServices:")
		prettyPrint(config.ServiceNames())

		fmt.Println("\nVolumes:")
		prettyPrint(config.VolumeNames())

		fmt.Println("\nNetworks:")
		prettyPrint(config.NetworkNames())
		return nil
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	fmt.Print(banner)

	// This is required to help with error handling from RunE , https://github.com/spf13/cobra/issues/914#issuecomment-548411337
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println(cmd.UsageString())
		return silentErr
	})
}

func load(workingDir string, paths []string) (*compose.Config, error) {
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
		WorkingDir:  workingDir,
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if err != silentErr {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
