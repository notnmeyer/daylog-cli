package tui

import (
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/notnmeyer/daylog-cli/internal/todo"
)

type mode int

const (
	modeBrowse mode = iota
	modeInput
	modeProjects
	modeTodos
	// modeLedger is the landing screen: a full-width list of days that is
	// also the day picker. it replaces the old modeDays modal
	modeLedger
)

type Model struct {
	project     string
	projectPath string
	today       time.Time
	mode        mode
	inputReturn mode                // mode to return to after the append input closes
	days        []string            // YYYY/MM/DD, newest first
	noLogToday  bool                // today has no non-empty log (force-prepended by loadDays)
	previews    map[string][]string // day -> first few log lines, cached per session
	dayIdx      int
	vp          viewport.Model
	input       textinput.Model
	picker      list.Model
	dayFilter   textinput.Model
	// ledger content search: day -> the first log line matching the current
	// filter query (shown as the filtered row's preview so you see WHY it
	// matched), plus the query it was computed for (so a stale set can't leak
	// into a rebuild). filterSeq drops superseded debounces
	contentMatches map[string]string
	contentQuery   string
	filterSeq      int
	todos          []todo.Item
	md             mdRenderer
	keys           keyMap
	help           help.Model
	styles         styles
	status         string
	width          int
	height         int
	ready          bool
}

