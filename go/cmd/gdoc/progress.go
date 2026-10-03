// The step list `gdoc update` draws on stderr for the person who typed it.
//
// It lives here beside update.go rather than in internal/emit, because update
// is the one command that wants it. It moves when a second command does.
//
// Two shapes, chosen by what stderr is. On a terminal the whole list is drawn
// in advance inside a box and redrawn in place: a pending step a dim ring, the
// running one a spinner, a done one a green tick, a failed one a red cross
// with its reason in the cell its detail was in. Anywhere else, which is every
// run a skill starts, each step prints once as a plain line when it ends, with
// no colour and no escape code. NO_COLOR turns the colour off on a terminal as
// well. Nothing here touches stdout, which carries the one object and nothing
// else.
//
// Whether stderr is a terminal, how much colour it takes, and every escape
// code written below come from internal/tty, which is the one room that holds
// them. The box is internal/panel's. This file knows a step list, and no
// colour name and no box-drawing character.

package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"gdoc/internal/panel"
	"gdoc/internal/tty"
)

// nameWidth is how wide a step name is padded, wide enough for the longest
// one with room after it. The detail starts one space later.
const nameWidth = 20

// boxColumn is where the detail cell starts in the drawn box: the mark and
// the blank after it, a name as wide as a plain line pads one, and the cells
// a panel column costs. It is a fixed number rather than a measured one,
// because a column that moved when a later step was planned would move the
// whole list under the reader's eye.
const boxColumn = 2 + nameWidth + cellPadding

// pendingMark is the step nothing has reached yet, on a terminal. A plain
// line is written when its step ends, so nothing there is ever pending.
const pendingMark = "○"

// spinInterval is how often the spinner turns while a step runs.
const spinInterval = 80 * time.Millisecond

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type stepState int

const (
	stepPending stepState = iota
	stepRunning
	stepDone
	stepFailed
)

type step struct {
	name   string
	detail string
	state  stepState
	reason string
}

// progress is one run's step list. Its methods are safe to call from the
// command while the spinner turns on its own goroutine.
type progress struct {
	mu    sync.Mutex
	out   io.Writer
	live  bool
	style tty.Style
	// name is the command the list is for, which titles the box a terminal
	// gets and names the command line a pipe opens with. box is the panel the
	// live list is drawn in, and label the words at the right end of its top
	// border.
	name  string
	box   panel.Panel
	label string
	steps []step
	drawn int
	frame int
	// settled is set once the list is drawn for the last time; finished once
	// the result line is written.
	settled  bool
	finished bool
	// unwrapped is set once auto-wrap is turned off, so the last draw knows
	// to turn it back on.
	unwrapped bool
	stop      chan struct{}
	stopped   chan struct{}
	stopOnce  sync.Once
}

// newProgress is the list for the command called name on w, live when w is a
// terminal, coloured as much as the environment says that terminal takes.
//
// A plain list opens with the command line a person would have typed, which
// is what it always opened with. A live one opens with nothing: its box is
// titled with the name instead, and the box is drawn on the first Plan.
func newProgress(w io.Writer, name string) *progress {
	if isTerminal(w) {
		return newLiveProgress(w, name, tty.NewStyle(tty.Colour(os.Getenv)))
	}
	p := &progress{out: w, name: name}
	fmt.Fprintln(w, "gdoc "+name)
	return p
}

// newLiveProgress is the redrawn list, whatever w is. A test hands it a
// buffer; newProgress hands it a terminal. The zero Style writes no escape
// byte, so a caller that wants the redraw without the colour hands that one.
//
// The box is as wide as the window, which for a buffer is what COLUMNS says
// and then eighty, so a recorded screen is a width a test named.
func newLiveProgress(w io.Writer, name string, style tty.Style) *progress {
	return &progress{
		out:   w,
		live:  true,
		style: style,
		name:  name,
		box:   panel.New(style, tty.Width(w, os.Getenv)).WithColumn(boxColumn),
	}
}

// Label is the words at the right end of the top border: the gdoc that is
// running, and the one this run is taking once it has chosen one. A plain
// list draws no border, so it draws no label either.
func (p *progress) Label(s string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.label = s
	p.redraw()
}

// Plan names steps in advance, so a terminal shows the whole path before the
// run is on it. A step that does not apply is never planned, and so never
// drawn.
func (p *progress) Plan(names ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, n := range names {
		p.steps = append(p.steps, step{name: n})
	}
	p.redraw()
}

// Start marks the named step as running, with what is known about it so far.
// A name nobody planned is added at the end.
func (p *progress) Start(name, detail string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	i := p.index(name)
	if i < 0 {
		p.steps = append(p.steps, step{name: name})
		i = len(p.steps) - 1
	}
	p.steps[i] = step{name: name, detail: detail, state: stepRunning}
	if p.live && p.stop == nil {
		p.stop, p.stopped = make(chan struct{}), make(chan struct{})
		go p.spin()
	}
	p.redraw()
}

// Done marks the running step done. An empty detail keeps what Start said.
func (p *progress) Done(detail string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	i := p.running()
	if i < 0 {
		return
	}
	s := p.steps[i]
	if detail != "" {
		s.detail = detail
	}
	s.state = stepDone
	p.steps[i] = s
	p.ended(s)
}

// Fail marks the running step failed with err as its reason. Nothing runs
// after a failure, so the list is settled here and not redrawn again.
func (p *progress) Fail(err error) {
	p.mu.Lock()
	i := p.running()
	if i >= 0 {
		s := p.steps[i]
		s.state, s.reason = stepFailed, err.Error()
		p.steps[i] = s
		p.ended(s)
	}
	p.mu.Unlock()
	p.settle()
}

