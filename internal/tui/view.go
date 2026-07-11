package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type dayItem string

func (d dayItem) FilterValue() string { return string(d) }

type dayDelegate struct {
	styles styles
	today  time.Time
}

func (dayDelegate) Height() int                             { return 2 }
func (dayDelegate) Spacing() int                            { return 1 }
func (dayDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d dayDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	day, ok := item.(dayItem)
	if !ok {
		return
	}

	primary, secondary := dayLabel(string(day), d.today)

	primaryStyle := d.styles.normalDay
	prefix := "  "
	if index == m.Index() {
		primaryStyle = d.styles.selectedDay
		prefix = "› "
	}

	fmt.Fprintf(w, "%s\n%s",
		primaryStyle.Render(prefix+primary),
		d.styles.dimDay.Render("  "+secondary),
	)
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

type pickerItem string

func (p pickerItem) FilterValue() string { return string(p) }

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
		fmt.Fprint(w, d.styles.selectedDay.Render("› "+string(p)))
		return
	}
	fmt.Fprint(w, d.styles.normalDay.Render("  "+string(p)))
}

func (m Model) View() string {
	if !m.ready {
		return "loading…"
	}

	listStyle, vpStyle := m.styles.blurredPane, m.styles.focusedPane
	if m.focus == focusDays {
		listStyle, vpStyle = m.styles.focusedPane, m.styles.blurredPane
	}

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		listStyle.Render(m.days.View()),
		vpStyle.Render(m.vp.View()),
	)

	if m.mode == modeProjects {
		body = m.pickerView("switch project", lipgloss.Height(body))
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.headerView(), body, m.footerView())
}

// pickerView renders the picker as a centered modal in place of the body
func (m Model) pickerView(title string, height int) string {
	box := m.styles.focusedPane.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.header.Render(title),
		m.picker.View(),
	))

	return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) headerView() string {
	return m.styles.header.Render("daylog · " + m.project)
}

func (m Model) footerView() string {
	if m.mode == modeInput {
		return m.styles.footer.UnsetFaint().Render(m.input.View())
	}

	if m.mode == modeProjects {
		return m.styles.footer.Render("enter select • esc cancel")
	}

	if m.status != "" {
		if strings.HasPrefix(m.status, "error:") {
			return m.styles.errText.Render(m.status)
		}
		return m.styles.footer.Render(m.status)
	}
	return m.styles.footer.Render(m.help.View(m.keys))
}
