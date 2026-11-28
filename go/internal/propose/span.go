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
		return docs.Range{}, MoreThanOneTab("the document has %d tabs, and a proposal is placed in a document with one", len(d.Tabs))
	}
	tab := d.Tabs[0]

	found, crossed := matchesIn(paragraphs(tab.Body), tab.ID, quoted)
	return pick(quoted, found, crossed, "")
}

// matchesIn is matches over a run of paragraphs, summed.
func matchesIn(ps []indexed, tabID, quoted string) (found []docs.Range, crossed int) {
	for _, p := range ps {
		hits, over := p.matches(tabID, quoted)
		found = append(found, hits...)
		crossed += over
	}
	return found, crossed
}

// pick is the exactly-once rule over what a search found. where is empty for
// the whole document, and otherwise the words that named the place searched,
// so a refusal says where the count was taken.
func pick(quoted string, found []docs.Range, crossed int, where string) (docs.Range, error) {
	in, more := "in the document as written", "; quote more of the sentence, or add near, words that stand once beside the occurrence that is meant"
	if where != "" {
		in = fmt.Sprintf("beside the near words %q, in their paragraph or their table row", where)
		more = "; name other near words, or quote more of the sentence"
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
			"the quoted text %q occurs %d times as written text and %d more across content this read does not index, a footnote mark, a picture or a page break among them, so which occurrence was meant is not clear%s",
			quoted, len(found), crossed, more)
	case len(found) == 1:
		return found[0], nil
	case len(found) == 0:
		// Text inside a pending suggestion is not text a proposal may stand on,
		// and it lands here rather than in its own message: from the caller's
		// side both mean the same thing, which is that these words are not
		// somewhere a change can be placed.
		return docs.Range{}, fmt.Errorf("the quoted text %q was not found %s; it may have changed, or it may be inside a pending suggestion", quoted, in)
	default:
		if where != "" {
			return docs.Range{}, fmt.Errorf("the quoted text %q occurs %d times %s%s", quoted, len(found), in, more)
		}
		return docs.Range{}, fmt.Errorf("the quoted text %q occurs %d times%s", quoted, len(found), more)
	}
}

// FindSpanNear is FindSpan for a quote that stands in the document more than
// once: near names the place, and the quote is counted there.
//
// A table cell is its whole paragraph, so a caller told to quote more of the
// sentence has no more to quote, and the same holds for a short heading. near
// is words that stand in the document exactly once, by the same walk and the
// same rule as any quote. The quote is then looked for in the paragraph that
// holds them, and, where that paragraph does not hold it as written text and
// sits in a table, in the paragraphs of its table row. Exactly once there, or
// it is refused.
//
// It is still text, read fresh: no entry carries an occurrence number or an
// index. An empty near is FindSpan.
func FindSpanNear(d *docs.Document, quoted, near string) (docs.Range, error) {
	r, _, err := FindPlaced(d, quoted, near)
	return r, err
}

// FindPlaced is FindSpanNear, and says which place the quote was counted in:
// row is true where it was near's table row rather than near's own
// paragraph. A read-back that asks the place again is handed that answer.
func FindPlaced(d *docs.Document, quoted, near string) (r docs.Range, row bool, err error) {
	if near == "" {
		r, err = FindSpan(d, quoted)
		return r, false, err
	}
	if quoted == "" {
		return docs.Range{}, false, fmt.Errorf("the proposal quotes no text, so there is nothing to find")
	}
	if len(d.Tabs) != 1 {
		return docs.Range{}, false, MoreThanOneTab("the document has %d tabs, and a proposal is placed in a document with one", len(d.Tabs))
	}
	tab := d.Tabs[0]
	own, around, err := placeNear(tab.Body, tab.ID, near)
	if err != nil {
		return docs.Range{}, false, err
	}
	// The paragraph is asked first, and by the walk a quote is found with, so
	// a copy inside somebody's pending suggestion does not keep the search in
	// a paragraph where nothing can be placed. A crossing occurrence counts:
	// it is an occurrence, and it is refused there rather than widened past.
	scope := []indexed{own}
	if found, crossed := own.matches(tab.ID, quoted); len(found)+crossed == 0 && around != nil {
		scope, row = around, true
	}
	found, crossed := matchesIn(scope, tab.ID, quoted)
	r, err = pick(quoted, found, crossed, near)
	return r, row, err
}

