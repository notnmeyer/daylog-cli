package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display today's log",
	Long:  "Display today's log",
	Run: func(cmd *cobra.Command, args []string) {
		t, err := parseDateFromArgs(args)
		if err != nil {
			log.Fatalf("Couldn't parse date: %s", err.Error())
		}

		logFile, err := setup(*t)
		if err != nil {
			log.Fatal(err)
		}

		fileContents, err := os.ReadFile(logFile)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Println(string(fileContents))
	},
}

func init() {
	rootCmd.AddCommand(showCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// showCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// showCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
