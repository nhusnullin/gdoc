package body

// The marks: the ==highlight== syntax, the one Pandoc span gdoc reads, the
// palette as the parser needs it, and the shapes that refuse the build.
//
// The highlight is the only way to say "a human still has to fill this in", so
// the house markdown depends on it and goldmark does not ship it. Written here
// rather than pulled in as a dependency: it is a delimiter pair, and the
// delimiter machinery goldmark exposes for strikethrough does all the work.
//
// The span is Pandoc's grammar, "[" content "]" immediately followed by "{"
// attributes "}", and goldmark ships no extension for it either. It cannot be
// a delimiter pair, because its two ends are not the same characters, so it is
// written the way goldmark's own link parser is written: one parser claims the
// bracket and leaves a node standing where it was, and the other finds that
// node again when the attribute arrives and moves the words parsed since into
// a span of their own.
//
// doc.go holds why a mark is refused where it is refused.

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"gdoc/internal/house"
)

// plainHighlight is the syntax a bare ==highlight== is written with. The house
// style gives it to one of the seven marks, and that mark is what a highlight
// with no colour emoji on it carries.
const plainHighlight = "=="

// pencilAttribute is the one attribute an author may write on a span. It is a
// constant here and not a value read off the house style because the parsers
// are built before any house file is read; the house file states the same
// string as the pencil mark's own syntax, and
// TestThePencilSyntaxIsTheHouseStyles pins the two together.
const pencilAttribute = ".pencil"

// spanCloser is the whole of what shuts a span: the bracket, the attribute and
// the brace, with nothing allowed between them.
const spanCloser = "]{" + pencilAttribute + "}"

// markTable is the house palette as the parser needs it: every syntax an
// author may write to the mark it names, which mark a bare ==highlight== is,
// and which marks are counted on the envelope.
//
// Its zero value names no mark at all, which is what a caller asking what a
// heading reads as or what an image's alt text says wants: neither reaches a
// run that could carry a colour.
type markTable struct {
	bySyntax map[string]string
	plain    string
	debts    []string
}

// newMarkTable reads the palette off the house style. No emoji and no hex is a
// constant in this package, for the reason doc.go gives.
func newMarkTable(cfg *house.Config) markTable {
	table := markTable{bySyntax: make(map[string]string, len(cfg.Marks))}
	for name, mark := range cfg.Marks {
		for _, syntax := range mark.Syntax {
			table.bySyntax[syntax] = name
		}
		if mark.Debt {
			table.debts = append(table.debts, name)
		}
	}
	sort.Strings(table.debts)
	table.plain = table.bySyntax[plainHighlight]
	return table
}

// highlight is the mark a ==highlight== carries and the opener to take off its
// words.
//
// A colour emoji at the start of the highlight is Obsidian's own syntax for
// its colour, so it is an instruction rather than words and does not reach the
// page. Anywhere else in the highlight it is an emoji somebody meant to print,
// which is why only the first rune is looked at.
func (t markTable) highlight(node ast.Node, source []byte) (name, opener string) {
	first, ok := node.FirstChild().(*ast.Text)
	if !ok {
		return t.plain, ""
	}
	words := first.Segment.Value(source)
	r, size := utf8.DecodeRune(words)
	if r == utf8.RuneError {
		return t.plain, ""
	}
	if name, found := t.bySyntax[string(words[:size])]; found {
		return name, string(words[:size])
	}
	return t.plain, ""
}

// span is the mark a [words]{.pencil} span carries.
func (t markTable) span(node *Span) string { return t.bySyntax[node.Syntax] }

// withoutOpener is the marked words with the colour emoji taken off the front,
// and the one space an author writes after it. The emoji says which colour;
// the space separates it from the words, and a highlight opening with a shaded
// space is not what the author typed.
func withoutOpener(words, opener string) string {
	if opener == "" {
		return words
	}
	return strings.TrimPrefix(strings.TrimPrefix(words, opener), " ")
}

// KindHighlight identifies a ==marked== span.
var KindHighlight = ast.NewNodeKind("Highlight")

type Highlight struct{ ast.BaseInline }

func (n *Highlight) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }
func (n *Highlight) Kind() ast.NodeKind            { return KindHighlight }

type highlightParser struct{}

func (p *highlightParser) Trigger() []byte { return []byte{'='} }

func (p *highlightParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, defaultHighlightProcessor)
	if node == nil {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

type highlightDelimiterProcessor struct{}

func (p *highlightDelimiterProcessor) IsDelimiter(b byte) bool { return b == '=' }

func (p *highlightDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *highlightDelimiterProcessor) OnMatch(consumes int) ast.Node { return &Highlight{} }

var defaultHighlightProcessor = &highlightDelimiterProcessor{}

// KindSpan identifies a [words]{.pencil} span.
var KindSpan = ast.NewNodeKind("Span")

// Span is one bracketed span and the attribute it carries, as the author wrote
// it. This package reads exactly one attribute, so Syntax is a record of what
// was matched rather than a choice the walk makes; it is named the way the
// house style names it, because that is what the mark is looked up by.
//
// It is not called Attribute because ast.Node has a method of that name.
type Span struct {
	ast.BaseInline
	Syntax string
}

func (n *Span) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Syntax": n.Syntax}, nil)
}

