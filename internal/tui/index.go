package tui

import "strings"

// enterIndex switches to the index mode, building entries from collections and projects.
func (a *App) enterIndex() {
	a.state.Mode = ModeIndex
	a.state.IdxCursor = 0
	a.state.IdxScroll = 0
	a.state.IdxFiltering = false
	a.state.IdxFilterBuf.Clear()
	a.reloadIndex()
}

func (a *App) reloadIndex() {
	var entries []IndexEntry

	// Collections
	names, err := a.colSvc.List()
	if err == nil {
		for _, n := range names {
			entries = append(entries, IndexEntry{Kind: "collection", Name: n})
		}
	}

	// Projects from config
	for _, p := range a.cfg.Projects {
		entries = append(entries, IndexEntry{Kind: "project", Name: p})
	}

	// Projects discovered from current day's entries
	seen := make(map[string]bool)
	for _, p := range a.cfg.Projects {
		seen[p] = true
	}
	for _, e := range a.allDay {
		if e.Project != "" && !seen[e.Project] {
			seen[e.Project] = true
			entries = append(entries, IndexEntry{Kind: "project", Name: e.Project})
		}
	}

	a.state.IdxEntries = entries
	a.applyIndexFilter()
}

func (a *App) applyIndexFilter() {
	query := strings.ToLower(a.state.IdxFilterBuf.String())
	a.state.IdxFiltered = nil

	for i, entry := range a.state.IdxEntries {
		if query == "" || strings.Contains(strings.ToLower(entry.Name), query) || strings.Contains(entry.Kind, query) {
			a.state.IdxFiltered = append(a.state.IdxFiltered, i)
		}
	}

	if a.state.IdxCursor >= len(a.state.IdxFiltered) {
		a.state.IdxCursor = len(a.state.IdxFiltered) - 1
	}
	if a.state.IdxCursor < 0 {
		a.state.IdxCursor = 0
	}
}

func (a *App) handleIndexKey(key Key) bool {
	if a.state.IdxFiltering {
		return a.handleIndexFilter(key)
	}

	switch {
	case key.Special == KeyEscape:
		a.state.Mode = ModeNormal

	case key.Char == 'q':
		return true

	case key.Char == 'j' || key.Special == KeyDown:
		if a.state.IdxCursor < len(a.state.IdxFiltered)-1 {
			a.state.IdxCursor++
		}

	case key.Char == 'k' || key.Special == KeyUp:
		if a.state.IdxCursor > 0 {
			a.state.IdxCursor--
		}

	case key.Char == 'G':
		if len(a.state.IdxFiltered) > 0 {
			a.state.IdxCursor = len(a.state.IdxFiltered) - 1
		}

	case key.Char == 'g':
		a.state.IdxCursor = 0

	case key.Char == '/':
		a.state.IdxFiltering = true
		a.state.IdxFilterBuf.Clear()

	case key.Special == KeyEnter:
		if a.state.IdxCursor < len(a.state.IdxFiltered) {
			entry := a.state.IdxEntries[a.state.IdxFiltered[a.state.IdxCursor]]
			switch entry.Kind {
			case "collection":
				a.enterCollection(entry.Name)
			case "project":
				// Navigate to normal mode with project filter
				a.state.Mode = ModeNormal
				a.state.FilterProject = entry.Name
				a.state.FilterPerson = ""
				a.state.FilterSymbol = ""
				a.state.FilterText = ""
				a.applyFilter()
			}
		}

	case key.Char == '?':
		a.state.Mode = ModeHelp
	}

	return false
}

func (a *App) handleIndexFilter(key Key) bool {
	switch {
	case key.Special == KeyEscape:
		a.state.IdxFiltering = false
		a.state.IdxFilterBuf.Clear()
		a.applyIndexFilter()

	case key.Special == KeyEnter:
		a.state.IdxFiltering = false
		// Keep the filter applied

	case key.Special == KeyBackspace:
		a.state.IdxFilterBuf.DeleteChar()
		a.applyIndexFilter()
	case key.Special == KeyDelete:
		a.state.IdxFilterBuf.DeleteCharForward()
		a.applyIndexFilter()
	case key.Special == KeyWordLeft:
		a.state.IdxFilterBuf.WordLeft()
	case key.Special == KeyWordRight:
		a.state.IdxFilterBuf.WordRight()
	case key.Special == KeyDeleteWord:
		a.state.IdxFilterBuf.DeleteWord()
		a.applyIndexFilter()
	case key.Special == KeyHome:
		a.state.IdxFilterBuf.Cursor = 0
	case key.Special == KeyEnd:
		a.state.IdxFilterBuf.Cursor = len(a.state.IdxFilterBuf.Data)
	case key.Special == KeyKillLine:
		a.state.IdxFilterBuf.KillLine()
		a.applyIndexFilter()
	case key.Special == KeyKillBack:
		a.state.IdxFilterBuf.KillBack()
		a.applyIndexFilter()
	case key.Char != 0:
		a.state.IdxFilterBuf.InsertChar(key.Char)
		a.applyIndexFilter()
	}

	return false
}
