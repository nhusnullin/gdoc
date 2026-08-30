// Package body renders the Markdown body into the shell using the template's
// own idioms.
//
// goldmark is used only as a Markdown parser, never as a docx writer, and the
// OOXML is written here. That distinction is the same one the Python renderer
// draws about pandoc, and for the same reason: a general-purpose docx writer
// emits its own styles, which this Google-Docs-exported template does not
// define, and LibreOffice responds to a dangling w:pStyle by discarding the
// entire w:pPr around it. Every bullet and every number then silently
// disappears.
//
// Unlike pandoc, goldmark is a library. Nothing here runs an external program,
// so the whole renderer is one static binary.
package body

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"spike/gdocgo/internal/docx"
	"spike/gdocgo/internal/ooxml"
)

// Error is anything that should stop the build with a message rather than a stack.
type Error struct{ msg string }

func (e *Error) Error() string { return e.msg }

func errorf(format string, args ...any) error { return &Error{msg: fmt.Sprintf(format, args...)} }

// Options are the choices a caller gets to make.
type Options struct {
	// HeadingNumbering is "auto" or "none".
	HeadingNumbering string
	// BaseDir is what a relative image path is resolved against. Empty means
	// the document may hold no pictures.
	BaseDir string
	// Hyperlinks writes a real w:hyperlink for a Markdown link. The Python
	// renderer drops the destination and keeps only the text, so leaving this
	// off is what makes the two outputs comparable.
	Hyperlinks bool
}

// linkTable hands out one relationship per distinct destination.
type linkTable struct {
	pkg    *docx.Package
	byURL  map[string]string
	nextID int
}

func newLinkTable(pkg *docx.Package, media *Media) *linkTable {
	return &linkTable{pkg: pkg, byURL: map[string]string{}, nextID: media.relID}
}

func (t *linkTable) idFor(url string) string {
	if id, ok := t.byURL[url]; ok {
		return id
	}
	id := fmt.Sprintf("rId%d", t.nextID)
	t.nextID++
	t.byURL[url] = id

	rels, _ := t.pkg.Get("word/_rels/document.xml.rels")
	entry := fmt.Sprintf(
		`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="%s" TargetMode="External"/>`,
		id, escapeAttr(url))
	t.pkg.Set("word/_rels/document.xml.rels",
		[]byte(strings.Replace(string(rels), "</Relationships>", entry+"</Relationships>", 1)))
	return id
}

func escapeAttr(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(value)
}

// Renderer holds the state one document's render needs.
type Renderer struct {
	opts      Options
	media     *Media
	links     *linkTable
	numbering *ooxml.Numbering
	numberer  *headingNumberer
	source    []byte
	emit      func(*etree.Element)
	count     int

	firstHeading bool
	afterTable   bool
}

// markdown is the parser, configured to match the extension set the Python
// renderer asks pandoc for: pipe tables, ==mark==, strikeout and task lists,
// plus the smart punctuation pandoc's `markdown` format enables by default.
func markdown() goldmark.Markdown {
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
					extension.LeftSingleQuote:  []byte("\u2018"),
					extension.RightSingleQuote: []byte("\u2019"),
					extension.LeftDoubleQuote:  []byte("\u201c"),
					extension.RightDoubleQuote: []byte("\u201d"),
					extension.EnDash:           []byte("\u2013"),
					extension.EmDash:           []byte("\u2014"),
					extension.Ellipsis:         []byte("\u2026"),
					extension.LeftAngleQuote:   []byte("\u00ab"),
					extension.RightAngleQuote:  []byte("\u00bb"),
					extension.Apostrophe:       []byte("\u2019"),
				}),
			),
			Mark,
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
}

// Render splices the rendered body into the document, before its sectPr.
func Render(pkg *docx.Package, document *etree.Document, body *etree.Element,
	numberingDoc *etree.Document, markdownText string, opts Options) (int, error) {

	sectPr := body.SelectElement("w:sectPr")
	if sectPr == nil {
		return 0, errorf("the shell has no sectPr, so the page setup is missing")
	}
	numberingRoot := numberingDoc.SelectElement("w:numbering")
	if numberingRoot == nil {
		return 0, errorf("word/numbering.xml has no w:numbering root")
	}

	source := []byte(markdownText)
	root := markdown().Parser().Parse(text.NewReader(source))

	media := NewMedia(pkg)
	renderer := &Renderer{
		opts:         opts,
		media:        media,
		links:        newLinkTable(pkg, media),
		numbering:    ooxml.NewNumbering(numberingRoot),
		source:       source,
		firstHeading: true,
	}
	renderer.numberer = newHeadingNumberer(
		opts.HeadingNumbering == "auto", shallowestHeadingLevel(root, source))

	renderer.emit = func(element *etree.Element) {
		body.InsertChildAt(sectPr.Index(), element)
		renderer.count++
	}

	if err := renderer.walk(root, 0, 0); err != nil {
		return 0, err
	}
	return renderer.count, nil
}

