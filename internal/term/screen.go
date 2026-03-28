package term

import (
	"fmt"
	"io"
)

// ANSI escape sequences for terminal control.

func ClearScreen(w io.Writer) {
	fmt.Fprint(w, "\x1b[2J")
}

func MoveCursor(w io.Writer, row, col int) {
	fmt.Fprintf(w, "\x1b[%d;%dH", row, col)
}

func HideCursor(w io.Writer) {
	fmt.Fprint(w, "\x1b[?25l")
}

func ShowCursor(w io.Writer) {
	fmt.Fprint(w, "\x1b[?25h")
}

func ClearLine(w io.Writer) {
	fmt.Fprint(w, "\x1b[2K")
}

func ClearToEnd(w io.Writer) {
	fmt.Fprint(w, "\x1b[0K")
}

// Color codes
const (
	Reset     = "\x1b[0m"
	Bold      = "\x1b[1m"
	Dim       = "\x1b[2m"
	Underline = "\x1b[4m"
	Reverse   = "\x1b[7m"

	FgBlack   = "\x1b[30m"
	FgRed     = "\x1b[31m"
	FgGreen   = "\x1b[32m"
	FgYellow  = "\x1b[33m"
	FgBlue    = "\x1b[34m"
	FgMagenta = "\x1b[35m"
	FgCyan    = "\x1b[36m"
	FgWhite   = "\x1b[37m"
	FgGray    = "\x1b[90m"

	BgDarkGray    = "\x1b[48;5;236m"
	BgReset       = "\x1b[49m"
	BgHighlight   = "\x1b[48;5;238m" // subtle highlight for selected row
	FgBrightWhite = "\x1b[97m"       // bright white for high contrast text
)

// UseAlternateScreen switches to the alternate screen buffer.
func UseAlternateScreen(w io.Writer) {
	fmt.Fprint(w, "\x1b[?1049h")
}

// UseMainScreen switches back to the main screen buffer.
func UseMainScreen(w io.Writer) {
	fmt.Fprint(w, "\x1b[?1049l")
}

// EnableMouseTracking enables basic mouse click tracking.
func EnableMouseTracking(w io.Writer) {
	fmt.Fprint(w, "\x1b[?1000h")
}

// DisableMouseTracking disables mouse tracking.
func DisableMouseTracking(w io.Writer) {
	fmt.Fprint(w, "\x1b[?1000l")
}
