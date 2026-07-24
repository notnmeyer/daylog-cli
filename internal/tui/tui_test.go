package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/notnmeyer/daylog-cli/internal/todo"
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
		name           string
		seed           []string
		want           []string
		wantNoLogToday bool
	}{
		{
			name:           "prepends today when its log is missing",
			seed:           []string{"2026/07/08", "2026/07/09"},
			want:           []string{"2026/07/10", "2026/07/09", "2026/07/08"},
			wantNoLogToday: true,
		},
		{
			name:           "no duplicate when today's log exists",
			seed:           []string{"2026/07/09", "2026/07/10"},
			want:           []string{"2026/07/10", "2026/07/09"},
			wantNoLogToday: false,
		},
		{
			name:           "empty project still shows today",
			seed:           nil,
			want:           []string{"2026/07/10"},
			wantNoLogToday: true,
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
			// the whole ledger empty-today decision pivots on this bit
			if loaded.noLogToday != tt.wantNoLogToday {
				t.Errorf("expected noLogToday=%v, got %v", tt.wantNoLogToday, loaded.noLogToday)
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
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()

	// today with no file renders the warm "start today's log" invite
	msg := renderDay(newMDRenderer(), projectPath, "2026/07/10", 80, today)()
	rendered, ok := msg.(dayRenderedMsg)
	if !ok {
		t.Fatalf("expected dayRenderedMsg, got %T", msg)
	}
	if !strings.Contains(rendered.content, "Nothing logged") {
		t.Errorf("expected warm empty content, got %q", rendered.content)
	}
	if !strings.Contains(rendered.content, "append") {
		t.Errorf("expected an append prompt in the invite, got %q", rendered.content)
	}

	// an older empty day gets the plainer framing, not "for today"
	msg = renderDay(newMDRenderer(), projectPath, "2026/07/03", 80, today)()
	rendered, _ = msg.(dayRenderedMsg)
	if strings.Contains(rendered.content, "for") {
		t.Errorf("expected older-day framing without \"for today\", got %q", rendered.content)
	}
}

// newTestModel builds a ready model landed in browse mode on today. the app
// launches on the ledger; most tests exercise browse keys, so this opens
// today (enter on the pinned row) to get there, matching what a user does
func newTestModel(t *testing.T, projectPath string, today time.Time) Model {
	t.Helper()

	m := newLedgerModel(t, projectPath, today)

	// enter on the pinned today row opens browse
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.mode != modeBrowse {
		t.Fatalf("expected browse mode after opening today, got %d", m.mode)
	}
	if rendered, ok := findDayRendered(t, execCmd(t, cmd)); ok {
		mm, _ = m.Update(rendered)
		m = mm.(Model)
	}

	return m
}

// newLedgerModel builds a ready model sitting on the ledger (the launch state)
func newLedgerModel(t *testing.T, projectPath string, today time.Time) Model {
	t.Helper()

	m := New(projectPath, "default", today)

	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mm.(Model)

	// Init loads the days; apply the load and any preview cmds it fires
	for _, msg := range execCmd(t, loadDays(projectPath, today)) {
		mm, cmd := m.Update(msg)
		m = mm.(Model)
		for _, follow := range execCmd(t, cmd) {
			mm, _ = m.Update(follow)
			m = mm.(Model)
		}
	}

	if m.mode != modeLedger {
		t.Fatalf("expected launch on the ledger, got mode %d", m.mode)
	}
	return m
}

func TestDaySteppingRendersDay(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "- ate a burrito\n")

	m := newTestModel(t, projectPath, today)

	// today is selected; h steps to the older 2026/07/09
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mm.(Model)

	day, _ := m.selectedDay()
	if day != "2026/07/09" {
		t.Fatalf("expected selection 2026/07/09, got %s", day)
	}

	rendered, ok := findDayRendered(t, execCmd(t, cmd))
	if !ok {
		t.Fatal("expected a dayRenderedMsg after stepping days")
	}
	if !strings.Contains(rendered.content, "burrito") {
		t.Errorf("expected rendered log content, got %q", rendered.content)
	}

	// h at the oldest day is a no-op
	mm, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mm.(Model)
	if day, _ := m.selectedDay(); day != "2026/07/09" {
		t.Errorf("expected selection clamped at oldest day, got %s", day)
	}
	if cmd != nil {
		t.Error("expected no cmd when clamped")
	}

	// l steps back to today
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = mm.(Model)
	if day, _ := m.selectedDay(); day != "2026/07/10" {
		t.Errorf("expected selection back on today, got %s", day)
	}
}

// the browse header shows a "N of M" position chip (replacing the cut spine);
// "1 of M" is the newest day, and it drops on a narrow terminal
func TestBrowseHeaderPositionChip(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "- nine\n")
	seedLog(t, projectPath, "2026/07/08", "- eight\n")

	m := newTestModel(t, projectPath, today) // browse on today (newest), width 80
	if h := stripANSI(m.headerView()); !strings.Contains(h, "1 of 3") {
		t.Errorf("expected '1 of 3' for the newest day, got %q", h)
	}

	// step to the oldest day
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mm.(Model)
	if day, _ := m.selectedDay(); day == "2026/07/08" {
		if h := stripANSI(m.headerView()); !strings.Contains(h, "3 of 3") {
			t.Errorf("expected '3 of 3' on the oldest day, got %q", h)
		}
	}

	// a narrow terminal drops the chip but keeps the date
	mm, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = mm.(Model)
	h := stripANSI(m.headerView())
	if strings.Contains(h, " of ") {
		t.Errorf("expected the chip dropped on a narrow terminal, got %q", h)
	}
	if !strings.Contains(h, "daylog") {
		t.Errorf("expected the header still present when narrow, got %q", h)
	}
}

