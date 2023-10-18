package cmd

import (
	"fmt"
	"os"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new Git repository in the log directory",
	Run: func(cmd *cobra.Command, args []string) {
		d, err := daylog.New(args)
		if err != nil {
			fmt.Printf("error initializing: %s", err)
			os.Exit(1)
		}

		if err := d.Init(); err != nil {
			fmt.Printf("couldn't create Git repo at %s: %s\n", d.RootPath, err)
			os.Exit(1)
		}

		fmt.Printf("Initialized a Git repository at %s\n", d.RootPath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
