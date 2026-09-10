package restyle

// The read-back: what the document held before the restyle, what it holds now,
// and whether the style gdoc sent is there.
//
// Two halves, and they answer two different questions. The preservation half
// asks what survived: the threads and their witness, the pending suggestions by
// id, and the chips. The landing half, in landing.go, asks whether the style is
// actually in the document. A run that preserved everything and landed nothing
// is a run that did no work, and a run that landed everything and lost the
// comment anchors is the failure this milestone exists to avoid. Neither half
// can see the other's failure, which is why there are two.
//
// The before side is M7's survey, read out of the file the caller passed to
// --from. That is why the survey carries a witness per thread rather than
// totals alone: a thread that was already detached before the run reads as
// damage the run did when only the counts are kept.
//
// Nothing here explains a change. A count that moved is reported, a thread that
// lost its anchor is named, and whether that matters is read in the document,
// and past that it is Nail's.

import (
	"fmt"
	"sort"

	"gdoc/internal/docx"
)

// ReadBack is what the run found when it read the document back.
type ReadBack struct {
	// After is the same survey the --dry-run half prints, taken again.
	After Report `json:"after"`
	// Preservation is the before against the after.
	Preservation Preservation `json:"preservation"`
	// Landing is whether the style reached the document.
	Landing Landing `json:"landing"`
	// Manual names what gdoc could not do at all, with the menu path for each.
	Manual []ManualStep `json:"manual"`
	// Verified is every check together: every landing check held, nothing was
	// lost, and the witness still answered for the threads it answered for
	// before. Anything less is reported with the route named, and is not a
	// failure: the batches are in the document either way.
	Verified bool `json:"verified"`
}

// Preservation is the before against the after, as facts. Every list is what
// moved, and none of them says whether the restyle was worth making.
type Preservation struct {
	Threads     ThreadChange     `json:"threads"`
	Suggestions SuggestionChange `json:"suggestions"`
	Chips       ChipChange       `json:"chips"`
	// Intact is the four losses being empty: no thread gone, none that lost its
	// anchor, no pending suggestion gone, and no fewer chips than before. It is
	// those facts and nothing else. A thread that arrived during the run and a
	// witness that stopped answering are both reported and neither is a loss.
	Intact bool `json:"intact"`
}

// ThreadChange is the threads before and after, by id.
type ThreadChange struct {
	Before int `json:"before"`
	After  int `json:"after"`
	// Gone is a thread the survey listed that the read back does not.
	Gone []string `json:"gone"`
	// Arrived is a thread that was not there before. Somebody commenting while
	// the run was writing is the ordinary reason, and it is a fact rather than
	// a fault.
	Arrived []string `json:"arrived"`
	// WitnessChanged is every thread whose witness reads differently now,
	// whichever way it moved.
	WitnessChanged []WitnessChange `json:"witness_changed"`
	// LostAnchor is the threads that were anchored and are now detached. That
	// is the one witness move that is damage: the docx export says the text the
	// comment was attached to has gone.
	LostAnchor []string `json:"lost_anchor"`
	// Unwitnessed is the threads the export answered for before and cannot
	// answer for now. It is not damage and it is not health: it is the witness
	// losing its answer, which is why verified is false over it.
	Unwitnessed []string `json:"unwitnessed"`
	// NowDetached is the threads the export reads detached now and gave no
	// answer for before. Detached is positive evidence the text a comment was
	// attached to has gone, so this is not the absence of evidence Unwitnessed
	// holds; what is missing is any answer about whether it was already detached
	// before the run, so it is not LostAnchor either. Verified is false over it.
	NowDetached []string `json:"now_detached"`
}

