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
//
// The line is the block's own, not the definition's, because a note written
// over several lines sends an author to the line they have to change. A
// picture is an inline and takes the line of the paragraph it sits in, which
// is that paragraph's own. The definition's line is the answer only for a
// block carrying no source position anywhere under it, which a horizontal rule
// is.
func (r *renderer) footnoteBlocks(note *east.Footnote) []*etree.Element {
	opens := 1
	if at, ok := r.descendantLine(note); ok {
		opens = at
	}
	var blocks []*etree.Element
	for child := note.FirstChild(); child != nil; child = child.NextSibling() {
		line := opens
		if at, ok := r.descendantLine(child); ok {
			line = at
		}
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

// footnoteReference is a footnote mark in the prose.
//
// A colon after the mark says nothing about it. The one "[^label]" that is not
// a mark is the definition's own opener, and that is known by where it sits
// rather than by what follows it: footnoteDefinition is anchored to the start
// of the line, and goldmark reads "[^label]:" in the middle of a sentence as a
// mark like any other. Skipping every mark with a colon after it refused a
// note whose reference happened to end a clause, and let an undefined one
// through. TestAFootnoteMarkFollowedByAColonIsStillAMark is the pin.
var footnoteReference = regexp.MustCompile(`\[\^([^\]]+)\]`)

// definitionOpener is where the definition's own "[^label]" starts on this
// line, or -1 where the line opens no definition.
func definitionOpener(l sourceLine) int {
	if !l.definition {
		return -1
	}
	return strings.Index(l.text, "[")
}

// atxHeading is a heading line, which is where a footnote mark is refused.
var atxHeading = regexp.MustCompile(`^ {0,3}#{1,6}(\s|$)`)

// checkFootnotes is the question asked of the note before a block is built:
// does every mark have a definition, does every definition have a mark, is a
// label defined twice, and is any mark inside a heading.
//
// Three of the four are read off the source lines, for the reason the package
// comment gives, and the whole file at once so that the line a refusal names
// is counted the one way internal/markers counts it. scan.go holds which
// lines are read and why a code block is not one of them.
//
// Which labels are defined is the parse's own answer, in definedFootnotes. A
// line scan sees a definition at the margin only, and goldmark reads one
// wherever a block may start: a definition inside a block quote or inside a
// list item is lifted into the document's notes and renders, and one inside a
// block of HTML defines nothing at all. Both were refused by the line the mark
// sits on, one of them with a sentence saying the file does not define a note
// it does define. TestADefinitionGoldmarkLiftsOutOfABlockQuoteOrAListItemIsNotRefused
// and TestAMarkWhoseDefinitionSitsInABlockOfHTMLIsRefused are the pins.
//
// The heading question is asked of the tree as well, in headingBlock, because
// an ATX heading is the only one a line scan can see.
func checkFootnotes(source []byte, root ast.Node) error {
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
	var refusal error
	defined := definedFootnotes(root)
	checkLines(source, root, func(l sourceLine) {
		if refusal != nil {
			return
		}
		if l.definition {
			if at, seen := defs[l.label]; seen {
				// goldmark matches a mark against the first definition of a
				// label and its transformer removes the rest, so the words
				// here would reach no page and nothing would say so.
				refusal = fmt.Errorf("line %d: the footnote [^%s] is defined again here, after line %d, and goldmark keeps the first definition only, so these words would reach no page of the document", l.at, l.label, at)
				return
			}
			defs[l.label] = l.at
			defOrder = append(defOrder, l.label)
		}
		heading := atxHeading.MatchString(l.text)
		opener := definitionOpener(l)
		for _, m := range footnoteReference.FindAllStringSubmatchIndex(l.text, -1) {
			label := l.text[m[2]:m[3]]
			if m[0] == opener || strings.TrimSpace(label) == "" {
				continue
			}
			if heading {
				refusal = fmt.Errorf("line %d: the footnote mark [^%s] is inside a heading, which Word copies into the contents field, so the note would be marked from the contents list as well as from the page", l.at, label)
				return
			}
			refs = append(refs, mark{label: label, line: l.at, prose: !l.inside})
		}
	})
	if refusal != nil {
		return refusal
	}
	for _, ref := range refs {
		if !defined[ref.label] {
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

// definedFootnotes is every label the parse resolved to a note under the page.
//
// It is the set goldmark's own transformer leaves standing: a definition
// nothing references is removed from the list, and a mark naming no definition
// is left in the prose as the words the author wrote. So the labels here are
// exactly the ones a mark in the prose may name, wherever in the file their
// definition was written.
func definedFootnotes(root ast.Node) map[string]bool {
	out := map[string]bool{}
	if root == nil {
		return out
	}
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if note, ok := node.(*east.Footnote); ok {
			out[string(note.Ref)] = true
		}
		return ast.WalkContinue, nil
	})
	return out
}

// checkFootnotesInPictures refuses a footnote mark inside a picture's alt
// text, which the line scan cannot see: the alt text is written on a line of
// prose like any other, so the scan reads the mark as the document's own
// words.
//
// It is a refusal rather than a warning, and for the reason the heading and
// the table cell are refused: an alt text becomes the picture's caption, the
// caption carries no mark, and the note would be written under the page with
// nothing on any page pointing at it. The envelope would then count a footnote
// the document does not have.
// TestAFootnoteMarkInsideAPicturesAltTextIsRefusedByLine is the pin.
func (r *renderer) checkFootnotesInPictures(root ast.Node) error {
	var refusal error
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if _, picture := node.(*ast.Image); !picture {
			return ast.WalkContinue, nil
		}
		if mark := firstFootnoteLink(node); mark != nil {
			refusal = fmt.Errorf("line %d: a footnote mark inside a picture's alt text is refused, because the alt text becomes the picture's caption and a caption carries no mark, so the note would be written under the page with nothing pointing at it", r.markLine(mark))
			return ast.WalkStop, nil
		}
		return ast.WalkSkipChildren, nil
	})
	return refusal
}

// firstFootnoteLink is the first footnote mark anywhere under this node.
func firstFootnoteLink(node ast.Node) ast.Node {
	var found ast.Node
	_ = ast.Walk(node, func(at ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || found != nil {
			return ast.WalkContinue, nil
		}
		if _, ok := at.(*east.FootnoteLink); ok {
			found = at
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}
