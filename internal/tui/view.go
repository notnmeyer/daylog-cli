package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

// itemKind distinguishes the special ledger rows from ordinary selectable
// rows. the zero value (itemDay) covers every existing picker item, so the
// project/todo/search pickers are unaffected
type itemKind int

const (
	itemDay    itemKind = iota // a real day: the anchor line of a day block
	itemGap                    // a non-selectable blank/gap spacer row
	itemNewDay                 // the "＋ New day…" affordance
	itemCont                   // a continuation line of the day block above it
)

// pickerItem is the shared item for the day/project/todo pickers.
// label is what renders; value carries the selection payload. ledger day rows
// additionally carry their columns (marker/date/weekday/preview) so the
// delegate can lay them out as an aligned table using the live list width.
//
// a ledger day is rendered as a BLOCK of rows: one itemDay anchor line
// carrying the marker + date rail + first log line, followed by itemCont lines
// (blank rail + subsequent log lines). every row in the block shares value =
// the day, so the cursor/selection can map a continuation back to its day
type pickerItem struct {
	label string
	value string
	kind  itemKind
	row   *ledgerRow // non-nil for ledger day/continuation rows; nil otherwise
}

// ledgerRow holds one rendered line of a ledger day block. the anchor line
// carries the date rail (marker/weekday/date); continuation lines leave those
// blank and only carry text
type ledgerRow struct {
	anchor  bool   // true for the first line of the block (draws the date rail)
	marker  string // ● / ○ / ＋ (anchor only)
	date    string // "Jun 17" / "Aug 09 2025" (anchor only)
	weekday string // "Mon".."Sun" (anchor only)
	text    string // this line's cleaned log text (or CTA / faint tick)
	bullet  bool   // prefix the text with a faint "• " (real log lines only)
	accent  bool   // render the whole block's rail/text in the accent color
	faint   bool   // render text faint (empty-day tick, today CTA, "…")
}

func (p pickerItem) FilterValue() string { return p.label }

// selectable reports whether enter should act on this row. only a day's anchor
// line is selectable; gaps and continuation lines are inert
func (p pickerItem) selectable() bool { return p.kind == itemDay || p.kind == itemNewDay }

