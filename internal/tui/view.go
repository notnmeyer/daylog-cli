package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

// pickerItem is the shared item for the day/project/todo pickers.
// label is what renders; value carries the selection payload
type pickerItem struct {
	label string
	value string
}

func (p pickerItem) FilterValue() string { return p.label }

type pickerDelegate struct {
	styles styles
}

func (pickerDelegate) Height() int                             { return 1 }
func (pickerDelegate) Spacing() int                            { return 0 }
func (pickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d pickerDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	p, ok := item.(pickerItem)
	if !ok {
		return
	}

	// the list only clamps height, so a long row (a full log line in
	// search) would widen the modal past the screen; truncate to the width
	if index == m.Index() {
		fmt.Fprint(w, d.styles.selected.Render(ansi.Truncate("› "+p.label, m.Width(), "…")))
		return
	}
	fmt.Fprint(w, d.styles.normal.Render(ansi.Truncate("  "+p.label, m.Width(), "…")))
}

// dayPickerItems builds picker items for days, fuzzy-filtered by query.
// matching considers both the raw date and its label, so "jun 17",
// "0617", and "yesterday" all work
func dayPickerItems(days []string, today time.Time, query string) []list.Item {
	labels := make([]string, len(days))
	targets := make([]string, len(days))
	for i, day := range days {
		primary, secondary := dayLabel(day, today)
		labels[i] = primary + " · " + secondary
		targets[i] = day + " " + labels[i]
	}

	if query == "" {
		items := make([]list.Item, len(days))
		for i, day := range days {
			items[i] = pickerItem{label: labels[i], value: day}
		}
		return items
	}

	matches := fuzzy.Find(query, targets)
	items := make([]list.Item, len(matches))
	for i, match := range matches {
		items[i] = pickerItem{label: labels[match.Index], value: days[match.Index]}
	}
	return items
}

// dayLabel turns "2026/07/10" into a compact date plus an addressable
// hint — something that works as `daylog -- <hint>`: "today", "yesterday",
// "N days ago" within the last week, or the full date beyond that
func dayLabel(day string, today time.Time) (string, string) {
	t, err := time.Parse(dayFormat, day)
	if err != nil {
		return day, ""
	}

	primary := t.Format("Jan 02")

	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	dayDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	daysAgo := int(todayDate.Sub(dayDate).Hours() / 24)

	var secondary string
	switch {
	case daysAgo == 0:
		secondary = "today"
	case daysAgo == 1:
		secondary = "yesterday"
	case daysAgo > 1 && daysAgo < 7:
		secondary = fmt.Sprintf("%d days ago", daysAgo)
	default:
		secondary = day
	}

	return primary, secondary
}

func (m Model) View() string {
	if !m.ready {
		return "loading…"
	}

	body := m.styles.pane.Render(m.vp.View())

	switch m.mode {
	case modeProjects:
		body = m.modalView("projects", "", "", lipgloss.Height(body))
	case modeTodos:
		day, _ := m.selectedDay()
		body = m.modalView("todos · "+day, "", "", lipgloss.Height(body))
	case modeDays:
		body = m.modalView("days", m.dayFilter.View(), "no matching days", lipgloss.Height(body))
	case modeSearch:
		body = m.modalView("search", m.searchInput.View(), m.searchEmpty(), lipgloss.Height(body))
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.headerView(), body, m.footerView())
}

// searchEmpty distinguishes the pre-search prompt from a query that
// found nothing
func (m Model) searchEmpty() string {
	if strings.TrimSpace(m.searchInput.Value()) == "" {
		return "type to search all logs"
	}
	return fmt.Sprintf("no matches for %q", strings.TrimSpace(m.searchInput.Value()))
}

func (m Model) headerView() string {
	header := "daylog · " + m.project
	if day, ok := m.selectedDay(); ok {
		primary, secondary := dayLabel(day, m.today)
		if t, err := time.Parse(dayFormat, day); err == nil {
			primary = t.Format("Mon Jan 02 2006")
		}
		header += " · " + primary
		// only show the relative hint (today/yesterday/N days ago); for
		// older days the secondary is just the raw date, which duplicates
		// what the primary already conveys
		if secondary != "" && secondary != day {
			header += " · " + secondary
		}
	}
	return m.styles.fit(m.styles.header, m.width).Render(header)
}

// modalView renders every picker as one centered modal in place of the
// body: a title, an optional live-filter input, then the list. passing an
// empty input omits the input row so static pickers hug their title.
// empty is the lowercase placeholder shown in place of bubbles' default
// "No items." when the list is empty
func (m Model) modalView(title, input, empty string, height int) string {
	rows := []string{m.styles.modalTitle.Render(title)}
	if input != "" {
		rows = append(rows, input, "")
	}
	if len(m.picker.Items()) == 0 && empty != "" {
		rows = append(rows, m.styles.normal.Render("  "+empty))
	} else {
		rows = append(rows, m.picker.View())
	}

	box := m.styles.modal.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) footerView() string {
	footer := m.styles.fit(m.styles.footer, m.width)

	switch m.mode {
	case modeInput:
		return footer.UnsetFaint().Render(m.input.View())
	case modeProjects:
		return footer.Render("↑/↓ move • enter select • esc cancel")
	case modeTodos:
		return footer.Render("space toggle • enter done • esc cancel")
	case modeDays:
		return footer.Render("type to filter • ↑/↓ move • enter select • esc cancel")
	case modeSearch:
		return footer.Render("type to search • ↑/↓ move • enter select • esc cancel")
	}

	if m.status != "" {
		if strings.HasPrefix(m.status, "error:") {
			return m.styles.fit(m.styles.errText, m.width).Render(m.status)
		}
		return footer.Render(m.status)
	}
	return footer.Render(m.help.View(m.keys))
}
