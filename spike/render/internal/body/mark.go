package body

// The ==highlight== syntax, which pandoc calls the "mark" extension.
//
// It is the only way to say "a human still has to fill this in", so the house
// markdown depends on it and goldmark does not ship it. Written here rather than
// pulled in as a dependency: it is a delimiter pair, and the delimiter machinery
// goldmark exposes for strikethrough does all the work.

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

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

// Mark enables ==highlight==.
var Mark goldmark.Extender = &markExtension{}

func (e *markExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&highlightParser{}, 500),
	))
}
