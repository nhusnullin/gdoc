// Package body renders a note's markdown into the house style's own idioms.
//
// goldmark is used only as a markdown parser, never as a docx writer, and the
// OOXML is written here. That distinction is the one v1 draws about pandoc,
// and for the same reason: a general-purpose docx writer emits its own styles,
// which the house style does not define, and a reader that meets a dangling
// w:pStyle discards the whole w:pPr around it. Every bullet and every number
// then silently disappears.
//
// Every size, colour, indent and alignment comes from house.Config. What is a
// constant in this package is a value the house file does not state, and each
// one carries the reason it is what it is.
//
// Nothing here reaches the network and nothing here runs a program. A picture
// is read from the note's own directory or decoded out of the markdown, and an
// http address is refused naming the line.
package body

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"gdoc/internal/house"
	"gdoc/internal/render"
)

// Media is one relationship the body needs, a picture or a link. It is
// render's own type, so the ids this package hands out are the ids the
// document's relationships are written from and the two cannot drift.
type Media = render.Media

// Counts is what the walker wrote, as facts. Whether a document with no
// headings or three tables is right is the skill's to say, not this package's.
type Counts struct {
	Paragraphs int `json:"paragraphs"`
	Headings   int `json:"headings"`
	Lists      int `json:"lists"`
	Tables     int `json:"tables"`
	Images     int `json:"images"`
}

// Result is one walk: the blocks, the relationships they name, what could not
// be rendered, and the counts.
type Result struct {
	Blocks   []*etree.Element
	Media    []Media
	Warnings []string
	Counts   Counts
}

// renderer holds the state one walk needs.
type renderer struct {
	cfg      *house.Config
	base     string
	source   []byte
	numberer *headingNumberer

	blocks   []*etree.Element
	media    []Media
	warnings []string
	counts   Counts

	relID      int
	linkIDs    map[string]string
	imageCount int
	docPr      int

	firstHeading bool
	afterTable   bool
	err          error
}

// Render walks the markdown into house-style blocks.
//
// base is the directory a relative picture is resolved against, and numbering
// says whether headings take their literal numbers, which is the note's own
// heading_numbering key.
//
// A construct the house style does not carry is a warning naming the line and
// no block. A picture that cannot be read is an error: a document published
// with a picture silently missing is the failure that only shows up once
// somebody reads it.
func Render(cfg *house.Config, markdown []byte, base string, numbering bool) (Result, error) {
	if cfg == nil {
		return Result{}, fmt.Errorf("body: no house style")
	}
	separator, err := separator(cfg.HeadingNumbering.Level1Format)
	if err != nil {
		return Result{}, fmt.Errorf("body: %w", err)
	}

	source := markdown
	root := parse().Parser().Parse(text.NewReader(source))

	r := &renderer{
		cfg: cfg, base: base, source: source,
		relID: render.FirstMediaRelID, linkIDs: map[string]string{},
		firstHeading: true,
	}
	r.numberer = newHeadingNumberer(numbering, separator, shallowestHeadingLevel(root, source))

	if err := r.walk(root, 0, ""); err != nil {
		return Result{}, err
	}
	if r.err != nil {
		return Result{}, r.err
	}
	return Result{Blocks: r.blocks, Media: r.media, Warnings: r.warnings, Counts: r.counts}, nil
}

// parse is goldmark, configured to the extension set v1 asks pandoc for: pipe
// tables, ==mark==, strikeout and task lists, plus the smart punctuation
// pandoc's markdown format enables by default.
func parse() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			// The default typographer substitutes HTML entities, because its
			// intended renderer is HTML. Written into a w:t those come out as
			// the literal text "&ldquo;", which is both wrong and a different
			// width, so every line after the first one that carries a quote
			// re-wraps. The characters themselves are what this renderer wants.
			extension.NewTypographer(
				extension.WithTypographicSubstitutions(extension.TypographicSubstitutions{
					extension.LeftSingleQuote:  []byte("‘"),
					extension.RightSingleQuote: []byte("’"),
					extension.LeftDoubleQuote:  []byte("“"),
					extension.RightDoubleQuote: []byte("”"),
					extension.EnDash:           []byte("–"),
					extension.EmDash:           []byte("—"),
					extension.Ellipsis:         []byte("…"),
					extension.LeftAngleQuote:   []byte("«"),
					extension.RightAngleQuote:  []byte("»"),
					extension.Apostrophe:       []byte("’"),
				}),
			),
			Mark,
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
}

// fail records the first failure and keeps the rest, because the first one is
// the one that explains the others.
func (r *renderer) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf(format, args...)
	}
}

// warn records a construct the house style does not carry.
func (r *renderer) warn(format string, args ...any) {
	r.warnings = append(r.warnings, fmt.Sprintf(format, args...))
}

