package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

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

	// Help screen replaces the full screen — no background needed.
	if vs.Mode == ModeHelp {
		renderHelpScreen(w, vs)
		return
	}

	// Form overlay: skip background redraw unless terminal was resized.
	if vs.Mode == ModeForm && vs.Form != nil {
		if vs.Width == vs.FormDrawnW && vs.Height == vs.FormDrawnH {
			renderForm(w, vs)
			return
		}
		// Size changed — fall through to redraw background, then overlay form.
	}
	if vs.Mode == ModeMigrate {
		renderMigrate(w, vs)
		return
	}
	if vs.Mode == ModeCalendar {
		renderCalendar(w, vs)
		return
	}
	if vs.Mode == ModeCollections {
		renderCollections(w, vs)
		return
	}
	if vs.Mode == ModeCollection {
		renderCollectionView(w, vs)
		return
	}
	if vs.Mode == ModeIndex {
		renderIndex(w, vs)
		return
	}
	if vs.Mode == ModeHabit {
		renderHabit(w, vs)
		return
	}
	if vs.Mode == ModeFuture {
		renderFuture(w, vs)
		return
	}

	term.MoveCursor(w, 1, 1)

	// Header
	header := " bujotui"
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
		term.FgWhite, nav,
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
	visible := vs.Height - 6 // header + separator + column header + separator + help + spare
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

	// Mode-specific bottom area.
	switch vs.Mode {
	case ModeFilter:
		renderInput(w, vs)
	case ModeConfirm:
		renderConfirmPrompt(w, vs.ConfirmMsg)
	default:
		if vs.StatusMsg != "" {
			// Error on the left, then help bar fills the rest of the same line.
			msg := truncateToWidth(vs.StatusMsg, width/2)
			fmt.Fprintf(w, " %s%s%s  ", term.Bold+term.FgRed, msg, term.Reset)
			renderHelpBar(w, width)
		} else {
			renderHelpBar(w, width)
		}
	}

	// Clear any leftover lines from previous renders
	fmt.Fprint(w, "\x1b[0J")

	// Form overlay — drawn on top of the background so resize redraws correctly.
	if vs.Mode == ModeForm && vs.Form != nil {
		renderForm(w, vs)
	}
}

