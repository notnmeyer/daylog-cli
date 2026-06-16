package file

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type FileLogProvider interface {
	GetLogs(string) ([]string, error)
}

type LogProvider struct{}

func NewLogProvider() LogProvider {
	return LogProvider{}
}

func (LogProvider) GetLogs(projectPath string) ([]string, error) {
	var validFiles []string
	err := filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == projectPath {
			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}

		relPath, err := filepath.Rel(projectPath, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) == 0 || !isYearDirectory(parts[0]) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return nil
		}

		validFiles = append(validFiles, convertLogToDisplayName(relPath))
		return nil
	})
	if err != nil {
		return nil, err
	}

	// most recent first; YYYY/MM/DD strings sort chronologically with regular comparison
	slices.SortFunc(validFiles, func(a, b string) int {
		return strings.Compare(b, a)
	})
	return validFiles, nil
}

func PreviousLog(path string, provider FileLogProvider, now time.Time) (string, error) {
	// get all the logs
	logs, err := provider.GetLogs(path)
	if err != nil {
		return "", err
	}

	// find the most recent existing log before today
	prev, exists := findPreviousLog(logs, now)
	if !exists {
		return "", fmt.Errorf("no previous log found")
	}

	return prev, nil
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

// converts 2025/12/02/log.md to 2025/12/02 which can be used directly when editing or showing a log
func convertLogToDisplayName(log string) string {
	split := strings.Split(log, string(filepath.Separator))
	return path.Join(split[0], split[1], split[2])
}

// logs is the list of keys of all logs. each log is YYYY/MM/DD, which the most recent date at index 0
func findPreviousLog(logs []string, now time.Time) (string, bool) {
	today := now.Format("2006/01/02")

	for _, log := range logs {
		if _, err := time.Parse("2006/01/02", log); err != nil {
			continue
		}
		if log < today {
			return log, true
		}
	}
	return "", false
}