// stripANSI removes SGR escapes for header assertions
func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		if r == 0x1b {
			esc = true
			continue
		}
		if esc {
			if r == 'm' {
				esc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// regression: a render that resolves after the user navigated to another day
// must NOT paint the old day's content under the new day's header. this was the
// "header says Jul 06 but content is Jul 09" desync
func TestStaleRenderDropped(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "# 2026/07/09\n\n- day nine content\n")
	seedLog(t, projectPath, "2026/07/08", "# 2026/07/08\n\n- day eight content\n")

	m := newTestModel(t, projectPath, today) // browse on today (07/10)

	// step to 07/09 and grab its (in-flight) render msg without applying it yet
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mm.(Model)
	nineRender, ok := findDayRendered(t, execCmd(t, cmd))
	if !ok {
		t.Fatal("expected a render for 07/09")
	}

	// the user keeps navigating to 07/08 before 07/09's render lands
	mm, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mm.(Model)
	if day, _ := m.selectedDay(); day != "2026/07/08" {
		t.Fatalf("expected to be on 07/08, got %s", day)
	}
	eightRender, _ := findDayRendered(t, execCmd(t, cmd))
	mm, _ = m.Update(eightRender)
	m = mm.(Model)

	// now 07/09's stale render finally arrives — it must be dropped, not painted
	mm, _ = m.Update(nineRender)
	m = mm.(Model)

	view := m.vp.View()
	if strings.Contains(view, "nine") {
		t.Error("stale 07/09 render painted into the 07/08 viewport")
	}
	if !strings.Contains(view, "eight") {
		t.Error("expected the current day's (07/08) content in the viewport")
	}
}

// a render must not paint while the user is on the ledger (no viewport shown)
func TestRenderDroppedInLedgerMode(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "# 2026/07/09\n\n- nine\n")

	m := newTestModel(t, projectPath, today) // browse on today
	// step to 07/09, capture its render, but go back to the ledger first
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mm.(Model)
	nineRender, _ := findDayRendered(t, execCmd(t, cmd))
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // back to ledger
	m = mm.(Model)
	if m.mode != modeLedger {
		t.Fatal("expected ledger after esc")
	}
	// the in-flight render lands while on the ledger — must be dropped
	before := m.vp.View()
	mm, _ = m.Update(nineRender)
	m = mm.(Model)
	if m.vp.View() != before {
		t.Error("a render painted the viewport while on the ledger")
	}
}

func TestSelectDayInsertsMissingDay(t *testing.T) {
	m := Model{days: []string{"2026/07/10", "2026/07/08", "2026/07/05"}}

	// an existing day just moves the cursor
	m.selectDay("2026/07/08")
	if m.dayIdx != 1 {
		t.Errorf("expected existing day at idx 1, got %d", m.dayIdx)
	}

	// a day the list never carried (e.g. a search hit GetLogs filtered
	// out) is inserted in newest-first order and selected
	m.selectDay("2026/07/09")
	if day, _ := m.selectedDay(); day != "2026/07/09" {
		t.Fatalf("expected to land on the inserted day, got %s", day)
	}
	want := []string{"2026/07/10", "2026/07/09", "2026/07/08", "2026/07/05"}
	if !slices.Equal(m.days, want) {
		t.Errorf("expected sorted insert %v, got %v", want, m.days)
	}
}

func TestLedgerFilterAndOpen(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "- yesterday entry\n")
	seedLog(t, projectPath, "2026/06/17", "- june entry\n")

	// launch lands on the ledger listing today + the two logged days
	m := newLedgerModel(t, projectPath, today)
	if item, _ := m.picker.SelectedItem().(pickerItem); item.value != "2026/07/10" {
		t.Errorf("expected today pre-selected on the ledger, got %q", item.value)
	}

	// / starts the in-place filter; typing narrows fuzzily
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	if !m.dayFilter.Focused() {
		t.Fatal("expected the filter to be focused after /")
	}
	m = typeString(t, m, "jun")

	// the first row is the "＋ New day…" affordance; the june day matches below it
	found := false
	for _, it := range m.picker.Items() {
		if p, ok := it.(pickerItem); ok && p.value == "2026/06/17" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the june day among the filtered rows")
	}

	// move past the New day row onto the june match and open it
	m.moveLedgerCursor(1)
	if item, _ := m.picker.SelectedItem().(pickerItem); item.value != "2026/06/17" {
		t.Fatalf("expected june day selected, got %q", item.value)
	}

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.mode != modeBrowse {
		t.Fatal("expected browse mode after enter")
	}
	if day, _ := m.selectedDay(); day != "2026/06/17" {
		t.Fatalf("expected jump to 2026/06/17, got %s", day)
	}
	rendered, ok := findDayRendered(t, execCmd(t, cmd))
	if !ok {
		t.Fatal("expected a dayRenderedMsg after jump")
	}
	// glamour styles each word separately, so match a single word
	if !strings.Contains(rendered.content, "june") {
		t.Errorf("expected june log rendered, got %q", rendered.content)
	}
}

// filterDays returns the day values of the current filtered ledger rows
// (skipping the ＋ New day… affordance)
func filterDays(m Model) []string {
	var days []string
	for _, it := range m.picker.Items() {
		if p, ok := it.(pickerItem); ok && p.kind == itemDay {
			days = append(days, p.value)
		}
	}
	return days
}

// runFilterSearch drives the debounce+search after typing, applying the
// resulting searchResultsMsg, and returns the model
func runFilterSearch(t *testing.T, m Model) Model {
	t.Helper()
	// fire the latest debounce, which returns a runSearch cmd
	mm, cmd := m.Update(searchDebounceMsg{seq: m.filterSeq})
	m = mm.(Model)
	for _, msg := range execCmd(t, cmd) {
		if msg == nil {
			continue
		}
		mm, _ = m.Update(msg)
		m = mm.(Model)
	}
	return m
}

