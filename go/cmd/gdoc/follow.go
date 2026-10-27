// This file is what a poll is bounded by, what it says it did, and what a
// window is written as: how long the two duration flags may ask for, what each
// of them refuses before any read is made, the waited object both modes report,
// the shape of one window line, and the envelope a follow ends on.
//
// read.go holds the two commands themselves. commentsFollow is there rather
// than here for one reason: it installs the signal handler, and read.go is the
// one file in this binary that names os/signal.

package main

import (
	"context"
	"fmt"
	"time"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/emit"
)

// waitInterval is how long a poll sleeps between looks, under --wait and under
// --follow alike. It is two seconds, a constant the spec names, and a package
// variable only so a test can drive two polls in no wall time. It is
// deliberately not a flag: a caller that could set it could poll Drive as fast
// as it liked, and nothing on the wire would say the interval had changed.
// TestTheWaitIntervalIsTwoSeconds pins the literal.
var waitInterval = 2 * time.Second

// maxWait is the longest one call will look, whichever flag asked. The skill
// asks for nine minutes under --wait, because the tool that runs the command
// waits ten at most, and for about half an hour under --follow, where nothing
// is waiting on the call to return. The hour is the bar that stops either of
// them from becoming the watcher principle 1 refuses to have.
const maxWait = time.Hour

// followItemCeiling is how many items one window line names. A document
// somebody pasted thirty comments into at once is still a line a reader has to
// be able to read, so the rest are counted rather than listed: the cursor moved
// past them either way, and the next listing is where their words are.
// TestALineListsAtMostTwentyItems.
const followItemCeiling = 20

// parseWait reads the value of a duration flag. Every refusal names the flag it
// was written on and quotes what it was given: a caller reading "not a
// duration" without its own word back cannot tell which argument it wrote
// wrongly. --wait's three sentences are what they were before --follow existed:
// TestTheWaitKeepsItsOwnWords.
func parseWait(flag, text string) (time.Duration, error) {
	d, err := time.ParseDuration(text)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not a length of time. Write it as 9m or 90s", flag, text)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s %q is not a length of time to look for", flag, text)
	}
	if d > maxWait {
		return 0, fmt.Errorf("%s %q is longer than the %s this command will look for", flag, text, maxWait)
	}
	return d, nil
}

// waitDeadline is how long --wait looks, and zero where the flag was not given.
//
// A wait with no cursor would hand back every thread in the document at once,
// which is the one-shot read under another name and reads to a session as news.
// The first read is the baseline and takes no --wait:
// TestAWaitWithoutACursorIsRefusedNamingTheCursor.
func waitDeadline(a *args, since *comments.Cursor) (time.Duration, error) {
	if !a.has("--wait") {
		return 0, nil
	}
	if since == nil {
		return 0, fmt.Errorf(
			"--wait needs --since: the cursor is what makes a window, and a wait without one answers with the whole document")
	}
	return parseWait("--wait", a.flags["--wait"])
}

// followDeadline is how long --follow looks, and zero where the flag was not
// given. The three refusals come before the wait's own, so a line carrying both
// flags is told which pair it wrote rather than which cursor it is missing.
//
// TestFollowRefusesWait, TestFollowRefusesWitness, TestFollowNeedsSince and
// TestFollowRefusesMoreThanAnHour.
func followDeadline(a *args, since *comments.Cursor) (time.Duration, error) {
	if !a.has("--follow") {
		return 0, nil
	}
	if a.has("--wait") {
		return 0, fmt.Errorf(
			"--follow and --wait are two answers to one question: --wait ends on the first window, --follow keeps looking and prints a line per window. Give one of them")
	}
	if a.has("--witness") {
		return 0, fmt.Errorf(
			"--follow and --witness cannot be given together: a witness is an export matched against the threads of one window, and a follow prints ids and times and ends with no threads at all")
	}
	if since == nil {
		return 0, fmt.Errorf(
			"--follow needs --since: the cursor is what makes a window, and a follow without one hands over every thread in the document")
	}
	return parseWait("--follow", a.flags["--follow"])
}

// waitedData is what one poll run did, under either flag, and every field is a
// fact. How many times it looked, how long it took, and whether the person
// stopped it. There is no field saying whether what came back was worth
// anything: that is the skill's, reading the same threads a one-shot listing
// prints.
type waitedData struct {
	Polls       int  `json:"polls"`
	Seconds     int  `json:"seconds"`
	Interrupted bool `json:"interrupted"`
}

