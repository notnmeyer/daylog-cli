package cmd

import (
	"fmt"
	"log"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search all logs for a string",
	Long:  "Search all logs for lines containing a string, printing the matching line and the date of its log",
	Example: `
		daylog search "burrito"
		daylog search -i "burrito"
		daylog search -p work "standup"
	`,
	Args: cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := daylog.EnsureProjectPath(config.Project)
		if err != nil {
			log.Fatal(err)
		}

		ignoreCase, err := cmd.Flags().GetBool("ignore-case")
		if err != nil {
			log.Fatal(err)
		}

		matches, err := daylog.Search(projectPath, args[0], ignoreCase)
		if err != nil {
			log.Fatal(err)
		}

		for _, m := range matches {
			fmt.Printf("%s: %s\n", m.Date, m.Line)
		}
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().BoolP("ignore-case", "i", false, "Case-insensitive search")
}
