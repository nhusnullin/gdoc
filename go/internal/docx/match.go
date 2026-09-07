package docx

import (
	"strings"

	"gdoc/internal/comments"
)

// The three answers the witness gives. They are facts about the export, not
// judgements about the thread: what a detached comment means, and whether it
// still needs an answer, is the skill's to decide.
const (
	// WitnessAnchored: the export carries this comment, attached to text.
	WitnessAnchored = "anchored"
	// WitnessDetached: the export carries this comment and no range for it.
	// The text it was written about is gone.
	WitnessDetached = "detached"
	// WitnessUnmatched: the export carries no comment with these words. A
	// comment made after the export, or one the match could not join.
	WitnessUnmatched = "unmatched"
)

// Match sets Witness on a copy of every thread and returns the copies in the
// order it was given them. The input slice is not written to: a caller that
// asked for the witness still has the threads it read.
//
// The join is on the comment's own words, because the docx has no Drive
// comment id in it: w:id is the export's own numbering. Whitespace is
// normalised, since the export writes a comment's runs back with its own line
// breaks, and the author's display name has to agree when both sides carry one.
// The same words from two people are two comments.
//
// Replies are not matched. The witness question is whether the thread is
// attached to text, and the first comment is the thread.
//
// A nil file is every thread unmatched rather than a crash: a run whose export
// failed still reports its threads.
func Match(threads []comments.Thread, f *File) []comments.Thread {
	out := make([]comments.Thread, 0, len(threads))
	var used []bool
	if f != nil {
		used = make([]bool, len(f.Comments))
	}
	for _, t := range threads {
		witnessed := t
		witnessed.Witness = WitnessUnmatched
		if f != nil {
			if i := find(f.Comments, used, t); i >= 0 {
				used[i] = true
				witnessed.Witness = WitnessDetached
				if f.Comments[i].Anchored {
					witnessed.Witness = WitnessAnchored
				}
			}
		}
		out = append(out, witnessed)
	}
	return out
}

// find is the first exported comment that matches and has not answered for
// another thread already. One exported comment answers for one thread, so two
// threads carrying the same words take two of them rather than both taking the
// first.
//
// When more than one unused comment matches and they do not agree on whether
// they are anchored, no answer is given and the thread comes back unmatched.
// The join is on words and an author name, and nothing orders the two sides
// against each other: Drive's comments.list defines no ordering and
// word/comments.xml is numbered by the export, so first-fit would hand one of
// the two thread ids a witness belonging to the other. --since makes that
// reachable with one thread in view, because the listing is narrowed to the
// cursor window and the export is not. detached means the text this thread was
// written about is gone, so pinning it on the wrong id points the skill at the
// wrong sentence. Refusing to guess is the same answer this milestone gives
// everywhere else it cannot stand behind a fact.
func find(cs []Comment, used []bool, t comments.Thread) int {
	want := normalise(t.Content)
	first := -1
	for i, c := range cs {
		if used[i] || normalise(c.Text) != want || !authorsAgree(c.Author, t.Author) {
			continue
		}
		if first < 0 {
			first = i
			continue
		}
		if c.Anchored != cs[first].Anchored {
			return -1
		}
	}
	return first
}

// authorsAgree is true when both sides name an author and it is the same one,
// or when either side names none. An export that carries no display name is
// still a witness to the comment being there.
func authorsAgree(a, b string) bool {
	return a == "" || b == "" || a == b
}

// normalise collapses every run of whitespace to one space and trims the ends.
// Drive answers a comment as the author typed it; the export writes the same
// comment back with its own paragraph breaks, and the words are what match.
func normalise(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