// THE regression: typing content ("burrito") must surface days whose LOGS
// contain it — even when the match is NOT in the shown preview lines — and must
// NOT appear on the instant date-fuzzy pass alone. this is the exact bug.
func TestLedgerFilterMatchesContent(t *testing.T) {
	today := time.Date(2026, 7, 12, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	// a day whose burrito line is on line 6 — BEYOND the 5-line preview window
	seedLog(t, projectPath, "2026/07/10",
		"# 2026/07/10\n\n- one\n- two\n- three\n- four\n- five\n- got a burrito finally\n")
	// a day that does NOT mention burrito at all
	seedLog(t, projectPath, "2026/06/17", "# 2026/06/17\n\n- shipped the parser\n")

	m := newLedgerModel(t, projectPath, today)

	// open the filter and type content
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	m = typeString(t, m, "burrito")

	// INSTANT pass (date-fuzzy only): "burrito" matches no date label, so the
	// day is absent — this is exactly the reported failure before the fix
	if got := filterDays(m); len(got) != 0 {
		t.Errorf("date-fuzzy alone should not match 'burrito', got %v", got)
	}

	// after the debounced content search resolves, the burrito day appears...
	m = runFilterSearch(t, m)
	if _, ok := m.contentMatches["2026/07/10"]; !ok {
		t.Error("expected the content search to match 2026/07/10")
	}
	// ...and the row shows the LINE that matched (so you see why), even though
	// the burrito line is beyond the day's normal 5-line preview window
	if line := m.contentMatches["2026/07/10"]; !strings.Contains(line, "burrito") {
		t.Errorf("expected the matched line stored for the row, got %q", line)
	}
	got := filterDays(m)
	if len(got) != 1 || got[0] != "2026/07/10" {
		t.Fatalf("expected only the burrito day, got %v", got)
	}
	// ...and the non-matching day stays out (the filter actually filters)
	for _, d := range got {
		if d == "2026/06/17" {
			t.Error("a non-matching day leaked into the content filter")
		}
	}
	// the ＋ New day… affordance is still row 0
	if first, _ := m.picker.Items()[0].(pickerItem); first.kind != itemNewDay {
		t.Error("expected the New day row at the top of a filtered ledger")
	}
	// the matched row shows the matching LINE, not a bare "·" tick — even
	// though that line is beyond the day's normal preview window
	anchor := ledgerAnchor(t, m, "2026/07/10")
	if !strings.Contains(anchor.text, "burrito") {
		t.Errorf("expected the filtered row to show the matching line, got %q", anchor.text)
	}
}

// the filter unions DATE matches with CONTENT matches, deduped, newest-first
func TestLedgerFilterUnionsDateAndContent(t *testing.T) {
	today := time.Date(2026, 7, 12, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	// content match only (mentions the word, date label doesn't)
	seedLog(t, projectPath, "2026/07/09", "# 2026/07/09\n\n- ate a burrito\n")
	// a plain day used to prove non-matches are excluded
	seedLog(t, projectPath, "2026/06/17", "# 2026/06/17\n\n- unrelated work\n")

	m := newLedgerModel(t, projectPath, today)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	m = typeString(t, m, "burrito")
	m = runFilterSearch(t, m)

	got := filterDays(m)
	if len(got) != 1 || got[0] != "2026/07/09" {
		t.Fatalf("expected the burrito day only, got %v", got)
	}

	// now filter by a DATE — instant, no content needed
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // clear
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	m = typeString(t, m, "jun")
	got = filterDays(m)
	found := false
	for _, d := range got {
		if d == "2026/06/17" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the June day to match the date filter 'jun', got %v", got)
	}
}

// a content result for an OLD query (the user kept typing) is dropped
func TestLedgerFilterDropsStaleResult(t *testing.T) {
	today := time.Date(2026, 7, 12, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "# 2026/07/09\n\n- ate a burrito\n")

	m := newLedgerModel(t, projectPath, today)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	m = typeString(t, m, "burrito")

	// a result for a DIFFERENT (old) query must be ignored
	mm, _ = m.Update(searchResultsMsg{query: "burr", matches: []daylog.SearchMatch{{Date: "2026/07/09", Line: "- x"}}})
	m = mm.(Model)
	if _, ok := m.contentMatches["2026/07/09"]; ok {
		t.Error("a stale-query result was accepted into contentMatches")
	}
	// and a stale debounce (older seq) fires no search
	if _, cmd := m.Update(searchDebounceMsg{seq: m.filterSeq - 1}); cmd != nil {
		t.Error("a superseded debounce should not run a search")
	}
}

// clearing the filter (esc) drops the content matches and shows the full ledger
func TestLedgerFilterEscClearsContent(t *testing.T) {
	today := time.Date(2026, 7, 12, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "# 2026/07/09\n\n- ate a burrito\n")
	seedLog(t, projectPath, "2026/06/17", "# 2026/06/17\n\n- other\n")

	m := newLedgerModel(t, projectPath, today)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	m = typeString(t, m, "burrito")
	m = runFilterSearch(t, m)
	if len(m.contentMatches) == 0 {
		t.Fatal("precondition: expected content matches before clearing")
	}

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)
	if m.mode != modeLedger || m.dayFilter.Focused() {
		t.Error("expected esc to clear the filter and stay on the ledger")
	}
	if len(m.contentMatches) != 0 || m.contentQuery != "" {
		t.Error("expected content matches cleared after esc")
	}
	// a late result arriving now (filter closed) must be ignored, not applied
	mm, _ = m.Update(searchResultsMsg{query: "burrito", matches: []daylog.SearchMatch{{Date: "2026/07/09", Line: "x"}}})
	m = mm.(Model)
	if len(m.contentMatches) != 0 {
		t.Error("a result arriving after the filter closed was applied")
	}
}

// an async content result must not yank the cursor off a day the user
// navigated onto during the debounce
func TestLedgerFilterResultPreservesCursor(t *testing.T) {
	today := time.Date(2026, 7, 12, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	// two days that both date-match "jul" so navigation has somewhere to go
	seedLog(t, projectPath, "2026/07/09", "# 2026/07/09\n\n- burrito\n")
	seedLog(t, projectPath, "2026/07/08", "# 2026/07/08\n\n- burrito\n")

	m := newLedgerModel(t, projectPath, today)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	m = typeString(t, m, "burrito")
	m = runFilterSearch(t, m)

	// move the cursor onto a specific match
	m.moveLedgerCursor(1)
	want, _ := m.selectedDay()
	if want == "" {
		t.Fatal("expected a selected day after moving the cursor")
	}

	// a fresh (same-query) result rebuild must keep the cursor on that day
	mm, _ = m.Update(searchResultsMsg{query: "burrito", matches: []daylog.SearchMatch{
		{Date: "2026/07/09", Line: "burrito"}, {Date: "2026/07/08", Line: "burrito"},
	}})
	m = mm.(Model)
	if got, _ := m.selectedDay(); got != want {
		t.Errorf("content result stole the cursor: was on %s, now %s", want, got)
	}
}

func TestLedgerEscClearsFilterThenStaysHome(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "- yesterday entry\n")

	m := newLedgerModel(t, projectPath, today)

	// start filtering, then esc clears the filter but stays on the ledger
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	m = typeString(t, m, "07/09")

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)
	if m.mode != modeLedger {
		t.Fatal("expected to stay on the ledger after clearing the filter")
	}
	if m.dayFilter.Focused() {
		t.Error("expected the filter cleared after esc")
	}

	// a second esc on the unfiltered ledger is inert (doesn't quit)
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)
	if m.mode != modeLedger {
		t.Error("expected esc on the home ledger to be a no-op")
	}
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("esc on the ledger must not quit")
		}
	}
}

