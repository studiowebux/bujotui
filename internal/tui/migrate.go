package tui

import "time"

// handleMigrateKey processes key events in the migrate date picker modal.
func (a *App) handleMigrateKey(key Key) bool {
	switch {
	case key.Special == KeyEscape:
		a.state.Mode = ModeNormal
		a.state.MigrateDate.Clear()

	case key.Special == KeyEnter:
		dateStr := a.state.MigrateDate.String()
		targetDate, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			a.state.StatusMsg = "invalid date format (expected YYYY-MM-DD)"
			return false
		}

		if err := a.svc.MigrateEntry(a.date, a.state.MigrateIndex, targetDate); err != nil {
			a.state.StatusMsg = err.Error()
			a.state.Mode = ModeNormal
			a.state.MigrateDate.Clear()
			return false
		}

		a.loadEntries()
		a.state.Mode = ModeNormal
		a.state.MigrateDate.Clear()

	case key.Special == KeyBackspace:
		a.state.MigrateDate.DeleteChar()

	case key.Special == KeyLeft:
		if a.state.MigrateDate.Cursor > 0 {
			a.state.MigrateDate.Cursor--
		}

	case key.Special == KeyRight:
		if a.state.MigrateDate.Cursor < len(a.state.MigrateDate.Data) {
			a.state.MigrateDate.Cursor++
		}

	case key.Special == KeyHome:
		a.state.MigrateDate.Cursor = 0

	case key.Special == KeyEnd:
		a.state.MigrateDate.Cursor = len(a.state.MigrateDate.Data)

	case key.Special == KeyWordLeft:
		a.state.MigrateDate.WordLeft()

	case key.Special == KeyWordRight:
		a.state.MigrateDate.WordRight()

	case key.Special == KeyDeleteWord:
		a.state.MigrateDate.DeleteWord()

	case key.Special == KeyKillLine:
		a.state.MigrateDate.KillLine()

	case key.Special == KeyKillBack:
		a.state.MigrateDate.KillBack()

	case key.Char != 0:
		a.state.MigrateDate.InsertChar(key.Char)
	}

	return false
}