func (n *Span) Kind() ast.NodeKind { return KindSpan }

// kindSpanOpen is the bracket a span opened with, standing in the inline tree
// until the attribute arrives and the span closes over the words between them.
//
// It is a node kind of its own rather than a text node carrying "[" because
// goldmark merges each new text segment into the last text child it finds: a
// text node here would swallow the span's own words, and the close would find
// nothing to move.
var kindSpanOpen = ast.NewNodeKind("SpanOpen")

type spanOpen struct {
	ast.BaseInline
	Segment text.Segment
}

func (n *spanOpen) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }
func (n *spanOpen) Kind() ast.NodeKind            { return kindSpanOpen }
func (n *spanOpen) Text(source []byte) []byte     { return n.Segment.Value(source) }

// spanStateKey holds the spans opened and not yet closed, innermost last.
var spanStateKey = parser.NewContextKey()

// spanState is one open span: the node standing where its bracket was, and the
// delimiter the words after it opened above, which is what emphasis inside the
// span is resolved against before those words are moved.
type spanState struct {
	open   *spanOpen
	bottom ast.Node
}

func pushSpan(pc parser.Context, state *spanState) {
	open, _ := pc.Get(spanStateKey).([]*spanState)
	pc.Set(spanStateKey, append(open, state))
}

func lastSpan(pc parser.Context) *spanState {
	open, _ := pc.Get(spanStateKey).([]*spanState)
	if len(open) == 0 {
		return nil
	}
	return open[len(open)-1]
}

func popSpan(pc parser.Context) {
	open, _ := pc.Get(spanStateKey).([]*spanState)
	if len(open) == 0 {
		return
	}
	pc.Set(spanStateKey, open[:len(open)-1])
}

// spanAhead says whether the reader sits on a bracket that opens a span this
// package reads: words with no bracket of their own, then the closer, all on
// this line.
//
// A bracket this says no to is left to goldmark's link parser, which runs
// after this one, so [words](url) is still a link and [an aside] is still the
// author's own words.
func spanAhead(line []byte) bool {
	for i := 1; i < len(line); i++ {
		switch line[i] {
		case '[', '\n':
			return false
		case ']':
			return i > 1 && bytes.HasPrefix(line[i:], []byte(spanCloser))
		}
	}
	return false
}

type spanOpenParser struct{}

func (p *spanOpenParser) Trigger() []byte { return []byte{'['} }

func (p *spanOpenParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()
	if len(line) == 0 || line[0] != '[' || !spanAhead(line) {
		return nil
	}
	open := &spanOpen{Segment: segment.WithStop(segment.Start + 1)}
	pushSpan(pc, &spanState{open: open, bottom: pc.LastDelimiter()})
	block.Advance(1)
	return open
}

type spanCloseParser struct{}

func (p *spanCloseParser) Trigger() []byte { return []byte{']'} }

func (p *spanCloseParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if !bytes.HasPrefix(line, []byte(spanCloser)) {
		return nil
	}
	// A state whose bracket is not in this paragraph belongs to a span that
	// can no longer close, because the words it would cover have been written
	// already. It is dropped rather than closed over somebody else's words.
	state := lastSpan(pc)
	for state != nil && state.open.Parent() != parent {
		popSpan(pc)
		state = lastSpan(pc)
	}
	if state == nil {
		return nil
	}
	popSpan(pc)
	block.Advance(len(spanCloser))
	// Emphasis inside the span is resolved here, while its delimiters are
	// still siblings of the words they cover. Left to the end of the
	// paragraph, the opener would be inside the span and the closer outside
	// it. goldmark's own link parser resolves a link's label the same way.
	parser.ProcessDelimiters(state.bottom, pc)
	span := &Span{Syntax: pencilAttribute}
	for child := state.open.NextSibling(); child != nil; {
		next := child.NextSibling()
		parent.RemoveChild(parent, child)
		span.AppendChild(span, child)
		child = next
	}
	parent.RemoveChild(parent, state.open)
	return span
}

type markExtension struct{}

// Mark enables ==highlight== and [words]{.pencil}.
var Mark goldmark.Extender = &markExtension{}

func (e *markExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&highlightParser{}, 500),
		// Both halves of the span run before goldmark's link parser, which is
		// at 200 and claims every bracket it is offered: a parser after it
		// would never be asked.
		util.Prioritized(&spanOpenParser{}, 150),
		util.Prioritized(&spanCloseParser{}, 150),
	))
}