// esc in browse returns to the ledger with the cursor on the day you were reading
func TestBrowseEscReturnsToLedger(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "- yesterday entry\n")

	m := newTestModel(t, projectPath, today) // browse on today
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mm.(Model) // step to 2026/07/09

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)
	if m.mode != modeLedger {
		t.Fatal("expected esc to return to the ledger")
	}
	if item, _ := m.picker.SelectedItem().(pickerItem); item.value != "2026/07/09" {
		t.Errorf("expected the ledger cursor on the day just read, got %q", item.value)
	}
}

func TestLedgerInitialMode(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "- yesterday entry\n")

	m := newLedgerModel(t, projectPath, today)
	if m.mode != modeLedger {
		t.Fatal("expected the app to launch on the ledger")
	}
	// today is pinned first, then the logged day
	if item, _ := m.picker.SelectedItem().(pickerItem); item.value != "2026/07/10" {
		t.Errorf("expected today pinned and selected, got %q", item.value)
	}
	// today has no file yet, so its anchor row invites creation
	item, _ := m.picker.SelectedItem().(pickerItem)
	if item.row == nil || !strings.Contains(item.row.text, "nothing logged") {
		t.Errorf("expected today's row to invite logging, got %+v", item.row)
	}
	if item.row == nil || item.row.marker != "＋" {
		t.Errorf("expected the ＋ marker on empty today, got %+v", item.row)
	}
}

// ledgerAnchor returns the anchor row for a given day from the picker items
func ledgerAnchor(t *testing.T, m Model, day string) ledgerRow {
	t.Helper()
	for _, it := range m.picker.Items() {
		if p, ok := it.(pickerItem); ok && p.kind == itemDay && p.value == day {
			if p.row == nil {
				t.Fatalf("day %s anchor has no row", day)
			}
			return *p.row
		}
	}
	t.Fatalf("no anchor row for %s in the ledger", day)
	return ledgerRow{}
}

// regression: today WITH a log must show ● + its real content, never the
// "nothing logged yet" CTA. this is the exact bug that shipped because only
// logless-today (＋) was covered
func TestLedgerTodayWithLogShowsContent(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/10", "# 2026/07/10\n\n- shipped the ledger\n- had a burrito\n")
	seedLog(t, projectPath, "2026/07/09", "- yesterday entry\n")

	m := newLedgerModel(t, projectPath, today) // drives loadDays -> previews fully

	anchor := ledgerAnchor(t, m, "2026/07/10")
	if anchor.marker != "●" {
		t.Errorf("expected ● marker on today-with-a-log, got %q", anchor.marker)
	}
	if strings.Contains(anchor.text, "nothing logged") {
		t.Errorf("today has a log but its anchor shows the CTA: %q", anchor.text)
	}
	if anchor.text != "shipped the ledger" {
		t.Errorf("expected today's first log line, got %q", anchor.text)
	}
	if strings.Contains(m.View(), "nothing logged yet") {
		t.Error("the rendered ledger still shows 'nothing logged yet' for a logged today")
	}
}

// regression: on the first paint (after daysLoadedMsg but BEFORE previews
// load), today-with-a-log must already show ● — never flash the ＋ CTA then swap
func TestLedgerTodayWithLogNoCtaFlashOnFirstPaint(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/10", "# 2026/07/10\n\n- shipped the ledger\n")

	m := New(projectPath, "default", today)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mm.(Model)

	// apply ONLY daysLoadedMsg (the first ledger build); do NOT drain the
	// previews cmd it returns, simulating the pre-previews first paint
	loaded := loadDays(projectPath, today)()
	mm, _ = m.Update(loaded)
	m = mm.(Model)

	anchor := ledgerAnchor(t, m, "2026/07/10")
	if anchor.marker != "●" {
		t.Errorf("first paint: expected ● (no CTA flash), got %q", anchor.marker)
	}
	if strings.Contains(m.View(), "nothing logged yet") {
		t.Error("first paint flashed the 'nothing logged yet' CTA for a logged today")
	}
	// today-with-a-log counts toward the header even before its preview loads
	if m.logCount() != 1 {
		t.Errorf("expected logCount 1 for an uncached logged today, got %d", m.logCount())
	}
}

