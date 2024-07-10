package cmd

import (
	"fmt"
	"log"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var outputFormats = []string{
	"markdown", "md",
	"text",
	"web",
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display today's log",
	Long:  "Display today's log",
	Run: func(cmd *cobra.Command, args []string) {
		dl, err := daylog.New(args, config.Project)
		if err != nil {
			log.Fatal(err)
		}

		format, err := cmd.PersistentFlags().GetString("output")
		if err != nil {
			log.Fatalf(err.Error())
		}

		if !validOutputFormat(format) {
			log.Fatalf("output must be one of %v\n", outputFormats)
		}

		logContents, err := dl.Show(format)
		if err != nil {
			log.Fatalf(err.Error())
		}

		fmt.Println(string(logContents))
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.PersistentFlags().StringP("output", "o", "markdown", "Format output")
}

func validOutputFormat(format string) bool {
	for _, v := range outputFormats {
		if v == format {
			return true
		}
	}
	return false
}
