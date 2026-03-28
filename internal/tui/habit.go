package tui

import "time"

// enterHabit switches to the habit tracker mode for the current month.
func (a *App) enterHabit() {
	month := time.Date(a.date.Year(), a.date.Month(), 1, 0, 0, 0, 0, a.date.Location())
	a.state.Mode = ModeHabit
	a.state.HabMonth = month
	a.state.HabRow = 0
	a.state.HabCol = a.date.Day() - 1
	a.state.HabAdding = false
	a.state.HabConfirm = false
	a.state.HabEditBuf.Clear()
	a.reloadHabits()
}

func (a *App) reloadHabits() {
	ht, err := a.habSvc.LoadMonth(a.state.HabMonth)
	if err != nil {
		a.state.StatusMsg = err.Error()
		a.state.HabTracker = nil
		return
	}

	numDays := daysInMonth(a.state.HabMonth)
	today := 0
	now := time.Now()
	if now.Year() == a.state.HabMonth.Year() && now.Month() == a.state.HabMonth.Month() {
		today = now.Day()
	} else {
		today = numDays
	}

	streaks := make(map[string]int)
	for _, h := range ht.Habits {
		streaks[h] = ht.Streak(h, today)
	}

	a.state.HabTracker = &HabitViewData{
		Habits:  ht.Habits,
		Done:    ht.Done,
		NumDays: numDays,
		Streaks: streaks,
	}
}

func (a *App) handleHabitKey(key Key) bool {
	if a.state.HabAdding {
		return a.handleHabitAdd(key)
	}
	if a.state.HabConfirm {
		return a.handleHabitConfirm(key)
	}

	ht := a.state.HabTracker
	if ht == nil {
		if key.Special == KeyEscape || key.Char == 'q' {
			a.state.Mode = ModeNormal
			return key.Char == 'q'
		}
		return false
	}

	switch {
	case key.Special == KeyEscape:
		a.state.Mode = ModeNormal

	case key.Char == 'q':
		return true

	case key.Char == 'j' || key.Special == KeyDown:
		if a.state.HabRow < len(ht.Habits)-1 {
			a.state.HabRow++
		}

	case key.Char == 'k' || key.Special == KeyUp:
		if a.state.HabRow > 0 {
			a.state.HabRow--
		}

	case key.Char == ']' || key.Special == KeyRight:
		if a.state.HabCol < ht.NumDays-1 {
			a.state.HabCol++
		}

	case key.Char == '[' || key.Special == KeyLeft:
		if a.state.HabCol > 0 {
			a.state.HabCol--
		}

	case key.Char == 'x' || key.Char == ' ' || key.Special == KeyEnter:
		if len(ht.Habits) > 0 && a.state.HabRow < len(ht.Habits) {
			habit := ht.Habits[a.state.HabRow]
			day := a.state.HabCol + 1
			if err := a.habSvc.Toggle(a.state.HabMonth, habit, day); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.reloadHabits()
			}
		}

	case key.Char == 'a':
		a.state.HabAdding = true
		a.state.HabEditBuf.Clear()

	case key.Char == 'd':
		if len(ht.Habits) > 0 && a.state.HabRow < len(ht.Habits) {
			a.state.HabConfirm = true
		}

	case key.Char == '[':
		a.state.HabMonth = a.state.HabMonth.AddDate(0, -1, 0)
		a.state.HabCol = 0
		a.state.HabRow = 0
		a.reloadHabits()

	case key.Char == ']':
		a.state.HabMonth = a.state.HabMonth.AddDate(0, 1, 0)
		a.state.HabCol = 0
		a.state.HabRow = 0
		a.reloadHabits()

	case key.Char == '?':
		a.state.Mode = ModeHelp
	}

	return false
}

func (a *App) handleHabitAdd(key Key) bool {
	switch {
	case key.Special == KeyEscape:
		a.state.HabAdding = false
		a.state.HabEditBuf.Clear()

	case key.Special == KeyEnter:
		name := a.state.HabEditBuf.String()
		if name != "" {
			if err := a.habSvc.AddHabit(a.state.HabMonth, name); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.reloadHabits()
				if a.state.HabTracker != nil {
					a.state.HabRow = len(a.state.HabTracker.Habits) - 1
				}
			}
		}
		a.state.HabAdding = false
		a.state.HabEditBuf.Clear()

	default:
		a.state.HabEditBuf.HandleKey(key)
	}

	return false
}

func (a *App) handleHabitConfirm(key Key) bool {
	switch {
	case key.Char == 'y' || key.Char == 'Y':
		if a.state.HabTracker != nil && a.state.HabRow < len(a.state.HabTracker.Habits) {
			name := a.state.HabTracker.Habits[a.state.HabRow]
			if err := a.habSvc.RemoveHabit(a.state.HabMonth, name); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.reloadHabits()
				if a.state.HabTracker != nil && a.state.HabRow >= len(a.state.HabTracker.Habits) && a.state.HabRow > 0 {
					a.state.HabRow--
				}
			}
		}
		a.state.HabConfirm = false
	case key.Char == 'n' || key.Char == 'N' || key.Special == KeyEscape:
		a.state.HabConfirm = false
	}
	return false
}
