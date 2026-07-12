package cmd

import (
	"fmt"
	"os"

	"github.com/notnmeyer/daylog-cli/internal/clipboard"
	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy the specified log to the clipboard",
	Long:  "Copy the specified log to the clipboard",
	Example: `
		daylog copy
		daylog copy -- yesterday
		daylog copy --prev
		daylog copy -p work -- 2023/01/07
    `,

	Run: runCommand(func(cmd *cobra.Command, dl *daylog.DayLog) error {
		logContents, err := dl.Show("text")
		if err != nil {
			return err
		}

		if err := clipboard.Copy([]byte(logContents)); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Copied to clipboard.")
		return nil
	}),
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
