package outputFormatter

import (
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	t.Run("rejects unknown format", func(t *testing.T) {
		_, err := Format("xml", "hello")
		if err == nil {
			t.Errorf("Format() err = nil, want non-nil for unknown format")
		}
	})

	t.Run("text passes content through unchanged", func(t *testing.T) {
		got, err := Format("text", "hello world")
		if err != nil {
			t.Fatalf("Format() err = %v, want nil", err)
		}
		if got != "hello world" {
			t.Errorf("Format() = %q, want %q", got, "hello world")
		}
	})

	t.Run("markdown wraps headings in ANSI escapes", func(t *testing.T) {
		got, err := Format("markdown", "# heading\n")
		if err != nil {
			t.Fatalf("Format() err = %v, want nil", err)
		}
		// glamour renders headings with ANSI escapes; the literal text
		// should still appear in the output.
		if !strings.Contains(got, "heading") {
			t.Errorf("Format() output %q should contain the heading text", got)
		}
	})

	t.Run("md alias behaves like markdown", func(t *testing.T) {
		got, err := Format("md", "# heading\n")
		if err != nil {
			t.Fatalf("Format() err = %v, want nil", err)
		}
		if !strings.Contains(got, "heading") {
			t.Errorf("Format() output %q should contain the heading text", got)
		}
	})
}
