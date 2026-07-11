package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.ready = true
		return m, m.renderSelected()

	case daysLoadedMsg:
		return m.onDaysLoaded(msg)

	case dayRenderedMsg:
		m.vp.SetContent(msg.content)
		m.vp.GotoTop()
		return m, nil

	case entryAppendedMsg:
		return m, loadDays(m.projectPath, m.today)

	case editorFinishedMsg:
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
		}
		return m, loadDays(m.projectPath, m.today)

	case projectsLoadedMsg:
		items := make([]list.Item, len(msg.projects))
		selected := 0
		for i, p := range msg.projects {
			items[i] = pickerItem(p)
			if p == m.project {
				selected = i
			}
		}
		m.picker.SetItems(items)
		m.picker.Select(selected)
		m.layout()
		return m, nil

	case projectSwitchedMsg:
		m.project = msg.name
		m.projectPath = msg.path
		m.mode = modeBrowse
		// drop the old project's selection so the reload lands on today
		m.days.SetItems(nil)
		return m, loadDays(m.projectPath, m.today)

	case copiedMsg:
		m.status = "Copied to clipboard."
		return m, clearStatusAfter(2 * time.Second)

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case errMsg:
		m.status = "error: " + msg.err.Error()
		return m, nil

	case tea.KeyMsg:
		return m.onKey(msg)
	}

	if m.mode == modeInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) onDaysLoaded(msg daysLoadedMsg) (tea.Model, tea.Cmd) {
	prev, _ := m.selectedDay()

	items := make([]list.Item, len(msg.days))
	selected := 0
	for i, d := range msg.days {
		items[i] = dayItem(d)
		if d == prev {
			selected = i
		}
	}

	m.days.SetItems(items)
	m.days.Select(selected)

	return m, m.renderSelected()
}

func (m Model) onKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeInput:
		return m.onInputKey(msg)
	case modeProjects:
		return m.onPickerKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Append):
		m.mode = modeInput
		m.status = ""
		return m, m.input.Focus()

	case key.Matches(msg, m.keys.Edit):
		day, ok := m.selectedDay()
		if !ok {
			return m, nil
		}
		return m, openEditor(m.projectPath, day)

	case key.Matches(msg, m.keys.Copy):
		day, ok := m.selectedDay()
		if !ok {
			return m, nil
		}
		return m, copyDay(m.projectPath, day)

	case key.Matches(msg, m.keys.Projects):
		m.mode = modeProjects
		m.status = ""
		return m, loadProjects()

	case key.Matches(msg, m.keys.Tab):
		if m.focus == focusDays {
			m.focus = focusViewport
		} else {
			m.focus = focusDays
		}
		return m, nil

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.layout()
		return m, nil
	}

	if m.focus == focusDays {
		before := m.days.Index()
		var cmd tea.Cmd
		m.days, cmd = m.days.Update(msg)
		if m.days.Index() != before {
			return m, tea.Batch(cmd, m.renderSelected())
		}
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.Top):
		m.vp.GotoTop()
		return m, nil
	case key.Matches(msg, m.keys.Bottom):
		m.vp.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m Model) onPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		item, ok := m.picker.SelectedItem().(pickerItem)
		if !ok {
			m.mode = modeBrowse
			return m, nil
		}
		return m, switchProject(string(item))

	case tea.KeyEsc:
		m.mode = modeBrowse
		return m, nil
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m Model) onInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		text := strings.TrimSpace(m.input.Value())
		m.input.Reset()
		m.input.Blur()
		m.mode = modeBrowse

		if text == "" {
			return m, nil
		}

		day, ok := m.selectedDay()
		if !ok {
			return m, nil
		}
		return m, appendEntry(m.projectPath, day, text)

	case tea.KeyEsc:
		m.input.Reset()
		m.input.Blur()
		m.mode = modeBrowse
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
