package comments

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/plaintext"
)

// tape is a scripted follow: a sequence of listings, the cursor each poll asked
// with, the windows handed over, and the fake clock the sleeper drives. Nothing
// here waits, so a half-hour follow costs no wall time.
//
// It is not the wait's script with a cursor bolted on, because the cursor is the
// difference between the two loops: a wait asks the same question every poll and
// a follow asks from the last cursor it handed over, so the recorded cursors are
// what half of these tests read.
type tape struct {
	t       *testing.T
	answers []answer
	polls   int
	reads   int
	asked   []string
	handed  []Window
	clock   time.Time
	// onSleep runs before each sleep advances the clock, so a test can cancel
	// the context in the middle of a follow.
	onSleep func()
	// onPoll runs at the start of each poll, before its answer is taken.
	onPoll func()
}

func newTape(t *testing.T, answers ...answer) *tape {
	t.Helper()
	p := &tape{t: t, answers: answers, clock: time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)}

	realNow, realSleep := now, sleep
	now = func() time.Time { return p.clock }
	sleep = func(ctx context.Context, d time.Duration) {
		if p.onSleep != nil {
			p.onSleep()
		}
		p.clock = p.clock.Add(d)
	}
	t.Cleanup(func() { now, sleep = realNow, realSleep })
	return p
}

// list is the cursor-aware listing, recording the cursor each poll asked with.
// It narrows the way Fetch narrows, because Drive's bound is inclusive and a
// tape that answered wider than the wire would test a loop nobody runs.
func (p *tape) list(_ context.Context, since *Cursor) ([]RawComment, error) {
	p.asked = append(p.asked, since.String())
	if p.onPoll != nil {
		p.onPoll()
	}
	if p.polls >= len(p.answers) {
		p.t.Fatalf("the follow polled %d times, and the tape holds %d answers", p.polls+1, len(p.answers))
	}
	a := p.answers[p.polls]
	p.polls++
	// A poll takes a moment, the way a Drive listing does.
	p.clock = p.clock.Add(100 * time.Millisecond)
	return since.narrow(a.raw), a.err
}

// read is the document read, counting how often the follow asked for it.
func (p *tape) read(context.Context) (*docs.Document, error) {
	p.reads++
	return p.answers[p.polls-1].doc, nil
}

func (p *tape) hand(w Window) error {
	p.handed = append(p.handed, w)
	return nil
}

// options is a follow of n polls when the deadline is n intervals: the first
// poll runs at once and the last gap is slept out rather than polled on.
func (p *tape) options(interval, deadline time.Duration) FollowOptions {
	return FollowOptions{
		Interval: interval,
		Deadline: deadline,
		List:     p.list,
		Read:     p.read,
		Hand:     p.hand,
		Mark:     plaintext.DefaultMark,
	}
}

// found is a poll that carried these comments.
func found(raw ...RawComment) answer {
	return answer{doc: &docs.Document{}, raw: raw}
}

// topComment is one comment with no replies, created and modified at one
// instant, which is what Drive answers for a comment nobody has touched since.
func topComment(id, content, created string) RawComment {
	return RawComment{
		ID:           id,
		Author:       Author{DisplayName: "Nail Khusnullin"},
		CreatedTime:  created,
		ModifiedTime: created,
		Content:      content,
	}
}

// edited is one comment written at one instant and changed at a later one.
func edited(id, content, created, modified string) RawComment {
	c := topComment(id, content, created)
	c.ModifiedTime = modified
	return c
}

// answered is a comment with replies on it, and it moves the thread's
// modifiedTime to the newest of them, the way Drive moves it.
func answered(c RawComment, rs ...RawReply) RawComment {
	c.Replies = rs
	for _, r := range rs {
		if r.CreatedTime > c.ModifiedTime {
			c.ModifiedTime = r.CreatedTime
		}
	}
	return c
}

