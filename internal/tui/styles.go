package tui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	header      lipgloss.Style
	focusedPane lipgloss.Style
	blurredPane lipgloss.Style
	selectedDay lipgloss.Style
	normalDay   lipgloss.Style
	dimDay      lipgloss.Style
	footer      lipgloss.Style
	errText     lipgloss.Style
}

func defaultStyles() styles {
	return styles{
		header:      lipgloss.NewStyle().Bold(true).Padding(0, 1),
		focusedPane: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1),
		blurredPane: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1),
		selectedDay: lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true),
		normalDay:   lipgloss.NewStyle(),
		dimDay:      lipgloss.NewStyle().Faint(true),
		footer:      lipgloss.NewStyle().Faint(true).Padding(0, 1),
		errText:     lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(0, 1),
	}
}

func lipglossHeight(s string) int {
	return lipgloss.Height(s)
}