// emit adds one block to the document.
func (r *renderer) emit(block *etree.Element) {
	r.blocks = append(r.blocks, block)
}

// nextRelID hands out one relationship id. Pictures and links share the run,
// because two runs could hand one id to each.
func (r *renderer) nextRelID() string {
	id := fmt.Sprintf("rId%d", r.relID)
	r.relID++
	return id
}

// linkID is the relationship one destination took, made once and reused: the
// same address written twice is one relationship.
func (r *renderer) linkID(url string) string {
	if id, ok := r.linkIDs[url]; ok {
		return id
	}
	id := r.nextRelID()
	r.linkIDs[url] = id
	r.media = append(r.media, Media{RelID: id, Target: url})
	return id
}

// line is the 1-based line a node starts on, which is what a warning names.
func (r *renderer) line(node ast.Node) int {
	offset := -1
	if lines := node.Lines(); lines != nil && lines.Len() > 0 {
		offset = lines.At(0).Start
	}
	if offset < 0 {
		if child := firstTextNode(node); child != nil {
			offset = child.Segment.Start
		}
	}
	if offset < 0 || offset > len(r.source) {
		return 1
	}
	return 1 + bytes.Count(r.source[:offset], []byte("\n"))
}

// firstTextNode finds a node's first text leaf, which is the only thing on an
// image or a heading that carries a source position.
func firstTextNode(node ast.Node) *ast.Text {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			return t
		}
		if found := firstTextNode(child); found != nil {
			return found
		}
	}
	return nil
}

// shallowestHeadingLevel finds the smallest heading level anywhere in the
// document, so a body lifted out of a note that already had its own title line
// still numbers from 1.
func shallowestHeadingLevel(root ast.Node, source []byte) int {
	found := 0
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := node.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		// A heading holding nothing but a picture is a figure, so it does not
		// set the document's heading depth either. Otherwise Drive's
		// "# ![][image1]" makes every real heading one level deeper than it is,
		// and they come out numbered "0.1-".
		if len(collectImages(heading, source)) > 0 &&
			strings.TrimSpace(plainText(inlineRuns(heading, source))) == "" {
			return ast.WalkSkipChildren, nil
		}
		if found == 0 || heading.Level < found {
			found = heading.Level
		}
		return ast.WalkContinue, nil
	})
	if found == 0 {
		return 1
	}
	return found
}

// walk renders a block subtree. level counts list nesting: 0 is the body, 1 is
// inside a list, and so on. numID names the list a nested paragraph belongs to.
func (r *renderer) walk(parent ast.Node, level int, numID string) error {
	for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
		if err := r.block(node, level, numID); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) block(node ast.Node, level int, numID string) error {
	switch typed := node.(type) {
	case *ast.Heading:
		return r.headingBlock(typed)

	case *ast.Paragraph, *ast.TextBlock:
		return r.paragraphBlock(node, level, numID)

	case *ast.List:
		// The two lists numbering.xml defines are the two a body may name. A
		// second numbered list therefore carries on from the first, which is
		// docs/backlog/one-numbered-list-per-document.md.
		id := render.BulletNumID
		if typed.IsOrdered() {
			id = render.NumberNumID
		}
		if level == 0 {
			r.counts.Lists++
		}
		for item := typed.FirstChild(); item != nil; item = item.NextSibling() {
			if err := r.walk(item, level+1, id); err != nil {
				return err
			}
		}
		r.afterTable = false
		return nil

	case *east.Table:
		r.emit(r.makeTable(r.tableRows(typed)))
		r.counts.Tables++
		r.afterTable = true
		return nil

	case *ast.Blockquote:
		return r.quoteBlock(typed, level, numID)

	case *ast.FencedCodeBlock:
		// Nail's decision: code blocks stay out of the house style. The note
		// says so on the envelope rather than losing the block in silence.
		//
		// The line named is the fence, which is what the author sees. A fenced
		// block's own lines start at its first line of code, one further down.
		r.warn("line %d: a code block is not rendered in the house style and was left out",
			r.line(node)-1)
		r.afterTable = false
		return nil

	case *ast.CodeBlock:
		r.warn("line %d: a code block is not rendered in the house style and was left out",
			r.line(node))
		r.afterTable = false
		return nil

	case *ast.ThematicBreak:
		r.emit(r.rule())
		r.afterTable = false
		return nil

	case *ast.HTMLBlock:
		r.warn("line %d: a block of HTML is not rendered in the house style and was left out",
			r.line(node))
		r.afterTable = false
		return nil

	default:
		// A container this walker does not name is walked through rather than
		// dropped: losing a block silently is the failure that only shows up
		// once somebody reads the published document.
		if node.HasChildren() {
			return r.walk(node, level, numID)
		}
		r.afterTable = false
		return nil
	}
}

