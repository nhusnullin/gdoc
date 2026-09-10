package prelude

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"gdoc/internal/cover"
	"gdoc/internal/docs"
	"gdoc/internal/guard"
)

// markedDocument is a document carrying one house prelude, marked, with the
// author's own paragraph behind it. suggested says whether the prelude is still
// a proposal: a pending prelude's runs carry an insertion id, and an accepted
// one's carry none.
func markedDocument(start, end int, suggested bool) *docs.Document {
	prelude := docs.Run{Kind: docs.KindText, Text: "A Policy", StartIndex: start, EndIndex: end}
	if suggested {
		prelude.InsertionIDs = []string{"suggest.abc123"}
	}
	return &docs.Document{
		ID: "DOC1",
		Tabs: []docs.Tab{{
			ID: "t.0",
			Body: []docs.Block{
				{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", StartIndex: start, EndIndex: end, Runs: []docs.Run{prelude}}},
				{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", StartIndex: end, EndIndex: end + 20, Runs: []docs.Run{
					{Kind: docs.KindText, Text: "The author's own.\n", StartIndex: end, EndIndex: end + 20},
				}}},
			},
			NamedRanges: []docs.NamedRange{{
				ID:     "kix.wi79lhqfq91l",
				Name:   MarkerName,
				Tab:    "t.0",
				Ranges: []docs.Range{{Tab: "t.0", Start: start, End: end}},
			}},
		}},
	}
}

// A document nobody has put a prelude in carries no marker, so the run proposes
// at the top of the body and deletes nothing.
func TestAFirstRunProposesAtTheTopAndDeletesNothing(t *testing.T) {
	// Arrange
	d := &docs.Document{ID: "DOC1", Tabs: []docs.Tab{{ID: "t.0"}}}

	// Act
	got, err := Decide(d)

	// Assert
	if err != nil {
		t.Fatalf("Decide() = %v", err)
	}
	if got.Start != 1 {
		t.Errorf("Start = %d, want 1, which is where a document's body begins", got.Start)
	}
	if got.Replaces != nil {
		t.Errorf("Replaces = %+v, want none on a document carrying no marker", got.Replaces)
	}
}

// A run marks what it proposed, and the request is the exact shape the guard's
// one marker grant carries: two fields, name and range, and a range of exactly
// two whole numbers.
func TestARunMarksWhatItProposed(t *testing.T) {
	// Arrange, Act
	got := MarkerRequest(1, 964)

	// Assert
	body, ok := got["createNamedRange"].(map[string]any)
	if !ok || len(got) != 1 {
		t.Fatalf("the marker request is %v, want one createNamedRange", got)
	}
	if len(body) != 2 {
		t.Errorf("createNamedRange carries %d fields, want exactly name and range", len(body))
	}
	if body["name"] != "gdoc:house-prelude" {
		t.Errorf("the marker is named %v, want gdoc:house-prelude", body["name"])
	}
	span, ok := body["range"].(map[string]any)
	if !ok || len(span) != 2 {
		t.Fatalf("the marker range is %v, want exactly startIndex and endIndex", body["range"])
	}
	if span["startIndex"] != 1 || span["endIndex"] != 964 {
		t.Errorf("the marker range is %v, want 1..964", span)
	}
}

// The second run is the whole reason the marker exists: it replaces the prelude
// the run before it left rather than putting a second cover in front of it.
func TestASecondRunReplacesTheMarkedPreludeRatherThanAddingASecond(t *testing.T) {
	// Arrange
	d := markedDocument(1, 964, false)

	// Act
	got, err := Decide(d)
	if err != nil {
		t.Fatalf("Decide() = %v", err)
	}

	// Assert
	if got.Replaces == nil {
		t.Fatal("Replaces is nil on a document carrying gdoc's own marker")
	}
	if got.Replaces.ID != "kix.wi79lhqfq91l" {
		t.Errorf("the marker id is %q, want the one the document carries", got.Replaces.ID)
	}
	if got.Replaces.Start != 1 || got.Replaces.End != 964 {
		t.Errorf("the marker covers %d..%d, want 1..964", got.Replaces.Start, got.Replaces.End)
	}
	if got.Start != 1 {
		t.Errorf("Start = %d, want the marked prelude's own start, so the new one lands where the old one was", got.Start)
	}
}

