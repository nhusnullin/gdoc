// The step list `gdoc update` draws on stderr for the person who typed it.
//
// It lives here beside update.go rather than in internal/emit, because update
// is the one command that wants it. It moves when a second command does.
//
// Two shapes, chosen by what stderr is. On a terminal the whole list is drawn
// in advance and redrawn in place: a pending step dim, the running one with a
// spinner, a done one with a green tick, a failed one with a red cross and its
// reason on the next line. Anywhere else, which is every run a skill starts,
// each step prints once as a plain line when it ends, with no colour and no
// escape code. NO_COLOR turns the colour off on a terminal as well. Nothing
// here touches stdout, which carries the one object and nothing else.
//
// Whether stderr is a terminal, how much colour it takes, and every escape
// code written below come from internal/tty, which is the one room that holds
// them. This file knows a step list and no colour name.

package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"gdoc/internal/tty"
)

// nameWidth is how wide a step name is padded, wide enough for the longest
// one with room after it. The detail starts one space later.
const nameWidth = 20

// liveWidth is where a redrawn line is cut. A line that wraps is two rows on
// the screen and one in the count, and the next redraw would then move the
// cursor up one row too few. Plain lines are never cut.
const liveWidth = 78

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

// isTerminal is internal/tty's question, through one variable so a test can
// answer it per writer. tty.IsTerminal is the only definition of a terminal in
// the tree: it is the terminal driver's answer and not the file's mode, which
// is why /dev/null is not one. TestOnlyATerminalDriverMakesATerminal holds
// that this variable is the whole of the decision made here.
var isTerminal = tty.IsTerminal

// newProgress is the list for w, live when w is a terminal, coloured as much
// as the environment says that terminal takes.
func newProgress(w io.Writer, title string) *progress {
	if isTerminal(w) {
		return newLiveProgress(w, title, tty.NewStyle(tty.Colour(os.Getenv)))
	}
	p := &progress{out: w}
	fmt.Fprintln(w, title)
	return p
}

// newLiveProgress is the redrawn list, whatever w is. A test hands it a
// buffer; newProgress hands it a terminal. The zero Style writes no escape
// byte, so a caller that wants the redraw without the colour hands that one.
func newLiveProgress(w io.Writer, title string, style tty.Style) *progress {
	p := &progress{out: w, live: true, style: style}
	fmt.Fprintln(w, title)
	return p
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
// and writes the one-line result under it. An empty line writes none.
func (p *progress) Finish(line string) {
	p.settle()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished {
		return
	}
	p.finished = true
	if line != "" {
		fmt.Fprintln(p.out, line)
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

// draw moves the cursor back over what was drawn before and writes the list.
// The last draw leaves out the steps that never ran and adds the reason under
// a failed one; the reason is never in a moving draw, because it may wrap.
// The first draw turns auto-wrap off and the last turns it back on before its
// lines, so the reason wraps as ordinary text once nothing moves any more.
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
	n := 0
	for _, s := range p.steps {
		if last && s.state == stepPending {
			continue
		}
		for _, l := range p.stepLines(s, last) {
			b.WriteString(l + "\n")
			n++
		}
	}
	p.drawn = n
	io.WriteString(p.out, b.String())
}

// stepLines is one step as text: its line, and under a failed one the reason
// when withReason says so.
func (p *progress) stepLines(s step, withReason bool) []string {
	text := strings.TrimRight(fmt.Sprintf("%-*s %s", nameWidth, s.name, s.detail), " ")
	if p.live {
		text = cut(text, liveWidth-4)
	}
	if s.state == stepPending {
		text = p.style.Dim(text)
	}
	line := "  " + p.mark(s.state) + " " + text
	if s.state != stepFailed || !withReason {
		return []string{line}
	}
	return []string{line, "    " + s.reason}
}

// mark is the glyph in front of a step, in the role its state is: the spinner
// is a title, a tick an ok and a cross a failure. A plain list carries the zero
// Style, so each is the glyph on its own.
func (p *progress) mark(s stepState) string {
	switch s {
	case stepRunning:
		return p.style.Title(spinnerFrames[p.frame%len(spinnerFrames)])
	case stepDone:
		return p.style.OK("✓")
	case stepFailed:
		return p.style.Fail("✗")
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
