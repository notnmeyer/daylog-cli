package tui

import (
	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"
)

type mdRenderer struct {
	style string
}

// newMDRenderer detects the terminal background before bubbletea puts the
// tty in raw mode — glamour's WithAutoStyle queries the terminal via OSC,
// and mid-program that response gets swallowed by bubbletea's input reader
func newMDRenderer() mdRenderer {
	style := "light"
	if termenv.HasDarkBackground() {
		style = "dark"
	}
	return mdRenderer{style: style}
}

func (r mdRenderer) render(content string, width int) (string, error) {
	tr, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(r.style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return tr.Render(content)
}