// WitnessChange is one thread's witness, before and after.
type WitnessChange struct {
	ID     string `json:"id"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// SuggestionChange is what was pending, by id. Ids rather than counts, because
// one suggestion destroyed and another created is two counts that do not move.
type SuggestionChange struct {
	Before  int      `json:"before"`
	After   int      `json:"after"`
	Gone    []string `json:"gone"`
	Arrived []string `json:"arrived"`
}

// ChipChange is the chips by kind, before and after. A chip is a live
// reference, so a kind with fewer of them than before is a reference that
// stopped being one.
type ChipChange struct {
	Before ChipCounts `json:"before"`
	After  ChipCounts `json:"after"`
	Fewer  bool       `json:"fewer"`
}

// ManualStep is one thing gdoc could not do, and where a person does it.
type ManualStep struct {
	What  string `json:"what"`
	Where string `json:"where"`
}

// The three the Docs API cannot create at all, measured on 2026-09-09. A
// first-page header is refused outright, and the other two are Word fields the
// docx writer emits and no request kind builds. SPEC has gdoc write this list
// into the document as a finishing checklist; Nail's decision of 2026-09-09 is
// that it reports them instead and the skill reads them out, which is the M2
// line held: the binary prints facts and the skill judges.
var apiCannot = []ManualStep{
	{
		What:  "the first-page header carrying the Altery logo",
		Where: "Insert > Headers & footers > Header, then Options > Header format > Different first page, then Insert > Image > Upload from computer",
	},
	{
		What:  "the contents list",
		Where: "Insert > Table of contents",
	},
	{
		What:  "the page numbers in the footer",
		Where: "Insert > Page numbers",
	},
}

// ManualSteps is the list of things gdoc could not do, the three the API cannot
// create and the two this milestone chose not to.
//
// The two are the lists and the tables. createParagraphBullets removes the
// leading tabs that set a bullet's nesting level, so it deletes text the author
// typed and is not one of the four kinds the in-place level carries. A table's
// column widths and row heights need two request kinds nobody measured, and
// both change a table's layout rather than its look.
//
// One gap is not on this list, and it is written down here rather than left to
// be discovered. TabRequests walks the tab's body alone, so a paragraph inside a
// footnote, a header or a footer keeps the look it had, and nothing below names
// it. An unconditional entry would fire on every document, including the ones
// holding none of the three, which is the warning people learn to ignore; a
// conditional one needs a count this function is not given, and for a header or
// a footer needs a decoder internal/docs does not have. Which of the three to
// report, and on what condition, is Nail's, in the shape the 2026-09-09 decision
// settled this list in.
// docs/backlog/restyle-skips-footnotes-headers-and-footers.md holds it.
func ManualSteps(p Plan) []ManualStep {
	out := append([]ManualStep{}, apiCannot...)
	if p.Bulleted > 0 {
		out = append(out, ManualStep{
			What: fmt.Sprintf("%d bulleted or numbered paragraph(s), left with the list they had", p.Bulleted),
			Where: "Format > Bullets & numbering > List options, " +
				"because the request that sets a list also removes the leading tabs that set its nesting level",
		})
	}
	if p.Tables > 0 {
		out = append(out, ManualStep{
			What:  fmt.Sprintf("the column widths and row heights of %d table(s)", p.Tables),
			Where: "click in the table, then Format > Table > Table properties",
		})
	}
	if len(p.Unstyled) > 0 {
		out = append(out, ManualStep{
			What:  fmt.Sprintf("the paragraphs whose named style the house has no look for: %v", p.Unstyled),
			Where: "Format > Paragraph styles, because gdoc maps no style to the nearest one it knows",
		})
	}
	return out
}

// Sent is what left the machine, and it is two lists because they answer
// differently.
//
// Confirmed is the requests inside the batches Docs answered for. Unconfirmed is
// the requests of the one batch Docs accepted whose answer could not be read:
// they may be in the document, so the landing half is asked about them too,
// which on that path is the only way anybody finds out. What they cannot do is
// make a run verified, because a check that holds there does not say the run
// knows what it wrote.
type Sent struct {
	Confirmed   []map[string]any
	Unconfirmed []map[string]any
}

// All is every request that reached Docs, in the order it was sent. It builds a
// new slice rather than appending to Confirmed, which is the caller's.
func (s Sent) All() []map[string]any {
	if len(s.Unconfirmed) == 0 {
		return s.Confirmed
	}
	out := make([]map[string]any, 0, len(s.Confirmed)+len(s.Unconfirmed))
	out = append(out, s.Confirmed...)
	return append(out, s.Unconfirmed...)
}

// Verify is the read-back, both halves. before is the survey the run was made
// against, after is the three reads taken once the batches landed, and raw is
// the bytes of that Docs read, which the landing half reads the styling out of.
//
// It never fails. A read the caller could not make is a warning, exactly as the
// survey handles one, because a read-back that refuses to answer tells nobody
// what is in the document.
func Verify(before Report, after Input, raw []byte, sent Sent, plan Plan) (ReadBack, []string) {
	rep, warnings := Survey(after)
	out := ReadBack{After: rep, Manual: ManualSteps(plan)}
	out.Preservation = Preserve(before, rep)
	landing, notes := Landed(raw, sent.All())
	out.Landing = landing
	warnings = append(warnings, notes...)
	warnings = append(warnings, out.Preservation.warnings()...)

	held := len(landing.Checks) > 0
	for _, c := range landing.Checks {
		if c.Held {
			continue
		}
		held = false
		warnings = append(warnings, landingWarning(c))
	}
	// A batch Docs accepted whose answer could not be read is never verified,
	// whatever the checks say. The landing half reads the first request of each
	// kind, so a batch of hundreds can hold one request that landed and the rest
	// that did not, and the run has no answer for the ones it did not read. This
	// is the same rule propose and publish hold: what cannot be confirmed is
	// reported, never claimed.
	out.Verified = held && len(sent.Unconfirmed) == 0 &&
		out.Preservation.Intact && len(out.Preservation.Threads.Unwitnessed) == 0 &&
		len(out.Preservation.Threads.NowDetached) == 0
	return out, warnings
}

// landingWarning names the route that did not hold, the way propose names one.
func landingWarning(c Check) string {
	if c.Note != "" {
		return fmt.Sprintf("the %s the run sent could not be read back at %s: %s", c.Kind, c.Where, c.Note)
	}
	return fmt.Sprintf("the %s the run sent is not in %s as it was sent: %v", c.Kind, c.Where, c.Missing)
}

// Preserve is the before survey against the after one.
func Preserve(before, after Report) Preservation {
	p := Preservation{
		Threads:     threadChange(before.Threads, after.Threads),
		Suggestions: suggestionChange(before.Suggestions, after.Suggestions),
		Chips:       chipChange(before.Chips, after.Chips),
	}
	p.Intact = len(p.Threads.Gone) == 0 && len(p.Threads.LostAnchor) == 0 &&
		len(p.Suggestions.Gone) == 0 && !p.Chips.Fewer
	return p
}

// threadChange is the threads by id, and the witness beside each.
//
// A witness that reads unmatched now is never counted as damage. Unmatched is
// the export giving no answer, which happens when it could not be read at all
// and when two exported comments share words and disagree, and calling absence
// of evidence a destroyed anchor is the cry-wolf warning this tool avoids
// everywhere else. It is reported as Unwitnessed instead, and verified is false
// over it because nothing then says the anchor survived.
func threadChange(before, after ThreadCounts) ThreadChange {
	was := witnessByID(before.Witnessed)
	is := witnessByID(after.Witnessed)
	c := ThreadChange{
		Before:         before.Open + before.Resolved,
		After:          after.Open + after.Resolved,
		Gone:           []string{},
		Arrived:        []string{},
		WitnessChanged: []WitnessChange{},
		LostAnchor:     []string{},
		Unwitnessed:    []string{},
		NowDetached:    []string{},
	}
	for _, id := range sortedKeys(was) {
		now, there := is[id]
		if !there {
			c.Gone = append(c.Gone, id)
			continue
		}
		then := was[id]
		if then == now {
			continue
		}
		c.WitnessChanged = append(c.WitnessChanged, WitnessChange{ID: id, Before: then, After: now})
		switch {
		case then == docx.WitnessAnchored && now == docx.WitnessDetached:
			c.LostAnchor = append(c.LostAnchor, id)
		case now == docx.WitnessUnmatched:
			c.Unwitnessed = append(c.Unwitnessed, id)
		case now == docx.WitnessDetached:
			// then is unmatched, because anchored is the case above and
			// detached would have read the same as now. The anchor is gone and
			// nothing says the run took it, so it is neither of the two lists
			// above and it is never a verified run.
			c.NowDetached = append(c.NowDetached, id)
		}
	}
	for _, id := range sortedKeys(is) {
		if _, there := was[id]; !there {
			c.Arrived = append(c.Arrived, id)
		}
	}
	return c
}

func witnessByID(list []ThreadWitness) map[string]string {
	out := make(map[string]string, len(list))
	for _, t := range list {
		out[t.ID] = t.Witness
	}
	return out
}

// suggestionChange is the pending ids before against after. A pending
// suggestion that is gone is one somebody accepted or rejected, or one a write
// destroyed, and this cannot tell those apart: it reports the id and says
// nothing about which.
func suggestionChange(before, after SuggestionCounts) SuggestionChange {
	was := set(before.IDs)
	is := set(after.IDs)
	c := SuggestionChange{
		Before:  before.Pending + before.OnElements,
		After:   after.Pending + after.OnElements,
		Gone:    []string{},
		Arrived: []string{},
	}
	for _, id := range sortedKeys(was) {
		if _, there := is[id]; !there {
			c.Gone = append(c.Gone, id)
		}
	}
	for _, id := range sortedKeys(is) {
		if _, there := was[id]; !there {
			c.Arrived = append(c.Arrived, id)
		}
	}
	return c
}

func chipChange(before, after ChipCounts) ChipChange {
	return ChipChange{
		Before: before,
		After:  after,
		Fewer: after.Person < before.Person || after.Date < before.Date ||
			after.RichLink < before.RichLink,
	}
}

// warnings names every loss, one sentence each, and never a reason for it.
func (p Preservation) warnings() []string {
	var out []string
	for _, id := range p.Threads.Gone {
		out = append(out, fmt.Sprintf(
			"comment thread %s was in the survey and is not in the document read back", id))
	}
	for _, id := range p.Threads.LostAnchor {
		out = append(out, fmt.Sprintf(
			"comment thread %s was anchored before the run and the export now reads it detached, so the text it was attached to has gone", id))
	}
	for _, id := range p.Threads.Unwitnessed {
		out = append(out, fmt.Sprintf(
			"comment thread %s had a witness before the run and has none now, so nothing here says whether its anchor survived", id))
	}
	for _, id := range p.Threads.NowDetached {
		out = append(out, fmt.Sprintf(
			"comment thread %s reads detached now and the survey's export gave no answer for it, so the text it was attached to has gone and nothing here says whether this run took it", id))
	}
	for _, id := range p.Suggestions.Gone {
		out = append(out, fmt.Sprintf(
			"suggestion %s was pending before the run and is not pending now", id))
	}
	if p.Chips.Fewer {
		out = append(out, fmt.Sprintf(
			"the document held %d person, %d date and %d link chip(s) before the run and holds %d, %d and %d now",
			p.Chips.Before.Person, p.Chips.Before.Date, p.Chips.Before.RichLink,
			p.Chips.After.Person, p.Chips.After.Date, p.Chips.After.RichLink))
	}
	return out
}

func set(ids []string) map[string]string {
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		out[id] = id
	}
	return out
}

// sortedKeys is a map's keys in one order, because a map is walked in none and
// one read-back must name the same threads in the same order twice.
func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
