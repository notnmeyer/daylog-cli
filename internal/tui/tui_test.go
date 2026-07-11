package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func seedLog(t *testing.T, projectPath, day, content string) {
	t.Helper()

	dir := filepath.Join(projectPath, day)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "log.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// execCmd runs a tea.Cmd and flattens any batches into a message slice
func execCmd(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()

	if cmd == nil {
		return nil
	}

	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, execCmd(t, c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func findDayRendered(t *testing.T, msgs []tea.Msg) (dayRenderedMsg, bool) {
	t.Helper()

	for _, msg := range msgs {
		if m, ok := msg.(dayRenderedMsg); ok {
			return m, true
		}
	}
	return dayRenderedMsg{}, false
}

func TestLoadDays(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name string
		seed []string
		want []string
	}{
		{
			name: "prepends today when its log is missing",
			seed: []string{"2026/07/08", "2026/07/09"},
			want: []string{"2026/07/10", "2026/07/09", "2026/07/08"},
		},
		{
			name: "no duplicate when today's log exists",
			seed: []string{"2026/07/09", "2026/07/10"},
			want: []string{"2026/07/10", "2026/07/09"},
		},
		{
			name: "empty project still shows today",
			seed: nil,
			want: []string{"2026/07/10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectPath := t.TempDir()
			for _, day := range tt.seed {
				seedLog(t, projectPath, day, "- entry\n")
			}

			msg := loadDays(projectPath, today)()
			loaded, ok := msg.(daysLoadedMsg)
			if !ok {
				t.Fatalf("expected daysLoadedMsg, got %T", msg)
			}

			if !slices.Equal(loaded.days, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, loaded.days)
			}
		})
	}
}

func TestDayLogFor(t *testing.T) {
	tests := []struct {
		name    string
		day     string
		wantErr bool
	}{
		{name: "valid day", day: "2026/07/10"},
		{name: "invalid day", day: "not-a-date", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectPath := t.TempDir()

			dl, err := dayLogFor(tt.day, projectPath)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			want := filepath.Join(projectPath, tt.day, "log.md")
			if dl.Path != want {
				t.Errorf("expected %s, got %s", want, dl.Path)
			}
		})
	}
}

func TestRenderDayMissingFile(t *testing.T) {
	projectPath := t.TempDir()

	msg := renderDay(newMDRenderer(), projectPath, "2026/07/10", 80)()
	rendered, ok := msg.(dayRenderedMsg)
	if !ok {
		t.Fatalf("expected dayRenderedMsg, got %T", msg)
	}
	if !strings.Contains(rendered.content, "no entries yet") {
		t.Errorf("expected placeholder content, got %q", rendered.content)
	}
}

func newTestModel(t *testing.T, projectPath string, today time.Time) Model {
	t.Helper()

	m := New(projectPath, "default", today)

	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mm.(Model)

	msg := loadDays(projectPath, today)()
	mm, cmd := m.Update(msg)
	m = mm.(Model)

	// apply the initial render of the selected day
	if rendered, ok := findDayRendered(t, execCmd(t, cmd)); ok {
		mm, _ = m.Update(rendered)
		m = mm.(Model)
	}

	return m
}

func TestSelectionChangeRendersDay(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "- ate a burrito\n")

	m := newTestModel(t, projectPath, today)

	// today is selected; j moves to 2026/07/09
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = mm.(Model)

	day, _ := m.selectedDay()
	if day != "2026/07/09" {
		t.Fatalf("expected selection 2026/07/09, got %s", day)
	}

	rendered, ok := findDayRendered(t, execCmd(t, cmd))
	if !ok {
		t.Fatal("expected a dayRenderedMsg after selection change")
	}
	if !strings.Contains(rendered.content, "burrito") {
		t.Errorf("expected rendered log content, got %q", rendered.content)
	}
}

