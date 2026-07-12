package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/notnmeyer/daylog-cli/internal/clipboard"
	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/notnmeyer/daylog-cli/internal/editor"
	"github.com/notnmeyer/daylog-cli/internal/file"
	"github.com/notnmeyer/daylog-cli/internal/todo"
)

const dayFormat = "2006/01/02"

// daysLoadedMsg carries the newest-first day list plus noLogToday: whether
// today was force-prepended because it has no non-empty log. GetLogs lists only
// non-empty logs, so today ∉ GetLogs iff it is genuinely logless — the
// authoritative "empty today" signal, computed here where it's known and then
// carried into the model (len(preview)==0 alone can't tell a logless today from
// one whose preview simply hasn't been read yet)
type daysLoadedMsg struct {
	days       []string
	noLogToday bool
}
// dayRenderedMsg carries the day it rendered so a stale render (one that
// resolves after the user navigated away) can be dropped instead of painting
// the wrong day's content into the viewport
type dayRenderedMsg struct {
	day     string
	content string
}

// entryAppendedMsg / editorFinishedMsg carry the day that changed so the ledger
// can evict its stale preview from the cache before reloading
type entryAppendedMsg struct{ day string }
type editorFinishedMsg struct {
	day string
	err error
}
type copiedMsg struct{}
type clearStatusMsg struct{}
type projectsLoadedMsg struct{ projects []string }
type projectSwitchedMsg struct{ name, path string }

type todosLoadedMsg struct {
	day   string
	todos []todo.Item
}
type todoToggledMsg struct{}

type searchDebounceMsg struct{ seq int }
type searchResultsMsg struct {
	query   string
	matches []daylog.SearchMatch
}
type previewsLoadedMsg struct{ previews map[string][]string }
type errMsg struct{ err error }

// loadDays lists all logs for the project, ensuring today is present
// even when its log file doesn't exist yet
func loadDays(projectPath string, today time.Time) tea.Cmd {
	return func() tea.Msg {
		days, err := file.LogProvider{}.GetLogs(projectPath)
		if err != nil {
			return errMsg{err}
		}

		t := today.Format(dayFormat)
		noLogToday := !slices.Contains(days, t)
		if noLogToday {
			days = append([]string{t}, days...)
		}

		return daysLoadedMsg{days: days, noLogToday: noLogToday}
	}
}

// renderDay reads a day's log without creating it and renders it as markdown.
// a day with no file renders a warm invite instead of a bare placeholder so
// opening an unlogged day (especially today on launch) feels like a front door
func renderDay(md mdRenderer, projectPath, day string, width int, today time.Time) tea.Cmd {
	return func() tea.Msg {
		raw, err := editor.Read(logPath(projectPath, day))
		if err != nil {
			if !os.IsNotExist(err) {
				return errMsg{err}
			}
			raw = emptyDayInvite(day, today)
		}

		content, err := md.render(raw, width)
		if err != nil {
			return errMsg{err}
		}

		return dayRenderedMsg{day: day, content: content}
	}
}

// emptyDayInvite is the markdown shown when a day has no log yet. today gets a
// warmer "start today's log" framing; an older backfill day gets a plainer one
func emptyDayInvite(day string, today time.Time) string {
	primary, secondary := dayLabel(day, today)
	when := primary
	if secondary != "" && secondary != day {
		when = secondary
	}

	if isToday(day, today) {
		return fmt.Sprintf("## Nothing logged yet for %s\n\nPress `a` to append an entry or `e` to open your editor.", when)
	}
	return fmt.Sprintf("## Nothing logged on %s\n\nPress `a` to append an entry or `e` to open your editor.", when)
}

func isToday(day string, today time.Time) bool {
	return day == today.Format(dayFormat)
}

// previewMaxLines is how many content lines each ledger day block shows; a
// log with more gets a trailing "…" line
const previewMaxLines = 5

// previewEllipsis is the marker appended as the last line when a log has more
// than previewMaxLines content lines
const previewEllipsis = "…"

// loadPreviews reads the first few content lines of each given day's log for
// the ledger's multi-line day blocks. days is expected to be a bounded window
// (the rows visible in the ledger), NOT the whole history — GetLogs never reads
// content, so this is the only file read the ledger adds. keep it O(visible):
// never call it with every day in the project. already-cached days are skipped
// by the caller, so this only reads days it hasn't seen
func loadPreviews(projectPath string, days []string) tea.Cmd {
	return func() tea.Msg {
		previews := make(map[string][]string, len(days))
		for _, day := range days {
			lines, err := previewLines(logPath(projectPath, day))
			if err != nil {
				// a missing/unreadable log just has no preview; don't fail the
				// whole batch over one day (a log can vanish between listing
				// and reading, like search guards against)
				continue
			}
			previews[day] = lines
		}
		return previewsLoadedMsg{previews: previews}
	}
}

