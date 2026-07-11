package tui

import (
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

	case errMsg:
		m.status = "error: " + msg.err.Error()
		return m, nil

	case tea.KeyMsg:
		return m.onKey(msg)
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
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

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
