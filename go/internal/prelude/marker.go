package prelude

import (
	"fmt"
	"sort"
	"strings"

	"gdoc/internal/cover"
	"gdoc/internal/docs"
	"gdoc/internal/house"
)

// MarkerName is the name of the named range gdoc puts over its own prelude,
// and it is the whole of gdoc's memory of having been here. A restyle writes
// no file beside the document, and there is no note to pair: the document
// itself is the record.
//
// The name is a constant and the marker is read by id. docs.NamedRange is keyed
// by id because two ranges may wear one name, and Docs answers with both under
// that one key. A run acting by name would act on both of them, so two ranges
// wearing this name are refused rather than guessed between.
const MarkerName = "gdoc:house-prelude"

// Marker is one named range wearing MarkerName, as this run found it.
//
// Pending is what a run cannot replace. The prelude is proposed, so between the
// run that proposed it and Nail accepting it the marked text exists only as a
// suggestion, and a second run over it would propose deleting text that has
// never been written.
type Marker struct {
	ID         string   `json:"id"`
	Tab        string   `json:"tab"`
	Start      int      `json:"start"`
	End        int      `json:"end"`
	PendingIDs []string `json:"pending_ids,omitempty"`
}

// Pending reports whether the text this marker covers is still a proposal.
func (m Marker) Pending() bool { return len(m.PendingIDs) > 0 }

// Decision is what one run does about the prelude a run before it may have
// left. Replaces is nil on a document carrying no marker, which is either a
// document gdoc has never touched or one whose prelude was rejected: a rejected
// insertion takes its named range with it, measured 2026-09-10, so both are
// proposed into cleanly and neither needs cleaning up.
type Decision struct {
	Replaces *Marker `json:"replaces,omitempty"`
	Start    int     `json:"start"`
}

// Decide reads the document and says what this run does.
//
// Three shapes, and the third is the refusal. No marker is a first run, which
// proposes at index 1. One marker over settled text is a second run, which
// proposes deleting that text and proposes a fresh prelude in its place. One
// marker over text that is still pending is neither, and the answer is the
// prelude already in front of Nail: he accepts it or rejects it, and the run
// after that has one of the first two shapes.
//
// Nothing here deletes a named range, and nothing needs to. The marker tracks
// the text it covers: measured on 2026-09-10, a rejected insertion took its
// marker with it, and an accepted one kept its id and its range over the text
// the accept made real. So a replaced prelude's marker goes when the deletion
// it is proposed under is accepted, and comes back when that deletion is
// rejected. Either way one marker is left, which is the shape this function
// requires. That last step is the inference the measurement makes, not a fifth
// measured row, and Task 10's live run is what confirms it.
func Decide(d *docs.Document) (Decision, error) {
	found, err := Markers(d)
	if err != nil {
		return Decision{}, err
	}
	switch len(found) {
	case 0:
		return Decision{Start: 1}, nil
	case 1:
	default:
		return Decision{}, fmt.Errorf(
			"prelude: this document carries %d named ranges called %q and gdoc makes one, %s. gdoc reads its marker by id and will not guess which of them is its own prelude, so remove the ones that are not by hand and run this again",
			len(found), MarkerName, strings.Join(ids(found), ", "))
	}
	m := found[0]
	if m.Pending() {
		return Decision{}, fmt.Errorf(
			"prelude: the house prelude marked by %s is still pending, as %s. Accept or reject it in the browser, then run this again: a run that replaced it would propose deleting text that has not been written",
			m.ID, strings.Join(m.PendingIDs, ", "))
	}
	return Decision{Replaces: &m, Start: m.Start}, nil
}

// Markers is every named range wearing MarkerName, in the order the document
// reports them, each with the suggestion ids the text it covers is carrying.
//
// A marker this function cannot read as one span of one tab's body is refused
// rather than reported, because what a caller does with it is delete the text
// under it. A span in a header, a footer or a footnote, and two spans with the
// author's own words between them, are both a marker gdoc did not make, and
// deleting the whole of either would propose deleting words gdoc never wrote.
// Two spans that touch are one span: Docs may cut a range at a boundary of its
// own, and the text is still contiguous.
func Markers(d *docs.Document) ([]Marker, error) {
	if d == nil {
		return nil, nil
	}
	var out []Marker
	for _, tab := range d.Tabs {
		for _, nr := range tab.NamedRanges {
			if nr.Name != MarkerName {
				continue
			}
			m, err := marker(nr)
			if err != nil {
				return nil, err
			}
			m.PendingIDs = suggestedIn(tab, m.Start, m.End)
			out = append(out, m)
		}
	}
	return out, nil
}

