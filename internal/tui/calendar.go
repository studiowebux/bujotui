package tui

import (
	"time"

	"github.com/studiowebux/bujotui/internal/model"
)

// enterCalendar switches to calendar mode, loading the current month's data.
func (a *App) enterCalendar() {
	month := time.Date(a.date.Year(), a.date.Month(), 1, 0, 0, 0, 0, a.date.Location())
	a.state.Mode = ModeCalendar
	a.state.CalMonth = month
	a.state.CalCursor = a.date.Day() - 1
	a.state.CalEditing = false
	a.reloadCalendar()
}

// daysInMonth returns the number of days in the given month.
func daysInMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}

func (a *App) handleCalendarKey(key Key) bool {
	// If editing a note, handle text input
	if a.state.CalEditing {
		return a.handleCalendarNoteEdit(key)
	}

	numDays := daysInMonth(a.state.CalMonth)

	switch {
	case key.Special == KeyEscape:
		a.state.Mode = ModeNormal

	case key.Char == 'q':
		return true

	case key.Char == 'j' || key.Special == KeyDown:
		if a.state.CalCursor < numDays-1 {
			a.state.CalCursor++
		}

	case key.Char == 'k' || key.Special == KeyUp:
		if a.state.CalCursor > 0 {
			a.state.CalCursor--
		}

	case key.Char == 'G':
		a.state.CalCursor = numDays - 1

	case key.Char == 'g':
		a.state.CalCursor = 0

	case key.Special == KeyEnter:
		// Open the selected day in normal view
		day := a.state.CalCursor + 1
		a.date = time.Date(
			a.state.CalMonth.Year(), a.state.CalMonth.Month(), day,
			0, 0, 0, 0, a.state.CalMonth.Location(),
		)
		a.loadEntries()
		a.state.Mode = ModeNormal

	case key.Char == 'i' || key.Char == 'n':
		// Edit note for selected day
		day := a.state.CalCursor + 1
		existing := a.state.CalNotes[day]
		a.state.CalEditing = true
		a.state.CalNoteBuf.Set(existing)

	case key.Char == '[':
		a.state.CalMonth = a.state.CalMonth.AddDate(0, -1, 0)
		a.state.CalCursor = 0
		a.reloadCalendar()

	case key.Char == ']':
		a.state.CalMonth = a.state.CalMonth.AddDate(0, 1, 0)
		a.state.CalCursor = 0
		a.reloadCalendar()

	case key.Char == '?':
		a.state.Mode = ModeHelp
	}

	return false
}

func (a *App) handleCalendarNoteEdit(key Key) bool {
	switch {
	case key.Special == KeyEscape:
		a.state.CalEditing = false

	case key.Special == KeyEnter:
		day := a.state.CalCursor + 1
		note := a.state.CalNoteBuf.String()
		date := time.Date(
			a.state.CalMonth.Year(), a.state.CalMonth.Month(), day,
			12, 0, 0, 0, a.state.CalMonth.Location(),
		)
		if err := a.svc.SaveNote(date, note); err != nil {
			a.state.StatusMsg = err.Error()
		} else {
			a.state.CalNotes[day] = note
		}
		a.state.CalEditing = false

	default:
		a.state.CalNoteBuf.HandleKey(key)
	}

	return false
}

func (a *App) reloadCalendar() {
	entries, err := a.svc.LoadMonth(a.state.CalMonth)
	if err != nil {
		a.state.StatusMsg = err.Error()
		a.state.CalEntries = make(map[int][]model.Entry)
		a.state.CalNotes = make(map[int]string)
		return
	}
	notes, err := a.svc.LoadMonthNotes(a.state.CalMonth)
	if err != nil {
		a.state.StatusMsg = err.Error()
		notes = make(map[int]string)
	}
	a.state.CalEntries = entries
	a.state.CalNotes = notes
}