func TestLedgerEnterOpensDay(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "- ate a burrito\n")

	m := newLedgerModel(t, projectPath, today)

	// move onto the logged day and open it
	m.moveLedgerCursor(1)
	if item, _ := m.picker.SelectedItem().(pickerItem); item.value != "2026/07/09" {
		t.Fatalf("expected the logged day selected, got %q", item.value)
	}

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.mode != modeBrowse {
		t.Fatal("expected browse mode after enter")
	}
	if day, _ := m.selectedDay(); day != "2026/07/09" {
		t.Fatalf("expected to open 2026/07/09, got %s", day)
	}
	rendered, ok := findDayRendered(t, execCmd(t, cmd))
	if !ok {
		t.Fatal("expected a dayRenderedMsg after opening a day")
	}
	if !strings.Contains(rendered.content, "burrito") {
		t.Errorf("expected the day's log rendered, got %q", rendered.content)
	}
}

// append from the ledger targets the SELECTED day (not today) and returns to
// the ledger with that day's block updated
func TestLedgerAppendFollowsSelection(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/08", "- existing entry\n")

	m := newLedgerModel(t, projectPath, today)
	// move off today onto the older logged day
	m.moveLedgerCursor(1) // 2026/07/08 (today has no log, so it's the next anchor)
	day, _ := m.selectedDay()
	if day != "2026/07/08" {
		t.Fatalf("expected 2026/07/08 selected, got %s", day)
	}

	// a opens the append input, returning to the ledger
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = mm.(Model)
	if m.mode != modeInput {
		t.Fatal("expected input mode after a")
	}
	if m.inputReturn != modeLedger {
		t.Error("expected append to return to the ledger")
	}
	m = typeString(t, m, "a new note")

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.mode != modeLedger {
		t.Fatal("expected to return to the ledger after append")
	}
	// the append must land on the SELECTED day, not today
	msgs := execCmd(t, cmd)
	if len(msgs) != 1 {
		t.Fatalf("expected one message, got %d", len(msgs))
	}
	if _, ok := msgs[0].(entryAppendedMsg); !ok {
		t.Fatalf("expected entryAppendedMsg, got %T", msgs[0])
	}
	content := string(readLog(t, projectPath, "2026/07/08"))
	if !strings.Contains(content, "- a new note") {
		t.Errorf("expected the note appended to the selected day, got %q", content)
	}
	// today must NOT have been written
	if _, err := os.Stat(logPath(projectPath, "2026/07/10")); !os.IsNotExist(err) {
		t.Error("append should not have created today's log")
	}
}

// regression: after appending, navigating back to the ledger must show the new
// content — the appended day's stale preview must be evicted and re-read
func TestLedgerReflectsAppendedContent(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/08", "- old first line\n")

	// newLedgerModel drives loadDays + previews, so 07/08's preview is cached
	m := newLedgerModel(t, projectPath, today)
	m.moveLedgerCursor(1) // select 2026/07/08

	// append a new line; drive the append cmd and the full reload chain
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = mm.(Model)
	m = typeString(t, m, "brand new line")
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	// run entryAppendedMsg -> loadDays -> daysLoadedMsg -> refreshLedger ->
	// loadVisiblePreviews -> previewsLoadedMsg, applying every follow-up
	drainAll(t, &m, cmd)

	// the appended day's block must now carry the new line, not just the old one
	appended := false
	for _, it := range m.picker.Items() {
		if p, ok := it.(pickerItem); ok && p.value == "2026/07/08" && p.row != nil {
			if strings.Contains(p.row.text, "brand new line") {
				appended = true
			}
		}
	}
	if !appended {
		t.Error("the ledger did not reflect the appended line after navigating back")
	}
	if got := m.previews["2026/07/08"]; len(got) < 2 {
		t.Errorf("expected the day's preview re-read with both lines, got %v", got)
	}
}

// drainAll recursively applies a cmd and every message/cmd it produces,
// simulating the tea runtime processing a full reload chain
func drainAll(t *testing.T, m *Model, cmd tea.Cmd) {
	t.Helper()
	for _, msg := range execCmd(t, cmd) {
		if msg == nil {
			continue
		}
		mm, next := m.Update(msg)
		*m = mm.(Model)
		if next != nil {
			drainAll(t, m, next)
		}
	}
}

// edit from the ledger opens $EDITOR on the SELECTED day
func TestLedgerEditFollowsSelection(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/08", "- existing entry\n")

	m := newLedgerModel(t, projectPath, today)
	m.moveLedgerCursor(1) // select 2026/07/08
	if day, _ := m.selectedDay(); day != "2026/07/08" {
		t.Fatalf("expected 2026/07/08 selected, got %s", day)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("expected an editor cmd for the selected day")
	}
	// the editor opens the selected day's existing file (it already exists)
	content := string(readLog(t, projectPath, "2026/07/08"))
	if !strings.Contains(content, "existing entry") {
		t.Errorf("expected the selected day's file, got %q", content)
	}
}

func TestLedgerNewDayRow(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()

	m := newLedgerModel(t, projectPath, today)

	// start a filter and type a date (same grammar the CLI accepts)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	m = typeString(t, m, "6/15")

	// the first row is the New day affordance
	first, _ := m.picker.Items()[0].(pickerItem)
	if first.kind != itemNewDay {
		t.Fatalf("expected a New day row at the top while filtering, got %q", first.label)
	}

	// enter on it resolves the typed date and opens that (empty) day
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.mode != modeBrowse {
		t.Fatal("expected browse mode after choosing a new day")
	}
	if day, _ := m.selectedDay(); day != "2026/06/15" {
		t.Fatalf("expected to open the new day 2026/06/15, got %s", day)
	}
	if !slices.Contains(m.days, "2026/06/15") {
		t.Error("expected the new day inserted into the day list")
	}
}