// previewLines returns up to previewMaxLines clean, plain-text lines from a
// log, reading only far enough to fill them rather than slurping the file.
// headings, blank lines, and code fences are skipped; list/todo markers and
// inline markdown are stripped so each line reads as human text. when the log
// has more content lines than the cap, a final "…" line is appended
func previewLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "```") {
			continue
		}
		cleaned := cleanPreview(line)
		if cleaned == "" {
			continue
		}
		if len(lines) == previewMaxLines {
			// there's at least one more content line than we show
			lines = append(lines, previewEllipsis)
			break
		}
		lines = append(lines, cleaned)
	}
	return lines, scanner.Err()
}

var (
	// leading list / task markers: "- ", "* ", "- [ ] ", "* [x] ", "[x] "
	listMarkerRe = regexp.MustCompile(`^\s*(?:[-*]\s+)?(?:\[[ xX]\]\s+)?`)
	linkRe       = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	emphasisRe   = regexp.MustCompile("[*_`~]{1,2}")
)

// cleanPreview turns a raw markdown line into plain preview text: it drops a
// leading list/task marker and unwraps inline emphasis, code, strikethrough,
// and links so "[x] TODO: ~~a cheeseburger?~~" reads as "TODO: a cheeseburger?"
func cleanPreview(line string) string {
	line = listMarkerRe.ReplaceAllString(line, "")
	line = linkRe.ReplaceAllString(line, "$1") // [text](url) -> text
	line = emphasisRe.ReplaceAllString(line, "")
	return strings.TrimSpace(line)
}

func appendEntry(projectPath, day, text string) tea.Cmd {
	return func() tea.Msg {
		dl, err := dayLogFor(day, projectPath)
		if err != nil {
			return errMsg{err}
		}

		if err := dl.Append(daylog.FormatEntry(text)); err != nil {
			return errMsg{err}
		}

		return entryAppendedMsg{day: day}
	}
}

// openEditor suspends the program and hands the terminal to $EDITOR.
// the command must be built before ExecProcess, so the log file is
// created here (which also carries over todos, matching the CLI)
func openEditor(projectPath, day string) tea.Cmd {
	dl, err := dayLogFor(day, projectPath)
	if err != nil {
		return func() tea.Msg { return errMsg{err} }
	}

	c, err := dl.EditorCommand()
	if err != nil {
		return func() tea.Msg { return errMsg{err} }
	}

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{day: day, err: err}
	})
}

// copyDay copies a day's raw log to the clipboard without creating it
func copyDay(projectPath, day string) tea.Cmd {
	return func() tea.Msg {
		raw, err := editor.Read(logPath(projectPath, day))
		if err != nil {
			if os.IsNotExist(err) {
				return errMsg{fmt.Errorf("nothing to copy for %s", day)}
			}
			return errMsg{err}
		}

		if err := clipboard.Copy([]byte(raw)); err != nil {
			return errMsg{err}
		}

		return copiedMsg{}
	}
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := daylog.ListProjects()
		if err != nil {
			return errMsg{err}
		}
		return projectsLoadedMsg{projects: projects}
	}
}

func switchProject(name string) tea.Cmd {
	return func() tea.Msg {
		path, err := daylog.EnsureProjectPath(name)
		if err != nil {
			return errMsg{err}
		}
		return projectSwitchedMsg{name: name, path: path}
	}
}

func loadTodos(projectPath, day string) tea.Cmd {
	return func() tea.Msg {
		dl, err := dayLogFor(day, projectPath)
		if err != nil {
			return errMsg{err}
		}

		todos, err := dl.Todos()
		if err != nil {
			return errMsg{err}
		}

		return todosLoadedMsg{day: day, todos: todos}
	}
}

func toggleTodo(projectPath, day string, item todo.Item) tea.Cmd {
	return func() tea.Msg {
		dl, err := dayLogFor(day, projectPath)
		if err != nil {
			return errMsg{err}
		}

		if err := dl.ToggleTodoItem(item); err != nil {
			return errMsg{err}
		}

		return todoToggledMsg{}
	}
}

// debounceSearch fires after a pause in typing; stale sequence numbers
// are dropped by the update loop
func debounceSearch(seq int) tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
		return searchDebounceMsg{seq: seq}
	})
}

// runSearch is case-insensitive: interactive search shouldn't demand
// exact casing (the CLI stays case-sensitive with its -i flag)
func runSearch(projectPath, query string) tea.Cmd {
	return func() tea.Msg {
		matches, err := daylog.Search(projectPath, query, true)
		if err != nil {
			return errMsg{err}
		}
		return searchResultsMsg{query: query, matches: matches}
	}
}

func logPath(projectPath, day string) string {
	return filepath.Join(projectPath, day, "log.md")
}

func dayLogFor(day, projectPath string) (*daylog.DayLog, error) {
	t, err := time.Parse(dayFormat, day)
	if err != nil {
		return nil, err
	}
	return daylog.New(t, projectPath)
}
