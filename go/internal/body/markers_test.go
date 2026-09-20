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
