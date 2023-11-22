package cmd

import (
	"log"
	"os"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

type Config struct {
	Project string
}

var (
	version, commit string
	config          Config
)

var rootCmd = &cobra.Command{
	Use:   "daylog",
	Short: "A tool for keeping track of what you did today",
	Long:  "DayLog: Fighter of the Night Log! A tool for keeping track of what you did today, yesterday, and tomorrow",

	Run: func(cmd *cobra.Command, args []string) {
		dl, err := daylog.New(args, config.Project)
		if err != nil {
			log.Fatal(err)
		}

		if err := dl.Edit(); err != nil {
			log.Fatal(err)
		}
	},
}

// entrypoint
func Execute(v, c string) {
	version, commit = v, c
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// global flags
	rootCmd.PersistentFlags().StringVarP(&config.Project, "project", "p", "default", "The daylog project to use")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().StringVarP(&logDir, "dir", "d", "./", "--dir log/directory")
}
