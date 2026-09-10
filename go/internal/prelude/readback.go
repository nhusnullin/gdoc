package prelude

// The prelude read back out of the document.
//
// Phase 1 proposes the house template and phase 2 styles the body, and this is
// the question the first of those leaves behind: is what is in the document a
// proposal, or is it text? Every character the prelude writes goes out in
// writeMode SUGGEST, and CLAUDE.md records that Google has honoured that field
// and once has not. So the claim is read back rather than assumed, exactly as
// `propose` reads its own writes back through a route the write did not go out
// on.
//
// Three things are asked, and none of them can see the other two's failure:
//
//   - Every piece of the prelude carries a suggestion id. A piece that does not
//     is text in somebody's document on gdoc's own authority.
//   - The marker is over the span that was proposed. Without it the next run
//     finds no prelude and proposes a second one in front of the first.
//   - The author's own text is what it was. That is this milestone's own claim,
//     so it is a check rather than a hope.
//
// Nothing here is a verdict. Whether the cover reads right is read in the
// document by the person being asked to accept it, and past that it is Nail's.

import (
	"fmt"
	"strings"

	"gdoc/internal/docs"
)

// settledShown is how many of the runs that carry no suggestion id a warning
// names. A prelude is hundreds of runs, and a warning listing all of them is
// one nobody reads to the end; the count beside it is the whole number.
const settledShown = 3

// Pieces is what the read found, counted the way Result counts what was sent,
// so the two sit beside each other in one report and a reader compares them
// without converting anything.
//
// Runs is the unit the classification is really made in: a paragraph, a table
// or a cell is proposed when every run of it inside the span carries a
// suggestion id, and written when one of them does not.
type Pieces struct {
	Paragraphs int `json:"paragraphs"`
	Tables     int `json:"tables"`
	Cells      int `json:"cells"`
	Runs       int `json:"runs"`
}

// any reports whether the read found anything at all to count.
func (p Pieces) any() bool { return p.Paragraphs+p.Tables+p.Cells+p.Runs > 0 }

// Check is the prelude read back, as facts.
type Check struct {
	// Start and End are the span the run proposed the prelude into, which is
	// the span this read looked in.
	Start int `json:"start"`
	End   int `json:"end"`
	// Proposed is what the read found inside that span carrying a suggested
	// insertion id. It is a count of suggestions and nothing more: it does not
	// say the prelude is complete, and it is not compared with what was sent,
	// because Docs numbers an empty paragraph differently from the request
	// that made it and a mismatch there would name the wrong problem.
	Proposed Pieces `json:"proposed"`
	// Written is the same count over the pieces carrying no insertion id at
	// all. Every one of them is a character in somebody's document on gdoc's
	// own authority, which is the one thing this milestone says it never does,
	// so anything but zero here is the failure rather than a difference.
	Written Pieces `json:"written"`
	// Settled is the words of the first few runs Written counts, so a warning
	// names them rather than only counting them.
	Settled []string `json:"settled,omitempty"`
	// Marked says a named range wearing MarkerName covers exactly the span
	// above. It is gdoc's whole memory of having been here.
	Marked bool `json:"marked"`
	// BodyUnchanged is the author's own text before the run against the same
	// text after it, character for character.
	BodyUnchanged bool `json:"body_unchanged"`
	// Verified is the three together, with the read having found the prelude at
	// all. False is not a failure: what was proposed is in the document either
	// way, and a caller told the run failed is a caller that runs it again.
	Verified bool `json:"verified"`
}

// Verify reads the prelude back out of the document.
//
// before is AuthorText of the document as it was read before a word was
// proposed, after is the document read once both phases had run, and the span
// is the one the run proposed into, which is the range the marker covers.
//
// It never fails. A route that could not answer is a warning naming it, because
// a read-back that refuses to answer tells nobody what is in the document.
//
// The first tab is the one it looks in. Both this command's callers refuse a
// document with more than one tab, twice each, before anything is sent: an
// index means nothing without saying which tab it is in.
func Verify(before string, after *docs.Document, start, end int) (Check, []string) {
	c := Check{Start: start, End: end}
	if after == nil || len(after.Tabs) == 0 {
		return c, []string{
			"the prelude was proposed and no document read reached the read-back, so nothing here says whether it is a suggestion or text"}
	}
	var warnings []string
	tab := after.Tabs[0]
	c.Proposed, c.Written, c.Settled = count(tab, start, end)
	if !c.Proposed.any() && !c.Written.any() {
		warnings = append(warnings, fmt.Sprintf(
			"the prelude was proposed into [%d,%d) and the document read back holds nothing there, so nothing here says it is in the document",
			start, end))
	}
	if c.Written.Runs > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d piece(s) of the prelude read back carrying no suggestion id, so that text is in the document rather than proposed into it: %s",
			c.Written.Runs, strings.Join(quoted(c.Settled), ", ")))
	}
	c.Marked, warnings = marked(after, start, end, warnings)
	c.BodyUnchanged = before == AuthorText(after)
	if !c.BodyUnchanged {
		warnings = append(warnings, fmt.Sprintf(
			"the author's own text held %d character(s) before the run and holds %d now, so this run did not leave it as it was",
			len([]rune(before)), len([]rune(AuthorText(after)))))
	}
	c.Verified = c.Proposed.any() && c.Written == Pieces{} && c.Marked && c.BodyUnchanged
	return c, warnings
}

