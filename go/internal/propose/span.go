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
	// pos holds one document index per byte offset, and nothing past the last
	// one. A match starts at pos[i], and its end is derived from the last rune
	// of the match rather than from the byte behind it: see matches for why the
	// byte behind it is the wrong place to ask.
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
// caller can always quote more of the sentence. An occurrence running across
// content the walk does not index counts towards that, so a quote that occurs
// once contiguously and once across a footnote mark is refused rather than
// placed on the contiguous one.
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
	crossed := 0
	for _, p := range paragraphs(tab.Body) {
		hits, over := p.matches(tab.ID, quoted)
		found = append(found, hits...)
		crossed += over
	}
	switch {
	case crossed > 0 && len(found) == 0:
		return docs.Range{}, fmt.Errorf(
			"the quoted text %q runs across content this read does not index, a footnote mark, a picture or a page break among them, so the span would take that content with it; quote a shorter run of words that does not cross it", quoted)
	case crossed > 0:
		// A crossing occurrence is still an occurrence. Placing the contiguous
		// one and saying nothing would pick for the caller, on a document where
		// the words they quoted appear more than once, which is the thing the
		// exactly-once rule exists to refuse. It is also what Carries rests on:
		// its doc comment reads the guarantee as "exactly once as written text",
		// and a dropped crossing match would be a second copy it can still find
		// after a direct edit took the first.
		return docs.Range{}, fmt.Errorf(
			"the quoted text %q occurs %d times as written text and %d more across content this read does not index, a footnote mark, a picture or a page break among them, so which occurrence was meant is not clear; quote more of the sentence",
			quoted, len(found), crossed)
	case len(found) == 1:
		return found[0], nil
	case len(found) == 0:
		// Text inside a pending suggestion is not text a proposal may stand on,
		// and it lands here rather than in its own message: from the caller's
		// side both mean the same thing, which is that these words are not
		// somewhere a change can be placed.
		return docs.Range{}, fmt.Errorf("the quoted text %q was not found in the document as written; it may have changed, or it may be inside a pending suggestion", quoted)
	default:
		return docs.Range{}, fmt.Errorf("the quoted text %q occurs %d times; quote more of the sentence", quoted, len(found))
	}
}

// Carries says whether the tab still holds want as written text.
//
// It is the second read-back's whole question, and it is asked by words rather
// than by index on purpose. The index the write was built from was counted in
// the view that shows pending suggestions; the preview hides them, so every
// position after one sits lower there. Asking at that index reports the second
// proposal of every run, and any document already carrying somebody's pending
// insertion, as a direct edit, which is the one warning that must never cry
// wolf.
//
// FindSpan has already required the quoted words to occur exactly once as
// written text, so a direct edit usually takes the only copy with it and this
// comes back false. Two things it cannot see on its own, and both are the price
// of asking by words rather than at an index:
//
// A replacement that contains the quote carries it through a direct edit, since
// the write puts the replacement where the quote was, so the words are still
// here and this still comes back true. Verify asks the second question in that
// shape, and this is why it has to.
//
// A second copy sitting inside somebody's pending suggested deletion is still
// shown by the preview, so a direct edit that took the written copy leaves this
// one behind. Nothing asks a second question there, and it is the cheaper of
// the two mistakes.
func Carries(d *docs.Document, tabID, want string) bool {
	for _, t := range d.Tabs {
		if t.ID != tabID {
			continue
		}
		for _, p := range paragraphs(t.Body) {
			if strings.Contains(p.text, want) {
				return true
			}
		}
	}
	return false
}

// matches is every place quoted occurs in this paragraph as written text, and
// the number of places it occurs across something the walk does not index.
//
// The second number is why this is not a plain search. index skips a run it has
// no text for, a footnote mark, a picture, an equation, a chip and a page break
// among them, but the document numbers them all, so the text either side of one
// is contiguous here and is not contiguous in the document. A match spanning
// that hole gives a range longer than the words in it, and the
// deleteContentRange built from it marks the skipped content for deletion too.
// withdraw.Span refuses two spans with somebody else's words between them for
// the same reason; this is that rule on this side.
func (p indexed) matches(tabID, quoted string) (out []docs.Range, crossed int) {
	if quoted == "" {
		return nil, 0
	}
	// The end is derived from the last rune of the match, never from the byte
	// behind it. pos of that byte is the start of the next indexed run, so on a
	// quote ending exactly where a footnote mark, a picture or a page break
	// begins it sits a unit or more past the words, and the quote would be
	// refused for crossing a hole it only touches.
	_, size := utf8.DecodeLastRuneInString(quoted)
	tail := utf16Len(quoted[len(quoted)-size:])
	for at := 0; ; {
		i := strings.Index(p.text[at:], quoted)
		if i < 0 {
			return out, crossed
		}
		i += at
		at = i + 1 // one byte, not one match: two matches may overlap.
		if p.anySuggested(i, i+len(quoted)) {
			continue
		}
		start := p.pos[i]
		end := start + utf16Len(quoted)
		if p.pos[i+len(quoted)-size] != end-tail {
			crossed++
			continue
		}
		out = append(out, docs.Range{Tab: tabID, Start: start, End: end})
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
		if b.Table == nil {
			continue
		}
		for _, row := range b.Table.Rows {
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
	}
	p.text = b.String()
	// One index per byte and no more. There used to be an extra entry for the
	// position just past the last byte, back when a match spanned pos[i] to
	// pos[i+len]; that is the scheme matches no longer uses, because pos of the
	// byte behind a match is the start of the next indexed run and measures a
	// quote that merely touches a footnote mark as crossing it. Nothing reads
	// past the last byte now, so nothing is stored there.
	return p
}
