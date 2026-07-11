package tui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	header     lipgloss.Style
	pane       lipgloss.Style
	modal      lipgloss.Style
	modalTitle lipgloss.Style
	selected   lipgloss.Style
	normal     lipgloss.Style
	footer     lipgloss.Style
	errText    lipgloss.Style
}

func defaultStyles() styles {
	return styles{
		header:     lipgloss.NewStyle().Bold(true).Padding(0, 1),
		pane:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1),
		modal:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(1, 2),
		modalTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).MarginBottom(1),
		selected:   lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true),
		normal:     lipgloss.NewStyle(),
		footer:     lipgloss.NewStyle().Faint(true).Padding(0, 1),
		errText:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(0, 1),
	}
}

func lipglossHeight(s string) int {
	return lipgloss.Height(s)
}
