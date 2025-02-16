package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/notnmeyer/daylog-cli/internal/file"
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

		showPrevious, err := cmd.PersistentFlags().GetBool("prev")
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		if showPrevious {
			prev, err := file.PreviousLog(dl.ProjectPath)
			if err != nil {
				log.Fatal(err)
			}
			dl.Path = filepath.Join(dl.ProjectPath, prev, "log.md")
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
	rootCmd.PersistentFlags().Bool("prev", false, "Open the most recent log that isn't today's")
}
