package tui

import "fmt"

// handleNormalKey processes key presses while in normal (list) mode.
// It returns true if the application should quit.
func (a *App) handleNormalKey(key Key) bool {
	switch {
	case key.Char == 'q':
		return true // quit

	case key.Special == KeyEscape:
		a.state.FilterProject = ""
		a.state.FilterPerson = ""
		a.state.FilterSymbol = ""
		a.state.FilterText = ""
		a.applyFilter()

	case key.Char == 'j' || key.Special == KeyDown:
		if a.state.Cursor < len(a.entries)-1 {
			a.state.Cursor++
		}

	case key.Char == 'k' || key.Special == KeyUp:
		if a.state.Cursor > 0 {
			a.state.Cursor--
		}

	case key.Char == 't':
		a.state.ShowTime = !a.state.ShowTime

	case key.Char == 'a':
		a.state.Mode = ModeForm
		a.state.Form = &Form{
			Fields: []FormField{
				{Label: "Status", Buf: EditBuffer{}, Type: "status"},
				{Label: "Symbol", Buf: EditBuffer{}, Type: "symbol"},
				{Label: "Project", Buf: EditBuffer{}, Type: "project"},
				{Label: "Assignee", Buf: EditBuffer{}, Type: "person"},
				{Label: "Description", Buf: EditBuffer{}, Type: "text"},
			},
			Active:  4, // focus description
			IsEdit:  false,
			EditIdx: -1,
		}

	case key.Char == 'e':
		if len(a.entries) > 0 && a.state.Cursor < len(a.entries) {
			e := a.entries[a.state.Cursor]
			realIdx := a.entryIndexMap[a.state.Cursor]
			a.state.Mode = ModeForm
			a.state.Form = &Form{
				Fields: []FormField{
					{Label: "Status", Buf: EditBuffer{Data: []byte(e.State), Cursor: len(e.State)}, Type: "status"},
					{Label: "Symbol", Buf: EditBuffer{Data: []byte(e.Symbol.Name), Cursor: len(e.Symbol.Name)}, Type: "symbol"},
					{Label: "Project", Buf: EditBuffer{Data: []byte(e.Project), Cursor: len(e.Project)}, Type: "project"},
					{Label: "Assignee", Buf: EditBuffer{Data: []byte(e.Person), Cursor: len(e.Person)}, Type: "person"},
					{Label: "Description", Buf: EditBuffer{Data: []byte(e.Description), Cursor: len(e.Description)}, Type: "text"},
				},
				Active:  0,
				IsEdit:  true,
				EditIdx: realIdx,
			}
		}

	case key.Char == 'x':
		if len(a.entries) > 0 && a.state.Cursor < len(a.entries) {
			realIdx := a.entryIndexMap[a.state.Cursor]
			if err := a.svc.TransitionEntry(a.date, realIdx, "done"); err != nil {
				a.state.StatusMsg = err.Error()
				return false
			}
			a.loadEntries()
		}

	case key.Char == '>':
		if len(a.entries) > 0 && a.state.Cursor < len(a.entries) {
			realIdx := a.entryIndexMap[a.state.Cursor]
			tomorrow := a.date.AddDate(0, 0, 1).Format("2006-01-02")
			a.state.Mode = ModeMigrate
			a.state.MigrateIndex = realIdx
			a.state.MigrateDate.Set(tomorrow)
		}

	case key.Char == '<':
		if len(a.entries) > 0 && a.state.Cursor < len(a.entries) {
			realIdx := a.entryIndexMap[a.state.Cursor]
			if err := a.svc.TransitionEntry(a.date, realIdx, "scheduled"); err != nil {
				a.state.StatusMsg = err.Error()
				return false
			}
			a.loadEntries()
		}

	case key.Char == 'c':
		if len(a.entries) > 0 && a.state.Cursor < len(a.entries) {
			realIdx := a.entryIndexMap[a.state.Cursor]
			if err := a.svc.TransitionEntry(a.date, realIdx, "cancelled"); err != nil {
				a.state.StatusMsg = err.Error()
				return false
			}
			a.loadEntries()
		}

	case key.Char == 'r':
		if len(a.entries) > 0 && a.state.Cursor < len(a.entries) {
			realIdx := a.entryIndexMap[a.state.Cursor]
			if err := a.svc.ResetState(a.date, realIdx); err != nil {
				a.state.StatusMsg = err.Error()
				return false
			}
			a.loadEntries()
		}

	case key.Char == 'd':
		if len(a.entries) > 0 && a.state.Cursor < len(a.entries) {
			e := a.entries[a.state.Cursor]
			realIdx := a.entryIndexMap[a.state.Cursor]
			a.state.Mode = ModeConfirm
			a.state.ConfirmMsg = fmt.Sprintf("Delete '%s'?", e.Description)
			a.state.ConfirmIndex = realIdx
		}

	case key.Char == '/':
		a.state.Mode = ModeFilter
		a.state.InputPrompt = "filter> "
		a.state.ClearInput()

	case key.Char == '[':
		a.date = a.date.AddDate(0, 0, -1)
		a.loadEntries()
		a.state.Cursor = 0

	case key.Char == ']':
		a.date = a.date.AddDate(0, 0, 1)
		a.loadEntries()
		a.state.Cursor = 0

	case key.Char == 'G':
		if len(a.entries) > 0 {
			a.state.Cursor = len(a.entries) - 1
		}

	case key.Char == 'g':
		a.state.Cursor = 0

	case key.Char == '?':
		a.state.Mode = ModeHelp
	}

	return false
}
