package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/studiowebux/bujotui/internal/term"
)

func renderFuture(w io.Writer, vs *ViewState) {
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
	header := " bujotui  Future Log"
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

	if len(vs.FutMonths) == 0 {
		fmt.Fprintf(w, " %sNo future months loaded.%s%s", term.FgBrightWhite, term.Reset, nl)
		for i := 3; i < height-2; i++ {
			fmt.Fprint(w, nl)
		}
		goto bottom
	}

	{
		// Month tabs
		var tabLine strings.Builder
		tabLine.WriteByte(' ')
		for i, m := range vs.FutMonths {
			// Short month label: "Jan", "Feb", etc.
			shortLabel := m.Label
			if len(shortLabel) > 3 {
				shortLabel = shortLabel[:3]
			}
			count := len(m.Entries)
			tab := fmt.Sprintf(" %s(%d) ", shortLabel, count)

			if i == vs.FutMonthIdx {
				tabLine.WriteString(fmt.Sprintf("%s%s%s", term.BgHighlight+term.Bold+term.FgCyan, tab, term.Reset))
			} else {
				tabLine.WriteString(fmt.Sprintf("%s%s%s", term.FgGray, tab, term.Reset))
			}
		}
		fmt.Fprint(w, tabLine.String()+nl)

		// Blank line
		fmt.Fprint(w, nl)

		// Selected month title
		selMonth := vs.FutMonths[vs.FutMonthIdx]
		fmt.Fprintf(w, " %s%s%s%s", term.Bold+term.FgCyan, selMonth.Label, term.Reset, nl)

		// Entries for selected month
		visible := height - 7 // header + sep + tabs + blank + title + sep + help
		if visible < 1 {
			visible = 5
		}

		if len(selMonth.Entries) == 0 {
			fmt.Fprintf(w, " %sNo entries. Press 'a' to add one.%s%s", term.FgBrightWhite, term.Reset, nl)
			for i := 1; i < visible; i++ {
				fmt.Fprint(w, nl)
			}
		} else {
			used := 0
			for i := 0; i < len(selMonth.Entries) && used < visible; i++ {
				e := selMonth.Entries[i]
				cursor := "  "
				if i == vs.FutItemIdx {
					cursor = "> "
				}

				symbolCol := fmt.Sprintf("%-10s", e.Symbol)
				desc := e.Desc
				maxDesc := width - displayWidth(cursor) - displayWidth(symbolCol) - 1
				if maxDesc > 0 && displayWidth(desc) > maxDesc {
					desc = truncateToWidth(desc, maxDesc-1) + "~"
				}

				padN := width - displayWidth(cursor) - displayWidth(symbolCol) - displayWidth(desc)
				if padN < 0 {
					padN = 0
				}

				if i == vs.FutItemIdx {
					fmt.Fprintf(w, "%s%s%s%s%s%s%s%s",
						term.BgHighlight+term.FgCyan+term.Bold, cursor,
						term.BgHighlight+term.FgBrightWhite, symbolCol,
						term.BgHighlight+term.FgBrightWhite, desc,
						strings.Repeat(" ", padN),
						term.Reset+nl)
				} else {
					fmt.Fprintf(w, "%s%s%s%s%s%s%s%s",
						term.FgBrightWhite, cursor,
						term.FgBrightWhite, symbolCol,
						term.FgBrightWhite, desc,
						strings.Repeat(" ", padN),
						term.Reset+nl)
				}
				used++
			}
			for i := used; i < visible; i++ {
				fmt.Fprint(w, nl)
			}
		}
	}

bottom:
	// Bottom separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	// Confirm / add / help bar
	if vs.FutConfirm && vs.FutMonthIdx < len(vs.FutMonths) {
		entries := vs.FutMonths[vs.FutMonthIdx].Entries
		if vs.FutItemIdx < len(entries) {
			renderConfirmPrompt(w, fmt.Sprintf("Delete '%s'?", entries[vs.FutItemIdx].Desc))
		}
		fmt.Fprint(w, "\x1b[0J")
		term.HideCursor(w)
		return
	}

	if vs.FutAdding {
		fmt.Fprintf(w, " %snew entry:%s %s",
			term.FgCyan+term.Bold, term.Reset, vs.FutEditBuf.String())
		fmt.Fprint(w, "\x1b[0J")
		col := len(" new entry: ") + vs.FutEditBuf.Cursor + 1
		row := height - 1
		term.MoveCursor(w, row, col)
		term.ShowCursor(w)
		return
	}

	if vs.StatusMsg != "" {
		fmt.Fprintf(w, " %s%s%s", term.FgRed+term.Bold, vs.StatusMsg, term.Reset)
	} else {
		fmt.Fprintf(w, " %sj/k%s move %s|%s %s[%s prev %s]%s next %s|%s %sa%s add %s|%s %sd%s del %s|%s %sEsc%s back %s|%s %s?%s help",
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset)
	}

	fmt.Fprint(w, "\x1b[0J")
	term.HideCursor(w)
}
