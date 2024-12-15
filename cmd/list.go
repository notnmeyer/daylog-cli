package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

		var validFiles []string
		err = filepath.Walk(dl.ProjectPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if path == dl.ProjectPath || info.Name() == ".git" {
				return nil
			}

			if info.Mode().IsRegular() {
				relPath, err := filepath.Rel(dl.ProjectPath, path)
				if err != nil {
					return err
				}

				parts := strings.Split(relPath, string(filepath.Separator))
				if len(parts) > 0 && isYearDirectory(parts[0]) {
					if isValidFile(path) {
						validFiles = append(validFiles, relPath)
					}
				}
			}
			return nil
		})

		for _, file := range validFiles {
			fmt.Println(file)
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

func isYearDirectory(s string) bool {
	if len(s) != 4 {
		return false
	}

	if _, err := strconv.Atoi(s); err != nil {
		return false
	}

	return true
}

func isValidFile(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return len(content) > 0 && strings.TrimSpace(string(content)) != ""
}
