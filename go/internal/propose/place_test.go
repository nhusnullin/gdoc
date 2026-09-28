package propose

import (
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// plainBlock is the content most placement tests carry: a heading and a
// paragraph, ending in plain body text, which is what the last position in a
// document allows.
func plainBlock(t *testing.T) []Para {
	t.Helper()
	paras, err := ParseContent("## Limits\n\nThe register holds ten suppliers.\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}
	return paras
}

func TestAfterPlacesTheBlockAtTheStartOfTheNextParagraph(t *testing.T) {
	got, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", plainBlock(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The paragraph after the anchor starts at 93, and that is where the text
	// goes in: at a start every new paragraph owns a mark of its own.
	if got.Tab != "t.0" || got.Insert != 93 {
		t.Errorf("placement = %+v, want the start of the paragraph at 93 in t.0", got)
	}
	if got.AtEnd {
		t.Error("the anchor is not the document's last paragraph, so AtEnd is wrong")
	}
	if got.Replaces() {
		t.Errorf("an after block deletes nothing, and Delete = %+v", got.Delete)
	}
	want := "The supplier register is reviewed annually by the operations team."
	if len(got.Paragraphs) != 1 || got.Paragraphs[0] != want {
		t.Errorf("Paragraphs = %q, want the anchor paragraph %q", got.Paragraphs, want)
	}
}

// TestAfterPlacesABlockBehindAListItem is the everyday list case: the anchor is
// a bullet with a plain paragraph behind it, so there is a paragraph start to
// go in at and nothing is refused.
func TestAfterPlacesABlockBehindAListItem(t *testing.T) {
	got, err := PlaceAfter(document(t, "block-body.json"), "top ten suppliers", plainBlock(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Insert != 185 {
		t.Errorf("Insert = %d, want the start of the paragraph at 185", got.Insert)
	}
}

func TestAReplaceCoversWholeParagraphs(t *testing.T) {
	got, err := PlaceReplace(document(t, "block-body.json"), "reviewed annually", "risk matrix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// From the start of the paragraph the first quote is in to the end of the
	// paragraph the second is in, marks included, so an accept can never merge
	// a neighbour.
	wantDelete := docs.Range{Tab: "t.0", Start: 26, End: 142}
	if got.Delete != wantDelete || !got.Replaces() {
		t.Errorf("Delete = %+v, want %+v", got.Delete, wantDelete)
	}
	if got.Insert != 26 {
		t.Errorf("Insert = %d, want the start of the first replaced paragraph", got.Insert)
	}
	want := []string{
		"The supplier register is reviewed annually by the operations team.",
		"Each supplier is scored against the risk matrix.",
	}
	if strings.Join(got.Paragraphs, "|") != strings.Join(want, "|") {
		t.Errorf("Paragraphs = %q, want %q", got.Paragraphs, want)
	}
}

func TestAReplaceInsideOneParagraphCoversThatParagraph(t *testing.T) {
	got, err := PlaceReplace(document(t, "block-busy.json"), "register lists", "every supplier")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantDelete := docs.Range{Tab: "t.0", Start: 1, End: 36}
	if got.Delete != wantDelete || got.Insert != 1 {
		t.Errorf("placement = %+v, want the one paragraph %+v", got, wantDelete)
	}
}

// TestAfterTheLastParagraphGoesBeforeTheFinalNewline is the shape MEASURED.md
// row 7 pins. There is no paragraph to go in front of, so the text goes in
// before the body's final mark and the block's last paragraph owns it.
func TestAfterTheLastParagraphGoesBeforeTheFinalNewline(t *testing.T) {
	got, err := PlaceAfter(document(t, "block-body.json"), "operations lead", plainBlock(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.AtEnd {
		t.Error("the anchor is the document's last paragraph, and AtEnd says so")
	}
	// The last paragraph ends at 224, and its own newline is the unit before.
	if got.Insert != 223 {
		t.Errorf("Insert = %d, want 223, which is before the body's final newline", got.Insert)
	}
}

func TestPlaceRefusesWhatItCannotPlaceAndNamesIt(t *testing.T) {
	heading, err := ParseContent("Body text first.\n\n## And a heading last\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}
	listLast, err := ParseContent("Body text first.\n\n- one\n- two\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}

	cases := []struct {
		name  string
		place func(t *testing.T) (Placement, error)
		says  []string
	}{
		{
			name: "a quote that is not in the document",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "block-body.json"), "reviewed monthly", plainBlock(t))
			},
			says: []string{"not found"},
		},
		{
			name: "a quote that occurs twice",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "twice.json"), "reviewed annually", plainBlock(t))
			},
			says: []string{"2 times", "quote more"},
		},
		{
			name: "a document with more than one tab",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "two-tabs.json"), "The first tab", plainBlock(t))
			},
			says: []string{"tabs"},
		},
		{
			name: "an anchor inside a table cell",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "block-table.json"), "Register review", plainBlock(t))
			},
			says: []string{"inside a table"},
		},
		{
			name: "an anchor whose next element is a table",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "block-table.json"), "listed below", plainBlock(t))
			},
			says: []string{"table", "start of a paragraph"},
		},
		{
			name: "an anchor whose next element is a contents list",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "block-toc.json"), "The policy", plainBlock(t))
			},
			says: []string{"contents list", "start of a paragraph"},
		},
		{
			name: "an anchor with a section break behind it",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "block-gap.json"), "ends here", plainBlock(t))
			},
			says: []string{"does not index", "section break"},
		},
		{
			name: "a block that does not end in a plain paragraph, after the last paragraph",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "block-body.json"), "operations lead", heading)
			},
			says: []string{"last paragraph", "plain paragraph"},
		},
		{
			name: "a block ending in a list item, after the last paragraph",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "block-body.json"), "operations lead", listLast)
			},
			says: []string{"last paragraph", "plain paragraph"},
		},
		{
			name: "a last paragraph that is itself a list item",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "block-last-is-list.json"), "operations lead", plainBlock(t))
			},
			says: []string{"list item", "bullet"},
		},
		{
			name: "a last paragraph that is itself a heading",
			place: func(t *testing.T) (Placement, error) {
				return PlaceAfter(document(t, "block-last-is-heading.json"), "operations lead", plainBlock(t))
			},
			says: []string{"HEADING_2", "final mark"},
		},
		{
			name: "a replace whose quotes are the wrong way round",
			place: func(t *testing.T) (Placement, error) {
				return PlaceReplace(document(t, "block-body.json"), "risk matrix", "reviewed annually")
			},
			says: []string{"comes before", "swap"},
		},
		{
			name: "a replace reaching the body's last paragraph",
			place: func(t *testing.T) (Placement, error) {
				return PlaceReplace(document(t, "block-body.json"), "Each supplier", "operations lead")
			},
			says: []string{"last paragraph", "final newline"},
		},
		{
			name: "a replace covering a table",
			place: func(t *testing.T) (Placement, error) {
				return PlaceReplace(document(t, "block-table.json"), "listed below", "operations lead")
			},
			says: []string{"table", "whole paragraphs"},
		},
		{
			name: "a replace covering a section break",
			place: func(t *testing.T) (Placement, error) {
				return PlaceReplace(document(t, "block-gap.json"), "ends here", "opens here")
			},
			says: []string{"does not index", "section break"},
		},
		{
			name: "a replace over a paragraph holding a picture",
			place: func(t *testing.T) (Placement, error) {
				return PlaceReplace(document(t, "block-inline-picture.json"), "reviewed annually", "Figure 1")
			},
			says: []string{"a picture", "leave that paragraph out"},
		},
		{
			// A floating object is anchored to a paragraph rather than sitting
			// in its text, so no run names it and the read-backs are as blind
			// to it as they are to an inline one. The deletion takes it with
			// the paragraph it is anchored to, so the run is refused by name.
			name: "a replace over a paragraph anchoring a floating picture",
			place: func(t *testing.T) (Placement, error) {
				return PlaceReplace(document(t, "block-floating-picture.json"), "reviewed annually", "shows the flow")
			},
			says: []string{"a floating picture", "leave that paragraph out"},
		},
		{
			name: "a replace over somebody's pending suggestion and comments",
			place: func(t *testing.T) (Placement, error) {
				return PlaceReplace(document(t, "block-busy.json"), "register lists", "operations lead")
			},
			says: []string{"sug.INSERTED", "sug.DELETED", "C-SUGGESTED", "C-ESCALATION"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.place(t)
			if err == nil {
				t.Fatalf("placement %+v was accepted, and it should be refused", got)
			}
			for _, says := range c.says {
				if !strings.Contains(err.Error(), says) {
					t.Errorf("error = %q, and it should say %q", err, says)
				}
			}
		})
	}
}

