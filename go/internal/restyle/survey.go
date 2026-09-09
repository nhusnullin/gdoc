// Package restyle is the survey of a document, and the requests that restyle
// one. The survey reports what a document holds before anything is done to it:
// its threads with a witness for each, what is pending, its chips, its tabs,
// its named ranges and the revision the reads were made against. The builders
// beside it, page.go first, turn the house style into the styling requests a
// batch carries.
//
// It takes no session and touches no wire, like every other reader package
// here. The caller makes the three reads and hands the decoded answers over, so
// every case in this file is testable on a value a test wrote out. A builder is
// the same rule from the other side: it is a pure function of the house style
// and the document, returning the request a caller sends, so nothing here
// decides when to send one.
//
// Nothing here judges anything. NothingToProtect is a fact about five counts
// being zero, never a recommendation to restyle: the threads, what is pending,
// the chips, the paragraph elements the read could not name, and the named
// ranges. Whether a
// document is worth restyling is Nail's, reading the counts. The witness has the two limits it
// has everywhere else, and they are worth stating rather than discovering: it
// names a destroyed anchor and not a moved one, and two exported comments that
// share words and disagree give no answer for either.
package restyle

import (
	"fmt"
	"sort"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/suggestions"
)

// Input is the three reads, already decoded: the comment listing, the Docs read
// and the docx export. The named ranges come out of the Docs read, which carries
// them already, so there is no fourth. Export is the docx the witness is read
// from, and ExportErr is why there is none: an export that failed is a
// warning on a survey that still carries its threads, exactly as
// `comments --witness` behaves, rather than a failed survey.
type Input struct {
	Document  *docs.Document
	Comments  []comments.RawComment
	Export    *docx.File
	ExportErr error
}

// Schema is the shape of a Report, and it is the version `restyle --from`
// insists on before it opens a direct-edit grant on the strength of one.
//
// It is here for the reason internal/frontmatter's schema is there. The apply
// half reads this file as the record of what the document held before the run,
// and a field added later is a field an older survey simply does not carry: the
// suggestion ids arrived at M7b, and an M7 survey read without them reported
// every pending suggestion as still pending, because nothing was there to
// compare. A strict decoder refuses a key it does not know and says nothing at
// all about a key that is absent, so the version is what closes that half.
// Bumping it is a decision, not a refactor.
const Schema = 1

// Report is the survey. Every field is a count, an id or a list.
type Report struct {
	// Schema is Schema above, on every report this binary prints.
	Schema           int               `json:"schema"`
	DocumentID       string            `json:"document_id"`
	RevisionID       string            `json:"revision_id"`
	Title            string            `json:"title"`
	Tabs             int               `json:"tabs"`
	Threads          ThreadCounts      `json:"threads"`
	Suggestions      SuggestionCounts  `json:"suggestions"`
	Chips            ChipCounts        `json:"chips"`
	NamedRanges      []docs.NamedRange `json:"named_ranges"`
	NothingToProtect bool              `json:"nothing_to_protect"`
}

// ThreadCounts is the threads by state, with the witness both ways: as counts,
// and one line per thread. The per-thread list is why the witness is read here
// at all. A later run comparing before with after needs to know which thread
// was already detached, because a thread detached before the work reads as
// damage the work did when only the totals are kept.
type ThreadCounts struct {
	Open      int             `json:"open"`
	Resolved  int             `json:"resolved"`
	Witness   WitnessCounts   `json:"witness"`
	Witnessed []ThreadWitness `json:"witnessed"`
}

// WitnessCounts is the three answers internal/docx gives, counted.
type WitnessCounts struct {
	Anchored  int `json:"anchored"`
	Detached  int `json:"detached"`
	Unmatched int `json:"unmatched"`
}

// ThreadWitness is one thread's witness, keyed by the Drive comment id.
type ThreadWitness struct {
	ID       string `json:"id"`
	Witness  string `json:"witness"`
	Resolved bool   `json:"resolved"`
}

