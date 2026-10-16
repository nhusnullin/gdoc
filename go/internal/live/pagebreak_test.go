// M19's measurement: what `read` gives back for the two kinds of page break.
//
// The docx route needed no gate. Drive's import of `w:pageBreakBefore` is
// already how the front matter's own breaks reach a document, and the live
// round trip has excused their lines since M13. What nobody had measured is
// the shape the Docs answer comes back in: whether a break paragraph of its
// own is a paragraph holding one `pageBreak` element and no text, and whether
// the break on the first body heading rides inside the heading paragraph or
// stands as a paragraph before it.
//
// So this publishes one note holding both, reads it back, and prints what it
// found. It is a measurement and it asserts nothing about the answer: a test
// that failed when Google answered differently would be deciding the thing it
// was sent to find out. It fails on a publish or a read that failed, and on
// nothing else. What it printed is written into docs/v2/MEASURED.md by hand.
//
// The answer, 2026-10-16: neither kind is a pageBreak element. Both are
// pageBreakBefore on the paragraph's own style, which internal/docs did not
// decode, so read printed nothing where either page turned. Task 5 of the
// milestone decodes the flag, so this run now prints a [page break] line for
// both kinds and export writes `---` for the body's. It still asserts nothing
// about the answer: the shape is Google's, the recheck line in MEASURED.md
// says when to ask again, and the offline tests of internal/docs,
// internal/view and internal/export are the pins on what gdoc does with it.

package live

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/view"
)

// pageBreakNoteSource is the note the measurement publishes: a heading, a
// paragraph, `---`, a paragraph. The heading is the first body heading, so the
// house's own `first_heading_page_break` puts a break there, and the `---`
// between the two paragraphs is the author's. One note, both kinds.
//
// It is written into a temp directory rather than kept as a fixture, for the
// reason TestLivePublish's note is: the document it makes is trashed a moment
// later and a fixture pointing at it would point at nothing.
const pageBreakNoteSource = `---
title: Live Page Break Measurement
doc_type: Note
version: 1.0
date: 2026-10-16
owner: Chief Risk Officer
classification: internal
---

# Where the page turns

This paragraph stands before the break the author wrote, and it is the last
thing on its page.

---

This paragraph stands after it, and a reader turns a page to reach it.
`

// The two phrases the measurement finds its paragraphs by. They are literals
// rather than words read back out of the note, because what the run reports is
// what a person really published.
const (
	pageBreakBefore = "the last"
	pageBreakAfter  = "turns a page"
)

// TestLivePageBreakShape is the measurement. It publishes the note above into
// the test folder, reads the document back through the same URL `read` asks
// for, prints every paragraph that carries a break in either shape, and
// trashes the document on the way out.
func TestLivePageBreakShape(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	dir := t.TempDir()
	note := filepath.Join(dir, "live-page-break.md")
	if err := os.WriteFile(note, []byte(pageBreakNoteSource), 0o600); err != nil {
		t.Fatalf("the note could not be written: %v", err)
	}
	built := renderNoteFile(t, note)
	title := "gdoc M19 page break measurement " + time.Now().UTC().Format(time.RFC3339)
	id := uploadForDrift(t, ctx, s, folder, title, built.docx)

	// The raw answer first, so the two readings below are the same bytes: the
	// tree `read` walks, and the fields that tree does not carry. docs.URL is
	// the read every command makes, tabs and suggestions included.
	var raw json.RawMessage
	if err := s.GetJSON(ctx, docs.URL(id), &raw); err != nil {
		t.Fatalf("the published document (%s) could not be read back: %v", id, err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("the read of %s did not parse: %v", id, err)
	}
	t.Logf("measuring document %s, %d tabs, titled %q", id, len(d.Tabs), d.Title)

	// Reading one: the tree, which is what view prints from. A pageBreak is a
	// run of a paragraph here, so a break paragraph of its own shows as a
	// paragraph whose only run is that kind.
	paragraphs := walkedParagraphs(d)
	t.Logf("the document walked as %d paragraphs", len(paragraphs))
	breaks := 0
	for _, p := range paragraphs {
		if !strings.Contains(strings.Join(p.kinds, " "), docs.KindPageBreak) {
			continue
		}
		breaks++
		t.Logf("walked paragraph %d holds a page break: style %s, runs [%s], text %q",
			p.at, p.style, strings.Join(p.kinds, ", "), p.text)
	}
	if breaks == 0 {
		t.Logf("no paragraph of the walked tree holds a %s run at all", docs.KindPageBreak)
	}

	// Reading two: the fields the tree does not carry. Docs states a break
	// before a paragraph two ways, as an element in the paragraph and as
	// pageBreakBefore on its style, and only one of them reaches view.
	for _, p := range rawParagraphs(raw) {
		if !p.breakBefore && !p.holds("pageBreak") {
			continue
		}
		t.Logf("raw paragraph: style %s, pageBreakBefore %t, elements [%s], text %q",
			p.style, p.breakBefore, strings.Join(p.elements, ", "), p.text)
	}

	// And the shape a person sees: the projection, around each of the two
	// paragraphs the note wrote.
	text, warnings := view.Text(d)
	for _, w := range warnings {
		t.Logf("projection warning: %s", w)
	}
	for _, anchor := range []string{pageBreakBefore, pageBreakAfter, "Where the page turns"} {
		t.Logf("what read prints around %q:\n%s", anchor, window(text, anchor))
	}
}

// walkedParagraph is one paragraph of the tree: where it stands in reading
// order, its named style, the kinds of its runs, and its text.
type walkedParagraph struct {
	at    int
	style string
	kinds []string
	text  string
}

// walkedParagraphs is every paragraph of every tab in reading order, tables
// and the contents list included, because a break could stand in any of them.
func walkedParagraphs(d *docs.Document) []walkedParagraph {
	var out []walkedParagraph
	var blocks func([]docs.Block)
	blocks = func(bs []docs.Block) {
		for _, b := range bs {
			switch {
			case b.Paragraph != nil:
				p := walkedParagraph{at: len(out) + 1, style: b.Paragraph.Style}
				for _, r := range b.Paragraph.Runs {
					p.kinds = append(p.kinds, r.Kind)
					p.text += r.Text
				}
				out = append(out, p)
			case b.Table != nil:
				for _, row := range b.Table.Rows {
					for _, cell := range row {
						blocks(cell.Blocks)
					}
				}
			case b.TOC != nil:
				blocks(b.TOC.Blocks)
			}
		}
	}
	for _, tab := range d.Tabs {
		blocks(tab.Body)
	}
	return out
}

// rawParagraph is one paragraph as Google sent it: the named style, whether
// the style says a page breaks before it, the member name of every element in
// it, and the text its runs hold.
type rawParagraph struct {
	style       string
	breakBefore bool
	elements    []string
	text        string
}

// holds says whether one element member name is in this paragraph.
func (p rawParagraph) holds(member string) bool {
	for _, e := range p.elements {
		if e == member {
			return true
		}
	}
	return false
}

// rawParagraphs is every paragraph anywhere under the answer, in no order
// anyone should rely on. It walks the decoded JSON rather than the tree,
// because pageBreakBefore is a field internal/docs does not decode and the
// question is whether Drive's import used it.
func rawParagraphs(raw json.RawMessage) []rawParagraph {
	var answer any
	if err := json.Unmarshal(raw, &answer); err != nil {
		return nil
	}
	var out []rawParagraph
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if p, ok := t["paragraph"].(map[string]any); ok {
				out = append(out, openRawParagraph(p))
			}
			keys := make([]string, 0, len(t))
			for k := range t {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(t[k])
			}
		case []any:
			for _, child := range t {
				walk(child)
			}
		}
	}
	walk(answer)
	return out
}