// TestAPlacementIsComputedFromTheReadAndNotStored is the rule the whole
// package stands on: a placement comes out of the read the write is built
// from, and nothing writes one down. The proposals file names words and never
// an index, so the same words against a document that has moved place the
// block where the words are now.
func TestAPlacementIsComputedFromTheReadAndNotStored(t *testing.T) {
	d := document(t, "block-body.json")
	first, err := PlaceAfter(d, "reviewed annually", plainBlock(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The same words against a document whose heading grew by five units place
	// the block five units further on. A stored index could not do that.
	moved := document(t, "block-body.json")
	heading := moved.Tabs[0].Body[0].Paragraph
	heading.Runs[0].Text = "Supplier register policy 2026\n"
	heading.Runs[0].EndIndex += 5
	heading.EndIndex += 5
	for _, b := range moved.Tabs[0].Body[1:] {
		b.Paragraph.StartIndex += 5
		b.Paragraph.EndIndex += 5
		for i := range b.Paragraph.Runs {
			b.Paragraph.Runs[i].StartIndex += 5
			b.Paragraph.Runs[i].EndIndex += 5
		}
	}
	second, err := PlaceAfter(moved, "reviewed annually", plainBlock(t))
	if err != nil {
		t.Fatalf("unexpected error on the moved document: %v", err)
	}
	if second.Insert != first.Insert+5 {
		t.Errorf("Insert = %d on the moved document and %d on the first; the placement follows the read", second.Insert, first.Insert)
	}
}
