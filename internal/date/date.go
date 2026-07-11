// package date parses the casual date references daylog accepts on the
// command line. it replaces go-dateparser, whose locale tables cost ~90ms
// of process startup on every invocation
package date

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	agoRe = regexp.MustCompile(`^(\d+|an?) (day|week|month|year)s? ago$`)
	inRe  = regexp.MustCompile(`^in (\d+|an?) (day|week|month|year)s?$`)
)

var weekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "sun": time.Sunday,
	"monday": time.Monday, "mon": time.Monday,
	"tuesday": time.Tuesday, "tue": time.Tuesday,
	"wednesday": time.Wednesday, "wed": time.Wednesday,
	"thursday": time.Thursday, "thu": time.Thursday,
	"friday": time.Friday, "fri": time.Friday,
	"saturday": time.Saturday, "sat": time.Saturday,
}

// year-less layouts get the current year filled in
// non-padded tokens accept both "1" and "01"; padded tokens require two digits
var layouts = []string{
	"2006/1/2",
	"2006-1-2",
	"1/2",
	"1-2",
	"Jan 2 2006",
	"Jan 2",
	"January 2 2006",
	"January 2",
}

// Parse resolves input to a date relative to now. weekday names resolve to
// the most recent occurrence strictly before now
func Parse(input string, now time.Time) (time.Time, error) {
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "today", "now":
		return now, nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	case "tomorrow":
		return now.AddDate(0, 0, 1), nil
	}

	if m := agoRe.FindStringSubmatch(input); m != nil {
		return shift(now, m[1], m[2], -1), nil
	}
	if m := inRe.FindStringSubmatch(input); m != nil {
		return shift(now, m[1], m[2], 1), nil
	}

	if wd, ok := weekdays[strings.TrimPrefix(input, "last ")]; ok {
		delta := int(now.Weekday() - wd)
		if delta <= 0 {
			delta += 7
		}
		return now.AddDate(0, 0, -delta), nil
	}

	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, input, now.Location())
		if err != nil {
			continue
		}
		if t.Year() == 0 {
			t = time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
		}
		return t, nil
	}

	return time.Time{}, fmt.Errorf("couldn't parse date %q", input)
}

func shift(now time.Time, count, unit string, sign int) time.Time {
	n := 1
	if count != "a" && count != "an" {
		n, _ = strconv.Atoi(count)
	}
	n *= sign

	switch unit {
	case "day":
		return now.AddDate(0, 0, n)
	case "week":
		return now.AddDate(0, 0, 7*n)
	case "month":
		return now.AddDate(0, n, 0)
	default:
		return now.AddDate(n, 0, 0)
	}
}
