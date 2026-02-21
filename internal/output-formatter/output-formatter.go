package outputFormatter

import (
	"fmt"
	"slices"

	"github.com/charmbracelet/glamour"
)

var MarkdownFormats = []string{
	"markdown", "md",
}

var OutputFormats = append(
	MarkdownFormats,
	"text",
)

func Format(format, content string) (string, error) {
	if !slices.Contains(OutputFormats, format) {
		return "", fmt.Errorf("output must be one of %v\n", OutputFormats)
	}

	switch format {
	case "markdown", "md":
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(0),
		)

		var err error
		content, err = renderer.Render(content)
		if err != nil {
			return "", err
		}
	}

	return content, nil
}
