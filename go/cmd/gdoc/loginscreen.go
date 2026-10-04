// The screen `gdoc auth login` draws on stderr for the person who typed it:
// the link in a box, and one line under it that waits.
//
// It sits beside helpscreen.go for the same reason: this is a room that knows
// what a command draws, and internal/tty and internal/panel are the rooms that
// know a colour and a box. So no colour name, no box-drawing character and no
// escape code is written here.
//
// The waiting line is one line and it rewrites itself with a carriage return,
// which is the whole of the drawing. `gdoc auth login` traps no signal, so a
// Ctrl-C can land anywhere in the three minutes the listener waits: a line
// that moved the cursor or turned auto-wrap off would leave a terminal half
// drawn, and the link above it rubbed out.
//
// A pipe, a file and a window with no room for a box get the two lines
// internal/auth has always printed, byte for byte, because a skill and a log
// read them: TestThePipeLoginLineIsTodays and
// TestANarrowWindowGetsTheLoginLineAndNoBox.

package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"

	"gdoc/internal/auth"
	"gdoc/internal/panel"
	"gdoc/internal/tty"
)

// The words the waiting line carries, and the ones it ends as. A failed wait
// ends it with the error itself, which is the same sentence the object on
// stdout carries.
const (
	loginWait = "waiting for the browser to come back"
	loginDone = "signed in"
)

// The marks the line can end with: the browser came back, or it did not.
const (
	doneMark   = "✓"
	failedMark = "✗"
)

// spinEvery is how often the waiting line turns. It is a variable so a
// recorded screen is one frame however long the machine takes between two
// writes, which is what oneFrame in the test hands it.
var spinEvery = spinInterval

// waitForBrowser hands the person the link and waits on the listener for the
// browser to come back. It is the CLI's half of a sign-in: internal/auth holds
// the trip, and the stream belongs to this package.
//
// The link goes out before the wait starts, always, because the wait is three
// minutes of somebody opening a browser and the link is what they need to open
// it with.
func waitForBrowser(ctx context.Context, errOut io.Writer, trip loginTrip, link string) error {
	p, style, ok := screenAt(errOut, false)
	if !ok {
		fmt.Fprint(errOut, auth.LinkLine(link))
		return trip.Wait(ctx)
	}
	writeLines(errOut, loginScreen(p, style, link))
	line := newSpinLine(errOut, style, loginWait)
	line.start()
	if err := trip.Wait(ctx); err != nil {
		// The reason is the error's own words, never shortened: the object on
		// stdout carries the same sentence, and a person comparing the two
		// must read one thing.
		line.end(style.Fail(failedMark), err.Error())
		return err
	}
	line.end(style.OK(doneMark), loginDone)
	return nil
}

// loginScreen is the link as a person reads it: the sentence in a box titled
// with the command and labelled with the host the link goes to, and the link
// itself flush left under the box.
//
// The link is outside the border and on one line of its own, which the
// terminal wraps itself, so a copy or a click gets all of it. That is what the
// example under `gdoc help <command>` does, and for the same reason.
func loginScreen(p panel.Panel, style tty.Style, link string) []string {
	out := p.Box("auth login", linkHost(link), lineRows(p, auth.LinkAsk))
	return append(out, style.Title(link))
}

// linkHost is the host at the right end of the top border, which says where
// the link is about to take the reader. A link that does not parse carries no
// label rather than a wrong one.
func linkHost(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return u.Host
}

// spinLine is one line a person reads while something is waited on: a spinner
// and some words, redrawn in place with a carriage return, and ending as a
// mark and the words a reader is left with. The rest of the line is wiped on
// every write, so a short ending leaves nothing of a long wait behind it.
//
// Its methods are safe to call while the spinner turns on its own goroutine,
// which is the whole reason it is a type and not two prints.
type spinLine struct {
	out   io.Writer
	style tty.Style
	words string

	mu       sync.Mutex
	frame    int
	started  bool
	stop     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
}

func newSpinLine(w io.Writer, style tty.Style, words string) *spinLine {
	return &spinLine{
		out:     w,
		style:   style,
		words:   words,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// start draws the first frame at once, so the line is on the screen before the
// first tick, and turns the spinner on its own goroutine.
func (s *spinLine) start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started = true
	s.draw()
	go s.spin()
}

// end stops the spinner and leaves the line as mark and words, with a newline
// after it: whatever comes next starts on a line of its own.
func (s *spinLine) end(mark, words string) {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		started := s.started
		s.mu.Unlock()
		close(s.stop)
		if started {
			<-s.stopped
		}
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	io.WriteString(s.out, spinFrame(mark, words, true))
}

func (s *spinLine) spin() {
	defer close(s.stopped)
	tick := time.NewTicker(spinEvery)
	defer tick.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-tick.C:
			s.mu.Lock()
			s.frame++
			s.draw()
			s.mu.Unlock()
		}
	}
}

// draw is the turning line. The caller holds the lock.
func (s *spinLine) draw() {
	mark := s.style.Title(spinnerFrames[s.frame%len(spinnerFrames)])
	io.WriteString(s.out, spinFrame(mark, s.style.Dim(s.words), false))
}

// spinFrame is one write of the line: back to its start, the mark, the words,
// and the rest of the line wiped. The last frame ends with a newline, and no
// frame moves the cursor anywhere but the start of its own line:
// TestTheLoginSpinnerMovesNoCursorButCarriageReturn.
func spinFrame(mark, words string, last bool) string {
	line := "\r" + mark + " " + words + tty.ClearLine
	if last {
		return line + "\n"
	}
	return line
}
