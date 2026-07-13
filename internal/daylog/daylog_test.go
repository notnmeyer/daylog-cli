package daylog

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/notnmeyer/daylog-cli/internal/todo"
)

// writePrevLog creates a log file for the given date under projectPath.
func writePrevLog(t *testing.T, projectPath, dateDir, content string) {
	t.Helper()
	dir := filepath.Join(projectPath, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating prev log dir: %v", err)
	}
	path := filepath.Join(dir, "log.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing prev log: %v", err)
	}
}

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

// regression: log files must be owner-only (0600), not world-readable
func TestAppend_CreatesFileWith0600(t *testing.T) {
	dl := testDayLog(t)

	if err := dl.Append("hello"); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	info, err := os.Stat(dl.Path)
	if err != nil {
		t.Fatalf("stat %q: %v", dl.Path, err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("perm = %o, want 0600", perm)
	}
}

func TestCarryOverTodos(t *testing.T) {
	tests := []struct {
		name         string
		prevContent  string
		expectedFile string
	}{
		{
			name:         "no todos in previous log",
			prevContent:  "# 2025/12/01\n\ndid some work\n",
			expectedFile: "# 2025/12/02\n\n",
		},
		{
			name:         "todos are copied to new log",
			prevContent:  "# 2025/12/01\n\n- TODO: write tests\n- TODO: fix bug\n- done thing\n",
			expectedFile: "# 2025/12/02\n\n- TODO: write tests\n- TODO: fix bug\n",
		},
		{
			name:         "only list items mentioning TODO are copied",
			prevContent:  "# 2025/12/01\n\n- TODO: write tests\nthinking about TODOs\n  - TODO: indented note\n- did a TODO\n",
			expectedFile: "# 2025/12/02\n\n- TODO: write tests\n- did a TODO\n",
		},
		{
			name:         "no previous log",
			prevContent:  "",
			expectedFile: "# 2025/12/02\n\n",
		},
		{
			name:         "checked todos are not carried over",
			prevContent:  "# 2025/12/01\n\n- [ ] TODO: still open\n- [x] TODO: finished thing\n",
			expectedFile: "# 2025/12/02\n\n- [ ] TODO: still open\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := testDayLog(t)

			if tt.prevContent != "" {
				writePrevLog(t, dl.ProjectPath, "2025/12/01", tt.prevContent)
			}

			if err := createIfMissing(dl); err != nil {
				t.Fatalf("createIfMissing() error = %v", err)
			}

			got := readFile(t, dl.Path)
			if got != tt.expectedFile {
				t.Errorf("file contents =\n%q\nwant\n%q", got, tt.expectedFile)
			}
		})
	}
}

