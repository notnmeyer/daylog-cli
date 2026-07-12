package tui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	header     lipgloss.Style
	pane       lipgloss.Style
	modal      lipgloss.Style
	modalTitle lipgloss.Style
	selected   lipgloss.Style
	normal     lipgloss.Style
	gap        lipgloss.Style
	spine      lipgloss.Style
	spineOn    lipgloss.Style
	footer     lipgloss.Style
	errText    lipgloss.Style
}

func defaultStyles() styles {
	return styles{
		header:     lipgloss.NewStyle().Bold(true).Padding(0, 1),
		pane:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1),
		modal:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("170")).Padding(1, 2),
		modalTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).MarginBottom(1),
		selected:   lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true),
		normal:     lipgloss.NewStyle(),
		gap:        lipgloss.NewStyle().Faint(true),
		// the spine's ● has-log markers echo the picker accent; empty days
		// and the separator stay faint so the current-day highlight reads
		spine:      lipgloss.NewStyle().Faint(true),
		spineOn:    lipgloss.NewStyle().Foreground(lipgloss.Color("170")),
		footer:     lipgloss.NewStyle().Faint(true).Padding(0, 1),
		errText:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(0, 1),
	}
}

func lipglossHeight(s string) int {
	return lipgloss.Height(s)
}

// fit clamps a full-width chrome line (header/footer) to the terminal
// width so it stays a single row and never drags the layout wider than
// the screen
func (s styles) fit(style lipgloss.Style, w int) lipgloss.Style {
	if w < 1 {
		w = 1
	}
	return style.MaxWidth(w).MaxHeight(1)
}
