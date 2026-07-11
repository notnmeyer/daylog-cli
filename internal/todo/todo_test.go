package todo

import (
	"slices"
	"strings"
	"testing"
)

const sampleLog = `# 2026/07/10

- plain entry
- TODO: buy tortillas
- [ ] TODO: water plants
- [x] TODO: eat a burrito
- fix the TODO in the parser
thinking about TODOs
  - TODO: indented note
-[ ] TODO: no space after dash`

func TestIsTodo(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "bare todo", line: "- TODO: x", want: true},
		{name: "unchecked checkbox", line: "- [ ] TODO: x", want: true},
		{name: "checked checkbox", line: "- [x] TODO: x", want: true},
		{name: "TODO mid-item", line: "- fix the TODO in the parser", want: true},
		{name: "plain list item", line: "- plain entry", want: false},
		{name: "prose mentioning TODO", line: "thinking about TODOs", want: false},
		{name: "indented list item", line: "  - TODO: x", want: false},
		{name: "no space after dash", line: "-[ ] TODO: x", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTodo(tt.line); got != tt.want {
				t.Errorf("IsTodo(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Item
	}{
		{
			name: "mixed content",
			in:   sampleLog,
			want: []Item{
				{Line: 3, Text: "TODO: buy tortillas", Done: false},
				{Line: 4, Text: "TODO: water plants", Done: false},
				{Line: 5, Text: "TODO: eat a burrito", Done: true},
				{Line: 6, Text: "fix the TODO in the parser", Done: false},
			},
		},
		{
			name: "no todos",
			in:   "# 2026/07/10\n\n- plain entry\n",
			want: nil,
		},
		{
			name: "empty content",
			in:   "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d items, got %d: %+v", len(tt.want), len(got), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("item %d: expected %+v, got %+v", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func TestToggle(t *testing.T) {
	tests := []struct {
		name     string
		line     int
		wantLine string
		wantErr  bool
	}{
		{name: "bare todo gains a checked box", line: 3, wantLine: "- [x] TODO: buy tortillas"},
		{name: "unchecked becomes checked", line: 4, wantLine: "- [x] TODO: water plants"},
		{name: "checked becomes unchecked", line: 5, wantLine: "- [ ] TODO: eat a burrito"},
		{name: "plain line errors", line: 2, wantErr: true},
		{name: "prose line errors", line: 7, wantErr: true},
		{name: "out of range errors", line: 99, wantErr: true},
		{name: "negative errors", line: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Toggle(sampleLog, tt.line)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			lines := strings.Split(got, "\n")
			if lines[tt.line] != tt.wantLine {
				t.Errorf("expected %q, got %q", tt.wantLine, lines[tt.line])
			}

			// everything else untouched
			orig := strings.Split(sampleLog, "\n")
			for i := range orig {
				if i != tt.line && lines[i] != orig[i] {
					t.Errorf("line %d changed unexpectedly: %q -> %q", i, orig[i], lines[i])
				}
			}
		})
	}
}

func TestToggleRoundTrip(t *testing.T) {
	// checkbox items round-trip exactly
	once, err := Toggle(sampleLog, 4)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := Toggle(once, 4)
	if err != nil {
		t.Fatal(err)
	}
	if twice != sampleLog {
		t.Error("expected double toggle to restore original content")
	}

	// a bare todo normalizes to checkbox form after a round trip
	once, err = Toggle(sampleLog, 3)
	if err != nil {
		t.Fatal(err)
	}
	twice, err = Toggle(once, 3)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Split(twice, "\n")[3] != "- [ ] TODO: buy tortillas" {
		t.Errorf("expected bare todo normalized to unchecked checkbox, got %q", strings.Split(twice, "\n")[3])
	}
}

func TestUnfinished(t *testing.T) {
	got := Unfinished(sampleLog)
	want := []string{
		"- TODO: buy tortillas",
		"- [ ] TODO: water plants",
		"- fix the TODO in the parser",
	}
	if !slices.Equal(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestFormatEntry(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		want   string
		wantOK bool
	}{
		{name: "bare TODO", in: "TODO: buy milk", want: "- [ ] TODO: buy milk", wantOK: true},
		{name: "TODO without colon", in: "TODO buy milk", want: "- [ ] TODO buy milk", wantOK: true},
		{name: "list item TODO", in: "- TODO: buy milk", want: "- [ ] TODO: buy milk", wantOK: true},
		{name: "already a checkbox", in: "- [ ] TODO: buy milk", want: "- [ ] TODO: buy milk", wantOK: false},
		{name: "plain entry", in: "ate a burrito", want: "ate a burrito", wantOK: false},
		{name: "plain list item", in: "- ate a burrito", want: "- ate a burrito", wantOK: false},
		{name: "TODO mid-entry is not a todo entry", in: "wrote a TODO list", want: "wrote a TODO list", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := FormatEntry(tt.in)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("FormatEntry(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
