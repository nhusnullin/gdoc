package markers

import "testing"

// TestMarkersNamesEachOpener is the list itself, written out as literals rather
// than read off Pairs. A test that reads the constant follows it wherever
// somebody moves it, and these eight strings are what `read` has already
// written into every exported file: changing one of them without changing this
// test is changing what a marker is.
func TestMarkersNamesEachOpener(t *testing.T) {
	want := [][2]string{
		{"{+", "+}"},
		{"{-", "-}"},
		{"[[c:", "[[/c]]"},
		{"[s:", "]"},
	}
	if len(Pairs) != len(want) {
		t.Fatalf("there are %d markers, and the four gdoc writes are %v", len(Pairs), want)
	}
	for i, p := range Pairs {
		if p.Open != want[i][0] || p.Shut != want[i][1] {
			t.Errorf("Pairs[%d] = %q..%q, want %q..%q", i, p.Open, p.Shut, want[i][0], want[i][1])
		}
	}
}

// TestFindNamesTheFirstMarkerOnTheLine is the question every route out of the
// hub asks. The answer is the marker nearest the start of the line, because
// that is the one a person is told to look at.
func TestFindNamesTheFirstMarkerOnTheLine(t *testing.T) {
	for _, c := range []struct{ line, want string }{
		{"{+added words+}[s:AAA]", "{+"},
		{"{-gone words-}[s:BBB]", "{-"},
		{"[[c:AAA]]the range[[/c]]", "[[c:"},
		{"the tail of a suggestion +}[s:AAA]", "+}"},
		{"the tail of a deletion -}[s:AAA]", "-}"},
		{"the tail of an anchor [[/c]] and the rest", "[[/c]]"},
		{"the words then [s:AAA] alone", "[s:"},
		{"a sentence, then {+added+} and then [[c:AAA]]", "{+"},
	} {
		got, ok := Find(c.line)
		if !ok || got != c.want {
			t.Errorf("Find(%q) = %q, %v, want %q, true", c.line, got, ok, c.want)
		}
	}
}

// TestFindLeavesAnEscapedMarkerAlone is the other half, and it is what keeps
// the rule usable. `read` writes the author's own "{+" with a backslash in
// front of it, so a file exported out of a document that talked about markers
// comes back with escaped ones all through it, and none of them is gdoc's.
//
// The parity is view's: an even run of backslashes is the author's text, an odd
// one ends in gdoc's escape.
func TestFindLeavesAnEscapedMarkerAlone(t *testing.T) {
	for _, line := range []string{
		`the author wrote \{+ himself`,
		`\{+not a marker\+}`,
		`\[[c:AAA\]] is what he typed`,
		`\[s:AAA\]`,
		`a plain sentence with no marker in it`,
		`a markdown [link](https://example.com) and a list [1]`,
		`\\\{+ is three backslashes and an escape`,
	} {
		if got, ok := Find(line); ok {
			t.Errorf("Find(%q) = %q, true, and there is no marker of gdoc's in it", line, got)
		}
	}
}

// TestAnAuthorsBackslashDoesNotHideAMarker is the parity read the other way. Two
// backslashes are the author's one, written twice, and the marker behind them is
// gdoc's.
func TestAnAuthorsBackslashDoesNotHideAMarker(t *testing.T) {
	for _, c := range []struct{ line, want string }{
		{`\\{+added+}[s:AAA]`, "{+"},
		{`\\\\{-gone-}[s:AAA]`, "{-"},
	} {
		got, ok := Find(c.line)
		if !ok || got != c.want {
			t.Errorf("Find(%q) = %q, %v, want %q, true", c.line, got, ok, c.want)
		}
	}
}

// TestLinesCountsFromOne is the line number every refusal names, counted here
// rather than in each caller. A person opens the file at that line.
func TestLinesCountsFromOne(t *testing.T) {
	text := "# Heading\n\nOne.\n\nTwo.\n\nThe team {+meets weekly+}[s:AAA].\n"
	m, line := Lines(text)
	if m != "{+" || line != 7 {
		t.Errorf("Lines = %q, %d, want %q, 7", m, line, "{+")
	}
	if m, line := Lines("# Heading\n\nA plain note.\n"); m != "" || line != 0 {
		t.Errorf(`Lines over a note with no marker = %q, %d, want "", 0`, m, line)
	}
}