// The requests say the same thing: the old prelude is proposed for deletion and
// the new one is proposed in its place, in that order, which is the order
// internal/propose sends a replacement in.
func TestASecondRunProposesTheDeletionOfTheOldPreludeFirst(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	d := markedDocument(1, 964, false)

	// Act
	got, err := Propose(cfg, cover.Fields{Title: "A Policy"}, d)
	if err != nil {
		t.Fatalf("Propose() = %v", err)
	}

	// Assert
	del, ok := got.Requests[0]["deleteContentRange"].(map[string]any)
	if !ok {
		t.Fatalf("the first request is %v, want the deletion of the prelude that is there", got.Requests[0])
	}
	span := del["range"].(map[string]any)
	if span["startIndex"] != 1 || span["endIndex"] != 964 {
		t.Errorf("the deletion covers %v, want the marked range 1..964", span)
	}
	if got.Replaces == nil || got.Replaces.ID != "kix.wi79lhqfq91l" {
		t.Errorf("Replaces = %+v, want the marker this run replaces", got.Replaces)
	}
	if got.Start != 1 {
		t.Errorf("the new prelude starts at %d, want 1", got.Start)
	}
	ins, ok := got.Requests[1]["insertText"].(map[string]any)
	if !ok {
		t.Fatalf("the second request is %v, want the first line of the new prelude", got.Requests[1])
	}
	if at := ins["location"].(map[string]any)["index"]; at != 1 {
		t.Errorf("the new prelude is inserted at %v, want 1, the start of the span the deletion covered", at)
	}
}

// A first run's requests are the front matter and nothing else. Nothing is
// deleted on a document that has no prelude to delete.
func TestAFirstRunDeletesNothingAtAll(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	d := &docs.Document{ID: "DOC1", Tabs: []docs.Tab{{ID: "t.0"}}}

	// Act
	got, err := Propose(cfg, cover.Fields{Title: "A Policy"}, d)
	if err != nil {
		t.Fatalf("Propose() = %v", err)
	}

	// Assert
	for i, r := range got.Requests {
		if _, ok := r["deleteContentRange"]; ok {
			t.Fatalf("request %d deletes a range on a document carrying no prelude", i)
		}
	}
	if got.Replaces != nil {
		t.Errorf("Replaces = %+v, want none", got.Replaces)
	}
}

// The third shape is neither of the other two: a marker over a prelude that is
// still a proposal. Replacing it would propose a deletion of text that has not
// been written, and the answer is the one already in front of Nail.
func TestASecondRunRefusesWhileTheFirstPreludeIsStillPending(t *testing.T) {
	// Arrange
	d := markedDocument(1, 964, true)

	// Act
	_, err := Decide(d)

	// Assert
	if err == nil {
		t.Fatal("a prelude still pending was replaced rather than refused")
	}
	for _, want := range []string{"pending", "Accept or reject", "suggest.abc123"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal is %q, and it does not say %q", err, want)
		}
	}
}

// A prelude this run proposed deleting is pending too: its runs carry a
// deletion id rather than an insertion one, and the answer is the same.
func TestAPreludePendingDeletionIsRefusedTheSameWay(t *testing.T) {
	// Arrange
	d := markedDocument(1, 964, false)
	d.Tabs[0].Body[0].Paragraph.Runs[0].DeletionIDs = []string{"suggest.def456"}

	// Act
	_, err := Decide(d)

	// Assert
	if err == nil {
		t.Fatal("a prelude proposed for deletion was replaced rather than refused")
	}
	if !strings.Contains(err.Error(), "suggest.def456") {
		t.Errorf("the refusal is %q, and it does not name the suggestion", err)
	}
}

