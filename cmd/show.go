package cmd

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/notnmeyer/daylog-cli/internal/file"
	"github.com/spf13/cobra"
)

type ShowConfig struct {
	Output string
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
			log.Fatalf("%s", err.Error())
		}

		showPrevious, err := cmd.Root().PersistentFlags().GetBool("prev")
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		if showPrevious {
			prev, err := file.PreviousLog(dl.ProjectPath, file.LogProvider{})
			if err != nil {
				log.Fatal(err)
			}
			dl.Path = filepath.Join(dl.ProjectPath, prev, "log.md")
		}

		logContents, err := dl.Show(format)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		fmt.Println(string(logContents))
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.PersistentFlags().StringP("output", "o", "markdown", "Format output")
}
