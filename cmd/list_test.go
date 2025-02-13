package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsYearDirectory(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"2024", true},
		{"abcd", false},
		{"123", false},
		{"20245", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isYearDirectory(tt.input)
			if result != tt.expected {
				t.Errorf("isYearDirectory(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testlog.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if isValidFile(tmpFile.Name()) {
		t.Errorf("expected false for empty file")
	}

	content := "ate a burrito"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	if !isValidFile(tmpFile.Name()) {
		t.Errorf("expected true for non-empty file")
	}
}

func TestConvertLogToDisplayName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2025/12/02/log.md", "2025/12/02"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertLogToDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("convertLogToDisplayName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetLogFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "2025/12/02"), 0755)

	// valid file
	validLogPath := filepath.Join(tmpDir, "2025/12/02/log.md")
	err := os.WriteFile(validLogPath, []byte("ate a burrito"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// empty file
	invalidLogPath := filepath.Join(tmpDir, "2025/12/03/empty.md")
	os.WriteFile(invalidLogPath, []byte(""), 0644)

	// empty file
	os.Mkdir(filepath.Join(tmpDir, "emptydir"), 0755)

	files, err := getLogFiles(tmpDir)
	if err != nil {
		t.Fatalf("getLogFiles() error = %v", err)
	}

	expectedFiles := []string{"2025/12/02/log.md"}
	if len(files) != len(expectedFiles) || files[0] != expectedFiles[0] {
		t.Errorf("getLogFiles() = %v, want %v", files, expectedFiles)
	}
}
