package daylog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/notnmeyer/daylog-cli/internal/editor"
	"github.com/notnmeyer/daylog-cli/internal/file"
	"github.com/notnmeyer/daylog-cli/internal/output-formatter"
	"github.com/notnmeyer/daylog-cli/internal/todo"
)

type DayLog struct {
	// the complete path to the log file
	Path string

	// the path to the project directory
	ProjectPath string

	// the date of the log
	Date *time.Time
}

func projectPathAt(base, project string) (string, error) {
	return filepath.Abs(filepath.Join(base, "daylog", project))
}

func ensureProjectPathAt(base, project string) (string, error) {
	p, err := projectPathAt(base, project)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return "", err
	}
	return p, nil
}

func listProjectsAt(base string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(base, "daylog"))
	if err != nil {
		return nil, err
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			projects = append(projects, entry.Name())
		}
	}
	return projects, nil
}

// ListProjects returns the names of all existing projects, sorted
func ListProjects() ([]string, error) {
	return listProjectsAt(xdg.DataHome)
}

func ProjectPath(project string) (string, error) {
	return projectPathAt(xdg.DataHome, project)
}

func EnsureProjectPath(project string) (string, error) {
	return ensureProjectPathAt(xdg.DataHome, project)
}

// new performs no filesystem i/o; date subdirs are created lazily by the methods that need them
func New(t time.Time, projectPath string) (*DayLog, error) {
	year, month, day := t.Year(), int(t.Month()), t.Day()
	logFile := filepath.Join(
		projectPath,
		strconv.Itoa(year),
		fmt.Sprintf("%02d", month),
		fmt.Sprintf("%02d", day),
		"log.md",
	)

	return &DayLog{
		Path:        logFile,
		ProjectPath: projectPath,
		Date:        &t,
	}, nil
}

// append content to the log for the specified date
func (d *DayLog) Append(content string) error {
	if err := createIfMissing(d); err != nil {
		return err
	}

	existing, err := os.ReadFile(d.Path)
	if err != nil {
		return err
	}

	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		existing = append(existing, '\n')
	}

	content = strings.TrimRight(content, "\n") + "\n"

	if err := os.WriteFile(d.Path, append(existing, []byte(content)...), 0644); err != nil {
		return err
	}

	return nil
}

// FormatEntry normalizes a one-line entry into a markdown list item.
// entries that read like todos become unchecked checkbox items
func FormatEntry(msg string) string {
	msg = strings.TrimSpace(msg)

	if formatted, ok := todo.FormatEntry(msg); ok {
		return formatted
	}

	if strings.HasPrefix(msg, "- ") {
		return msg
	}
	return "- " + msg
}

// EditorCommand creates the log if missing and returns an unstarted
// editor command with no stdio wired, for callers like tea.ExecProcess
func (d *DayLog) EditorCommand() (*exec.Cmd, error) {
	if err := createIfMissing(d); err != nil {
		return nil, err
	}

	return editor.Command(d.Path)
}

// edit the log for the specified date
func (d *DayLog) Edit() error {
	if err := createIfMissing(d); err != nil {
		return err
	}

	if err := editor.Open(d.Path); err != nil {
		return err
	}

	return nil
}

func (d *DayLog) Show(format string) (string, error) {
	if err := createIfMissing(d); err != nil {
		return "", err
	}

	contents, err := editor.Read(d.Path)
	if err != nil {
		return "", err
	}

	contents, err = outputformatter.Format(format, contents)
	if err != nil {
		return "", err
	}

	return contents, nil
}

// Todos parses the log's TODO/DONE lines. a missing log has no todos
func (d *DayLog) Todos() ([]todo.Item, error) {
	content, err := os.ReadFile(d.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return todo.Parse(string(content)), nil
}

// ToggleTodo flips the TODO/DONE prefix on the given 0-based line
func (d *DayLog) ToggleTodo(line int) error {
	content, err := os.ReadFile(d.Path)
	if err != nil {
		return err
	}

	updated, err := todo.Toggle(string(content), line)
	if err != nil {
		return err
	}

	return os.WriteFile(d.Path, []byte(updated), 0644)
}

// ToggleTodoItem flips item's checkbox, verifying the line still holds
// that todo so a stale picker index can't toggle the wrong entry
func (d *DayLog) ToggleTodoItem(item todo.Item) error {
	content, err := os.ReadFile(d.Path)
	if err != nil {
		return err
	}

	updated, err := todo.ToggleMatching(string(content), item)
	if err != nil {
		return err
	}

	return os.WriteFile(d.Path, []byte(updated), 0644)
}

// usePrevious mutates d.Path to point at the most recent log before now.
func (d *DayLog) UsePrevious(now time.Time) error {
	prev, err := file.PreviousLog(d.ProjectPath, file.LogProvider{}, now)
	if err != nil {
		return err
	}
	d.Path = filepath.Join(d.ProjectPath, prev, "log.md")
	return nil
}

func createIfMissing(d *DayLog) error {
	_, err := os.Stat(d.Path)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(d.Path), 0755); err != nil {
		return err
	}

	f, err := os.Create(d.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	year, month, day := d.Date.Year(), int(d.Date.Month()), d.Date.Day()
	header := fmt.Sprintf("# %d/%02d/%02d\n\n", year, month, day)
	if _, err := f.WriteString(header); err != nil {
		return err
	}

	if todos := carryOverTodos(d.ProjectPath, *d.Date); len(todos) > 0 {
		_, err = f.WriteString(strings.Join(todos, "\n") + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

type SearchMatch struct {
	// the date of the log containing the match, as YYYY/MM/DD
	Date string

	// the line that matched
	Line string
}

// search every log in the project for lines containing query, most recent log first
func Search(projectPath, query string, ignoreCase bool) ([]SearchMatch, error) {
	logs, err := file.NewLogProvider().GetLogs(projectPath)
	if err != nil {
		return nil, err
	}

	if ignoreCase {
		query = strings.ToLower(query)
	}

	var matches []SearchMatch
	for _, log := range logs {
		content, err := os.ReadFile(filepath.Join(projectPath, log, "log.md"))
		if err != nil {
			// a log removed between listing and reading shouldn't sink the
			// whole search; skip it and return what we can
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, line := range strings.Split(string(content), "\n") {
			haystack := line
			if ignoreCase {
				haystack = strings.ToLower(line)
			}
			if strings.Contains(haystack, query) {
				matches = append(matches, SearchMatch{Date: log, Line: line})
			}
		}
	}
	return matches, nil
}

// carryOverTodos reads the log before `before` and returns its unfinished todos.
func carryOverTodos(projectPath string, before time.Time) []string {
	prev, err := file.PreviousLog(projectPath, file.NewLogProvider(), before)
	if err != nil {
		return nil
	}

	prevPath := filepath.Join(projectPath, prev, "log.md")
	content, err := os.ReadFile(prevPath)
	if err != nil {
		return nil
	}

	return todo.Unfinished(string(content))
}
