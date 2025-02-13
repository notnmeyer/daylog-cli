package cmd

import (
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/notnmeyer/daylog-cli/internal/file"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all log files",
	Long:  "list all log files relative to the project directory",
	Run: func(cmd *cobra.Command, args []string) {
		dl, err := daylog.New(args, config.Project)
		if err != nil {
			log.Fatal(err)
		}

		files, err := file.GetLogFiles(dl.ProjectPath)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			fmt.Println(convertLogToDisplayName(file))
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

// converts 2025/12/02/log.md to 2025/12/02 which can be used directly when editing or showing a log
func convertLogToDisplayName(log string) string {
	split := strings.Split(log, string(filepath.Separator))
	return path.Join(split[0], split[1], split[2])
}