// countMarks is how many of each debt mark the note holds, counted once per
// mark the author wrote rather than once per run: a highlight holding a bold
// word is several runs and one mark.
//
// A debt with none is counted as zero, because a note somebody has been
// through is not the same fact as a note nobody has read. The four marks that
// are not a debt are not counted at all: whether a blue aside matters is the
// reader's to say, and this package counts rather than judges.
func countMarks(root ast.Node, source []byte, table markTable) map[string]int {
	counts := make(map[string]int, len(table.debts))
	for _, name := range table.debts {
		counts[name] = 0
	}
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		name := ""
		switch typed := node.(type) {
		case *Highlight:
			name, _ = table.highlight(typed, source)
		case *Span:
			name = table.span(typed)
		}
		if _, counted := counts[name]; counted {
			counts[name]++
		}
		return ast.WalkContinue, nil
	})
	return counts
}

// isMark says whether a node is one of the two shapes a mark is written in.
func isMark(node ast.Node) bool {
	switch node.(type) {
	case *Highlight, *Span:
		return true
	}
	return false
}

// checkMarks refuses a mark inside a mark and a mark inside a heading.
//
// Both are read off the tree rather than off the source lines, because both
// are questions about what a mark sits inside and goldmark answers those: a
// source scan would have to decide for itself whether "a == b == c" is a
// highlight, and the flanking rules that say it is not are goldmark's.
func (r *renderer) checkMarks(root ast.Node) error {
	var refusal error
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || !isMark(node) {
			return ast.WalkContinue, nil
		}
		line := r.markLine(node)
		for parent := node.Parent(); parent != nil; parent = parent.Parent() {
			if isMark(parent) {
				refusal = fmt.Errorf("line %d: a mark inside a mark carries two colours on one stretch of words and Word carries one, so the house style has none; the inner mark's own words are what gdoc would have to drop", line)
				return ast.WalkStop, nil
			}
			if _, heading := parent.(*ast.Heading); heading {
				refusal = fmt.Errorf("line %d: a mark inside a heading is refused, because Word copies a heading into the contents field and the mark would be in the contents list as well as on the page", line)
				return ast.WalkStop, nil
			}
		}
		return ast.WalkContinue, nil
	})
	return refusal
}

// markLine is the line a mark's refusal names. descendantLine cannot be asked
// for it: a mark is an inline node, and goldmark's Lines panics on one. The
// first text inside the mark is where it starts, and a mark holding no text at
// all resolves through the block around it.
func (r *renderer) markLine(node ast.Node) int {
	for at := node; at != nil; at = at.Parent() {
		if text := firstTextNode(at); text != nil {
			return r.lineAt(text.Segment.Start)
		}
	}
	return 1
}

// spanWritten is one whole span as an author writes it: bracketed words
// carrying no bracket of their own, then the attribute list.
var spanWritten = regexp.MustCompile(`\[([^\[\]]*)\]\{([^{}]*)\}`)

// spanClosed is the end of a span alone, which is what is left on a line whose
// span does not open on it.
var spanClosed = regexp.MustCompile(`\]\{\.[^{}]*\}`)

// checkSpans is the question the source lines are asked before the note is
// parsed: does every span carry the one attribute gdoc reads, does every span
// mark some words, and does every span open and close on one line.
//
// It is the source rather than the tree because goldmark is silent in all
// three shapes: a span it does not recognise is prose, and prose carrying
// braces is a document with the author's syntax printed in it. A fenced code
// block is skipped, and so is a code span, for the reason checkFootnotes skips
// both: nothing inside backticks is markdown.
func checkSpans(source []byte) error {
	inFence := false
	for i, line := range strings.Split(string(source), "\n") {
		if fence.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		at := i + 1
		line = withoutCode(line)
		rest := line
		for _, m := range spanWritten.FindAllStringSubmatch(line, -1) {
			whole, words, attribute := m[0], m[1], m[2]
			if attribute != pencilAttribute {
				return fmt.Errorf("line %d: the span %s carries {%s}, and %s is the only attribute gdoc reads; written as it is, the brackets and the braces would reach the document as words", at, whole, attribute, pencilAttribute)
			}
			if strings.TrimSpace(words) == "" {
				return fmt.Errorf("line %d: the span %s marks no words, so the mark would be a colour on nothing", at, whole)
			}
			rest = strings.Replace(rest, whole, "", 1)
		}
		if closer := spanClosed.FindString(rest); closer != "" {
			return fmt.Errorf("line %d: %s closes a span this line does not open; a span marks words on one line, and one that crosses a line break or a paragraph would reach the document as the syntax the author wrote", at, closer)
		}
	}
	return nil
}