func TestTabTogglesFocus(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	m := newTestModel(t, t.TempDir(), today)

	if m.focus != focusDays {
		t.Fatal("expected initial focus on day list")
	}

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = mm.(Model)
	if m.focus != focusViewport {
		t.Error("expected focus on viewport after tab")
	}

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = mm.(Model)
	if m.focus != focusDays {
		t.Error("expected focus back on day list after second tab")
	}
}

func TestQuitKeys(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{name: "q", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{name: "esc", key: tea.KeyMsg{Type: tea.KeyEsc}},
		{name: "ctrl+c", key: tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, t.TempDir(), today)

			_, cmd := m.Update(tt.key)
			if cmd == nil {
				t.Fatal("expected quit cmd")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Error("expected tea.QuitMsg")
			}
		})
	}
}

func typeString(t *testing.T, m Model, s string) Model {
	t.Helper()

	for _, r := range s {
		mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mm.(Model)
	}
	return m
}

func TestAppendFlow(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()

	m := newTestModel(t, projectPath, today)

	// a enters input mode
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = mm.(Model)
	if m.mode != modeInput {
		t.Fatal("expected input mode after a")
	}

	m = typeString(t, m, "ate a burrito")

	// enter submits and returns to browse
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.mode != modeBrowse {
		t.Fatal("expected browse mode after enter")
	}
	if m.input.Value() != "" {
		t.Error("expected input to be reset")
	}

	msgs := execCmd(t, cmd)
	if len(msgs) != 1 {
		t.Fatalf("expected one message, got %d", len(msgs))
	}
	if _, ok := msgs[0].(entryAppendedMsg); !ok {
		t.Fatalf("expected entryAppendedMsg, got %T", msgs[0])
	}

	content := string(readLog(t, projectPath, "2026/07/10"))
	if !strings.Contains(content, "- ate a burrito") {
		t.Errorf("expected formatted entry in log, got %q", content)
	}
	if !strings.HasPrefix(content, "# 2026/07/10") {
		t.Errorf("expected header in new log, got %q", content)
	}
}

func TestInputEscCancels(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()

	m := newTestModel(t, projectPath, today)

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = mm.(Model)
	m = typeString(t, m, "discarded")

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)

	if m.mode != modeBrowse {
		t.Fatal("expected esc to return to browse mode, not quit")
	}
	if cmd != nil {
		t.Error("expected no cmd on cancel")
	}
	if m.input.Value() != "" {
		t.Error("expected input to be reset on cancel")
	}
	if _, err := os.Stat(logPath(projectPath, "2026/07/10")); !os.IsNotExist(err) {
		t.Error("expected no log file after cancel")
	}
}

func TestEditKeyCreatesLogAndReturnsCmd(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")

	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()

	m := newTestModel(t, projectPath, today)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("expected an exec cmd for the editor")
	}

	content := string(readLog(t, projectPath, "2026/07/10"))
	if !strings.HasPrefix(content, "# 2026/07/10") {
		t.Errorf("expected log created with header before editor opens, got %q", content)
	}
}

func TestCopyKey(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/10", "- copy me\n")

	m := newTestModel(t, projectPath, today)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected a copy cmd")
	}
	// don't execute it — that would touch the real clipboard
}

func TestCopyMissingLogReportsError(t *testing.T) {
	projectPath := t.TempDir()

	msg := copyDay(projectPath, "2026/07/10")()
	e, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("expected errMsg for missing log, got %T", msg)
	}
	if !strings.Contains(e.err.Error(), "nothing to copy") {
		t.Errorf("expected friendly message, got %q", e.err.Error())
	}
}

func TestCopiedStatusSetsAndClears(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	m := newTestModel(t, t.TempDir(), today)

	mm, cmd := m.Update(copiedMsg{})
	m = mm.(Model)
	if m.status != "Copied to clipboard." {
		t.Errorf("expected copied status, got %q", m.status)
	}
	if cmd == nil {
		t.Fatal("expected a clear-status tick cmd")
	}

	mm, _ = m.Update(clearStatusMsg{})
	m = mm.(Model)
	if m.status != "" {
		t.Errorf("expected status cleared, got %q", m.status)
	}
}

