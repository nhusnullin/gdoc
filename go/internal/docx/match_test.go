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
		{ID: "1", Author: "Nail Khusnullin", Text: "ai: same words"},
	}}
	in := []comments.Thread{
		{ID: "AAAA", Author: "Nail Khusnullin", Content: "ai: same words"},
		{ID: "BBBB", Author: "Nail Khusnullin", Content: "ai: same words"},
	}

	got := Match(in, f)

	if got[0].Witness != WitnessAnchored {
		t.Errorf("threads[0] witness = %q, want anchored", got[0].Witness)
	}
	if got[1].Witness != WitnessDetached {
		t.Errorf("threads[1] witness = %q; one exported comment answers for one thread", got[1].Witness)
	}
}

func TestMatchOfNoThreadsIsNoThreads(t *testing.T) {
	if got := Match(nil, &File{}); len(got) != 0 {
		t.Errorf("Match(nil) = %+v, want none", got)
	}
}
