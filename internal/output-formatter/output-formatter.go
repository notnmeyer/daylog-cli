package outputFormatter

import (
	"github.com/charmbracelet/glamour"
)

var OutputFormats = []string{
	"markdown", "md",
	"text",
	"web",
}

func Format(format, content string) (string, error) {
	switch format {
	case "markdown", "md":
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(100),
		)

		var err error
		content, err = renderer.Render(content)
		if err != nil {
			return "", err
		}
	}

	return content, nil
}
