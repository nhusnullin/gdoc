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
//
// A comment marker is one of the eight and is asked about here too, because
// the escape has to survive every check the note then meets: the opener is
// also a comment's, so the unclosed-opener refusal in comment.go reads the
// same line, and counting the escaped opener there refused this note while
// the marker refusal was leaving it alone.
func TestBuildLeavesAnEscapedMarkerAlone(t *testing.T) {
	for _, line := range []string{
		`The insertion marker is written \{+words\+} in the export.`,
		`A floating picture comes home as \<!-- image: floating, kix.p1 -->.`,
		`A tab line is written \<!-- tab t.0: Operations -->.`,
	} {
		note := "# Heading\n\n" + line + "\n"
		if _, err := Render(config(t), []byte(note), "testdata/docs", true); err != nil {
			t.Fatalf("Render refused a note whose markers are all escaped: %v", err)
		}
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

// TestAnAsideOpeningLikeGdocsOwnCommentSaysSo. Four of the eight markers are
// a comment opening with an ordinary English word, and this milestone is what
// made them so: before it, every marker was syntax nobody types by hand, and
// "the note was exported out of a document" was true of every note that
// carried one. An author who writes "<!-- tab order matters -->" for
// themselves exported nothing, so that sentence sends them looking for an
// export that never happened. The refusal names the opener as gdoc's own and
// says what to do about it instead.
func TestAnAsideOpeningLikeGdocsOwnCommentSaysSo(t *testing.T) {
	for _, c := range []struct{ name, note string }{
		{"a tab", "One.\n\nText with <!-- tab order matters --> inside.\n"},
		{"an image", "One.\n\nA <!-- image: see the appendix --> aside.\n"},
		{"a drawing", "One.\n\nA <!-- drawing: by hand --> aside.\n"},
		{"an object", "One.\n\nA <!-- object: of the exercise --> aside.\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Render(config(t), []byte(c.note), "testdata/docs", true)
			if err == nil {
				t.Fatal("the note was built, and an opener of gdoc's own has to be refused")
			}
			got := err.Error()
			if !strings.Contains(got, "line 3") {
				t.Errorf("the refusal is %q, want it to name line 3", got)
			}
			if strings.Contains(got, "exported out of a document") {
				t.Errorf("the refusal is %q, and it blames an export the author never made", got)
			}
			if !strings.Contains(got, "reword") {
				t.Errorf("the refusal is %q, and it names no way out for the author", got)
			}
		})
	}
}

// The other side of it: a marker that is gdoc's own markup keeps the sentence
// it has always had, because a note carrying one really did come out of a
// document.
func TestAMarkupMarkerStillNamesTheExport(t *testing.T) {
	note := "One.\n\nThe team {+meets weekly+}[s:AAA] in Q3.\n"

	_, err := Render(config(t), []byte(note), "testdata/docs", true)
	if err == nil {
		t.Fatal("the note was built, and a marker has to be refused")
	}
	if !strings.Contains(err.Error(), "exported out of a document") {
		t.Errorf("the refusal is %q, want the export named", err.Error())
	}
}
