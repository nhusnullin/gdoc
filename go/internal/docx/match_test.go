package docx

import (
	"errors"
	"reflect"
	"testing"

	"gdoc/internal/comments"
)

var errRefused = errors.New("DOC1 was refused: it was not given to this command")

func threeThreads() []comments.Thread {
	return []comments.Thread{
		{ID: "AAAA", Author: "Nail Khusnullin", Content: "ai? which register  does this refer to\n"},
		{ID: "BBBB", Author: "Nail Khusnullin", Content: "ai! this one lost its text"},
		{ID: "CCCC", Author: "Nail Khusnullin", Content: "ai: this one is not in the export"},
	}
}

func TestMatchGivesAnchoredDetachedAndUnmatched(t *testing.T) {
	f, err := Parse(wholeExport(t))
	if err != nil {
		t.Fatal(err)
	}
	in := threeThreads()

	got := Match(in, f)

	want := []string{WitnessAnchored, WitnessDetached, WitnessUnmatched}
	for i, w := range want {
		if got[i].Witness != w {
			t.Errorf("threads[%d] (%s) witness = %q, want %q", i, got[i].ID, got[i].Witness, w)
		}
	}
	if !reflect.DeepEqual(in, threeThreads()) {
		t.Error("Match wrote into the slice it was given")
	}
	if got[0].ID != "AAAA" || got[2].ID != "CCCC" {
		t.Error("Match reordered the threads")
	}
}

func TestMatchWithoutAnExportLeavesEveryThreadUnmatched(t *testing.T) {
	got := Match(threeThreads(), nil)

	for _, th := range got {
		if th.Witness != WitnessUnmatched {
			t.Errorf("%s witness = %q, and there was no export to match it in", th.ID, th.Witness)
		}
	}
}

func TestADifferentAuthorIsNotTheSameComment(t *testing.T) {
	f, err := Parse(wholeExport(t))
	if err != nil {
		t.Fatal(err)
	}
	in := []comments.Thread{{ID: "AAAA", Author: "Someone Else", Content: "ai? which register does this refer to"}}

	got := Match(in, f)

	if got[0].Witness != WitnessUnmatched {
		t.Errorf("witness = %q; the same words from another author are another comment", got[0].Witness)
	}
}

func TestTwoThreadsWithTheSameWordsTakeTwoExportedComments(t *testing.T) {
	f := &File{Comments: []Comment{
		{ID: "0", Author: "Nail Khusnullin", Text: "ai: same words", Anchored: true, Span: "one"},
		{ID: "1", Author: "Nail Khusnullin", Text: "ai: same words", Anchored: true, Span: "two"},
	}}
	in := []comments.Thread{
		{ID: "AAAA", Author: "Nail Khusnullin", Content: "ai: same words"},
		{ID: "BBBB", Author: "Nail Khusnullin", Content: "ai: same words"},
		{ID: "CCCC", Author: "Nail Khusnullin", Content: "ai: same words"},
	}

	got := Match(in, f)

	if got[0].Witness != WitnessAnchored {
		t.Errorf("threads[0] witness = %q, want anchored", got[0].Witness)
	}
	if got[1].Witness != WitnessAnchored {
		t.Errorf("threads[1] witness = %q; one exported comment answers for one thread", got[1].Witness)
	}
	// The third thread carries the same words and the export holds two such
	// comments, both already answering for another thread. Its truthful witness
	// is unmatched: it was created after the export. Handing it an exported
	// comment that has answered already would be a witness for something the
	// export does not hold, in the field the skill judges from.
	if got[2].Witness != WitnessUnmatched {
		t.Errorf("threads[2] witness = %q; the export holds two such comments and both are spoken for", got[2].Witness)
	}
}

// Two exported comments carrying the same words from the same author, one
// anchored and one not, say nothing about which Drive thread is which. Neither
// side is ordered against the other, so first-fit would hand one thread id the
// other one's witness, and detached names the wrong sentence when it does.
func TestAnAmbiguousWitnessIsUnmatchedRatherThanGuessed(t *testing.T) {
	f := &File{Comments: []Comment{
		{ID: "0", Author: "Nail Khusnullin", Text: "ai: same words", Anchored: true, Span: "one"},
		{ID: "1", Author: "Nail Khusnullin", Text: "ai: same words"},
	}}
	in := []comments.Thread{
		{ID: "AAAA", Author: "Nail Khusnullin", Content: "ai: same words"},
		{ID: "BBBB", Author: "Nail Khusnullin", Content: "ai: same words"},
	}

	got := Match(in, f)

	for i, th := range got {
		if th.Witness != WitnessUnmatched {
			t.Errorf("threads[%d] witness = %q, want unmatched: the two exported comments disagree", i, th.Witness)
		}
	}
}

// The same ambiguity with one thread in view, which is what --since produces:
// the listing is narrowed to the cursor window and the export is not.
func TestOneThreadWithTwoDisagreeingCandidatesIsUnmatched(t *testing.T) {
	f := &File{Comments: []Comment{
		{ID: "0", Author: "Nail Khusnullin", Text: "ai: same words", Anchored: true, Span: "one"},
		{ID: "1", Author: "Nail Khusnullin", Text: "ai: same words"},
	}}
	in := []comments.Thread{{ID: "BBBB", Author: "Nail Khusnullin", Content: "ai: same words"}}

	got := Match(in, f)

	if got[0].Witness != WitnessUnmatched {
		t.Errorf("witness = %q, want unmatched: nothing says which of the two this thread is", got[0].Witness)
	}
}

func TestMatchOfNoThreadsIsNoThreads(t *testing.T) {
	if got := Match(nil, &File{}); len(got) != 0 {
		t.Errorf("Match(nil) = %+v, want none", got)
	}
}
