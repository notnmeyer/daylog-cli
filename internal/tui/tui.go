package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/notnmeyer/daylog-cli/internal/daylog"
)

type mode int

const (
	modeBrowse mode = iota
	modeInput
)

type focusArea int

const (
	focusDays focusArea = iota
	focusViewport
)

// wide enough for "  wednesday '25"
const dayListWidth = 15

type Model struct {
	project     string
	projectPath string
	today       time.Time
	mode        mode
	focus       focusArea
	days        list.Model
	vp          viewport.Model
	input       textinput.Model
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

	l := list.New(nil, dayDelegate{styles: st, today: today}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	input := textinput.New()
	input.Prompt = "append › "
	input.Placeholder = "what did you do?"

	return Model{
		project:     project,
		projectPath: projectPath,
		today:       today,
		mode:        modeBrowse,
		focus:       focusDays,
		days:        l,
		vp:          viewport.New(0, 0),
		input:       input,
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

	frameW, frameH := m.styles.focusedPane.GetFrameSize()
	headerH := lipglossHeight(m.headerView())
	footerH := lipglossHeight(m.footerView())

	bodyH := m.height - headerH - footerH - frameH
	if bodyH < 1 {
		bodyH = 1
	}

	m.days.SetSize(dayListWidth, bodyH)

	vpW := m.width - dayListWidth - 2*frameW
	if vpW < 1 {
		vpW = 1
	}
	m.vp.Width = vpW
	m.vp.Height = bodyH

	m.input.Width = m.width - len(m.input.Prompt) - 4
}

func (m Model) selectedDay() (string, bool) {
	it, ok := m.days.SelectedItem().(dayItem)
	if !ok {
		return "", false
	}
	return string(it), true
}

// renderSelected re-renders the currently selected day into the viewport
func (m Model) renderSelected() tea.Cmd {
	day, ok := m.selectedDay()
	if !m.ready || !ok {
		return nil
	}
	return renderDay(m.md, m.projectPath, day, m.vp.Width)
}
