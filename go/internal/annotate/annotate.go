// This file is the annotation itself: Annotation, the shape rule asked of every
// entry before the first one leaves the machine, and Body, the one place the
// robot is put in front of the words. doc.go holds the package comment.

package annotate

import (
	"errors"
	"fmt"
	"strings"

	"gdoc/internal/plaintext"
)

// Prefix is what every comment gdoc writes opens with. It is the same mark
// reply requires on a reply and propose puts on a reason, and it is the only
// record of authorship there is: the Docs API cannot set an author, so
// everything gdoc writes is signed by whoever is logged in.
const Prefix = plaintext.Prefix

// Robot is the mark without its space, which is what Check asks about. See
// plaintext.Robot for why the check is not the whole prefix.
const Robot = plaintext.Robot

// Annotation is one comment, as the skill hands it over: the words to comment
// on, why, and optionally who should answer for it.
//
// Quoted is text, never an index. An index computed from an earlier read is the
// hazard the whole API has, so the placement is made of words and looked up in
// a document that has just come back.
type Annotation struct {
	Quoted   string `json:"quoted"`
	Why      string `json:"why"`
	Assignee string `json:"assignee,omitempty"`
}

// Check is the shape rule for one annotation, as shape rather than as meaning.
// It answers from the annotation alone, so the caller can ask it of every entry
// in the file before the first one leaves the machine: a third entry refused
// after the first two have landed is a run that half happened.
//
// Nothing here says whether the comment is worth making. The words arrive
// written and the placement arrives quoted, and this package writes what it was
// told.
func (a Annotation) Check() error {
	if a.Quoted == "" {
		return errors.New("the annotation quotes no text, so there is nothing to comment on")
	}
	// The quote is looked up by propose.FindSpan, which walks one paragraph at
	// a time, so a quote carrying a break is in no single paragraph and could
	// only ever come back as "not found". Refusing it here names the real
	// mistake, and it names it before the document is read.
	if strings.ContainsAny(a.Quoted, "\n\r") {
		return fmt.Errorf("the quoted text %q carries a line break; a comment is anchored inside one paragraph, and the words without the break are always quotable instead", a.Quoted)
	}
	// The reason is read with its surrounding space taken off, which is the
	// shape propose.Check and reply.Check both read a body in. A reason of
	// nothing but spaces is a comment that is a bare signature.
	why := strings.TrimSpace(a.Why)
	if why == "" {
		return errors.New("the annotation carries no reason, and a comment that says nothing is one Nail has to guess at")
	}
	// Body writes the comment as Prefix + Why, so a reason that already opens
	// with the robot lands as two of them. reply.Check requires the prefix and
	// this one refuses it, which is the two writers disagreeing about who owns
	// the marker: a caller carrying the reply convention over would sign the
	// comment twice, and nothing downstream would catch it. The read-backs
	// compare against the string that was sent, so both of them hold over a
	// doubled mark in the one string that records who wrote the comment.
	//
	// The test is the robot itself rather than the robot and its space, because
	// every near miss lands the same doubled mark: the robot with nothing
	// behind it, a robot behind a newline, a robot behind a leading space.
	if strings.HasPrefix(why, Robot) {
		return fmt.Errorf("the reason for the annotation already opens with %q, and gdoc adds it; write the reason without it", Prefix)
	}
	// The reason is written into a comment thread, where Docs renders markdown
	// literally. It is the same rule reply.Check and propose.Check hold, and it
	// is asked of the reason alone rather than of Prefix + Why: the heading arm
	// is anchored to a line start, so the prefix in front of it would move the
	// first character off offset zero and let a reason opening with "# "
	// through.
	if m := plaintext.Markdown(a.Why); m != "" {
		return fmt.Errorf("the reason for the annotation carries markdown (%q); a Docs thread renders it literally, so it would arrive as typed", m)
	}
	return nil
}

// Body is what the comment holds: the robot, one space, and the reason as it
// was given. It is the one place the mark is added, so a later reader asking
// what gdoc wrote is asking about one string built in one room.
func Body(why string) string {
	return Prefix + why
}
