package markdown

import (
	"fmt"
	"strings"

	"github.com/studiowebux/bujotui/internal/model"
)

// FormatFile renders DayLogs back to markdown bytes.
// It preserves raw lines for roundtrip fidelity.
func FormatFile(days []model.DayLog) []byte {
	var b strings.Builder

	for i, day := range days {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "# %s\n", day.Date.Format(dateLayout))

		if len(day.Raw) > 0 {
			// Roundtrip mode: use Raw lines to preserve original structure.
			// Strip trailing blank lines — they would accumulate on every
			// write because the separator '\n' between days already provides one.
			raw := day.Raw
			for len(raw) > 0 {
				last := raw[len(raw)-1]
				if !last.IsEntry && strings.TrimSpace(last.Text) == "" {
					raw = raw[:len(raw)-1]
				} else {
					break
				}
			}
			for _, rl := range raw {
				if rl.IsEntry {
					if rl.EntryIndex < len(day.Entries) {
						b.WriteString(FormatEntry(day.Entries[rl.EntryIndex]))
						b.WriteByte('\n')
					}
				} else {
					b.WriteString(rl.Text)
					b.WriteByte('\n')
				}
			}
		} else {
			// Fresh entries: no raw lines to preserve
			b.WriteByte('\n')
			if day.Note != "" {
				fmt.Fprintf(&b, "> %s\n", day.Note)
			}
			for _, e := range day.Entries {
				b.WriteString(FormatEntry(e))
				b.WriteByte('\n')
			}
		}
	}

	return []byte(b.String())
}

// FormatEntry renders a single entry to its markdown line format.
// Format: - {symbol} {YYYY-MM-DDThh:mm} [{project}] @{person} {description}
// Project and person are omitted when empty.
func FormatEntry(e model.Entry) string {
	var b strings.Builder
	b.WriteString("- ")
	b.WriteString(e.Symbol.Char)
	b.WriteByte(' ')
	b.WriteString(e.DateTime.Format(dateTimeLayout))
	if e.Project != "" {
		fmt.Fprintf(&b, " [%s]", e.Project)
	}
	if e.Person != "" {
		fmt.Fprintf(&b, " @%s", e.Person)
	}
	if e.State != "" {
		fmt.Fprintf(&b, " state:%s", e.State)
	}
	if e.MigratedTo != "" {
		fmt.Fprintf(&b, " ->%s", e.MigratedTo)
	}
	if e.MigratedFrom != "" {
		fmt.Fprintf(&b, " <-%s", e.MigratedFrom)
	}
	if e.ID != "" {
		fmt.Fprintf(&b, " id:%s ts:%d", e.ID, e.UpdatedAt)
	}
	b.WriteByte(' ')
	b.WriteString(e.Description)
	return b.String()
}
