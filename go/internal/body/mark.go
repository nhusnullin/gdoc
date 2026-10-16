package body

// The marks: the ==highlight== syntax, the palette as the parser needs it,
// and the shapes that refuse the build.
//
// The highlight is the only way to say "a human still has to fill this in", so
// the house markdown depends on it and goldmark does not ship it. Written here
// rather than pulled in as a dependency: it is a delimiter pair, and the
// delimiter machinery goldmark exposes for strikethrough does all the work.
//
// The grey mark is not here. It is a comment, which comment.go reads and
// obsidian.go gives its second spelling, and the one thing left of the Pandoc
// span it used to be is the refusal in comment.go that names the comment to
// write instead.
//
// doc.go holds why a mark is refused where it is refused.

import (
	"fmt"
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

// comment is the mark a comment carries, by the delimiter it was written
// with. It is asked of the table rather than named here for the reason the
// highlight's colour is: the house style says which mark a spelling is, and
// this package says which spellings there are.
func (t markTable) comment(opener string) string { return t.bySyntax[opener] }

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

type markExtension struct{}

// Mark enables ==highlight== and the two spellings of a comment.
var Mark goldmark.Extender = &markExtension{}

func (e *markExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&highlightParser{}, 500),
		// The Obsidian comment runs where the highlight runs, after
		// goldmark's code span parser at 100: nothing inside backticks is a
		// comment.
		util.Prioritized(&obsidianInlineParser{}, 500),
	), parser.WithBlockParsers(
		// Where goldmark's own block of HTML runs, which is after every fence
		// and before the paragraph: a fence inside a code block opens nothing.
		util.Prioritized(&obsidianBlockParser{}, 900),
		// In front of goldmark's block of HTML at 900, because the line this
		// one claims is a line goldmark would take whole. Still behind every
		// fence, for the reason above.
		util.Prioritized(&commentLineParser{}, 850),
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
		if h, highlight := node.(*Highlight); highlight {
			name, _ = table.highlight(h, source)
		}
		if _, counted := counts[name]; counted {
			counts[name]++
		}
		return ast.WalkContinue, nil
	})
	return counts
}

// isMark says whether a node is one of the two shapes a mark is written in.
// The source is what the second needs: a comment is raw HTML, and only its
// words say whether it is a comment at all and whether it marks anything.
func isMark(node ast.Node, source []byte) bool {
	if _, highlight := node.(*Highlight); highlight {
		return true
	}
	return isPencil(node, source)
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
		if !entering || !isMark(node, r.source) {
			return ast.WalkContinue, nil
		}
		line := r.markLine(node)
		if h, highlight := node.(*Highlight); highlight {
			// The emoji is the instruction and not the words, so a highlight
			// holding nothing else marks no words: withoutOpener takes it off,
			// merge drops the empty run, and the author's own character is gone
			// from the document. countMarks counts the node either way, so the
			// envelope named a colour the document does not carry.
			if _, opener := r.marks.highlight(h, r.source); opener != "" &&
				strings.TrimSpace(withoutOpener(markWords(h, r.source), opener)) == "" {
				refusal = fmt.Errorf("line %d: the ==%s== mark holds its colour emoji and no words, so the mark would be a colour on nothing and the emoji would be dropped from the text", line, opener)
				return ast.WalkStop, nil
			}
		}
		for parent := node.Parent(); parent != nil; parent = parent.Parent() {
			if isMark(parent, r.source) {
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

// inlineLine is the line an inline node sits on. goldmark's Lines panics on an
// inline, so the first text inside it is where it starts, and a node holding
// no text at all resolves through the block around it.
//
// Raw HTML is asked first, because it is the one inline node that holds its
// own characters as segments rather than as text children: a pencil written
// as a comment has no ast.Text under it at all, so climbing to the parent
// named the first text anywhere above, which for "# <!-- tbd -->" is the
// first line of the note. TestAPencilsRefusalNamesItsOwnLine is the pin.
func inlineLine(source []byte, node ast.Node) int {
	for at := node; at != nil; at = at.Parent() {
		if raw, ok := at.(*ast.RawHTML); ok && raw.Segments.Len() > 0 {
			return lineOf(source, raw.Segments.At(0).Start)
		}
		if text := firstTextNode(at); text != nil {
			return lineOf(source, text.Segment.Start)
		}
	}
	return 1
}
