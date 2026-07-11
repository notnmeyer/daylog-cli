package cmd

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/markusmobius/go-dateparser"
	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

type Config struct {
	Project string
	Append  string
}

var (
	version, commit string
	config          Config
)

var rootCmd = &cobra.Command{
	Use:   "daylog",
	Short: "A tool for keeping track of what you did today",
	Long:  "DayLog: Fighter of the Night Log! A tool for keeping track of what you did today, yesterday, and tomorrow",
	Example: `
		daylog
		daylog -- yesterday
		daylog -a "ate a burrito"
		daylog -a "ate a burrito" -- yesterday
		echo "note" | daylog
		daylog -p work -- 2023/01/07
	`,

	Run: runCommand(func(cmd *cobra.Command, dl *daylog.DayLog) error {
		if config.Append != "" {
			return dl.Append(daylog.FormatEntry(config.Append))
		}

		piped, err := stdinIsPiped()
		if err != nil {
			return err
		}

		if piped {
			content, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			return dl.Append(formatStdinContent(string(content)))
		}

		return dl.Edit()
	}),
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

	rootCmd.Flags().StringVarP(&config.Append, "append", "a", "", "Append a one-line entry to the log instead of opening an editor")
}

func runCommand(fn func(cmd *cobra.Command, dl *daylog.DayLog) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		projectPath, err := daylog.EnsureProjectPath(config.Project)
		if err != nil {
			log.Fatal(err)
		}

		date, err := parseDateFromArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		dl, err := daylog.New(date, projectPath)
		if err != nil {
			log.Fatal(err)
		}

		showPrevious, err := cmd.Root().PersistentFlags().GetBool("prev")
		if err != nil {
			log.Fatal(err)
		}

		if showPrevious {
			if err := dl.UsePrevious(time.Now()); err != nil {
				log.Fatal(err)
			}
		}

		if err := fn(cmd, dl); err != nil {
			log.Fatal(err)
		}
	}
}

func parseDateFromArgs(args []string) (time.Time, error) {
	cfg := &dateparser.Configuration{
		DefaultTimezone: time.Local,
	}

	if len(args) == 0 {
		return time.Now(), nil
	}

	d, err := dateparser.Parse(cfg, strings.Join(args, " "))
	if err != nil {
		return time.Time{}, err
	}
	return d.Time, nil
}

func formatStdinContent(content string) string {
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
