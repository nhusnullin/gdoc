package comments

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docs"
)

// answer is one scripted poll: what the fake Fetch hands back that time.
type answer struct {
	doc *docs.Document
	raw []RawComment
	err error
}

// script is a fake wait: a sequence of polls, a fake clock the sleeper drives,
// and the order the two happened in. Nothing here waits, so a test of a
// ten-second interval costs no wall time.
type script struct {
	t       *testing.T
	answers []answer
	polls   int
	events  []string
	clock   time.Time
	// onSleep runs before each sleep advances the clock, so a test can cancel
	// the context in the middle of a wait.
	onSleep func()
	// onPoll runs at the start of each poll, before its answer is taken, so a
	// test can cancel the context while a poll is in flight rather than before
	// the wait ever reaches one.
	onPoll func()
}

func newScript(t *testing.T, answers ...answer) *script {
	t.Helper()
	s := &script{t: t, answers: answers, clock: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)}

	realNow, realSleep := now, sleep
	now = func() time.Time { return s.clock }
	sleep = func(ctx context.Context, d time.Duration) {
		s.events = append(s.events, "sleep")
		if s.onSleep != nil {
			s.onSleep()
		}
		s.clock = s.clock.Add(d)
	}
	t.Cleanup(func() { now, sleep = realNow, realSleep })
	return s
}

// fetch is the Poll the wait calls. A script that runs out of answers fails the
// test rather than blocking: an extra poll is the defect the test is looking
// for.
func (s *script) fetch(ctx context.Context) (*docs.Document, []RawComment, error) {
	s.events = append(s.events, "poll")
	if s.onPoll != nil {
		s.onPoll()
	}
	if s.polls >= len(s.answers) {
		s.t.Fatalf("the wait polled %d times, and the script holds %d answers", s.polls+1, len(s.answers))
	}
	a := s.answers[s.polls]
	s.polls++
	// A poll takes a moment, the way a Docs read and a Drive listing do.
	s.clock = s.clock.Add(100 * time.Millisecond)
	return a.doc, a.raw, a.err
}

func (s *script) options(interval, deadline time.Duration) WaitOptions {
	return WaitOptions{Interval: interval, Deadline: deadline, Fetch: s.fetch}
}

// quiet is a poll that found nothing.
func quiet() answer { return answer{doc: &docs.Document{}} }

// oneComment is a poll that found one comment, at the given instant.
func oneComment(id, content, modified string) answer {
	return answer{
		doc: &docs.Document{},
		raw: []RawComment{{
			ID:           id,
			Author:       Author{DisplayName: "Nail Khusnullin"},
			CreatedTime:  modified,
			ModifiedTime: modified,
			Content:      content,
		}},
	}
}

func TestWaitPollsAtOnceRatherThanSleepingFirst(t *testing.T) {
	// Latency is the point of a live session. A wait that sleeps before its
	// first look costs the interval on every call the skill makes.
	s := newScript(t, oneComment("AAAA1111", "ai? which register", "2026-09-07T12:00:05Z"))

	got, err := Wait(context.Background(), nil, s.options(10*time.Second, time.Minute))

	if err != nil {
		t.Fatal(err)
	}
	if len(s.events) == 0 || s.events[0] != "poll" {
		t.Errorf("events = %v, want the first one to be a poll", s.events)
	}
	if got.Polls != 1 {
		t.Errorf("Polls = %d, want 1", got.Polls)
	}
	if len(got.Threads) != 1 || got.Threads[0].ID != "AAAA1111" {
		t.Errorf("Threads = %+v, want the one comment", got.Threads)
	}
}

