package date

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	// friday
	now := time.Date(2026, 7, 10, 15, 4, 5, 0, time.Local)

	tests := []struct {
		input    string
		expected string
	}{
		{"today", "2026/07/10"},
		{"now", "2026/07/10"},
		{"yesterday", "2026/07/09"},
		{"tomorrow", "2026/07/11"},
		{"TODAY", "2026/07/10"},
		{" yesterday ", "2026/07/09"},

		{"1 day ago", "2026/07/09"},
		{"3 days ago", "2026/07/07"},
		{"a day ago", "2026/07/09"},
		{"a week ago", "2026/07/03"},
		{"2 weeks ago", "2026/06/26"},
		{"2 months ago", "2026/05/10"},
		{"1 year ago", "2025/07/10"},
		{"in 2 days", "2026/07/12"},
		{"in 1 week", "2026/07/17"},
		{"in a month", "2026/08/10"},

		// weekdays resolve to the most recent one strictly before now
		{"friday", "2026/07/03"},
		{"last friday", "2026/07/03"},
		{"thursday", "2026/07/09"},
		{"monday", "2026/07/06"},
		{"sat", "2026/07/04"},
		{"last sunday", "2026/07/05"},

		{"2023/01/07", "2023/01/07"},
		{"2023-01-07", "2023/01/07"},
		{"2023/1/7", "2023/01/07"},
		{"01/07", "2026/01/07"},
		{"1/7", "2026/01/07"},
		{"12-25", "2026/12/25"},
		{"jan 5", "2026/01/05"},
		{"Jan 5 2024", "2024/01/05"},
		{"january 5", "2026/01/05"},
		{"march 5", "2026/03/05"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input, now)
			if err != nil {
				t.Fatalf("Parse(%q) err = %v, want nil", tt.input, err)
			}
			if got := result.Format("2006/01/02"); got != tt.expected {
				t.Errorf("Parse(%q) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	now := time.Date(2026, 7, 10, 15, 4, 5, 0, time.Local)

	for _, input := range []string{"banana", "", "next friday", "13/45", "ago", "5 fortnights ago"} {
		t.Run(input, func(t *testing.T) {
			if _, err := Parse(input, now); err == nil {
				t.Errorf("Parse(%q) err = nil, want error", input)
			}
		})
	}
}
