package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
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
		projectPath, err := daylog.EnsureProjectPath(config.Project)
		if err != nil {
			log.Fatal(err)
		}

		dl, err := daylog.New(args, projectPath)
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
			if err := dl.UsePrevious(time.Now()); err != nil {
				log.Fatal(err)
			}
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