// padRight pads s with spaces to display width w, measuring with lipgloss.Width
// so double-width glyphs (＋) and any styling don't throw the columns off. a
// string already at/over w is returned unchanged
func padRight(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

type pickerDelegate struct {
	styles styles
}

func (pickerDelegate) Height() int                             { return 1 }
func (pickerDelegate) Spacing() int                            { return 0 }
func (pickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d pickerDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	p, ok := item.(pickerItem)
	if !ok {
		return
	}

	// gap dividers are inert: faint, never marked selected even when the
	// cursor lands on one mid-scroll
	if p.kind == itemGap {
		fmt.Fprint(w, d.styles.gap.Render(ansi.Truncate("  "+p.label, m.Width(), "…")))
		return
	}

	// ledger day rows lay out as an aligned table using the live width. the
	// whole block of the selected day highlights, not just its anchor line, so
	// the multi-line entry reads as one selected unit
	if p.row != nil {
		selected := false
		if sel, ok := m.SelectedItem().(pickerItem); ok && sel.value == p.value && p.value != "" {
			selected = true
		}
		fmt.Fprint(w, d.renderLedgerRow(*p.row, m.Width(), selected))
		return
	}

	// the list only clamps height, so a long row (a full log line in
	// search) would widen the modal past the screen; truncate to the width
	if index == m.Index() {
		fmt.Fprint(w, d.styles.selected.Render(ansi.Truncate("› "+p.label, m.Width(), "…")))
		return
	}
	fmt.Fprint(w, d.styles.normal.Render(ansi.Truncate("  "+p.label, m.Width(), "…")))
}

// ledger column widths. the date column holds the widest date form,
// "Aug 09 2025" (11); the weekday is always 3. the fixed left block is
// "marker weekday date │ " with the rule the eye reads down
const (
	ledgerDateW = 11 // "Aug 09 2025" / "Jun 17     "
	ledgerWdayW = 3  // "Mon"
)

// renderLedgerRow lays out one line of a day block as a fixed grid: a left rail
// "marker weekday date │" (drawn only on the block's anchor line; blank on
// continuation lines) then the log text filling the rest. all padding is
// display-width aware so the ＋ (double-width) today marker keeps the same
// column origins as the ●/○ rows, and the │ rule stays dead-straight down the
// whole list
func (d pickerDelegate) renderLedgerRow(r ledgerRow, width int, selected bool) string {
	// the delegate owns the 2-col cursor gutter; lay the rest to width-2 and
	// pad to fill so the pane spans full width without an explicit .Width()
	inner := max(1, width-2)

	text := r.text
	if text == "" {
		text = "·" // an empty preview isn't orphaned — it shows a faint tick
	}
	// a faint "• " leads real log lines so lowercase entries in a block don't
	// run together; the plain (unstyled) form is used for width math
	bullet := ""
	if r.bullet {
		bullet = "• "
	}

	// plain (unstyled) rail so the column math is exact. continuation lines
	// keep the width but blank the marker/weekday/date, so the │ still aligns
	marker, weekday, date := r.marker, r.weekday, r.date
	if !r.anchor {
		marker, weekday, date = "", "", ""
	}
	rail := padRight(marker, 2) + " " + padRight(weekday, ledgerWdayW) + " " + padRight(date, ledgerDateW) + " │ "

	// selected: one accent wash over the plain, padded line. the "›" cursor
	// marks the block once (on its anchor); continuation lines keep the
	// highlight but a blank gutter so the arrow doesn't repeat down the block
	if selected {
		gutter := "  "
		if r.anchor {
			gutter = "› "
		}
		body := padRight(ansi.Truncate(rail+bullet+text, inner, "…"), inner)
		return d.styles.selected.Render(gutter + body)
	}

	// style each segment. the today block accents its rail + text; a faint
	// line (tick, CTA, "…") dims its text
	dateStyle := d.styles.normal
	markerStr := marker
	if r.accent && r.anchor {
		dateStyle = d.styles.selected
		markerStr = d.styles.selected.Render(marker)
	}
	railStyled := padRight(markerStr, 2) + " " +
		d.styles.gap.Render(padRight(weekday, ledgerWdayW)) + " " +
		dateStyle.Render(padRight(date, ledgerDateW)) +
		d.styles.gap.Render(" │ ")

	txt := text
	switch {
	case r.accent:
		txt = d.styles.selected.Render(text)
	case r.faint:
		txt = d.styles.gap.Render(text)
	}
	if bullet != "" {
		txt = d.styles.gap.Render(bullet) + txt // faint bullet before the text
	}

	// pad to the inner width so the rule aligns and the pane fills full width
	line := ansi.Truncate(railStyled+txt, inner, "…")
	if gap := inner - lipgloss.Width(rail) - lipgloss.Width(bullet) - lipgloss.Width(text); gap > 0 {
		line += strings.Repeat(" ", gap)
	}
	return "  " + line
}

// ledgerItems builds the landing ledger. each day becomes a BLOCK of rows: an
// anchor line (marker + date rail + first log line) plus a continuation line
// per further log line (blank rail + text), up to previewMaxLines then "…". a
// blank spacer separates blocks so days breathe. when filtering, a "＋ New
// day…" affordance leads and blocks collapse to their anchor line so more days
// fit; matching is fuzzy over date + label
func ledgerItems(days []string, today time.Time, previews map[string][]string, noLogToday bool, query string, contentMatches map[string]string) []list.Item {
	targets := make([]string, len(days))
	for i, day := range days {
		primary, secondary := dayLabel(day, today)
		targets[i] = day + " " + primary + " " + secondary
	}

	if query != "" {
		// a day appears if its DATE LABEL fuzzy-matches OR its LOG CONTENT
		// contains the query (contentMatches, from the debounced daylog.Search)
		dateHit := make(map[int]bool)
		for _, match := range fuzzy.Find(query, targets) {
			dateHit[match.Index] = true
		}

		items := make([]list.Item, 0, len(days)+1)
		items = append(items, pickerItem{
			label: "＋ New day…   type a date to start (e.g. \"jun 15\", \"6/15\", \"yesterday\")",
			kind:  itemNewDay,
		})
		// walk days in their canonical newest-first order so the union is
		// ordered and deduped by construction (each day emitted at most once,
		// even when it matches by both date and content)
		for i, day := range days {
			matchLine, isContent := contentMatches[day]
			if !dateHit[i] && !isContent {
				continue
			}
			// filtered rows collapse to just the anchor line to stay scannable.
			// a content match shows the LINE that matched (so you see why),
			// instead of the day's generic first-line preview
			preview := previews[day]
			if isContent && matchLine != "" {
				preview = []string{matchLine}
			}
			items = append(items, dayBlock(day, today, preview, noLogToday)[0])
		}
		return items
	}

	items := make([]list.Item, 0, len(days)*(previewMaxLines+1))
	for i, day := range days {
		if i > 0 {
			items = append(items, pickerItem{kind: itemGap}) // blank spacer
		}
		items = append(items, dayBlock(day, today, previews[day], noLogToday)...)
	}
	return items
}

// dayBlock builds the rows of one day's block: the anchor line carrying the
// date rail and the first log line, then a continuation line per further log
// line. a genuinely-logless today (noLogToday) gets the ＋ marker and a
// call-to-action. every row in the block shares value=day so the
// cursor/selection maps back to the day.
//
// noLogToday is the authoritative "empty today" signal from loadDays. it gates
// the ＋ CTA so it fires ONLY for a today with no log — never for a
// today-with-a-log whose preview simply hasn't loaded yet (that keeps ● + a
// faint tick until its real preview arrives, so no wrong CTA flashes)
func dayBlock(day string, today time.Time, preview []string, noLogToday bool) []list.Item {
	t, _ := time.Parse(dayFormat, day)
	marker := "●"
	weekday := t.Format("Mon")
	date := compactDate(t, today)

	// the text lines of the block; a logless today gets the CTA, other empty
	// days (incl. today-with-a-log awaiting its preview) a faint tick
	lines := preview
	accent := false
	faintFirst := false
	if len(lines) == 0 {
		if isToday(day, today) && noLogToday {
			marker = "＋"
			lines = []string{"nothing logged yet · a append · e edit"}
			accent = true
		} else {
			lines = []string{""} // renders as a faint tick
			faintFirst = true
		}
	}

	items := make([]list.Item, 0, len(lines))
	for i, text := range lines {
		row := ledgerRow{text: text, accent: accent}
		if i == 0 {
			row.anchor = true
			row.marker = marker
			row.weekday = weekday
			row.date = date
			row.faint = faintFirst
		}
		// real log lines get a bullet so lowercase entries don't run together;
		// the CTA, the empty-day tick, and the "…" overflow marker do not
		row.bullet = !accent && !faintFirst && text != previewEllipsis
		if text == previewEllipsis {
			// the "…" overflow marker reads faint
			row.faint = true
		}
		kind := itemCont
		if i == 0 {
			kind = itemDay
		}
		items = append(items, pickerItem{kind: kind, value: day, row: &row})
	}
	return items
}

// compactDate formats a date for the ledger's date column. the two most recent
// days read as "today"/"yesterday" (the friendly labels people actually use);
// otherwise it's "Jun 17" in the current year, "Aug 09 2025" beyond it — the
// year shown only when it differs so cross-year rows read correctly
func compactDate(t, today time.Time) string {
	switch daysBetween(today, t) {
	case 0:
		return "today"
	case 1:
		return "yesterday"
	}
	if t.Year() == today.Year() {
		return t.Format("Jan 02")
	}
	return t.Format("Jan 02 2006")
}

// daysBetween returns whole calendar days from b to a (a newer than b => >0),
// ignoring clock time within each day
func daysBetween(a, b time.Time) int {
	ad := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	bd := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(ad.Sub(bd).Hours() / 24)
}

// dayLabel turns "2026/07/10" into a compact date plus an addressable
// hint — something that works as `daylog -- <hint>`: "today", "yesterday",
// "N days ago" within the last week, or the full date beyond that
func dayLabel(day string, today time.Time) (string, string) {
	t, err := time.Parse(dayFormat, day)
	if err != nil {
		return day, ""
	}

	// show the year only when it isn't the current year, so a cross-year date
	// ("Aug 09 2025") reads correctly among this-year rows instead of looking
	// scrambled after the newest-first sort
	primary := t.Format("Jan 02")
	if t.Year() != today.Year() {
		primary = t.Format("Jan 02 2006")
	}

	daysAgo := daysBetween(today, t)

	var secondary string
	switch {
	case daysAgo == 0:
		secondary = "today"
	case daysAgo == 1:
		secondary = "yesterday"
	case daysAgo > 1 && daysAgo < 7:
		secondary = fmt.Sprintf("%d days ago", daysAgo)
	default:
		secondary = day
	}

	return primary, secondary
}

func (m Model) View() string {
	if !m.ready {
		return "loading…"
	}

	var body string
	if m.mode == modeLedger {
		body = m.ledgerView()
	} else {
		body = m.styles.pane.Render(m.vp.View())
	}

	switch m.mode {
	case modeProjects:
		body = m.modalView("projects", lipgloss.Height(body))
	case modeTodos:
		day, _ := m.selectedDay()
		body = m.modalView("todos · "+day, lipgloss.Height(body))
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.headerView(), body, m.footerView())
}


// hasLog reports whether a day has a non-empty log. today is force-prepended
// into m.days even when empty, so membership alone isn't enough — a day has a
// log iff it isn't the injected-empty today. the preview cache confirms real
// content; absent a cached preview we fall back to "not today-empty"
func (m Model) hasLog(day string) bool {
	if p, ok := m.previews[day]; ok {
		return len(p) > 0
	}
	// today is the only day that can appear without a log file, and only when
	// loadDays flagged it logless. today WITH a log (and every other day, all
	// from GetLogs's non-empty list) has a log even before its preview is read
	return !(isToday(day, m.today) && m.noLogToday)
}

// ledgerView renders the day list inside the bordered pane, sized to fill the
// terminal width so the list can breathe (short rows won't collapse the box),
// with a filter prompt line when the user is filtering
func (m Model) ledgerView() string {
	view := m.picker.View()
	if m.dayFilter.Focused() {
		view = lipgloss.JoinVertical(lipgloss.Left, m.dayFilter.View(), "", m.picker.View())
	} else if len(m.picker.Items()) == 0 {
		view = m.styles.normal.Render("no logs yet — press a to start today's")
	}

	// every ledger row is padded to the picker (content) width, so the pane
	// grows to full width on its own — no explicit .Width(), which wraps
	// content by a few cells once border+padding are added
	return m.styles.pane.Render(view)
}

func (m Model) headerView() string {
	header := "daylog · " + m.project

	// the ledger header counts logs instead of naming a single day
	if m.mode == modeLedger {
		n := m.logCount()
		suffix := fmt.Sprintf("%d logs · newest first", n)
		if n == 1 {
			suffix = "1 log · newest first"
		}
		if m.dayFilter.Focused() {
			suffix = "filtering"
		}
		return m.styles.fit(m.styles.header, m.width).Render(header + " · " + suffix)
	}

	if day, ok := m.selectedDay(); ok {
		primary, secondary := dayLabel(day, m.today)
		if t, err := time.Parse(dayFormat, day); err == nil {
			primary = t.Format("Mon Jan 02 2006")
		}
		header += " · " + primary
		// only show the relative hint (today/yesterday/N days ago); for
		// older days the secondary is just the raw date, which duplicates
		// what the primary already conveys
		if secondary != "" && secondary != day {
			header += " · " + secondary
		}
	}
	// a faint "N of M" position chip, so you keep a sense of where you are in
	// history now that the dot spine is gone. "1 of M" is the newest day, which
	// matches the "N days ago" phrasing beside it. gated on a min width so a
	// narrow terminal keeps the date/age and drops only the chip
	if _, ok := m.selectedDay(); ok && len(m.days) > 1 && m.width >= 50 {
		header += fmt.Sprintf(" · %d of %d", m.dayIdx+1, len(m.days))
	}
	return m.styles.fit(m.styles.header, m.width).Render(header)
}

// modalView renders every picker as one centered modal in place of the
// body: a title, then the list
func (m Model) modalView(title string, height int) string {
	rows := []string{m.styles.modalTitle.Render(title), m.picker.View()}
	box := m.styles.modal.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center, box)
}

// logCount is the number of days that actually have a log, ignoring the
// injected-empty today
func (m Model) logCount() int {
	n := 0
	for _, day := range m.days {
		if m.hasLog(day) {
			n++
		}
	}
	return n
}

func (m Model) footerView() string {
	footer := m.styles.fit(m.styles.footer, m.width)

	switch m.mode {
	case modeInput:
		return footer.UnsetFaint().Render(m.input.View())
	case modeProjects:
		return footer.Render("↑/↓ move • enter select • esc cancel")
	case modeTodos:
		return footer.Render("space toggle • enter done • esc cancel")
	case modeLedger:
		if m.dayFilter.Focused() {
			return footer.Render("filter by date or text • ↑/↓ move • enter open • esc clear")
		}
		return footer.Render("↑/↓ move • enter open • a append • c copy • n new day • / filter • p project • ? help")
	}

	if m.status != "" {
		if strings.HasPrefix(m.status, "error:") {
			return m.styles.fit(m.styles.errText, m.width).Render(m.status)
		}
		return footer.Render(m.status)
	}
	return footer.Render(m.help.View(m.keys))
}
