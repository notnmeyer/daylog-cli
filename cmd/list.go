package cmd

import (
	"fmt"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/notnmeyer/daylog-cli/internal/file"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all log files",
	Long:  "list all log files relative to the project directory",
	Run: runCommand(func(cmd *cobra.Command, dl *daylog.DayLog) error {
		logs, err := file.LogProvider{}.GetLogs(dl.ProjectPath)
		if err != nil {
			return err
		}

		for _, log := range logs {
			fmt.Println(log)
		}

		return nil
	}),
}

func init() {
	rootCmd.AddCommand(listCmd)
}