// headingBlock is one heading, its number and any picture it names.
//
// Drive exports a picture that sits on its own line as a heading,
// "# ![][image1]". The inline path renders no picture at all, so it used to
// vanish, leaving an empty heading that still took a number and an empty line
// in the contents list.
func (r *renderer) headingBlock(heading *ast.Heading) error {
	images := collectImages(heading, r.source)
	runs := inlineRuns(heading, r.source)
	plain := plainText(runs)
	r.warnRawHTML(heading, r.line(heading))

	if len(images) > 0 && strings.TrimSpace(plain) == "" {
		// Nothing but a picture. It is a figure, not a heading, so it takes no
		// number and no contents entry.
		if err := r.emitImages(images, r.line(heading)); err != nil {
			return err
		}
		r.afterTable = false
		return nil
	}

	if prefix := r.numberer.prefix(heading.Level, plain); prefix != "" {
		// Merged, so the number and the first word of the heading are one run.
		// Two runs carrying the same marks read the same on the page and make
		// the contents entry two pieces of text to match.
		runs = merge(append([]Run{{Text: prefix}}, runs...))
	}
	// The break goes on the first heading whatever its level, so the body
	// always starts on a clean page after the contents list. Keyed on level 1
	// alone it silently does nothing for a document whose top heading is "##".
	pageBreak := r.firstHeading && r.cfg.Body.FirstHeadingPageBreak
	r.firstHeading = false

	r.emit(r.heading(r.numberer.styleLevel(heading.Level), runs, pageBreak))
	r.counts.Headings++

	// A heading that names a picture keeps its words and gets the picture
	// underneath it, where it reads.
	if err := r.emitImages(images, r.line(heading)); err != nil {
		return err
	}
	r.afterTable = false
	return nil
}

// paragraphBlock is one paragraph, or one list item when it sits inside a list.
func (r *renderer) paragraphBlock(node ast.Node, level int, numID string) error {
	images := collectImages(node, r.source)
	runs := inlineRuns(node, r.source)
	line := r.line(node)
	r.warnRawHTML(node, line)

	if len(images) > 0 && level == 0 {
		// The words first, then each picture on its own line. A picture in a
		// list item stays on the inline path: a figure inside a bullet would
		// break the numbering it sits in.
		if len(runs) > 0 {
			r.emit(r.paragraph(runs))
			r.counts.Paragraphs++
		}
		if err := r.emitImages(images, line); err != nil {
			return err
		}
		r.afterTable = false
		return nil
	}

	if level == 0 {
		r.emit(r.paragraph(runs))
		r.counts.Paragraphs++
	} else {
		r.emit(r.listItem(runs, numID, min(level-1, maxListLevel), numID == render.NumberNumID))
		r.counts.Paragraphs++
	}
	r.afterTable = false
	return nil
}

// quoteBlock indents what the quote holds, one step per level of quoting.
func (r *renderer) quoteBlock(quote *ast.Blockquote, level int, numID string) error {
	for node := quote.FirstChild(); node != nil; node = node.NextSibling() {
		paragraph, ok := node.(*ast.Paragraph)
		if !ok || level > 0 {
			if err := r.block(node, level, numID); err != nil {
				return err
			}
			continue
		}
		r.emit(r.quote(inlineRuns(paragraph, r.source), 1))
		r.counts.Paragraphs++
	}
	r.afterTable = false
	return nil
}

// emitImages places each picture on its own centred line, with its alt text as
// the caption.
func (r *renderer) emitImages(images []ImageRef, line int) error {
	for _, image := range images {
		paragraph, err := r.place(image.Target, line)
		if err != nil {
			return err
		}
		r.emit(paragraph)
		r.counts.Images++
		if alt := merge(image.Alt); len(alt) > 0 {
			r.emit(r.caption(alt))
		}
	}
	return nil
}

// warnRawHTML names a line whose inline HTML the house style does not carry.
//
// goldmark reads <angle brackets> as HTML, so a sentence about "<ampersands>"
// loses those words on the way in. Saying so is the same rule the block form
// follows: what was left out is named, never dropped in silence.
func (r *renderer) warnRawHTML(node ast.Node, line int) {
	if !hasRawHTML(node) {
		return
	}
	r.warn("line %d: inline HTML is not rendered in the house style and was left out", line)
}

// hasRawHTML says whether an inline subtree holds raw HTML, however deeply it
// is nested.
func hasRawHTML(node ast.Node) bool {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*ast.RawHTML); ok {
			return true
		}
		if hasRawHTML(child) {
			return true
		}
	}
	return false
}

// tableRows flattens a GFM table into runs. The header is row zero, which is
// what the builder expects.
func (r *renderer) tableRows(table *east.Table) [][][]Run {
	var rows [][][]Run
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		var cells [][]Run
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, inlineRuns(cell, r.source))
		}
		rows = append(rows, cells)
	}
	return rows
}
