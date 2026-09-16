// This file is the walk itself: the renderer's state, the block dispatch, the
// warnings each block raises, and the Result the caller reads. build.go holds
// the paragraph, heading, list item and caption builders, inline.go turns a
// paragraph's children into runs, table.go writes a pipe table, image.go embeds
// a picture, numbering.go builds a heading's number, mark.go is the
// ==highlight== extension, and doc.go holds the package comment.

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
// be rendered, the counts, and the files the walk read.
//
// Sources is every picture the note named as a file, resolved against the
// note's own directory, whether or not the walk placed it: a picture inside a
// bullet is left out of the document and is still a file the run read the note
// for. The caller is the one that knows where it is about to write, and a
// picture is an input the run must not replace. A path in here is what the note
// named, not a promise the file is there.
type Result struct {
	Blocks   []*etree.Element
	Media    []Media
	Warnings []string
	Counts   Counts
	Sources  []string
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
	sources  []string

	// pendingMark is the list marker the item being walked has not spent yet.
	// It is the renderer's rather than a field on listCtx because the block
	// that spends it can be a container or two down from the item.
	pendingMark bool

	// numberedLists counts the top-level ordered lists the walk has reached,
	// because they all name one w:num and the second one carries on from the
	// first. See warnListNumbers.
	numberedLists int

	relID      int
	linkIDs    map[string]string
	imageCount int
	docPr      int

	firstHeading bool
	afterTable   bool
	quoteDepth   int
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

	if err := r.walk(root, 0, listCtx{}); err != nil {
		return Result{}, err
	}
	if r.err != nil {
		return Result{}, r.err
	}
	return Result{Blocks: r.blocks, Media: r.media, Warnings: r.warnings,
		Counts: r.counts, Sources: r.sources}, nil
}

// parse is goldmark, configured to the extension set the previous generator
// asked pandoc for: pipe tables, ==mark==, strikeout and task lists, plus the
// smart punctuation pandoc's markdown format enables by default.
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
			// Footnotes are not in the house style, and the extension is
			// what makes that a warning rather than a silent publish: with
			// it off, "[^1]" and "[^1]: the detail" are ordinary text, so a
			// note read back out of a Google Doc, where `read` writes its
			// footnotes in exactly that shape, publishes the markers as
			// prose.
			extension.Footnote,
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
	return r.lineAt(offset)
}

// lineAt turns a source offset into the 1-based line holding it. An offset
// nothing carried is line 1, which is the only answer left.
func (r *renderer) lineAt(offset int) int {
	if offset < 0 || offset > len(r.source) {
		return 1
	}
	return 1 + bytes.Count(r.source[:offset], []byte("\n"))
}

// itemLine is the line a list item's own warning names. A ListItem carries no
// source position: its Offset is a column inside the line, and goldmark
// appends lines to leaf blocks only, so the item's line has to come from
// something inside it. An item holding a table resolves through the text in
// its cells, and one holding only a code block or a block of HTML resolves
// through neither Lines nor a Text child, which named line 1: the note's own
// front-matter delimiter, and the wrong end of the file to send an author to.
//
// An item holding nothing at all carries no position anywhere in it, so the
// nearest sibling item's line is the closest true answer there is. A line or
// two out still names the list.
func (r *renderer) itemLine(item ast.Node) int {
	if line, ok := r.descendantLine(item); ok {
		return line
	}
	for sibling := item.PreviousSibling(); sibling != nil; sibling = sibling.PreviousSibling() {
		if line, ok := r.descendantLine(sibling); ok {
			return line
		}
	}
	for sibling := item.NextSibling(); sibling != nil; sibling = sibling.NextSibling() {
		if line, ok := r.descendantLine(sibling); ok {
			return line
		}
	}
	return 1
}

// descendantLine is the line the node starts on, or the line of the first
// descendant that carries one. A fence is read the way the code block's own
// warning reads it, one line above the block's first line of code, because
// that is the line the author sees.
func (r *renderer) descendantLine(node ast.Node) (int, bool) {
	if lines := node.Lines(); lines != nil && lines.Len() > 0 {
		line := r.lineAt(lines.At(0).Start)
		if _, fenced := node.(*ast.FencedCodeBlock); fenced && line > 1 {
			line--
		}
		return line, true
	}
	if text, ok := node.(*ast.Text); ok {
		return r.lineAt(text.Segment.Start), true
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if line, ok := r.descendantLine(child); ok {
			return line, true
		}
	}
	return 0, false
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

// listCtx is what a block inherits from the list it sits in: the list in
// numbering.xml it belongs to, which is what the item's indent is read off.
//
// Whether a block carries the item's marker is not in here. The marker belongs
// to the first paragraph the item actually emits, and which block that is
// cannot be read off a node type from outside: it is the renderer's own
// pendingMark, which itemBlocks arms and paragraphBlock spends.
type listCtx struct {
	numID string
}

// walk renders a block subtree. level counts list nesting: 0 is the body, 1 is
// inside a list, and so on. list names the list a nested paragraph belongs to.
func (r *renderer) walk(parent ast.Node, level int, list listCtx) error {
	for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
		if err := r.block(node, level, list); err != nil {
			return err
		}
	}
	return nil
}

