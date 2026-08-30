package pagination

import (
	"testing"

	"spike/gdocgo/internal/contents"
)

func headings(texts ...string) []contents.Heading {
	out := make([]contents.Heading, len(texts))
	for i, text := range texts {
		out[i] = contents.Heading{Level: 1, Text: text}
	}
	return out
}

func TestEachHeadingTakesItsOwnOutlinePage(t *testing.T) {
	outline := []Entry{{"1-Purpose ", 4}, {"2-Scope ", 5}, {"3-Governance ", 7}}

	pages := Resolve(outline, headings("1-Purpose", "2-Scope", "3-Governance"))

	for heading, want := range map[string]int{"1-Purpose": 4, "2-Scope": 5, "3-Governance": 7} {
		if pages[heading] != want {
			t.Errorf("%q = %d, want %d", heading, pages[heading], want)
		}
	}
}

func TestTwoSectionsWithTheSameNameResolveSeparately(t *testing.T) {
	// Matched by text alone, both would land on whichever comes first. The
	// cursor only moves forward, which is what keeps them apart.
	outline := []Entry{{"Overview", 4}, {"Detail", 5}, {"Overview", 8}}

	pages := Resolve(outline, headings("Overview", "Detail", "Overview"))

	if pages["Overview"] != 8 {
		t.Errorf("the second Overview resolved to %d, want 8", pages["Overview"])
	}
}

func TestAHeadingTheOutlineNeverMentionsIsLeftOut(t *testing.T) {
	// A blank page number is honest. A guessed one is not.
	outline := []Entry{{"1-Purpose", 4}}

	pages := Resolve(outline, headings("1-Purpose", "2-Not in the outline"))

	if _, present := pages["2-Not in the outline"]; present {
		t.Error("an unmatched heading must be absent from the result, not guessed at")
	}
}

func TestOutlineEntriesTheHeadingsDoNotShareAreSkipped(t *testing.T) {
	// Google outlines the front matter's own headings too.
	outline := []Entry{{"Version Control", 2}, {"Contents", 3}, {"1-Purpose", 4}}

	pages := Resolve(outline, headings("1-Purpose"))

	if pages["1-Purpose"] != 4 {
		t.Errorf("1-Purpose = %d, want 4", pages["1-Purpose"])
	}
}

func TestTrailingSpaceInAnOutlineTitleIsIgnored(t *testing.T) {
	// Google writes the heading with the trailing space its paragraph carries.
	pages := Resolve([]Entry{{"1-Purpose ", 4}}, headings("1-Purpose"))

	if pages["1-Purpose"] != 4 {
		t.Errorf("= %d, want the trailing space normalised away", pages["1-Purpose"])
	}
}

func TestDriftNamesOnlyWhatMoved(t *testing.T) {
	written := map[string]int{"a": 4, "b": 5}
	published := map[string]int{"a": 4, "b": 6}

	moved := Drift(written, published)

	if len(moved) != 1 {
		t.Fatalf("moved = %#v, want only b", moved)
	}
	if moved["b"] != [2]int{5, 6} {
		t.Errorf("b = %v, want written 5 and published 6", moved["b"])
	}
}

func TestNoDriftMeansTheContentsListDescribesItsOwnDocument(t *testing.T) {
	same := map[string]int{"a": 4}

	if got := Drift(same, same); len(got) != 0 {
		t.Errorf("drift = %#v, want none", got)
	}
}
