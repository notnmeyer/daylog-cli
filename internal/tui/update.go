package tui

import (
	"slices"
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
		prev, _ := m.selectedDay()
		m.days = msg.days
		m.dayIdx = 0
		if idx := slices.Index(m.days, prev); idx >= 0 {
			m.dayIdx = idx
		}
		return m, m.renderSelected()

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
		// a load that resolves after the user left the picker must not
		// overwrite whatever modal is now open (the picker is shared)
		if m.mode != modeProjects {
			return m, nil
		}
		items := make([]list.Item, len(msg.projects))
		selected := 0
		for i, p := range msg.projects {
			items[i] = pickerItem{label: p, value: p}
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
		m.days = nil
		return m, loadDays(m.projectPath, m.today)

	case todosLoadedMsg:
		// same guard as the project picker: a stale load carrying empty
		// values must not leak todo rows into another modal
		if m.mode != modeTodos {
			return m, nil
		}
		if len(msg.todos) == 0 {
			m.mode = modeBrowse
			m.status = "no todos for " + msg.day
			return m, clearStatusAfter(2 * time.Second)
		}

		// keep the cursor in place across toggle reloads
		selected := min(m.picker.Index(), len(msg.todos)-1)

		m.todos = msg.todos
		items := make([]list.Item, len(msg.todos))
		for i, td := range msg.todos {
			box := "[ ] "
			if td.Done {
				box = "[✓] "
			}
			items[i] = pickerItem{label: box + td.Text}
		}
		m.picker.SetItems(items)
		m.picker.Select(selected)
		m.layout()
		return m, nil

	case todoToggledMsg:
		day, ok := m.selectedDay()
		if !ok {
			return m, nil
		}
		return m, tea.Batch(loadTodos(m.projectPath, day), m.renderSelected())

	case searchDebounceMsg:
		// only the latest keystroke's debounce runs the search
		if m.mode != modeSearch || msg.seq != m.searchSeq {
			return m, nil
		}
		query := strings.TrimSpace(m.searchInput.Value())
		if query == "" {
			m.picker.SetItems(nil)
			return m, nil
		}
		return m, runSearch(m.projectPath, query)

	case searchResultsMsg:
		if m.mode != modeSearch || msg.query != strings.TrimSpace(m.searchInput.Value()) {
			return m, nil
		}
		items := make([]list.Item, len(msg.matches))
		for i, match := range msg.matches {
			items[i] = pickerItem{label: match.Date + ": " + match.Line, value: match.Date}
		}
		m.picker.SetItems(items)
		m.picker.Select(0)
		m.layout()
		return m, nil

	case copiedMsg:
		m.status = "copied to clipboard"
		return m, clearStatusAfter(2 * time.Second)

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case errMsg:
		// auto-clear like every other status; a stuck error line otherwise
		// lingers over unrelated actions
		m.status = "error: " + msg.err.Error()
		return m, clearStatusAfter(5 * time.Second)

	case tea.KeyMsg:
		return m.onKey(msg)
	}

	if m.mode == modeInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	if m.mode == modeDays {
		var cmd tea.Cmd
		m.dayFilter, cmd = m.dayFilter.Update(msg)
		return m, cmd
	}

	if m.mode == modeSearch {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) onKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeInput:
		return m.onInputKey(msg)
	case modeProjects:
		return m.onPickerKey(msg)
	case modeTodos:
		return m.onTodoKey(msg)
	case modeDays:
		return m.onDayPickerKey(msg)
	case modeSearch:
		return m.onSearchKey(msg)
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

	case key.Matches(msg, m.keys.Todos):
		day, ok := m.selectedDay()
		if !ok {
			return m, nil
		}
		m.mode = modeTodos
		m.status = ""
		m.picker.SetItems(nil)
		return m, loadTodos(m.projectPath, day)

	case key.Matches(msg, m.keys.Older):
		if m.dayIdx < len(m.days)-1 {
			m.dayIdx++
			return m, m.renderSelected()
		}
		return m, nil

	case key.Matches(msg, m.keys.Newer):
		if m.dayIdx > 0 {
			m.dayIdx--
			return m, m.renderSelected()
		}
		return m, nil

	case key.Matches(msg, m.keys.JumpDay):
		m.mode = modeDays
		m.status = ""
		m.dayFilter.Reset()
		m.picker.SetItems(dayPickerItems(m.days, m.today, ""))
		m.picker.Select(m.dayIdx)
		m.layout()
		return m, m.dayFilter.Focus()

	case key.Matches(msg, m.keys.Search):
		m.mode = modeSearch
		m.status = ""
		m.searchInput.Reset()
		m.picker.SetItems(nil)
		m.layout()
		return m, m.searchInput.Focus()

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.layout()
		return m, nil

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

func (m Model) onDayPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		item, ok := m.picker.SelectedItem().(pickerItem)
		m.mode = modeBrowse
		m.dayFilter.Blur()
		if !ok {
			return m, nil
		}
		m.selectDay(item.value)
		return m, m.renderSelected()

	case tea.KeyEsc:
		m.mode = modeBrowse
		m.dayFilter.Blur()
		return m, nil

	case tea.KeyUp, tea.KeyCtrlP:
		m.picker.CursorUp()
		return m, nil

	case tea.KeyDown, tea.KeyCtrlN:
		m.picker.CursorDown()
		return m, nil
	}

	var cmd tea.Cmd
	m.dayFilter, cmd = m.dayFilter.Update(msg)
	m.picker.SetItems(dayPickerItems(m.days, m.today, strings.TrimSpace(m.dayFilter.Value())))
	m.picker.Select(0)
	m.layout()
	return m, cmd
}

func (m Model) onSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		item, ok := m.picker.SelectedItem().(pickerItem)
		m.mode = modeBrowse
		m.searchInput.Blur()
		if !ok {
			return m, nil
		}
		m.selectDay(item.value)
		return m, m.renderSelected()

	case tea.KeyEsc:
		m.mode = modeBrowse
		m.searchInput.Blur()
		return m, nil

	case tea.KeyUp, tea.KeyCtrlP:
		m.picker.CursorUp()
		return m, nil

	case tea.KeyDown, tea.KeyCtrlN:
		m.picker.CursorDown()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.searchSeq++
	return m, tea.Batch(cmd, debounceSearch(m.searchSeq))
}

func (m Model) onPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		item, ok := m.picker.SelectedItem().(pickerItem)
		if !ok {
			m.mode = modeBrowse
			return m, nil
		}
		return m, switchProject(item.value)

	case tea.KeyEsc:
		m.mode = modeBrowse
		return m, nil
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m Model) onTodoKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeySpace:
		idx := m.picker.Index()
		if idx < 0 || idx >= len(m.todos) {
			return m, nil
		}
		day, ok := m.selectedDay()
		if !ok {
			return m, nil
		}
		return m, toggleTodo(m.projectPath, day, m.todos[idx])

	case tea.KeyEnter, tea.KeyEsc:
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
