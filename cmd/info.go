package cmd

import (
	"fmt"
	"log"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display the full path to the logs directory.",
	Long:  "Display the full path to the logs directory.",
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := daylog.ProjectPath(config.Project)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(projectPath)
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
