package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/studiowebux/bujotui/internal/term"
)

func renderCollections(w io.Writer, vs *ViewState) {
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
	header := " bujotui  Collections"
	nav := "Esc:back "
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

	visible := height - 5 // header + separator + separator + help + adding
	if visible < 1 {
		visible = 10
	}

	// Adjust scroll
	if vs.ColCursor < vs.ColScroll {
		vs.ColScroll = vs.ColCursor
	}
	if vs.ColCursor >= vs.ColScroll+visible {
		vs.ColScroll = vs.ColCursor - visible + 1
	}

	if len(vs.ColNames) == 0 {
		fmt.Fprintf(w, " %sNo collections. Press 'a' to create one.%s%s", term.FgBrightWhite, term.Reset, nl)
		for i := 1; i < visible; i++ {
			fmt.Fprint(w, nl)
		}
	} else {
		used := 0
		for i := vs.ColScroll; i < len(vs.ColNames) && i < vs.ColScroll+visible; i++ {
			name := vs.ColNames[i]
			cursor := "  "
			if i == vs.ColCursor {
				cursor = "> "
			}

			line := cursor + name
			padN := width - displayWidth(line)
			if padN < 0 {
				padN = 0
			}

			if i == vs.ColCursor {
				fmt.Fprintf(w, "%s%s%s%s%s%s",
					term.BgHighlight+term.FgCyan+term.Bold, cursor,
					term.BgHighlight+term.FgBrightWhite, name,
					strings.Repeat(" ", padN),
					term.Reset+nl)
			} else {
				fmt.Fprintf(w, "%s%s%s%s%s%s",
					term.FgBrightWhite, cursor,
					term.FgBrightWhite, name,
					strings.Repeat(" ", padN),
					term.Reset+nl)
			}
			used++
		}
		for i := used; i < visible; i++ {
			fmt.Fprint(w, nl)
		}
	}

	// Bottom separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	// Confirm delete
	if vs.ColConfirm && vs.ColCursor < len(vs.ColNames) {
		renderConfirmPrompt(w, fmt.Sprintf("Delete '%s'?", vs.ColNames[vs.ColCursor]))
		fmt.Fprint(w, "\x1b[0J")
		term.HideCursor(w)
		return
	}

	// Status / add input
	if vs.ColAdding {
		fmt.Fprintf(w, " %snew collection:%s %s",
			term.FgCyan+term.Bold, term.Reset, vs.ColEditBuf.String())
		fmt.Fprint(w, "\x1b[0J")
		// Position cursor
		col := len(" new collection: ") + vs.ColEditBuf.Cursor + 1
		row := height - 1
		term.MoveCursor(w, row, col)
		term.ShowCursor(w)
		return
	}

	if vs.StatusMsg != "" {
		fmt.Fprintf(w, " %s%s%s", term.FgRed+term.Bold, vs.StatusMsg, term.Reset)
	} else {
		fmt.Fprintf(w, " %sj/k%s move %s|%s %sEnter%s open %s|%s %sa%s add %s|%s %sd%s delete %s|%s %sEsc%s back %s|%s %s?%s help",
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset)
	}

	fmt.Fprint(w, "\x1b[0J")
	term.HideCursor(w)
}

