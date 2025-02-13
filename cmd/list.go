package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
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

		files, err := getLogFiles(dl.ProjectPath)
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

// converts 2025/12/02/log.md to 2025/12/02 which can be used directly when editing or showing a log
func convertLogToDisplayName(log string) string {
	split := strings.Split(log, string(filepath.Separator))
	return path.Join(split[0], split[1], split[2])
}

func getLogFiles(projectPath string) ([]string, error) {
	var validFiles []string
	err := filepath.Walk(projectPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == projectPath || info.Name() == ".git" {
			return nil
		}

		if info.Mode().IsRegular() {
			relPath, err := filepath.Rel(projectPath, path)
			if err != nil {
				return err
			}

			parts := strings.Split(relPath, string(filepath.Separator))
			if len(parts) > 0 && isYearDirectory(parts[0]) {
				if isValidFile(path) {
					// insert at the front of the slice
					validFiles = slices.Insert(validFiles, 0, relPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return validFiles, nil
}
