package cmd

import (
	"testing"
)

func TestConvertLogToDisplayName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2025/12/02/log.md", "2025/12/02"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertLogToDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("convertLogToDisplayName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