func renderCollectionView(w io.Writer, vs *ViewState) {
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
	header := fmt.Sprintf(" bujotui  %s", vs.ColName)
	count := fmt.Sprintf(" %d items ", len(vs.ColItems))
	nav := "Esc:back "
	pad := width - len(header) - len(count) - len(nav)
	if pad < 1 {
		pad = 1
	}
	fmt.Fprintf(w, "%s%s%s%s%s%s%s%s%s",
		term.Bold+term.FgCyan, header,
		term.Reset+term.FgGray, count,
		strings.Repeat(" ", pad),
		term.FgGray, nav,
		term.Reset, nl)

	// Separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	visible := height - 5
	if visible < 1 {
		visible = 10
	}

	// Adjust scroll
	if vs.ColItemCursor < vs.ColItemScroll {
		vs.ColItemScroll = vs.ColItemCursor
	}
	if vs.ColItemCursor >= vs.ColItemScroll+visible {
		vs.ColItemScroll = vs.ColItemCursor - visible + 1
	}

	if len(vs.ColItems) == 0 {
		fmt.Fprintf(w, " %sEmpty collection. Press 'a' to add an item.%s%s", term.FgBrightWhite, term.Reset, nl)
		for i := 1; i < visible; i++ {
			fmt.Fprint(w, nl)
		}
	} else {
		used := 0
		for i := vs.ColItemScroll; i < len(vs.ColItems) && i < vs.ColItemScroll+visible; i++ {
			item := vs.ColItems[i]
			cursor := "  "
			if i == vs.ColItemCursor {
				cursor = "> "
			}

			check := "[ ] "
			checkColor := term.FgBrightWhite
			if item.Done {
				check = "[x] "
				checkColor = term.FgGreen
			}

			text := item.Text
			maxText := width - 8 // cursor(2) + check(4) + margin
			if maxText > 0 && displayWidth(text) > maxText {
				text = truncateToWidth(text, maxText-1) + "~"
			}

			padN := width - displayWidth(cursor) - displayWidth(check) - displayWidth(text)
			if padN < 0 {
				padN = 0
			}

			if i == vs.ColItemCursor {
				fmt.Fprintf(w, "%s%s%s%s%s%s%s%s",
					term.BgHighlight+term.FgCyan+term.Bold, cursor,
					term.BgHighlight+checkColor, check,
					term.BgHighlight+term.FgBrightWhite, text,
					strings.Repeat(" ", padN),
					term.Reset+nl)
			} else {
				fmt.Fprintf(w, "%s%s%s%s%s%s%s%s",
					term.FgBrightWhite, cursor,
					checkColor, check,
					term.FgBrightWhite, text,
					strings.Repeat(" ", padN),
					term.Reset+nl)
			}
			used++
		}
		for i := used; i < visible; i++ {
			fmt.Fprint(w, nl)
		}
	}

	// Bottom separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	// Edit/add input or help bar
	if vs.ColEditing {
		label := "add item:"
		if vs.ColEditIdx >= 0 {
			label = "edit item:"
		}
		fmt.Fprintf(w, " %s%s%s %s",
			term.FgCyan+term.Bold, label, term.Reset, vs.ColEditBuf.String())
		fmt.Fprint(w, "\x1b[0J")
		col := len(" "+label+" ") + vs.ColEditBuf.Cursor + 1
		row := height - 1
		term.MoveCursor(w, row, col)
		term.ShowCursor(w)
		return
	}

	if vs.StatusMsg != "" {
		fmt.Fprintf(w, " %s%s%s", term.FgRed+term.Bold, vs.StatusMsg, term.Reset)
	} else {
		fmt.Fprintf(w, " %sj/k%s move %s|%s %sx%s toggle %s|%s %sa%s add %s|%s %se%s edit %s|%s %sd%s del %s|%s %sJ/K%s reorder %s|%s %sEsc%s back",
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset)
	}

	fmt.Fprint(w, "\x1b[0J")
	term.HideCursor(w)
}

