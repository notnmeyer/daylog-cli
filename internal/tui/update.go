package tui

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/notnmeyer/daylog-cli/internal/date"
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
		m.noLogToday = msg.noLogToday
		m.dayIdx = 0
		if idx := slices.Index(m.days, prev); idx >= 0 {
			m.dayIdx = idx
		}
		if m.mode == modeLedger {
			return m, m.refreshLedger()
		}
		return m, m.renderSelected()

	case previewsLoadedMsg:
		maps.Copy(m.previews, msg.previews)
		if m.mode == modeLedger {
			// re-render rows now that previews are in; keep the cursor put
			m.rebuildLedgerItems()
		}
		return m, nil

	case dayRenderedMsg:
		// drop a stale render: if the user navigated to another day (or into
		// the ledger) before this resolved, painting it would show the wrong
		// day's content under the current header
		if day, ok := m.selectedDay(); !ok || day != msg.day || m.mode == modeLedger {
			return m, nil
		}
		m.vp.SetContent(msg.content)
		m.vp.GotoTop()
		return m, nil

	case entryAppendedMsg:
		// the day's preview is now stale; evict it so the reload re-reads it
		delete(m.previews, msg.day)
		return m, loadDays(m.projectPath, m.today)

	case editorFinishedMsg:
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
		}
		// the edited day's preview may have changed; evict so it re-reads
		delete(m.previews, msg.day)
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
		// previews are keyed by date, which collides across projects — clear
		// them so today (or any day) can't render another project's content
		m.previews = map[string][]string{}
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

	if m.mode == modeLedger && m.dayFilter.Focused() {
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
	case modeLedger:
		return m.onLedgerKey(msg)
	case modeSearch:
		return m.onSearchKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Append):
		m.inputReturn = modeBrowse
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

	case key.Matches(msg, m.keys.JumpDay), msg.Type == tea.KeyEsc:
		// d and esc both return to the ledger (the day list); d preserves the
		// old "jump to day" muscle memory now that the ledger is that list
		return m, m.openLedger()

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

func (m Model) onLedgerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtering := m.dayFilter.Focused()

	switch msg.Type {
	case tea.KeyEnter:
		return m.openLedgerRow()

	case tea.KeyEsc:
		if filtering {
			// clear the filter back to the full ledger, staying home
			return m, m.openLedger()
		}
		return m, nil

	case tea.KeyUp, tea.KeyCtrlP:
		m.moveLedgerCursor(-1)
		return m, nil

	case tea.KeyDown, tea.KeyCtrlN:
		m.moveLedgerCursor(1)
		return m, nil
	}

	// unfiltered browse keys (once filtering, letters feed the filter instead)
	if !filtering {
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Append):
			// append to the selected day; the cursor already tracks it via
			// moveLedgerCursor keeping dayIdx in sync. return to the ledger
			if _, ok := m.selectedDay(); !ok {
				return m, nil
			}
			m.inputReturn = modeLedger
			m.mode = modeInput
			m.status = ""
			return m, m.input.Focus()

		case key.Matches(msg, m.keys.Edit):
			// open the selected day in $EDITOR; on finish the reload refreshes
			// the ledger in place
			day, ok := m.selectedDay()
			if !ok {
				return m, nil
			}
			return m, openEditor(m.projectPath, day)

		case key.Matches(msg, m.keys.Older):
			m.moveLedgerCursor(1) // older = further down the newest-first list
			return m, nil

		case key.Matches(msg, m.keys.Newer):
			m.moveLedgerCursor(-1)
			return m, nil

		case key.Matches(msg, m.keys.Projects):
			m.mode = modeProjects
			m.status = ""
			return m, loadProjects()

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.layout()
			return m, nil

		case msg.String() == "n" || key.Matches(msg, m.keys.Search):
			// both start the filter/new-day prompt in place
			m.status = ""
			m.dayFilter.Reset()
			m.rebuildLedgerItems()
			m.layout()
			return m, m.dayFilter.Focus()
		}
		// any other key in the unfiltered ledger is inert (j/k handled above)
		return m, nil
	}

	// filtering: feed the key to the filter input and re-rank live
	var cmd tea.Cmd
	m.dayFilter, cmd = m.dayFilter.Update(msg)
	m.rebuildLedgerItems()
	m.layout()
	return m, cmd
}

// openLedgerRow acts on the highlighted ledger row: open a day, resolve a typed
// date on the "＋ New day…" row, or ignore a gap divider
func (m Model) openLedgerRow() (tea.Model, tea.Cmd) {
	item, ok := m.picker.SelectedItem().(pickerItem)
	if !ok || !item.selectable() {
		return m, nil // gap divider or empty list
	}

	if item.kind == itemNewDay {
		day, err := m.resolveNewDay()
		if err != nil {
			m.status = "error: " + err.Error()
			return m, clearStatusAfter(3 * time.Second)
		}
		m.dayFilter.Blur()
		m.mode = modeBrowse
		m.selectDay(day)
		return m, m.renderSelected()
	}

	m.dayFilter.Blur()
	m.mode = modeBrowse
	m.selectDay(item.value)
	return m, m.renderSelected()
}

// moveLedgerCursor moves the picker cursor by dir (±1), skipping gap dividers,
// and keeps dayIdx in sync so the header/spine track the highlighted day
func (m *Model) moveLedgerCursor(dir int) {
	items := m.picker.Items()
	if len(items) == 0 {
		return
	}
	i := m.picker.Index()
	for {
		i += dir
		if i < 0 || i >= len(items) {
			return // ran off an end; leave the cursor put
		}
		if p, ok := items[i].(pickerItem); ok && p.selectable() {
			m.picker.Select(i)
			if p.kind == itemDay {
				if idx := slices.Index(m.days, p.value); idx >= 0 {
					m.dayIdx = idx
				}
			}
			return
		}
	}
}

// resolveNewDay parses the current filter text as a date reference and returns
// its YYYY/MM/DD form so a not-yet-logged day can be opened for backfill
func (m Model) resolveNewDay() (string, error) {
	text := strings.TrimSpace(m.dayFilter.Value())
	if text == "" {
		return "", fmt.Errorf("type a date first")
	}
	t, err := date.Parse(text, m.today)
	if err != nil {
		return "", fmt.Errorf("couldn't read %q as a date", text)
	}
	return t.Format(dayFormat), nil
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
		m.mode = m.inputReturn // back to wherever append was opened from

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
		m.mode = m.inputReturn
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
