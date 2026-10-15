package body

// The footnotes: the numbers a reference takes, the notes under the page, and
// the three shapes that refuse the build.
//
// Two of the three refusals are read off the source lines rather than off the
// tree, because goldmark is silent in both: its inline parser leaves "[^x]" as
// prose when nothing defines x, and its transformer removes a definition
// nothing references. doc.go holds why that is a refusal and not a warning.

import (
	"fmt"
	"regexp"
	"strings"

	"gdoc/internal/render"
	"github.com/beevik/etree"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// Footnote is one note under the page: the number the body's reference run
// names, and the paragraphs the note holds. internal/render writes them into
// word/footnotes.xml, one w:footnote per entry, in this order.
//
// It is render's own type, the way Media is, so the numbers this package hands
// out are the numbers the part is written from and the two cannot drift.
type Footnote = render.Footnote

// The two styles a footnote is set in, as Word names them. internal/render
// writes both into word/styles.xml: the mark in the prose names the character
// style, which is what raises it, and every paragraph of the note names the
// paragraph style, which is the house's footnote_text.
const (
	footnoteRefStyle  = "FootnoteReference"
	footnoteTextStyle = "FootnoteText"
	// Word's own name for a raised character, which a reference mark is. The
	// run states it as well as naming the style, because Google Docs has no
	// character styles and keeps only what the run itself says.
	vertAlignSuperscript = "superscript"
)

// footnoteNumbers is every footnote mark in the note's prose, in the order a
// reader meets one, and the number each takes.
//
// The number is the position in that order and never east.FootnoteLink.Index,
// which is the definition's: goldmark gives both marks of a label referenced
// twice the same Index, and decision 2.2 B is that each mark is its own note.
//
// The definitions themselves are skipped. A mark inside one is not prose, so
// it takes no number, and footnoteBlocks names it on the envelope.
func footnoteNumbers(root ast.Node) ([]*east.FootnoteLink, map[ast.Node]int) {
	var order []*east.FootnoteLink
	number := map[ast.Node]int{}
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if _, defs := node.(*east.FootnoteList); defs {
			return ast.WalkSkipChildren, nil
		}
		if link, ok := node.(*east.FootnoteLink); ok {
			order = append(order, link)
			number[link] = len(order)
		}
		return ast.WalkContinue, nil
	})
	return order, number
}

// footnoteList writes the notes under the page, one per mark in the prose.
//
// goldmark collects every definition into one list at the end of the document
// and sorts it by first reference, so the list is walked through the marks
// rather than through its own children: a label referenced twice is two notes
// carrying one text, and the text is rendered once and copied. Copied rather
// than shared, because adding one element to two trees moves it out of the
// first.
func (r *renderer) footnoteList(list *east.FootnoteList) error {
	byIndex := map[int]*east.Footnote{}
	for note := list.FirstChild(); note != nil; note = note.NextSibling() {
		if typed, ok := note.(*east.Footnote); ok {
			byIndex[typed.Index] = typed
		}
	}
	rendered := map[int][]*etree.Element{}
	for _, link := range r.footnoteOrder {
		note, ok := byIndex[link.Index]
		if !ok {
			// goldmark makes a link only for a definition it found, so there
			// is no note here to lose. Said rather than passed over.
			r.fail("line %d: a footnote mark names a note this walk cannot find", r.curLine)
			return nil
		}
		blocks, seen := rendered[link.Index]
		if !seen {
			blocks = r.footnoteBlocks(note)
			rendered[link.Index] = blocks
		} else {
			blocks = copyBlocks(blocks)
		}
		r.footnotes = append(r.footnotes, Footnote{ID: r.footnoteNumber[link], Blocks: blocks})
		r.counts.Footnotes++
	}
	r.afterTable = false
	return nil
}