// A pending prelude is pending whichever cell of its own tables the suggestion
// sits in. The front matter is three tables, so a walk that read paragraphs
// alone would call a wholly proposed prelude settled.
func TestAPendingSuggestionInsideAPreludeTableIsFound(t *testing.T) {
	// Arrange
	d := markedDocument(1, 964, false)
	d.Tabs[0].Body[0].Table = &docs.Table{StartIndex: 400, Rows: [][]docs.Cell{{{Blocks: []docs.Block{
		{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", StartIndex: 402, EndIndex: 410, Runs: []docs.Run{
			{Kind: docs.KindText, Text: "Owner", StartIndex: 402, EndIndex: 407, InsertionIDs: []string{"suggest.incell"}},
		}}},
	}}}}}
	d.Tabs[0].Body[0].Paragraph = nil

	// Act
	_, err := Decide(d)

	// Assert
	if err == nil {
		t.Fatal("a suggestion inside one of the prelude's own tables was not found")
	}
	if !strings.Contains(err.Error(), "suggest.incell") {
		t.Errorf("the refusal is %q, and it does not name the suggestion", err)
	}
}

// The state every replace run leaves while its suggestions are unsettled: the
// new prelude marked and pending as an insertion, the old one marked and
// pending as a deletion, because nothing deletes a named range. That is two
// markers and it is not an ambiguity, so the refusal is the one that says to
// settle the suggestions in the browser rather than the one that says to remove
// a named range by hand, which nothing in the Docs UI can do.
func TestAReplaceRunsTwoPendingMarkersAskForTheBrowserRatherThanAHandEdit(t *testing.T) {
	// Arrange
	d := &docs.Document{
		ID: "DOC1",
		Tabs: []docs.Tab{{
			ID: "t.0",
			Body: []docs.Block{
				{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", StartIndex: 1, EndIndex: 900, Runs: []docs.Run{
					{Kind: docs.KindText, Text: "The fresh prelude", StartIndex: 1, EndIndex: 900,
						InsertionIDs: []string{"suggest.newone"}},
				}}},
				{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", StartIndex: 900, EndIndex: 1000, Runs: []docs.Run{
					{Kind: docs.KindText, Text: "The prelude before it", StartIndex: 900, EndIndex: 1000,
						DeletionIDs: []string{"suggest.oldone"}},
				}}},
			},
			NamedRanges: []docs.NamedRange{
				{ID: "kix.fresh", Name: MarkerName, Tab: "t.0",
					Ranges: []docs.Range{{Tab: "t.0", Start: 1, End: 900}}},
				{ID: "kix.older", Name: MarkerName, Tab: "t.0",
					Ranges: []docs.Range{{Tab: "t.0", Start: 900, End: 1000}}},
			},
		}},
	}

	// Act
	_, err := Decide(d)

	// Assert
	if err == nil {
		t.Fatal("a replace run's own two markers were read as a document this run may replace")
	}
	if !strings.Contains(err.Error(), "still pending") {
		t.Errorf("the refusal is %q, and it does not say the prelude is still pending", err)
	}
	if strings.Contains(err.Error(), "by hand") {
		t.Errorf("the refusal is %q, and it asks for a hand edit the Docs UI cannot make", err)
	}
	for _, want := range []string{"suggest.newone", "suggest.oldone"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal is %q, and it does not name the suggestion %q to settle", err, want)
		}
	}
}

