package daylog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

func TestProjectPathAt(t *testing.T) {
	t.Run("returns path under base", func(t *testing.T) {
		base := t.TempDir()
		got, err := projectPathAt(base, "myproject")
		if err != nil {
			t.Fatalf("projectPathAt: %v", err)
		}
		want := filepath.Join(base, "daylog", "myproject")
		if got != want {
			t.Errorf("projectPathAt() = %q, want %q", got, want)
		}
	})

	t.Run("does not create directories", func(t *testing.T) {
		base := t.TempDir()
		nonExistentBase := filepath.Join(base, "no-such-parent")

		got, err := projectPathAt(nonExistentBase, "myproject")
		if err != nil {
			t.Fatalf("projectPathAt: %v", err)
		}

		want := filepath.Join(nonExistentBase, "daylog", "myproject")
		if got != want {
			t.Errorf("projectPathAt() = %q, want %q", got, want)
		}

		if _, err := os.Stat(nonExistentBase); !os.IsNotExist(err) {
			t.Errorf("projectPathAt() created base %q; should be read-only", nonExistentBase)
		}
		if _, err := os.Stat(want); !os.IsNotExist(err) {
			t.Errorf("projectPathAt() created %q; should be read-only", want)
		}
	})
}

func TestEnsureProjectPathAt(t *testing.T) {
	t.Run("creates directory with 0755", func(t *testing.T) {
		base := t.TempDir()

		got, err := ensureProjectPathAt(base, "myproject")
		if err != nil {
			t.Fatalf("ensureProjectPathAt: %v", err)
		}
		want := filepath.Join(base, "daylog", "myproject")
		if got != want {
			t.Errorf("ensureProjectPathAt() = %q, want %q", got, want)
		}

		info, err := os.Stat(want)
		if err != nil {
			t.Fatalf("stat %q: %v", want, err)
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", want)
		}
		if perm := info.Mode().Perm(); perm != 0755 {
			t.Errorf("perm = %o, want 0755", perm)
		}
	})

	t.Run("idempotent on existing directory", func(t *testing.T) {
		base := t.TempDir()

		first, err := ensureProjectPathAt(base, "myproject")
		if err != nil {
			t.Fatalf("first ensureProjectPathAt: %v", err)
		}
		second, err := ensureProjectPathAt(base, "myproject")
		if err != nil {
			t.Fatalf("second ensureProjectPathAt: %v", err)
		}
		if first != second {
			t.Errorf("paths differ: %q vs %q", first, second)
		}
	})
}

// regression: --prev previously created today's tree as a side effect
func TestNew_DoesNotCreateDateSubdir(t *testing.T) {
	base := t.TempDir()
	today := time.Now()

	dl, err := New(today, base)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	yearDir := filepath.Join(base, strconv.Itoa(today.Year()))
	if _, err := os.Stat(yearDir); !os.IsNotExist(err) {
		t.Errorf("New() created date subdir %q; should be lazy", yearDir)
	}

	wantLogFile := filepath.Join(base, strconv.Itoa(today.Year()),
		fmt.Sprintf("%02d", int(today.Month())),
		fmt.Sprintf("%02d", today.Day()),
		"log.md")
	if dl.Path != wantLogFile {
		t.Errorf("dl.Path = %q, want %q", dl.Path, wantLogFile)
	}

	if dl.ProjectPath != base {
		t.Errorf("dl.ProjectPath = %q, want %q", dl.ProjectPath, base)
	}
}

func TestUsePrevious(t *testing.T) {
	t.Run("mutates Path to the most recent log before now", func(t *testing.T) {
		project := t.TempDir()

		prevDir := filepath.Join(project, "2025/12/01")
		if err := os.MkdirAll(prevDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(prevDir, "log.md"), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}

		dl := &DayLog{
			Path:        filepath.Join(project, "2025/12/02/log.md"),
			ProjectPath: project,
		}
		now := time.Date(2025, 12, 2, 0, 0, 0, 0, time.UTC)

		if err := dl.UsePrevious(now); err != nil {
			t.Fatalf("UsePrevious: %v", err)
		}

		want := filepath.Join(project, "2025/12/01/log.md")
		if dl.Path != want {
			t.Errorf("dl.Path = %q, want %q", dl.Path, want)
		}
	})

	t.Run("returns error when no previous log exists", func(t *testing.T) {
		project := t.TempDir()
		dl := &DayLog{
			Path:        filepath.Join(project, "2025/12/02/log.md"),
			ProjectPath: project,
		}
		now := time.Date(2025, 12, 2, 0, 0, 0, 0, time.UTC)

		if err := dl.UsePrevious(now); err == nil {
			t.Errorf("UsePrevious() err = nil, want non-nil")
		}
		// path should not have been mutated on error.
		want := filepath.Join(project, "2025/12/02/log.md")
		if dl.Path != want {
			t.Errorf("dl.Path = %q, want unchanged %q", dl.Path, want)
		}
	})
}