func TestWaitReturnsTheFirstWindowWithActivityAndAdvancesTheCursor(t *testing.T) {
	since := &Cursor{At: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)}
	s := newScript(t,
		quiet(),
		oneComment("BBBB2222", "ai! add it to the register", "2026-09-07T12:00:20.250Z"),
	)

	got, err := Wait(context.Background(), since, s.options(10*time.Second, time.Minute))

	if err != nil {
		t.Fatal(err)
	}
	if got.Polls != 2 {
		t.Errorf("Polls = %d, want 2", got.Polls)
	}
	if len(got.Threads) != 1 || got.Threads[0].Marker != "ai!" {
		t.Fatalf("Threads = %+v, want the one marked comment", got.Threads)
	}
	if got.Interrupted {
		t.Error("a wait that ended on news came back interrupted")
	}
	want := time.Date(2026, 9, 7, 12, 0, 20, 250000000, time.UTC)
	if got.Cursor == nil || !got.Cursor.At.Equal(want) {
		t.Errorf("cursor = %v, want the thread's own instant %v", got.Cursor, want)
	}
	if got.Waited <= 0 {
		t.Errorf("Waited = %v, want the time the polls and the sleep took", got.Waited)
	}
	if want := []string{"poll", "sleep", "poll"}; !reflect.DeepEqual(s.events, want) {
		t.Errorf("events = %v, want %v", s.events, want)
	}
}

func TestWaitCarriesAFailedPollOutWithThePollsSoFar(t *testing.T) {
	// A read that failed is not retried in silence. The skill sees it, says so
	// and calls again; a binary that swallowed it would report a quiet document
	// on a document it could not read.
	refused := errors.New("the comment listing was refused: file was not given to this command")
	s := newScript(t, quiet(), answer{err: refused})

	got, err := Wait(context.Background(), nil, s.options(10*time.Second, time.Minute))

	if !errors.Is(err, refused) {
		t.Fatalf("err = %v, want the poll's own error", err)
	}
	if got.Polls != 2 {
		t.Errorf("Polls = %d, want the two polls the wait made", got.Polls)
	}
	if len(got.Threads) != 0 {
		t.Errorf("Threads = %+v, want none on a failed poll", got.Threads)
	}
}

