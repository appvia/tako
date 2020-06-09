package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const banner = `
o              
| /            
OO   o-o o   o 
| \  |-'  \ /  
o  o o-o   o      `

var silentErr = errors.New("silentErr")
var rootCmd = &cobra.Command{
	Use: "kev",
	Short: "Reuse and run your Docker Compose applications on Kubernetes",
	Long: `(kev) transforms your Docker Compose applications 
                  into Cloud Native applications you can run on Kubernetes.`,
	SilenceErrors: true,
	SilenceUsage: true,
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if err != silentErr {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
