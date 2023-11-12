package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("daylog %s, git:%s", version, commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
