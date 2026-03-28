package tui

import "strings"

// handleFormKey processes key events while the multi-field form is active.
func (a *App) handleFormKey(key Key) bool {
	form := a.state.Form
	if form == nil {
		a.state.Mode = ModeNormal
		return false
	}

	field := form.ActiveField()

	switch {
	case key.Special == KeyEscape:
		a.state.Mode = ModeNormal
		a.state.Form = nil
		a.state.ClearCompletions()

	case key.Special == KeyTab:
		// If completions are showing, cycle through them
		if len(a.state.Completions) > 0 {
			a.state.CompletionIdx = (a.state.CompletionIdx + 1) % len(a.state.Completions)
			return false
		}
		a.state.ClearCompletions()
		form.NextField()

	case key.Special == KeyShiftTab:
		a.state.ClearCompletions()
		form.PrevField()

	case key.Special == KeyEnter:
		// If completion is active, accept it
		if a.state.CompletionIdx >= 0 && len(a.state.Completions) > 0 {
			a.acceptFormCompletion()
			return false
		}
		// Submit form
		a.submitForm()

	case key.Special == KeyBackspace:
		if field != nil {
			form.FieldDeleteChar()
			a.updateFormCompletions()
		}

	case key.Special == KeyDelete:
		if field != nil {
			form.FieldDeleteCharForward()
			a.updateFormCompletions()
		}

	case key.Special == KeyLeft:
		if field != nil && field.Buf.Cursor > 0 {
			field.Buf.Cursor--
		}

	case key.Special == KeyRight:
		if field != nil && field.Buf.Cursor < len(field.Buf.Data) {
			field.Buf.Cursor++
		}

	case key.Special == KeyWordLeft:
		form.FieldWordLeft()

	case key.Special == KeyWordRight:
		form.FieldWordRight()

	case key.Special == KeyDeleteWord:
		form.FieldDeleteWord()
		a.updateFormCompletions()

	case key.Special == KeyHome:
		if field != nil {
			field.Buf.Cursor = 0
		}

	case key.Special == KeyEnd:
		if field != nil {
			field.Buf.Cursor = len(field.Buf.Data)
		}

	case key.Special == KeyKillLine:
		if field != nil {
			field.Buf.KillLine()
			a.state.ClearCompletions()
		}

	case key.Special == KeyKillBack:
		if field != nil {
			field.Buf.KillBack()
			a.updateFormCompletions()
		}

	case key.Char != 0:
		if field != nil {
			// Accept completion on space for completable fields
			if a.state.CompletionIdx >= 0 && key.Char == ' ' && field.Type != "text" {
				a.acceptFormCompletion()
				return false
			}
			form.FieldInsertChar(key.Char)
			a.updateFormCompletions()
		}
	}

	return false
}

// updateFormCompletions refreshes the autocomplete list based on the
// active form field's current value.
func (a *App) updateFormCompletions() {
	form := a.state.Form
	if form == nil {
		return
	}
	field := form.ActiveField()
	if field == nil {
		a.state.ClearCompletions()
		return
	}

	prefix := field.Buf.String()
	lower := strings.ToLower(prefix)
	switch field.Type {
	case "status":
		var matches []string
		for _, s := range a.cfg.Symbols.StateNames() {
			if prefix == "" || strings.HasPrefix(strings.ToLower(s), lower) {
				matches = append(matches, s)
			}
		}
		a.state.Completions = matches
		a.state.CompletionType = "status"
	case "symbol":
		a.state.Completions = a.completer.CompleteSymbol(prefix)
		a.state.CompletionType = "symbol"
	case "project":
		a.state.Completions = a.completer.CompleteProject(prefix)
		a.state.CompletionType = "project"
	case "person":
		a.state.Completions = a.completer.CompletePerson(prefix)
		a.state.CompletionType = "person"
	default:
		a.state.ClearCompletions()
		return
	}

	if len(a.state.Completions) > 0 {
		a.state.CompletionIdx = 0
	} else {
		a.state.CompletionIdx = -1
	}
}

// acceptFormCompletion replaces the active field's value with the
// currently highlighted completion.
func (a *App) acceptFormCompletion() {
	form := a.state.Form
	if form == nil {
		return
	}
	field := form.ActiveField()
	if field == nil || a.state.CompletionIdx < 0 || a.state.CompletionIdx >= len(a.state.Completions) {
		return
	}
	completion := a.state.Completions[a.state.CompletionIdx]
	field.Buf.Set(completion)
	a.state.ClearCompletions()
}

// submitForm validates the form fields and either adds a new entry or
// edits an existing one via the EntryService, then returns to normal mode.
func (a *App) submitForm() {
	form := a.state.Form
	if form == nil {
		return
	}

	status := form.FieldValue("status")
	symName := form.FieldValue("symbol")
	project := form.FieldValue("project")
	person := form.FieldValue("person")
	desc := form.FieldValue("text")

	if desc == "" {
		a.state.StatusMsg = "description is required"
		return
	}

	if form.IsEdit && form.EditIdx >= 0 {
		err := a.svc.EditEntry(a.date, form.EditIdx, symName, status, project, person, desc)
		if err != nil {
			a.state.StatusMsg = err.Error()
			return
		}
		a.loadEntries()
	} else {
		_, err := a.svc.AddEntry(symName, status, project, person, desc)
		if err != nil {
			a.state.StatusMsg = err.Error()
			return
		}
		a.loadEntries()
		a.state.Cursor = len(a.entries) - 1
	}

	a.state.Mode = ModeNormal
	a.state.Form = nil
	a.state.ClearCompletions()
}
