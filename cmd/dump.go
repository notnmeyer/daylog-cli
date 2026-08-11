package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/notnmeyer/daylog-cli/internal/date"
	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Concatenate all logs",
	Long:  "Concatenate every log in the project, oldest first",
	Example: `
		daylog dump
		daylog dump -r
		daylog dump --reverse
		daylog dump -o text
		daylog dump -p work
		daylog dump --since "1 week ago"
		daylog dump --since 2023/01/07 --until yesterday
	`,
	Args: cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := daylog.EnsureProjectPath(config.Project)
		if err != nil {
			log.Fatal(err)
		}

		format, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal(err)
		}

		reverse, err := cmd.Flags().GetBool("reverse")
		if err != nil {
			log.Fatal(err)
		}

		since, err := parseDateFlag(cmd, "since")
		if err != nil {
			log.Fatal(err)
		}

		until, err := parseDateFlag(cmd, "until")
		if err != nil {
			log.Fatal(err)
		}

		contents, err := daylog.Dump(projectPath, format, reverse, since, until)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(contents)
	},
}

// parseDateFlag reads a string flag and, if set, parses it with the same
// casual date syntax the rest of daylog accepts ("yesterday", "3 days ago",
// "2023/01/07", etc). an unset/empty flag returns a nil *time.Time.
func parseDateFlag(cmd *cobra.Command, name string) (*time.Time, error) {
	raw, err := cmd.Flags().GetString(name)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}

	t, err := date.Parse(raw, time.Now())
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func init() {
	rootCmd.AddCommand(dumpCmd)
	dumpCmd.Flags().StringP("output", "o", "markdown", "Format output")
	dumpCmd.Flags().BoolP("reverse", "r", false, "Newest logs first instead of oldest first")
	dumpCmd.Flags().String("since", "", "Only include logs on or after this date")
	dumpCmd.Flags().String("until", "", "Only include logs on or before this date")
}