// itemBlocks renders one list item, and the item's marker goes on the first
// paragraph the item actually emits.
//
// goldmark gives a loose item one *ast.Paragraph per block, so numbering every
// one of them turns a two-paragraph item into two items: the author's "2."
// prints as "3.", and a bulleted list grows a bullet on every continuation
// paragraph. A continuation keeps the item's indent and takes no marker.
//
// The marker is pending rather than handed to a child chosen here, because
// neither the node type nor the position says which child emits the item's
// first paragraph. A block quote is a container whose paragraphs come back
// through block, so an item that is nothing but a quote has no *ast.Paragraph
// child at all: marking only those left that item with no number anywhere in
// it, and the author's "3." printed as "2.". Handing the marker to the item's
// first child instead breaks the mirror of that, an item opening with a fenced
// code block, which is a child that renders nothing and would spend the marker
// on a paragraph nobody sees. Pending, both take exactly one marker.
func (r *renderer) itemBlocks(item ast.Node, level int, numID string) error {
	// A nested list is walked from inside this loop, so the item's own marker
	// is put back on the way out: the sub-list's items arm and spend their own,
	// and an outer item whose first paragraph comes after the sub-list still
	// has one waiting.
	outer := r.pendingMark
	r.pendingMark = true
	defer func() { r.pendingMark = outer }()
	for node := item.FirstChild(); node != nil; node = node.NextSibling() {
		if err := r.block(node, level, listCtx{numID: numID}); err != nil {
			return err
		}
	}
	// The marker is still armed, so nothing this item holds emitted a list
	// paragraph and the item takes no marker.
	//
	// A table is the case that reached here in silence: it renders, at body
	// width rather than inside the item, so the document looks deliberate and
	// only the list is wrong. A code block warns about itself and says nothing
	// about the marker it cost. Both are named here instead, on the walker's
	// own rule that what did not reach the document in the shape the author
	// wrote is named on the envelope.
	//
	// What it costs depends on which list this is, so the two are not one
	// sentence. A number is a count Word carries on: the items after this one
	// print one lower and the author's "3." reads as "2.". A bullet is not,
	// so a bulleted list loses nothing but this item's own bullet and indent,
	// and telling the author to check numbering that is not wrong is the
	// cry-wolf warning this tool avoids everywhere else.
	if r.pendingMark {
		if numID == render.NumberNumID {
			r.warn("line %d: this list item has no text of its own, so it takes no number and the items after it are numbered one lower",
				r.itemLine(item))
		} else {
			r.warn("line %d: this list item has no text of its own, so it takes no bullet and no indent",
				r.itemLine(item))
		}
	}
	return nil
}

// warnListNumbers names a numbered list whose numbers are not the author's.
//
// Two shapes, and neither is a construct the walker declined to render: the
// list is there and its numbers are somebody else's. numbering.xml defines one
// w:num per list kind, so every ordered list in the body names the same one
// and a second top-level list carries on from the first: 1. and 2. print as
// 3. and 4. Every level of that part states w:start 1, so an author's "5."
// opens at 1 whatever depth it sits at.
//
// A nested list is not in the count. An absent w:lvlRestart restarts a level
// whenever the level above it moves, so the sub-lists under two items of one
// list each start again on their own, and warning about them would be the
// cry-wolf warning this tool avoids everywhere else.
//
// The structural fix is docs/backlog/one-numbered-list-per-document.md. The
// silence is not deferred with it: the prose around a numbered list
// cross-references the numbers the author wrote, so a document that prints
// others has to say so on the envelope.
func (r *renderer) warnListNumbers(list *ast.List, level int) {
	if level == 0 {
		r.numberedLists++
		if r.numberedLists > 1 {
			r.warn("line %d: this numbered list carries on from the one above it rather than starting again at 1",
				r.itemLine(list))
		}
	}
	if list.Start != 1 {
		r.warn("line %d: this numbered list starts at %d in the note and at 1 in the document",
			r.itemLine(list), list.Start)
	}
}

