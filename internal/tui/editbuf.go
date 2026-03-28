package tui

// EditBuffer is a shared byte-buffer + cursor used for any single-line text
// editing (filter input, form fields, etc.).
type EditBuffer struct {
	Data   []byte
	Cursor int
}

// InsertChar inserts a character at the cursor position.
func (b *EditBuffer) InsertChar(c byte) {
	if b.Cursor >= len(b.Data) {
		b.Data = append(b.Data, c)
	} else {
		b.Data = append(b.Data, 0)
		copy(b.Data[b.Cursor+1:], b.Data[b.Cursor:])
		b.Data[b.Cursor] = c
	}
	b.Cursor++
}

// DeleteChar removes the character before the cursor (backspace).
func (b *EditBuffer) DeleteChar() {
	if b.Cursor > 0 && len(b.Data) > 0 {
		b.Data = append(b.Data[:b.Cursor-1], b.Data[b.Cursor:]...)
		b.Cursor--
	}
}

// DeleteCharForward removes the character at the cursor (Delete key).
func (b *EditBuffer) DeleteCharForward() {
	if b.Cursor < len(b.Data) {
		b.Data = append(b.Data[:b.Cursor], b.Data[b.Cursor+1:]...)
	}
}

// WordLeft moves the cursor to the start of the previous word.
func (b *EditBuffer) WordLeft() {
	for b.Cursor > 0 && b.Data[b.Cursor-1] == ' ' {
		b.Cursor--
	}
	for b.Cursor > 0 && b.Data[b.Cursor-1] != ' ' {
		b.Cursor--
	}
}

// WordRight moves the cursor to the end of the next word.
func (b *EditBuffer) WordRight() {
	for b.Cursor < len(b.Data) && b.Data[b.Cursor] != ' ' {
		b.Cursor++
	}
	for b.Cursor < len(b.Data) && b.Data[b.Cursor] == ' ' {
		b.Cursor++
	}
}

// DeleteWord removes the word before the cursor (Option+Backspace / Ctrl+W).
func (b *EditBuffer) DeleteWord() {
	if b.Cursor == 0 {
		return
	}
	end := b.Cursor
	for b.Cursor > 0 && b.Data[b.Cursor-1] == ' ' {
		b.Cursor--
	}
	for b.Cursor > 0 && b.Data[b.Cursor-1] != ' ' {
		b.Cursor--
	}
	b.Data = append(b.Data[:b.Cursor], b.Data[end:]...)
}

// KillLine deletes from cursor to end of line (Ctrl+K).
func (b *EditBuffer) KillLine() {
	b.Data = b.Data[:b.Cursor]
}

// KillBack deletes from cursor to start of line (Ctrl+U).
func (b *EditBuffer) KillBack() {
	b.Data = b.Data[b.Cursor:]
	b.Cursor = 0
}

// String returns the buffer contents as a string.
func (b *EditBuffer) String() string {
	return string(b.Data)
}

// Clear resets the buffer to empty.
func (b *EditBuffer) Clear() {
	b.Data = nil
	b.Cursor = 0
}

// Set replaces the buffer contents and moves the cursor to the end.
func (b *EditBuffer) Set(s string) {
	b.Data = []byte(s)
	b.Cursor = len(b.Data)
}

// HandleKey processes common text-editing keys on the buffer.
// Returns true if the key was handled.
func (b *EditBuffer) HandleKey(key Key) bool {
	switch {
	case key.Special == KeyBackspace:
		b.DeleteChar()
	case key.Special == KeyDelete:
		b.DeleteCharForward()
	case key.Special == KeyLeft:
		if b.Cursor > 0 {
			b.Cursor--
		}
	case key.Special == KeyRight:
		if b.Cursor < len(b.Data) {
			b.Cursor++
		}
	case key.Special == KeyWordLeft:
		b.WordLeft()
	case key.Special == KeyWordRight:
		b.WordRight()
	case key.Special == KeyDeleteWord:
		b.DeleteWord()
	case key.Special == KeyHome:
		b.Cursor = 0
	case key.Special == KeyEnd:
		b.Cursor = len(b.Data)
	case key.Special == KeyKillLine:
		b.KillLine()
	case key.Special == KeyKillBack:
		b.KillBack()
	case key.Char != 0:
		b.InsertChar(key.Char)
	default:
		return false
	}
	return true
}