// waitedFrom is that object from what a loop came back with, so a wait and a
// follow cannot report the same three facts two different ways.
func waitedFrom(polls int, took time.Duration, interrupted bool) *waitedData {
	return &waitedData{
		Polls: polls,
		// Rounded, because the caller reads it to know how long the document
		// was quiet and not to measure the binary.
		Seconds:     int((took + time.Second/2) / time.Second),
		Interrupted: interrupted,
	}
}

// followItem is one new comment or reply as a window line names it. Four facts:
// which entry it is, which thread it sits in, the marker its own first token
// carries, and when it was written. No author, no content and no quoted text,
// because a line is read to decide whether to look, and looking is what
// `comments` is for.
type followItem struct {
	ID      string `json:"id"`
	Thread  string `json:"thread"`
	Marker  string `json:"marker"`
	Created string `json:"created"`
}

// followLine is one window, written the moment it is found. line names the kind
// of line this is, so a reader of the stream tells a window from the envelope
// without counting: the envelope carries no such field.
type followLine struct {
	Line    string       `json:"line"`
	Since   string       `json:"since"`
	Cursor  string       `json:"cursor"`
	Noticed string       `json:"noticed"`
	Items   []followItem `json:"items"`
	More    int          `json:"more"`
}

// followLineOf is one window as its line. Noticed is in UTC, as every instant
// gdoc prints is, so two lines from two machines sort against each other.
func followLineOf(w comments.Window) followLine {
	items, more := followItemsOf(w.Items)
	return followLine{
		Line:    "window",
		Since:   w.Since.String(),
		Cursor:  w.Cursor.String(),
		Noticed: w.Noticed.UTC().Format(time.RFC3339),
		Items:   items,
		More:    more,
	}
}

// followItemsOf is the window's items up to the ceiling, and the count of the
// ones that did not fit. Items is never nil, so a line always carries a list
// and no reader has to handle an absent one.
//
// The items that woke the window go first, and the held ones fill what is left.
// Drive's own order would not do: a window carries every new item, gdoc's own
// receipts included, and a pass that answered twenty threads leaves twenty
// receipts ahead of the comment a colleague wrote after them. A ceiling taking
// the first twenty of that would name none of the work, count it in `more`, and
// hand over a cursor past it. Inside each group the order is Drive's still.
// TestALineListsAtMostTwentyItems and TestALineNamesTheItemThatWokeIt.
func followItemsOf(from []comments.NewItem) ([]followItem, int) {
	items := make([]followItem, 0, len(from))
	for _, wake := range []bool{true, false} {
		for _, it := range from {
			if len(items) == followItemCeiling {
				return items, len(from) - len(items)
			}
			if wakesTheWindow(it) != wake {
				continue
			}
			items = append(items, followItem{ID: it.ID, Thread: it.Thread, Marker: it.Marker, Created: it.Created})
		}
	}
	return items, len(from) - len(items)
}

// wakesTheWindow is the wake rule as a line reads it: an item carrying one of
// the three markers that gdoc did not write itself. internal/comments asks the
// same question of the same two fields to decide whether to hand the window
// over at all, and this is the one place in the binary that has to name what
// that answer rested on. It judges nothing: both fields are facts off the wire,
// and what the items mean is still the skill's to read.
func wakesTheWindow(it comments.NewItem) bool {
	return it.Marker != comments.MarkerNone && !it.ByGdoc
}

// followResult is the last line: the ordinary comments envelope, with the
// cursor the last window handed over and no threads at all. The windows were
// the answer, and a reader that wants the threads of one calls `comments` with
// the cursor the line carried.
//
// A failed poll is the one ending that is a failure, and the polls it made
// still go out: the skill says so, waits, and calls again with the cursor it
// already had.
func followResult(r *reach, d *docs.Document, f comments.Followed, err error) emit.Result {
	data := commentsBase(r, d, f.Cursor)
	data.Waited = waitedFrom(f.Polls, f.Waited, f.Interrupted)
	if err != nil {
		return withStop(emit.Result{OK: false, Error: err.Error(), Data: data, Warnings: r.warnings()}, err)
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings()}
}

// followList is the listing one poll of a follow makes, asked from the cursor
// the follow chose rather than from a cursor fixed for the call. It is the one
// thing the follow needs that the wait does not: a window nobody printed leaves
// the cursor where it was, so the next poll asks the same question again.
func followList(ctx context.Context, r *reach, c *comments.Cursor) ([]comments.RawComment, error) {
	return comments.Fetch(ctx, r.session, r.id, c)
}