func renderIndex(w io.Writer, vs *ViewState) {
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
	header := " bujotui  Index"
	count := fmt.Sprintf(" %d items ", len(vs.IdxFiltered))
	nav := "Esc:back "
	pad := width - len(header) - len(count) - len(nav)
	if pad < 1 {
		pad = 1
	}
	fmt.Fprintf(w, "%s%s%s%s%s%s%s%s%s",
		term.Bold+term.FgCyan, header,
		term.Reset+term.FgGray, count,
		strings.Repeat(" ", pad),
		term.FgGray, nav,
		term.Reset, nl)

	// Separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	// Filter display
	filterText := vs.IdxFilterBuf.String()
	if filterText != "" || vs.IdxFiltering {
		fmt.Fprintf(w, " %sfilter:%s %s%s%s%s",
			term.FgCyan, term.Reset,
			term.FgYellow, filterText,
			term.Reset, nl)
	}

	visible := height - 5
	if filterText != "" || vs.IdxFiltering {
		visible--
	}
	if visible < 1 {
		visible = 10
	}

	// Adjust scroll
	if vs.IdxCursor < vs.IdxScroll {
		vs.IdxScroll = vs.IdxCursor
	}
	if vs.IdxCursor >= vs.IdxScroll+visible {
		vs.IdxScroll = vs.IdxCursor - visible + 1
	}

	if len(vs.IdxFiltered) == 0 {
		msg := "No entries in index."
		if filterText != "" {
			msg = "No matching entries. Press '/' to change filter."
		}
		fmt.Fprintf(w, " %s%s%s%s", term.FgBrightWhite, msg, term.Reset, nl)
		for i := 1; i < visible; i++ {
			fmt.Fprint(w, nl)
		}
	} else {
		used := 0
		for i := vs.IdxScroll; i < len(vs.IdxFiltered) && i < vs.IdxScroll+visible; i++ {
			entry := vs.IdxEntries[vs.IdxFiltered[i]]
			cursor := "  "
			if i == vs.IdxCursor {
				cursor = "> "
			}

			kindLabel := fmt.Sprintf("%-12s", entry.Kind)
			name := entry.Name

			maxName := width - displayWidth(cursor) - displayWidth(kindLabel) - 1
			if maxName > 0 && displayWidth(name) > maxName {
				name = truncateToWidth(name, maxName-1) + "~"
			}

			padN := width - displayWidth(cursor) - displayWidth(kindLabel) - displayWidth(name)
			if padN < 0 {
				padN = 0
			}

			kindColor := term.FgGray
			if entry.Kind == "collection" {
				kindColor = term.FgCyan
			}

			if i == vs.IdxCursor {
				fmt.Fprintf(w, "%s%s%s%s%s%s%s%s",
					term.BgHighlight+term.FgCyan+term.Bold, cursor,
					term.BgHighlight+kindColor, kindLabel,
					term.BgHighlight+term.FgBrightWhite, name,
					strings.Repeat(" ", padN),
					term.Reset+nl)
			} else {
				fmt.Fprintf(w, "%s%s%s%s%s%s%s%s",
					term.FgBrightWhite, cursor,
					kindColor, kindLabel,
					term.FgBrightWhite, name,
					strings.Repeat(" ", padN),
					term.Reset+nl)
			}
			used++
		}
		for i := used; i < visible; i++ {
			fmt.Fprint(w, nl)
		}
	}

	// Bottom separator
	fmt.Fprintf(w, "%s%s%s%s", term.FgGray, strings.Repeat("─", width), term.Reset, nl)

	if vs.IdxFiltering {
		fmt.Fprintf(w, " %sfilter>%s %s",
			term.FgCyan+term.Bold, term.Reset, filterText)
		fmt.Fprint(w, "\x1b[0J")
		col := len(" filter> ") + vs.IdxFilterBuf.Cursor + 1
		row := height - 1
		term.MoveCursor(w, row, col)
		term.ShowCursor(w)
		return
	}

	if vs.StatusMsg != "" {
		fmt.Fprintf(w, " %s%s%s", term.FgRed+term.Bold, vs.StatusMsg, term.Reset)
	} else {
		fmt.Fprintf(w, " %sj/k%s move %s|%s %sEnter%s open %s|%s %s/%s filter %s|%s %sEsc%s back %s|%s %s?%s help",
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset, term.FgGray, term.Reset,
			term.FgCyan, term.Reset)
	}

	fmt.Fprint(w, "\x1b[0J")
	term.HideCursor(w)
}
