// The marks on the way home. The house style says which colour each mark is
// and what an author types for it; this file turns the two into the list the
// projection writes back, and counts the colours no mark claims.
//
// doc.go holds why a colour that is not a mark is left as plain words.

package export

import (
	"strings"

	"gdoc/internal/docs"
	"gdoc/internal/house"
	"gdoc/internal/view"
)

// plainHighlight is the syntax an author writes for a highlight carrying no
// colour emoji, and the delimiter every highlight is written with. The house
// style gives it to one of the seven marks, and that mark is the one written
// without an emoji.
const plainHighlight = "=="

// Palette is the house marks as the projection writes them back, in the order
// house.yaml states them.
//
// A highlight is Obsidian's "==", the mark's own colour emoji and one space,
// which is the space internal/body takes off again, so a phrase published
// from a note comes home spelled the way it left. The plain highlight carries
// no emoji: an author who wrote ==words== gets ==words== back.
//
// A mark that is not a highlight is the run's own text colour, and gdoc reads
// exactly one span for it, Pandoc's "[words]{.pencil}", whose attribute is the
// mark's syntax.
func Palette(cfg *house.Config) []view.Mark {
	if cfg == nil {
		return nil
	}
	names := house.MarkNames()
	out := make([]view.Mark, 0, len(names))
	for _, name := range names {
		m, ok := cfg.Marks[name]
		if !ok {
			continue
		}
		if m.IsHighlight() {
			out = append(out, view.Mark{
				Background: m.Color,
				Open:       plainHighlight + emoji(m),
				Shut:       plainHighlight,
			})
			continue
		}
		out = append(out, view.Mark{
			Colour: m.Color,
			Open:   "[",
			Shut:   "]{" + m.Syntax[0] + "}",
		})
	}
	return out
}

// emoji is the colour emoji a highlight is written with, and the space after
// it, or nothing at all for the mark an author may write bare.
//
// A mark may be typed in more than one way and the file holds one of them.
// The bare delimiter wins where the house gives it to this mark, because
// ==words== is what an author writes when they mean nothing by the colour,
// and coming home as ==🟡 words== would put an instruction in the file that
// the author never typed.
func emoji(m house.Mark) string {
	var colour string
	for _, syntax := range m.Syntax {
		if syntax == plainHighlight {
			return ""
		}
		if colour == "" {
			colour = syntax + " "
		}
	}
	return colour
}

// otherColours counts the backgrounds a document carries that no mark in the
// palette claims, by hex, over the blocks this projection wrote. Adjacent runs
// of one colour are one count, because Docs splits a run wherever the
// formatting changes and three counts for one highlighted phrase would say the
// document holds three.
//
// Backgrounds alone. A background is somebody marking words, so one gdoc has
// no syntax for is a mark it brought home as prose and the session should know
// how many. A text colour is how a document is designed: counting every
// coloured word in a brochure would say nothing, and the one text colour that
// is a mark comes home as its span when it matches.
//
// skip is the caller's, so the cover's own yellow placeholders are not
// counted: they are the prelude, they leave the file, and the probe measured
// them as a different hex from the yellow mark.
func otherColours(bs []docs.Block, skip func(docs.Block) bool, palette []view.Mark, into map[string]int) {
	for _, b := range bs {
		if skip != nil && skip(b) {
			continue
		}
		switch {
		case b.Paragraph != nil:
			countRuns(b.Paragraph.Runs, palette, into)
		case b.Table != nil:
			for _, row := range b.Table.Rows {
				for _, c := range row {
					otherColours(c.Blocks, skip, palette, into)
				}
			}
		case b.TOC != nil:
			otherColours(b.TOC.Blocks, skip, palette, into)
		}
	}
}

// countRuns counts one paragraph's unclaimed backgrounds, one per span of
// adjacent runs carrying the same one.
func countRuns(rs []docs.Run, palette []view.Mark, into map[string]int) {
	last := ""
	for _, r := range rs {
		hex := unclaimed(r, palette)
		if hex != "" && !strings.EqualFold(hex, last) {
			into[hex]++
		}
		last = hex
	}
}

// unclaimed is the run's background when no mark in the palette names that
// colour, and nothing otherwise.
func unclaimed(r docs.Run, palette []view.Mark) string {
	if r.Kind != docs.KindText || r.Background == "" {
		return ""
	}
	for _, m := range palette {
		if strings.EqualFold(m.Background, r.Background) {
			return ""
		}
	}
	return r.Background
}