// the cursor lands only on a day's anchor line, skipping its continuation
// lines and the blank spacers between blocks; enter on a non-anchor row is inert
func TestContinuationLinesSkipped(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	// a multi-line log so its block has continuation rows below the anchor
	seedLog(t, projectPath, "2026/07/09", "- line one\n- line two\n- line three\n")
	seedLog(t, projectPath, "2026/07/04", "- last week\n")

	m := newLedgerModel(t, projectPath, today)
	items := m.picker.Items()

	// find a continuation row (part of the 3-line 07/09 block)
	contIdx := -1
	for i, it := range items {
		if p, ok := it.(pickerItem); ok && p.kind == itemCont {
			contIdx = i
			break
		}
	}
	if contIdx < 1 {
		t.Fatal("expected continuation rows under a multi-line day block")
	}

	// enter while parked on a continuation row is inert
	m.picker.Select(contIdx)
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.mode != modeLedger {
		t.Error("expected enter on a continuation row to be a no-op")
	}
	if cmd != nil {
		t.Error("expected no cmd from entering a continuation row")
	}

	// moving the cursor only ever lands on a day anchor, never a cont/spacer
	m.picker.Select(0) // today anchor
	for range items {
		m.moveLedgerCursor(1)
		if item, ok := m.picker.SelectedItem().(pickerItem); ok && item.kind != itemDay {
			t.Fatalf("cursor landed on a non-anchor row: kind=%v", item.kind)
		}
	}
}

func TestPreviewLines(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "skips heading and blanks", content: "# 2026/07/10\n\n- did a thing\n", want: []string{"did a thing"}},
		{name: "strips list marker", content: "* starred item\n", want: []string{"starred item"}},
		{name: "plain first lines", content: "just prose\nmore\n", want: []string{"just prose", "more"}},
		{name: "empty file", content: "", want: nil},
		{name: "only heading and blanks", content: "# title\n\n\n", want: nil},
		{name: "checked todo with strikethrough", content: "- [x] TODO: ~~a cheeseburger?~~\n", want: []string{"TODO: a cheeseburger?"}},
		{name: "skips a bare code fence", content: "```\ncode inside\n```\n", want: []string{"code inside"}},
		{name: "unwraps bold and inline code", content: "- **bold** and `code`\n", want: []string{"bold and code"}},
		{name: "unwraps a link", content: "- [go get a burrito](http://x) now\n", want: []string{"go get a burrito now"}},
		{
			name:    "caps at five lines then an ellipsis",
			content: "- l1\n- l2\n- l3\n- l4\n- l5\n- l6\n- l7\n",
			want:    []string{"l1", "l2", "l3", "l4", "l5", "…"},
		},
		{
			name:    "exactly five lines has no ellipsis",
			content: "- l1\n- l2\n- l3\n- l4\n- l5\n",
			want:    []string{"l1", "l2", "l3", "l4", "l5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".md")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := previewLines(path)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}

	// a missing file is an error the caller skips over
	if _, err := previewLines(filepath.Join(dir, "nope.md")); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestLoadPreviews(t *testing.T) {
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "# 2026/07/09\n\n- ate a burrito\n- and a taco\n")
	seedLog(t, projectPath, "2026/07/08", "planned the week\n")

	msg := loadPreviews(projectPath, []string{"2026/07/09", "2026/07/08", "2026/07/07"})()
	loaded, ok := msg.(previewsLoadedMsg)
	if !ok {
		t.Fatalf("expected previewsLoadedMsg, got %T", msg)
	}
	if !slices.Equal(loaded.previews["2026/07/09"], []string{"ate a burrito", "and a taco"}) {
		t.Errorf("expected two-line preview, got %q", loaded.previews["2026/07/09"])
	}
	if !slices.Equal(loaded.previews["2026/07/08"], []string{"planned the week"}) {
		t.Errorf("expected week preview, got %q", loaded.previews["2026/07/08"])
	}
	// a day with no file simply has no preview (not an error)
	if _, present := loaded.previews["2026/07/07"]; present {
		t.Error("expected no preview entry for a missing day")
	}
}

func TestQuitKeys(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{name: "q", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
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

	// esc returns to the ledger from browse, so it must not quit the session
	m := newTestModel(t, t.TempDir(), today)
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc}); cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("esc should not quit in browse mode")
		}
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

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if cmd == nil {
		t.Fatal("expected a copy cmd")
	}
	// don't execute it — that would touch the real clipboard
}

