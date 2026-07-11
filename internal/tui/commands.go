package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/notnmeyer/daylog-cli/internal/clipboard"
	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/notnmeyer/daylog-cli/internal/editor"
	"github.com/notnmeyer/daylog-cli/internal/file"
)

const dayFormat = "2006/01/02"

type daysLoadedMsg struct{ days []string }
type dayRenderedMsg struct{ content string }
type entryAppendedMsg struct{}
type editorFinishedMsg struct{ err error }
type copiedMsg struct{}
type clearStatusMsg struct{}
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
		if !slices.Contains(days, t) {
			days = append([]string{t}, days...)
		}

		return daysLoadedMsg{days: days}
	}
}

// renderDay reads a day's log without creating it and renders it as markdown
func renderDay(md mdRenderer, projectPath, day string, width int) tea.Cmd {
	return func() tea.Msg {
		raw, err := editor.Read(logPath(projectPath, day))
		if err != nil {
			if !os.IsNotExist(err) {
				return errMsg{err}
			}
			raw = "*no entries yet*"
		}

		content, err := md.render(raw, width)
		if err != nil {
			return errMsg{err}
		}

		return dayRenderedMsg{content: content}
	}
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

		return entryAppendedMsg{}
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
		return editorFinishedMsg{err: err}
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