func TestWaitAtItsDeadlineIsEmptyWithTheCursorItWasGiven(t *testing.T) {
	since := &Cursor{At: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC), Ids: []string{"AAAA1111"}}
	s := newScript(t, quiet(), quiet(), quiet(), quiet())

	got, err := Wait(context.Background(), since, s.options(10*time.Second, 30*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if len(got.Threads) != 0 {
		t.Errorf("Threads = %+v, want none at the deadline", got.Threads)
	}
	if got.Threads == nil {
		t.Error("an empty window came back as a null thread list, and a caller has to special-case it")
	}
	if got.Cursor != since {
		t.Errorf("cursor = %v, want the one handed in unchanged", got.Cursor)
	}
	if got.Interrupted {
		t.Error("a wait that reached its deadline came back interrupted")
	}
	if got.Polls < 2 {
		t.Errorf("Polls = %d, want more than one inside a 30 second deadline at a 10 second interval", got.Polls)
	}
	if got.Waited > 30*time.Second+time.Second {
		t.Errorf("Waited = %v, want no more than the deadline plus one poll", got.Waited)
	}
}

func TestWaitDoesNotSleepPastItsDeadline(t *testing.T) {
	// The sleep is clamped to what is left, so a nine minute deadline does not
	// answer at nine minutes and ten seconds.
	s := newScript(t, quiet(), quiet())

	got, err := Wait(context.Background(), nil, s.options(10*time.Second, 5*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if got.Waited > 6*time.Second {
		t.Errorf("Waited = %v on a 5 second deadline", got.Waited)
	}
}

func TestAnInterruptEndsTheWaitAsAnAnswer(t *testing.T) {
	// Ctrl-C is how every live session ends. It is an answer with no threads
	// and the cursor handed in, never a crash and never a lost cursor.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	since := &Cursor{At: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)}
	s := newScript(t, quiet(), quiet())
	s.onSleep = cancel

	got, err := Wait(ctx, since, s.options(10*time.Second, time.Hour))

	if err != nil {
		t.Fatalf("an interrupt came back as an error: %v", err)
	}
	if !got.Interrupted {
		t.Error("Interrupted = false after the context was cancelled")
	}
	if s.polls != 1 {
		t.Errorf("the wait polled %d times, want 1: a cancelled context polls no further", s.polls)
	}
	if len(got.Threads) != 0 {
		t.Errorf("Threads = %+v, want none", got.Threads)
	}
	if got.Cursor != since {
		t.Errorf("cursor = %v, want the one handed in unchanged", got.Cursor)
	}
}

func TestAContextAlreadyCancelledPollsNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := newScript(t)

	got, err := Wait(ctx, nil, s.options(10*time.Second, time.Minute))

	if err != nil {
		t.Fatal(err)
	}
	if !got.Interrupted || got.Polls != 0 {
		t.Errorf("Interrupted = %v, Polls = %d, want true and 0", got.Interrupted, got.Polls)
	}
}

func TestAPollThatFailedBecauseOfTheInterruptIsReportedAsTheInterrupt(t *testing.T) {
	// A real Fetch takes the context, so the signal that ends the session ends
	// the request in flight. Blaming Drive for the person's Ctrl-C would send
	// somebody to look at a read that was fine.
	//
	// The cancel happens inside the poll, not before the wait. Cancelled first,
	// the loop's own pre-poll check answers and this test becomes a second copy
	// of TestAContextAlreadyCancelledPollsNothing, guarding a branch it never
	// reaches. Polls is what says the poll really ran.
	ctx, cancel := context.WithCancel(context.Background())
	s := newScript(t, answer{err: context.Canceled})
	s.onPoll = cancel

	got, err := Wait(ctx, nil, s.options(10*time.Second, time.Minute))

	if err != nil {
		t.Fatalf("err = %v, want the interrupt reported as an answer", err)
	}
	if !got.Interrupted {
		t.Error("Interrupted = false on a poll the interrupt cut short")
	}
	if got.Polls != 1 {
		t.Errorf("Polls = %d, want 1: the poll the interrupt cut short must have run", got.Polls)
	}
}

func TestAWindowThatIsOnlyAGdocReplyIsStillReturned(t *testing.T) {
	// Facts only. gdoc's own receipt landing after a poll is activity, so the
	// wait returns it; whether it is work is the skill's to read.
	s := newScript(t, answer{
		doc: &docs.Document{},
		raw: []RawComment{{
			ID:           "CCCC3333",
			Author:       Author{DisplayName: "Nail Khusnullin"},
			CreatedTime:  "2026-09-07T11:59:00Z",
			ModifiedTime: "2026-09-07T12:00:10Z",
			Content:      "ai? which register",
			Replies: []RawReply{{
				ID:          "r1",
				Author:      Author{DisplayName: "Nail Khusnullin"},
				CreatedTime: "2026-09-07T12:00:10Z",
				Content:     "🤖 The operations register.",
			}},
		}},
	})

	got, err := Wait(context.Background(), nil, s.options(10*time.Second, time.Minute))

	if err != nil {
		t.Fatal(err)
	}
	if len(got.Threads) != 1 {
		t.Fatalf("Threads = %+v, want the receipt's thread", got.Threads)
	}
	if len(got.Threads[0].Replies) != 1 || !got.Threads[0].Replies[0].ByGdoc {
		t.Errorf("replies = %+v, want the one 🤖 reply reported as gdoc's", got.Threads[0].Replies)
	}
}

func TestWaitCarriesTheIdsTheDocsReadCouldNotPlace(t *testing.T) {
	s := newScript(t, answer{
		doc: &docs.Document{
			Tabs:          []docs.Tab{tabCovering("t.0", 1, 20)},
			CommentRanges: map[string]docs.Range{"BACKWARDS": {Tab: "t.0", Start: 9, End: 3}},
		},
		raw: []RawComment{{ID: "BACKWARDS", ModifiedTime: "2026-09-07T12:00:05Z", Content: "ai? here"}},
	})

	got, err := Wait(context.Background(), nil, s.options(10*time.Second, time.Minute))

	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"BACKWARDS"}; !reflect.DeepEqual(got.Unplaced, want) {
		t.Errorf("Unplaced = %v, want %v", got.Unplaced, want)
	}
	if len(got.Threads) != 1 || got.Threads[0].Range != nil {
		t.Errorf("Threads = %+v, want the thread with no range", got.Threads)
	}
}

func TestWaitRefusesOptionsItCannotRun(t *testing.T) {
	// A zero interval is a spin, and no Fetch is a wait that can never look.
	// Both are the caller's mistake, and both are refused naming the field
	// rather than run.
	cases := []struct {
		name string
		o    WaitOptions
		want string
	}{
		{"no interval", WaitOptions{Deadline: time.Minute, Fetch: func(context.Context) (*docs.Document, []RawComment, error) { return nil, nil, nil }}, "interval"},
		{"negative interval", WaitOptions{Interval: -time.Second, Deadline: time.Minute, Fetch: func(context.Context) (*docs.Document, []RawComment, error) { return nil, nil, nil }}, "interval"},
		{"no fetch", WaitOptions{Interval: time.Second, Deadline: time.Minute}, "poll"},
		{"negative deadline", WaitOptions{Interval: time.Second, Deadline: -time.Minute, Fetch: func(context.Context) (*docs.Document, []RawComment, error) { return nil, nil, nil }}, "deadline"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Wait(context.Background(), nil, c.o)
			if err == nil {
				t.Fatalf("Wait(%+v) was run", c.o)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %q, want it to name %q", err, c.want)
			}
		})
	}
}

