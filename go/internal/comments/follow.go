// This file is the follow: the poll held open, what one window carries, the
// wake rule, and the hold. wait.go is the poll that ends on the first window,
// and the two share the deadline context and the ending of a failed poll.
// doc.go holds why a window is handed over at all and why the rest is held.

package comments

import (
	"context"
	"errors"
	"time"

	"gdoc/internal/plaintext"
)

// FollowList is the listing one poll of a follow makes, asked from the cursor
// the follow chose. It takes the cursor where Wait's List does not, because that
// is the difference between the two loops: a wait asks one question for the
// whole call, and a follow asks from the last cursor it handed over, so a window
// nobody printed is asked for again on the next poll.
//
// It is a parameter for the reason List is: this package holds no session and no
// URL, so the command builds the Fetch call and a test hands in a script.
type FollowList func(ctx context.Context, since *Cursor) ([]RawComment, error)

// FollowOptions is what one follow needs, and nothing more.
type FollowOptions struct {
	// Interval is the time between polls. The command passes a constant.
	Interval time.Duration
	// Deadline is how long to look before the follow ends of its own accord.
	Deadline time.Duration
	// List is the listing one poll makes, from the cursor it is given.
	List FollowList
	// Read is the document read the first poll makes, once, for the fields the
	// caller's envelope reports.
	Read Read
	// Mark is the mark this run writes, read beside the three marks every gdoc
	// knows to say whether an item is gdoc's own.
	Mark string
	// Hand is where one window goes, as soon as it is found. It is a callback
	// rather than a returned list because the caller writes a line per window
	// while the follow is still running: a follow that collected its windows and
	// answered at the end would be a wait with a longer deadline.
	Hand func(Window) error
}

// NewItem is one comment or reply in a window that the cursor it was asked from
// had not reported. Every field is a fact: which entry it is, which thread it
// sits in, the marker its own first token carries, when it was written, and
// whether it opens with a mark gdoc recognises. Nothing here says whether it is
// work, or whether it was answered.
type NewItem struct {
	ID      string
	Thread  string
	Marker  string
	Created string
	ByGdoc  bool
}

// Window is one poll's worth of news: the cursor the poll asked from, the cursor
// the next one asks from, when gdoc saw it, and every new item in it, the held
// ones included.
type Window struct {
	Since   *Cursor
	Cursor  *Cursor
	Noticed time.Time
	Items   []NewItem
}

// Followed is what one follow did. Every field is a count, a duration, a cursor
// or a flag, and there is none saying whether the hour was worth anything.
type Followed struct {
	// Cursor is the cursor of the last window handed over, and the one handed in
	// when there was none.
	Cursor      *Cursor
	Polls       int
	Waited      time.Duration
	Interrupted bool
}

// Follow polls until its deadline passes, a poll fails, or ctx is done, handing
// over every window that carries something new addressed to gdoc.
//
// It is Wait with two differences, and both of them are about what a poll costs
// the caller. Wait returns on the first poll whose listing carried anything, so
// the caller spends a turn on it, and that includes gdoc's own reply landing a
// second after it was written. Follow hands over only a window with a new
// comment or reply whose marker is one of the three and which does not open with
// a mark gdoc recognises, and keeps looking. So a quiet half hour costs one
// listing every interval and nothing else at all.
//
// A window nobody was handed leaves the cursor where it was, so the next poll
// asks the same question and the items it held ride in the next window that is
// handed over. Nothing is dropped, and nothing is reported twice: the cursor the
// follow comes back with is the one the last window carried.
//
// The document is read once, on the first poll, because the caller reads it for
// the title and the tab count its envelope reports. A window is judged on the
// listing alone: Threads is asked for a nil document, so every thread comes back
// unplaced and no range is computed for a line that names ids and times.
//
// The three endings are the ones Wait has. The deadline and a signal are
// answers, with the last cursor handed over; a failed poll is the error, because
// a caller told the document was quiet would believe it.
func Follow(ctx context.Context, since *Cursor, o FollowOptions) (Followed, error) {
	if o.List == nil {
		return Followed{}, errors.New("the follow was given no listing to poll")
	}
	if o.Read == nil {
		return Followed{}, errors.New("the follow was given no document read to poll")
	}
	if o.Hand == nil {
		return Followed{}, errors.New("the follow was given nothing to hand a window to")
	}
	if o.Interval <= 0 {
		return Followed{}, errors.New("the follow needs a positive interval between polls")
	}
	if o.Deadline < 0 {
		return Followed{}, errors.New("the follow needs a deadline that is not negative")
	}

	start := now()
	out := Followed{Cursor: since}
	// The cursor the next poll asks from, which moves only when a window is
	// handed over.
	cursor := since

	poll, cancel := pollContext(ctx, o.Deadline)
	defer cancel()

	done := func() Followed {
		out.Waited = now().Sub(start)
		return out
	}
	stopped := func() Followed {
		out.Interrupted = true
		return done()
	}
	failed := func(err error) (Followed, error) {
		switch endingOf(ctx, poll, err) {
		case endingInterrupt:
			return stopped(), nil
		case endingDeadline:
			return done(), nil
		}
		return done(), err
	}

	read := false
	for {
		if ctx.Err() != nil {
			return stopped(), nil
		}

		raw, err := o.List(poll, cursor)
		out.Polls++
		if err != nil {
			return failed(err)
		}
		if !read {
			if _, err := o.Read(poll); err != nil {
				return failed(err)
			}
			read = true
		}

		// A nil document on purpose: see the doc comment.
		threads, _ := Threads(raw, nil, o.Mark)
		if hasNewMarked(threads, cursor, o.Mark) {
			next := NextCursor(cursor, threads)
			w := Window{
				Since:   cursor,
				Cursor:  next,
				Noticed: now(),
				Items:   newItems(threads, cursor, o.Mark),
			}
			if err := o.Hand(w); err != nil {
				// A line that could not be written is not a window the caller
				// was told about, so the cursor does not move and the error
				// carries out. Writing the next line to a stream that refused
				// the last one would report windows nobody read.
				return done(), err
			}
			cursor = next
			out.Cursor = next
		}

		left := o.Deadline - now().Sub(start)
		if left <= 0 {
			return done(), nil
		}
		if left <= o.Interval {
			// The last gap is slept out rather than polled on, for the reason
			// Wait sleeps it out: a poll starting on the instant the deadline
			// lands on runs on a context already done, so it sends nothing and
			// Polls would count a request that never left the process.
			sleep(ctx, left)
			if ctx.Err() != nil {
				return stopped(), nil
			}
			return done(), nil
		}
		sleep(ctx, o.Interval)
	}
}