// footnoteBlocks is one definition as the paragraphs the note holds.
//
// Inside a note, paragraphs with inline marks only. A list, a table, a
// picture, a code block, a heading, a link or a second footnote mark is named
// by its line and left out: a note under the page is a sentence, and the house
// style has no geometry for a table in one.
func (r *renderer) footnoteBlocks(note *east.Footnote) []*etree.Element {
	line := 1
	if at, ok := r.descendantLine(note); ok {
		line = at
	}
	var blocks []*etree.Element
	for child := note.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.(type) {
		case *ast.Paragraph, *ast.TextBlock:
		case *east.FootnoteBacklink:
			// goldmark's own jump back up to the mark, which is for HTML and
			// says nothing in a document. The inline walk skips it too, and a
			// note whose last block is not a paragraph carries it here.
			continue
		default:
			r.warn("line %d: %s inside a footnote is not rendered in the house style and was left out",
				line, footnoteBlockName(child))
			continue
		}
		r.curLine = line
		r.warnImages(collectImages(child, r.source), line, "a footnote")
		r.warnRawHTML(child, line)
		runs := r.footnoteRuns(child, line)
		if len(runs) == 0 {
			continue
		}
		blocks = append(blocks, r.footnoteParagraph(runs, len(blocks) == 0))
	}
	if len(blocks) == 0 {
		// A definition whose every block the house style cannot set still
		// needs a note, or the mark in the prose points at nothing.
		blocks = append(blocks, r.footnoteParagraph(nil, true))
	}
	return blocks
}

// footnoteRuns is one of a note's paragraphs as runs, with what the footnotes
// part cannot carry taken out and named.
//
// A link is a relationship in word/_rels/document.xml.rels, which the
// footnotes part does not name, so its words stay and the link goes. A second
// footnote mark took no number, for the reason footnoteNumbers gives, so it
// reaches here as a run with no number and no text.
func (r *renderer) footnoteRuns(node ast.Node, line int) []Run {
	runs := inlineRuns(node, r.source, r.footnoteNumber, r.marks)
	out := make([]Run, 0, len(runs))
	linked := false
	for _, run := range runs {
		if run.Footnote > 0 {
			// A mark inside a definition takes no number, for the reason
			// footnoteNumbers gives, so nothing reaching here carries one.
			// Guarded rather than trusted: a note holding a note is a
			// footnotes part Word reads as damaged.
			continue
		}
		if run.Link != "" {
			linked = true
			run.Link = ""
		}
		out = append(out, run)
	}
	if linked {
		r.warn("line %d: a link inside a footnote is not rendered in the house style and its words are printed as plain text", line)
	}
	if hasFootnoteLink(node) {
		r.warn("line %d: a footnote inside a footnote is not rendered in the house style and was left out", line)
	}
	return merge(out)
}

// hasFootnoteLink says whether an inline subtree holds a footnote mark,
// however deeply it is nested.
func hasFootnoteLink(node ast.Node) bool {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*east.FootnoteLink); ok {
			return true
		}
		if hasFootnoteLink(child) {
			return true
		}
	}
	return false
}

// footnoteBlockName is what a block inside a note is called in the sentence
// that says it was left out. The words are the ones the block's own warning
// uses elsewhere in this package, so an author reads one vocabulary.
func footnoteBlockName(node ast.Node) string {
	switch node.(type) {
	case *ast.List:
		return "a list"
	case *east.Table:
		return "a table"
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		return "a code block"
	case *ast.Heading:
		return "a heading"
	case *ast.Blockquote:
		return "a block quote"
	case *ast.HTMLBlock:
		return "a block of HTML"
	case *ast.ThematicBreak:
		return "a horizontal rule"
	default:
		return "this block"
	}
}

// copyBlocks is a note's paragraphs again, for the second mark that names it.
func copyBlocks(blocks []*etree.Element) []*etree.Element {
	out := make([]*etree.Element, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, block.Copy())
	}
	return out
}

// footnoteDefinition is a line that defines a footnote: up to three spaces of
// indent, then "[^label]:". Four spaces is a continuation line of the
// definition above it, which is why the indent is bounded.
var footnoteDefinition = regexp.MustCompile(`^ {0,3}\[\^([^\]]+)\]:`)

// footnoteReference is a footnote mark in the prose. A "[^label]" followed by
// a colon is the opening of a definition rather than a mark, and is left to
// footnoteDefinition.
var footnoteReference = regexp.MustCompile(`\[\^([^\]]+)\](:?)`)

// atxHeading is a heading line, which is where a footnote mark is refused.
var atxHeading = regexp.MustCompile(`^ {0,3}#{1,6}(\s|$)`)

// fence opens or shuts a code block, inside which nothing is markdown.
var fence = regexp.MustCompile("^ {0,3}(```+|~~~+)")

