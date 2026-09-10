package prelude

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// preludeAsRead is the document as the read-back sees it: gdoc's own prelude
// at the top, every run of it carrying one suggestion id, the marker over it,
// and the author's own paragraph behind it untouched.
//
// The prelude is a paragraph and a one-cell table, because the house front
// matter is both and a walk that read paragraphs alone would call a wholly
// proposed table settled.
func preludeAsRead(suggested bool) *docs.Document {
	id := []string{"suggest.abc123"}
	if !suggested {
		id = nil
	}
	return &docs.Document{
		ID: "DOC1",
		Tabs: []docs.Tab{{
			ID: "t.0",
			Body: []docs.Block{
				{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", StartIndex: 1, EndIndex: 10, Runs: []docs.Run{
					{Kind: docs.KindText, Text: "A Policy\n", StartIndex: 1, EndIndex: 10, InsertionIDs: id},
				}}},
				{Table: &docs.Table{StartIndex: 10, Rows: [][]docs.Cell{{{Blocks: []docs.Block{
					{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", StartIndex: 12, EndIndex: 20, Runs: []docs.Run{
						{Kind: docs.KindText, Text: "Version\n", StartIndex: 12, EndIndex: 20, InsertionIDs: id},
					}}},
				}}}}}},
				{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", StartIndex: 21, EndIndex: 39, Runs: []docs.Run{
					{Kind: docs.KindText, Text: "The author's own.\n", StartIndex: 21, EndIndex: 39},
				}}},
			},
			NamedRanges: []docs.NamedRange{{
				ID:     "kix.wi79lhqfq91l",
				Name:   MarkerName,
				Tab:    "t.0",
				Ranges: []docs.Range{{Tab: "t.0", Start: 1, End: 21}},
			}},
		}},
	}
}

// The author's own text as it stood before a word of the prelude was proposed.
const authorBefore = "The author's own.\n"