// openRawParagraph reads the three facts off one paragraph object.
func openRawParagraph(p map[string]any) rawParagraph {
	var out rawParagraph
	if style, ok := p["paragraphStyle"].(map[string]any); ok {
		out.style, _ = style["namedStyleType"].(string)
		out.breakBefore, _ = style["pageBreakBefore"].(bool)
	}
	elements, _ := p["elements"].([]any)
	for _, raw := range elements {
		e, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		keys := make([]string, 0, len(e))
		for k := range e {
			if k == "startIndex" || k == "endIndex" {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out.elements = append(out.elements, keys...)
		if run, ok := e["textRun"].(map[string]any); ok {
			content, _ := run["content"].(string)
			out.text += content
		}
	}
	return out
}

// window is the line holding one phrase with the line either side of it, which
// is the smallest reading that says what stands next to a break. It prints
// every line as a quoted literal, because a blank line and a line holding a
// marker are the answer here and both vanish when printed plainly.
func window(text, phrase string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if !strings.Contains(line, phrase) {
			continue
		}
		from, to := i-2, i+2
		if from < 0 {
			from = 0
		}
		if to >= len(lines) {
			to = len(lines) - 1
		}
		var b strings.Builder
		for n := from; n <= to; n++ {
			b.WriteString("  ")
			b.WriteString(strconv.Quote(lines[n]))
			b.WriteString("\n")
		}
		return strings.TrimRight(b.String(), "\n")
	}
	return "  (no line holds it)"
}

// TestThePageBreakNoteRendersWithNoNetwork is this file's offline pin, and it
// runs in `make test` like the other three.
//
// The measurement renders a note before it reaches Drive, and a note that no
// longer renders, or one somebody edited until it held no `---` at all, would
// otherwise leave a live run green and the question unasked. So this renders
// it with neither variable set and holds the note to what it is for: one body
// break, one first heading, and the two paragraphs the window above is read
// around.
func TestThePageBreakNoteRendersWithNoNetwork(t *testing.T) {
	path := filepath.Join(t.TempDir(), "live-page-break.md")
	if err := os.WriteFile(path, []byte(pageBreakNoteSource), 0o600); err != nil {
		t.Fatalf("the measurement's note could not be written: %v", err)
	}
	doc := renderNoteFile(t, path)
	if len(doc.docx) == 0 {
		t.Error("the measurement's note rendered no bytes")
	}
	if n := strings.Count(pageBreakNoteSource, "\n---\n"); n != 2 {
		// Two: the front matter's closing fence, and the author's own break.
		t.Errorf("the note holds %d `---` lines, and it holds the closing fence and one body break", n)
	}
	if !strings.Contains(pageBreakNoteSource, "\n# ") {
		t.Error("the note holds no body heading, so the first heading's own break is not in the document")
	}
	for _, phrase := range []string{pageBreakBefore, pageBreakAfter} {
		if !strings.Contains(pageBreakNoteSource, phrase) {
			t.Errorf("the note does not read %q, so the projection window finds nothing", phrase)
		}
	}
}
