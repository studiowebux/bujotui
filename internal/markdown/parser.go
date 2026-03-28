package markdown

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/studiowebux/bujotui/internal/model"
)

const (
	dateHeadingPrefix = "# "
	entryPrefix       = "- "
	dateLayout        = "2006-01-02"
	// dateTimeLayout uses a local-time format (no timezone offset).
	// All parsed and formatted times are interpreted in the system's local timezone.
	dateTimeLayout = "2006-01-02T15:04"
)

// ParseFile parses a monthly markdown file into DayLogs.
// It uses the provided SymbolSet to resolve symbol characters.
func ParseFile(r io.Reader, symbols *model.SymbolSet) ([]model.DayLog, error) {
	scanner := bufio.NewScanner(r)
	var days []model.DayLog
	var current *model.DayLog

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Date heading: # 2026-03-27
		if strings.HasPrefix(trimmed, dateHeadingPrefix) {
			dateStr := strings.TrimSpace(trimmed[len(dateHeadingPrefix):])
			// Parsed in local timezone; see dateTimeLayout comment.
			t, err := time.ParseInLocation(dateLayout, dateStr, time.Local)
			if err == nil {
				if current != nil {
					days = append(days, *current)
				}
				current = &model.DayLog{Date: t}
				continue
			}
			// Not a valid date heading — treat as raw line
		}

		// Daily note: > free text
		if current != nil && strings.HasPrefix(trimmed, "> ") {
			current.Note = trimmed[2:]
			current.Raw = append(current.Raw, model.RawLine{IsEntry: false, Text: line})
			continue
		}

		// Entry line: - symbol 2026-03-27T14:30 [project] @person description
		if current != nil && strings.HasPrefix(trimmed, entryPrefix) {
			entry, ok := ParseEntryLine(trimmed, symbols)
			if ok {
				idx := len(current.Entries)
				current.Entries = append(current.Entries, entry)
				current.Raw = append(current.Raw, model.RawLine{IsEntry: true, EntryIndex: idx})
				continue
			}
		}

		// Raw line (preserved for roundtrip)
		if current != nil {
			current.Raw = append(current.Raw, model.RawLine{IsEntry: false, Text: line})
		}
	}

	if current != nil {
		days = append(days, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse markdown: %w", err)
	}

	return days, nil
}

// ParseEntryLine parses a single entry line.
// Format: - {symbol} {YYYY-MM-DDThh:mm} [{project}] @{person} {description}
func ParseEntryLine(line string, symbols *model.SymbolSet) (model.Entry, bool) {
	rest := line
	if !strings.HasPrefix(rest, entryPrefix) {
		return model.Entry{}, false
	}
	rest = rest[len(entryPrefix):]

	// Extract symbol (first rune)
	symChar, size := utf8.DecodeRuneInString(rest)
	if symChar == utf8.RuneError || size == 0 {
		return model.Entry{}, false
	}
	sym, ok := symbols.LookupByChar(string(symChar))
	if !ok {
		return model.Entry{}, false
	}
	rest = strings.TrimLeft(rest[size:], " ")

	// Extract datetime
	var dt time.Time
	if len(rest) >= len(dateTimeLayout) {
		// Parsed in local timezone; see dateTimeLayout comment.
		parsed, err := time.ParseInLocation(dateTimeLayout, rest[:len(dateTimeLayout)], time.Local)
		if err == nil {
			dt = parsed
			rest = strings.TrimLeft(rest[len(dateTimeLayout):], " ")
		}
	}

	// Extract project: [project-name]
	var project string
	if strings.HasPrefix(rest, "[") {
		end := strings.IndexByte(rest, ']')
		if end < 0 {
			return model.Entry{}, false
		}
		project = rest[1:end]
		rest = strings.TrimLeft(rest[end+1:], " ")
	}

	// Extract person: @person
	var person string
	if strings.HasPrefix(rest, "@") {
		spaceIdx := strings.IndexByte(rest, ' ')
		if spaceIdx < 0 {
			person = rest[1:]
			rest = ""
		} else {
			person = rest[1:spaceIdx]
			rest = strings.TrimLeft(rest[spaceIdx+1:], " ")
		}
	}

	// Extract state: state:VALUE (optional)
	var state string
	if strings.HasPrefix(rest, "state:") {
		spaceIdx := strings.IndexByte(rest, ' ')
		if spaceIdx < 0 {
			state = rest[len("state:"):]
			rest = ""
		} else {
			state = rest[len("state:"):spaceIdx]
			rest = strings.TrimLeft(rest[spaceIdx+1:], " ")
		}
	}

	// Extract migration links: ->YYYY-MM-DD and <-YYYY-MM-DD (optional)
	var migratedTo, migratedFrom string
	if strings.HasPrefix(rest, "->") {
		tok := rest[2:]
		spaceIdx := strings.IndexByte(tok, ' ')
		if spaceIdx < 0 {
			migratedTo = tok
			rest = ""
		} else {
			migratedTo = tok[:spaceIdx]
			rest = strings.TrimLeft(tok[spaceIdx+1:], " ")
		}
	}
	if strings.HasPrefix(rest, "<-") {
		tok := rest[2:]
		spaceIdx := strings.IndexByte(tok, ' ')
		if spaceIdx < 0 {
			migratedFrom = tok
			rest = ""
		} else {
			migratedFrom = tok[:spaceIdx]
			rest = strings.TrimLeft(tok[spaceIdx+1:], " ")
		}
	}

	// Remaining is description
	desc := rest

	return model.Entry{
		Symbol:       sym,
		State:        state,
		Project:      project,
		Person:       person,
		Description:  desc,
		DateTime:     dt,
		MigratedTo:   migratedTo,
		MigratedFrom: migratedFrom,
	}, true
}
