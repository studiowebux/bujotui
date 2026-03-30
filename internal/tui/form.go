package tui

import "strings"

// isSelectField returns true if the field type uses a select-list UX.
func isSelectField(fieldType string) bool {
	return fieldType == "status" || fieldType == "symbol"
}

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
		a.acceptAndNextField(form)

	case key.Special == KeyShiftTab:
		a.acceptAndPrevField(form)

	case key.Special == KeyEnter:
		if len(a.state.Completions) > 0 && a.state.CompletionIdx >= 0 {
			// Accept selection and move to next field
			a.acceptFormCompletion()
			form.NextField()
			a.showFieldOptions()
		} else if field != nil && isSelectField(field.Type) {
			// On select field with no completions, just move on
			form.NextField()
			a.showFieldOptions()
		} else {
			// Submit form (only from text fields)
			a.submitForm()
		}

	case key.Special == KeyDown:
		if len(a.state.Completions) > 0 {
			a.state.CompletionIdx = (a.state.CompletionIdx + 1) % len(a.state.Completions)
		}

	case key.Special == KeyUp:
		if len(a.state.Completions) > 0 {
			a.state.CompletionIdx = (a.state.CompletionIdx - 1 + len(a.state.Completions)) % len(a.state.Completions)
		}

	default:
		if field != nil && !isSelectField(field.Type) {
			// Text/project/person fields: use EditBuffer for typing
			handled := false
			switch {
			case key.Special == KeyBackspace:
				form.FieldDeleteChar()
				handled = true
			case key.Special == KeyDelete:
				form.FieldDeleteCharForward()
				handled = true
			case key.Special == KeyLeft:
				if field.Buf.Cursor > 0 {
					field.Buf.Cursor--
				}
				handled = true
			case key.Special == KeyRight:
				if field.Buf.Cursor < len(field.Buf.Data) {
					field.Buf.Cursor++
				}
				handled = true
			case key.Special == KeyWordLeft:
				form.FieldWordLeft()
				handled = true
			case key.Special == KeyWordRight:
				form.FieldWordRight()
				handled = true
			case key.Special == KeyDeleteWord:
				form.FieldDeleteWord()
				handled = true
			case key.Special == KeyHome:
				field.Buf.Cursor = 0
				handled = true
			case key.Special == KeyEnd:
				field.Buf.Cursor = len(field.Buf.Data)
				handled = true
			case key.Special == KeyKillLine:
				field.Buf.KillLine()
				handled = true
			case key.Special == KeyKillBack:
				field.Buf.KillBack()
				handled = true
			case key.Char != 0:
				form.FieldInsertChar(key.Char)
				handled = true
			}
			if handled {
				a.updateFormCompletions()
			}
		}
		// Select fields: j/k handled above, typing not allowed (use arrows to pick)
	}

	return false
}

// acceptAndNextField accepts any active completion and moves to the next field.
func (a *App) acceptAndNextField(form *Form) {
	if a.state.CompletionIdx >= 0 && len(a.state.Completions) > 0 {
		a.acceptFormCompletion()
	}
	form.NextField()
	a.showFieldOptions()
}

// acceptAndPrevField accepts any active completion and moves to the previous field.
func (a *App) acceptAndPrevField(form *Form) {
	if a.state.CompletionIdx >= 0 && len(a.state.Completions) > 0 {
		a.acceptFormCompletion()
	}
	form.PrevField()
	a.showFieldOptions()
}

// showFieldOptions shows all available options for the current field.
func (a *App) showFieldOptions() {
	a.updateFormCompletions()
}

// updateFormCompletions refreshes the completion list based on the
// active form field. For select fields, shows all options. For text
// fields, shows matching completions.
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

	switch field.Type {
	case "status":
		// Show all states — select fields must not filter by prefix,
		// otherwise editing locks the user into the current value.
		matches := []string{"(none)"}
		matches = append(matches, a.cfg.Symbols.StateNames()...)
		a.state.Completions = matches
		a.state.CompletionType = "status"
		// Highlight the current value
		a.state.CompletionIdx = 0
		for i, m := range matches {
			if strings.EqualFold(m, prefix) {
				a.state.CompletionIdx = i
				break
			}
		}
		return
	case "symbol":
		matches := a.cfg.Symbols.SymbolNames()
		a.state.Completions = matches
		a.state.CompletionType = "symbol"
		a.state.CompletionIdx = 0
		for i, m := range matches {
			if strings.EqualFold(m, prefix) {
				a.state.CompletionIdx = i
				break
			}
		}
		return
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
	if completion == "(none)" {
		completion = ""
	}
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