func TestTheDefaultSleeperEndsEarlyOnAnInterrupt(t *testing.T) {
	// The package variable the tests replace has to be right too: a sleeper
	// that ignored the context would hold a session open for the rest of an
	// interval after Ctrl-C.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	sleep(ctx, 30*time.Second)

	if time.Since(start) > time.Second {
		t.Errorf("the sleeper took %v on a cancelled context", time.Since(start))
	}
}

func TestTheDeadlineBoundsTheCallAndNotOnlyTheGapsBetweenPolls(t *testing.T) {
	// A poll that stalls is the one way a wait can outlive its deadline: the
	// only other bound on one is the client's timeout, which is minutes and
	// applies per request. A nine minute wait that answers at fourteen is
	// killed by the harness, and a killed wait comes back as interrupted, which
	// the skill reads as the person having stopped the session.
	//
	// So the deadline is checked with a real clock here, and the fake one the
	// other tests drive is deliberately not in the way.
	since := &Cursor{At: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)}
	polls := 0
	stalled := func(ctx context.Context) (*docs.Document, []RawComment, error) {
		polls++
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}

	// On its own goroutine, so a Wait that never returns fails here in five
	// seconds naming the deadline rather than hanging until go test's own
	// timeout ten minutes later. Without the derived context the poll blocks on
	// a background context that is never done, which is exactly the regression
	// this test is for, and a regression has to report itself.
	type answer struct {
		w   Waited
		err error
	}
	done := make(chan answer, 1)
	go func() {
		w, err := Wait(context.Background(), since, WaitOptions{
			Interval: 10 * time.Second,
			Deadline: 50 * time.Millisecond,
			Fetch:    stalled,
		})
		done <- answer{w, err}
	}()

	var w Waited
	var err error
	select {
	case a := <-done:
		w, err = a.w, a.err
	case <-time.After(5 * time.Second):
		t.Fatal("the wait had not answered five seconds into a 50ms deadline, so the stalled poll was unbounded")
	}

	if err != nil {
		// The deadline is a quiet ending, not a failed poll: the request was
		// cut short by this call's own clock and nothing about Drive is known.
		t.Fatalf("the deadline came back as a failure: %v", err)
	}
	if polls != 1 {
		t.Errorf("the wait polled %d times, and the first one never returned on its own", polls)
	}
	if len(w.Threads) != 0 {
		t.Errorf("threads = %v, want none: the poll never answered", w.Threads)
	}
	if w.Interrupted {
		t.Error("interrupted is true, and nothing sent a signal: a harness reading it stops the session")
	}
	if w.Cursor != since {
		t.Errorf("cursor = %v, want the one handed in: the next call asks the same question", w.Cursor)
	}
}

