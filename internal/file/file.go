package file

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
	"time"

	"github.com/notnmeyer/daylog-cli/internal/dateutil"
)

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

func GetLogs(projectPath string) ([]string, error) {
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
					log := convertLogToDisplayName(relPath)
					// insert at the front of the slice
					validFiles = slices.Insert(validFiles, 0, log)
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

func PreviousLog(path string) (string, error) {
	// get all the logs
	logs, err := GetLogs(path)
	if err != nil {
		log.Fatal(err)
	}

	// find the most recent existing log before today
	prev, exists := findPreviousLog(logs)
	if !exists {
		return "", fmt.Errorf("no previous log found")
	}

	return prev, nil
}

// converts 2025/12/02/log.md to 2025/12/02 which can be used directly when editing or showing a log
func convertLogToDisplayName(log string) string {
	split := strings.Split(log, string(filepath.Separator))
	return path.Join(split[0], split[1], split[2])
}

// logs is the list of keys of all logs. each log is YYYY/MM/DD, which the most recent date at index 0
func findPreviousLog(logs []string) (string, bool) {
	todayTime, err := time.Parse("2006/01/02", dateutil.GetCurrent().String())
	if err != nil {
		return "", false
	}

	for _, log := range logs {
		l, err := time.Parse("2006/01/02", log)
		if err == nil && l.Before(todayTime) {
			return log, true
		}
	}
	return "", false
}
