package cmd

import (
	"fmt"
	"log"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display today's log",
	Long:  "Display today's log",
	Run: func(cmd *cobra.Command, args []string) {
		dl, err := daylog.New(args)
		if err != nil {
			log.Fatal(err)
		}

		render, _ := cmd.Flags().GetBool("render")

		logContents, err := dl.Show(render)
		if err != nil {
			log.Fatalf(err.Error())
		}

		fmt.Println(string(logContents))
	},
}

func init() {
	rootCmd.AddCommand(showCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// showCmd.PersistentFlags().String("foo", "", "A help for foo")

	showCmd.Flags().BoolP("render", "r", false, "Render markdown in terminal")
}
