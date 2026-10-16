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

// commentCloser shuts the comment a pencil is written as. It is a constant
// here and not a value off the house style because the house states what an
// author opens a mark with, and one pair has a closer that is not its opener.
// internal/body states the same string, and
// TestAPencilRunComesHomeAsAnHTMLComment reads this one back through a file.
const commentCloser = "-->"

// Palette is the house marks as the projection writes them back, in the order
// house.yaml states them.
//
// A highlight is Obsidian's "==", the mark's own colour emoji and one space,
// which is the space internal/body takes off again, so a phrase published
// from a note comes home spelled the way it left. The plain highlight carries
// no emoji: an author who wrote ==words== gets ==words== back.
//
// A mark that is not a highlight is the run's own text colour, and it comes
// home as the comment an author writes it as, "<!-- words -->", with the one
// space inside each delimiter that internal/body takes off again. The opener
// is the mark's first syntax rather than a constant here, because the house
// style is the one place the palette is stated, and the first of the two
// comment spellings is the one that is written: Obsidian's "%%" is a spelling
// an author may type and never one a file comes home in.
//
// The words inside it are literal, which is what Literal says: nothing in a
// comment is markdown, so the projection writes the document's own characters
// and names whatever markup it would otherwise have written around them.
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
			mark := view.Mark{
				Background: m.Color,
				Open:       plainHighlight + emoji(m),
				Shut:       plainHighlight,
			}
			// The mark an author may write bare says no colour, so words of
			// the author's own that open with a colour emoji would be read as
			// naming one. Where they do, the colour is written out.
			if emoji(m) == "" {
				if e := colourEmoji(m); e != "" {
					mark.Explicit = plainHighlight + e + " "
					mark.Ambiguous = paletteEmoji(cfg)
				}
			}
			out = append(out, mark)
			continue
		}
		out = append(out, view.Mark{
			Colour: m.Color,
			Open:   m.Syntax[0] + " ",
			Shut:   " " + commentCloser,
			// The spellings that are never written are still read by the
			// next build, so the projection has to know them: a document's
			// own "%%" is a loss to name, and one inside another mark's
			// words is a mark inside a mark.
			Others:  m.Syntax[1:],
			Literal: true,
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

// colourEmoji is the mark's own colour emoji, whether or not the mark may also
// be written bare.
func colourEmoji(m house.Mark) string {
	for _, syntax := range m.Syntax {
		if syntax != plainHighlight {
			return syntax
		}
	}
	return ""
}

// paletteEmoji is every colour emoji the house states. It is what the words of
// a bare highlight must not begin with: internal/body reads the first rune of
// a highlight as its colour, so "==🟢 go==" written for the plain mark would
// come home green with the emoji dropped from the words.
func paletteEmoji(cfg *house.Config) []string {
	var out []string
	for _, name := range house.MarkNames() {
		m, ok := cfg.Marks[name]
		if !ok || !m.IsHighlight() {
			continue
		}
		for _, syntax := range m.Syntax {
			if syntax != plainHighlight {
				out = append(out, syntax)
			}
		}
	}
	return out
}

// markedNotes is the footnotes whose colours say the author marked words
// inside them, out of the colours the read collected for each.
//
// It is the rule otherColours states, asked of a footnote: a background is
// somebody marking words, so any of them counts, and a text colour is how a
// document is designed unless it is the one the house claims, which is the
// pencil. A link inserted in Google Docs carries the link blue as a direct
// text colour on its run, and a footnote is where a citation lives: counting
// that would tell the session a mark came home plain in most of the footnotes
// anybody wrote by hand.
// TestAFootnoteWhoseOnlyColourIsALinksIsNotWarnedAboutAMark is the pin over
// the colour that is not a mark, and
// TestAPencilMarkInsideAFootnoteIsNamedAsALoss over the one that is: the
// pencil is a text colour and no background, so a rule counting backgrounds
// alone would say nothing about it.
func markedNotes(colours map[string]docs.NoteColours, palette []view.Mark) map[string]bool {
	out := map[string]bool{}
	for id, c := range colours {
		if len(c.Backgrounds) > 0 || claimedColour(c.Colours, palette) {
			out[id] = true
		}
	}
	return out
}

// claimedColour says whether one of these text colours is a mark's own.
func claimedColour(hexes []string, palette []view.Mark) bool {
	for _, hex := range hexes {
		for _, m := range palette {
			if m.Colour != "" && strings.EqualFold(m.Colour, hex) {
				return true
			}
		}
	}
	return false
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
// is a mark comes home as its comment when it matches.
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
