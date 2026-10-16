package body

// Obsidian's own spelling of a comment, %% words %%, which goldmark has no
// node for.
//
// Two parsers, one meaning. Both produce goldmark's own nodes, an ast.RawHTML
// inside a sentence and an ast.HTMLBlock on lines of its own, with the
// delimiters left inside the segments they hand over. So comment.go reads one
// shape, the walk has one arm per shape rather than two, and every rule a
// comment already has, what it may hold, where it is refused, which lines a
// refusal is read off, holds for this spelling without being written again.
// Both node types carry their text as exported segments, so nothing here
// reaches into goldmark's unexported state.
//
// A fence that never closes runs to the end of the note, which is what
// goldmark does with an unclosed HTML comment. Either way what is left out is
// named by line, as the comment it is, and comment.go's unclosedComment is
// where the two spellings are asked:
// TestAnUnterminatedObsidianFenceIsNamedAndLeftOut and
// TestAnUnclosedHTMLCommentIsNamedAsAComment are the pins.

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// obsidianInlineParser claims %% words %% inside a sentence.
//
// It runs where the highlight runs, after goldmark's code span parser, so
// nothing inside backticks is read as a comment: a note explaining the syntax
// keeps it. TestAnObsidianCommentInCodeIsCode is the pin.
type obsidianInlineParser struct{}

func (p *obsidianInlineParser) Trigger() []byte { return []byte{'%'} }

// Parse walks forward to the closer the way goldmark's own comment parser
// does, over the lines of the paragraph in hand and no further, and puts the
// position back where it found it when there is no closer to reach.
//
// The closer is looked for past the end of the opening line because an author
// writes one aside and their editor wraps it. Shutting on the opening line
// instead left the pair unclaimed, which published "%% an aside that" and
// "wraps over two lines %%" as the words the author typed, delimiters and all,
// with the build saying ok: the one failure every line check in this package
// exists to prevent, in the one spelling Obsidian hides. The HTML spelling has
// always reached across the wrap, so this is the rule that spelling already
// has rather than a new one. Two unrelated pairs of percent signs grey what
// stands between them, which is what they already did on one line and what
// export names as a loss on the way home.
// TestAWrappedObsidianCommentIsOnePencil is the pin.
func (p *obsidianInlineParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()
	delimiter := []byte(obsidianDelimiter)
	if !bytes.HasPrefix(line, delimiter) {
		return nil
	}
	savedLine, savedSegment := block.Position()
	node := &ast.RawHTML{Segments: text.NewSegments()}
	for offset := len(delimiter); ; offset = 0 {
		if shut := bytes.Index(line[offset:], delimiter); shut >= 0 {
			length := offset + shut + len(delimiter)
			node.Segments.Append(segment.WithStop(segment.Start + length))
			block.Advance(length)
			return node
		}
		node.Segments.Append(segment)
		block.AdvanceLine()
		if line, segment = block.PeekLine(); line == nil {
			block.SetPosition(savedLine, savedSegment)
			return nil
		}
	}
}

// obsidianBlockParser claims a %% alone on a line, and everything to the next
// %% alone on a line, as one comment standing on lines of its own.
//
// Only the fence form is claimed here. A whole comment written on one line of
// its own is a paragraph holding one comment, which the inline parser reads
// and which reaches the document as the same single grey paragraph:
// TestAnObsidianCommentOnOneLineOfItsOwnIsOneGreyParagraph is the pin.
type obsidianBlockParser struct{}

func (p *obsidianBlockParser) Trigger() []byte { return []byte{'%'} }

func (p *obsidianBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	if !isObsidianFence(line) {
		return nil, parser.NoChildren
	}
	// HTMLBlockType2 is goldmark's own name for a comment, which is what this
	// is: the walk asks blockComment what the block holds rather than asking
	// the type, and the type is what Dump prints.
	node := ast.NewHTMLBlock(ast.HTMLBlockType2)
	reader.AdvanceToEOL()
	node.Lines().Append(segment)
	return node, parser.NoChildren
}

func (p *obsidianBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	block := node.(*ast.HTMLBlock)
	line, segment := reader.PeekLine()
	if isObsidianFence(line) {
		block.ClosureLine = segment
		reader.AdvanceToEOL()
		return parser.Close
	}
	block.Lines().Append(segment)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (p *obsidianBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

// CanInterruptParagraph is true because a comment is one of the blocks
// CommonMark lets open in the middle of a paragraph and goldmark's own comment
// does: an aside written straight under a line of prose is an aside, not the
// prose's own next line. TestAnObsidianFenceInterruptsAParagraph is the pin.
func (p *obsidianBlockParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine is false, as it is for a block of HTML: four spaces in
// is a code block, and nothing inside one is markdown.
func (p *obsidianBlockParser) CanAcceptIndentedLine() bool { return false }

// isObsidianFence says whether a line is the delimiter and nothing else.
func isObsidianFence(line []byte) bool {
	return strings.TrimSpace(string(line)) == obsidianDelimiter
}

// obsidianFenceBlock says whether a block of HTML is one this file's own
// parser opened, which is the question "was it written with %%": a block
// opens here on a fence and on nothing else. Whether that fence closed is
// unclosedComment's.
func obsidianFenceBlock(node *ast.HTMLBlock, source []byte) bool {
	lines := node.Lines()
	if lines.Len() == 0 {
		return false
	}
	first := lines.At(0)
	return isObsidianFence(first.Value(source))
}
