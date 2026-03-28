package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/term"
)

// nl clears to end of line then moves to the next line.
// This avoids ClearScreen which causes flicker.
const nl = "\x1b[0K\r\n"

// Render draws the full TUI screen.
func Render(w io.Writer, entries []model.Entry, date string, vs *ViewState) {
	term.HideCursor(w)

	width := vs.Width
	if width < 40 {
		width = 80
	}

	// Overlays: only redraw the overlay, not the full background
	if vs.Mode == ModeForm && vs.Form != nil {
		renderForm(w, vs)
		return
	}
	if vs.Mode == ModeHelp {
		renderHelpScreen(w, vs)
		return
	}

	term.MoveCursor(w, 1, 1)

	// Header
	header := fmt.Sprintf(" bujotui")
	dateLabel := fmt.Sprintf("  %s  ", date)
	nav := "[ prev  ] next "
	pad := width - len(header) - len(dateLabel) - len(nav)
	if pad < 1 {
		pad = 1
	}
	fmt.Fprintf(w, "%s%s%s%s%s%s%s%s%s%s",
		term.Bold+term.FgCyan, header,
		term.Reset+term.Bold+term.FgBrightWhite, dateLabel,
		term.Reset,
		strings.Repeat(" ", pad),
		term.FgGray, nav,
		term.Reset, nl)

	// Separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	// Active filter display
	if vs.FilterProject != "" || vs.FilterPerson != "" || vs.FilterSymbol != "" {
		var filterParts []string
		if vs.FilterProject != "" {
			filterParts = append(filterParts, "project:"+vs.FilterProject)
		}
		if vs.FilterPerson != "" {
			filterParts = append(filterParts, "person:"+vs.FilterPerson)
		}
		if vs.FilterSymbol != "" {
			filterParts = append(filterParts, "symbol:"+vs.FilterSymbol)
		}
		fmt.Fprintf(w, " %sfilter:%s %s%s%s", term.FgCyan, term.Reset, term.FgYellow, strings.Join(filterParts, " "), term.Reset+nl)
	}

	// Column headers
	timeHeader := ""
	if vs.ShowTime {
		timeHeader = fmt.Sprintf("%-*s", colTime, "TIME")
	}
	colHeader := fmt.Sprintf("  %-*s%-*s%s%-*s%-*s%s",
		colStatus, "STATUS",
		colSymbol, "SYMBOL",
		timeHeader,
		colProject, "PROJECT",
		colAssign, "ASSIGNEE",
		"DESCRIPTION")
	padH := width - displayWidth(colHeader)
	if padH < 0 {
		padH = 0
	}
	fmt.Fprintf(w, "%s%s%s%s%s", term.FgCyan, colHeader, strings.Repeat(" ", padH), term.Reset, nl)

	// Entries
	visible := vs.Height - 6 // header + separator + column header + status + separator + help
	if visible < 1 {
		visible = 10
	}

	// Adjust scroll offset to keep cursor visible
	if vs.Cursor < vs.ScrollOffset {
		vs.ScrollOffset = vs.Cursor
	}
	if vs.Cursor >= vs.ScrollOffset+visible {
		vs.ScrollOffset = vs.Cursor - visible + 1
	}

	for i := vs.ScrollOffset; i < len(entries) && i < vs.ScrollOffset+visible; i++ {
		renderEntryLine(w, entries[i], i == vs.Cursor, vs, width)
	}

	// Fill remaining space
	used := len(entries) - vs.ScrollOffset
	if used > visible {
		used = visible
	}
	if used < 0 {
		used = 0
	}
	if len(entries) == 0 {
		msg := "No entries for this day. Press 'a' to add one."
		if vs.FilterProject != "" || vs.FilterPerson != "" || vs.FilterSymbol != "" || vs.FilterText != "" {
			msg = "No matching entries. Press Esc then '/' to change filter."
		}
		fmt.Fprintf(w, " %s%s%s%s", term.FgBrightWhite, msg, term.Reset, nl)
		used = 1
	}
	for i := used; i < visible; i++ {
		fmt.Fprint(w, nl)
	}

	// Bottom separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	// Status message (error/info from last action)
	if vs.StatusMsg != "" {
		fmt.Fprintf(w, " %s%s%s%s", term.Bold+term.FgRed, vs.StatusMsg, term.Reset, nl)
	}

	// Mode-specific bottom area
	switch vs.Mode {
	case ModeFilter:
		renderInput(w, vs)
	case ModeConfirm:
		fmt.Fprintf(w, " %s%s%s %s(y/n)%s", term.Bold+term.FgRed, vs.ConfirmMsg, term.Reset, term.FgGray, term.Reset)
	case ModeHelp:
		renderHelpBar(w, width)
	case ModeForm:
		renderHelpBar(w, width)
	default:
		renderHelpBar(w, width)
	}

	// Clear any leftover lines from previous renders
	fmt.Fprint(w, "\x1b[0J")

}