// SuggestionCounts is what is pending. Pending is suggestions.All's and not
// List's: List drops a suggestion whose text is only whitespace, and a
// whitespace-only suggestion is still something a replacement would destroy, so
// counting from List reports nothing to protect on a document that has
// something to protect.
//
// OnElements is the other half, and it is here because the pending walk cannot
// see it. suggestions.All reads text runs alone, on purpose: a footnote's
// number reported as suggested text would be a lie, and a suggested picture has
// no text to report at all. So a suggestion carried only by a run that walk
// skips is in no listing this binary prints, and counting nothing there would
// answer nothing to protect on a document with a pending change in it.
//
// The rule is what the listing can report, not whether the run holds text. A
// footnote reference carries its number and is counted here all the same,
// beside the chips, the breaks, the rule, the auto text and the picture, and
// for the same reason: the pending walk skips it, so nothing else speaks for
// it. It counts ids rather than the insert-and-delete pairs Pending counts, so
// one replacement is 2 there and 1 here, and an id the pending walk already saw
// on a text run is not counted twice.
type SuggestionCounts struct {
	Pending    int `json:"pending"`
	OnElements int `json:"on_elements"`
	// IDs is every pending suggestion id the read could see, both the ones the
	// listing reports and the ones only an element carries, sorted and once
	// each. It is here for M7b's read-back, which compares what was pending
	// before a restyle with what is pending after: two counts that do not move
	// cannot tell one suggestion destroyed and another created from nothing
	// having happened at all, and an id can.
	IDs []string `json:"ids"`
}

// ChipCounts is the smart chips by kind. A chip is a live reference, so a
// document holding one holds something a rewrite would turn into dead text.
type ChipCounts struct {
	Person   int `json:"person"`
	Date     int `json:"date"`
	RichLink int `json:"rich_link"`
}

// Total is every chip, whatever kind.
func (c ChipCounts) Total() int { return c.Person + c.Date + c.RichLink }

// Survey reports what the document holds, and the warnings that came with
// reading it. It never fails: a read the caller could not make is a warning
// naming what is missing, because a survey that refuses to answer tells nobody
// what is in the document.
func Survey(in Input) (Report, []string) {
	var warnings []string
	threads, unplaced := comments.Threads(in.Comments, in.Document)
	// Only when there was a document to place them in. comments.Threads reports
	// every comment unplaced on a nil document, so warning per comment there
	// blames a Docs read nobody made, and the one warning below is the whole of
	// what is true.
	if in.Document != nil {
		for _, id := range unplaced {
			warnings = append(warnings, fmt.Sprintf(
				"comment %s: the Docs read placed no range for it, so the survey carries the thread with no position", id))
		}
	}
	if in.ExportErr != nil {
		warnings = append(warnings, fmt.Sprintf(
			"the docx export could not be read, so every thread is reported unmatched: %v", in.ExportErr))
	}
	threads = docx.Match(threads, in.Export)

	// The list is never null. encoding/json writes a nil slice as null, and a
	// skill reading named_ranges must not get two shapes for the one fact that
	// a document has none. Witnessed is built with make for the same reason.
	r := Report{
		Schema:      Schema,
		Threads:     threadCounts(threads),
		NamedRanges: []docs.NamedRange{},
		Suggestions: SuggestionCounts{IDs: []string{}},
	}
	if in.Document == nil {
		// Not an error, and not nothing to protect either. A survey with no
		// document read knows nothing about chips, suggestions or ranges, and
		// answering "nothing to protect" would be the one false fact in the
		// field a later run reads before it writes.
		return r, append(warnings,
			"no document read reached the survey, so the chips, the pending suggestions and the named ranges are unknown")
	}
	r.DocumentID = in.Document.ID
	r.RevisionID = in.Document.RevisionID
	r.Title = in.Document.Title
	r.Tabs = len(in.Document.Tabs)
	r.NamedRanges = append(r.NamedRanges, in.Document.NamedRanges()...)

	chips, unnamed, onElements := walkTabs(in.Document)
	r.Chips = chips
	// Every id, from both sides, before the two are told apart. The listing's
	// ids and the elements' ids are one set to the read-back, which asks what
	// is still pending rather than which walk saw it.
	pendingIDs := map[string]bool{}
	for id := range onElements {
		pendingIDs[id] = true
	}
	// An id the pending walk saw is already in Pending, so what is left is the
	// suggestions no listing this binary prints can report.
	for _, id := range suggestions.IDs(in.Document) {
		pendingIDs[id] = true
		delete(onElements, id)
	}
	r.Suggestions = SuggestionCounts{
		Pending:    len(suggestions.All(in.Document)),
		OnElements: len(onElements),
		IDs:        sortedSet(pendingIDs),
	}
	for _, member := range unnamed {
		warnings = append(warnings, fmt.Sprintf(
			"the document holds a paragraph element this read cannot name (%s), so the survey counts it as nothing", member))
	}
	if n := r.Suggestions.OnElements; n > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d pending suggestion(s) sit only on elements the suggestions listing cannot report, so they are counted and not listed", n))
	}
	// A named range is the fifth, and it is the one this field is read for. It
	// is a label Docs keeps in step with its own edits, so a replacement of the
	// words it covers takes it with them, and M7b reads this field before it
	// writes. The survey already lists them for that reason.
	r.NothingToProtect = len(threads) == 0 && r.Suggestions.Pending == 0 &&
		r.Suggestions.OnElements == 0 && chips.Total() == 0 && len(unnamed) == 0 &&
		len(r.NamedRanges) == 0
	return r, warnings
}