// CarriesNear is Carries asked of the place near names, which is the only
// question that tells a suggestion from a direct edit when the quote stands
// elsewhere too. row is the place FindPlaced chose, its table row or its own
// paragraph, and it is not worked out again here: after a direct edit the
// paragraph no longer holds the quote, and widening to the row then would
// find another cell's copy and pass.
//
// placed is false where the document does not hold near exactly once, so
// there is no such place and the question has no answer: a copy of near
// inside somebody's pending deletion shows in the preview and makes two. An
// empty near asks the whole tab, and is always placed.
func CarriesNear(d *docs.Document, tabID, want, near string, row bool) (held, placed bool) {
	if near == "" {
		return Carries(d, tabID, want), true
	}
	for _, t := range d.Tabs {
		if t.ID != tabID {
			continue
		}
		own, around, err := placeNear(t.Body, t.ID, near)
		if err != nil {
			return false, false
		}
		scope := []indexed{own}
		if row {
			if around == nil {
				return false, false
			}
			scope = around
		}
		for _, p := range scope {
			if strings.Contains(p.text, want) {
				return true, true
			}
		}
		return false, true
	}
	return false, false
}

// placeNear is the place near names: the one paragraph that holds it, and
// the paragraphs of that paragraph's table row, nil where it stands in no
// table.
func placeNear(body []docs.Block, tabID, near string) (own indexed, around []indexed, err error) {
	rows := rowsOf(body, nil)
	at, in, n := -1, -1, 0
	for i, r := range rows {
		for j, p := range r.paragraphs {
			hits, crossed := p.matches(tabID, near)
			if k := len(hits) + crossed; k > 0 {
				n += k
				in, at = i, j
			}
		}
	}
	switch {
	case n == 0:
		return indexed{}, nil, fmt.Errorf("the near words %q were not found in the document as written, so they name no place; near is words that stand once beside the quote, in its paragraph or its table row", near)
	case n > 1:
		return indexed{}, nil, fmt.Errorf("the near words %q occur %d times, so they name no one place; near is words that stand in the document exactly once", near, n)
	}
	if rows[in].table {
		around = rows[in].paragraphs
	}
	return rows[in].paragraphs[at], around, nil
}

// row is one place a quote can be counted in: the paragraphs of one table
// row, or one paragraph that stands in no table.
type row struct {
	paragraphs []indexed
	table      bool
}

// rowsOf walks a body into rows, tables walked into. A table inside a cell
// gives rows of its own, and its paragraphs belong to those rather than to
// the row around them.
func rowsOf(bs []docs.Block, out []row) []row {
	for _, b := range bs {
		if b.Paragraph != nil {
			out = append(out, row{paragraphs: []indexed{index(b.Paragraph.Runs)}})
			continue
		}
		if b.Table == nil {
			continue
		}
		for _, cells := range b.Table.Rows {
			r := row{table: true}
			var inner []row
			for _, c := range cells {
				for _, cb := range c.Blocks {
					if cb.Paragraph != nil {
						r.paragraphs = append(r.paragraphs, index(cb.Paragraph.Runs))
					}
				}
				inner = rowsOf(tablesOnly(c.Blocks), inner)
			}
			out = append(append(out, r), inner...)
		}
	}
	return out
}

// tablesOnly is the tables among a cell's blocks.
func tablesOnly(bs []docs.Block) []docs.Block {
	var out []docs.Block
	for _, b := range bs {
		if b.Table != nil {
			out = append(out, b)
		}
	}
	return out
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
// prelude's Decide refuses a marker broken into two spans with the document's
// own words between them for the same reason; this is that rule on this side.
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
