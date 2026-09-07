package suggestions

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/frontmatter"
)

const testDocumentID = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"

// fixture parses one of internal/docs's testdata files, so the walk is tested
// against the same documents the tree and the text projection are.
func fixture(t *testing.T, name string) *docs.Document {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "docs", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("docs.Parse(%s): %v", name, err)
	}
	return d
}

// document is one inline Docs response, for a rule that needs a shape no
// fixture carries.
func document(t *testing.T, raw string) *docs.Document {
	t.Helper()
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// text is a run carrying the two id lists, which is the only run shape a
// suggestion is read from.
func text(s string, insertions, deletions []string) docs.Run {
	return docs.Run{Kind: docs.KindText, Text: s, InsertionIDs: insertions, DeletionIDs: deletions}
}

func para(style string, runs ...docs.Run) docs.Block {
	return docs.Block{Paragraph: &docs.Paragraph{Style: style, Runs: runs}}
}

func doc(blocks ...docs.Block) *docs.Document {
	return &docs.Document{ID: testDocumentID, Tabs: []docs.Tab{{ID: "t.0", Body: blocks}}}
}

// TestTheFixtureReadsAsOneReplacement is the whole walk over a real read: one
// suggestion id carried by a deletion run and by two insertion runs comes back
// as two pendings, one per kind, with the insertions joined.
func TestTheFixtureReadsAsOneReplacement(t *testing.T) {
	got := List(fixture(t, "single-tab.json"))
	want := []Pending{
		{ID: "suggest.a1", Kind: frontmatter.KindDeletion, Section: "Scope", Text: "annually"},
		{ID: "suggest.a1", Kind: frontmatter.KindInsertion, Section: "Scope", Text: "every quarter"},
	}
	assertPending(t, got, want)
}

// TestRunsSharingOneIDJoin is the reason the join exists: Docs cuts a typed
// sentence at every formatting boundary, so one edit arrives as several runs.
func TestRunsSharingOneIDJoin(t *testing.T) {
	d := doc(
		para("HEADING_1", text("Scope\n", nil, nil)),
		para("NORMAL_TEXT",
			text("every ", []string{"suggest.a1"}, nil),
			text("quarter", []string{"suggest.a1"}, nil),
		),
	)
	assertPending(t, List(d), []Pending{
		{ID: "suggest.a1", Kind: frontmatter.KindInsertion, Section: "Scope", Text: "every quarter"},
	})
}

// TestARunWithTwoIDsIsTwoPendings is where the port leaves v1 behind. v1 took
// ids[0] and lost the rest; every id a run carries is a suggestion.
func TestARunWithTwoIDsIsTwoPendings(t *testing.T) {
	d := doc(para("NORMAL_TEXT", text("critical ", []string{"suggest.a1", "suggest.b2"}, nil)))
	assertPending(t, List(d), []Pending{
		{ID: "suggest.a1", Kind: frontmatter.KindInsertion, Text: "critical "},
		{ID: "suggest.b2", Kind: frontmatter.KindInsertion, Text: "critical "},
	})
}

// TestARunSuggestedBothWaysIsADeletionThenAnInsertion follows what Docs shows
// and what the read text prints, so the two commands agree about one run.
func TestARunSuggestedBothWaysIsADeletionThenAnInsertion(t *testing.T) {
	d := doc(para("NORMAL_TEXT", text("annually", []string{"suggest.a1"}, []string{"suggest.a1"})))
	assertPending(t, List(d), []Pending{
		{ID: "suggest.a1", Kind: frontmatter.KindDeletion, Text: "annually"},
		{ID: "suggest.a1", Kind: frontmatter.KindInsertion, Text: "annually"},
	})
}

// TestTheWalkGoesIntoATableAndCarriesTheHeading: a suggested fee in a table is
// exactly the edit that must not be missed, and the heading above the table is
// still the heading above it.
func TestTheWalkGoesIntoATableAndCarriesTheHeading(t *testing.T) {
	cell := docs.Cell{Blocks: []docs.Block{
		para("NORMAL_TEXT", text("2.5%", []string{"suggest.fee"}, nil)),
	}}
	d := doc(
		para("HEADING_2", text("Fees\n", nil, nil)),
		docs.Block{Table: docs.Table{{docs.Cell{Blocks: []docs.Block{para("NORMAL_TEXT", text("Rate\n", nil, nil))}}, cell}}},
	)
	assertPending(t, List(d), []Pending{
		{ID: "suggest.fee", Kind: frontmatter.KindInsertion, Section: "Fees", Text: "2.5%"},
	})
}

// TestAHeadingInsideACellBecomesTheSection is the same walk seen from the other
// side: the section is whatever heading the walk passed last, wherever it was.
func TestAHeadingInsideACellBecomesTheSection(t *testing.T) {
	cell := docs.Cell{Blocks: []docs.Block{
		para("HEADING_3", text("Inner\n", nil, nil)),
		para("NORMAL_TEXT", text("new", []string{"suggest.in"}, nil)),
	}}
	d := doc(
		para("HEADING_1", text("Outer\n", nil, nil)),
		docs.Block{Table: docs.Table{{cell}}},
		para("NORMAL_TEXT", text("after", []string{"suggest.after"}, nil)),
	)
	assertPending(t, List(d), []Pending{
		{ID: "suggest.in", Kind: frontmatter.KindInsertion, Section: "Inner", Text: "new"},
		{ID: "suggest.after", Kind: frontmatter.KindInsertion, Section: "Inner", Text: "after"},
	})
}

// TestBeforeTheFirstHeadingTheSectionIsEmpty. "" is the honest answer, and it
// is what the front matter stores.
func TestBeforeTheFirstHeadingTheSectionIsEmpty(t *testing.T) {
	d := doc(para("NORMAL_TEXT", text("early", []string{"suggest.a1"}, nil)))
	assertPending(t, List(d), []Pending{
		{ID: "suggest.a1", Kind: frontmatter.KindInsertion, Section: "", Text: "early"},
	})
}

// TestEachTabStartsWithNoHeading: a heading in one tab is not above anything in
// the next one.
func TestEachTabStartsWithNoHeading(t *testing.T) {
	d := &docs.Document{ID: testDocumentID, Tabs: []docs.Tab{
		{ID: "t.0", Body: []docs.Block{para("HEADING_1", text("First\n", nil, nil))}},
		{ID: "t.1", Body: []docs.Block{para("NORMAL_TEXT", text("second", []string{"suggest.b"}, nil))}},
	}}
	assertPending(t, List(d), []Pending{
		{ID: "suggest.b", Kind: frontmatter.KindInsertion, Section: "", Text: "second"},
	})
}

// TestWhitespaceOnlyTextIsDropped. A suggestion whose whole text is a newline
// says nothing a reader can act on, and v1 dropped it for the same reason.
func TestWhitespaceOnlyTextIsDropped(t *testing.T) {
	d := doc(para("NORMAL_TEXT",
		text("\n", []string{"suggest.blank"}, nil),
		text("kept ", []string{"suggest.kept"}, nil),
	))
	assertPending(t, List(d), []Pending{
		{ID: "suggest.kept", Kind: frontmatter.KindInsertion, Text: "kept "},
	})
}

// TestTheTextIsCarriedAsItIs: the trailing space of an inserted word is part of
// the edit, so nothing here trims it away.
func TestTheTextIsCarriedAsItIs(t *testing.T) {
	d := doc(para("NORMAL_TEXT", text("critical ", []string{"suggest.a1"}, nil)))
	if got := List(d)[0].Text; got != "critical " {
		t.Errorf("Text = %q, want %q", got, "critical ")
	}
}

// TestOnlyTextRunsAreSuggestions. A footnote reference carries its number as
// text, and reporting "1" as a suggested insertion would be a lie.
func TestOnlyTextRunsAreSuggestions(t *testing.T) {
	d := doc(para("NORMAL_TEXT",
		docs.Run{Kind: docs.KindFootnoteRef, Text: "1", FootnoteID: "kix.fn1", InsertionIDs: []string{"suggest.fn"}},
		docs.Run{Kind: docs.KindImage, InsertionIDs: []string{"suggest.img"}},
	))
	assertPending(t, List(d), nil)
}

// TestADocumentWithNoSuggestionsIsEmpty, and empty is a nil slice rather than a
// zero-length one, so the envelope prints null rather than [].
func TestADocumentWithNoSuggestionsIsEmpty(t *testing.T) {
	if got := List(fixture(t, "two-tabs.json")); got != nil {
		t.Errorf("List() = %v, want nil", got)
	}
}

func TestGoneListsExactlyTheIDsThatLeft(t *testing.T) {
	at := time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC)
	seen := &frontmatter.SuggestionsSeen{At: at, Items: []frontmatter.SuggestionSeen{
		{ID: "suggest.stay", Kind: frontmatter.KindInsertion, Section: "Scope", Text: "critical "},
		{ID: "suggest.left", Kind: frontmatter.KindDeletion, Section: "Controls", Text: "annually"},
	}}
	now := []string{"suggest.stay"}

	got := GoneSince(seen, now)
	want := []Gone{{
		Pending: Pending{ID: "suggest.left", Kind: frontmatter.KindDeletion, Section: "Controls", Text: "annually"},
		SeenAt:  at,
	}}
	if len(got) != len(want) {
		t.Fatalf("GoneSince() = %+v, want %+v", got, want)
	}
	if got[0] != want[0] {
		t.Errorf("GoneSince()[0] = %+v, want %+v", got[0], want[0])
	}
}

