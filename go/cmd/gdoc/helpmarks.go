// gdoc help marks: the seven authoring marks as the house style holds them.
//
// It is a topic and not a command. A mark belongs to no one command: build and
// publish write them into a docx, export reads them back out of a Doc, and the
// palette itself is house.yaml's. So there is nothing for a command entry to
// run, and a colleague still needs one place to ask what a green highlight
// means and which colour it lands as.
//
// The seven come out of house.Load every time the topic is answered. Nothing
// here and nothing in a skill carries a second copy of the palette: a colour
// changed in house.yaml is the colour this prints, and a skill that wants the
// list runs the topic.
//
// The object and the words a person reads walk the same rows, as the command
// help does, so what a skill parses and what a colleague reads cannot
// disagree.

package main

import (
	"fmt"
	"io"
	"strings"

	"gdoc/internal/emit"
	"gdoc/internal/house"
)

// The fate of a mark: whether the two commands that make a document count it
// on their envelope. Counted is what house.yaml calls a debt, and it is a
// fact about the envelope rather than advice about the note. Whether a
// document carrying one may be published is the publish skill's to ask, which
// is the whole reason the binary says a number here and not a word like
// "blocking".
const (
	fateCounted = "counted"
	fateCarried = "carried"
)

// marksReport is what `gdoc help marks` prints. One field, so a skill reading
// `data.marks` of this topic and `data.commands` of the command help never
// reads one for the other.
type marksReport struct {
	Marks []markEntry `json:"marks"`
}

// markEntry is one mark as a caller reads it: what an author writes, how it
// reaches the document, the colour it lands as, and whether the envelope
// counts it. Syntax is always there, a list even where it holds one opener,
// because a skill telling "takes none" from "said nothing" is why the help
// exists at all.
type markEntry struct {
	Name   string   `json:"name"`
	Syntax []string `json:"syntax"`
	Kind   string   `json:"kind"`
	Colour string   `json:"color"`
	Fate   string   `json:"fate"`
}

// cmdHelpMarks is the topic answered. The house style is loaded rather than
// read from a table here, so a draft style that has drifted is refused naming
// itself rather than half printed.
//
// Like the command help, the words go to stderr and the object to stdout, and
// on a terminal that asked for neither the object is dropped: see run().
// TestHelpMarksPrintsTheSevenFromTheBinary.
func cmdHelpMarks(json bool, errOut io.Writer) emit.Result {
	cfg, err := house.Load()
	if err != nil {
		return failed(err)
	}
	entries := markEntries(cfg)
	writeMarks(errOut, entries)
	return emit.Result{OK: true, Data: marksReport{Marks: entries}, Screen: !json}
}

// markEntries is the palette in the order house.MarkNames gives, which is the
// order render writes the character styles in. Each field is copied, so
// nothing a caller is handed points back into the style.
func markEntries(cfg *house.Config) []markEntry {
	names := house.MarkNames()
	entries := make([]markEntry, 0, len(names))
	for _, name := range names {
		mark := cfg.Marks[name]
		entries = append(entries, markEntry{
			Name:   name,
			Syntax: append([]string{}, mark.Syntax...),
			Kind:   mark.Kind,
			Colour: mark.Color,
			Fate:   fateOf(mark),
		})
	}
	return entries
}

// fateOf is the one word that says whether the envelope counts this mark.
func fateOf(m house.Mark) string {
	if m.Debt {
		return fateCounted
	}
	return fateCarried
}

// marksTitle is what the table opens with, and marksFoot the line that says
// where the counting shows up, because a reader who met
// "counted" without it would not know on what.
const (
	marksTitle = "The seven authoring marks, from the house style."
	marksFoot  = "A counted mark is on body.marks of the build and publish envelopes, zero included."
)

// writeMarks is the words of the topic, the same table on every stream.
//
// This is the one page that draws no box, and the reason is the emoji. Six of
// the seven marks are written with one, internal/tty measures a column as a
// rune, and an emoji takes two columns on a terminal. So every row carrying
// one would run a column past the border and the box would come out ragged.
// A plain table that lines up beats a box that does not, and widening the
// measurement is a change to internal/tty that every screen would have to be
// re-recorded for.
func writeMarks(errOut io.Writer, entries []markEntry) {
	fmt.Fprint(errOut, marksProse(entries))
}

// marksProse is the seven as plain text, the name and what it is aligned the
// way one command's flags are.
func marksProse(entries []markEntry) string {
	var b strings.Builder
	b.WriteString(marksTitle + "\n\n")
	width := 0
	for _, e := range entries {
		width = max(width, len(e.Name))
	}
	for _, e := range entries {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, e.Name, markWords(e))
	}
	b.WriteString("\n" + marksFoot + "\n")
	return b.String()
}

// markWords is one mark said in a line: what an author writes, how it reaches
// the document, the colour it lands as, and its fate.
func markWords(e markEntry) string {
	return fmt.Sprintf("%s  %s %s, %s",
		strings.Join(e.Syntax, " "), e.Kind, e.Colour, e.Fate)
}