func renderHelpBar(w io.Writer, width int) {
	// Compact help bar that fits within terminal width
	items := []struct{ key, label string }{
		{"j/k", "move"},
		{"x", "done"},
		{">", "migrate"},
		{"<", "sched"},
		{"c", "cancel"},
		{"r", "reset"},
		{"a", "add"},
		{"e", "edit"},
		{"d", "del"},
		{"t", "time"},
		{"/", "filter"},
		{"?", "help"},
		{"q", "quit"},
	}

	var b strings.Builder
	b.WriteByte(' ')
	col := 1
	for i, item := range items {
		segment := fmt.Sprintf("%s%s%s %s%s%s", term.FgCyan, item.key, term.Reset, term.FgGray, item.label, term.Reset)
		// visible length (without ANSI codes)
		visLen := len(item.key) + 1 + len(item.label)
		if i > 0 {
			visLen += 3 // for " | " separator
		}
		if col+visLen > width {
			break
		}
		if i > 0 {
			b.WriteString(fmt.Sprintf(" %s|%s ", term.FgGray, term.Reset))
			col += 3
		}
		b.WriteString(segment)
		col += visLen - 3 // already counted separator
	}
	fmt.Fprint(w, b.String())
}

// Column widths for entry display.
const (
	colStatus  = 12 // done, migrated, scheduled, cancelled, or empty
	colSymbol  = 10 // task, event, note, idea, urgent, waiting
	colTime    = 6  // HH:MM
	colProject = 14 // project name
	colAssign  = 12 // assignee
)

func renderEntryLine(w io.Writer, e model.Entry, selected bool, vs *ViewState, width int) {
	// Columns: cursor(2) | status(12) | symbol(10) | time(6) | project(14) | assignee(12) | description(rest)

	statusCol := fmt.Sprintf("%-*s", colStatus, e.State)
	symbolCol := fmt.Sprintf("%-*s", colSymbol, e.Symbol.Name)
	timeCol := ""
	if vs.ShowTime {
		timeCol = fmt.Sprintf("%-*s", colTime, e.DateTime.Format("15:04"))
	}
	projCol := fmt.Sprintf("%-*s", colProject, e.Project)
	assignCol := fmt.Sprintf("%-*s", colAssign, e.Person)
	cursor := "  "
	if selected {
		cursor = "> "
	}

	// Calculate space remaining for description and truncate if needed
	fixedWidth := displayWidth(cursor + statusCol + symbolCol + timeCol + projCol + assignCol)
	descMaxWidth := width - fixedWidth
	if descMaxWidth < 0 {
		descMaxWidth = 0
	}
	desc := e.Description
	if displayWidth(desc) > descMaxWidth {
		desc = truncateToWidth(desc, descMaxWidth-1) + "~"
	}

	padN := width - fixedWidth - displayWidth(desc)
	if padN < 0 {
		padN = 0
	}

	sc := stateColor(vs, e.State)

	if selected {
		bg := term.BgHighlight
		fmt.Fprintf(w, "%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s",
			bg+term.FgCyan+term.Bold, cursor,
			bg+sc, statusCol,
			bg+term.FgBrightWhite, symbolCol,
			bg+term.FgBrightWhite, timeCol,
			bg+term.FgBrightWhite, projCol,
			bg+term.FgBrightWhite, assignCol,
			bg+term.Bold+term.FgBrightWhite, desc,
			strings.Repeat(" ", padN),
			term.Reset+nl)
	} else {
		fmt.Fprintf(w, "%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s",
			term.FgBrightWhite, cursor,
			sc, statusCol,
			term.FgBrightWhite, symbolCol,
			term.FgBrightWhite, timeCol,
			term.FgBrightWhite, projCol,
			term.FgBrightWhite, assignCol,
			term.FgBrightWhite, desc,
			strings.Repeat(" ", padN),
			term.Reset+nl)
	}
}

