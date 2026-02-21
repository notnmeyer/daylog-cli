package daylog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testDayLog(t *testing.T) *DayLog {
	t.Helper()
	dir := t.TempDir()
	date := time.Date(2025, 12, 2, 0, 0, 0, 0, time.UTC)
	return &DayLog{
		Path:        filepath.Join(dir, "log.md"),
		ProjectPath: dir,
		Date:        &date,
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	return string(b)
}

func TestAppend(t *testing.T) {
	tests := []struct {
		name          string
		existing      *string // nil means file doesn't exist yet
		content       string
		expectedFinal string
	}{
		{
			name:          "new file gets header then content",
			existing:      nil,
			content:       "hello world",
			expectedFinal: "# 2025/12/02\n\nhello world\n",
		},
		{
			name:          "existing file with trailing newline",
			existing:      strPtr("# 2025/12/02\n\nexisting entry\n"),
			content:       "new entry",
			expectedFinal: "# 2025/12/02\n\nexisting entry\nnew entry\n",
		},
		{
			name:          "existing file without trailing newline",
			existing:      strPtr("# 2025/12/02\n\nexisting entry"),
			content:       "new entry",
			expectedFinal: "# 2025/12/02\n\nexisting entry\nnew entry\n",
		},
		{
			name:          "trailing newlines in content are normalized to one",
			existing:      strPtr("# 2025/12/02\n\n"),
			content:       "hello\n\n",
			expectedFinal: "# 2025/12/02\n\nhello\n",
		},
		{
			name:          "multi-line content (code block) appended intact",
			existing:      strPtr("# 2025/12/02\n\n"),
			content:       "```\nline one\nline two\n```",
			expectedFinal: "# 2025/12/02\n\n```\nline one\nline two\n```\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := testDayLog(t)

			if tt.existing != nil {
				if err := os.WriteFile(dl.Path, []byte(*tt.existing), 0644); err != nil {
					t.Fatalf("writing existing file: %v", err)
				}
			}

			if err := dl.Append(tt.content); err != nil {
				t.Fatalf("Append() error = %v", err)
			}

			got := readFile(t, dl.Path)
			if got != tt.expectedFinal {
				t.Errorf("file contents =\n%q\nwant\n%q", got, tt.expectedFinal)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