func TestSearch(t *testing.T) {
	project := t.TempDir()
	writePrevLog(t, project, "2025/12/01", "# 2025/12/01\n\n- ate a burrito\n- wrote tests\n")
	writePrevLog(t, project, "2025/12/02", "# 2025/12/02\n\n- burrito again\n- reviewed a PR\n")

	tests := []struct {
		name       string
		query      string
		ignoreCase bool
		expected   []SearchMatch
	}{
		{
			name:  "matches across logs, most recent first",
			query: "burrito",
			expected: []SearchMatch{
				{Date: "2025/12/02", Line: "- burrito again"},
				{Date: "2025/12/01", Line: "- ate a burrito"},
			},
		},
		{
			name:  "single match",
			query: "PR",
			expected: []SearchMatch{
				{Date: "2025/12/02", Line: "- reviewed a PR"},
			},
		},
		{
			name:     "no matches",
			query:    "taco",
			expected: nil,
		},
		{
			name:     "match is case-sensitive by default",
			query:    "Burrito",
			expected: nil,
		},
		{
			name:       "ignore case matches different casing",
			query:      "Burrito",
			ignoreCase: true,
			expected: []SearchMatch{
				{Date: "2025/12/02", Line: "- burrito again"},
				{Date: "2025/12/01", Line: "- ate a burrito"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Search(project, tt.query, tt.ignoreCase)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			if !slices.Equal(got, tt.expected) {
				t.Errorf("Search() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func TestSanitizeProject(t *testing.T) {
	valid := []struct {
		input string
		want  string
	}{
		{"default", "default"},
		{"work", "work"},
		{"  work  ", "work"},
		{"my-project_2023", "my-project_2023"},
		{"日本語", "日本語"},
		{"foo.bar", "foo.bar"},
	}
	for _, tt := range valid {
		t.Run("valid/"+tt.input, func(t *testing.T) {
			got, err := sanitizeProject(tt.input)
			if err != nil {
				t.Fatalf("sanitizeProject(%q) error = %v, want nil", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("sanitizeProject(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}

	invalid := []string{
		"",
		"   ",
		"..",
		".",
		"../../etc",
		"../etc",
		"/etc",
		"/etc/passwd",
		"work/client-x",
		"foo/../bar",
		`a\b`,
		`a\..\b`,
	}
	for _, input := range invalid {
		t.Run("invalid/"+input, func(t *testing.T) {
			if _, err := sanitizeProject(input); err == nil {
				t.Errorf("sanitizeProject(%q) error = nil, want error", input)
			}
		})
	}
}

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

	t.Run("default project is allowed", func(t *testing.T) {
		base := t.TempDir()
		got, err := projectPathAt(base, "default")
		if err != nil {
			t.Fatalf("projectPathAt: %v", err)
		}
		want := filepath.Join(base, "daylog", "default")
		if got != want {
			t.Errorf("projectPathAt() = %q, want %q", got, want)
		}
	})

	t.Run("rejects traversal and stays within base", func(t *testing.T) {
		base := t.TempDir()
		got, err := projectPathAt(base, "../../etc")
		if err == nil {
			t.Fatalf("projectPathAt(%q) error = nil, want error", "../../etc")
		}
		// on error the returned path must be empty, never an escaping path
		if got != "" {
			daylogDir := filepath.Join(base, "daylog")
			if got != daylogDir && !strings.HasPrefix(got, daylogDir+string(filepath.Separator)) {
				t.Errorf("projectPathAt() = %q, escaped %q", got, daylogDir)
			}
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

	// security: validation must run before MkdirAll so a traversal name never
	// creates a directory outside the daylog dir
	t.Run("rejects traversal without creating a directory", func(t *testing.T) {
		base := t.TempDir()

		if _, err := ensureProjectPathAt(base, "../../evil"); err == nil {
			t.Fatal("ensureProjectPathAt(../../evil) error = nil, want error")
		}

		escaped := filepath.Join(base, "..", "..", "evil")
		if _, err := os.Stat(escaped); !os.IsNotExist(err) {
			t.Errorf("ensureProjectPathAt created %q; nothing should exist outside the daylog dir", escaped)
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

func TestFormatEntry(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain message gets a list marker",
			input:    "ate a burrito",
			expected: "- ate a burrito",
		},
		{
			name:     "surrounding whitespace is trimmed",
			input:    "  ate a burrito  ",
			expected: "- ate a burrito",
		},
		{
			name:     "existing list marker is preserved",
			input:    "- ate a burrito",
			expected: "- ate a burrito",
		},
		{
			name:     "todo entry becomes a checkbox",
			input:    "TODO: buy milk",
			expected: "- [ ] TODO: buy milk",
		},
		{
			name:     "todo list item becomes a checkbox",
			input:    "- TODO: buy milk",
			expected: "- [ ] TODO: buy milk",
		},
		{
			name:     "existing checkbox is preserved",
			input:    "- [ ] TODO: buy milk",
			expected: "- [ ] TODO: buy milk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatEntry(tt.input)
			if result != tt.expected {
				t.Errorf("FormatEntry(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEditorCommand(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")

	dl := testDayLog(t)

	cmd, err := dl.EditorCommand()
	if err != nil {
		t.Fatal(err)
	}

	if len(cmd.Args) != 2 || cmd.Args[0] != "vim" || cmd.Args[1] != dl.Path {
		t.Errorf("expected [vim %s], got %v", dl.Path, cmd.Args)
	}
	if cmd.Stdin != nil || cmd.Stdout != nil || cmd.Stderr != nil {
		t.Error("expected no stdio wired so tea.ExecProcess can attach its own")
	}

	// the log should have been created with its header
	content := readFile(t, dl.Path)
	if content != "# 2025/12/02\n\n" {
		t.Errorf("expected header-only log, got %q", content)
	}
}

func TestListProjectsAt(t *testing.T) {
	base := t.TempDir()
	for _, project := range []string{"default", "work"} {
		if err := os.MkdirAll(filepath.Join(base, "daylog", project), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// stray files must be excluded
	if err := os.WriteFile(filepath.Join(base, "daylog", "notes.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	projects, err := listProjectsAt(base)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"default", "work"}
	if len(projects) != len(want) {
		t.Fatalf("expected %v, got %v", want, projects)
	}
	for i := range want {
		if projects[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, projects)
		}
	}
}

func TestTodosAndToggle(t *testing.T) {
	dl := testDayLog(t)

	t.Run("missing log has no todos", func(t *testing.T) {
		todos, err := dl.Todos()
		if err != nil {
			t.Fatal(err)
		}
		if len(todos) != 0 {
			t.Errorf("expected no todos, got %+v", todos)
		}
	})

	if err := os.WriteFile(dl.Path, []byte("# 2025/12/02\n\n- TODO: buy tortillas\n- [x] TODO: eat a burrito\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("parses open and checked todos", func(t *testing.T) {
		todos, err := dl.Todos()
		if err != nil {
			t.Fatal(err)
		}
		if len(todos) != 2 {
			t.Fatalf("expected 2 todos, got %d", len(todos))
		}
		if todos[0].Text != "TODO: buy tortillas" || todos[0].Done {
			t.Errorf("unexpected first todo: %+v", todos[0])
		}
		if todos[1].Text != "TODO: eat a burrito" || !todos[1].Done {
			t.Errorf("unexpected second todo: %+v", todos[1])
		}
	})

	t.Run("ToggleTodoItem rewrites the matching line in place", func(t *testing.T) {
		if err := dl.ToggleTodoItem(todo.Item{Line: 2, Text: "TODO: buy tortillas"}); err != nil {
			t.Fatal(err)
		}
		got := readFile(t, dl.Path)
		want := "# 2025/12/02\n\n- [x] TODO: buy tortillas\n- [x] TODO: eat a burrito\n"
		if got != want {
			t.Errorf("file contents =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("ToggleTodoItem errors when the line no longer holds that todo", func(t *testing.T) {
		if err := dl.ToggleTodoItem(todo.Item{Line: 0, Text: "TODO: buy tortillas"}); err == nil {
			t.Error("expected an error for a stale item")
		}
	})
}