// stateColor returns the ANSI color for a state, using config colors with hardcoded fallbacks.
func stateColor(vs *ViewState, state string) string {
	if state == "" {
		return term.FgBrightWhite
	}
	if vs.Cfg != nil {
		return vs.Cfg.LookupColor(state, term.FgBrightWhite)
	}
	return term.FgBrightWhite
}

const ghostFilter = "project:name @person symbol:name text"

// inputGhost returns ghost placeholder text for the current input mode.
func inputGhost(input string, mode Mode) string {
	if mode == ModeFilter && input == "" {
		return ghostFilter
	}
	return ""
}

func renderForm(w io.Writer, vs *ViewState) {
	form := vs.Form
	width := vs.Width
	height := vs.Height

	boxW := 56
	if boxW > width-4 {
		boxW = width - 4
	}

	bg := term.BgDarkGray + term.FgWhite
	hi := term.BgDarkGray + term.Bold + term.FgCyan
	dim := term.BgDarkGray + term.FgGray
	activeLbl := term.BgDarkGray + term.Bold + term.FgWhite
	activeField := "\x1b[48;5;238m" + term.FgWhite
	inactiveField := "\x1b[48;5;237m" + term.FgGray

	labelW := 13                // "Description:" padded to %-13s
	fieldW := boxW - labelW - 4 // 2 left pad + 2 right pad
	if fieldW < 10 {
		fieldW = 10
	}

	// Calculate box height
	boxH := 5 // title + blank + blank + footer + blank
	for range form.Fields {
		boxH++ // one row per field
	}
	// Completions
	if len(vs.Completions) > 0 {
		n := len(vs.Completions)
		if n > 6 {
			n = 7 // 6 + "more" line
		}
		boxH += n
	}

	if boxH > height-2 {
		boxH = height - 2
	}

	startCol := (width - boxW) / 2
	startRow := (height - boxH) / 2
	if startCol < 1 {
		startCol = 1
	}
	if startRow < 1 {
		startRow = 1
	}

	// Draw a full-width background line
	drawBg := func(row int) {
		term.MoveCursor(w, row, startCol)
		fmt.Fprintf(w, "%s%s%s", bg, strings.Repeat(" ", boxW), term.Reset)
	}

	row := startRow

	// Title
	drawBg(row)
	term.MoveCursor(w, row, startCol+2)
	title := "ADD ENTRY"
	if form.IsEdit {
		title = "EDIT ENTRY"
	}
	fmt.Fprintf(w, "%s%s%s", hi, title, term.Reset)
	row++

	// Blank
	drawBg(row)
	row++

	cursorRow := 0
	cursorCol := 0

	for i, field := range form.Fields {
		isActive := i == form.Active

		// Label
		label := fmt.Sprintf("%-13s", field.Label+":")
		lblStyle := dim
		if isActive {
			lblStyle = activeLbl
		}

		// Field value — scroll to keep cursor visible
		fullVal := field.Buf.String()
		cursor := field.Buf.Cursor
		scrollOffset := 0
		if cursor > fieldW-1 {
			scrollOffset = cursor - fieldW + 1
		}
		val := fullVal
		if scrollOffset > 0 && scrollOffset < len(val) {
			val = val[scrollOffset:]
		}
		if len(val) > fieldW {
			val = val[:fieldW]
		}
		valPad := fieldW - len(val)
		if valPad < 0 {
			valPad = 0
		}

		fldStyle := inactiveField
		if isActive {
			fldStyle = activeField
		}

		// Single write per row: position once, write label + field + padding
		fieldCol := startCol + 2 + labelW
		rightPad := boxW - 2 - 13 - fieldW
		if rightPad < 0 {
			rightPad = 0
		}
		term.MoveCursor(w, row, startCol)
		fmt.Fprintf(w, "%s  %s%s%s%s%s%s%s%s",
			bg, lblStyle, label, bg,
			fldStyle, val, strings.Repeat(" ", valPad),
			bg, strings.Repeat(" ", rightPad)+term.Reset)

		if isActive {
			cursorRow = row
			cursorCol = fieldCol + cursor - scrollOffset
		}

		row++

		// Completions below active field
		if isActive && len(vs.Completions) > 0 {
			maxShow := 6
			for ci, c := range vs.Completions {
				if ci >= maxShow {
					drawBg(row)
					term.MoveCursor(w, row, startCol+4)
					remaining := len(vs.Completions) - maxShow
					fmt.Fprintf(w, "%s...%d more%s", dim, remaining, term.Reset)
					row++
					break
				}
				drawBg(row)
				term.MoveCursor(w, row, startCol+2)
				if ci == vs.CompletionIdx {
					fmt.Fprintf(w, "%s> %s%s", activeLbl, c, term.Reset)
				} else {
					fmt.Fprintf(w, "%s  %s%s", dim, c, term.Reset)
				}
				row++
			}
		}
	}

	// Blank
	drawBg(row)
	row++

	// Footer
	drawBg(row)
	term.MoveCursor(w, row, startCol+2)
	fmt.Fprintf(w, "%sTab:next  S-Tab:prev  Enter:ok  Esc:cancel%s", dim, term.Reset)
	row++

	// Bottom blank
	drawBg(row)

	// Position cursor
	if cursorRow > 0 {
		term.MoveCursor(w, cursorRow, cursorCol)
		term.ShowCursor(w)
	} else {
		term.HideCursor(w)
	}
}