// threadCounts is the threads by state and by witness. It is one walk, so the
// per-thread list and the counts cannot disagree.
func threadCounts(threads []comments.Thread) ThreadCounts {
	c := ThreadCounts{Witnessed: make([]ThreadWitness, 0, len(threads))}
	for _, t := range threads {
		if t.Resolved {
			c.Resolved++
		} else {
			c.Open++
		}
		switch t.Witness {
		case docx.WitnessAnchored:
			c.Witness.Anchored++
		case docx.WitnessDetached:
			c.Witness.Detached++
		default:
			c.Witness.Unmatched++
		}
		c.Witnessed = append(c.Witnessed, ThreadWitness{ID: t.ID, Witness: t.Witness, Resolved: t.Resolved})
	}
	return c
}

// tally is what one walk of the tabs collects: the chips by kind, the members
// of the paragraph elements the read could not name, and the suggestion ids
// carried by runs the pending listing skips. One walk, because three walks over
// one document are three chances for the counts to disagree about it.
type tally struct {
	chips      ChipCounts
	unnamed    map[string]bool
	onElements map[string]bool
}

// walkTabs counts the chips in every tab, names the paragraph elements the read
// could not name, and collects the suggestion ids the pending walk cannot see.
// The second return is why the decoder reports an unknown element instead of
// dropping it: an element gdoc has never seen is the one thing a count of chips
// cannot speak for. The third is the same argument for a suggestion: an id on a
// run the pending walk skips reaches no listing, so the survey has to count it
// here or answer nothing to protect over it.
func walkTabs(d *docs.Document) (ChipCounts, []string, map[string]bool) {
	t := tally{unnamed: map[string]bool{}, onElements: map[string]bool{}}
	for _, tab := range d.Tabs {
		countBlocks(tab.Body, &t)
	}
	unnamed := make([]string, 0, len(t.unnamed))
	for member := range t.unnamed {
		unnamed = append(unnamed, member)
	}
	// Sorted, because a map is walked in no order and one document must survey
	// the same way twice.
	sort.Strings(unnamed)
	return t.chips, unnamed, t.onElements
}

// countBlocks walks paragraphs and table cells, because a chip in a cell is a
// chip in the document.
func countBlocks(bs []docs.Block, t *tally) {
	for _, b := range bs {
		if b.Paragraph != nil {
			for _, r := range b.Paragraph.Runs {
				count(r, t)
			}
		}
		if b.Table == nil {
			continue
		}
		for _, row := range b.Table.Rows {
			for _, cell := range row {
				countBlocks(cell.Blocks, t)
			}
		}
	}
}

// count is one run. A footnote's text is flattened by the decoder, so a chip
// inside one is in no tab's blocks and is not counted here: that is the same
// hole suggestions inside footnotes already fall into, and it is written down
// in docs/backlog rather than guessed at.
func count(r docs.Run, t *tally) {
	if r.Kind != docs.KindText {
		// Every kind of run carries its own suggestion id lists, and
		// suggestions.All reads the text ones alone. Whichever of these the
		// pending walk also saw is deleted by the caller.
		for _, id := range r.InsertionIDs {
			t.onElements[id] = true
		}
		for _, id := range r.DeletionIDs {
			t.onElements[id] = true
		}
	}
	switch r.Kind {
	case docs.KindPerson:
		t.chips.Person++
	case docs.KindDate:
		t.chips.Date++
	case docs.KindRichLink:
		t.chips.RichLink++
	case docs.KindUnknown:
		member := "no member at all"
		if r.Detail != nil && r.Detail.Member != "" {
			member = r.Detail.Member
		}
		t.unnamed[member] = true
	}
}

// sortedSet is a set of ids as a list, in one order and never null. A map is
// walked in no order, and a skill reading ids must not get two shapes for the
// one fact that a document has none pending.
func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
