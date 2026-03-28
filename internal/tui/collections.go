package tui

// enterCollections switches to the collections list mode.
func (a *App) enterCollections() {
	a.state.Mode = ModeCollections
	a.state.ColCursor = 0
	a.state.ColScroll = 0
	a.state.ColAdding = false
	a.state.ColEditBuf.Clear()
	a.reloadCollections()
}

func (a *App) reloadCollections() {
	names, err := a.colSvc.List()
	if err != nil {
		a.state.StatusMsg = err.Error()
		a.state.ColNames = nil
		return
	}
	a.state.ColNames = names
}

func (a *App) handleCollectionsKey(key Key) bool {
	// Adding a new collection
	if a.state.ColAdding {
		return a.handleCollectionsAdd(key)
	}

	// Confirming a delete
	if a.state.ColConfirm {
		return a.handleCollectionsConfirm(key)
	}

	switch {
	case key.Special == KeyEscape:
		a.state.Mode = ModeNormal

	case key.Char == 'q':
		return true

	case key.Char == 'j' || key.Special == KeyDown:
		if a.state.ColCursor < len(a.state.ColNames)-1 {
			a.state.ColCursor++
		}

	case key.Char == 'k' || key.Special == KeyUp:
		if a.state.ColCursor > 0 {
			a.state.ColCursor--
		}

	case key.Char == 'G':
		if len(a.state.ColNames) > 0 {
			a.state.ColCursor = len(a.state.ColNames) - 1
		}

	case key.Char == 'g':
		a.state.ColCursor = 0

	case key.Special == KeyEnter:
		if len(a.state.ColNames) > 0 && a.state.ColCursor < len(a.state.ColNames) {
			a.enterCollection(a.state.ColNames[a.state.ColCursor])
		}

	case key.Char == 'a':
		a.state.ColAdding = true
		a.state.ColEditBuf.Clear()

	case key.Char == 'd':
		if len(a.state.ColNames) > 0 && a.state.ColCursor < len(a.state.ColNames) {
			a.state.ColConfirm = true
		}

	case key.Char == '?':
		a.state.Mode = ModeHelp
	}

	return false
}

func (a *App) handleCollectionsConfirm(key Key) bool {
	switch {
	case key.Char == 'y' || key.Char == 'Y':
		name := a.state.ColNames[a.state.ColCursor]
		if err := a.colSvc.Delete(name); err != nil {
			a.state.StatusMsg = err.Error()
		} else {
			a.state.StatusMsg = "Deleted: " + name
			a.reloadCollections()
			if a.state.ColCursor >= len(a.state.ColNames) && a.state.ColCursor > 0 {
				a.state.ColCursor--
			}
		}
		a.state.ColConfirm = false
	case key.Char == 'n' || key.Char == 'N' || key.Special == KeyEscape:
		a.state.ColConfirm = false
	}
	return false
}

func (a *App) handleCollectionsAdd(key Key) bool {
	switch {
	case key.Special == KeyEscape:
		a.state.ColAdding = false
		a.state.ColEditBuf.Clear()

	case key.Special == KeyEnter:
		name := a.state.ColEditBuf.String()
		if name != "" {
			if _, err := a.colSvc.Create(name); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.reloadCollections()
				// Move cursor to the new collection
				for i, n := range a.state.ColNames {
					if n == name {
						a.state.ColCursor = i
						break
					}
				}
			}
		}
		a.state.ColAdding = false
		a.state.ColEditBuf.Clear()

	case key.Special == KeyBackspace:
		a.state.ColEditBuf.DeleteChar()
	case key.Special == KeyDelete:
		a.state.ColEditBuf.DeleteCharForward()
	case key.Special == KeyLeft:
		if a.state.ColEditBuf.Cursor > 0 {
			a.state.ColEditBuf.Cursor--
		}
	case key.Special == KeyRight:
		if a.state.ColEditBuf.Cursor < len(a.state.ColEditBuf.Data) {
			a.state.ColEditBuf.Cursor++
		}
	case key.Special == KeyWordLeft:
		a.state.ColEditBuf.WordLeft()
	case key.Special == KeyWordRight:
		a.state.ColEditBuf.WordRight()
	case key.Special == KeyDeleteWord:
		a.state.ColEditBuf.DeleteWord()
	case key.Special == KeyHome:
		a.state.ColEditBuf.Cursor = 0
	case key.Special == KeyEnd:
		a.state.ColEditBuf.Cursor = len(a.state.ColEditBuf.Data)
	case key.Special == KeyKillLine:
		a.state.ColEditBuf.KillLine()
	case key.Special == KeyKillBack:
		a.state.ColEditBuf.KillBack()
	case key.Char != 0:
		a.state.ColEditBuf.InsertChar(key.Char)
	}

	return false
}

// enterCollection switches to viewing a specific collection.
func (a *App) enterCollection(name string) {
	col, err := a.colSvc.Get(name)
	if err != nil {
		a.state.StatusMsg = err.Error()
		return
	}

	a.state.Mode = ModeCollection
	a.state.ColName = col.Name
	a.state.ColItemCursor = 0
	a.state.ColItemScroll = 0
	a.state.ColEditing = false
	a.state.ColEditBuf.Clear()

	items := make([]ColViewItem, len(col.Items))
	for i, item := range col.Items {
		items[i] = ColViewItem{Text: item.Text, Done: item.Done}
	}
	a.state.ColItems = items
}

