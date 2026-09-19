// This file is the projection itself: one file per tab, the pieces the house
// prelude was made of, and the warnings. strip.go holds what a prelude is and
// how a heading loses its number, and doc.go holds the package comment.

package export

import (
	"fmt"
	"strings"

	"gdoc/internal/docs"
	"gdoc/internal/frontmatter"
	"gdoc/internal/prelude"
	"gdoc/internal/view"
)

// TabFile is one tab of a document as Markdown. Body is the text alone: the
// gdoc: block goes in front of it in File, because what the block says is the
// caller's and the same body is written beside a note and on its own.
type TabFile struct {
	TabID    string `json:"tab_id,omitempty"`
	TabTitle string `json:"tab_title,omitempty"`
	Body     string `json:"-"`
}

// Piece is one thing this projection took out of the body, with the text it
// held. It decides nothing about it: the session reading the file compares the
// text with the note's own keys and says whether anything was lost.
type Piece struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// The kinds a piece may be. Every one of them is a position in the house
// layout rather than a judgement about the words.
const (
	PieceCover         = "cover"
	PieceTable         = "table"
	PieceLegend        = "legend"
	PieceContents      = "contents"
	PieceHeadingNumber = "heading number"
)

// PictureNames says what the picture with this object id is written as, or
// nothing for a picture with no file. It is a function rather than a map
// because the caller numbers its files as it writes them.
type PictureNames func(objectID string) string

// Project is the document as Markdown, one file per tab, with the house
// prelude taken out and listed.
//
// Nothing here reaches the network or the disk, and nothing here judges. What
// comes back is the files, every piece the prelude was made of, and the
// warnings: what the projection could not carry, and what the file loses on
// its way to becoming a note.
func Project(d *docs.Document, pics PictureNames) ([]TabFile, []Piece, []string) {
	if d == nil {
		return nil, nil, nil
	}
	files := make([]TabFile, 0, len(d.Tabs))
	var pieces []Piece
	var warnings []string

	marks, err := prelude.Markers(d)
	if err != nil {
		// A marker this run cannot read as one span is the prelude package's
		// refusal, and the answer here is the answer to anything else it
		// cannot recognise: the words stay in the file and the reason is
		// named. Nothing is stripped on a guess.
		warnings = append(warnings, fmt.Sprintf("%s, so nothing was stripped and the prelude is in the file", err))
		marks = nil
	}

	for _, t := range d.Tabs {
		skip, cut, w := stripPrelude(t, markerFor(marks, t.ID))
		pieces, warnings = append(pieces, cut...), append(warnings, w...)

		body, tw := view.Project(oneTab(d, t), view.Options{
			Picture:     picture(pics),
			Skip:        skip,
			ChipTargets: true,
		})
		warnings = append(warnings, tw...)

		body, numbers := stripHeadingNumbers(body)
		pieces = append(pieces, numbers...)
		warnings = append(warnings, footnoteWarnings(t)...)

		files = append(files, TabFile{TabID: t.ID, TabTitle: t.Title, Body: body})
	}
	return files, pieces, warnings
}

// File is one tab's body with the gdoc: block in front of it, which is the
// whole of the front matter export writes: the author's own keys are the
// session's to add when it makes the file a note.
//
// It goes through frontmatter.Write, the one writer of that block, so a file
// export creates and a note export stamps carry the same bytes for the same
// block.
func File(body string, b *frontmatter.Block) ([]byte, error) {
	return frontmatter.Write([]byte(body), b)
}

// picture wraps the caller's naming so view is handed nothing when there is
// nothing to name: a nil Picture is what leaves the placeholder and its
// warning where they are.
func picture(pics PictureNames) func(string) string {
	if pics == nil {
		return nil
	}
	return func(id string) string { return pics(id) }
}

// markerFor is this tab's house prelude marker, or nothing. Two markers in one
// tab is prelude.Decide's refusal rather than this one's: a projection that
// picked one of them would strip a span on a guess, so it strips neither and
// says so.
func markerFor(marks []prelude.Marker, tab string) *prelude.Marker {
	var found *prelude.Marker
	for i, m := range marks {
		if m.Tab != tab {
			continue
		}
		if found != nil {
			return nil
		}
		found = &marks[i]
	}
	return found
}

