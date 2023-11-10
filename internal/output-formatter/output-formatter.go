package outputFormatter

import (
	"fmt"

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
	if !contains(OutputFormats, format) {
		return "", fmt.Errorf("output must be one of %v\n", OutputFormats)
	}

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

func isMarkdownFormat() {

}

func contains(list []string, item string) bool {
	for _, val := range list {
		if val == item {
			return true
		}
	}
	return false
}