func renderHelpBar(w io.Writer, width int) {
	// Compact help bar that fits within terminal width
	items := []struct{ key, label string }{
		{"j/k", "move"},
		{"Ret", "jump"},
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
		{"m", "calendar"},
		{"f", "future"},
		{"h", "habits"},
		{"p", "collections"},
		{"I", "index"},
		{"?", "help"},
		{"q", "quit"},
	}

	var b strings.Builder
	b.WriteByte(' ')
	col := 1
	for i, item := range items {
		segment := fmt.Sprintf("%s%s%s %s%s%s", term.FgCyan, item.key, term.Reset, term.FgWhite, item.label, term.Reset)
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
	colProject = 20 // project name
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

	// Migration link suffix
	migLink := ""
	if e.MigratedTo != "" {
		migLink = " -> " + e.MigratedTo
	} else if e.MigratedFrom != "" {
		migLink = " <- " + e.MigratedFrom
	}

	// Calculate space remaining for description and truncate if needed
	fixedWidth := displayWidth(cursor + statusCol + symbolCol + timeCol + projCol + assignCol)
	descMaxWidth := width - fixedWidth - displayWidth(migLink)
	if descMaxWidth < 0 {
		descMaxWidth = 0
	}
	desc := e.Description
	if displayWidth(desc) > descMaxWidth {
		desc = truncateToWidth(desc, descMaxWidth-1) + "~"
	}
	padN := width - fixedWidth - displayWidth(desc) - displayWidth(migLink)
	if padN < 0 {
		padN = 0
	}

	sc := stateColor(vs, e.State)

	if selected {
		bg := term.BgHighlight
		fmt.Fprintf(w, "%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s",
			bg+term.FgCyan+term.Bold, cursor,
			bg+sc, statusCol,
			bg+term.FgBrightWhite, symbolCol,
			bg+term.FgBrightWhite, timeCol,
			bg+term.FgBrightWhite, projCol,
			bg+term.FgBrightWhite, assignCol,
			bg+term.Bold+term.FgBrightWhite, desc,
			bg+term.FgCyan, migLink,
			strings.Repeat(" ", padN),
			term.Reset+nl)
	} else {
		fmt.Fprintf(w, "%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s",
			term.FgBrightWhite, cursor,
			sc, statusCol,
			term.FgBrightWhite, symbolCol,
			term.FgBrightWhite, timeCol,
			term.FgBrightWhite, projCol,
			term.FgBrightWhite, assignCol,
			term.FgBrightWhite, desc,
			term.FgCyan, migLink,
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

func renderMigrate(w io.Writer, vs *ViewState) {
	width := vs.Width
	height := vs.Height

	boxW := 44
	if boxW > width-4 {
		boxW = width - 4
	}

	bg := term.BgDarkGray + term.FgWhite
	hi := term.BgDarkGray + term.Bold + term.FgCyan
	dim := term.BgDarkGray + term.FgWhite
	activeLbl := term.BgDarkGray + term.Bold + term.FgWhite
	activeField := "\x1b[48;5;238m" + term.FgWhite

	boxH := 7 // title + blank + label+field + blank + footer + blank + status
	startCol := (width - boxW) / 2
	startRow := (height - boxH) / 2
	if startCol < 1 {
		startCol = 1
	}
	if startRow < 1 {
		startRow = 1
	}

	drawBg := func(row int) {
		term.MoveCursor(w, row, startCol)
		fmt.Fprintf(w, "%s%s%s", bg, strings.Repeat(" ", boxW), term.Reset)
	}

	row := startRow

	// Title
	drawBg(row)
	term.MoveCursor(w, row, startCol+2)
	fmt.Fprintf(w, "%sMIGRATE TO DATE%s", hi, term.Reset)
	row++

	// Blank
	drawBg(row)
	row++

	// Date field
	labelW := 12
	fieldW := boxW - labelW - 4
	if fieldW < 10 {
		fieldW = 10
	}

	val := vs.MigrateDate.String()
	cursor := vs.MigrateDate.Cursor
	scrollOffset := 0
	if cursor > fieldW-1 {
		scrollOffset = cursor - fieldW + 1
	}
	dispVal := val
	if scrollOffset > 0 && scrollOffset < len(dispVal) {
		dispVal = dispVal[scrollOffset:]
	}
	if len(dispVal) > fieldW {
		dispVal = dispVal[:fieldW]
	}
	valPad := fieldW - len(dispVal)
	if valPad < 0 {
		valPad = 0
	}

	fieldCol := startCol + 2 + labelW
	rightPad := boxW - 2 - labelW - fieldW
	if rightPad < 0 {
		rightPad = 0
	}
	term.MoveCursor(w, row, startCol)
	label := fmt.Sprintf("%-12s", "Date:")
	fmt.Fprintf(w, "%s  %s%s%s%s%s%s%s%s",
		bg, activeLbl, label, bg,
		activeField, dispVal, strings.Repeat(" ", valPad),
		bg, strings.Repeat(" ", rightPad)+term.Reset)
	row++

	// Blank
	drawBg(row)
	row++

	// Footer
	drawBg(row)
	term.MoveCursor(w, row, startCol+2)
	fmt.Fprintf(w, "%sEnter:migrate  Esc:cancel%s", dim, term.Reset)
	row++

	// Bottom blank
	drawBg(row)

	// Cursor
	term.MoveCursor(w, startRow+2, fieldCol+cursor-scrollOffset)
	term.ShowCursor(w)
}

func renderCalendar(w io.Writer, vs *ViewState) {
	width := vs.Width
	height := vs.Height
	if width < 40 {
		width = 80
	}
	if height < 10 {
		height = 24
	}

	term.MoveCursor(w, 1, 1)

	// Header
	monthLabel := vs.CalMonth.Format("January 2006")
	header := fmt.Sprintf(" bujotui  %s", monthLabel)
	nav := "[ prev  ] next  Esc:back  Enter:open "
	pad := width - len(header) - len(nav)
	if pad < 1 {
		pad = 1
	}
	fmt.Fprintf(w, "%s%s%s%s%s%s%s",
		term.Bold+term.FgCyan, header,
		term.Reset, strings.Repeat(" ", pad),
		term.FgWhite, nav,
		term.Reset+nl)

	// Separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	// Layout: left panel (day list) | right panel (day detail)
	leftW := 30
	rightW := width - leftW - 3 // 3 for " | " separator
	if rightW < 20 {
		rightW = 20
		leftW = width - rightW - 3
	}

	numDays := daysInMonth(vs.CalMonth)
	visible := height - 4 // header + separator + bottom separator + help bar

	// Scroll the day list to keep cursor visible
	scrollOffset := 0
	if vs.CalCursor >= visible {
		scrollOffset = vs.CalCursor - visible + 1
	}

	// Selected day entries for right panel
	selectedDay := vs.CalCursor + 1
	selectedEntries := vs.CalEntries[selectedDay]

	cursorRow := 0
	cursorCol := 0

	for row := 0; row < visible; row++ {
		dayIdx := scrollOffset + row
		term.MoveCursor(w, row+3, 1) // row+3 because header(1) + separator(1) + 1-based

		// Left panel: day row with note
		if dayIdx < numDays {
			day := dayIdx + 1
			date := time.Date(vs.CalMonth.Year(), vs.CalMonth.Month(), day, 0, 0, 0, 0, vs.CalMonth.Location())
			dayName := date.Format("Mon")[:2]
			entries := vs.CalEntries[day]
			count := len(entries)
			note := vs.CalNotes[day]

			// Show note text (or entry count if no note)
			noteText := note
			if noteText == "" && count > 0 {
				noteText = fmt.Sprintf("(%d entries)", count)
			}

			// If editing this day's note, show the edit buffer
			isSelected := dayIdx == vs.CalCursor
			if isSelected && vs.CalEditing {
				noteText = vs.CalNoteBuf.String()
			}

			maxNote := leftW - 8 // " DD Da "
			if maxNote < 0 {
				maxNote = 0
			}
			if len(noteText) > maxNote {
				noteText = noteText[:maxNote-1] + "~"
			}

			prefix := fmt.Sprintf(" %02d %s ", day, dayName)
			padL := leftW - len(prefix) - len(noteText)
			if padL < 0 {
				padL = 0
			}

			if isSelected && vs.CalEditing {
				// Editing: highlight bg with cursor
				fmt.Fprintf(w, "%s%s%s%s%s%s%s",
					term.BgHighlight+term.FgCyan, prefix,
					term.BgHighlight+term.FgBrightWhite, noteText,
					strings.Repeat(" ", padL),
					term.Reset, "")
				cursorRow = row + 3
				cursorCol = len(prefix) + vs.CalNoteBuf.Cursor + 1
				if cursorCol > leftW {
					cursorCol = leftW
				}
			} else if isSelected {
				fmt.Fprintf(w, "%s%s%s%s%s",
					term.BgHighlight+term.Bold+term.FgBrightWhite, prefix,
					noteText, strings.Repeat(" ", padL),
					term.Reset)
			} else if note != "" || count > 0 {
				fmt.Fprintf(w, "%s%s%s%s%s%s%s",
					term.FgCyan, prefix[:7],
					term.FgBrightWhite, noteText,
					strings.Repeat(" ", padL),
					term.Reset, "")
			} else {
				fmt.Fprintf(w, "%s%s%s%s",
					term.FgGray, prefix,
					strings.Repeat(" ", leftW-len(prefix)),
					term.Reset)
			}
		} else {
			fmt.Fprint(w, strings.Repeat(" ", leftW))
		}

		// Separator
		fmt.Fprintf(w, " %s│%s ", term.FgGray, term.Reset)

		// Right panel: entries for selected day
		if row == 0 {
			// Day header
			selDate := time.Date(vs.CalMonth.Year(), vs.CalMonth.Month(), selectedDay, 0, 0, 0, 0, vs.CalMonth.Location())
			dayHeader := selDate.Format("Monday, January 2")
			if len(dayHeader) > rightW {
				dayHeader = dayHeader[:rightW]
			}
			padR := rightW - len(dayHeader)
			if padR < 0 {
				padR = 0
			}
			fmt.Fprintf(w, "%s%s%s%s", term.Bold+term.FgCyan, dayHeader, strings.Repeat(" ", padR), term.Reset)
		} else if row == 1 {
			// Blank separator
			fmt.Fprint(w, strings.Repeat(" ", rightW))
		} else {
			entryIdx := row - 2
			if entryIdx < len(selectedEntries) {
				e := selectedEntries[entryIdx]

				// Status + Symbol + Description, truncated to fit
				status := ""
				if e.State != "" {
					status = e.State + " "
				}
				prefix := fmt.Sprintf("%-12s%-10s", status, e.Symbol.Name)
				desc := e.Description
				maxDesc := rightW - len(prefix)
				if maxDesc < 0 {
					maxDesc = 0
				}
				if len(desc) > maxDesc {
					desc = desc[:maxDesc-1] + "~"
				}
				line := prefix + desc
				padR := rightW - len(line)
				if padR < 0 {
					padR = 0
				}

				sc := stateColor(vs, e.State)
				fmt.Fprintf(w, "%s%-12s%s%-10s%s%s%s%s",
					sc, status,
					term.FgBrightWhite, e.Symbol.Name,
					term.FgBrightWhite, desc,
					strings.Repeat(" ", padR),
					term.Reset)
			} else {
				fmt.Fprint(w, strings.Repeat(" ", rightW))
			}
		}

		fmt.Fprint(w, nl)
	}

	// Bottom separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	// Help bar
	if vs.CalEditing {
		fmt.Fprintf(w, " %sEditing note:%s Enter:save  Esc:cancel",
			term.FgCyan, term.Reset)
	} else {
		fmt.Fprintf(w, " %sj/k%s move %s|%s %si/n%s edit note %s|%s %sEnter%s open %s|%s %s[%s prev %s]%s next %s|%s %sEsc%s back %s|%s %s?%s help",
			term.FgCyan, term.Reset, term.FgWhite, term.Reset,
			term.FgCyan, term.Reset, term.FgWhite, term.Reset,
			term.FgCyan, term.Reset, term.FgWhite, term.Reset,
			term.FgCyan, term.Reset, term.FgCyan, term.Reset, term.FgWhite, term.Reset,
			term.FgCyan, term.Reset, term.FgWhite, term.Reset,
			term.FgCyan, term.Reset)
	}

	// Clear to end
	fmt.Fprint(w, "\x1b[0J")

	// Position cursor when editing
	if vs.CalEditing && cursorRow > 0 {
		term.MoveCursor(w, cursorRow, cursorCol)
		term.ShowCursor(w)
	} else {
		term.HideCursor(w)
	}
}

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
	vs.FormDrawnW = width
	vs.FormDrawnH = height

	boxW := 56
	if boxW > width-4 {
		boxW = width - 4
	}

	bg := term.BgDarkGray + term.FgWhite
	hi := term.BgDarkGray + term.Bold + term.FgCyan
	dim := term.BgDarkGray + term.FgWhite
	activeLbl := term.BgDarkGray + term.Bold + term.FgWhite
	activeField := "\x1b[48;5;238m" + term.FgWhite
	inactiveField := "\x1b[48;5;237m" + term.FgBrightWhite

	labelW := 13                // "Description:" padded to %-13s
	fieldW := boxW - labelW - 4 // 2 left pad + 2 right pad
	if fieldW < 10 {
		fieldW = 10
	}

	// Calculate box height — reserve space for fields + select options + completions
	maxCompletions := 10 // 8 visible + scroll indicators
	boxH := 5 + len(form.Fields) + maxCompletions
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
		isSelect := field.Type == "status" || field.Type == "symbol"

		// Label
		label := fmt.Sprintf("%-13s", field.Label+":")
		lblStyle := dim
		if isActive {
			lblStyle = activeLbl
		}

		fieldCol := startCol + 2 + labelW
		rightPad := boxW - 2 - 13 - fieldW
		if rightPad < 0 {
			rightPad = 0
		}

		if isSelect {
			// Select field: show the current value or the highlighted option
			val := field.Buf.String()
			if isActive && vs.CompletionIdx >= 0 && vs.CompletionIdx < len(vs.Completions) {
				val = vs.Completions[vs.CompletionIdx]
			}
			if val == "" {
				val = "(none)"
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

			term.MoveCursor(w, row, startCol)
			fmt.Fprintf(w, "%s  %s%s%s%s%s%s%s%s",
				bg, lblStyle, label, bg,
				fldStyle, val, strings.Repeat(" ", valPad),
				bg, strings.Repeat(" ", rightPad)+term.Reset)
			row++

			// Show options list for active select field
			if isActive && len(vs.Completions) > 0 {
				maxShow := 8
				scrollStart := 0
				if vs.CompletionIdx >= maxShow {
					scrollStart = vs.CompletionIdx - maxShow + 1
				}
				scrollEnd := scrollStart + maxShow
				if scrollEnd > len(vs.Completions) {
					scrollEnd = len(vs.Completions)
				}

				if scrollStart > 0 {
					drawBg(row)
					term.MoveCursor(w, row, startCol+4)
					fmt.Fprintf(w, "%s...%d above%s", dim, scrollStart, term.Reset)
					row++
				}
				for ci := scrollStart; ci < scrollEnd; ci++ {
					c := vs.Completions[ci]
					drawBg(row)
					term.MoveCursor(w, row, startCol+2)
					if ci == vs.CompletionIdx {
						fmt.Fprintf(w, "%s > %-*s%s", hi, boxW-6, c, term.Reset)
					} else {
						fmt.Fprintf(w, "%s   %-*s%s", dim, boxW-6, c, term.Reset)
					}
					row++
				}
				if scrollEnd < len(vs.Completions) {
					drawBg(row)
					term.MoveCursor(w, row, startCol+4)
					fmt.Fprintf(w, "%s...%d more%s", dim, len(vs.Completions)-scrollEnd, term.Reset)
					row++
				}
			}
		} else {
			// Text field: editable with cursor
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

			// Completions below active text field
			if isActive && len(vs.Completions) > 0 {
				maxShow := 6
				scrollStart := 0
				if vs.CompletionIdx >= maxShow {
					scrollStart = vs.CompletionIdx - maxShow + 1
				}
				scrollEnd := scrollStart + maxShow
				if scrollEnd > len(vs.Completions) {
					scrollEnd = len(vs.Completions)
				}

				if scrollStart > 0 {
					drawBg(row)
					term.MoveCursor(w, row, startCol+4)
					fmt.Fprintf(w, "%s...%d above%s", dim, scrollStart, term.Reset)
					row++
				}
				for ci := scrollStart; ci < scrollEnd; ci++ {
					c := truncateToWidth(vs.Completions[ci], boxW-6)
					drawBg(row)
					term.MoveCursor(w, row, startCol+2)
					if ci == vs.CompletionIdx {
						fmt.Fprintf(w, "%s> %-*s%s", activeLbl, boxW-6, c, term.Reset)
					} else {
						fmt.Fprintf(w, "%s  %-*s%s", dim, boxW-6, c, term.Reset)
					}
					row++
				}
				if scrollEnd < len(vs.Completions) {
					drawBg(row)
					term.MoveCursor(w, row, startCol+4)
					fmt.Fprintf(w, "%s...%d more%s", dim, len(vs.Completions)-scrollEnd, term.Reset)
					row++
				}
			}
		}
	}

	// Blank
	drawBg(row)
	row++

	// Footer
	drawBg(row)
	term.MoveCursor(w, row, startCol+2)
	helpText := truncateToWidth(" arrows:pick  Tab:next  S-Tab:prev  Enter:ok  Esc:cancel", boxW-4)
	fmt.Fprintf(w, "%s%-*s%s", dim, boxW-4, helpText, term.Reset)
	row++

	// Fill remaining rows to clear stale content
	endRow := startRow + boxH
	for row <= endRow {
		drawBg(row)
		row++
	}

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
	dim := term.BgDarkGray + term.FgWhite

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
		mkKey("Enter", "Jump to migration link"),
		mkKey("a", "Add new entry"),
		mkKey("e", "Edit selected entry"),
		mkKey("d", "Delete (confirm y/n)"),
		mkKey("/", "Filter entries"),
		mkKey("[ ]", "Previous/next day"),
		mkKey("t", "Toggle time display"),
		mkKey("m", "Calendar view"),
		mkKey("f", "Future log"),
		mkKey("h", "Habit tracker"),
		mkKey("p", "Collections"),
		mkKey("I", "Index"),
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
			fmt.Fprintf(w, "   %s...%d above%s%s", term.FgWhite, scrollStart, term.Reset, nl)
		}
		for i := scrollStart; i < scrollEnd; i++ {
			c := vs.Completions[i]
			if i == vs.CompletionIdx {
				fmt.Fprintf(w, "   %s> %s%s%s", term.FgCyan+term.Bold, c, term.Reset, nl)
			} else {
				fmt.Fprintf(w, "     %s%s%s%s", term.FgWhite, c, term.Reset, nl)
			}
		}
		if scrollEnd < len(vs.Completions) {
			fmt.Fprintf(w, "   %s...%d more%s", term.FgWhite, len(vs.Completions)-scrollEnd, term.Reset)
		}
	}

	// Restore cursor to saved position (right at Input.Cursor in the input line)
	fmt.Fprint(w, "\x1b[u")
	term.ShowCursor(w)
}

// renderConfirmPrompt draws a "(y/n)" confirmation prompt with the given message.
func renderConfirmPrompt(w io.Writer, msg string) {
	fmt.Fprintf(w, " %s%s%s %s(y/n)%s", term.Bold+term.FgRed, msg, term.Reset, term.FgWhite, term.Reset)
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
