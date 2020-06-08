package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var silentErr = errors.New("silentErr")
var rootCmd = &cobra.Command{
	Use:   "kev",
		Short: "Reuse and run your Docker Compose applications on Kubernetes",
	Long: `Kev helps you transform your Docker Compose applications 
   into Cloud Native applications you can run on Kubernetes.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello World!")
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
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
