//go:build unix

package main

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// lockIsFree says whether nobody holds the app's lock on the file at path.
//
// The app holds flock(LOCK_EX) on that file for its whole life, so this is the
// one question that says whether it is running, and the kernel answers it:
// a process that died, however it died, holds nothing. gdoc asks for the same
// lock without blocking and lets go at once.
//
// Two things it never does. It creates nothing: a file that is not there is an
// app that never ran, which is free, and gdoc writing into the presence
// directory here would make it the writer of a file it is only asking about.
// And it reads no byte, because the record is the one file gdoc appends to and
// never reads, and the lock is the kernel's rather than the file's.
//
// Anything else that goes wrong is not free. A file gdoc cannot open says
// nothing about who is running, and not knowing never resolves to starting
// another process: TestTheOpenerAsksThreeThingsAndRunsOpenGOnce.
func lockIsFree(path string) bool {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	if err != nil {
		return false
	}
	defer f.Close()
	fd := int(f.Fd())
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return false
	}
	// Let go at once. The close below would do it too, and both are here
	// because the descriptor is the lock: a gdoc that held it for the rest of
	// its run would be a gdoc the app could not start under.
	_ = syscall.Flock(fd, syscall.LOCK_UN)
	return true
}
