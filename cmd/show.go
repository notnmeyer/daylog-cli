package cmd

import (
	"fmt"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display today's log",
	Long:  "Display today's log",
	Run: runCommand(func(cmd *cobra.Command, dl *daylog.DayLog) error {
		format, err := cmd.PersistentFlags().GetString("output")
		if err != nil {
			return err
		}

		logContents, err := dl.Show(format)
		if err != nil {
			return err
		}

		fmt.Println(string(logContents))
		return nil
	}),
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.PersistentFlags().StringP("output", "o", "markdown", "Format output")
}