func TestAnInterruptDuringAStalledPollIsStillAnInterrupt(t *testing.T) {
	// The two contexts must not fold into one. A signal during a poll that the
	// deadline would also have cut is the person stopping the session, and the
	// flag is the only thing that says so.
	ctx, cancel := context.WithCancel(context.Background())
	stalled := func(ctx context.Context) (*docs.Document, []RawComment, error) {
		cancel()
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}

	w, err := Wait(ctx, nil, WaitOptions{
		Interval: 10 * time.Second,
		Deadline: time.Minute,
		Fetch:    stalled,
	})

	if err != nil {
		t.Fatalf("an interrupt is an answer, not a failure: %v", err)
	}
	if !w.Interrupted {
		t.Error("interrupted is false, and the context was cancelled during the poll")
	}
}

func TestTheWaitDoesNotPollOnTheDeadlineItHasAlreadyRunOutOf(t *testing.T) {
	// Thirty seconds at a ten second interval is three polls, not four. The
	// fourth used to run on the instant the last sleep landed on, which is the
	// instant the derived context is done, so both reads failed at once and
	// nothing reached Drive. polls is what the skill reads to know how many
	// times the binary asked, and a request that never left the process is not
	// an ask. The script holds three answers, so a fourth poll fails here.
	s := newScript(t, quiet(), quiet(), quiet())

	got, err := Wait(context.Background(), nil, s.options(10*time.Second, 30*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if got.Polls != 3 {
		t.Errorf("Polls = %d, want 3 inside a 30 second deadline at a 10 second interval", got.Polls)
	}
	if got.Waited != 30*time.Second {
		t.Errorf("Waited = %v, want the whole deadline: the last gap is slept out, not cut short", got.Waited)
	}
}

func TestAPollThatFailedOfItsOwnAccordPastTheDeadlineIsStillAFailure(t *testing.T) {
	// The deadline landing while a poll is in flight is not evidence that the
	// poll was cut short by it. Drive answering 503 at the tail of the window,
	// or a body that did not decode, arrives with the context already done, and
	// gating the quiet ending on the context alone reported that as a quiet
	// document. A degraded Drive answering slower than the interval makes it the
	// last poll of every wait, so the skill counts no failure and reads an
	// outage as a document nobody is writing in.
	//
	// A real clock, for the reason the deadline test uses one: the fake one the
	// other tests drive cannot put a poll's answer after the deadline.
	since := &Cursor{At: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)}
	drive := errors.New("drive answered 503")
	polls := 0
	late := func(ctx context.Context) (*docs.Document, []RawComment, error) {
		polls++
		// Past the deadline, and then a failure that is Drive's own rather than
		// the context's.
		time.Sleep(80 * time.Millisecond)
		return nil, nil, drive
	}

	type outcome struct {
		w   Waited
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		w, err := Wait(context.Background(), since, WaitOptions{
			Interval: 10 * time.Second,
			Deadline: 20 * time.Millisecond,
			Fetch:    late,
		})
		done <- outcome{w, err}
	}()

	var got outcome
	select {
	case got = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the wait had not answered five seconds into a 20ms deadline")
	}

	if !errors.Is(got.err, drive) {
		t.Fatalf("err = %v, want the failure Drive gave: a failed poll reported as a quiet window is the one silence this call forbids", got.err)
	}
	if polls != 1 {
		t.Errorf("polls = %d, want 1", polls)
	}
	if got.w.Polls != 1 {
		t.Errorf("Waited.Polls = %d, want 1: the poll that failed still went out", got.w.Polls)
	}
	if got.w.Interrupted {
		t.Error("interrupted is true, and nothing sent a signal")
	}
}
