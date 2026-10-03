//go:build darwin

package tty

import (
	"syscall"
	"unsafe"
)

func init() {
	isatty = ioctlIsatty
	winsize = ioctlWidth
}

// ioctlIsatty asks the terminal driver for the descriptor's line settings. Only
// a descriptor a terminal driver owns has them, so a pipe, a regular file and
// /dev/null each come back with an errno here. This is the isatty question,
// which is why gdoc needs no library to ask it.
func ioctlIsatty(fd uintptr) bool {
	var settings syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGETA, uintptr(unsafe.Pointer(&settings)))
	return errno == 0
}

// ioctlWidth asks the same driver how wide its window is. The second field of
// the answer is the column count; the pixel sizes beside it are from a time
// when a terminal was a screen.
func ioctlWidth(fd uintptr) (int, bool) {
	var window struct {
		rows, cols, xpixel, ypixel uint16
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&window)))
	if errno != 0 {
		return 0, false
	}
	return int(window.cols), true
}
