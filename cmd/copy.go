package cmd

import (
	"fmt"
	"log"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
	"golang.design/x/clipboard"
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy the specified log to the clipboard",
	Long:  "Copy the specified log to the clipboard",
	Run: func(cmd *cobra.Command, args []string) {
		dl, err := daylog.New(args, config.Project)
		if err != nil {
			log.Fatal(err)
		}

		logContents, err := dl.Show("text")
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		clipboard.Write(clipboard.FmtText, []byte(logContents))
		fmt.Println("Copied to clipboard.")
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