// TestGoneCarriesWhatTheSuggestionSaidLastTime: the text is gone from the
// document, so the snapshot is the only record of what it said.
func TestGoneCarriesWhatTheSuggestionSaidLastTime(t *testing.T) {
	at := time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC)
	seen := &frontmatter.SuggestionsSeen{At: at, Items: []frontmatter.SuggestionSeen{
		{ID: "suggest.left", Kind: frontmatter.KindDeletion, Section: "Controls", Text: "annually"},
	}}
	got := GoneSince(seen, nil)
	if len(got) != 1 {
		t.Fatalf("GoneSince() = %+v, want one item", got)
	}
	if got[0].Text != "annually" || got[0].Section != "Controls" || !got[0].SeenAt.Equal(at) {
		t.Errorf("GoneSince()[0] = %+v, want the item as it was seen at %v", got[0], at)
	}
}

// TestGoneMatchesOnTheIDAlone. One replacement is one id carried by a deletion
// and an insertion, so an id that left takes both of its halves with it.
func TestGoneMatchesOnTheIDAlone(t *testing.T) {
	seen := &frontmatter.SuggestionsSeen{Items: []frontmatter.SuggestionSeen{
		{ID: "suggest.a1", Kind: frontmatter.KindDeletion, Text: "annually"},
		{ID: "suggest.a1", Kind: frontmatter.KindInsertion, Text: "every quarter"},
	}}
	if got := GoneSince(seen, []string{"suggest.a1"}); got != nil {
		t.Errorf("GoneSince() = %+v, want nil: the id is still pending", got)
	}
	if got := GoneSince(seen, nil); len(got) != 2 {
		t.Errorf("GoneSince() = %+v, want both halves of the id", got)
	}
}

