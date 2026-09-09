// Package restyle is the survey of a document, and nothing else yet. It reports
// what a document holds before anything is done to it: its threads with a
// witness for each, what is pending, its chips, its tabs, its named ranges and
// the revision the reads were made against.
//
// It takes no session and touches no wire, like every other reader package
// here. The caller makes the four reads and hands the decoded answers over, so
// every case in this file is testable on a value a test wrote out.
//
// Nothing here judges anything. NothingToProtect is a fact about three counts
// being zero, never a recommendation to restyle: whether a document is worth
// restyling is Nail's, reading the counts. The witness has the two limits it
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

// Input is the four reads, already decoded. Export is the docx the witness is
// read from, and ExportErr is why there is none: an export that failed is a
// warning on a survey that still carries its threads, exactly as
// `comments --witness` behaves, rather than a failed survey.
type Input struct {
	Document  *docs.Document
	Comments  []comments.RawComment
	Export    *docx.File
	ExportErr error
}

// Report is the survey. Every field is a count, an id or a list.
type Report struct {
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

// SuggestionCounts is what is pending. The count is suggestions.All's and not
// List's: List drops a suggestion whose text is only whitespace, and a
// whitespace-only suggestion is still something a replacement would destroy, so
// counting from List reports nothing to protect on a document that has
// something to protect.
type SuggestionCounts struct {
	Pending int `json:"pending"`
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
	for _, id := range unplaced {
		warnings = append(warnings, fmt.Sprintf(
			"comment %s: the Docs read placed no range for it, so the survey carries the thread with no position", id))
	}
	if in.ExportErr != nil {
		warnings = append(warnings, fmt.Sprintf(
			"the docx export could not be read, so every thread is reported unmatched: %v", in.ExportErr))
	}
	threads = docx.Match(threads, in.Export)

	r := Report{Threads: threadCounts(threads)}
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
	r.Suggestions = SuggestionCounts{Pending: len(suggestions.All(in.Document))}
	r.NamedRanges = in.Document.NamedRanges()

	chips, unnamed := walkTabs(in.Document)
	r.Chips = chips
	for _, member := range unnamed {
		warnings = append(warnings, fmt.Sprintf(
			"the document holds a paragraph element this read cannot name (%s), so the survey counts it as nothing", member))
	}
	r.NothingToProtect = len(threads) == 0 && r.Suggestions.Pending == 0 &&
		chips.Total() == 0 && len(unnamed) == 0
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

// walkTabs counts the chips in every tab, and names the paragraph elements the
// read could not name. The second return is why the decoder reports an unknown
// element instead of dropping it: an element gdoc has never seen is the one
// thing a count of chips cannot speak for.
func walkTabs(d *docs.Document) (ChipCounts, []string) {
	var chips ChipCounts
	seen := map[string]bool{}
	for _, t := range d.Tabs {
		countBlocks(t.Body, &chips, seen)
	}
	unnamed := make([]string, 0, len(seen))
	for member := range seen {
		unnamed = append(unnamed, member)
	}
	// Sorted, because a map is walked in no order and one document must survey
	// the same way twice.
	sort.Strings(unnamed)
	return chips, unnamed
}

// countBlocks walks paragraphs and table cells, because a chip in a cell is a
// chip in the document.
func countBlocks(bs []docs.Block, chips *ChipCounts, seen map[string]bool) {
	for _, b := range bs {
		if b.Paragraph != nil {
			for _, r := range b.Paragraph.Runs {
				count(r, chips, seen)
			}
		}
		for _, row := range b.Table {
			for _, cell := range row {
				countBlocks(cell.Blocks, chips, seen)
			}
		}
	}
}

// count is one run. A footnote's text is flattened by the decoder, so a chip
// inside one is in no tab's blocks and is not counted here: that is the same
// hole suggestions inside footnotes already fall into, and it is written down
// in docs/backlog rather than guessed at.
func count(r docs.Run, chips *ChipCounts, seen map[string]bool) {
	switch r.Kind {
	case docs.KindPerson:
		chips.Person++
	case docs.KindDate:
		chips.Date++
	case docs.KindRichLink:
		chips.RichLink++
	case docs.KindUnknown:
		member := "no member at all"
		if r.Detail != nil && r.Detail.Member != "" {
			member = r.Detail.Member
		}
		seen[member] = true
	}
}
