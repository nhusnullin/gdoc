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
//
// A code span is stepped over, because nothing inside backticks is markdown
// and checkSpans reads the line the same way: a bracket in a code example
// would otherwise stop the scan here, leave the span unopened, and print the
// author's own syntax into the document with checkSpans seeing nothing wrong.
// TestASpanWhoseWordsHoldACodeSpanWithABracketIsStillASpan is the pin.
func spanAhead(line []byte) bool {
	for i := 1; i < len(line); i++ {
		switch line[i] {
		case '`':
			if shut := codeSpanEnd(line, i); shut > 0 {
				i = shut - 1
			}
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

// checkMarks refuses a mark inside a mark, a mark inside a heading, a mark
// inside a picture's alt text, and a highlight whose only content is the colour
// emoji that opened it.
//
// All four are read off the tree rather than off the source lines, because each
// is a question about what a mark sits inside or holds and goldmark answers
// those: a source scan would have to decide for itself whether "a == b == c" is
// a highlight, and the flanking rules that say it is not are goldmark's.
func (r *renderer) checkMarks(root ast.Node) error {
	var refusal error
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || !isMark(node) {
			return ast.WalkContinue, nil
		}
		line := r.markLine(node)
		if h, highlight := node.(*Highlight); highlight {
			// The emoji is the instruction and not the words, so a highlight
			// holding nothing else marks no words: withoutOpener takes it off,
			// merge drops the empty run, and the author's own character is gone
			// from the document. countMarks counts the node either way, so the
			// envelope named a colour the document does not carry. An empty
			// span is refused for the same reason.
			if _, opener := r.marks.highlight(h, r.source); opener != "" &&
				strings.TrimSpace(withoutOpener(markWords(h, r.source), opener)) == "" {
				refusal = fmt.Errorf("line %d: the ==%s== mark holds its colour emoji and no words, so the mark would be a colour on nothing and the emoji would be dropped from the text", line, opener)
				return ast.WalkStop, nil
			}
		}
		for parent := node.Parent(); parent != nil; parent = parent.Parent() {
			if isMark(parent) {
				refusal = fmt.Errorf("line %d: a mark inside a mark carries two colours on one stretch of words and Word carries one, so the house style has none; the inner mark's own words are what gdoc would have to drop", line)
				return ast.WalkStop, nil
			}
			if _, heading := parent.(*ast.Heading); heading {
				refusal = fmt.Errorf("line %d: a mark inside a heading is refused, because Word copies a heading into the contents field and the mark would be in the contents list as well as on the page", line)
				return ast.WalkStop, nil
			}
			if _, picture := parent.(*ast.Image); picture {
				// An alt text becomes the picture's caption, and the caption
				// carries no colour: markTable's zero value is what the walk
				// reads it with. Left standing, the colour emoji that opens a
				// highlight reached the caption as words and the envelope
				// counted a mark the document does not carry.
				refusal = fmt.Errorf("line %d: a mark inside a picture's alt text is refused, because the alt text becomes the picture's caption and a caption carries no colour, so the mark would be lost and the syntax for it would reach the document as words", line)
				return ast.WalkStop, nil
			}
		}
		return ast.WalkContinue, nil
	})
	return refusal
}

// markWords is every character of the author's own text inside a mark,
// descendants included. nodeText reads a node's direct children only, which for
// this question would call "==\U0001F7E2 **go**==" a mark holding no words.
func markWords(node ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch typed := n.(type) {
		case *ast.Text:
			b.Write(typed.Segment.Value(source))
		case *ast.String:
			b.Write(typed.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// markLine is the line a mark's refusal names. descendantLine cannot be asked
// for it: a mark is an inline node, and inlineLine holds why and what the
// answer is instead.
func (r *renderer) markLine(node ast.Node) int { return inlineLine(r.source, node) }

// spanWritten is one whole span as an author writes it: bracketed words
// carrying no bracket of their own, then the attribute list.
var spanWritten = regexp.MustCompile(`\[([^\[\]]*)\]\{([^{}]*)\}`)

// spanClosed is the end of a span alone, which is what is left on a line whose
// span does not open on it.
var spanClosed = regexp.MustCompile(`\]\{\.[^{}]*\}`)

// spanAttribute is the attribute list a span is closed with, which is what a
// line carries once for every span written on it.
const spanAttribute = "{" + pencilAttribute + "}"

// checkSpans is the question asked of the note before a block is built: does
// every span carry the one attribute gdoc reads, does every span mark some
// words, does every span open and close on one line, and did the parse form a
// span wherever the attribute was written.
//
// The first three are read off the source lines because goldmark is silent in
// all three shapes: a span it does not recognise is prose, and prose carrying
// braces is a document with the author's syntax printed in it. scan.go holds
// which lines are read, and it is the same set checkFootnotes reads: nothing
// inside backticks, nothing inside a code block and nothing inside a block of
// HTML is markdown.
//
// Which is why the words question is asked of a line that lost no code. The
// lines come with their code spans taken out, so a span whose whole content is
// one, "[`code`]{.pencil}", reads here as "[]{.pencil}" and was refused for
// marking no words, naming a span nowhere in the file. The parser forms that
// span, because spanAhead steps over the code, and the words reach the
// document grey. An empty span on a line that held code is still refused, by
// the attribute count below: the parse forms no span for a bracket with
// nothing in it, so the attribute closes none.
// TestASpanWhoseWordsAreOnlyACodeSpanIsStillASpan and
// TestAnEmptySpanBesideACodeSpanIsStillRefused are the pins.
//
// The fourth is the parse's own answer, because a bracket this package's
// parser never gets to claim still reads as a span by line. Three of
// goldmark's parsers claim one first: a bracket after "!" is a picture's, a
// bracket holding a caret is a footnote mark's, and a bracket followed by a
// target or a reference is a link's. In each of those the attribute closes
// nothing and "{.pencil}" was published as words with the build saying ok, so
// the attribute is counted against the spans the parse formed on that line.
// TestAPencilAttributeThatOpensNoSpanIsRefusedByLine and
// TestASpanTheParseFormsIsNotRefused are the pins.
func checkSpans(source []byte, root ast.Node) error {
	var refusal error
	formed := spanLines(source, root)
	checkLines(source, root, func(l sourceLine) {
		if refusal != nil {
			return
		}
		rest := l.text
		for _, m := range spanWritten.FindAllStringSubmatch(l.text, -1) {
			whole, words, attribute := m[0], m[1], m[2]
			if attribute != pencilAttribute {
				refusal = fmt.Errorf("line %d: the span %s carries {%s}, and %s is the only attribute gdoc reads; written as it is, the brackets and the braces would reach the document as words", l.at, whole, attribute, pencilAttribute)
				return
			}
			if strings.TrimSpace(words) == "" && !l.code {
				refusal = fmt.Errorf("line %d: the span %s marks no words, so the mark would be a colour on nothing", l.at, whole)
				return
			}
			rest = strings.Replace(rest, whole, "", 1)
		}
		if closer := spanClosed.FindString(rest); closer != "" {
			refusal = fmt.Errorf("line %d: %s closes a span this line does not open; a span marks words on one line, and one that does not open and close on the same line would reach the document as the syntax the author wrote", l.at, closer)
			return
		}
		if written := strings.Count(l.text, spanAttribute); written > formed[l.at] {
			refusal = fmt.Errorf("line %d: %s on this line closes no span gdoc reads, because the bracket in front of it opens a picture, a link or a footnote mark rather than a span; written as it is, the attribute would reach the document as words", l.at, spanAttribute)
		}
	})
	return refusal
}

// spanLines is how many spans the parse formed, by the line each sits on.
//
// A span opens and closes on one line, so the line a span's own words start on
// is the line its attribute was written on. A span holding no text at all
// resolves through the block around it, the way a mark's refusal does.
func spanLines(source []byte, root ast.Node) map[int]int {
	out := map[int]int{}
	if root == nil {
		return out
	}
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if _, ok := node.(*Span); ok {
			out[inlineLine(source, node)]++
		}
		return ast.WalkContinue, nil
	})
	return out
}

// inlineLine is the line an inline node sits on. goldmark's Lines panics on an
// inline, so the first text inside it is where it starts, and a node holding
// no text at all resolves through the block around it.
func inlineLine(source []byte, node ast.Node) int {
	for at := node; at != nil; at = at.Parent() {
		if text := firstTextNode(at); text != nil {
			return lineOf(source, text.Segment.Start)
		}
	}
	return 1
}