func TestGoneWithNoSnapshotIsEmpty(t *testing.T) {
	if got := GoneSince(nil, []string{"suggest.a1"}); got != nil {
		t.Errorf("GoneSince(nil, ...) = %+v, want nil: nothing was seen, so nothing left", got)
	}
}

// TestSnapshotRoundTrips is what the whole snapshot is for: what this run wrote
// is what the next run reads back.
func TestSnapshotRoundTrips(t *testing.T) {
	at := time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC)
	now := List(fixture(t, "single-tab.json"))
	snapshot := Snapshot(now, at)

	block := &frontmatter.Block{Schema: frontmatter.Schema, DocumentID: testDocumentID, SuggestionsSeen: snapshot}
	src := []byte("---\ntitle: Supplier register policy\n---\n\n# Scope\n")
	out, err := frontmatter.Write(src, block)
	if err != nil {
		t.Fatalf("frontmatter.Write: %v", err)
	}
	back, err := frontmatter.Read(out)
	if err != nil {
		t.Fatalf("frontmatter.Read: %v", err)
	}
	if back.SuggestionsSeen == nil {
		t.Fatal("the snapshot did not survive the round trip")
	}
	if !back.SuggestionsSeen.At.Equal(at) {
		t.Errorf("at = %v, want %v", back.SuggestionsSeen.At, at)
	}
	if len(back.SuggestionsSeen.Items) != len(now) {
		t.Fatalf("items = %+v, want %d", back.SuggestionsSeen.Items, len(now))
	}
	for i, item := range back.SuggestionsSeen.Items {
		want := frontmatter.SuggestionSeen{ID: now[i].ID, Kind: now[i].Kind, Section: now[i].Section, Text: now[i].Text}
		if item != want {
			t.Errorf("items[%d] = %+v, want %+v", i, item, want)
		}
	}
	// The snapshot is the input to the next run's Gone, so the two functions
	// have to agree about what a match is.
	nowIDs := make([]string, 0, len(now))
	for _, p := range now {
		nowIDs = append(nowIDs, p.ID)
	}
	if got := GoneSince(back.SuggestionsSeen, nowIDs); got != nil {
		t.Errorf("GoneSince(snapshot, the same pendings) = %+v, want nil", got)
	}
}

