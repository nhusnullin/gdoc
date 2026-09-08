// The wait is one call that looks more than once.
//
// A live review session is the skill calling `comments --since <cursor> --wait
// <duration>` in a loop. The waiting happens here rather than in the skill for
// one measured reason: a Claude Code session cannot wake more often than once a
// minute, and every idle tick would spend a model turn. So a quiet document
// costs nothing, and a comment is seen within one interval of being written.
// Nail's decision, 2026-09-07.
//
// Nothing here is a watcher. The call ends: on the first window with activity
// in it, at its deadline, on a failed poll, or on the signal that ends the
// session. It keeps no session state and writes no file of its own, and the
// cursor it hands back is the only thing that carries to the next call. The one
// file a poll can replace is the saved OAuth token, which the session refreshes
// as it does for every other command: that is the credential, not anything the
// wait learned.
//
// And nothing here judges. A window is what Fetch narrowed and Threads joined,
// gdoc's own replies included. Whether a thread is work, and whether a window
// that is only receipts is worth a word, is the skill's, reading the same facts
// the one-shot listing prints.
package comments

import (
	"context"
	"errors"
	"time"

	"gdoc/internal/docs"
)

// now is time.Now, and sleep is a context-aware time.Sleep. Both are package
// variables so a test can drive a ten-second interval in no wall time. They are
// not options: a caller that could pass its own clock could make a wait that
// never ends.
var (
	now = time.Now

	sleep = func(ctx context.Context, d time.Duration) {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-t.C:
		case <-ctx.Done():
			// Ctrl-C during a sleep ends the session now, rather than holding
			// the terminal for the rest of the interval.
		}
	}
)

// Poll is one look at the document: the Docs read and the comment listing, as
// the command already makes them. It is a parameter so this package still holds
// no session and no URL, which is what keeps net/http out of this room and lets
// every test hand in a script.
type Poll func(ctx context.Context) (*docs.Document, []RawComment, error)

// WaitOptions is what one wait needs, and nothing more.
type WaitOptions struct {
	// Interval is the time between polls. The command passes a constant.
	Interval time.Duration
	// Deadline is how long to look before answering with an empty window.
	Deadline time.Duration
	// Fetch is one poll.
	Fetch Poll
}

// Waited is what one wait saw. Every field is a fact: what arrived, where the
// next call starts, how many times it looked, how long it took, and whether the
// person stopped it. There is no field saying whether any of that was work.
type Waited struct {
	Threads     []Thread
	Unplaced    []string
	Cursor      *Cursor
	Polls       int
	Waited      time.Duration
	Interrupted bool
}

// Wait polls until there is activity after the cursor, the deadline passes, a
// poll fails, or ctx is done.
//
// The first poll happens at once rather than after an interval, because latency
// is the point of a live session: a wait that slept first would cost the
// interval on every call the skill makes, quiet document or not.
//
// The three quiet endings are answers, not failures. A deadline reached and an
// interrupt both come back with no threads and the cursor handed in unchanged,
// so the next call asks the same question and nothing is lost. A failed poll is
// the one that comes back as an error: a Docs read or a Drive listing that
// failed is not retried in silence, because a caller told the document was
// quiet would believe it.
func Wait(ctx context.Context, since *Cursor, o WaitOptions) (Waited, error) {
	if o.Fetch == nil {
		return Waited{}, errors.New("the wait was given no poll to make")
	}
	if o.Interval <= 0 {
		return Waited{}, errors.New("the wait needs a positive interval between polls")
	}
	if o.Deadline < 0 {
		return Waited{}, errors.New("the wait needs a deadline that is not negative")
	}

	start := now()
	// Never nil, so a caller reading the JSON does not special-case an empty
	// window as null.
	out := Waited{Threads: []Thread{}, Cursor: since}

	// The deadline bounds the whole call, not just the gaps between polls. The
	// only other bound on one poll is the client's own timeout, which is
	// minutes and applies per request, so a poll that stalls late in the window
	// could carry a nine minute wait well past the ten the nine was chosen to
	// fit inside. What ends such a call is the harness killing it, and a killed
	// wait comes back as interrupted, which the skill reads as the person
	// having stopped the session.
	//
	// It is a second context rather than the caller's, so the two endings stay
	// apart: ctx being done is the signal, and this one being done is the
	// deadline, which is an empty window and not a failure.
	poll := ctx
	if o.Deadline > 0 {
		var cancel context.CancelFunc
		poll, cancel = context.WithTimeout(ctx, o.Deadline)
		defer cancel()
	}

	for {
		if ctx.Err() != nil {
			return interrupted(out, start), nil
		}

		d, raw, err := o.Fetch(poll)
		out.Polls++
		if err != nil {
			if ctx.Err() != nil {
				// The signal that ends the session ends the request in flight.
				// Blaming Drive for the person's Ctrl-C would send somebody to
				// look at a read that was fine.
				return interrupted(out, start), nil
			}
			if poll.Err() != nil && errors.Is(err, context.DeadlineExceeded) {
				// The deadline cut the poll short. That is the quiet ending
				// this call already has a shape for: no threads, the cursor
				// handed in, and the next call asks the same question.
				//
				// The gate is the error's own identity and not just the
				// context's state, because the two answer different questions.
				// A poll that is still in flight when the deadline lands can
				// return a Drive failure of its own a moment later: a 503 read
				// at the tail of the window, or a body that did not decode. The
				// context is done by then either way, so reading only that
				// reported a failed poll as a quiet document, which is the one
				// thing this call's doc comment promises it never does. A
				// degraded Drive answering slower than the interval makes that
				// the last poll of every wait rather than a knife-edge race.
				out.Waited = now().Sub(start)
				return out, nil
			}
			out.Waited = now().Sub(start)
			return out, err
		}

		threads, unplaced := Threads(raw, d)
		if len(threads) > 0 {
			out.Threads = threads
			out.Unplaced = unplaced
			out.Cursor = NextCursor(since, threads)
			out.Waited = now().Sub(start)
			return out, nil
		}

		left := o.Deadline - now().Sub(start)
		if left <= 0 {
			out.Waited = now().Sub(start)
			return out, nil
		}
		if left <= o.Interval {
			// Less than an interval is left, so this is the last of the wait.
			// It sleeps out the remainder rather than polling on the boundary:
			// a nine minute deadline answers at nine minutes, not at nine
			// minutes and one interval, and not one poll short of nine either.
			//
			// The poll that used to run there is why this is a return. The
			// derived context is done by the instant the clamped sleep lands
			// on, so both reads failed at once, nothing reached Drive, and
			// polls counted a request that never left the process.
			sleep(ctx, left)
			if ctx.Err() != nil {
				return interrupted(out, start), nil
			}
			out.Waited = now().Sub(start)
			return out, nil
		}
		sleep(ctx, o.Interval)
	}
}

// interrupted is the answer a stopped session gets: no threads, the cursor it
// was given, and the flag that says which of the two quiet endings this was.
func interrupted(out Waited, start time.Time) Waited {
	out.Interrupted = true
	out.Waited = now().Sub(start)
	return out
}
