package file

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/notnmeyer/daylog-cli/internal/dateutil"
)

type mockLogProvider struct {
	logs []string
	err  error
}

func (m *mockLogProvider) GetLogs(string) ([]string, error) {
	return m.logs, m.err
}

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

	files, err := LogProvider{}.GetLogs(tmpDir)
	if err != nil {
		t.Fatalf("getLogFiles() error = %v", err)
	}

	expectedFiles := []string{"2025/12/02"}
	if len(files) != len(expectedFiles) || files[0] != expectedFiles[0] {
		t.Errorf("getLogFiles() = %v, want %v", files, expectedFiles)
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

func TestPreviousLog(t *testing.T) {
	dateutil.GetCurrent = func() dateutil.Date {
		return dateutil.Date{
			Year:  2025,
			Month: 12,
			Day:   2,
		}
	}

	tests := []struct {
		name     string
		logs     []string
		expected string
		err      error
	}{
		{
			name:     "previous log exists",
			logs:     []string{"2025/12/01", "2025/11/30", "2025/11/29"},
			expected: "2025/12/01",
			err:      nil,
		},
		{
			name:     "no previous log found",
			logs:     []string{"2025/12/02", "2025/12/03"},
			expected: "",
			err:      fmt.Errorf("no previous log found"),
		},
		{
			name:     "no previous logs at all",
			logs:     []string{},
			expected: "",
			err:      fmt.Errorf("no previous log found"),
		},
	}

	for _, tt := range tests {
		provider := &mockLogProvider{
			logs: tt.logs,
		}
		t.Run(tt.name, func(t *testing.T) {
			prev, err := PreviousLog("mock/path", provider)
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("findPreviousLog(%v) = found: %v, want: %v", tt.logs, err, tt.err)
			}
			if prev != tt.expected {
				t.Errorf("findPreviousLog(%v) = %q, want %q", tt.logs, prev, tt.expected)
			}
		})
	}
}

func TestFindPreviousLog(t *testing.T) {
	dateutil.GetCurrent = func() dateutil.Date {
		return dateutil.Date{
			Year:  2025,
			Month: 12,
			Day:   2,
		}
	}

	tests := []struct {
		name     string
		logs     []string
		expected string
		found    bool
	}{
		{
			name:     "previous log exists",
			logs:     []string{"2025/12/01", "2025/11/30", "2025/11/29"},
			expected: "2025/12/01",
			found:    true,
		},
		{
			name:     "no previous logs found",
			logs:     []string{"2025/12/02", "2025/12/03"},
			expected: "",
			found:    false,
		},
		{
			name:     "invalid log key",
			logs:     []string{"invalid-log", "2025/11/28", "2025/11/27"},
			expected: "2025/11/28",
			found:    true,
		},
		{
			name:     "no logs at all",
			logs:     []string{},
			expected: "",
			found:    false,
		},
		{
			name:     "only future logs",
			logs:     []string{"2025/12/03", "2025/12/04"},
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev, found := findPreviousLog(tt.logs)
			if found != tt.found {
				t.Errorf("findPreviousLog(%v) = found: %v, want: %v", tt.logs, found, tt.found)
			}
			if prev != tt.expected {
				t.Errorf("findPreviousLog(%v) = %q, want %q", tt.logs, prev, tt.expected)
			}
		})
	}
}
