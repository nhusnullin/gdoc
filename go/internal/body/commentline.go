package body

// A comment with the author's own words after it on one line.
//
// CommonMark opens a block of HTML on any line beginning "<!--" and shuts it
// on the first line holding "-->", and the whole of that line belongs to the
// block. So "<!-- todo --> We ship in May." was one block of HTML: named by
// line and left out whole, with the author's sentence never reaching the
// document. Obsidian shows that same line as the sentence with the comment
// hidden, which is the reading a pencil is built on, and internal/view writes
// exactly this shape whenever a paragraph opens with a grey run, so a document
// exported and published again lost the paragraph.
//
// The line is a paragraph here, which hands it to the inline parsers: the
// comment is read as the pencil it is and the rest as the prose it is. Two
// comments on one line are two pencils by the same route, where before the
// pair came home as one grey paragraph with both delimiters inside it as
// words.
//
// A comment alone on its line is not claimed. That one stays goldmark's block
// of HTML and the walk reads it as grey paragraphs, which is the second of a
// comment's two shapes. Wrapped over two lines, where the closer sits inside
// a block of HTML, the line is not claimed either: body.go leaves it out and
// names it as a comment with words after its closer rather than as a block of
// HTML. TestACommentWithProseAfterItIsAPencilAndThatProse and
// TestTwoCommentsOnOneLineAreTwoPencils are the pins over all three.
//
// A claimed line wraps the way any other paragraph's line wraps, which
// Continue explains and TestAPencilsLineWrapsLikeAnyOtherParagraph pins.

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// commentLineParser claims such a line as the paragraph it reads as.
type commentLineParser struct{}

func (p *commentLineParser) Trigger() []byte { return []byte{'<'} }

func (p *commentLineParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	if !commentThenWords(string(line)) {
		return nil, parser.NoChildren
	}
	node := ast.NewParagraph()
	node.Lines().Append(segment)
	reader.AdvanceToEOL()
	return node, parser.NoChildren
}

// Continue is goldmark's own paragraph continuation, because the node this
// opened is a paragraph and nothing else would hand the next line back.
//
// Shutting here on every line lost that next line whole. goldmark treats any
// open paragraph as continuable, so it skips Continue and offers the line to
// the parsers that may interrupt a paragraph; its own paragraph parser is not
// one of them, and this Continue then said Close. Nothing had opened, so the
// line was advanced past and reached no node at all: "<!-- todo --> We ship."
// with "Costs fell." under it published the first line and lost the second,
// with no warning naming it. A pencil's line wraps the way any other
// paragraph's line wraps, which is also the reading Obsidian shows.
// TestAPencilsLineWrapsLikeAnyOtherParagraph is the pin.
func (p *commentLineParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if util.IsBlank(line) {
		return parser.Close
	}
	node.Lines().Append(segment)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

// Close trims the paragraph's spaces, which is what goldmark's own paragraph
// parser does: the leading ones are the indent and the trailing ones are
// nothing the author typed for.
func (p *commentLineParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	lines := node.Lines()
	if lines.Len() == 0 {
		node.Parent().RemoveChild(node.Parent(), node)
		return
	}
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		lines.Set(i, line.TrimLeftSpace(reader.Source()))
	}
	last := lines.Len() - 1
	line := lines.At(last)
	lines.Set(last, line.TrimRightSpace(reader.Source()))
}

// CanInterruptParagraph is true because the block of HTML this stands in front
// of may interrupt one: an aside written straight under a line of prose is an
// aside, not the prose's own next line.
func (p *commentLineParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine is false, as it is for a block of HTML: four spaces in
// is a code block, and nothing inside one is markdown.
func (p *commentLineParser) CanAcceptIndentedLine() bool { return false }

// commentThenWords says whether this line opens with a comment that shuts on
// it and carries something other than space after the closer.
func commentThenWords(line string) bool {
	rest := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(rest, commentOpener) {
		return false
	}
	shut := strings.Index(rest[len(commentOpener):], commentCloser)
	if shut < 0 {
		return false
	}
	after := rest[len(commentOpener)+shut+len(commentCloser):]
	return strings.TrimSpace(after) != ""
}
