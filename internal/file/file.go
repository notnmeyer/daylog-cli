package file

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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

func GetLogFiles(projectPath string) ([]string, error) {
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