// marker is one named range as one span, or the reason it is not one.
func marker(nr docs.NamedRange) (Marker, error) {
	if len(nr.Ranges) == 0 {
		return Marker{}, fmt.Errorf("prelude: the marker %s covers nothing at all, so there is no prelude under it to replace", nr.ID)
	}
	spans := append([]docs.Range(nil), nr.Ranges...)
	sort.Slice(spans, func(i, j int) bool { return spans[i].Start < spans[j].Start })
	first := spans[0]
	for _, s := range spans {
		if s.Segment != "" {
			return Marker{}, fmt.Errorf("prelude: the marker %s sits in %s rather than in the body, and gdoc puts its prelude in the body", nr.ID, s.Segment)
		}
		if s.Tab != first.Tab {
			return Marker{}, fmt.Errorf("prelude: the marker %s covers text in two tabs, %s and %s, and gdoc's prelude is one run of one tab", nr.ID, first.Tab, s.Tab)
		}
		if s.End <= s.Start {
			return Marker{}, fmt.Errorf("prelude: the marker %s covers [%d,%d), which does not end after it starts", nr.ID, s.Start, s.End)
		}
	}
	end := first.End
	for _, s := range spans[1:] {
		if s.Start > end {
			return Marker{}, fmt.Errorf(
				"prelude: the marker %s is broken into spans with the document's own text between them, [%d,%d) and [%d,%d). Deleting the whole of it would propose deleting words gdoc never wrote, so this one is left where it is",
				nr.ID, first.Start, end, s.Start, s.End)
		}
		if s.End > end {
			end = s.End
		}
	}
	return Marker{ID: nr.ID, Tab: first.Tab, Start: first.Start, End: end}, nil
}

// suggestedIn is every suggestion id carried by a run that overlaps this span,
// sorted and without repeats.
//
// Both lists are read. An insertion id says the prelude has not been accepted
// yet, and a deletion id says a run before this one already proposed replacing
// it. Neither is text this run may propose deleting, and the answer to both is
// the same: the suggestion in front of Nail is the one that settles it.
//
// The walk goes into tables, because the front matter is three of them. A walk
// that read paragraphs alone would call a wholly proposed prelude settled.
func suggestedIn(tab docs.Tab, start, end int) []string {
	seen := map[string]bool{}
	var walk func(blocks []docs.Block)
	walk = func(blocks []docs.Block) {
		for _, b := range blocks {
			if b.Paragraph != nil {
				for _, r := range b.Paragraph.Runs {
					if r.EndIndex <= start || r.StartIndex >= end {
						continue
					}
					for _, id := range r.InsertionIDs {
						seen[id] = true
					}
					for _, id := range r.DeletionIDs {
						seen[id] = true
					}
				}
			}
			if b.Table != nil {
				for _, row := range b.Table.Rows {
					for _, cell := range row {
						walk(cell.Blocks)
					}
				}
			}
		}
	}
	walk(tab.Body)
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// ids is the ids of these markers, for a refusal that names them.
func ids(found []Marker) []string {
	out := make([]string, 0, len(found))
	for _, m := range found {
		out = append(out, m.ID)
	}
	return out
}

// Propose is the whole of what one run sends in SUGGEST mode: the front matter,
// and on a second run the suggested deletion of the prelude the run before it
// left, in front of it.
//
// The deletion comes first and the insert lands at its start, which is the
// order internal/propose sends a replacement in. A deleteContentRange in
// SUGGEST mode marks text rather than removing it, so nothing behind it moves
// and every index the front matter computed from Start is still the index it
// named.
func Propose(cfg *house.Config, f cover.Fields, d *docs.Document) (Result, error) {
	dec, err := Decide(d)
	if err != nil {
		return Result{}, err
	}
	res, err := FrontMatter(cfg, f, dec.Start)
	if err != nil {
		return Result{}, err
	}
	if dec.Replaces == nil {
		return res, nil
	}
	del := map[string]any{"deleteContentRange": map[string]any{
		"range": span(dec.Replaces.Start, dec.Replaces.End),
	}}
	res.Requests = append([]map[string]any{del}, res.Requests...)
	res.Replaces = dec.Replaces
	return res, nil
}

// MarkerRequest is the createNamedRange that marks what this run proposed.
//
// It is the one thing a restyle writes directly, because Docs answered
// "createNamedRange: Request does not support application as suggestion" on
// 2026-09-10. It is safe on its own terms: a named range adds and removes no
// character. The guard holds it to exactly this shape, granted for exactly this
// range, in Policy.AllowMarker.
func MarkerRequest(start, end int) map[string]any {
	return map[string]any{"createNamedRange": map[string]any{
		"name":  MarkerName,
		"range": span(start, end),
	}}
}
