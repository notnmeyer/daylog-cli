package tui

import (
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
	modeDays
	modeSearch
)

type Model struct {
	project     string
	projectPath string
	today       time.Time
	mode        mode
	days        []string // YYYY/MM/DD, newest first
	dayIdx      int
	vp          viewport.Model
	input       textinput.Model
	picker      list.Model
	dayFilter   textinput.Model
	searchInput textinput.Model
	searchSeq   int
	todos       []todo.Item
	md          mdRenderer
	keys        keyMap
	help        help.Model
	styles      styles
	status      string
	width       int
	height      int
	ready       bool
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
	dayFilter.Prompt = " ⌕ "
	dayFilter.Placeholder = "type to filter"

	searchInput := textinput.New()
	searchInput.Prompt = " / "
	searchInput.Placeholder = "search all logs"

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
		mode:        modeBrowse,
		vp:          vp,
		input:       input,
		picker:      picker,
		dayFilter:   dayFilter,
		searchInput: searchInput,
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

	m.input.Width = m.width - len(m.input.Prompt) - 4

	pickerW := min(60, m.width-8)
	if m.mode == modeSearch {
		// search rows carry whole log lines; give them more room
		pickerW = min(90, m.width-8)
	}
	pickerH := min(15, bodyH-4)
	// the day and search pickers keep a stable height while typing;
	// the project/todo pickers shrink to fit their items
	if m.mode != modeDays && m.mode != modeSearch {
		pickerH = min(pickerH, max(1, len(m.picker.Items())))
	}
	if pickerH < 1 {
		pickerH = 1
	}
	m.picker.SetSize(pickerW, pickerH)
	m.dayFilter.Width = pickerW - len(m.dayFilter.Prompt) - 2
	m.searchInput.Width = pickerW - len(m.searchInput.Prompt) - 2
}

func (m Model) selectedDay() (string, bool) {
	if m.dayIdx < 0 || m.dayIdx >= len(m.days) {
		return "", false
	}
	return m.days[m.dayIdx], true
}

// renderSelected re-renders the currently selected day into the viewport
func (m Model) renderSelected() tea.Cmd {
	day, ok := m.selectedDay()
	if !m.ready || !ok {
		return nil
	}
	return renderDay(m.md, m.projectPath, day, m.vp.Width)
}
