// What a sub-list is indented by, and what a heading's anchor is taken from.
// Both are the same question asked twice: the file the projection writes is
// read back by goldmark, so a rule here that goldmark reads differently is a
// document that changes shape on the way home.

package view

import (
	"fmt"
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// item is one list item at a level, in the list the id names.
func item(ordered bool, level int, listID, s string, at int) docs.Block {
	return docs.Block{Paragraph: &docs.Paragraph{
		Style:  "NORMAL_TEXT",
		Bullet: &docs.Bullet{Ordered: ordered, NestingLevel: level, ListID: listID},
		Runs:   []docs.Run{run(s, at)},
	}}
}

// body is the blocks of a one-tab document, projected.
func body(bs ...docs.Block) string {
	d := &docs.Document{Tabs: []docs.Tab{{ID: "t.0", Body: bs}}}
	text, _ := Text(d)
	return text
}

// TestASubListIndentsToItsParentsColumn: a child is indented to the column its
// parent's content starts at, not to a width taken from its own marker.
// CommonMark nests a sub-list only from that column, so "1. one" followed by a
// bullet under two spaces is not a nested list to any reader: it is an ordered
// list and then a separate bulleted one.
func TestASubListIndentsToItsParentsColumn(t *testing.T) {
	got := body(
		item(true, 0, "l1", "one", 1),
		item(false, 1, "l1", "sub", 10),
	)
	if want := "1. one\n\n   - sub\n"; got != want {
		t.Errorf("the projection is %q, want %q", got, want)
	}
}

// TestASubListUnderAWideMarkerClearsIt: the tenth item's marker is one
// character wider than the ninth's, so a child indented by a fixed three
// spaces falls back inside the parent's own marker and goldmark reads it as
// the eleventh item of the outer list rather than as a child of the tenth.
func TestASubListUnderAWideMarkerClearsIt(t *testing.T) {
	var bs []docs.Block
	for i := 1; i <= 11; i++ {
		bs = append(bs, item(true, 0, "l2", fmt.Sprintf("item %d", i), i*20))
	}
	bs = append(bs, item(true, 1, "l2", "sub", 400))
	got := body(bs...)

	if want := "11. item 11\n\n    1. sub\n"; !strings.HasSuffix(got, want) {
		t.Errorf("the projection ends %q, want it to end %q", tail(got, 3), want)
	}
}

// TestADeeperSubListClearsItsOwnParent: three levels, each one indented to the
// level above it rather than to a multiple of one width.
func TestADeeperSubListClearsItsOwnParent(t *testing.T) {
	got := body(
		item(true, 0, "l3", "one", 1),
		item(true, 1, "l3", "two", 10),
		item(false, 2, "l3", "three", 20),
	)
	if want := "1. one\n\n   1. two\n\n      - three\n"; got != want {
		t.Errorf("the projection is %q, want %q", got, want)
	}
}

// TestASubListWithNoParentFallsBackToItsLevel: a list whose first item is at a
// nesting level above zero has no parent column recorded, so the indent is the
// level's own width. Nothing else is knowable, and a negative indent is not a
// thing a file can hold.
func TestASubListWithNoParentFallsBackToItsLevel(t *testing.T) {
	got := body(item(false, 2, "l4", "orphan", 1))
	if want := "    - orphan\n"; got != want {
		t.Errorf("the projection is %q, want %q", got, want)
	}
}

// TestAHeadingsAnchorIsTheIDOfTheLineItProjects: the anchor a link inside the
// document is written with is goldmark's id for the heading line the export
// writes, not for the heading's text runs. M13 made the two differ: a heading
// holding a hyperlink projects as "[words](url)", and a heading holding a chip
// projects as "[person: Ann]", so an anchor taken from the text runs alone
// names a heading no reader of the file can find.
func TestAHeadingsAnchorIsTheIDOfTheLineItProjects(t *testing.T) {
	for _, c := range []struct {
		name string
		head docs.Paragraph
		want string
	}{
		{
			name: "a hyperlink in the heading",
			head: docs.Paragraph{Style: "HEADING_1", HeadingID: "h.p", Runs: []docs.Run{
				run("See ", 1),
				{Kind: docs.KindText, Text: "the policy", StartIndex: 5, EndIndex: 15,
					Link: &docs.Link{URL: "https://e.com"}},
			}},
			want: "#see-the-policyhttpsecom",
		},
		{
			name: "a person chip in the heading",
			head: docs.Paragraph{Style: "HEADING_1", HeadingID: "h.p", Runs: []docs.Run{
				run("Owner: ", 1),
				{Kind: docs.KindPerson, StartIndex: 8, EndIndex: 9,
					Detail: &docs.Detail{Label: "Ann"}},
			}},
			want: "#owner-person-ann",
		},
		{
			name: "plain words are what they always were",
			head: docs.Paragraph{Style: "HEADING_1", HeadingID: "h.p", Runs: []docs.Run{
				run("Risk and Control", 1),
			}},
			want: "#risk-and-control",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			head := c.head
			d := &docs.Document{Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{
				{Paragraph: &head},
				{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", Runs: []docs.Run{
					{Kind: docs.KindText, Text: "go", StartIndex: 40, EndIndex: 42,
						Link: &docs.Link{HeadingID: "h.p"}},
				}}},
			}}}}
			text, _ := Text(d)
			if !strings.Contains(text, "[go]("+c.want+")") {
				t.Errorf("the projection is %q, want the link to carry %q", text, c.want)
			}
		})
	}
}

// TestAContentsListPrintsNothing pins the decision the M13 plan recorded: the
// projection walks paragraphs and tables and steps over a contents list, which
// Docs generates from the headings that are printed anyway. It is pinned
// because every other walk in this package and in internal/docs does recurse
// into the element, so the one that does not is a decision rather than a gap.
func TestAContentsListPrintsNothing(t *testing.T) {
	got := body(
		docs.Block{Paragraph: &docs.Paragraph{Style: "TITLE", Runs: []docs.Run{run("The policy", 1)}}},
		docs.Block{TOC: &docs.TOC{Blocks: []docs.Block{
			{Paragraph: &docs.Paragraph{Style: "NORMAL_TEXT", Runs: []docs.Run{run("1-Scope", 14)}}},
		}}},
		docs.Block{Paragraph: &docs.Paragraph{Style: "HEADING_1", HeadingID: "h.scope",
			Runs: []docs.Run{run("1-Scope", 48)}}},
	)
	if want := "The policy\n\n# 1-Scope\n"; got != want {
		t.Errorf("the projection is %q, want %q", got, want)
	}
}

// tail is the last n chunks of a projection, for a failure to name.
func tail(s string, n int) string {
	parts := strings.Split(strings.TrimSuffix(s, "\n"), "\n\n")
	if len(parts) > n {
		parts = parts[len(parts)-n:]
	}
	return strings.Join(parts, "\n\n") + "\n"
}