func (a *App) reloadCollection() {
	col, err := a.colSvc.Get(a.state.ColName)
	if err != nil {
		a.state.StatusMsg = err.Error()
		return
	}
	items := make([]ColViewItem, len(col.Items))
	for i, item := range col.Items {
		items[i] = ColViewItem{Text: item.Text, Done: item.Done}
	}
	a.state.ColItems = items
}

func (a *App) handleCollectionKey(key Key) bool {
	// Editing/adding an item
	if a.state.ColEditing {
		return a.handleCollectionEdit(key)
	}

	switch {
	case key.Special == KeyEscape:
		// Back to collections list
		a.enterCollections()

	case key.Char == 'q':
		return true

	case key.Char == 'j' || key.Special == KeyDown:
		if a.state.ColItemCursor < len(a.state.ColItems)-1 {
			a.state.ColItemCursor++
		}

	case key.Char == 'k' || key.Special == KeyUp:
		if a.state.ColItemCursor > 0 {
			a.state.ColItemCursor--
		}

	case key.Char == 'G':
		if len(a.state.ColItems) > 0 {
			a.state.ColItemCursor = len(a.state.ColItems) - 1
		}

	case key.Char == 'g':
		a.state.ColItemCursor = 0

	case key.Char == 'a':
		a.state.ColEditing = true
		a.state.ColEditIdx = -1
		a.state.ColEditBuf.Clear()

	case key.Char == 'e':
		if len(a.state.ColItems) > 0 && a.state.ColItemCursor < len(a.state.ColItems) {
			a.state.ColEditing = true
			a.state.ColEditIdx = a.state.ColItemCursor
			a.state.ColEditBuf.Set(a.state.ColItems[a.state.ColItemCursor].Text)
		}

	case key.Char == 'x' || key.Char == ' ':
		if len(a.state.ColItems) > 0 && a.state.ColItemCursor < len(a.state.ColItems) {
			if err := a.colSvc.ToggleItem(a.state.ColName, a.state.ColItemCursor); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.reloadCollection()
			}
		}

	case key.Char == 'd':
		if len(a.state.ColItems) > 0 && a.state.ColItemCursor < len(a.state.ColItems) {
			if err := a.colSvc.RemoveItem(a.state.ColName, a.state.ColItemCursor); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.reloadCollection()
				if a.state.ColItemCursor >= len(a.state.ColItems) && a.state.ColItemCursor > 0 {
					a.state.ColItemCursor--
				}
			}
		}

	case key.Char == 'J':
		// Move item down
		if a.state.ColItemCursor < len(a.state.ColItems)-1 {
			if err := a.colSvc.MoveItem(a.state.ColName, a.state.ColItemCursor, a.state.ColItemCursor+1); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.state.ColItemCursor++
				a.reloadCollection()
			}
		}

	case key.Char == 'K':
		// Move item up
		if a.state.ColItemCursor > 0 {
			if err := a.colSvc.MoveItem(a.state.ColName, a.state.ColItemCursor, a.state.ColItemCursor-1); err != nil {
				a.state.StatusMsg = err.Error()
			} else {
				a.state.ColItemCursor--
				a.reloadCollection()
			}
		}

	case key.Char == '?':
		a.state.Mode = ModeHelp
	}

	return false
}

func (a *App) handleCollectionEdit(key Key) bool {
	switch {
	case key.Special == KeyEscape:
		a.state.ColEditing = false
		a.state.ColEditBuf.Clear()

	case key.Special == KeyEnter:
		text := a.state.ColEditBuf.String()
		if text != "" {
			if a.state.ColEditIdx < 0 {
				// Add new item
				if err := a.colSvc.AddItem(a.state.ColName, text); err != nil {
					a.state.StatusMsg = err.Error()
				} else {
					a.reloadCollection()
					a.state.ColItemCursor = len(a.state.ColItems) - 1
				}
			} else {
				// Edit existing item
				if err := a.colSvc.EditItem(a.state.ColName, a.state.ColEditIdx, text); err != nil {
					a.state.StatusMsg = err.Error()
				} else {
					a.reloadCollection()
				}
			}
		}
		a.state.ColEditing = false
		a.state.ColEditBuf.Clear()

	case key.Special == KeyBackspace:
		a.state.ColEditBuf.DeleteChar()
	case key.Special == KeyDelete:
		a.state.ColEditBuf.DeleteCharForward()
	case key.Special == KeyLeft:
		if a.state.ColEditBuf.Cursor > 0 {
			a.state.ColEditBuf.Cursor--
		}
	case key.Special == KeyRight:
		if a.state.ColEditBuf.Cursor < len(a.state.ColEditBuf.Data) {
			a.state.ColEditBuf.Cursor++
		}
	case key.Special == KeyWordLeft:
		a.state.ColEditBuf.WordLeft()
	case key.Special == KeyWordRight:
		a.state.ColEditBuf.WordRight()
	case key.Special == KeyDeleteWord:
		a.state.ColEditBuf.DeleteWord()
	case key.Special == KeyHome:
		a.state.ColEditBuf.Cursor = 0
	case key.Special == KeyEnd:
		a.state.ColEditBuf.Cursor = len(a.state.ColEditBuf.Data)
	case key.Special == KeyKillLine:
		a.state.ColEditBuf.KillLine()
	case key.Special == KeyKillBack:
		a.state.ColEditBuf.KillBack()
	case key.Char != 0:
		a.state.ColEditBuf.InsertChar(key.Char)
	}

	return false
}
