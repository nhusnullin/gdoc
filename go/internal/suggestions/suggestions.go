// Package suggestions is what is pending in a document, and what stopped being
// pending since the last look.
//
// Drive's markdown export renders a document as though its pending suggestions
// did not exist, so a review done entirely in suggesting mode exports identical
// to its baseline and a diff sees nothing at all. That silent failure is why
// this package reads the Docs tree instead: SUGGESTIONS_INLINE is the only view
// carrying the ids, and an id is what says this text is pending rather than
// written.
//
// Nothing here judges anything. A suggestion that left the document was
// accepted or rejected, and which one it was is not in the API: the id is gone
// either way. So Gone reports that it left, with what it said last time, and
// the skill reads the document text and decides. This is the milestone's
// facts-only rule at its sharpest point.
package suggestions

import (
	"strings"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/frontmatter"
)

// headingStyles are the named styles that set the section a suggestion is
// reported under. TITLE and SUBTITLE count: they are text above the first
// heading, and "under the title" is a better answer than "".
var headingStyles = []string{"HEADING_", "TITLE", "SUBTITLE"}

// Pending is one suggestion the document is carrying right now. The id is
// Docs's own, so it is the same string across reads and the one thing a
// snapshot can be compared on.
type Pending struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Section string `json:"section"`
	Text    string `json:"text"`
}

// Gone is a suggestion the last snapshot saw and this read does not. SeenAt is
// when the snapshot was taken, which is the last moment the suggestion is known
// to have been pending.
type Gone struct {
	Pending
	SeenAt time.Time `json:"seen_at"`
}

// List is every pending suggestion in the document, in reading order.
//
// Runs sharing one id and one kind are joined into one suggestion: Docs cuts a
// typed sentence at every formatting boundary, so one edit arrives as several
// runs. Every id a run carries is reported, because a run inside two
// overlapping suggestions belongs to both of them.
func List(d *docs.Document) []Pending {
	var w walker
	for _, tab := range d.Tabs {
		// A heading in one tab is not above anything in the next one.
		w.section = ""
		w.blocks(tab.Body)
	}
	return w.kept()
}

// GoneSince is every item in the snapshot whose id is not pending now, with
// what it said when it was last seen. A nil snapshot means nothing was ever
// seen, so nothing can have left.
//
// The match is on the id alone. One replacement is one id carried by a deletion
// run and an insertion run, so an id that left takes both of its halves with
// it, and each half still reports its own text.
func GoneSince(seen *frontmatter.SuggestionsSeen, now []Pending) []Gone {
	if seen == nil {
		return nil
	}
	pending := make(map[string]bool, len(now))
	for _, p := range now {
		pending[p.ID] = true
	}
	var out []Gone
	for _, item := range seen.Items {
		if pending[item.ID] {
			continue
		}
		out = append(out, Gone{
			Pending: Pending{ID: item.ID, Kind: item.Kind, Section: item.Section, Text: item.Text},
			SeenAt:  seen.At,
		})
	}
	return out
}

// Snapshot is what the front matter records after a successful read: every
// pending suggestion as it looked, and when the look happened. A read that
// found nothing still gets a snapshot, because "nothing was pending at this
// time" is the fact the next run needs to say what went.
//
// The time is carried in UTC, so two machines write the same line.
func Snapshot(now []Pending, at time.Time) *frontmatter.SuggestionsSeen {
	items := make([]frontmatter.SuggestionSeen, 0, len(now))
	for _, p := range now {
		items = append(items, frontmatter.SuggestionSeen{
			ID: p.ID, Kind: p.Kind, Section: p.Section, Text: p.Text,
		})
	}
	return &frontmatter.SuggestionsSeen{At: at.UTC(), Items: items}
}

// walker collects suggestions in reading order, tracking the heading above
// them. found keeps the order; at maps an id and kind to where its suggestion
// sits in found, so runs that share one id join whatever ran between them.
type walker struct {
	section string
	found   []Pending
	at      map[string]int
}

// blocks walks a body. A table holds its own content, and a suggested fee in a
// table is exactly the edit that must not be missed, so the walk goes into it.
func (w *walker) blocks(bs []docs.Block) {
	for _, b := range bs {
		if b.Paragraph != nil {
			w.paragraph(b.Paragraph)
			continue
		}
		for _, row := range b.Table {
			for _, cell := range row {
				w.blocks(cell.Blocks)
			}
		}
	}
}

func (w *walker) paragraph(p *docs.Paragraph) {
	if isHeading(p.Style) {
		w.section = paragraphText(p)
	}
	for _, r := range p.Runs {
		// Only a text run is a suggestion here. A footnote reference carries
		// its number as text, and reporting "1" as a suggested insertion would
		// be a lie; a suggested picture is
		// docs/backlog/read-pictures-and-drawings.md.
		if r.Kind != docs.KindText {
			continue
		}
		// A deletion before an insertion, which is the order Docs shows a
		// replacement in and the order the read text prints it in.
		for _, id := range r.DeletionIDs {
			w.add(id, frontmatter.KindDeletion, r.Text)
		}
		for _, id := range r.InsertionIDs {
			w.add(id, frontmatter.KindInsertion, r.Text)
		}
	}
}

// add joins the run onto the suggestion it belongs to, or starts one.
func (w *walker) add(id, kind, text string) {
	key := kind + "\x00" + id
	if i, has := w.at[key]; has {
		w.found[i].Text += text
		return
	}
	if w.at == nil {
		w.at = map[string]int{}
	}
	w.at[key] = len(w.found)
	w.found = append(w.found, Pending{ID: id, Kind: kind, Section: w.section, Text: text})
}

// kept drops the suggestions whose text is whitespace, which say nothing a
// reader can act on. The text of the rest is carried as it is: the trailing
// space of an inserted word is part of the edit.
func (w *walker) kept() []Pending {
	var out []Pending
	for _, p := range w.found {
		if strings.TrimSpace(p.Text) == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func isHeading(style string) bool {
	for _, prefix := range headingStyles {
		if strings.HasPrefix(style, prefix) {
			return true
		}
	}
	return false
}

// paragraphText is the heading's own text, with the newline Docs ends every
// paragraph with trimmed off.
func paragraphText(p *docs.Paragraph) string {
	var b strings.Builder
	for _, r := range p.Runs {
		if r.Kind == docs.KindText {
			b.WriteString(r.Text)
		}
	}
	return strings.TrimSpace(b.String())
}
