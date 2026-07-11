package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	if index == m.Index() {
		fmt.Fprint(w, d.styles.selected.Render("› "+p.label))
		return
	}
	fmt.Fprint(w, d.styles.normal.Render("  "+p.label))
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
		body = m.pickerView("switch project", lipgloss.Height(body))
	case modeTodos:
		day, _ := m.selectedDay()
		body = m.pickerView("todos · "+day, lipgloss.Height(body))
	case modeDays:
		body = m.dayPickerView(lipgloss.Height(body))
	case modeSearch:
		body = m.searchView(lipgloss.Height(body))
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.headerView(), body, m.footerView())
}

func (m Model) headerView() string {
	header := "daylog · " + m.project
	if day, ok := m.selectedDay(); ok {
		primary, secondary := dayLabel(day, m.today)
		header += " · " + primary
		if secondary != "" {
			header += " · " + secondary
		}
	}
	return m.styles.header.Render(header)
}

// pickerView renders the picker as a centered modal in place of the body
func (m Model) pickerView(title string, height int) string {
	box := m.styles.pane.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.header.Render(title),
		m.picker.View(),
	))

	return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center, box)
}

// dayPickerView is the picker plus a live fuzzy-filter input
func (m Model) dayPickerView(height int) string {
	box := m.styles.pane.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.header.Render("jump to day"),
		m.dayFilter.View(),
		m.picker.View(),
	))

	return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center, box)
}

// searchView is the picker plus a live search input; rows match the
// CLI's `date: line` output
func (m Model) searchView(height int) string {
	box := m.styles.pane.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.header.Render("search"),
		m.searchInput.View(),
		m.picker.View(),
	))

	return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) footerView() string {
	switch m.mode {
	case modeInput:
		return m.styles.footer.UnsetFaint().Render(m.input.View())
	case modeProjects:
		return m.styles.footer.Render("enter select • esc cancel")
	case modeTodos:
		return m.styles.footer.Render("space toggle • enter/esc close")
	case modeDays:
		return m.styles.footer.Render("type to filter • ↑/↓ move • enter jump • esc cancel")
	case modeSearch:
		return m.styles.footer.Render("type to search • ↑/↓ move • enter open • esc cancel")
	}

	if m.status != "" {
		if strings.HasPrefix(m.status, "error:") {
			return m.styles.errText.Render(m.status)
		}
		return m.styles.footer.Render(m.status)
	}
	return m.styles.footer.Render(m.help.View(m.keys))
}