func renderHelpScreen(w io.Writer, vs *ViewState) {
	width := vs.Width
	height := vs.Height

	boxW := 44
	if boxW > width-4 {
		boxW = width - 4
	}

	// bg sets background for the whole line, bold resets only boldness not bg
	bg := term.BgDarkGray + term.FgWhite
	hi := term.BgDarkGray + term.Bold + term.FgWhite // highlighted key
	lo := term.BgDarkGray + term.FgWhite             // normal text after key
	section := term.BgDarkGray + term.Bold + term.FgCyan
	dim := term.BgDarkGray + term.FgGray

	type helpLine struct {
		text string // plain text content (no ANSI) for padding calc
		ansi string // full ANSI rendered content
	}

	mkSection := func(title string) helpLine {
		t := "  -- " + title + " --"
		return helpLine{t, fmt.Sprintf("  %s-- %s --%s", section, title, bg)}
	}
	mkKey := func(key, desc string) helpLine {
		t := fmt.Sprintf("  %-10s %s", key, desc)
		return helpLine{t, fmt.Sprintf("  %s%-10s%s %s", hi, key, lo, desc)}
	}
	mkEmpty := func() helpLine {
		return helpLine{"", ""}
	}

	lines := []helpLine{
		mkEmpty(),
		mkSection("NORMAL MODE"),
		mkEmpty(),
		mkKey("j/k", "Move up/down"),
		mkKey("G", "Go to last entry"),
		mkKey("g", "Go to first entry"),
		mkKey("a", "Add new entry"),
		mkKey("e", "Edit selected entry"),
		mkKey("d", "Delete (confirm y/n)"),
		mkKey("/", "Filter entries"),
		mkKey("[ ]", "Previous/next day"),
		mkKey("t", "Toggle time display"),
		mkKey("q", "Quit"),
		mkEmpty(),
		mkSection("STATE TRANSITIONS"),
		mkEmpty(),
		mkKey("x", "-> done"),
		mkKey(">", "-> migrated"),
		mkKey("<", "-> scheduled"),
		mkKey("c", "-> cancelled"),
		mkKey("r", "-> reset (clear state)"),
		mkEmpty(),
		mkSection("FORM (add/edit)"),
		mkEmpty(),
		mkKey("Tab", "Next field / cycle completions"),
		mkKey("S-Tab", "Previous field"),
		mkKey("Enter", "Accept completion or submit"),
		mkKey("Esc", "Cancel"),
		mkEmpty(),
		mkSection("TEXT EDITING"),
		mkEmpty(),
		mkKey("Opt+L/R", "Jump word"),
		mkKey("Opt+Bksp", "Delete word"),
		mkKey("Ctrl+A", "Go to start of field"),
		mkKey("Ctrl+E", "Go to end of field"),
		mkKey("Ctrl+K", "Delete to end of field"),
		mkKey("Ctrl+U", "Delete to start of field"),
		mkEmpty(),
		{" Press any key to close", fmt.Sprintf(" %s%s", dim, "Press any key to close")},
		mkEmpty(),
	}

	boxH := len(lines)
	if boxH > height-2 {
		boxH = height - 2
	}

	startCol := (width - boxW) / 2
	startRow := (height - boxH) / 2
	if startCol < 1 {
		startCol = 1
	}
	if startRow < 1 {
		startRow = 1
	}

	for i := 0; i < boxH && i < len(lines); i++ {
		term.MoveCursor(w, startRow+i, startCol)
		line := lines[i]
		pad := boxW - displayWidth(line.text)
		if pad < 0 {
			pad = 0
		}
		fmt.Fprintf(w, "%s%s%s%s", bg, line.ansi, strings.Repeat(" ", pad), term.Reset)
	}

	term.HideCursor(w)
}

