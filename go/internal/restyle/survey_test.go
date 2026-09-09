package restyle

import (
	"errors"
	"os"
	"strings"
	"testing"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/docx"
)

// doc is a one tab document with the blocks a test hands it, so a test states
// the document it is surveying rather than a fixture path.
func doc(blocks ...docs.Block) *docs.Document {
	return &docs.Document{
		ID:         "DOC1",
		Title:      "Supplier Register Policy",
		RevisionID: "ALm37BXrev1",
		Tabs:       []docs.Tab{{ID: "t.0", Body: blocks}},
	}
}

// para is one paragraph of the runs given, indexed from start.
func para(start int, runs ...docs.Run) docs.Block {
	end := start
	if len(runs) > 0 {
		end = runs[len(runs)-1].EndIndex
	}
	return docs.Block{Paragraph: &docs.Paragraph{
		Style: "NORMAL_TEXT", Runs: runs, StartIndex: start, EndIndex: end,
	}}
}

func text(start int, s string) docs.Run {
	return docs.Run{Kind: docs.KindText, Text: s, StartIndex: start, EndIndex: start + len(s)}
}

func TestTheSurveyCountsWhatTheDocumentHolds(t *testing.T) {
	// Arrange
	d := doc(para(1, text(1, "Hello.\n")))
	d.CommentRanges = map[string]docs.Range{
		"c1": {Tab: "t.0", Start: 1, End: 6},
		"c2": {Tab: "t.0", Start: 1, End: 6},
	}
	raw := []comments.RawComment{
		{ID: "c1", Content: "ai? what about this", Author: comments.Author{DisplayName: "Nail"},
			QuotedFileContent: &comments.Quoted{Value: "Hello"}},
		{ID: "c2", Content: "settled", Resolved: true, Author: comments.Author{DisplayName: "Nail"},
			QuotedFileContent: &comments.Quoted{Value: "Hello"}},
	}
	export := &docx.File{Comments: []docx.Comment{
		{Author: "Nail", Text: "ai? what about this", Anchored: true},
		{Author: "Nail", Text: "settled", Anchored: false},
	}}

	// Act
	got, warnings := Survey(Input{Document: d, Comments: raw, Export: export})

	// Assert
	if got.DocumentID != "DOC1" || got.Title != "Supplier Register Policy" {
		t.Errorf("document id and title: got %q and %q, want DOC1 and Supplier Register Policy",
			got.DocumentID, got.Title)
	}
	if got.RevisionID != "ALm37BXrev1" {
		t.Errorf("revision id: got %q, want ALm37BXrev1", got.RevisionID)
	}
	if got.Tabs != 1 {
		t.Errorf("tabs: got %d, want 1", got.Tabs)
	}
	if got.Threads.Open != 1 || got.Threads.Resolved != 1 {
		t.Errorf("threads: got %d open and %d resolved, want 1 and 1", got.Threads.Open, got.Threads.Resolved)
	}
	if got.Threads.Witness.Anchored != 1 || got.Threads.Witness.Detached != 1 || got.Threads.Witness.Unmatched != 0 {
		t.Errorf("witness counts: got %+v, want 1 anchored, 1 detached, 0 unmatched", got.Threads.Witness)
	}
	if len(got.Threads.Witnessed) != 2 {
		t.Fatalf("per-thread witness: got %d entries, want 2", len(got.Threads.Witnessed))
	}
	if got.Threads.Witnessed[0].ID != "c1" || got.Threads.Witnessed[0].Witness != docx.WitnessAnchored {
		t.Errorf("first thread: got %+v, want c1 anchored", got.Threads.Witnessed[0])
	}
	if got.Threads.Witnessed[1].ID != "c2" || got.Threads.Witnessed[1].Witness != docx.WitnessDetached {
		t.Errorf("second thread: got %+v, want c2 detached", got.Threads.Witnessed[1])
	}
	if got.NothingToProtect {
		t.Error("nothing_to_protect: got true on a document carrying two threads, want false")
	}
	if len(warnings) != 0 {
		t.Errorf("warnings: got %v, want none", warnings)
	}
}