func (r *renderer) block(node ast.Node, level int, list listCtx) error {
	switch typed := node.(type) {
	case *ast.Heading:
		return r.headingBlock(typed)

	case *ast.Paragraph, *ast.TextBlock:
		return r.paragraphBlock(node, level, list)

	case *ast.List:
		// The two lists numbering.xml defines are the two a body may name. A
		// second numbered list therefore carries on from the first, which is
		// docs/backlog/one-numbered-list-per-document.md.
		id := render.BulletNumID
		if typed.IsOrdered() {
			id = render.NumberNumID
			r.warnListNumbers(typed, level)
		}
		if level == 0 {
			r.counts.Lists++
		}
		for item := typed.FirstChild(); item != nil; item = item.NextSibling() {
			if err := r.itemBlocks(item, level+1, id); err != nil {
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
		return r.quoteBlock(typed, level, list)

	case *ast.FencedCodeBlock:
		// Nail's decision: code blocks stay out of the house style. The note
		// says so on the envelope rather than losing the block in silence.
		//
		// The line named is the fence, which is what the author sees. A fenced
		// block's own lines start at its first line of code, one further down.
		// An empty fence carries no lines at all, so r.line falls back to 1
		// and the step back would name line 0, which is no line in any file.
		r.warn("line %d: a code block is not rendered in the house style and was left out",
			max(r.line(node)-1, 1))
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

	case *east.FootnoteList:
		// goldmark collects every footnote definition into one list at the
		// end of the document, whatever order the author wrote them in, so
		// each one is named by its own line rather than by the list's.
		for note := typed.FirstChild(); note != nil; note = note.NextSibling() {
			line := 1
			if at, ok := r.descendantLine(note); ok {
				line = at
			}
			r.warn("line %d: a footnote is not rendered in the house style and was left out",
				line)
		}
		r.afterTable = false
		return nil

	default:
		// A container this walker does not name is walked through rather than
		// dropped: losing a block silently is the failure that only shows up
		// once somebody reads the published document.
		if node.HasChildren() {
			return r.walk(node, level, list)
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

	prefix, skipped := r.numberer.prefix(heading.Level, plain)
	if prefix != "" {
		// Merged, so the number and the first word of the heading are one run.
		// Two runs carrying the same marks read the same on the page and make
		// the contents entry two pieces of text to match.
		runs = merge(append([]Run{{Text: prefix}}, runs...))
	}
	if skipped {
		// The number carries a zero, because it is built from every counter
		// down to this heading's own level and a level nothing reached is
		// still 0. Changing that number is Nail's decision, so the number
		// stays as it is and what the note gets is the line to look at.
		r.warn("line %d: this heading skips a level, so its number reads %q",
			r.line(heading), prefix)
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
func (r *renderer) paragraphBlock(node ast.Node, level int, list listCtx) error {
	images := collectImages(node, r.source)
	runs := inlineRuns(node, r.source)
	line := r.line(node)
	r.warnRawHTML(node, line)
	if level > 0 {
		// A figure inside a bullet would break the numbering it sits in, so
		// the picture stays out. The note says so rather than losing it in
		// silence, which is the rule every other unrendered block follows.
		r.warnImages(images, line, "a list item")
	}

	if len(images) > 0 && level == 0 {
		// The words first, then each picture on its own line.
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
		// The first paragraph an item emits spends the item's marker, and
		// every paragraph after it keeps the indent and takes none: numbering
		// a continuation turns one item into two, so the author's "2." prints
		// as "3.". numID is then empty and the ordered-list geometry is read
		// off the list rather than off the marker being written.
		numID := ""
		if r.pendingMark {
			numID = list.numID
			r.pendingMark = false
		}
		r.emit(r.listItem(runs, numID, min(level-1, maxListLevel),
			list.numID == render.NumberNumID))
		r.counts.Paragraphs++
	}
	r.afterTable = false
	return nil
}

// quoteBlock indents what the quote holds, one step per level of quoting.
func (r *renderer) quoteBlock(quote *ast.Blockquote, level int, list listCtx) error {
	// A quote inside a quote re-enters here, so the depth is the renderer's
	// rather than the caller's: "> > text" is two steps in, and reading it off
	// one parameter indented it exactly as far as one step.
	r.quoteDepth++
	defer func() { r.quoteDepth-- }()
	for node := quote.FirstChild(); node != nil; node = node.NextSibling() {
		paragraph, ok := node.(*ast.Paragraph)
		if !ok || level > 0 {
			if err := r.block(node, level, list); err != nil {
				return err
			}
			continue
		}
		line := r.line(paragraph)
		r.warnRawHTML(paragraph, line)
		r.warnImages(collectImages(paragraph, r.source), line, "a block quote")
		r.emit(r.quote(inlineRuns(paragraph, r.source), r.quoteDepth))
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

// warnImages names a line whose picture the house style does not place here.
//
// The house style puts a figure on its own centred line, which it cannot be
// inside a bullet, a quote or a table cell. Saying so is the same rule the
// block form follows: what was left out is named, never dropped in silence,
// and a picture that vanished is only found once somebody reads the published
// document.
// A picture left out here is still a file the note names, so it is recorded
// beside the ones that were embedded. Without that, `--out diagram.png --force`
// on a note whose only picture sits in a bullet passed the caller's check and
// wrote the document over the picture: the same loss the check exists for, and
// worse, because nothing had read those bytes into the document either.
func (r *renderer) warnImages(images []ImageRef, line int, where string) {
	if len(images) == 0 {
		return
	}
	for _, image := range images {
		if path := imagePath(image.Target, r.base); path != "" {
			r.sources = append(r.sources, path)
		}
	}
	r.warn("line %d: a picture inside %s is not rendered in the house style and was left out",
		line, where)
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
			line := r.line(cell)
			r.warnRawHTML(cell, line)
			r.warnImages(collectImages(cell, r.source), line, "a table cell")
			cells = append(cells, inlineRuns(cell, r.source))
		}
		rows = append(rows, cells)
	}
	return rows
}