// TestSnapshotOfNothingIsStillASnapshot. "Nothing was pending at this time" is
// a fact, and it is the one that lets the next run report what went.
func TestSnapshotOfNothingIsStillASnapshot(t *testing.T) {
	at := time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC)
	got := Snapshot(nil, at)
	if got == nil {
		t.Fatal("Snapshot(nil, at) = nil, want a snapshot with no items")
	}
	if len(got.Items) != 0 || !got.At.Equal(at) {
		t.Errorf("Snapshot(nil, at) = %+v, want an empty snapshot at %v", got, at)
	}
	if err := (&frontmatter.Block{Schema: frontmatter.Schema, DocumentID: testDocumentID, SuggestionsSeen: got}).Validate(); err != nil {
		t.Errorf("the empty snapshot does not validate: %v", err)
	}
}

// TestSnapshotIsInUTC, so two runs on two machines write the same line.
func TestSnapshotIsInUTC(t *testing.T) {
	somewhere := time.FixedZone("UTC+4", 4*60*60)
	at := time.Date(2026, 9, 6, 15, 0, 0, 0, somewhere)
	got := Snapshot(nil, at)
	if _, offset := got.At.Zone(); offset != 0 {
		t.Errorf("Snapshot at = %v, want UTC", got.At)
	}
	if !got.At.Equal(at) {
		t.Errorf("Snapshot at = %v, want the same instant as %v", got.At, at)
	}
}

// TestSnapshotDoesNotAliasItsInput: the caller keeps its slice, and the
// snapshot keeps its own copy.
func TestSnapshotDoesNotAliasItsInput(t *testing.T) {
	now := []Pending{{ID: "suggest.a1", Kind: frontmatter.KindInsertion, Text: "critical "}}
	got := Snapshot(now, time.Now())
	now[0].Text = "changed"
	if got.Items[0].Text != "critical " {
		t.Errorf("the snapshot changed with its input: %q", got.Items[0].Text)
	}
}

func assertPending(t *testing.T, got, want []Pending) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("List() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("List()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// A suggestion whose text the author has edited down to whitespace is still
// pending. List drops it, because it says nothing a reader can act on, and
// comparing the snapshot against that filtered list reported it as gone: a
// false fact in the one field the skill judges accepted-or-rejected from.
func TestASuggestionEditedDownToWhitespaceIsNotGone(t *testing.T) {
	d := document(t, `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":3,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":3,"textRun":{"content":" \n","suggestedInsertionIds":["suggest.thin"]}}]}}]}}`)

	if got := List(d); len(got) != 0 {
		t.Fatalf("List() = %+v, want nothing: the text says nothing a reader can act on", got)
	}
	if got := IDs(d); len(got) != 1 || got[0] != "suggest.thin" {
		t.Fatalf("IDs() = %v, want the id, which is still pending", got)
	}
	// The snapshot is built from All, not List: forgetting the suggestion here
	// means the run that sees it accepted or rejected cannot report it gone.
	if got := All(d); len(got) != 1 || got[0].ID != "suggest.thin" {
		t.Fatalf("All() = %+v, want the suggestion, which is still pending", got)
	}
	seen := &frontmatter.SuggestionsSeen{Items: []frontmatter.SuggestionSeen{
		{ID: "suggest.thin", Kind: frontmatter.KindInsertion, Text: "was a whole sentence"},
	}}
	if got := GoneSince(seen, IDs(d)); got != nil {
		t.Errorf("GoneSince() = %+v, want nil: the suggestion is still in the document", got)
	}
}