// shallowestHeadingLevel finds the smallest header level anywhere in the
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
func (r *Renderer) walk(parent ast.Node, level, numID int) error {
	for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
		if err := r.block(node, level, numID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) block(node ast.Node, level, numID int) error {
	switch typed := node.(type) {
	case *ast.Heading:
		return r.heading(typed)

	case *ast.Paragraph, *ast.TextBlock:
		return r.paragraph(node, level, numID)

	case *ast.List:
		start := typed.Start
		if start == 0 {
			start = 1
		}
		newID := r.numbering.New(typed.IsOrdered(), start)
		for item := typed.FirstChild(); item != nil; item = item.NextSibling() {
			if err := r.walk(item, level+1, newID); err != nil {
				return err
			}
		}
		r.afterTable = false
		return nil

	case *east.Table:
		rows, err := r.tableRows(typed)
		if err != nil {
			return err
		}
		table, err := r.makeTable(rows)
		if err != nil {
			return err
		}
		r.emit(table)
		r.afterTable = true
		return nil

	case *ast.Blockquote:
		if err := r.walk(typed, level, numID); err != nil {
			return err
		}
		r.afterTable = false
		return nil

	case *ast.FencedCodeBlock, *ast.CodeBlock:
		r.emit(r.makeParagraph(
			[]Run{{Text: r.blockLines(node), Mono: true}},
			paragraphOpts{}))
		r.afterTable = false
		return nil

	case *ast.ThematicBreak:
		r.emit(r.makeParagraph(nil, paragraphOpts{justify: true}))
		r.afterTable = false
		return nil

	case *ast.HTMLBlock:
		r.afterTable = false
		return nil

	default:
		// A container this renderer does not name is walked through rather than
		// dropped: losing a block silently is the failure mode that only shows
		// up once somebody reads the published document.
		if node.HasChildren() {
			return r.walk(node, level, numID)
		}
		r.afterTable = false
		return nil
	}
}

func (r *Renderer) heading(heading *ast.Heading) error {
	// Drive exports a picture that sits on its own line as a heading,
	// "# ![][image1]". The inline path renders no picture at all, so it used to
	// vanish, leaving an empty heading that still took a number and an empty
	// line in the contents list.
	images := collectImages(heading, r.source)
	runs := inlineRuns(heading, r.source)
	plain := plainText(runs)

	if len(images) > 0 && strings.TrimSpace(plain) == "" {
		// Nothing but a picture. It is a figure, not a heading, so it gets no
		// number and no contents entry.
		if err := r.emitImages(images); err != nil {
			return err
		}
		r.afterTable = false
		return nil
	}

	if prefix := r.numberer.prefix(heading.Level, plain); prefix != "" {
		runs = append([]Run{{Text: prefix}}, runs...)
	}
	// The break goes on the first heading whatever its level, so the body always
	// starts on a clean page after the contents list. Keyed on level 1 alone it
	// silently does nothing for a document whose top heading is "##".
	pageBreak := r.firstHeading
	r.firstHeading = false

	r.emit(r.makeHeading(r.numberer.styleLevel(heading.Level), runs, pageBreak))

	// A heading that names a picture keeps its words and gets the picture
	// underneath it, where it reads.
	if err := r.emitImages(images); err != nil {
		return err
	}
	r.afterTable = false
	return nil
}

func (r *Renderer) paragraph(node ast.Node, level, numID int) error {
	images := collectImages(node, r.source)
	runs := inlineRuns(node, r.source)

	if len(images) > 0 && level == 0 {
		// The words first, then each picture on its own line. A picture in a
		// list item stays on the inline path: a figure inside a bullet would
		// break the numbering it sits in.
		if len(runs) > 0 {
			r.emit(r.makeParagraph(runs, paragraphOpts{justify: true, before: r.beforeSpacing()}))
		}
		if err := r.emitImages(images); err != nil {
			return err
		}
		r.afterTable = false
		return nil
	}

	if level == 0 {
		// A table carries no space beneath it, so the next paragraph would
		// otherwise sit hard against its bottom border.
		r.emit(r.makeParagraph(runs, paragraphOpts{justify: true, before: r.beforeSpacing()}))
	} else {
		r.emit(r.makeListItem(runs, numID, min(level-1, ooxml.MaxListLevel)))
	}
	r.afterTable = false
	return nil
}

func (r *Renderer) beforeSpacing() string {
	if r.afterTable {
		return "240"
	}
	return "0"
}

// emitImages places each picture on its own centred line, with its alt as caption.
func (r *Renderer) emitImages(images []ImageRef) error {
	if len(images) == 0 {
		return nil
	}
	if r.opts.BaseDir == "" {
		return errorf("the document has an image (%s) but the renderer was given "+
			"no base directory to resolve it against", images[0].Target)
	}
	for _, image := range images {
		paragraph, err := r.media.Place(image.Target, r.opts.BaseDir)
		if err != nil {
			return err
		}
		r.emit(paragraph)
		if alt := merge(image.Alt); len(alt) > 0 {
			r.emit(r.makeCaption(alt))
		}
	}
	return nil
}

// tableRows extracts cell runs from a GFM table. The header is row zero, which
// is what the builder expects.
func (r *Renderer) tableRows(table *east.Table) ([][][]Run, error) {
	var rows [][][]Run
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		var cells [][]Run
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, inlineRuns(cell, r.source))
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

// blockLines reads the literal text of a code block, which goldmark holds as
// segments of the source rather than on the node.
func (r *Renderer) blockLines(node ast.Node) string {
	var b strings.Builder
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		b.Write(segment.Value(r.source))
	}
	return strings.TrimRight(b.String(), "\n")
}
