package cmd

import (
	"log"
	"os"

	"github.com/arl/dirtree"
	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		dl, err := daylog.New(args, config.Project)
		if err != nil {
			log.Fatal(err)
		}

		err = dirtree.Write(
			os.Stdout,
			dl.ProjectPath,
			dirtree.Type("f"),
			dirtree.ExcludeRoot,
			dirtree.PrintMode(0),
			// uhhhhhh, kind of lame
			dirtree.Ignore(".git/*"),
			dirtree.Ignore(".git/*/*"),
			dirtree.Ignore(".git/*/*/*"),
			dirtree.Ignore(".git/*/*/*/*"),
		)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
