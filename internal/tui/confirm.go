package tui

func (a *App) handleConfirmKey(key Key) bool {
	switch {
	case key.Char == 'y' || key.Char == 'Y':
		if err := a.svc.DeleteEntry(a.date, a.state.ConfirmIndex); err != nil {
			a.state.StatusMsg = err.Error()
		} else {
			a.loadEntries()
		}
		a.state.Mode = ModeNormal

	case key.Char == 'n' || key.Char == 'N' || key.Special == KeyEscape:
		a.state.Mode = ModeNormal
	}

	return false
}

func (a *App) handleHelpKey(key Key) bool {
	// Any key closes help
	a.state.Mode = ModeNormal
	return false
}
