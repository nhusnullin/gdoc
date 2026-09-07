package propose

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"gdoc/internal/docs"
)

// indexed is one paragraph as text, with the document index of every byte
// offset in it and a flag saying which of those bytes are somebody's pending
// suggestion.
//
// The three travel together because a match has to answer all three questions
// at once: where the words are in the text, what index the Docs API calls that
// place, and whether the words are written or only proposed. Building them
// separately is how the second answer drifts from the first.
type indexed struct {
	text string
	// pos holds one document index per byte offset, plus one at the end, so a
	// match at byte offset i spans pos[i] to pos[i+len].
	pos []int
	// suggested holds one flag per byte offset: true when that byte came out of
	// a run carrying an insertion or a deletion id.
	suggested []bool
}

// FindSpan is where a proposal is placed, and it is placed by its words.
//
// An index computed from a read a minute ago is the hazard the whole Docs API
// has: the document moves under it and the delete then cuts a word in half. So
// the skill hands over the exact words, this reads a document that just came
// back, and the index never leaves the run that produced it.
//
// Exactly one match is required. None is a refusal, and so is more than one:
// picking the first would place the change somewhere nobody chose, and the
// caller can always quote more of the sentence.
//
// Indexes are UTF-16 code units, which is what the Docs API counts. A rune
// outside the basic plane, an emoji most of the time, is two of them.
func FindSpan(d *docs.Document, quoted string) (docs.Range, error) {
	if quoted == "" {
		return docs.Range{}, fmt.Errorf("the proposal quotes no text, so there is nothing to find")
	}
	if len(d.Tabs) != 1 {
		return docs.Range{}, fmt.Errorf("the document has %d tabs, and a proposal is placed in a document with one", len(d.Tabs))
	}
	tab := d.Tabs[0]

	var found []docs.Range
	for _, p := range paragraphs(tab.Body) {
		found = append(found, p.matches(tab.ID, quoted)...)
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		// Text inside a pending suggestion is not text a proposal may stand on,
		// and it lands here rather than in its own message: from the caller's
		// side both mean the same thing, which is that these words are not
		// somewhere a change can be placed.
		return docs.Range{}, fmt.Errorf("the quoted text %q was not found in the document as written; it may have changed, or it may be inside a pending suggestion", quoted)
	default:
		return docs.Range{}, fmt.Errorf("the quoted text %q occurs %d times; quote more of the sentence", quoted, len(found))
	}
}

// StartsAt says whether the document carries want at the index r begins at.
//
// It is the second read-back's whole question. In the view that hides pending
// suggestions the original words must still be there, because a suggestion
// changes nothing until somebody accepts it. Finding the replacement there
// instead means the write was a direct edit wearing a suggestion's clothes.
func StartsAt(d *docs.Document, r docs.Range, want string) bool {
	for _, t := range d.Tabs {
		if t.ID != r.Tab {
			continue
		}
		for _, p := range paragraphs(t.Body) {
			// A paragraph's last index is the next paragraph's first, because
			// the newline belongs to this one. So an offset at the very end of
			// the text is this paragraph answering about the next paragraph's
			// first character, and the walk goes on instead.
			if off, ok := p.offsetOf(r.Start); ok && off < len(p.text) {
				return strings.HasPrefix(p.text[off:], want)
			}
		}
	}
	return false
}

// matches is every place quoted occurs in this paragraph as written text.
func (p indexed) matches(tabID, quoted string) []docs.Range {
	var out []docs.Range
	for at := 0; ; {
		i := strings.Index(p.text[at:], quoted)
		if i < 0 {
			return out
		}
		i += at
		at = i + 1 // one byte, not one match: two matches may overlap.
		if p.anySuggested(i, i+len(quoted)) {
			continue
		}
		out = append(out, docs.Range{Tab: tabID, Start: p.pos[i], End: p.pos[i+len(quoted)]})
	}
}

// anySuggested says whether any byte of the half-open range came out of a run
// carrying a suggestion id.
func (p indexed) anySuggested(start, end int) bool {
	for i := start; i < end && i < len(p.suggested); i++ {
		if p.suggested[i] {
			return true
		}
	}
	return false
}

// offsetOf turns a document index back into a byte offset in this paragraph's
// text, when the index is one this paragraph covers.
func (p indexed) offsetOf(index int) (int, bool) {
	for off, at := range p.pos {
		if at == index {
			return off, true
		}
	}
	return 0, false
}

// paragraphs is every paragraph of a body, indexed, tables walked into.
//
// A cell's paragraphs are paragraphs: they carry their own indexes in the same
// tab, so text in a table is text a proposal can be placed in, and refusing to
// look there would report "not found" about words that are plainly on screen.
func paragraphs(bs []docs.Block) []indexed {
	var out []indexed
	for _, b := range bs {
		if b.Paragraph != nil {
			out = append(out, index(b.Paragraph.Runs))
			continue
		}
		for _, row := range b.Table {
			for _, c := range row {
				out = append(out, paragraphs(c.Blocks)...)
			}
		}
	}
	return out
}

// index walks one paragraph's runs and builds the three parallel views of it.
//
// The document index advances by UTF-16 code units per rune, counted from the
// run's own start index rather than from the paragraph's: Docs numbers each run
// itself, and a run this walk skips still moves the numbering on.
func index(runs []docs.Run) indexed {
	var b strings.Builder
	var p indexed
	end := 0
	for _, r := range runs {
		if r.Kind != docs.KindText {
			continue
		}
		suggested := len(r.InsertionIDs) > 0 || len(r.DeletionIDs) > 0
		at := r.StartIndex
		raw := []byte(r.Text)
		for off := 0; off < len(raw); {
			c, size := utf8.DecodeRune(raw[off:])
			for k := 0; k < size; k++ {
				p.pos = append(p.pos, at)
				p.suggested = append(p.suggested, suggested)
			}
			at += len(utf16.Encode([]rune{c}))
			off += size
		}
		b.WriteString(r.Text)
		end = r.EndIndex
	}
	p.text = b.String()
	// One index per byte, and one more for the position just past the last
	// byte, so a match ending at the end of the paragraph has an end index.
	p.pos = append(p.pos, end)
	return p
}
