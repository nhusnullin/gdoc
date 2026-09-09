package restyle

import (
	"encoding/json"
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

func TestNothingToProtectIsTrueOnlyWhenAllFiveCountsAreZero(t *testing.T) {
	// Arrange
	empty := doc(para(1, text(1, "Nothing here.\n")))
	chip := doc(para(1, docs.Run{Kind: docs.KindPerson, StartIndex: 1, EndIndex: 2,
		Detail: &docs.Detail{ID: "kix.p1", Label: "A Placeholder"}}))
	suggested := doc(para(1, docs.Run{Kind: docs.KindText, Text: "new words", StartIndex: 1, EndIndex: 10,
		InsertionIDs: []string{"suggest.one"}}))
	commented := doc(para(1, text(1, "Hello.\n")))
	commented.CommentRanges = map[string]docs.Range{"c1": {Tab: "t.0", Start: 1, End: 6}}
	unnamed := doc(para(1, docs.Run{Kind: docs.KindUnknown, StartIndex: 1, EndIndex: 2,
		Detail: &docs.Detail{Member: "tomorrowsElement"}}))
	// A pending suggestion carried only by a run that holds no text. The
	// pending walk reads text runs alone, so this one reaches no listing, and
	// answering "nothing to protect" over it would be the one false fact in the
	// field a later run reads before it writes.
	onElement := doc(para(1, docs.Run{Kind: docs.KindPageBreak, StartIndex: 1, EndIndex: 2,
		InsertionIDs: []string{"suggest.break"}}))
	// A named range is a label Docs keeps in step with its own edits, so a
	// replacement of the words it covers takes it with them. The survey lists
	// them because M7b has to check them, and this is the field M7b reads
	// before it writes.
	named := doc(para(1, text(1, "Hello.\n")))
	named.Tabs[0].NamedRanges = []docs.NamedRange{{ID: "kix.nr1", Name: "owner", Tab: "t.0",
		Ranges: []docs.Range{{Tab: "t.0", Start: 1, End: 6}}}}

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
		{"an element the read cannot name", Input{Document: unnamed}, false},
		{"a suggestion on an element that holds no text", Input{Document: onElement}, false},
		{"a named range", Input{Document: named}, false},
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

// A suggestion the pending listing cannot report is counted, and it is counted
// apart from that listing rather than folded into it. suggestions.All reads
// text runs alone, on purpose, so a suggested page break, a suggested chip or a
// suggested footnote reference is in no listing this binary prints. Counting it
// nowhere answered "nothing to protect" over a pending change, which is the one
// false fact in the field M7b reads before it overwrites.
//
// The footnote reference is here because it is the case the rule is named
// after. It carries its number as text, so "the run holds no text" would leave
// it out, and the pending walk skips it all the same.
func TestASuggestionTheListingCannotReportIsCounted(t *testing.T) {
	// Arrange
	d := doc(para(1,
		docs.Run{Kind: docs.KindPageBreak, StartIndex: 1, EndIndex: 2,
			InsertionIDs: []string{"suggest.break"}},
		docs.Run{Kind: docs.KindPerson, StartIndex: 2, EndIndex: 3,
			Detail: &docs.Detail{ID: "kix.p1", Label: "A Placeholder"}, DeletionIDs: []string{"suggest.chip"}},
		docs.Run{Kind: docs.KindFootnoteRef, Text: "1", StartIndex: 3, EndIndex: 4,
			FootnoteID: "kix.f1", InsertionIDs: []string{"suggest.footnote"}}))

	// Act
	got, warnings := Survey(Input{Document: d})

	// Assert
	if got.Suggestions.Pending != 0 {
		t.Errorf("pending: got %d, want 0, because the pending walk reads text runs alone", got.Suggestions.Pending)
	}
	if got.Suggestions.OnElements != 3 {
		t.Errorf("on_elements: got %d, want 3", got.Suggestions.OnElements)
	}
	if got.NothingToProtect {
		t.Error("nothing_to_protect: got true over three pending suggestions, want false")
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "the suggestions listing cannot report") {
		t.Errorf("warnings: got %v, want one naming the suggestions that are counted and not listed", warnings)
	}
}

// An id the pending walk already saw is not counted twice. A chip inside a
// pending insertion carries the same id as the text either side of it, and it
// is one suggestion.
func TestAnIDThePendingWalkSawIsNotCountedOnElementsAsWell(t *testing.T) {
	// Arrange
	d := doc(para(1,
		docs.Run{Kind: docs.KindText, Text: "signed by ", StartIndex: 1, EndIndex: 11,
			InsertionIDs: []string{"suggest.one"}},
		docs.Run{Kind: docs.KindPerson, StartIndex: 11, EndIndex: 12,
			Detail: &docs.Detail{ID: "kix.p1", Label: "A Placeholder"}, InsertionIDs: []string{"suggest.one"}}))

	// Act
	got, _ := Survey(Input{Document: d})

	// Assert
	if got.Suggestions.Pending != 1 {
		t.Errorf("pending: got %d, want 1", got.Suggestions.Pending)
	}
	if got.Suggestions.OnElements != 0 {
		t.Errorf("on_elements: got %d, want 0, because the pending walk already saw that id", got.Suggestions.OnElements)
	}
}

// named_ranges is a list on every document, never null. encoding/json writes a
// nil slice as null, and a skill reading the field must not get two shapes for
// the one fact that a document has no named ranges.
func TestNamedRangesIsAlwaysAList(t *testing.T) {
	// Arrange
	d := doc(para(1, text(1, "Nothing here.\n")))

	// Act
	got, _ := Survey(Input{Document: d})
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if got.NamedRanges == nil {
		t.Error("named_ranges: got nil, want an empty list")
	}
	if !strings.Contains(string(b), `"named_ranges":[]`) {
		t.Errorf("json = %s, want named_ranges as an empty list", b)
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

// A survey with no document read knows nothing about where a comment sits, and
// saying the Docs read placed no range for it names a read that was never made.
// The one warning about the missing document is the whole of what is true.
func TestASurveyWithNoDocumentBlamesNoDocsRead(t *testing.T) {
	// Act
	got, warnings := Survey(Input{Comments: []comments.RawComment{
		{ID: "c1", Content: "ai? what about this"},
		{ID: "c2", Content: "settled"},
	}})

	// Assert
	if got.Threads.Open != 2 {
		t.Errorf("threads.open: got %d, want 2, the survey still carries its threads", got.Threads.Open)
	}
	if warned(warnings, "placed no range") {
		t.Errorf("warnings: got %v, want none blaming a Docs read that never happened", warnings)
	}
	if !warned(warnings, "no document") {
		t.Errorf("warnings: got %v, want one saying no document was read", warnings)
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

// A suggestion on an element gdoc has never seen is counted like any other one
// the listing cannot report. The count is the survey's own, so it is only as
// good as what the decoder hands it: an arm that kept the member name and threw
// the ids away answered "nothing pending on an element" over a document with a
// pending change in it, and the warning naming the member said nothing about
// somebody waiting on it. This test reads the JSON rather than building runs by
// hand, because the decoder is the half that was wrong.
func TestASuggestionOnAnElementGdocCannotNameIsCounted(t *testing.T) {
	// Arrange
	raw := `{"documentId":"DOC1","title":"Supplier Register Policy","revisionId":"rev1",
		"body":{"content":[
		{"startIndex":1,"endIndex":6,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[
		   {"startIndex":1,"endIndex":5,"textRun":{"content":"See "}},
		   {"startIndex":5,"endIndex":6,"tomorrowsElement":{"suggestedInsertionIds":["suggest.x1"]}}]}}]}}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}

	// Act
	got, warnings := Survey(Input{Document: d})

	// Assert
	if got.Suggestions.OnElements != 1 {
		t.Errorf("on_elements: got %d, want 1", got.Suggestions.OnElements)
	}
	if got.NothingToProtect {
		t.Error("nothing_to_protect: got true over a pending suggestion, want false")
	}
	var named, counted bool
	for _, w := range warnings {
		if strings.Contains(w, "tomorrowsElement") {
			named = true
		}
		if strings.Contains(w, "the suggestions listing cannot report") {
			counted = true
		}
	}
	if !named || !counted {
		t.Errorf("warnings: got %v, want one naming the member and one naming the uncounted suggestion", warnings)
	}
}