func TestProjectSwitcherFlow(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	oldPath := t.TempDir()
	seedLog(t, oldPath, "2026/07/09", "- old project entry\n")

	m := newTestModel(t, oldPath, today)

	// p opens the picker
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = mm.(Model)
	if m.mode != modeProjects {
		t.Fatal("expected projects mode after p")
	}
	if cmd == nil {
		t.Fatal("expected loadProjects cmd")
	}

	// simulate the load; current project should be pre-selected
	mm, _ = m.Update(projectsLoadedMsg{projects: []string{"default", "work"}})
	m = mm.(Model)
	if item, _ := m.picker.SelectedItem().(pickerItem); string(item) != "default" {
		t.Errorf("expected current project pre-selected, got %q", item)
	}

	// navigate to "work"; simulate the switch completing
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = mm.(Model)
	if item, _ := m.picker.SelectedItem().(pickerItem); string(item) != "work" {
		t.Fatalf("expected work selected, got %q", item)
	}

	newPath := t.TempDir()
	seedLog(t, newPath, "2026/07/05", "- work project entry\n")
	mm, cmd = m.Update(projectSwitchedMsg{name: "work", path: newPath})
	m = mm.(Model)

	if m.mode != modeBrowse {
		t.Error("expected browse mode after switch")
	}
	if m.project != "work" || m.projectPath != newPath {
		t.Errorf("expected project swapped, got %s %s", m.project, m.projectPath)
	}

	// the reload should list the new project's days with today selected
	msgs := execCmd(t, cmd)
	if len(msgs) != 1 {
		t.Fatalf("expected one message, got %d", len(msgs))
	}
	loaded, ok := msgs[0].(daysLoadedMsg)
	if !ok {
		t.Fatalf("expected daysLoadedMsg, got %T", msgs[0])
	}
	if !slices.Equal(loaded.days, []string{"2026/07/10", "2026/07/05"}) {
		t.Errorf("expected new project days, got %v", loaded.days)
	}

	mm, _ = m.Update(loaded)
	m = mm.(Model)
	if day, _ := m.selectedDay(); day != "2026/07/10" {
		t.Errorf("expected selection reset to today, got %s", day)
	}
}

func TestPickerEscCancels(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	m := newTestModel(t, t.TempDir(), today)

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = mm.(Model)

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)

	if m.mode != modeBrowse {
		t.Error("expected esc to close picker, not quit")
	}
	if cmd != nil {
		t.Error("expected no cmd on cancel")
	}
	if m.project != "default" {
		t.Errorf("expected project unchanged, got %s", m.project)
	}
}

func readLog(t *testing.T, projectPath, day string) []byte {
	t.Helper()

	b, err := os.ReadFile(logPath(projectPath, day))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDayLabel(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name          string
		day           string
		wantPrimary   string
		wantSecondary string
	}{
		{name: "today", day: "2026/07/10", wantPrimary: "Jul 10", wantSecondary: "today"},
		{name: "yesterday", day: "2026/07/09", wantPrimary: "Jul 09", wantSecondary: "yesterday"},
		{name: "within a week", day: "2026/07/07", wantPrimary: "Jul 07", wantSecondary: "3 days ago"},
		{name: "six days ago", day: "2026/07/04", wantPrimary: "Jul 04", wantSecondary: "6 days ago"},
		{name: "a week ago", day: "2026/07/03", wantPrimary: "Jul 03", wantSecondary: "2026/07/03"},
		{name: "other year", day: "2025/12/31", wantPrimary: "Dec 31", wantSecondary: "2025/12/31"},
		{name: "unparseable", day: "garbage", wantPrimary: "garbage", wantSecondary: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary, secondary := dayLabel(tt.day, today)
			if primary != tt.wantPrimary {
				t.Errorf("expected primary %q, got %q", tt.wantPrimary, primary)
			}
			if secondary != tt.wantSecondary {
				t.Errorf("expected secondary %q, got %q", tt.wantSecondary, secondary)
			}
		})
	}
}
