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
		dl, err := daylog.New(args, config.Project)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(dl.ProjectPath)
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