// Do is Start, run, and then Done with the detail run answers or Fail with
// its error, which it hands back.
func (p *progress) Do(name, detail string, run func() (string, error)) error {
	p.Start(name, detail)
	done, err := run()
	if err != nil {
		p.Fail(err)
		return err
	}
	p.Done(done)
	return nil
}

// Finish draws the list for the last time, without the steps that never ran,
// and writes the lines a person reads under it. An empty line writes none,
// and no line at all is what a run that failed gives: its cross and its
// reason are already the last thing on the screen.
//
// On a terminal a line wider than the box is wrapped into lines that fit it,
// so what is under the box is as square as the box. On a pipe a line is a
// line, which is what a log and a skill have always read.
func (p *progress) Finish(lines ...string) {
	p.settle()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished {
		return
	}
	p.finished = true
	for _, line := range lines {
		if line == "" {
			continue
		}
		if !p.live {
			fmt.Fprintln(p.out, line)
			continue
		}
		for _, l := range panel.Wrap(line, p.box.Width()) {
			fmt.Fprintln(p.out, l)
		}
	}
}

// settle stops the spinner and draws the list as it ended. It is done once,
// and the spinner is stopped outside the lock, because the spinner takes it.
func (p *progress) settle() {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		stop, stopped := p.stop, p.stopped
		p.mu.Unlock()
		if stop != nil {
			close(stop)
			<-stopped
		}
	})
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.settled {
		return
	}
	p.settled = true
	if p.live {
		p.draw(true)
	}
}

// ended is the plain line for a step that just finished; a live list redraws
// instead. The caller holds the lock.
func (p *progress) ended(s step) {
	if p.live {
		p.redraw()
		return
	}
	for _, l := range p.stepLines(s, true) {
		fmt.Fprintln(p.out, l)
	}
}

func (p *progress) spin() {
	defer close(p.stopped)
	tick := time.NewTicker(spinInterval)
	defer tick.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-tick.C:
			p.mu.Lock()
			p.frame++
			p.redraw()
			p.mu.Unlock()
		}
	}
}

// redraw is a live list drawn again while it is still moving. The caller holds
// the lock.
func (p *progress) redraw() {
	if p.live && !p.settled {
		p.draw(false)
	}
}

// draw moves the cursor back over what was drawn before and writes the box.
// The last draw leaves out the steps that never ran. Every line it writes is
// one row on the screen, because the box wrapped them itself, so the count it
// moves back over next time is the count of the lines it wrote.
//
// The first draw turns auto-wrap off and the last turns it back on, so a
// window narrower than the box cannot turn one row into two while the list is
// still moving.
func (p *progress) draw(last bool) {
	var b strings.Builder
	if !last && !p.unwrapped {
		b.WriteString(tty.WrapOff)
		p.unwrapped = true
	}
	if p.drawn > 0 {
		b.WriteString(tty.Up(p.drawn) + "\r")
	}
	if last && p.unwrapped {
		b.WriteString(tty.WrapOn)
	}
	b.WriteString(tty.ClearBelow)
	lines := p.boxLines(last)
	for _, l := range lines {
		b.WriteString(l + "\n")
	}
	p.drawn = len(lines)
	io.WriteString(p.out, b.String())
}

// boxLines is the list as a box: the command in the top border, the versions
// at its right end, and a row for each step with its marked name in the left
// cell and what is known about it in the right one. A failed step carries its
// reason there instead of its detail, where the box wraps it. The last draw
// leaves out the steps that never ran.
func (p *progress) boxLines(last bool) []string {
	rows := make([]panel.Row, 0, len(p.steps))
	for _, s := range p.steps {
		if last && s.state == stepPending {
			continue
		}
		name := s.name
		if s.state == stepPending {
			name = p.style.Dim(name)
		}
		detail := s.detail
		if s.state == stepFailed {
			detail = s.reason
		}
		rows = append(rows, panel.Pair(p.mark(s.state)+" "+name, detail))
	}
	return p.box.Box(p.name, p.label, rows)
}

// stepLines is one step as the plain lines a pipe gets: its line, and under a
// failed one the reason when withReason says so. A terminal gets boxLines
// instead, so nothing here is ever cut or coloured.
func (p *progress) stepLines(s step, withReason bool) []string {
	text := strings.TrimRight(fmt.Sprintf("%-*s %s", nameWidth, s.name, s.detail), " ")
	line := "  " + p.mark(s.state) + " " + text
	if s.state != stepFailed || !withReason {
		return []string{line}
	}
	return []string{line, "    " + s.reason}
}

// mark is the glyph in front of a step, in the role its state is: the spinner
// is a title, a tick an ok, a cross a failure and a step nothing has reached
// yet a muted ring. A plain line is written when its step ends, so there is
// never a pending step there and the fourth mark is the blank it always was.
// A plain list carries the zero Style, so each glyph is itself alone.
func (p *progress) mark(s stepState) string {
	switch s {
	case stepRunning:
		return p.style.Title(spinnerFrames[p.frame%len(spinnerFrames)])
	case stepDone:
		return p.style.OK("✓")
	case stepFailed:
		return p.style.Fail("✗")
	}
	if p.live {
		return p.style.Dim(pendingMark)
	}
	return " "
}

func (p *progress) index(name string) int {
	for i, s := range p.steps {
		if s.name == name {
			return i
		}
	}
	return -1
}

func (p *progress) running() int {
	for i, s := range p.steps {
		if s.state == stepRunning {
			return i
		}
	}
	return -1
}

// cut shortens s to n characters, marking the cut.
func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// humanSize is a byte count as a person reads it, and empty for a count the
// listing did not give.
func humanSize(n int64) string {
	switch {
	case n <= 0:
		return ""
	case n < 1_000_000:
		return fmt.Sprintf("%d KB", (n+999)/1000)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/1_000_000)
}