func renderInput(w io.Writer, vs *ViewState) {
	prompt := vs.InputPrompt
	inputBefore := string(vs.Input.Data[:vs.Input.Cursor])
	inputAfter := string(vs.Input.Data[vs.Input.Cursor:])

	// Print prompt + input up to cursor, then save cursor position
	fmt.Fprintf(w, " %s%s%s%s", term.FgCyan+term.Bold, prompt, term.Reset, inputBefore)
	fmt.Fprint(w, "\x1b[s") // save cursor position
	fmt.Fprint(w, inputAfter)

	// Show ghost placeholder when no completions are active
	if len(vs.Completions) == 0 {
		ghost := inputGhost(vs.Input.String(), vs.Mode)
		if ghost != "" {
			fmt.Fprintf(w, "%s %s%s", term.FgGray, ghost, term.Reset)
		}
	}

	// Show completions below if available
	if len(vs.Completions) > 0 {
		fmt.Fprint(w, nl)
		maxVisible := 6
		// Determine scroll window to keep selected item visible
		scrollStart := 0
		if vs.CompletionIdx >= maxVisible {
			scrollStart = vs.CompletionIdx - maxVisible + 1
		}
		scrollEnd := scrollStart + maxVisible
		if scrollEnd > len(vs.Completions) {
			scrollEnd = len(vs.Completions)
		}
		if scrollStart > 0 {
			fmt.Fprintf(w, "   %s...%d above%s%s", term.FgGray, scrollStart, term.Reset, nl)
		}
		for i := scrollStart; i < scrollEnd; i++ {
			c := vs.Completions[i]
			if i == vs.CompletionIdx {
				fmt.Fprintf(w, "   %s> %s%s%s", term.FgCyan+term.Bold, c, term.Reset, nl)
			} else {
				fmt.Fprintf(w, "     %s%s%s%s", term.FgGray, c, term.Reset, nl)
			}
		}
		if scrollEnd < len(vs.Completions) {
			fmt.Fprintf(w, "   %s...%d more%s", term.FgGray, len(vs.Completions)-scrollEnd, term.Reset)
		}
	}

	// Restore cursor to saved position (right at Input.Cursor in the input line)
	fmt.Fprint(w, "\x1b[u")
	term.ShowCursor(w)
}

// displayWidth returns the visible width of a string (ASCII-only approximation).
// Multi-byte UTF-8 chars like emoji are counted as 2 columns.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if r < 128 {
			w++
		} else {
			w += 2
		}
	}
	return w
}

// truncateToWidth truncates a string to fit within maxWidth display columns.
func truncateToWidth(s string, maxWidth int) string {
	w := 0
	for i, r := range s {
		rw := 1
		if r >= 128 {
			rw = 2
		}
		if w+rw > maxWidth {
			return s[:i]
		}
		w += rw
	}
	return s
}
