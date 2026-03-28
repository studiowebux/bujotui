//go:build linux

package term

import (
	"fmt"
	"syscall"
	"unsafe"
)

// termios matches the Linux struct termios layout.
type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Line   byte
	Cc     [32]byte
	Ispeed uint32
	Ospeed uint32
}

const (
	tcgets = 0x5401
	tcsets = 0x5402
)

const (
	icanon = 0x00000002
	echo   = 0x00000008
	isig   = 0x00000001
	icrnl  = 0x00000100
	ixon   = 0x00000400
	istrip = 0x00000020
	opost  = 0x00000001
)

// EnableRawMode puts the terminal into raw mode and returns a restore function.
func EnableRawMode(fd uintptr) (restore func() error, err error) {
	var orig termios
	if err := ioctl(fd, tcgets, &orig); err != nil {
		return nil, fmt.Errorf("tcgetattr: %w", err)
	}

	raw := orig
	raw.Iflag &^= icrnl | ixon | istrip
	raw.Oflag &^= opost
	raw.Lflag &^= icanon | echo | isig
	raw.Cc[6] = 1 // VMIN
	raw.Cc[5] = 0 // VTIME

	if err := ioctl(fd, tcsets, &raw); err != nil {
		return nil, fmt.Errorf("tcsetattr: %w", err)
	}

	return func() error {
		if err := ioctl(fd, tcsets, &orig); err != nil {
			return fmt.Errorf("tcsetattr restore: %w", err)
		}
		return nil
	}, nil
}

func ioctl(fd uintptr, request uint, argp *termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(request), uintptr(unsafe.Pointer(argp)))
	if errno != 0 {
		return errno
	}
	return nil
}

// GetSize returns the terminal dimensions.
func GetSize(fd uintptr) (rows, cols int, err error) {
	var ws winsize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(0x5413), uintptr(unsafe.Pointer(&ws))) // TIOCGWINSZ
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
