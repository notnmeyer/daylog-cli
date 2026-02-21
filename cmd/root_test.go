package cmd

import "testing"

func TestFormatStdinContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "single line with trailing newline",
			input:    "hello world\n",
			expected: "hello world",
		},
		{
			name:     "multi-line becomes code block",
			input:    "hello\nworld",
			expected: "```\nhello\nworld\n```",
		},
		{
			name:     "multi-line with trailing newline becomes code block",
			input:    "hello\nworld\n",
			expected: "```\nhello\nworld\n```",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStdinContent(tt.input)
			if result != tt.expected {
				t.Errorf("formatStdinContent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
