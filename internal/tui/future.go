package tui

import "time"

// enterFuture switches to the future log mode, showing 6 months ahead.
func (a *App) enterFuture() {
	now := time.Now()
	a.state.Mode = ModeFuture
	a.state.FutYear = now.Year()
	a.state.FutMonthIdx = 0
	a.state.FutItemIdx = 0
	a.state.FutAdding = false
	a.state.FutConfirm = false
	a.state.FutEditBuf.Clear()
	a.reloadFuture()
}

func (a *App) reloadFuture() {
	now := time.Now()
	startMonth := now.Month()
	startYear := now.Year()

	var viewMonths []FutureViewMonth

	// Load up to 6 months starting from current
	for i := 0; i < 6; i++ {
		m := int(startMonth) + i
		y := startYear
		for m > 12 {
			m -= 12
			y++
		}

		label := time.Month(m).String() + " " + time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.Local).Format("2006")
		vm := FutureViewMonth{
			Year:  y,
			Month: m,
			Label: label,
		}

		// Load entries for this month's year
		months, err := a.futSvc.LoadYear(y)
		if err != nil {
			a.state.StatusMsg = err.Error()
		} else {
			for _, fm := range months {
				if fm.Month == m {
					for _, e := range fm.Entries {
						vm.Entries = append(vm.Entries, FutureViewEntry{
							Symbol: e.Symbol.Name,
							Desc:   e.Description,
						})
					}
					break
				}
			}
		}

		viewMonths = append(viewMonths, vm)
	}

	a.state.FutMonths = viewMonths

	// Clamp cursors
	if a.state.FutMonthIdx >= len(viewMonths) {
		a.state.FutMonthIdx = len(viewMonths) - 1
	}
	if a.state.FutMonthIdx < 0 {
		a.state.FutMonthIdx = 0
	}
	a.clampFutItemIdx()
}

func (a *App) clampFutItemIdx() {
	if a.state.FutMonthIdx < len(a.state.FutMonths) {
		entries := a.state.FutMonths[a.state.FutMonthIdx].Entries
		if a.state.FutItemIdx >= len(entries) {
			a.state.FutItemIdx = len(entries) - 1
		}
	}
	if a.state.FutItemIdx < 0 {
		a.state.FutItemIdx = 0
	}
}

func (a *App) handleFutureKey(key Key) bool {
	if a.state.FutAdding {
		return a.handleFutureAdd(key)
	}
	if a.state.FutConfirm {
		return a.handleFutureConfirm(key)
	}

	switch {
	case key.Special == KeyEscape:
		a.state.Mode = ModeNormal

	case key.Char == 'q':
		return true

	// Navigate months with [/]
	case key.Char == ']':
		if a.state.FutMonthIdx < len(a.state.FutMonths)-1 {
			a.state.FutMonthIdx++
			a.state.FutItemIdx = 0
		}

	case key.Char == '[':
		if a.state.FutMonthIdx > 0 {
			a.state.FutMonthIdx--
			a.state.FutItemIdx = 0
		}

	// Navigate entries within month
	case key.Char == 'j' || key.Special == KeyDown:
		if a.state.FutMonthIdx < len(a.state.FutMonths) {
			entries := a.state.FutMonths[a.state.FutMonthIdx].Entries
			if a.state.FutItemIdx < len(entries)-1 {
				a.state.FutItemIdx++
			}
		}

	case key.Char == 'k' || key.Special == KeyUp:
		if a.state.FutItemIdx > 0 {
			a.state.FutItemIdx--
		}

	case key.Char == 'a':
		a.state.FutAdding = true
		a.state.FutEditBuf.Clear()

	case key.Char == 'd':
		if a.state.FutMonthIdx < len(a.state.FutMonths) {
			entries := a.state.FutMonths[a.state.FutMonthIdx].Entries
			if len(entries) > 0 && a.state.FutItemIdx < len(entries) {
				a.state.FutConfirm = true
			}
		}

	case key.Char == '?':
		a.state.Mode = ModeHelp
	}

	return false
}

func (a *App) handleFutureAdd(key Key) bool {
	switch {
	case key.Special == KeyEscape:
		a.state.FutAdding = false
		a.state.FutEditBuf.Clear()

	case key.Special == KeyEnter:
		desc := a.state.FutEditBuf.String()
		if desc != "" && a.state.FutMonthIdx < len(a.state.FutMonths) {
			vm := a.state.FutMonths[a.state.FutMonthIdx]
			if err := a.futSvc.AddEntry(vm.Year, vm.Month, "", desc); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.reloadFuture()
				// Move to last entry
				if a.state.FutMonthIdx < len(a.state.FutMonths) {
					a.state.FutItemIdx = len(a.state.FutMonths[a.state.FutMonthIdx].Entries) - 1
				}
			}
		}
		a.state.FutAdding = false
		a.state.FutEditBuf.Clear()

	default:
		a.state.FutEditBuf.HandleKey(key)
	}

	return false
}

func (a *App) handleFutureConfirm(key Key) bool {
	switch {
	case key.Char == 'y' || key.Char == 'Y':
		if a.state.FutMonthIdx < len(a.state.FutMonths) {
			vm := a.state.FutMonths[a.state.FutMonthIdx]
			if err := a.futSvc.RemoveEntry(vm.Year, vm.Month, a.state.FutItemIdx); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.reloadFuture()
				a.clampFutItemIdx()
			}
		}
		a.state.FutConfirm = false
	case key.Char == 'n' || key.Char == 'N' || key.Special == KeyEscape:
		a.state.FutConfirm = false
	}
	return false
}