// The marker is read by id, and two ranges may wear one name. Docs puts both
// under that one key, so a run that acted by name would act on both.
//
// Two markers over settled text is the ambiguity this cannot guess between, and
// it stays a refusal naming both ids.
func TestTwoRangesWearingTheMarkerNameAreRefusedByID(t *testing.T) {
	// Arrange
	d := markedDocument(1, 964, false)
	d.Tabs[0].NamedRanges = append(d.Tabs[0].NamedRanges, docs.NamedRange{
		ID:     "kix.second",
		Name:   MarkerName,
		Tab:    "t.0",
		Ranges: []docs.Range{{Tab: "t.0", Start: 1000, End: 1100}},
	})

	// Act
	_, err := Decide(d)

	// Assert
	if err == nil {
		t.Fatal("two ranges wearing the marker name were read as one prelude")
	}
	for _, want := range []string{"kix.wi79lhqfq91l", "kix.second"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal is %q, and it does not name %q", err, want)
		}
	}
}

// A marker whose spans have the author's own text between them is not a span
// this run may delete. Deleting the whole of it would propose deleting words
// gdoc never wrote.
func TestAMarkerBrokenAcrossTheAuthorsTextIsRefused(t *testing.T) {
	// Arrange
	d := markedDocument(1, 964, false)
	d.Tabs[0].NamedRanges[0].Ranges = []docs.Range{
		{Tab: "t.0", Start: 1, End: 400},
		{Tab: "t.0", Start: 500, End: 964},
	}

	// Act
	_, err := Decide(d)

	// Assert
	if err == nil {
		t.Fatal("a marker with the author's text inside it was read as one prelude")
	}
	if !strings.Contains(err.Error(), "kix.wi79lhqfq91l") {
		t.Errorf("the refusal is %q, and it does not name the marker", err)
	}
}

// Two spans that touch are one span. Docs cuts a range at a boundary of its
// own, and a prelude cut in two is still contiguous text gdoc wrote.
func TestAMarkerCutIntoTwoTouchingSpansIsOnePrelude(t *testing.T) {
	// Arrange
	d := markedDocument(1, 964, false)
	d.Tabs[0].NamedRanges[0].Ranges = []docs.Range{
		{Tab: "t.0", Start: 1, End: 400},
		{Tab: "t.0", Start: 400, End: 964},
	}

	// Act
	got, err := Decide(d)

	// Assert
	if err != nil {
		t.Fatalf("Decide() = %v", err)
	}
	if got.Replaces == nil || got.Replaces.Start != 1 || got.Replaces.End != 964 {
		t.Errorf("Replaces = %+v, want the whole of 1..964", got.Replaces)
	}
}

// A named range sitting in a header, a footer or a footnote names a position in
// text this run never measured. The prelude is in the body, so a marker that is
// not is one gdoc did not make.
func TestAMarkerInASegmentIsRefused(t *testing.T) {
	// Arrange
	d := markedDocument(1, 964, false)
	d.Tabs[0].NamedRanges[0].Ranges[0].Segment = "h.headerone"

	// Act
	_, err := Decide(d)

	// Assert
	if err == nil {
		t.Fatal("a marker in a header was read as a prelude in the body")
	}
	if !strings.Contains(err.Error(), "kix.wi79lhqfq91l") {
		t.Errorf("the refusal is %q, and it does not name the marker", err)
	}
}

// A named range wearing somebody else's name is somebody else's. gdoc reads one
// name and leaves every other range where it is.
func TestARangeWearingAnotherNameIsNotAPrelude(t *testing.T) {
	// Arrange
	d := markedDocument(1, 964, false)
	d.Tabs[0].NamedRanges[0].Name = "the author's own bookmark"

	// Act
	got, err := Decide(d)

	// Assert
	if err != nil {
		t.Fatalf("Decide() = %v", err)
	}
	if got.Replaces != nil {
		t.Errorf("Replaces = %+v, want none: that range is not gdoc's marker", got.Replaces)
	}
}