// The whole claim of this milestone, read back out of the document: every piece
// of the prelude is a suggestion, the marker is over it, and not a character of
// the author's own text moved.
func TestThePreludeReadsBackAsSuggestions(t *testing.T) {
	// Arrange
	d := preludeAsRead(true)

	// Act
	got, warnings := Verify(authorBefore, d, 1, 21)

	// Assert
	if !got.Verified {
		t.Fatalf("Verified = false on a prelude that is wholly suggested: %+v, %v", got, warnings)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
	if got.Proposed.Paragraphs != 2 || got.Proposed.Tables != 1 || got.Proposed.Cells != 1 {
		t.Errorf("proposed = %+v, want 2 paragraphs, 1 table and 1 cell, counted the way the run counts what it sent", got.Proposed)
	}
	if got.Proposed.Runs != 2 {
		t.Errorf("proposed.runs = %d, want 2", got.Proposed.Runs)
	}
	if (got.Written != Pieces{}) {
		t.Errorf("written = %+v, want nothing at all: a piece with no suggestion id is a character gdoc put in on its own authority", got.Written)
	}
	if !got.Marked || !got.BodyUnchanged {
		t.Errorf("marked = %v, body_unchanged = %v, want both", got.Marked, got.BodyUnchanged)
	}
}

// A run inside the span carrying no suggestion id is text gdoc wrote rather
// than proposed, which is the one thing this milestone says it never does. It
// is reported as written, it is named, and the run is not verified.
func TestARunWithNoSuggestionIDIsWrittenRatherThanProposed(t *testing.T) {
	// Arrange
	d := preludeAsRead(true)
	d.Tabs[0].Body[0].Paragraph.Runs[0].InsertionIDs = nil

	// Act
	got, warnings := Verify(authorBefore, d, 1, 21)

	// Assert
	if got.Verified {
		t.Errorf("Verified = true over a prelude run carrying no suggestion id: %+v", got)
	}
	if got.Written.Runs != 1 || got.Written.Paragraphs != 1 {
		t.Errorf("written = %+v, want the one run and the paragraph holding it", got.Written)
	}
	if got.Proposed.Runs != 1 {
		t.Errorf("proposed.runs = %d, want the cell's run, which is still a suggestion", got.Proposed.Runs)
	}
	if !strings.Contains(strings.Join(warnings, " "), "A Policy") {
		t.Errorf("warnings = %v, want the words that are in the document named", warnings)
	}
}

// The check the milestone's own claim rests on: the author's text before the
// run against the author's text after it. A test rather than a hope.
func TestTheAuthorsOwnTextIsCheckedBeforeAgainstAfter(t *testing.T) {
	// Arrange
	d := preludeAsRead(true)
	d.Tabs[0].Body[2].Paragraph.Runs[0].Text = "The author's own words, edited.\n"

	// Act
	got, warnings := Verify(authorBefore, d, 1, 21)

	// Assert
	if got.BodyUnchanged {
		t.Errorf("body_unchanged = true over text that moved: %+v", got)
	}
	if got.Verified {
		t.Errorf("Verified = true over an author's text that is not what it was")
	}
	if !strings.Contains(strings.Join(warnings, " "), "the author") {
		t.Errorf("warnings = %v, want the author's own text named as the route that did not hold", warnings)
	}
}

// A prelude nothing marked is one the next run cannot find, so the read-back
// says so rather than reporting a clean run.
func TestAPreludeWithNoMarkerOverItIsNotVerified(t *testing.T) {
	// Arrange
	d := preludeAsRead(true)
	d.Tabs[0].NamedRanges = nil

	// Act
	got, warnings := Verify(authorBefore, d, 1, 21)

	// Assert
	if got.Marked || got.Verified {
		t.Errorf("marked = %v, verified = %v, want both false with no marker in the document", got.Marked, got.Verified)
	}
	if !strings.Contains(strings.Join(warnings, " "), MarkerName) {
		t.Errorf("warnings = %v, want the marker named", warnings)
	}
}

// A span the read found nothing in is a prelude nothing says is there, and
// silence is the answer that would report it as a clean run.
func TestASpanTheReadFoundNothingInIsNotVerified(t *testing.T) {
	// Arrange
	d := preludeAsRead(true)

	// Act
	got, warnings := Verify(authorBefore, d, 900, 960)

	// Assert
	if got.Verified {
		t.Errorf("Verified = true over a span holding nothing: %+v", got)
	}
	if !strings.Contains(strings.Join(warnings, " "), "read back") {
		t.Errorf("warnings = %v, want the read named", warnings)
	}
}

// AuthorText is the document with gdoc's proposal taken out of it: a suggested
// insertion is not the author's, and a suggested deletion still is, because
// nothing has removed those characters yet.
func TestAuthorTextDropsInsertionsAndKeepsDeletions(t *testing.T) {
	// Arrange
	d := &docs.Document{Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{
		{Paragraph: &docs.Paragraph{Runs: []docs.Run{
			{Kind: docs.KindText, Text: "reviewed ", StartIndex: 1, EndIndex: 10},
			{Kind: docs.KindText, Text: "annually", StartIndex: 10, EndIndex: 18, DeletionIDs: []string{"suggest.a1"}},
			{Kind: docs.KindText, Text: "quarterly", StartIndex: 18, EndIndex: 27, InsertionIDs: []string{"suggest.a1"}},
		}}},
	}}}}

	// Act
	got := AuthorText(d)

	// Assert
	if got != "reviewed annually" {
		t.Errorf("AuthorText() = %q, want the author's own words with the proposal taken out", got)
	}
}

// The M2 line, held at the field most likely to cross it. Every field here is a
// count, a span, a list or a flag about what the read found. Whether the cover
// is right is read in the document, by the person being asked to accept it.
func TestNothingInThePreludeReadBackIsAVerdict(t *testing.T) {
	// Arrange
	banned := []string{"looks_right", "complete", "ready", "good", "correct", "accepted", "handled", "drift"}
	raw, err := json.Marshal(Check{})
	if err != nil {
		t.Fatalf("Check does not marshal: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("Check is not an object: %v", err)
	}

	// Act, Assert
	for _, name := range banned {
		if _, there := fields[name]; there {
			t.Errorf("Check carries a field called %q, which is a verdict rather than a fact", name)
		}
	}
	if _, there := fields["proposed"]; !there {
		t.Errorf("Check carries %v, want proposed, which is a count of suggestions", reflect.ValueOf(fields).MapKeys())
	}
}
