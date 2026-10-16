// The footnotes rule of the round trip comparison, asked offline. The live run
// needs a published document and a network; the rule itself is a reading of a
// note, so it is pinned here over the two notes that hold the shapes: the
// kitchen sink, whose second note is defined first, and the edge cases, whose
// label is a word and whose second label is marked twice.

package live

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTheFootnoteRuleNumbersTheNotesTheWayTheDocumentWill reads both notes and
// states every number as a literal, because the numbering is the whole claim.
func TestTheFootnoteRuleNumbersTheNotesTheWayTheDocumentWill(t *testing.T) {
	for _, c := range []struct {
		note  string
		lines map[string]string
		extra []string
		plain []string
	}{{
		note: "01-kitchen-sink.md",
		lines: map[string]string{
			"Two of the claims above are cited under the page rather than argued in the sentence,[^master] which is where a citation costs a reader least.[^cost]": "Two of the claims above are cited under the page rather than argued in the sentence,[^1] which is where a citation costs a reader least.[^2]",
			// The definitions are written the other way round, and the number
			// is the mark's rather than the definition's.
			"[^cost]: Principle 4, every word costs a reader's attention.":               "[^2]: Principle 4, every word costs a reader's attention.",
			"[^master]: The master document the drift gate measures this build against.": "[^1]: The master document the drift gate measures this build against.",
		},
		plain: []string{"[^1]: Principle 4, every word costs a reader's attention.", "[^2]: anything"},
	}, {
		note: "05-edge-cases.md",
		lines: map[string]string{
			// One paragraph carrying three marks, two of them the same label:
			// each mark takes its own number.
			"The first section called Overview. There is a second one further down carrying exactly the same heading text, which is what makes the page-number lookup interesting: matched by text alone, both would resolve to whichever comes first.[^lookup] One label may be marked twice,[^twice] and the same label again here[^twice] takes a number of its own.": "The first section called Overview. There is a second one further down carrying exactly the same heading text, which is what makes the page-number lookup interesting: matched by text alone, both would resolve to whichever comes first.[^1] One label may be marked twice,[^2] and the same label again here[^3] takes a number of its own.",
			"[^lookup]: A label that is a word rather than a number.": "[^1]: A label that is a word rather than a number.",
			"[^twice]: One definition, and two notes under the page.": "[^2]: One definition, and two notes under the page.",
		},
		// The third mark is the second of one label, so read writes a third
		// definition line the note never had.
		extra: []string{"[^3]: One definition, and two notes under the page."},
		plain: []string{"[^1]: A label that is a word rather than a number.", "[^2]: One definition, and two notes under the page."},
	}} {
		t.Run(c.note, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(bodyNotes, c.note))
			if err != nil {
				t.Fatalf("read %s: %v", c.note, err)
			}
			numbering := numberFootnotes(src)

			for from, want := range c.lines {
				if got := numbering.numbered(from); got != want {
					t.Errorf("the line\n  %q\nis read as\n  %q\nwant\n  %q", from, got, want)
				}
			}
			for _, line := range c.extra {
				if !numbering.repeated(line) {
					t.Errorf("%q is not excused as a second definition the document wrote", line)
				}
			}
			for _, line := range c.plain {
				if numbering.repeated(line) {
					t.Errorf("%q is excused as a line the note never had, and it is the note's own", line)
				}
			}
			if got := numbering.numbered("A line with no footnote in it."); got != "A line with no footnote in it." {
				t.Errorf("a line with no mark came back as %q", got)
			}
		})
	}
}

// TestARoundTripOverAFootnoteLeavesNothingUnnamed is the two rules through
// roundTrip itself, over the shape the live run will see: one label marked
// twice, numbered in the prose, with a rule and a definition line per mark
// under the body. Nothing of it may reach the unnamed list.
func TestARoundTripOverAFootnoteLeavesNothingUnnamed(t *testing.T) {
	note := []byte("A claim.[^a] And again.[^a]\n\n[^a]: the detail.\n")
	body := "A claim.[^1] And again.[^2]\n\n---\n\n[^1]: the detail.\n[^2]: the detail.\n"

	report := roundTrip(note, body)

	for _, u := range report.unnamed {
		t.Errorf("a difference no name covers, %s: %q", u.side, u.text)
	}
	named := map[string]int{}
	for _, n := range report.named {
		named[n.name] = n.count
	}
	// The rule and the second definition line are the two the file added, and
	// the prose line and the definition line are the two the note spelled with
	// a label.
	if named["footnotes"] != 4 {
		t.Errorf("the footnotes name covered %d lines, want the four: %v", named["footnotes"], report.named)
	}
}
