package tui

import (
	"strings"
	"time"

	"github.com/studiowebux/bujotui/internal/complete"
	"github.com/studiowebux/bujotui/internal/model"
)

// ParseInput parses the input buffer into an Entry using the given defaults and symbols.
// Input format: [-s symbol] [-p project] [-a person] description
func ParseInput(input string, symbols *model.SymbolSet, defaultSymbol, defaultProject, defaultPerson string) (model.Entry, bool) {
	args := splitArgs(input)
	if len(args) == 0 {
		return model.Entry{}, false
	}

	symName := defaultSymbol
	project := ""
	person := defaultPerson
	var descParts []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-s":
			if i+1 < len(args) {
				symName = args[i+1]
				i++
			}
		case "-p":
			if i+1 < len(args) {
				project = args[i+1]
				i++
			}
		case "-a":
			if i+1 < len(args) {
				person = args[i+1]
				i++
			}
		default:
			// Handle @person inline
			if strings.HasPrefix(args[i], "@") {
				person = args[i][1:]
			} else if strings.HasPrefix(args[i], "[") && strings.HasSuffix(args[i], "]") {
				project = args[i][1 : len(args[i])-1]
			} else {
				descParts = append(descParts, args[i])
			}
		}
	}

	if len(descParts) == 0 {
		return model.Entry{}, false
	}

	sym, ok := symbols.LookupByName(symName)
	if !ok {
		return model.Entry{}, false
	}

	return model.Entry{
		Symbol:      sym,
		Project:     project,
		Person:      person,
		Description: strings.Join(descParts, " "),
		DateTime:    time.Now(),
	}, true
}

// UpdateCompletions determines what to complete based on cursor context.
func UpdateCompletions(vs *ViewState, comp *complete.Completer) {
	// Find current token
	tokenStart := vs.Input.Cursor
	for tokenStart > 0 && vs.Input.Data[tokenStart-1] != ' ' {
		tokenStart--
	}
	token := string(vs.Input.Data[tokenStart:vs.Input.Cursor])

	// Determine context
	prevToken := ""
	if tokenStart > 0 {
		// Find the token before current
		end := tokenStart - 1
		for end > 0 && vs.Input.Data[end-1] == ' ' {
			end--
		}
		start := end
		for start > 0 && vs.Input.Data[start-1] != ' ' {
			start--
		}
		prevToken = string(vs.Input.Data[start:end])
	}

	switch {
	case prevToken == "-s":
		vs.Completions = comp.CompleteSymbol(token)
		vs.CompletionType = "symbol"
	case prevToken == "-p":
		vs.Completions = comp.CompleteProject(token)
		vs.CompletionType = "project"
	case prevToken == "-a":
		vs.Completions = comp.CompletePerson(token)
		vs.CompletionType = "person"
	case strings.HasPrefix(token, "@"):
		vs.Completions = comp.CompletePerson(token[1:])
		vs.CompletionType = "person"
	case strings.HasPrefix(token, "["):
		prefix := strings.TrimPrefix(token, "[")
		prefix = strings.TrimSuffix(prefix, "]")
		vs.Completions = comp.CompleteProject(prefix)
		vs.CompletionType = "project"
	default:
		vs.ClearCompletions()
		return
	}

	if len(vs.Completions) > 0 {
		vs.CompletionIdx = 0
	} else {
		vs.CompletionIdx = -1
	}
}

func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false

	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
