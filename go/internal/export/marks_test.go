// The marks on the way home: the syntax export writes for a colour the house
// claims, the words it writes for a colour nobody claims, and the read, which
// writes neither.
//
// Every hex, emoji and delimiter here is a literal. A test that read the
// palette out of house.yaml would follow the file wherever somebody moved it
// and agree with a document gdoc never wrote.

package export

import (
	"strings"
	"testing"

	"gdoc/internal/house"
	"gdoc/internal/view"
)

// palette is the house marks as the projection writes them. It goes through
// Palette rather than a list of its own, because what this file is testing is
// that the seven in house.yaml reach a file as Obsidian's syntax.
func palette(t *testing.T) []view.Mark {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("house.Load: %v", err)
	}
	return Palette(cfg)
}

// marked is the marks fixture projected with the palette passed, which is what
// export does, and the body of its one file.
func marked(t *testing.T) (string, map[string]int) {
	t.Helper()
	files, _, other, _ := Project(fixture(t, "marks.json"), nil, palette(t))
	if len(files) != 1 {
		t.Fatalf("marks.json gave %d files, want 1", len(files))
	}
	return files[0].Body, other
}

// The six highlights come home as Obsidian's ==, with the colour emoji at the
// start of the words and the one space internal/body takes off again, and the
// pencil comes home as Pandoc's span. Yellow is the plain highlight and
// carries no emoji: an author writing ==words== gets ==words== back.
func TestExportWritesObsidiansSyntaxForEachPaletteColour(t *testing.T) {
	body, _ := marked(t)
	want := "==🔴 red== ==🟠 orange== ==yellow== ==🟢 green== ==🔵 blue== ==🟣 purple== [pencil]{.pencil}"
	if !strings.Contains(body, want) {
		t.Errorf("the seven marks came home as\n%s\nwant the line\n%s", body, want)
	}
}

// A colour the house claims no mark for is the words as they stand, because
// gdoc did not write it and has no syntax that means it. It is counted by hex
// so the session reading the file can say how much of the document it is
// bringing home as plain prose, and adjacent runs of one colour are one count:
// Docs splits a run wherever the formatting changes, and three counts for one
// highlighted phrase would say the document holds three.
//
// #FFFF00 is Word's named yellow, which the cover's own placeholders carry and
// which the mark probe measured as a different colour from the mark yellow
// #FFD500. Telling the two apart is what this count rests on.
func TestAColourOutsideThePaletteIsPlainWordsAndCounted(t *testing.T) {
	body, other := marked(t)

	if !strings.Contains(body, "stale words and more.") {
		t.Errorf("the unclaimed colour did not come home as plain words:\n%s", body)
	}
	if got := other["#FFFF00"]; got != 2 {
		t.Errorf("other[#FFFF00] = %d, want 2", got)
	}
	if len(other) != 1 {
		t.Errorf("other = %v, want #FFFF00 alone", other)
	}
}

// Docs splits a run wherever the formatting changes, so a mark over a phrase
// that holds bold or code arrives as several runs sharing one background. They
// are one mark in the document and one mark in the file: a pair of == around
// each run would say the author marked the words twice.
func TestAMarkSplitOverSeveralRunsComesHomeWhole(t *testing.T) {
	body, _ := marked(t)
	if !strings.Contains(body, "==🟢 a claim to confirm==") {
		t.Errorf("the split green mark came home as\n%s", body)
	}
	if strings.Contains(body, "==🟢 a claim== ==🟢 to confirm==") {
		t.Errorf("the split green mark came home as two marks:\n%s", body)
	}
}

// read is unchanged. The words of a marked phrase reach a reviewing session
// and the mark around them does not, because a session quotes what it reads
// and a quote carrying == is not text the document holds.
func TestReadPrintsNoMark(t *testing.T) {
	text, _ := view.Text(fixture(t, "marks.json"))
	want := "The figure and a claim to confirm, pencilled, stale words and more.\n\n" +
		"red orange yellow green blue purple pencil\n"
	if text != want {
		t.Errorf("read printed\n%q\nwant\n%q", text, want)
	}
}
