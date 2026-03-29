package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/studiowebux/bujotui/internal/term"
)

func renderHabit(w io.Writer, vs *ViewState) {
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
	monthLabel := vs.HabMonth.Format("January 2006")
	header := fmt.Sprintf(" bujotui  Habits  %s", monthLabel)
	nav := "[ prev  ] next  Esc:back "
	pad := width - len(header) - len(nav)
	if pad < 1 {
		pad = 1
	}
	fmt.Fprintf(w, "%s%s%s%s%s%s%s",
		term.Bold+term.FgCyan, header,
		term.Reset, strings.Repeat(" ", pad),
		term.FgGray, nav,
		term.Reset+nl)

	// Separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	ht := vs.HabTracker
	if ht == nil || len(ht.Habits) == 0 {
		fmt.Fprintf(w, " %sNo habits. Press 'a' to add one.%s%s", term.FgBrightWhite, term.Reset, nl)
		for i := 3; i < height-2; i++ {
			fmt.Fprint(w, nl)
		}
		goto bottom
	}

	{
		// Calculate layout
		nameW := 18 // habit name column width
		streakW := 8 // streak column width
		gridW := width - nameW - streakW - 2 // remaining for day columns

		// Each day cell is 3 chars wide for readability
		cellW := 3
		maxDays := gridW / cellW
		if maxDays > ht.NumDays {
			maxDays = ht.NumDays
		}
		if maxDays < 1 {
			maxDays = 1
		}

		// Scroll days to keep cursor visible
		dayScroll := 0
		if vs.HabCol >= maxDays {
			dayScroll = vs.HabCol - maxDays + 1
		}

		// Day number header
		fmt.Fprintf(w, "%s%-*s%s", term.FgCyan, nameW, " HABIT", term.Reset)
		for d := dayScroll; d < dayScroll+maxDays && d < ht.NumDays; d++ {
			dayNum := d + 1
			if d == vs.HabCol {
				fmt.Fprintf(w, "%s%3d%s", term.Bold+term.FgCyan, dayNum, term.Reset)
			} else {
				fmt.Fprintf(w, "%s%3d%s", term.FgGray, dayNum, term.Reset)
			}
		}
		fmt.Fprintf(w, "  %s%-*s%s%s", term.FgCyan, streakW, "STREAK", term.Reset, nl)

		// Habit rows
		visible := height - 5 // header + separator + day header + separator + help
		if visible < 1 {
			visible = 5
		}

		// Scroll habits to keep cursor visible
		habScroll := 0
		if vs.HabRow >= visible {
			habScroll = vs.HabRow - visible + 1
		}

		used := 0
		for i := habScroll; i < len(ht.Habits) && used < visible; i++ {
			habit := ht.Habits[i]
			isSelected := i == vs.HabRow

			// Habit name (truncated)
			name := habit
			if len(name) > nameW-3 {
				name = name[:nameW-4] + "~"
			}

			if isSelected {
				fmt.Fprintf(w, "%s%-*s%s", term.BgHighlight+term.Bold+term.FgBrightWhite, nameW, " "+name, term.Reset)
			} else {
				fmt.Fprintf(w, "%s%-*s%s", term.FgBrightWhite, nameW, " "+name, term.Reset)
			}

			// Day cells
			for d := dayScroll; d < dayScroll+maxDays && d < ht.NumDays; d++ {
				day := d + 1
				done := ht.Done[habit][day]
				isCursor := isSelected && d == vs.HabCol

				marker := " · "
				markerColor := term.FgGray
				if done {
					marker = " ■ "
					markerColor = term.FgGreen
				}

				if isCursor {
					fmt.Fprintf(w, "%s%s%s", term.BgHighlight+markerColor+term.Bold, marker, term.Reset)
				} else if isSelected {
					fmt.Fprintf(w, "%s%s%s", term.BgHighlight+markerColor, marker, term.Reset)
				} else {
					fmt.Fprintf(w, "%s%s%s", markerColor, marker, term.Reset)
				}
			}

			// Streak
			streak := ht.Streaks[habit]
			streakStr := fmt.Sprintf(" %d", streak)
			streakColor := term.FgGray
			if streak > 0 {
				streakColor = term.FgYellow
			}
			if streak >= 7 {
				streakColor = term.FgGreen
			}

			if isSelected {
				padR := width - nameW - maxDays*cellW - len(streakStr)
				if padR < 0 {
					padR = 0
				}
				fmt.Fprintf(w, "%s%s%s%s%s",
					term.BgHighlight+streakColor, streakStr,
					strings.Repeat(" ", padR),
					term.Reset, nl)
			} else {
				padR := width - nameW - maxDays*cellW - len(streakStr)
				if padR < 0 {
					padR = 0
				}
				fmt.Fprintf(w, "%s%s%s%s%s",
					streakColor, streakStr,
					strings.Repeat(" ", padR),
					term.Reset, nl)
			}

			used++
		}

		for i := used; i < visible; i++ {
			fmt.Fprint(w, nl)
		}
	}

bottom:
	// Bottom separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	// Confirm / add / help bar
	if vs.HabConfirm && vs.HabTracker != nil && vs.HabRow < len(vs.HabTracker.Habits) {
		renderConfirmPrompt(w, fmt.Sprintf("Delete '%s'?", vs.HabTracker.Habits[vs.HabRow]))
		fmt.Fprint(w, "\x1b[0J")
		term.HideCursor(w)
		return
	}

	if vs.HabAdding || vs.HabEditing {
		label := "new habit:"
		buf := &vs.HabEditBuf
		if vs.HabEditing {
			label = "edit habit:"
			buf = &vs.HabEditBuf
		}
		fmt.Fprintf(w, " %s%s%s %s",
			term.FgCyan+term.Bold, label, term.Reset, buf.String())
		// Save cursor position right after the text we just wrote
		fmt.Fprint(w, "\x1b[0J")
		fmt.Fprintf(w, "\x1b[%d;%dH", height, len(" "+label+" ")+buf.Cursor+1)
		term.ShowCursor(w)
		return
	}

	if vs.StatusMsg != "" {
		fmt.Fprintf(w, " %s%s%s", term.FgRed+term.Bold, vs.StatusMsg, term.Reset)
	} else {
		fmt.Fprintf(w, " %sj/k%s move %s|%s %s[%s prev %s]%s next %s|%s %sx%s toggle %s|%s %sa%s add %s|%s %se%s edit %s|%s %sd%s del %s|%s %sEsc%s back",
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset)
	}

	fmt.Fprint(w, "\x1b[0J")
	term.HideCursor(w)
}
