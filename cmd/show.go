package cmd

import (
	"fmt"
	"log"

	"github.com/notnmeyer/daylog-cli/internal/dateutil"
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

		showPrevious, err := cmd.PersistentFlags().GetBool("prev")
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		// override the log
		if showPrevious {
			// build today's log key
			today := dateutil.GetCurrent()

			// get all the logs
			logs, err := file.GetLogs(dl.ProjectPath)
			if err != nil {
				log.Fatalf("%s", err.Error())
			}

			// find the most recent existing log before today
			prev, exists := getPreviousLog(logs, fmt.Sprintf("%s", today))
			if !exists {
				log.Fatalf("no previous log found")
			}

			// TODO: fix this shit
			dl.Path = dl.ProjectPath + "/" + prev + "/log.md"
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
	showCmd.PersistentFlags().Bool("prev", false, "Open the most recent log that isn't today's")
}

func getPreviousLog(logs []string, target string) (string, bool) {
	for i, v := range logs {
		if v == target && i+1 < len(logs) {
			return logs[i+1], true
		}
	}
	return "", false
}