// The request this package builds is the request the guard grants. Two files
// spelling one shape are two shapes that drift, and the guard reads
// createNamedRange exactly: a third field, another spelling of an index or a
// range off by one is refused. So the pin is the real policy judging the real
// bytes rather than a second copy of the shape written out here.
func TestTheMarkerRequestIsWhatTheGuardGrants(t *testing.T) {
	// Arrange: a phase 2 policy, granted the marker this run is about to make.
	const start, end = 1, 964
	p := guard.NewPolicy()
	p.AllowFile("DOC1", guard.LevelSuggest)
	p.GrantInPlace("DOC1")
	p.AllowMarker(MarkerName, start, end)
	body, err := json.Marshal(map[string]any{"requests": []map[string]any{MarkerRequest(start, end)}})
	if err != nil {
		t.Fatal(err)
	}
	at, err := url.Parse("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if err != nil {
		t.Fatal(err)
	}

	// Act, Assert
	if err := p.Judge("POST", at, body); err != nil {
		t.Fatalf("the guard refused the marker this package builds: %v", err)
	}
	if len(p.Warnings()) != 0 {
		t.Errorf("the marker grant warned: %v", p.Warnings())
	}
}

// The other direction, and it is the one that matters: a run that has not been
// granted this marker cannot make it. The grant is per-run and names one range,
// so a prelude that ended somewhere else is a marker the guard refuses.
func TestAMarkerOverAnotherRangeIsRefused(t *testing.T) {
	// Arrange
	p := guard.NewPolicy()
	p.AllowFile("DOC1", guard.LevelSuggest)
	p.GrantInPlace("DOC1")
	p.AllowMarker(MarkerName, 1, 964)
	body, err := json.Marshal(map[string]any{"requests": []map[string]any{MarkerRequest(1, 2000)}})
	if err != nil {
		t.Fatal(err)
	}
	at, err := url.Parse("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if err != nil {
		t.Fatal(err)
	}

	// Act, Assert
	if p.Judge("POST", at, body) == nil {
		t.Fatal("a marker over a range this run did not grant must be refused")
	}
}

// A first run's own words are the prelude and nothing else, so what phase 2
// walks past is exactly what the marker covers.
func TestAFirstRunOccupiesTheSpanItProposed(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	d := &docs.Document{ID: "DOC1", Tabs: []docs.Tab{{ID: "t.0"}}}

	// Act
	got, err := Propose(cfg, cover.Fields{Title: "A Policy"}, d)
	if err != nil {
		t.Fatalf("Propose() = %v", err)
	}
	start, end := got.Occupies()

	// Assert
	if start != got.Start || end != got.End {
		t.Errorf("Occupies() = %d..%d, want the proposed span %d..%d", start, end, got.Start, got.End)
	}
}

// A replace run's own words are two preludes, not one. The deletion goes out in
// SUGGEST mode, which marks text rather than removing it, so the prelude the run
// before this one left is still real text sitting immediately behind the words
// that replace it. Phase 2 has to walk past both: styled, the old cover is
// flattened to body prose by direct edit, and rejecting the suggestion puts the
// text back and not its look.
func TestAReplaceRunOccupiesTheOldPreludeToo(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	const oldStart, oldEnd = 1, 964
	d := markedDocument(oldStart, oldEnd, false)

	// Act
	got, err := Propose(cfg, cover.Fields{Title: "A Policy"}, d)
	if err != nil {
		t.Fatalf("Propose() = %v", err)
	}
	start, end := got.Occupies()

	// Assert
	if got.Replaces == nil {
		t.Fatal("Replaces is nil on a document carrying gdoc's own marker")
	}
	if start != got.Start {
		t.Errorf("Occupies() starts at %d, want the prelude's own start %d", start, got.Start)
	}
	want := got.End + (oldEnd - oldStart)
	if end != want {
		t.Errorf("Occupies() ends at %d, want %d: the old prelude is %d units long and the insert at %d pushed it to [%d,%d)",
			end, want, oldEnd-oldStart, got.Start, got.End, want)
	}
	if end <= got.End {
		t.Errorf("Occupies() = %d..%d, which leaves the replaced prelude for phase 2 to style", start, end)
	}
}
