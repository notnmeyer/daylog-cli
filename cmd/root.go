package cmd

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/fang"
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

		if err := applyPrevFlag(cmd, dl); err != nil {
			log.Fatal(err)
		}

		piped, err := stdinIsPiped()
		if err != nil {
			log.Fatal(err)
		}

		if piped {
			content, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			formatted := formatStdinContent(string(content))
			if err := dl.Append(formatted); err != nil {
				log.Fatal(err)
			}
			return
		}

		if err := dl.Edit(); err != nil {
			log.Fatal(err)
		}
	},
}

// entrypoint
func Execute(v, c string) {
	version, commit = v, c

	err := fang.Execute(context.TODO(), rootCmd,
		fang.WithoutManpage(),
		fang.WithVersion(version),
		fang.WithCommit(commit),
	)

	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// global flags
	rootCmd.PersistentFlags().StringVarP(&config.Project, "project", "p", "default", "The daylog project to use")
	rootCmd.PersistentFlags().Bool("prev", false, "Operate on the most recent log that isn't today's")
}

func applyPrevFlag(cmd *cobra.Command, dl *daylog.DayLog) error {
	showPrevious, err := cmd.PersistentFlags().GetBool("prev")
	if err != nil {
		return err
	}
	if showPrevious {
		prev, err := file.PreviousLog(dl.ProjectPath, file.LogProvider{})
		if err != nil {
			return err
		}
		dl.Path = filepath.Join(dl.ProjectPath, prev, "log.md")
	}
	return nil
}

func formatStdinContent(content string) string {
	// if a string like "hello\nworld" is piped, the markdown formatter (the default) will smash
	// it together like "helloworld". so if there are \n's just make it a code block so the line
	// breaks render correctly.
	content = strings.TrimRight(content, "\n")
	if strings.Contains(content, "\n") {
		return "```\n" + content + "\n```"
	}
	return content
}

func stdinIsPiped() (bool, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false, err
	}
	// terminals set ModeCharDevice. pipes don't. so if the bit is zero, we have piped input
	return (stat.Mode() & os.ModeCharDevice) == 0, nil
}
