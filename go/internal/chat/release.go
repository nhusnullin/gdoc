// The release: the words a call has to send back to release a held write, and
// the silence it needs behind it.

package chat

import (
	"fmt"
	"time"
)

// quietGap is the silence a release needs behind it: no tool call in this
// session for this long before the one that sends a held write.
//
// Measurement 11 of docs/v2/MEASURED.md is where the number comes from. The
// server sees a release only after the person has approved the card, and the two
// people measured took 95 and 121 seconds over it, so five seconds costs an
// honest release nothing. What it does cost is the one shape that is not honest:
// a model that releases a write in the same breath as the call that was held,
// which is what both runs of the measurement showed it doing, unasked and when
// asked not to. Only an ended turn makes a gap this long.
const quietGap = 5 * time.Second

// Card is the words the person read before they approved, as the releasing call
// sends them back.
//
// Every field is compared byte for byte, so what the person saw on the card is
// what the hold said and never what a model wrote about it:
// TestTheCardsWordsArePinnedToTheByte.
type Card struct {
	Hold   string
	Title  string
	Reason string
	Text   string
}

// Card is the one card that releases this hold.
func (h Hold) Card() Card {
	return Card{Hold: h.ID, Title: h.Title, Reason: h.Reason, Text: h.Text}
}

// Release judges the call that would send this held write. It answers nothing
// where the write may go, and the sentence a model reads otherwise.
//
// The hold is kept on either refusal, because both are answerable: one by
// sending the card's own words, and the other by waiting. lastCall is the instant
// the session's previous tool call ended, and the clock is handed in, as it is
// everywhere else here.
func (h Hold) Release(got Card, lastCall, now time.Time) error {
	if err := h.words(got); err != nil {
		return err
	}
	// Whichever is later: a session that recorded no call at all still cannot
	// release a hold it made a moment ago.
	since := h.Created
	if lastCall.After(since) {
		since = lastCall
	}
	if now.Sub(since) < quietGap {
		return fmt.Errorf(
			"this approval came less than %d seconds after the last call in this session, "+
				"so gdoc kept the hold and sent nothing. End your turn, "+
				"and ask the person to approve it again in a moment",
			int(quietGap/time.Second))
	}
	return nil
}

// words is the card the call sent against the card the hold makes, field by
// field. The field that differs is named, because the words to send are in the
// held answer and saying which one is wrong is a fact about the call.
func (h Hold) words(got Card) error {
	want := h.Card()
	for _, f := range []struct {
		name string
		want string
		got  string
	}{
		{"hold", want.Hold, got.Hold},
		{"title", want.Title, got.Title},
		{"reason", want.Reason, got.Reason},
		{"text", want.Text, got.Text},
	} {
		if f.got == f.want {
			continue
		}
		return fmt.Errorf(
			"%s is not what the person approved, so gdoc kept the hold and sent nothing: "+
				"the card says %q and this call said %q", f.name, f.want, f.got)
	}
	return nil
}
