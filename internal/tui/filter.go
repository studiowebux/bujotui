package tui

import "strings"

func (a *App) handleFilterKey(key Key) bool {
	switch {
	case key.Special == KeyEscape:
		a.state.Mode = ModeNormal
		a.state.FilterProject = ""
		a.state.FilterPerson = ""
		a.state.FilterSymbol = ""
		a.state.FilterText = ""
		a.state.ClearInput()
		a.applyFilter()

	case key.Special == KeyEnter:
		// If a completion is selected, accept it instead of submitting
		if a.state.CompletionIdx >= 0 && len(a.state.Completions) > 0 {
			a.state.AcceptCompletion()
			return false
		}
		input := a.state.InputString()
		a.parseFilter(input)
		a.state.Mode = ModeNormal
		a.state.ClearInput()
		a.applyFilter()

	case key.Special == KeyBackspace:
		a.state.Input.DeleteChar()
		a.state.ClearCompletions()

	case key.Special == KeyTab:
		if len(a.state.Completions) > 0 {
			a.state.CompletionIdx = (a.state.CompletionIdx + 1) % len(a.state.Completions)
		} else {
			UpdateCompletions(a.state, a.completer)
		}

	case key.Special == KeyLeft:
		if a.state.Input.Cursor > 0 {
			a.state.Input.Cursor--
		}

	case key.Special == KeyRight:
		if a.state.Input.Cursor < len(a.state.Input.Data) {
			a.state.Input.Cursor++
		}

	case key.Special == KeyWordLeft:
		a.state.Input.WordLeft()

	case key.Special == KeyWordRight:
		a.state.Input.WordRight()

	case key.Special == KeyDeleteWord:
		a.state.Input.DeleteWord()
		a.state.ClearCompletions()
		UpdateCompletions(a.state, a.completer)

	case key.Special == KeyDelete:
		a.state.Input.DeleteCharForward()
		a.state.ClearCompletions()
		UpdateCompletions(a.state, a.completer)

	case key.Special == KeyHome:
		a.state.Input.Cursor = 0

	case key.Special == KeyEnd:
		a.state.Input.Cursor = len(a.state.Input.Data)

	case key.Special == KeyKillLine:
		a.state.Input.KillLine()
		a.state.ClearCompletions()

	case key.Special == KeyKillBack:
		a.state.Input.KillBack()
		a.state.ClearCompletions()
		UpdateCompletions(a.state, a.completer)

	case key.Char != 0:
		if a.state.CompletionIdx >= 0 && key.Char == ' ' {
			a.state.AcceptCompletion()
		}
		a.state.Input.InsertChar(key.Char)
		a.state.ClearCompletions()
		UpdateCompletions(a.state, a.completer)
	}

	return false
}

func (a *App) parseFilter(input string) {
	a.state.FilterProject = ""
	a.state.FilterPerson = ""
	a.state.FilterSymbol = ""
	a.state.FilterText = ""

	parts := strings.Fields(input)
	var textParts []string

	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, "project:") || strings.HasPrefix(part, "p:"):
			val := part[strings.IndexByte(part, ':')+1:]
			a.state.FilterProject = val
		case strings.HasPrefix(part, "person:") || strings.HasPrefix(part, "@"):
			val := strings.TrimPrefix(part, "person:")
			val = strings.TrimPrefix(val, "@")
			a.state.FilterPerson = val
		case strings.HasPrefix(part, "symbol:") || strings.HasPrefix(part, "s:"):
			val := part[strings.IndexByte(part, ':')+1:]
			a.state.FilterSymbol = val
		default:
			textParts = append(textParts, part)
		}
	}

	if len(textParts) > 0 {
		a.state.FilterText = strings.Join(textParts, " ")
	}
}
