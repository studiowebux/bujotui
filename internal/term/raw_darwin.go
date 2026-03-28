//go:build darwin

package term

import (
	"fmt"
	"syscall"
	"unsafe"
)

// termios matches the Darwin struct termios layout.
type termios struct {
	Iflag  uint64
	Oflag  uint64
	Cflag  uint64
	Lflag  uint64
	Cc     [20]byte
	Ispeed uint64
	Ospeed uint64
}

const (
	tiocgeta = 0x40487413 // TIOCGETA on darwin/arm64 and darwin/amd64
	tiocseta = 0x80487414 // TIOCSETA
)

// Local mode flags
const (
	icanon = 0x00000100
	echo   = 0x00000008
	isig   = 0x00000080
)

// Input mode flags
const (
	icrnl  = 0x00000100
	ixon   = 0x00000200
	istrip = 0x00000020
)

// Output mode flags
const (
	opost = 0x00000001
)

// EnableRawMode puts the terminal into raw mode and returns a restore function.
func EnableRawMode(fd uintptr) (restore func() error, err error) {
	var orig termios
	if err := ioctl(fd, tiocgeta, &orig); err != nil {
		return nil, fmt.Errorf("tcgetattr: %w", err)
	}

	raw := orig
	raw.Iflag &^= icrnl | ixon | istrip
	raw.Oflag &^= opost
	raw.Lflag &^= icanon | echo | isig
	raw.Cc[6] = 1 // VMIN
	raw.Cc[5] = 0 // VTIME

	if err := ioctl(fd, tiocseta, &raw); err != nil {
		return nil, fmt.Errorf("tcsetattr: %w", err)
	}

	return func() error {
		if err := ioctl(fd, tiocseta, &orig); err != nil {
			return fmt.Errorf("tcsetattr restore: %w", err)
		}
		return nil
	}, nil
}

func ioctl(fd uintptr, request uint64, argp *termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(request), uintptr(unsafe.Pointer(argp)))
	if errno != 0 {
		return errno
	}
	return nil
}

// GetSize returns the terminal dimensions.
func GetSize(fd uintptr) (rows, cols int, err error) {
	var ws winsize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(0x40087468), uintptr(unsafe.Pointer(&ws))) // TIOCGWINSZ
	if errno != 0 {
		return 0, 0, errno
	}
	return int(ws.Row), int(ws.Col), nil
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}