// oneTab is the document as this tab alone sees it: the tab, and the comment
// ranges anchored in it.
//
// One file is one tab, so the projection is asked one tab at a time. The
// ranges are filtered with it because a range naming another tab would be
// warned about here as a range naming a tab the document does not have, and
// the tab it names is in the file beside this one.
//
// A link to a heading in another tab is named by its id rather than by that
// heading's words, which is what view does for a heading it cannot see. That
// heading is in another file, so a "#slug" into this one would point at
// nothing.
func oneTab(d *docs.Document, t docs.Tab) *docs.Document {
	ranges := map[string]docs.Range{}
	for id, r := range d.CommentRanges {
		if r.Tab == t.ID {
			ranges[id] = r
		}
	}
	if len(ranges) == 0 {
		ranges = nil
	}
	return &docs.Document{
		ID:            d.ID,
		Title:         d.Title,
		RevisionID:    d.RevisionID,
		Tabs:          []docs.Tab{t},
		CommentRanges: ranges,
		Footnotes:     d.Footnotes,
	}
}

// footnoteWarnings names what a file holding a footnote loses on its way to
// becoming a note. The text is written and the notes are under the body, so
// nothing is dropped here; what is warned about is what happens next.
//
// Three sentences, because there are three losses and a reader has to act on
// each: publish refuses a note that holds one, a suggestion inside a footnote
// arrives as plain text, and a comment anchored inside one is never marked.
func footnoteWarnings(t docs.Tab) []string {
	n := 0
	walk(t.Body, func(b docs.Block) {
		if b.Paragraph == nil {
			return
		}
		for _, r := range b.Paragraph.Runs {
			if r.Kind == docs.KindFootnoteRef {
				n++
			}
		}
	})
	if n == 0 {
		return nil
	}
	return []string{
		fmt.Sprintf("this tab holds %d footnote reference(s), written as [^n] with the notes under a --- rule at the end: publish refuses a note that holds a footnote, so they have to be resolved before this file becomes one", n),
		"a suggestion inside a footnote is written as plain text, because the read does not carry the ids there: docs/backlog/suggestions-inside-footnotes.md",
		"a comment anchored inside a footnote, a header or a footer carries no marker in this file: docs/backlog/comment-anchors-in-headers-and-footnotes.md",
	}
}

// walk calls f on every block of a body, table cells and contents entries
// included, which is where a footnote reference may also sit.
func walk(bs []docs.Block, f func(docs.Block)) {
	for _, b := range bs {
		f(b)
		switch {
		case b.Table != nil:
			for _, row := range b.Table.Rows {
				for _, c := range row {
					walk(c.Blocks, f)
				}
			}
		case b.TOC != nil:
			walk(b.TOC.Blocks, f)
		}
	}
}

// plainText is a block's own words, with the runs joined and the whitespace
// collapsed. It is what a piece holds, and it is the document's text alone: a
// placeholder, a marker and a chip's label are the projection's, and a piece
// says what stood in the document rather than what this file would have
// printed for it.
func plainText(b docs.Block) string {
	switch {
	case b.Paragraph != nil:
		var sb strings.Builder
		for _, r := range b.Paragraph.Runs {
			if r.Kind == docs.KindText {
				sb.WriteString(r.Text)
			}
		}
		return collapse(sb.String())
	case b.Table != nil:
		rows := make([]string, 0, len(b.Table.Rows))
		for _, row := range b.Table.Rows {
			cells := make([]string, 0, len(row))
			for _, c := range row {
				cells = append(cells, cellText(c))
			}
			rows = append(rows, strings.Join(cells, " | "))
		}
		return strings.Join(rows, "; ")
	case b.TOC != nil:
		entries := make([]string, 0, len(b.TOC.Blocks))
		for _, inner := range b.TOC.Blocks {
			if s := plainText(inner); s != "" {
				entries = append(entries, s)
			}
		}
		return strings.Join(entries, ", ")
	}
	return ""
}

func cellText(c docs.Cell) string {
	parts := make([]string, 0, len(c.Blocks))
	for _, b := range c.Blocks {
		if s := plainText(b); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// collapse is one run of whitespace as one space, with the ends trimmed. A
// piece is read in a reply beside other pieces, and the document's own line
// breaks inside one would break that list into lines nothing joins back up.
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }
