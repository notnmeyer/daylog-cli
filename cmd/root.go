package cmd

import (
	"log"
	"os"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "daylog",
	Short: "A tool for keeping track of what you did today",
	Long:  `DayLog: Fighter of the Night Log!`,

	Run: func(cmd *cobra.Command, args []string) {
		dl, err := daylog.New(args)
		if err != nil {
			log.Fatal(err)
		}

		if err := dl.Edit(); err != nil {
			log.Fatal(err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// func init() {
// Here you will define your flags and configuration settings.
// Cobra supports persistent flags, which, if defined here,
// will be global for your application.

// rootCmd.PersistentFlags().StringVarP(&logDir, "dir", "d", dataDir, "--dir log/directory")

// Cobra also supports local flags, which will only run
// when this action is called directly.
// rootCmd.Flags().StringVarP(&logDir, "dir", "d", "./", "--dir log/directory")
// }