func Run(dl *daylog.DayLog, project string) error {
	m := New(dl.ProjectPath, project, *dl.Date)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func New(projectPath, project string, today time.Time) Model {
	st := defaultStyles()

	input := textinput.New()
	input.Prompt = "append › "
	input.Placeholder = "what did you do?"

	dayFilter := textinput.New()
	dayFilter.Prompt = "filter › "
	dayFilter.Placeholder = "date or text"

	// shared by the day, project, and todo pickers
	picker := list.New(nil, pickerDelegate{styles: st}, 0, 0)
	picker.SetShowTitle(false)
	picker.SetShowStatusBar(false)
	picker.SetShowHelp(false)
	picker.SetShowPagination(false)
	picker.SetFilteringEnabled(false)
	picker.DisableQuitKeybindings()

	vp := viewport.New(0, 0)
	// d opens the day picker, so rebind half-page-down off of it
	vp.KeyMap.HalfPageDown = key.NewBinding(key.WithKeys("ctrl+d"))
	vp.KeyMap.HalfPageUp = key.NewBinding(key.WithKeys("u", "ctrl+u"))

	return Model{
		project:     project,
		projectPath: projectPath,
		today:       today,
		// land on the ledger: a list of days, not an empty today
		mode:        modeLedger,
		previews:    map[string][]string{},
		vp:          vp,
		input:       input,
		picker:      picker,
		dayFilter:   dayFilter,
		md:          newMDRenderer(),
		keys:        defaultKeyMap(),
		help:        help.New(),
		styles:      st,
	}
}

func (m Model) Init() tea.Cmd {
	return loadDays(m.projectPath, m.today)
}

func (m *Model) layout() {
	m.help.Width = m.width

	frameW, frameH := m.styles.pane.GetFrameSize()
	headerH := lipglossHeight(m.headerView())
	footerH := lipglossHeight(m.footerView())

	bodyH := m.height - headerH - footerH - frameH
	if bodyH < 1 {
		bodyH = 1
	}

	vpW := m.width - frameW
	if vpW < 1 {
		vpW = 1
	}
	m.vp.Width = vpW
	m.vp.Height = bodyH

	// clamp every input width: a negative width panics bubbles' textinput
	m.input.Width = max(1, m.width-len(m.input.Prompt)-4)

	if m.mode == modeLedger {
		// the ledger IS the body: fill the whole pane. the picker width is the
		// pane's inner content width, and every ledger row is padded to exactly
		// that width, so the pane grows to full width naturally (no explicit
		// lipgloss .Width(), which wraps content off-by-a-few with border+pad)
		pickerW := max(1, m.width-frameW)
		pickerH := max(1, bodyH)
		m.picker.SetSize(pickerW, pickerH)
		m.dayFilter.Width = max(1, pickerW-len(m.dayFilter.Prompt)-2)
		return
	}

	pickerW := max(1, min(60, m.width-12))
	// every picker shrinks to fit its items, capped so a long list
	// still leaves room for the modal frame, title, input, and gaps
	pickerH := min(15, bodyH-8)
	pickerH = min(pickerH, max(1, len(m.picker.Items())))
	if pickerH < 1 {
		pickerH = 1
	}
	m.picker.SetSize(pickerW, pickerH)
	m.dayFilter.Width = max(1, pickerW-len(m.dayFilter.Prompt)-2)
}

// selectDay points the browse view at day, inserting it into the
// newest-first day list if a search landed on a date the list doesn't
// carry (e.g. a log that GetLogs filtered out). without this a jump to
// such a day would silently no-op
func (m *Model) selectDay(day string) {
	if idx := slices.Index(m.days, day); idx >= 0 {
		m.dayIdx = idx
		return
	}
	idx, _ := slices.BinarySearchFunc(m.days, day, func(a, b string) int {
		return strings.Compare(b, a) // days are newest-first
	})
	m.days = slices.Insert(m.days, idx, day)
	m.dayIdx = idx
}

func (m Model) selectedDay() (string, bool) {
	if m.dayIdx < 0 || m.dayIdx >= len(m.days) {
		return "", false
	}
	return m.days[m.dayIdx], true
}

// openLedger switches to the ledger, clearing any filter and landing the
// cursor on the current day. shared by launch, d, and esc-from-browse
func (m *Model) openLedger() tea.Cmd {
	m.mode = modeLedger
	m.status = ""
	m.dayFilter.Reset()
	m.dayFilter.Blur()
	m.contentMatches = nil
	m.contentQuery = ""
	return m.refreshLedger()
}

// refreshLedger rebuilds the ledger rows, keeps the cursor on the current day,
// and returns a cmd to load previews for the visible days that aren't cached
// yet. call it whenever the day list changes while the ledger is showing
func (m *Model) refreshLedger() tea.Cmd {
	m.rebuildLedgerItems()
	m.layout()
	return m.loadVisiblePreviews()
}

// rebuildLedgerItems repopulates the picker from the current days/previews and
// restores the cursor onto the selected day (rows include gap dividers, so the
// picker index and dayIdx differ)
func (m *Model) rebuildLedgerItems() {
	query := ""
	var content map[string]string
	if m.dayFilter.Focused() {
		query = strings.TrimSpace(m.dayFilter.Value())
		// only union the content-match set when it was computed for THIS exact
		// query — a rebuild triggered by an unrelated event (e.g. a preview
		// arriving mid-filter) must never surface a stale set
		if query != "" && query == m.contentQuery {
			content = m.contentMatches
		}
	}
	items := ledgerItems(m.days, m.today, m.previews, m.noLogToday, query, content)
	m.picker.SetItems(items)

	if query != "" {
		m.picker.Select(0)
		return
	}
	if day, ok := m.selectedDay(); ok {
		for i, it := range items {
			// land on the day's anchor line (itemDay), not a continuation
			if p, ok := it.(pickerItem); ok && p.kind == itemDay && p.value == day {
				m.picker.Select(i)
				return
			}
		}
	}
	m.picker.Select(0)
}

// loadVisiblePreviews reads first-line previews for the days that have a log
// but aren't cached yet, bounded to a window near the cursor so this stays
// O(visible) rather than O(history)
func (m Model) loadVisiblePreviews() tea.Cmd {
	const window = 20
	start := max(0, m.dayIdx-window/2)
	end := min(len(m.days), start+window)

	var need []string
	for _, day := range m.days[start:end] {
		if _, cached := m.previews[day]; cached {
			continue
		}
		if isToday(day, m.today) && m.noLogToday {
			// only a genuinely-logless today has no file to read; its row shows
			// the create prompt, not a preview, so skip it. today WITH a log is
			// read like any other day (once — the cache guard above stops a
			// re-queue), so its real preview loads instead of a wrong CTA
			continue
		}
		need = append(need, day)
	}
	if len(need) == 0 {
		return nil
	}
	return loadPreviews(m.projectPath, need)
}

// renderSelected re-renders the currently selected day into the viewport
func (m Model) renderSelected() tea.Cmd {
	day, ok := m.selectedDay()
	if !m.ready || !ok {
		return nil
	}
	return renderDay(m.md, m.projectPath, day, m.vp.Width, m.today)
}
