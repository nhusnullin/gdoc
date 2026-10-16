package body

import (
	"strings"
	"testing"
)

// TestBuildRefusesAMarkerByLine is the door decision 3 closes on the way back
// into a document. A file exported out of a Google Doc carries gdoc's own
// markers until somebody resolves them, and a note still carrying one is a note
// whose suggested words would be published as prose, brackets and all.
//
// The refusal names the line, because that is where the person has to look, and
// the marker, because the four do not look alike to somebody reading the
// sentence.
func TestBuildRefusesAMarkerByLine(t *testing.T) {
	for _, c := range []struct{ name, marker, line string }{
		{"an insertion", "{+", "The team {+meets weekly+}[s:AAA] in Q3."},
		{"a deletion", "{-", "The team {-met monthly-}[s:BBB] in Q2."},
		{"a comment anchor", "[[c:", "The [[c:CCC]]operations team[[/c]] owns it."},
		{"a suggestion id on its own", "[s:", "The team meets weekly [s:AAA] in Q3."},
		{"a tab line", "<!-- tab ", "<!-- tab t.0: Operations -->"},
		{"a floating picture", "<!-- image: ", "<!-- image: floating, kix.p1 -->"},
	} {
		t.Run(c.name, func(t *testing.T) {
			note := "# Heading\n\nOne.\n\nTwo.\n\n" + c.line + "\n"
			if got := strings.Split(note, "\n")[6]; got != c.line {
				t.Fatalf("the note puts %q on another line than 7, so the refusal cannot be read", got)
			}
			_, err := Render(config(t), []byte(note), "testdata/docs", true)
			if err == nil {
				t.Fatalf("Render accepted a body carrying %q", c.marker)
			}
			if !strings.Contains(err.Error(), "line 7") {
				t.Errorf("the refusal does not name line 7: %v", err)
			}
			if !strings.Contains(err.Error(), c.marker) {
				t.Errorf("the refusal does not name %q: %v", c.marker, err)
			}
		})
	}
}

// TestBuildLeavesAnEscapedMarkerAlone is the half that keeps a note about gdoc
// publishable. `read` escapes the author's own "{+", so a note written about
// the markers themselves carries escaped ones and none of them is gdoc's.
func TestBuildLeavesAnEscapedMarkerAlone(t *testing.T) {
	note := "# Heading\n\n" + `The insertion marker is written \{+words\+} in the export.` + "\n"
	if _, err := Render(config(t), []byte(note), "testdata/docs", true); err != nil {
		t.Fatalf("Render refused a note whose markers are all escaped: %v", err)
	}
}

// TestGdocsOwnCommentsAreMarkersAndNotPencil is the one place the two rules
// this milestone wrote could have met and do not. A comment is the grey mark
// now, so a note carrying the comment `read` writes for a floating picture
// would otherwise publish "image: floating, kix.p1" into the document as a
// grey aside, which says a thing about the old document and nothing about this
// one. The marker refusal is asked before the note is parsed, so the pencil
// never sees it.
func TestGdocsOwnCommentsAreMarkersAndNotPencil(t *testing.T) {
	for _, c := range []struct{ name, marker, line string }{
		{"a floating picture", "<!-- image: ", "<!-- image: floating, kix.p1 -->"},
		{"a tab line", "<!-- tab ", "<!-- tab t.0: Operations -->"},
	} {
		t.Run(c.name, func(t *testing.T) {
			note := "# Heading\n\nOne.\n\n" + c.line + "\n"
			got, err := Render(config(t), []byte(note), "testdata/docs", true)
			if err == nil {
				t.Fatalf("Render accepted a note carrying %q, and built %d blocks", c.marker, len(got.Blocks))
			}
			if !strings.Contains(err.Error(), "line 5") {
				t.Errorf("the refusal does not name line 5: %v", err)
			}
			if !strings.Contains(err.Error(), c.marker) {
				t.Errorf("the refusal does not name %q: %v", c.marker, err)
			}
			if len(got.Blocks) != 0 {
				t.Errorf("a refused note builds no block, and this one built %d", len(got.Blocks))
			}
		})
	}
}

// TestAnAuthorsOwnCommentIsStillAPencil is the other half of the line. Only
// gdoc's two comments are markers; a comment whose words happen to start with
// the word "image" is the author's aside and comes out grey.
func TestAnAuthorsOwnCommentIsStillAPencil(t *testing.T) {
	out := walk(t, "# Heading\n\nThe figure <!-- image sizes are wrong --> is next.\n")

	run := runElement(t, out.Blocks, "image sizes are wrong")
	rPr := run.SelectElement("w:rPr")
	if rPr == nil {
		t.Fatal("the aside carries no w:rPr, want the pencil")
	}
	if got := value(rPr, "w:rStyle"); got != "Pencil" {
		t.Errorf("the aside names style %q, want Pencil", got)
	}
}