// marked reports whether a named range wearing MarkerName covers the span the
// run proposed into.
//
// A marker this package cannot read as one span is a warning rather than a
// silence, because Markers refuses one for a reason a person has to hear. More
// than one marker is not itself a fault here: a second run's marker sits beside
// the marker of the prelude it proposed deleting, and that older one goes when
// the deletion is accepted.
func marked(d *docs.Document, start, end int, warnings []string) (bool, []string) {
	found, err := Markers(d)
	if err != nil {
		return false, append(warnings, fmt.Sprintf(
			"the prelude was proposed and its marker could not be read back: %v", err))
	}
	for _, m := range found {
		if m.Start == start && m.End == end {
			return true, warnings
		}
	}
	return false, append(warnings, fmt.Sprintf(
		"the prelude was proposed into [%d,%d) and no named range called %q covers it in the document read back, "+
			"so gdoc has no record of having written this prelude: accept or reject it in the browser before running this again, "+
			"or the next run proposes a second one in front of it",
		start, end, MarkerName))
}

// count walks the tab and sorts what overlaps the span into the two buckets.
//
// A piece is proposed when every run of it inside the span carries a suggested
// insertion id, and written when one of them does not. The walk goes into table
// cells, because the house front matter is three tables and a walk reading
// paragraphs alone would call a wholly proposed table settled.
func count(tab docs.Tab, start, end int) (proposed, written Pieces, settled []string) {
	var walk func(blocks []docs.Block, p, w *Pieces)
	walk = func(blocks []docs.Block, p, w *Pieces) {
		for _, b := range blocks {
			if b.Paragraph != nil {
				seen, clean := 0, true
				for _, r := range b.Paragraph.Runs {
					if r.EndIndex <= start || r.StartIndex >= end {
						continue
					}
					seen++
					if len(r.InsertionIDs) > 0 {
						p.Runs++
						continue
					}
					w.Runs++
					clean = false
					if len(settled) < settledShown {
						settled = append(settled, strings.TrimRight(r.Text, "\n"))
					}
				}
				switch {
				case seen == 0:
				case clean:
					p.Paragraphs++
				default:
					w.Paragraphs++
				}
			}
			if b.Table != nil {
				var tp, tw Pieces
				for _, row := range b.Table.Rows {
					for _, cell := range row {
						var cp, cw Pieces
						walk(cell.Blocks, &cp, &cw)
						add(&tp, cp)
						add(&tw, cw)
						switch {
						case !cp.any() && !cw.any():
						case !cw.any():
							tp.Cells++
						default:
							tw.Cells++
						}
					}
				}
				switch {
				case !tp.any() && !tw.any():
				case !tw.any():
					tp.Tables++
				default:
					tw.Tables++
				}
				add(p, tp)
				add(w, tw)
			}
		}
	}
	walk(tab.Body, &proposed, &written)
	return proposed, written, settled
}

// add folds one count into another.
func add(into *Pieces, from Pieces) {
	into.Paragraphs += from.Paragraphs
	into.Tables += from.Tables
	into.Cells += from.Cells
	into.Runs += from.Runs
}

// AuthorText is the document's own text with gdoc's proposal taken out of it:
// every text run that carries no suggested insertion id, in reading order,
// tables included.
//
// Both sides of the comparison are read by this one rule, which is what makes
// the answer mean something. A suggested insertion is nobody's text yet,
// whoever proposed it, so dropping it on both sides leaves the text the
// document really holds. A suggested deletion is kept, because a deletion in
// SUGGEST mode marks characters rather than removing them and they are still
// the author's until somebody accepts it.
//
// Text runs alone. A chip, a picture and a page break carry no text, and what
// happened to those is the preservation half's question: it counts the chips by
// kind, before against after.
func AuthorText(d *docs.Document) string {
	if d == nil {
		return ""
	}
	var b strings.Builder
	var walk func(blocks []docs.Block)
	walk = func(blocks []docs.Block) {
		for _, block := range blocks {
			if block.Paragraph != nil {
				for _, r := range block.Paragraph.Runs {
					if r.Kind != docs.KindText || len(r.InsertionIDs) > 0 {
						continue
					}
					b.WriteString(r.Text)
				}
			}
			if block.Table != nil {
				for _, row := range block.Table.Rows {
					for _, cell := range row {
						walk(cell.Blocks)
					}
				}
			}
		}
	}
	for _, tab := range d.Tabs {
		walk(tab.Body)
	}
	return b.String()
}

// quoted puts each of these in quotes, for a warning that names words rather
// than running them into the sentence around them.
func quoted(list []string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, fmt.Sprintf("%q", s))
	}
	return out
}
