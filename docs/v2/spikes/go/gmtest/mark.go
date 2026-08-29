package main

// A ==highlight== inline parser, modelled on goldmark's own Strikethrough.
// This is the whole extension. Written to measure the real cost of the one
// pandoc feature goldmark lacks.

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type Mark struct{ ast.BaseInline }

var KindMark = ast.NewNodeKind("Mark")

func (n *Mark) Kind() ast.NodeKind                                    { return KindMark }
func (n *Mark) Dump(source []byte, level int)                         { ast.DumpHelper(n, source, level, nil, nil) }

type markParser struct{}

func (s markParser) Trigger() []byte { return []byte{'='} }

func (s markParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, defaultMarkDelimiterProcessor)
	if node == nil || node.OriginalLength > 2 || before == '=' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

type markDelimiterProcessor struct{}

func (p *markDelimiterProcessor) IsDelimiter(b byte) bool                { return b == '=' }
func (p *markDelimiterProcessor) CanOpenCloser(o, c *parser.Delimiter) bool { return o.Char == c.Char }
func (p *markDelimiterProcessor) OnMatch(consumes int) ast.Node          { return &Mark{} }

var defaultMarkDelimiterProcessor = &markDelimiterProcessor{}

type markExt struct{}

func (e *markExt) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(markParser{}, 500),
	))
}

var MarkExtension = &markExt{}