// c copies from the ledger too, hitting the highlighted day (not today, when a
// different day is highlighted). mirrors the log-view TestCopyKey
func TestCopyKeyFromLedger(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/08", "- older entry\n")

	m := newLedgerModel(t, projectPath, today)
	m.moveLedgerCursor(1) // highlight 2026/07/08
	if day, _ := m.selectedDay(); day != "2026/07/08" {
		t.Fatalf("expected 2026/07/08 selected, got %s", day)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if cmd == nil {
		t.Fatal("expected a copy cmd from the ledger")
	}
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
	if m.status != "copied to clipboard" {
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

// switching projects must clear the date-keyed preview cache so today (or any
// day) can't render the previous project's content
func TestProjectSwitchClearsPreviewCache(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	m := newTestModel(t, t.TempDir(), today)
	m.previews["2026/07/10"] = []string{"project A content"}

	newPath := t.TempDir()
	mm, _ := m.Update(projectSwitchedMsg{name: "b", path: newPath})
	m = mm.(Model)

	if len(m.previews) != 0 {
		t.Errorf("expected the preview cache cleared on project switch, got %v", m.previews)
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
	if item, _ := m.picker.SelectedItem().(pickerItem); item.value != "default" {
		t.Errorf("expected current project pre-selected, got %q", item.value)
	}

	// navigate to "work"; simulate the switch completing
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = mm.(Model)
	if item, _ := m.picker.SelectedItem().(pickerItem); item.value != "work" {
		t.Fatalf("expected work selected, got %q", item.value)
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

func TestTodoFlow(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/10", "# 2026/07/10\n\n- TODO: buy tortillas\n- [x] TODO: eat a burrito\n")

	m := newTestModel(t, projectPath, today)

	// t opens the todo picker
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = mm.(Model)
	if m.mode != modeTodos {
		t.Fatal("expected todos mode after t")
	}

	msgs := execCmd(t, cmd)
	if len(msgs) != 1 {
		t.Fatalf("expected one message, got %d", len(msgs))
	}
	mm, _ = m.Update(msgs[0])
	m = mm.(Model)

	if len(m.todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(m.todos))
	}
	if item, _ := m.picker.SelectedItem().(pickerItem); item.label != "[ ] TODO: buy tortillas" {
		t.Errorf("expected unchecked first item, got %q", item.label)
	}

	// space toggles the selected todo on disk
	mm, cmd = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = mm.(Model)
	msgs = execCmd(t, cmd)
	if len(msgs) != 1 {
		t.Fatalf("expected one message, got %d", len(msgs))
	}
	if _, ok := msgs[0].(todoToggledMsg); !ok {
		t.Fatalf("expected todoToggledMsg, got %T", msgs[0])
	}

	content := string(readLog(t, projectPath, "2026/07/10"))
	if !strings.Contains(content, "- [x] TODO: buy tortillas") {
		t.Errorf("expected todo toggled to done on disk, got %q", content)
	}

	// the toggle triggers a reload; picker stays open with updated checkbox
	mm, cmd = m.Update(msgs[0])
	m = mm.(Model)
	if m.mode != modeTodos {
		t.Error("expected todo picker to stay open after toggle")
	}
	for _, msg := range execCmd(t, cmd) {
		mm, _ = m.Update(msg)
		m = mm.(Model)
	}
	if item, _ := m.picker.SelectedItem().(pickerItem); item.label != "[✓] TODO: buy tortillas" {
		t.Errorf("expected checked first item after reload, got %q", item.label)
	}

	// enter confirms/closes, same as esc
	mm, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.mode != modeBrowse {
		t.Error("expected enter to close todo picker")
	}
	if cmd != nil {
		t.Error("expected no cmd when closing with enter")
	}
}

func TestTodosEmptyDayShowsStatus(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	m := newTestModel(t, t.TempDir(), today)

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = mm.(Model)

	msgs := execCmd(t, cmd)
	mm, _ = m.Update(msgs[0])
	m = mm.(Model)

	if m.mode != modeBrowse {
		t.Error("expected to fall back to browse when day has no todos")
	}
	if !strings.Contains(m.status, "no todos") {
		t.Errorf("expected no-todos status, got %q", m.status)
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

// anchorRow returns the anchor ledgerRow of a day's block for assertions
func anchorRow(t *testing.T, day string, today time.Time, preview []string) ledgerRow {
	t.Helper()
	// noLogToday=true so an empty today still gets its ＋ CTA; non-today days
	// ignore it (the CTA branch requires isToday)
	block := dayBlock(day, today, preview, true)
	if len(block) == 0 {
		t.Fatalf("empty block for %s", day)
	}
	p, ok := block[0].(pickerItem)
	if !ok || p.row == nil || p.kind != itemDay {
		t.Fatalf("first block item isn't a day anchor: %+v", block[0])
	}
	return *p.row
}

func TestLedgerRowColumns(t *testing.T) {
	today := time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name        string
		day         string
		preview     []string
		wantMarker  string
		wantDate    string
		wantWeekday string
		wantText    string
		wantAccent  bool
	}{
		{name: "this-year day", day: "2026/06/17", preview: []string{"total 48"}, wantMarker: "●", wantDate: "Jun 17", wantWeekday: "Wed", wantText: "total 48"},
		{name: "cross-year shows full year", day: "2025/08/09", preview: []string{"burrito"}, wantMarker: "●", wantDate: "Aug 09 2025", wantWeekday: "Sat", wantText: "burrito"},
		{name: "today reads as the word today", day: "2026/07/11", preview: []string{"shipped it"}, wantMarker: "●", wantDate: "today", wantWeekday: "Sat", wantText: "shipped it"},
		{name: "yesterday reads as the word yesterday", day: "2026/07/10", preview: []string{"did a thing"}, wantMarker: "●", wantDate: "yesterday", wantWeekday: "Fri", wantText: "did a thing"},
		{name: "empty today invites", day: "2026/07/11", preview: nil, wantMarker: "＋", wantDate: "today", wantWeekday: "Sat", wantText: "nothing logged yet · a append · e edit", wantAccent: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := anchorRow(t, tt.day, today, tt.preview)
			if r.marker != tt.wantMarker {
				t.Errorf("marker: want %q got %q", tt.wantMarker, r.marker)
			}
			if r.date != tt.wantDate {
				t.Errorf("date: want %q got %q", tt.wantDate, r.date)
			}
			if r.weekday != tt.wantWeekday {
				t.Errorf("weekday: want %q got %q", tt.wantWeekday, r.weekday)
			}
			if r.text != tt.wantText {
				t.Errorf("text: want %q got %q", tt.wantText, r.text)
			}
			if r.accent != tt.wantAccent {
				t.Errorf("accent: want %v got %v", tt.wantAccent, r.accent)
			}
			if !r.anchor {
				t.Error("expected the first block row to be an anchor")
			}
		})
	}
}

// a multi-line log block has one anchor line (with the date rail) plus a
// continuation line per further log line; the "…" caps an over-long log
func TestDayBlockMultiLine(t *testing.T) {
	today := time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local)
	block := dayBlock("2026/06/17", today, []string{"line one", "line two", "…"}, false)
	if len(block) != 3 {
		t.Fatalf("expected 3 rows in the block, got %d", len(block))
	}
	first, _ := block[0].(pickerItem)
	if first.kind != itemDay || !first.row.anchor || first.row.date != "Jun 17" {
		t.Errorf("first row should be the dated anchor, got %+v", first.row)
	}
	for i, it := range block[1:] {
		p, _ := it.(pickerItem)
		if p.kind != itemCont {
			t.Errorf("continuation row %d should be itemCont, got %v", i, p.kind)
		}
		if p.selectable() {
			t.Errorf("continuation row %d should not be selectable", i)
		}
		if p.row.anchor || p.row.marker != "" || p.row.date != "" {
			t.Errorf("continuation row %d should have a blank rail, got %+v", i, p.row)
		}
	}
	// the "…" continuation renders faint
	if last, _ := block[2].(pickerItem); !last.row.faint {
		t.Error("the … overflow line should be faint")
	}
}

// every rendered ledger row must be exactly the picker width, so the │ rule
// aligns down the whole list and the pane fills without wrapping
func TestLedgerRowsAlign(t *testing.T) {
	today := time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local)
	d := pickerDelegate{styles: defaultStyles()}
	// anchor rows (double-width marker, cross-year date) and a continuation row
	rows := []ledgerRow{
		anchorRow(t, "2026/07/11", today, nil),                   // ＋ today
		anchorRow(t, "2026/06/17", today, []string{"total 48"}),  // this-year
		anchorRow(t, "2025/08/09", today, []string{"a burrito"}), // cross-year
		{text: "a continuation line"},                            // blank-rail continuation
	}
	const width = 90
	for i, r := range rows {
		for _, selected := range []bool{false, true} {
			line := d.renderLedgerRow(r, width, selected)
			if w := lipgloss.Width(line); w != width {
				t.Errorf("row %d selected=%v: width %d, want %d", i, selected, w, width)
			}
		}
	}
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
		{name: "other year", day: "2025/12/31", wantPrimary: "Dec 31 2025", wantSecondary: "2025/12/31"},
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

// a delayed todo load must not overwrite a picker the user has since
// opened for another mode; without the mode guard its empty-value rows
// leak into the projects picker and enter switches to the data root
func TestTodosLoadDoesNotLeakIntoAnotherModal(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	m := newTestModel(t, t.TempDir(), today)

	// enter todos mode (fires loadTodos), then leave and open projects
	m.mode = modeProjects
	mm, _ := m.Update(projectsLoadedMsg{projects: []string{"default", "work"}})
	m = mm.(Model)

	// the in-flight todo load resolves after the switch
	mm, _ = m.Update(todosLoadedMsg{day: "2026/07/10", todos: []todo.Item{{Line: 1, Text: "TODO buy milk"}}})
	m = mm.(Model)

	if item, _ := m.picker.SelectedItem().(pickerItem); strings.Contains(item.label, "TODO") {
		t.Errorf("todo row leaked into projects picker: %q", item.label)
	}
}

// mirror guard for the project picker
func TestProjectsLoadDoesNotLeakIntoAnotherModal(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	seedPath := t.TempDir()
	seedLog(t, seedPath, "2026/07/10", "- [ ] TODO water plants\n")
	m := newTestModel(t, seedPath, today)

	// open todos, then a late projectsLoadedMsg arrives
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = mm.(Model)
	for _, msg := range execCmd(t, cmd) {
		mm, _ = m.Update(msg)
		m = mm.(Model)
	}
	before := len(m.picker.Items())

	mm, _ = m.Update(projectsLoadedMsg{projects: []string{"default", "work"}})
	m = mm.(Model)

	if len(m.picker.Items()) != before {
		t.Errorf("projects leaked into todo picker: items went %d -> %d", before, len(m.picker.Items()))
	}
}

// pressing enter on a content-matched row opens that day's log
func TestLedgerFilterOpensContentMatch(t *testing.T) {
	today := time.Date(2026, 7, 12, 12, 0, 0, 0, time.Local)
	projectPath := t.TempDir()
	seedLog(t, projectPath, "2026/07/09", "# 2026/07/09\n\n- got a burrito\n")
	seedLog(t, projectPath, "2026/06/17", "# 2026/06/17\n\n- shipped it\n")

	m := newLedgerModel(t, projectPath, today)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mm.(Model)
	m = typeString(t, m, "burrito")
	m = runFilterSearch(t, m)

	// the burrito day is the only content match; open it
	m.restoreLedgerCursor("2026/07/09")
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	if m.mode != modeBrowse {
		t.Fatal("expected browse mode after opening a content match")
	}
	if day, _ := m.selectedDay(); day != "2026/07/09" {
		t.Fatalf("expected to open 2026/07/09, got %s", day)
	}
	rendered, ok := findDayRendered(t, execCmd(t, cmd))
	if !ok {
		t.Fatal("expected a render after opening the match")
	}
	if !strings.Contains(rendered.content, "burrito") {
		t.Errorf("expected the matched log rendered, got %q", rendered.content)
	}
}

// tiny terminals and long content must not overflow or panic across every mode
func TestLayoutRobustAtExtremes(t *testing.T) {
	today := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local)
	m := newTestModel(t, t.TempDir(), today)

	// a very long ledger preview line is truncated to the terminal width
	m.mode = modeLedger
	longRow := ledgerRow{anchor: true, marker: "●", weekday: "Fri", date: "Jul 10", text: strings.Repeat("x", 200), bullet: true}
	if w := lipgloss.Width(pickerDelegate{styles: defaultStyles()}.renderLedgerRow(longRow, 80, false)); w > 80 {
		t.Fatalf("ledger row width %d exceeds terminal width 80", w)
	}

	// extreme sizes across every mode must not panic
	for _, s := range [][2]int{{0, 0}, {1, 1}, {2, 2}, {80, 3}} {
		mm, _ := m.Update(tea.WindowSizeMsg{Width: s[0], Height: s[1]})
		m = mm.(Model)
		for _, mode := range []mode{modeBrowse, modeInput, modeLedger, modeProjects, modeTodos} {
			m.mode = mode
			_ = m.View()
		}
	}
}