func TestNothingToProtectIsTrueOnlyWhenAllThreeCountsAreZero(t *testing.T) {
	// Arrange
	empty := doc(para(1, text(1, "Nothing here.\n")))
	chip := doc(para(1, docs.Run{Kind: docs.KindPerson, StartIndex: 1, EndIndex: 2,
		Detail: &docs.Detail{ID: "kix.p1", Label: "A Placeholder"}}))
	suggested := doc(para(1, docs.Run{Kind: docs.KindText, Text: "new words", StartIndex: 1, EndIndex: 10,
		InsertionIDs: []string{"suggest.one"}}))
	commented := doc(para(1, text(1, "Hello.\n")))
	commented.CommentRanges = map[string]docs.Range{"c1": {Tab: "t.0", Start: 1, End: 6}}

	cases := []struct {
		name string
		in   Input
		want bool
	}{
		{"nothing at all", Input{Document: empty}, true},
		{"a chip", Input{Document: chip}, false},
		{"a pending suggestion", Input{Document: suggested}, false},
		{"a thread", Input{Document: commented,
			Comments: []comments.RawComment{{ID: "c1", Content: "hi"}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Act
			got, _ := Survey(c.in)

			// Assert
			if got.NothingToProtect != c.want {
				t.Errorf("nothing_to_protect on %s: got %v, want %v", c.name, got.NothingToProtect, c.want)
			}
		})
	}
}

func TestThePendingCountComesFromAllAndNotFromList(t *testing.T) {
	// Arrange: a suggestion whose text is only whitespace. List drops it and
	// All keeps it, and a replacement would destroy it either way.
	d := doc(para(1, docs.Run{Kind: docs.KindText, Text: "   ", StartIndex: 1, EndIndex: 4,
		InsertionIDs: []string{"suggest.blank"}}))

	// Act
	got, _ := Survey(Input{Document: d})

	// Assert
	if got.Suggestions.Pending != 1 {
		t.Errorf("pending: got %d, want 1, so the count is All's and not List's", got.Suggestions.Pending)
	}
	if got.NothingToProtect {
		t.Error("nothing_to_protect: got true on a document carrying a whitespace-only suggestion, want false")
	}
}

func TestTheChipsAreCountedByKindAcrossTablesToo(t *testing.T) {
	// Arrange
	d := doc(
		para(1, docs.Run{Kind: docs.KindPerson, StartIndex: 1, EndIndex: 2},
			docs.Run{Kind: docs.KindDate, StartIndex: 2, EndIndex: 3}),
		docs.Block{Table: docs.Table{{{Blocks: []docs.Block{
			para(4, docs.Run{Kind: docs.KindRichLink, StartIndex: 4, EndIndex: 5},
				docs.Run{Kind: docs.KindPerson, StartIndex: 5, EndIndex: 6}),
		}}}}},
	)

	// Act
	got, _ := Survey(Input{Document: d})

	// Assert
	if got.Chips.Person != 2 || got.Chips.Date != 1 || got.Chips.RichLink != 1 {
		t.Errorf("chips: got %+v, want 2 person, 1 date, 1 rich link", got.Chips)
	}
}

func TestTheChipsInTheFixtureAreCounted(t *testing.T) {
	// Arrange
	b, err := os.ReadFile("../docs/testdata/elements.json")
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	d, err := docs.Parse(b)
	if err != nil {
		t.Fatalf("parsing the fixture: %v", err)
	}

	// Act
	got, _ := Survey(Input{Document: d})

	// Assert
	if got.Chips.Person != 1 || got.Chips.Date != 1 || got.Chips.RichLink != 1 {
		t.Errorf("chips of elements.json: got %+v, want one of each", got.Chips)
	}
}

func TestTheNamedRangesAreCarriedWithTheirIDs(t *testing.T) {
	// Arrange
	d := doc(para(1, text(1, "Hello.\n")))
	d.Tabs[0].NamedRanges = []docs.NamedRange{
		{ID: "kix.abc123", Name: "gdoc-checklist", Tab: "t.0",
			Ranges: []docs.Range{{Tab: "t.0", Start: 1, End: 6}}},
		{ID: "kix.def456", Name: "gdoc-checklist", Tab: "t.0"},
	}

	// Act
	got, _ := Survey(Input{Document: d})

	// Assert
	if len(got.NamedRanges) != 2 {
		t.Fatalf("named ranges: got %d, want 2, so two ranges sharing a name both survive", len(got.NamedRanges))
	}
	if got.NamedRanges[0].ID != "kix.abc123" || got.NamedRanges[1].ID != "kix.def456" {
		t.Errorf("named range ids: got %q and %q, want kix.abc123 and kix.def456",
			got.NamedRanges[0].ID, got.NamedRanges[1].ID)
	}
}

func TestAThreadWithNoRangeIsSurveyedWithAWarning(t *testing.T) {
	// Arrange: the Docs read placed no range for c1.
	d := doc(para(1, text(1, "Hello.\n")))
	raw := []comments.RawComment{{ID: "c1", Content: "ai? unplaced"}}

	// Act
	got, warnings := Survey(Input{Document: d, Comments: raw})

	// Assert
	if got.Threads.Open != 1 {
		t.Errorf("threads: got %d open, want 1, so an unplaced thread is surveyed rather than dropped", got.Threads.Open)
	}
	if !warned(warnings, "c1") {
		t.Errorf("warnings: got %v, want one naming c1", warnings)
	}
}

func TestAnExportThatCouldNotBeReadLeavesEveryThreadUnmatched(t *testing.T) {
	// Arrange
	d := doc(para(1, text(1, "Hello.\n")))
	d.CommentRanges = map[string]docs.Range{"c1": {Tab: "t.0", Start: 1, End: 6}}
	raw := []comments.RawComment{{ID: "c1", Content: "ai? still here",
		Author: comments.Author{DisplayName: "Nail"}}}

	// Act
	got, warnings := Survey(Input{Document: d, Comments: raw,
		ExportErr: errors.New("the export is not a docx")})

	// Assert
	if got.Threads.Witness.Unmatched != 1 || got.Threads.Witness.Anchored != 0 {
		t.Errorf("witness counts: got %+v, want every thread unmatched", got.Threads.Witness)
	}
	if got.Threads.Witnessed[0].Witness != docx.WitnessUnmatched {
		t.Errorf("per-thread witness: got %q, want %q", got.Threads.Witnessed[0].Witness, docx.WitnessUnmatched)
	}
	if !warned(warnings, "the export is not a docx") {
		t.Errorf("warnings: got %v, want one carrying the export failure", warnings)
	}
}

func TestAnElementTheReadCannotNameIsWarnedAbout(t *testing.T) {
	// Arrange
	d := doc(para(1, docs.Run{Kind: docs.KindUnknown, StartIndex: 1, EndIndex: 2,
		Detail: &docs.Detail{Member: "somethingNew"}}))

	// Act
	got, warnings := Survey(Input{Document: d})

	// Assert
	if !warned(warnings, "somethingNew") {
		t.Errorf("warnings: got %v, want one naming the element this read cannot name", warnings)
	}
	if got.NothingToProtect {
		t.Error("nothing_to_protect: got true on a document holding an element gdoc cannot name, want false")
	}
}

func TestASurveyWithNoDocumentSaysSoRatherThanReportingNothingToProtect(t *testing.T) {
	// Act
	got, warnings := Survey(Input{})

	// Assert
	if got.NothingToProtect {
		t.Error("nothing_to_protect: got true with no document read at all, want false")
	}
	if !warned(warnings, "no document") {
		t.Errorf("warnings: got %v, want one saying no document was read", warnings)
	}
}

// warned says whether any warning carries the words.
func warned(warnings []string, words string) bool {
	for _, w := range warnings {
		if strings.Contains(w, words) {
			return true
		}
	}
	return false
}