// checkFootnotes is the question the source lines are asked before the note is
// parsed: does every mark have a definition, does every definition have a
// mark, and is any mark inside a heading.
//
// It is the source rather than the tree for the reason the package comment
// gives, and it is the whole file at once so that the line a refusal names is
// counted the one way internal/markers counts it. A fenced code block is
// skipped: the block is warned about and left out whole, so the syntax written
// inside one reaches no page and defines nothing.
func checkFootnotes(source []byte) error {
	type mark struct {
		label string
		line  int
		// prose says whether this mark is in the document's own words rather
		// than inside a definition. Only a mark in the prose makes a note, so
		// only a mark in the prose answers for a definition.
		prose bool
	}
	var refs []mark
	defs := map[string]int{}
	var defOrder []string
	inFence := false
	// openDefinition says whether the line before this one was part of a
	// definition, so that an indented line under it is read as its own.
	openDefinition := false
	for i, line := range strings.Split(string(source), "\n") {
		if fence.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		at := i + 1
		indented := strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")
		blank := strings.TrimSpace(line) == ""
		line = withoutCode(line)
		definition := false
		if m := footnoteDefinition.FindStringSubmatch(line); m != nil {
			if label := m[1]; strings.TrimSpace(label) != "" {
				definition = true
				if _, seen := defs[label]; !seen {
					defs[label] = at
					defOrder = append(defOrder, label)
				}
			}
		}
		// A definition runs on through its blank and indented lines, which is
		// goldmark's own rule for continuing one, and ends at the first line
		// that is neither.
		if definition {
			openDefinition = true
		} else if !indented && !blank {
			openDefinition = false
		}
		heading := atxHeading.MatchString(line)
		// A mark on a definition's own line, or on one of its indented
		// continuation lines, is inside a note rather than in the prose.
		inside := definition || (indented && openDefinition)
		for _, m := range footnoteReference.FindAllStringSubmatch(line, -1) {
			label, colon := m[1], m[2]
			if colon == ":" || strings.TrimSpace(label) == "" {
				continue
			}
			if heading {
				return fmt.Errorf("line %d: the footnote mark [^%s] is inside a heading, which Word copies into the contents field, so the note would be marked from the contents list as well as from the page", at, label)
			}
			refs = append(refs, mark{label: label, line: at, prose: !inside})
		}
	}
	for _, ref := range refs {
		if _, ok := defs[ref.label]; !ok {
			return fmt.Errorf("line %d: the footnote mark [^%s] names a note this file does not define, so the document would carry a mark with nothing under the page; goldmark publishes it as the words [^%s] instead", ref.line, ref.label, ref.label)
		}
	}
	// Only a mark in the prose answers for a definition. A mark inside another
	// note takes no number, because the numbers are the prose's reading order
	// and Word has one level of note, so a definition marked only from inside
	// another one would reach no page either.
	marked := map[string]bool{}
	for _, ref := range refs {
		if ref.prose {
			marked[ref.label] = true
		}
	}
	for _, label := range defOrder {
		if !marked[label] {
			return fmt.Errorf("line %d: the footnote [^%s] is defined and nothing in this file's own words marks it, so its words would reach no page of the document; a mark inside another footnote does not count, because Word has one level of note", defs[label], label)
		}
	}
	return nil
}

// withoutCode is the line with its code spans taken out, because nothing
// inside backticks is markdown. goldmark reads no footnote mark in a span, and
// a note explaining a regular expression writes "[^abc]" in one: read as a
// mark, it would refuse a note nothing is wrong with.
//
// A backtick run with no closer on the line is text, which is what goldmark
// makes of it, so the scan carries on from just after it rather than giving up
// on the rest of the line and missing a real mark.
func withoutCode(line string) string {
	var b strings.Builder
	for i := 0; i < len(line); {
		if line[i] != '`' {
			b.WriteByte(line[i])
			i++
			continue
		}
		open := ticks(line, i)
		shut := -1
		for j := i + open; j < len(line); {
			if line[j] != '`' {
				j++
				continue
			}
			n := ticks(line, j)
			if n == open {
				shut = j + n
				break
			}
			j += n
		}
		if shut < 0 {
			b.WriteString(line[i : i+open])
			i += open
			continue
		}
		i = shut
	}
	return b.String()
}

// ticks is the length of the run of backticks starting at i.
func ticks(line string, i int) int {
	n := 0
	for i+n < len(line) && line[i+n] == '`' {
		n++
	}
	return n
}