// aReply is one reply of a thread.
func aReply(id, content, created string) RawReply {
	return RawReply{
		ID:          id,
		Author:      Author{DisplayName: "Nail Khusnullin"},
		CreatedTime: created,
		Content:     content,
	}
}

// itemIDs is the ids of what a window carried, in the window's own order.
func itemIDs(w Window) []string {
	out := make([]string, 0, len(w.Items))
	for _, it := range w.Items {
		out = append(out, it.ID)
	}
	return out
}

func TestAWindowWithAMarkedCommentIsHandedOver(t *testing.T) {
	since := &Cursor{At: at("2026-10-27T12:00:00Z")}
	p := newTape(t, found(topComment("c1", "ai? which register", "2026-10-27T12:00:05Z")))

	got, err := Follow(context.Background(), since, p.options(2*time.Second, 2*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if len(p.handed) != 1 {
		t.Fatalf("%d windows were handed over, want 1", len(p.handed))
	}
	w := p.handed[0]
	if w.Since != since {
		t.Errorf("Since = %v, want the cursor the poll asked with", w.Since)
	}
	if w.Cursor == nil || !w.Cursor.At.Equal(at("2026-10-27T12:00:05Z")) {
		t.Errorf("Cursor = %v, want the comment's own instant", w.Cursor)
	}
	if want := at("2026-10-27T12:00:00.1Z"); !w.Noticed.Equal(want) {
		t.Errorf("Noticed = %v, want %v: the instant the poll that found it came back", w.Noticed, want)
	}
	if want := []string{"c1"}; !reflect.DeepEqual(itemIDs(w), want) {
		t.Fatalf("items = %+v, want %v", w.Items, want)
	}
	if w.Items[0].Marker != "ai?" || w.Items[0].Thread != "c1" {
		t.Errorf("item = %+v, want the thread's own id and its marker", w.Items[0])
	}
	if got.Cursor != w.Cursor {
		t.Errorf("Followed.Cursor = %v, want the cursor the window was handed over with", got.Cursor)
	}
}

func TestAWindowOfOnlyGdocsRepliesIsHeld(t *testing.T) {
	// A reply gdoc wrote is the thing the follow exists not to pay for: under
	// --wait it ended the call, the skill spent a turn, and the turn read its own
	// receipt. The mark is the only record of authorship a thread carries, so
	// every mark gdoc recognises is held: the three every gdoc knows, and the one
	// this run writes when a colleague set their own.
	cases := []struct {
		name     string
		inEffect string
		opening  string
	}{
		{"the mark in effect", "✚", "✚"},
		{"the robot", plaintext.DefaultMark, "🤖"},
		{"the four dots", plaintext.DefaultMark, "⁘"},
		{"the therefore sign", plaintext.DefaultMark, "∴"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			since := &Cursor{At: at("2026-10-27T12:00:00Z"), Ids: []string{"c1"}}
			p := newTape(t, found(answered(
				topComment("c1", "Please check this", "2026-10-27T11:00:00Z"),
				aReply("r1", c.opening+" The operations register.", "2026-10-27T12:00:03Z"),
			)))
			o := p.options(2*time.Second, 2*time.Second)
			o.Mark = c.inEffect

			got, err := Follow(context.Background(), since, o)

			if err != nil {
				t.Fatal(err)
			}
			if len(p.handed) != 0 {
				t.Errorf("%d windows were handed over for a reply opening with %q, want none", len(p.handed), c.opening)
			}
			if got.Cursor != since {
				t.Errorf("Cursor = %v, want the one handed in: a held window prints none", got.Cursor)
			}
		})
	}
}

func TestGdocsReplyInAnOldMarkedThreadIsHeld(t *testing.T) {
	// The thread's own marker is not the question. An ai? comment handed over on
	// an earlier window is answered by gdoc, Drive moves the thread's
	// modifiedTime, and the listing carries the whole thread again: marker and
	// all. What decides is whether anything in it is new and not gdoc's.
	since := &Cursor{At: at("2026-10-27T12:00:00Z"), Ids: []string{"c1"}}
	p := newTape(t, found(answered(
		topComment("c1", "ai? which register", "2026-10-27T11:00:00Z"),
		aReply("r1", plaintext.Prefixed(plaintext.DefaultMark, "The operations register."), "2026-10-27T12:00:03Z"),
	)))

	_, err := Follow(context.Background(), since, p.options(2*time.Second, 2*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if len(p.handed) != 0 {
		t.Errorf("%d windows were handed over, want none: the only new item was gdoc's own reply", len(p.handed))
	}
}

func TestAnEditedOldCommentIsHeld(t *testing.T) {
	// A cursor moves on modifiedTime, so an old comment somebody edited comes
	// back on every poll until something else moves the instant. Reading the
	// edit as news would hand the same thread over for ever, which is the
	// failure the cursor's own ids exist to stop under --wait.
	since := &Cursor{At: at("2026-10-27T12:00:00Z"), Ids: []string{"c1"}}
	p := newTape(t, found(edited("c1", "ai? which register", "2026-10-27T11:00:00Z", "2026-10-27T12:00:05Z")))

	got, err := Follow(context.Background(), since, p.options(2*time.Second, 2*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if len(p.handed) != 0 {
		t.Errorf("%d windows were handed over, want none: nothing in the thread is new", len(p.handed))
	}
	if got.Cursor != since {
		t.Errorf("Cursor = %v, want the one handed in", got.Cursor)
	}
}

func TestAnUnmarkedReplyIsHeldAndRidesInTheNextWindow(t *testing.T) {
	// Held, not dropped. The item is in the window the next marked item opens,
	// so the report of that window names it and the session acts on neither
	// until something addressed to gdoc arrives.
	since := &Cursor{At: at("2026-10-27T12:00:00Z"), Ids: []string{"c1"}}
	old := topComment("c1", "Please check this", "2026-10-27T11:00:00Z")
	chat := aReply("r1", "Looks fine to me.", "2026-10-27T12:00:02Z")
	ask := aReply("r2", "ai! add it to the register", "2026-10-27T12:00:04Z")
	p := newTape(t,
		found(answered(old, chat)),
		found(answered(old, chat, ask)),
	)

	_, err := Follow(context.Background(), since, p.options(2*time.Second, 4*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if len(p.handed) != 1 {
		t.Fatalf("%d windows were handed over, want 1", len(p.handed))
	}
	if want := []string{"r1", "r2"}; !reflect.DeepEqual(itemIDs(p.handed[0]), want) {
		t.Fatalf("items = %+v, want %v: the held reply rides in the window", p.handed[0].Items, want)
	}
	if p.handed[0].Items[0].Marker != MarkerNone || p.handed[0].Items[1].Marker != "ai!" {
		t.Errorf("markers = %q and %q, want none and ai!",
			p.handed[0].Items[0].Marker, p.handed[0].Items[1].Marker)
	}
}

func TestEveryMarkerCounts(t *testing.T) {
	// The three markers are one rule. ai: was the one most easily left out,
	// because it is the marker a colleague uses for a note rather than a
	// question, and a follow that slept through it is a comment nobody answers.
	cases := []struct {
		content string
		want    int
	}{
		{"ai? which register", 1},
		{"ai! add it to the register", 1},
		{"ai: the register is the operations one", 1},
		{"Please check this", 0},
	}
	for _, c := range cases {
		t.Run(c.content, func(t *testing.T) {
			since := &Cursor{At: at("2026-10-27T12:00:00Z")}
			p := newTape(t, found(topComment("c1", c.content, "2026-10-27T12:00:05Z")))

			_, err := Follow(context.Background(), since, p.options(2*time.Second, 2*time.Second))

			if err != nil {
				t.Fatal(err)
			}
			if len(p.handed) != c.want {
				t.Errorf("%d windows were handed over for %q, want %d", len(p.handed), c.content, c.want)
			}
		})
	}
}

func TestAMarkedReplyAtTheCursorsInstantIsNews(t *testing.T) {
	// Drive writes to the millisecond, so two items can share an instant with
	// only one of them reported. The cursor's ids are what tell them apart, and
	// the follow asks the cursor rather than inventing a second rule.
	since := &Cursor{At: at("2026-10-27T12:00:05Z"), Ids: []string{"other"}}
	p := newTape(t, found(answered(
		topComment("c2", "Please check this", "2026-10-27T11:00:00Z"),
		aReply("r1", "ai! add it to the register", "2026-10-27T12:00:05Z"),
	)))

	_, err := Follow(context.Background(), since, p.options(2*time.Second, 2*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if len(p.handed) != 1 {
		t.Fatalf("%d windows were handed over, want 1: the thread is not among the ids the cursor reported", len(p.handed))
	}
	if want := []string{"r1"}; !reflect.DeepEqual(itemIDs(p.handed[0]), want) {
		t.Errorf("items = %+v, want %v", p.handed[0].Items, want)
	}
}

func TestAHandedOverWindowIsNotHandedOverAgain(t *testing.T) {
	// The second poll asks from the cursor the window printed, whose instant is
	// the comment's own and whose ids name it. Without either half the same
	// comment is news on every poll for ever.
	since := &Cursor{At: at("2026-10-27T12:00:00Z")}
	ask := topComment("c1", "ai? which register", "2026-10-27T12:00:05Z")
	p := newTape(t, found(ask), found(ask))

	_, err := Follow(context.Background(), since, p.options(2*time.Second, 4*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if len(p.handed) != 1 {
		t.Errorf("%d windows were handed over, want 1", len(p.handed))
	}
	if p.polls != 2 {
		t.Errorf("polls = %d, want 2", p.polls)
	}
}

func TestAHeldPollAsksFromTheLastPrintedCursor(t *testing.T) {
	since := &Cursor{At: at("2026-10-27T12:00:00Z"), Ids: []string{"c1"}}
	held := found(answered(
		topComment("c1", "Please check this", "2026-10-27T11:00:00Z"),
		aReply("r1", "Looks fine to me.", "2026-10-27T12:00:02Z"),
	))
	p := newTape(t, held, held, held)

	_, err := Follow(context.Background(), since, p.options(2*time.Second, 6*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	want := []string{since.String(), since.String(), since.String()}
	if !reflect.DeepEqual(p.asked, want) {
		t.Errorf("the polls asked from %v, want the same cursor each time: nothing was printed", p.asked)
	}
}

func TestThePollAfterAWindowAsksFromItsCursor(t *testing.T) {
	since := &Cursor{At: at("2026-10-27T12:00:00Z")}
	ask := topComment("c1", "ai? which register", "2026-10-27T12:00:05Z")
	p := newTape(t, found(ask), found(ask))

	_, err := Follow(context.Background(), since, p.options(2*time.Second, 4*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if len(p.handed) != 1 {
		t.Fatalf("%d windows were handed over, want 1", len(p.handed))
	}
	want := []string{since.String(), p.handed[0].Cursor.String()}
	if !reflect.DeepEqual(p.asked, want) {
		t.Errorf("the polls asked from %v, want %v", p.asked, want)
	}
}

func TestAHeldWindowDoesNotReadTheDocument(t *testing.T) {
	// The document is read once, on the first poll, for the title and the tab
	// count the envelope reports. A window is judged on the listing alone,
	// because the line names ids and times and nothing a range could place.
	since := &Cursor{At: at("2026-10-27T12:00:00Z"), Ids: []string{"c1"}}
	held := found(answered(
		topComment("c1", "Please check this", "2026-10-27T11:00:00Z"),
		aReply("r1", "Looks fine to me.", "2026-10-27T12:00:02Z"),
	))
	p := newTape(t, held, held, held)

	_, err := Follow(context.Background(), since, p.options(2*time.Second, 6*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if p.polls != 3 {
		t.Fatalf("polls = %d, want 3", p.polls)
	}
	if p.reads != 1 {
		t.Errorf("the follow read the document %d times, want 1", p.reads)
	}
}

func TestFollowRunsToItsDeadline(t *testing.T) {
	since := &Cursor{At: at("2026-10-27T12:00:00Z")}
	p := newTape(t, quiet(), quiet(), quiet())

	got, err := Follow(context.Background(), since, p.options(2*time.Second, 6*time.Second))

	if err != nil {
		t.Fatal(err)
	}
	if got.Interrupted {
		t.Error("Interrupted = true at a deadline nobody stopped")
	}
	if got.Polls != 3 {
		t.Errorf("Polls = %d, want 3 inside a 6 second deadline at a 2 second interval", got.Polls)
	}
	if got.Waited != 6*time.Second {
		t.Errorf("Waited = %v, want the whole deadline", got.Waited)
	}
	if got.Cursor != since {
		t.Errorf("Cursor = %v, want the one handed in unchanged", got.Cursor)
	}
	if len(p.handed) != 0 {
		t.Errorf("%d windows were handed over on a quiet document", len(p.handed))
	}
}

func TestHeldItemsAtTheEndLeaveTheLastPrintedCursor(t *testing.T) {
	// The cursor the follow comes back with is the one the last line printed,
	// never one that moved past an item no line named. The next follow starts
	// there, so a held item is picked up again rather than lost.
	ask := topComment("c1", "ai? which register", "2026-10-27T12:00:03Z")
	held := answered(
		topComment("c2", "Please check this", "2026-10-27T11:00:00Z"),
		aReply("r1", "Looks fine to me.", "2026-10-27T12:00:08Z"),
	)

	t.Run("at the deadline", func(t *testing.T) {
		since := &Cursor{At: at("2026-10-27T12:00:00Z"), Ids: []string{"c2"}}
		p := newTape(t, found(ask), found(ask, held))

		got, err := Follow(context.Background(), since, p.options(2*time.Second, 4*time.Second))

		if err != nil {
			t.Fatal(err)
		}
		if len(p.handed) != 1 {
			t.Fatalf("%d windows were handed over, want 1", len(p.handed))
		}
		if got.Cursor != p.handed[0].Cursor {
			t.Errorf("Cursor = %v, want the one the window printed, %v", got.Cursor, p.handed[0].Cursor)
		}
	})

	t.Run("on a signal", func(t *testing.T) {
		since := &Cursor{At: at("2026-10-27T12:00:00Z"), Ids: []string{"c2"}}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p := newTape(t, found(ask), found(ask, held))
		sleeps := 0
		p.onSleep = func() {
			sleeps++
			if sleeps == 2 {
				cancel()
			}
		}

		got, err := Follow(ctx, since, p.options(2*time.Second, time.Hour))

		if err != nil {
			t.Fatal(err)
		}
		if !got.Interrupted {
			t.Error("Interrupted = false after the context was cancelled")
		}
		if len(p.handed) != 1 {
			t.Fatalf("%d windows were handed over, want 1", len(p.handed))
		}
		if got.Cursor != p.handed[0].Cursor {
			t.Errorf("Cursor = %v, want the one the window printed, %v", got.Cursor, p.handed[0].Cursor)
		}
	})
}

func TestASignalEndsTheFollowAndSaysSo(t *testing.T) {
	// Ctrl-C is how a live review ends, and it is an answer: the window already
	// handed over stands, and the flag is what tells the skill the person
	// stopped rather than the clock.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	since := &Cursor{At: at("2026-10-27T12:00:00Z")}
	p := newTape(t, found(topComment("c1", "ai? which register", "2026-10-27T12:00:03Z")), quiet())
	p.onSleep = cancel

	got, err := Follow(ctx, since, p.options(2*time.Second, time.Hour))

	if err != nil {
		t.Fatalf("an interrupt came back as an error: %v", err)
	}
	if !got.Interrupted {
		t.Error("Interrupted = false after the context was cancelled")
	}
	if got.Polls != 1 {
		t.Errorf("Polls = %d, want 1: a cancelled context polls no further", got.Polls)
	}
	if len(p.handed) != 1 {
		t.Errorf("%d windows were handed over, want the one the first poll found", len(p.handed))
	}
}

func TestAFailedPollEndsTheFollow(t *testing.T) {
	// Nothing is retried in silence. A follow that swallowed a refused listing
	// would print no line for an hour and read as a quiet document.
	refused := errors.New("the comment listing was refused: file was not given to this command")
	since := &Cursor{At: at("2026-10-27T12:00:00Z")}
	p := newTape(t,
		found(topComment("c1", "ai? which register", "2026-10-27T12:00:03Z")),
		answer{err: refused},
	)

	got, err := Follow(context.Background(), since, p.options(2*time.Second, time.Hour))

	if !errors.Is(err, refused) {
		t.Fatalf("err = %v, want the poll's own error", err)
	}
	if got.Polls != 2 {
		t.Errorf("Polls = %d, want the two polls the follow made", got.Polls)
	}
	if len(p.handed) != 1 {
		t.Fatalf("%d windows were handed over, want 1", len(p.handed))
	}
	if got.Cursor != p.handed[0].Cursor {
		t.Errorf("Cursor = %v, want the one the window printed, %v", got.Cursor, p.handed[0].Cursor)
	}
	if got.Interrupted {
		t.Error("Interrupted = true, and nothing sent a signal")
	}
}

func TestFollowRefusesOptionsItCannotRun(t *testing.T) {
	// A follow with nothing to hand a window to is a poll whose findings go
	// nowhere, and it is the caller's mistake rather than something a retry
	// could fix.
	cases := []struct {
		name string
		o    FollowOptions
		want string
	}{
		{"no listing", FollowOptions{Interval: time.Second, Deadline: time.Minute, Read: noRead, Hand: noHand}, "listing"},
		{"no read", FollowOptions{Interval: time.Second, Deadline: time.Minute, List: noFollowList, Hand: noHand}, "read"},
		{"nothing to hand to", FollowOptions{Interval: time.Second, Deadline: time.Minute, List: noFollowList, Read: noRead}, "hand"},
		{"no interval", FollowOptions{Deadline: time.Minute, List: noFollowList, Read: noRead, Hand: noHand}, "interval"},
		{"negative deadline", FollowOptions{Interval: time.Second, Deadline: -time.Minute, List: noFollowList, Read: noRead, Hand: noHand}, "deadline"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Follow(context.Background(), nil, c.o)
			if err == nil {
				t.Fatalf("Follow(%+v) was run", c.o)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %q, want it to name %q", err, c.want)
			}
		})
	}
}

// noFollowList and noHand answer nothing, for the options table.
func noFollowList(context.Context, *Cursor) ([]RawComment, error) { return nil, nil }
func noHand(Window) error                                         { return nil }

func TestAWindowThatCouldNotBeWrittenEndsTheFollow(t *testing.T) {
	// The cursor moves when the caller was told, not when the follow found
	// something. A follow that moved it past a line the stream refused would
	// report a window nobody read, and the next line would start after it.
	refused := errors.New("the window line could not be written: broken pipe")
	since := &Cursor{At: at("2026-10-27T12:00:00Z")}
	p := newTape(t, found(topComment("c1", "ai? which register", "2026-10-27T12:00:03Z")))
	o := p.options(2*time.Second, time.Hour)
	o.Hand = func(Window) error { return refused }

	got, err := Follow(context.Background(), since, o)

	if !errors.Is(err, refused) {
		t.Fatalf("err = %v, want the write's own error", err)
	}
	if got.Polls != 1 {
		t.Errorf("Polls = %d, want the one poll the follow made", got.Polls)
	}
	if got.Cursor != since {
		t.Errorf("Cursor = %v, want the one handed in, %v", got.Cursor, since)
	}
}
