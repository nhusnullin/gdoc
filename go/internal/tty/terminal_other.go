//go:build !darwin

package tty

func init() {
	isatty = neverATerminal
	winsize = neverMeasured
}

// neverATerminal is every platform but darwin. Nail's call of 2026-10-03: the
// screens are drawn on his machine, Windows is out of scope, and a platform
// gdoc cannot measure gets today's plain text and the JSON object, which is the
// safe answer rather than the pretty one.
func neverATerminal(uintptr) bool { return false }

// neverMeasured is the same choice for the width. Nothing is measured here, so
// COLUMNS or eighty answers, and the plain text does not care either way.
func neverMeasured(uintptr) (int, bool) { return 0, false }
