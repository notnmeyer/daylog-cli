package todo

import (
	"fmt"
	"strings"
)

const (
	listPrefix    = "- "
	uncheckedMark = "- [ ] "
	checkedMark   = "- [x] "
)

type Item struct {
	// 0-based line index in the file
	Line int
	Text string
	Done bool
}

// IsTodo reports whether a line is a todo: a list item that mentions TODO,
// with or without a checkbox
func IsTodo(line string) bool {
	return strings.HasPrefix(line, listPrefix) && strings.Contains(line, "TODO")
}

func isChecked(line string) bool {
	return strings.HasPrefix(line, checkedMark) || strings.HasPrefix(line, "- [X] ")
}

func Parse(content string) []Item {
	var items []Item
	for i, line := range strings.Split(content, "\n") {
		if !IsTodo(line) {
			continue
		}
		items = append(items, Item{Line: i, Text: itemText(line), Done: isChecked(line)})
	}
	return items
}

// itemText strips a todo line's checkbox/list marker down to its text
func itemText(line string) string {
	switch {
	case isChecked(line):
		return strings.TrimSpace(line[len(checkedMark):])
	case strings.HasPrefix(line, uncheckedMark):
		return strings.TrimSpace(line[len(uncheckedMark):])
	default:
		return strings.TrimSpace(strings.TrimPrefix(line, listPrefix))
	}
}

// ToggleMatching flips the checkbox on item's line, but only if the
// line still holds that todo's text. this guards against an edit or
// append shifting lines between the picker loading and the toggle
func ToggleMatching(content string, item Item) (string, error) {
	lines := strings.Split(content, "\n")
	if item.Line < 0 || item.Line >= len(lines) {
		return "", fmt.Errorf("todo %q is no longer at line %d", item.Text, item.Line)
	}
	// compare the extracted text exactly; a substring test would let a
	// stale index toggle a longer todo that merely contains this one's text
	if !IsTodo(lines[item.Line]) || itemText(lines[item.Line]) != item.Text {
		return "", fmt.Errorf("todo %q moved; reopen the list", item.Text)
	}
	return Toggle(content, item.Line)
}

// Toggle flips the checkbox on the given todo line. a todo without a
// checkbox gains a checked one
func Toggle(content string, line int) (string, error) {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return "", fmt.Errorf("line %d is out of range", line)
	}

	l := lines[line]
	if !IsTodo(l) {
		return "", fmt.Errorf("line %d is not a todo", line)
	}

	switch {
	case isChecked(l):
		lines[line] = uncheckedMark + l[len(checkedMark):]
	case strings.HasPrefix(l, uncheckedMark):
		lines[line] = checkedMark + l[len(uncheckedMark):]
	default:
		lines[line] = checkedMark + strings.TrimPrefix(l, listPrefix)
	}

	return strings.Join(lines, "\n"), nil
}

// Unfinished returns the raw lines of unchecked todos, for carrying
// into the next day's log
func Unfinished(content string) []string {
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		if IsTodo(line) && !isChecked(line) {
			lines = append(lines, line)
		}
	}
	return lines
}

// FormatEntry turns an entry that reads like a todo into an unchecked
// checkbox item
func FormatEntry(msg string) (string, bool) {
	if rest, ok := strings.CutPrefix(msg, listPrefix); ok {
		if strings.HasPrefix(rest, "TODO") {
			return uncheckedMark + rest, true
		}
		return msg, false
	}

	if strings.HasPrefix(msg, "TODO") {
		return uncheckedMark + msg, true
	}

	return msg, false
}