// hasNewMarked is the wake rule, and the one question that decides whether a
// poll costs the caller a turn: is there an item here that is news by the
// cursor's own rule, carries one of the three markers, and does not open with a
// mark gdoc recognises.
//
// Without the news part an old marked thread wakes the follow on every poll
// Drive returns it on. Without the marker part an unmarked reply between two
// colleagues costs a turn, which is what the hold exists to stop.
//
// The mark part does not decide anything today, and it is written out anyway.
// markerOf reads the first token and a mark is what the text opens with, so an
// item that opens with a mark has the mark as its first token and its marker is
// none already: the two conditions cannot both be true. What the clause says is
// that the hold on gdoc's own words does not rest on that coincidence, which is
// the one thing this whole loop is for. A mark that could ever read as a marker,
// or a markerOf that looked past the opening, would otherwise have the session
// answering its own receipts again.
//
// TestEveryMarkerCounts, TestAWindowOfOnlyGdocsRepliesIsHeld and
// TestAnEditedOldCommentIsHeld are the three pins.
func hasNewMarked(threads []Thread, since *Cursor, mark string) bool {
	for _, item := range newItems(threads, since, mark) {
		if item.Marker != MarkerNone && !item.ByGdoc {
			return true
		}
	}
	return false
}

// newItems is every comment and reply in the listing the cursor had not
// reported, in the order Drive answered.
//
// The instant it asks about is createdTime, for a thread as well as a reply,
// and modifiedTime is deliberately not in the rule. A cursor advances on
// modifiedTime, which is what Drive filters on, so an old comment somebody
// edited and a thread Drive touched for another reason come back on every poll
// until something else moves the instant. Reading that as a new item would hand
// the same thread over for ever, which is the failure the cursor's own ids exist
// to stop under a wait.
func newItems(threads []Thread, since *Cursor, mark string) []NewItem {
	var out []NewItem
	for _, t := range threads {
		if isNewAt(since, t.Created, t.ID) {
			out = append(out, NewItem{
				ID:     t.ID,
				Thread: t.ID,
				Marker: t.Marker,
				// A top-level comment gdoc writes opens with the mark too, so
				// the question is asked of the thread's own content and not
				// only of its replies.
				ByGdoc:  plaintext.OpensWithMark(t.Content, mark),
				Created: t.Created,
			})
		}
		for _, r := range t.Replies {
			if isNewAt(since, r.Created, t.ID) {
				out = append(out, NewItem{
					ID:      r.ID,
					Thread:  t.ID,
					Marker:  r.Marker,
					ByGdoc:  r.ByGdoc,
					Created: r.Created,
				})
			}
		}
	}
	return out
}

// isNewAt is the cursor's own boundary rule, asked of one instant on one
// thread: strictly newer than the cursor, or on the cursor's instant with the
// thread not among the ids it reported there. It is the cursor's three
// predicates and no fourth rule of its own, so the follow and the listing
// cannot disagree about what the boundary is.
//
// A nil cursor has reported nothing, so everything is news: that is a follow
// with no --since, which cmd/gdoc refuses, and the honest answer here.
func isNewAt(since *Cursor, created, thread string) bool {
	if since == nil {
		return true
	}
	if since.newer(created) {
		return true
	}
	if since.reported(thread) {
		return false
	}
	return since.sameInstant(created)
}
