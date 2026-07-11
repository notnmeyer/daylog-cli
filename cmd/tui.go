package cmd

import (
	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/notnmeyer/daylog-cli/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Browse and edit logs in an interactive terminal UI",
	Long:  "Browse and edit logs in an interactive terminal UI. The --prev flag is ignored; use the day list to view past logs.",
	Example: `
		daylog tui
		daylog tui -p work
	`,

	Run: runCommand(func(cmd *cobra.Command, dl *daylog.DayLog) error {
		return tui.Run(dl, config.Project)
	}),
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
