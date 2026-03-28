package tui

// Key represents a parsed key input.
type Key struct {
	Char    byte // regular character (0 if special key)
	Rune    rune // for multi-byte chars
	Special SpecialKey
}

type SpecialKey int

const (
	KeyNone SpecialKey = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEscape
	KeyBackspace
	KeyDelete
	KeyTab
	KeyHome
	KeyEnd
	KeyWordLeft   // Option+Left / Alt+b
	KeyWordRight  // Option+Right / Alt+f
	KeyDeleteWord // Option+Backspace
	KeyKillLine   // Ctrl+K — delete to end of line
	KeyKillBack   // Ctrl+U — delete to start of line
	KeyShiftTab   // Shift+Tab
)

// ParseKey reads a key from the input buffer.
func ParseKey(buf []byte, n int) Key {
	if n == 0 {
		return Key{}
	}

	// Escape sequences
	if buf[0] == 0x1b {
		if n == 1 {
			return Key{Special: KeyEscape}
		}
		if n >= 3 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return Key{Special: KeyUp}
			case 'B':
				return Key{Special: KeyDown}
			case 'C':
				return Key{Special: KeyRight}
			case 'D':
				return Key{Special: KeyLeft}
			case 'H':
				return Key{Special: KeyHome}
			case 'F':
				return Key{Special: KeyEnd}
			case 'Z':
				return Key{Special: KeyShiftTab}
			case '3':
				if n >= 4 && buf[3] == '~' {
					return Key{Special: KeyDelete}
				}
			case '1':
				// Modified arrow keys: \x1b[1;3C = Option+Right, \x1b[1;3D = Option+Left
				if n >= 6 && buf[3] == ';' {
					switch buf[5] {
					case 'C':
						if buf[4] == '3' {
							return Key{Special: KeyWordRight}
						}
						return Key{Special: KeyRight}
					case 'D':
						if buf[4] == '3' {
							return Key{Special: KeyWordLeft}
						}
						return Key{Special: KeyLeft}
					case 'A':
						return Key{Special: KeyUp}
					case 'B':
						return Key{Special: KeyDown}
					}
				}
			}
		}
		// Alt+b = word left, Alt+f = word right
		if n == 2 && buf[1] == 'b' {
			return Key{Special: KeyWordLeft}
		}
		if n == 2 && buf[1] == 'f' {
			return Key{Special: KeyWordRight}
		}
		// Alt+Backspace = delete word (0x1b 0x7f)
		if n == 2 && buf[1] == 0x7f {
			return Key{Special: KeyDeleteWord}
		}
		// Ignore unrecognized escape sequences
		if n > 1 {
			return Key{}
		}
		return Key{Special: KeyEscape}
	}

	switch buf[0] {
	case 13, 10: // CR, LF
		return Key{Special: KeyEnter}
	case 127, 8: // DEL, BS
		return Key{Special: KeyBackspace}
	case 9:
		return Key{Special: KeyTab}
	case 1: // Ctrl+A
		return Key{Special: KeyHome}
	case 5: // Ctrl+E
		return Key{Special: KeyEnd}
	case 11: // Ctrl+K
		return Key{Special: KeyKillLine}
	case 21: // Ctrl+U
		return Key{Special: KeyKillBack}
	case 23: // Ctrl+W — same as Option+Backspace
		return Key{Special: KeyDeleteWord}
	default:
		return Key{Char: buf[0]}
	}
}
