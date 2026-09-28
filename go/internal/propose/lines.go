// This file is where in a block's content a refusal points: the line a
// construct sits on, and the line it sits on when goldmark gave it no source
// position at all. block.go holds the content itself.

package propose

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// refuse names the line the node starts on and says what is wrong with it.
func (w *blockWalk) refuse(node ast.Node, says string) error {
	return w.refuseAt(w.line(node), says)
}

func (w *blockWalk) refuseAt(line int, says string) error {
	return fmt.Errorf("the block's content, line %d: %s", line, says)
}

// line is the 1-based line a block starts on.
func (w *blockWalk) line(node ast.Node) int {
	if lines := node.Lines(); lines != nil && lines.Len() > 0 {
		return w.lineAt(lines.At(0).Start)
	}
	if child := firstText(node); child != nil {
		return w.lineAt(child.Segment.Start)
	}
	return w.unpositionedLine(node)
}

// unpositionedLine is the line a block carrying no source position of its own
// sits on. A thematic break is the one the subset meets: goldmark builds it with
// no lines and no text inside it, and naming line 1 sends the author to the top
// of a file whose rule is thirty lines down. The first line holding anything
// after the block before it is this block's own line, because a blank line is
// what separated the two.
func (w *blockWalk) unpositionedLine(node ast.Node) int {
	from := 0
	for sibling := node.PreviousSibling(); sibling != nil; sibling = sibling.PreviousSibling() {
		if end, ok := lastOffset(sibling); ok {
			from = end
			break
		}
	}
	for offset := from; offset < len(w.source); {
		stop := bytes.IndexByte(w.source[offset:], '\n')
		if stop < 0 {
			stop = len(w.source)
		} else {
			stop += offset
		}
		if strings.TrimSpace(string(w.source[offset:stop])) != "" {
			return w.lineAt(offset)
		}
		offset = stop + 1
	}
	return w.lineAt(from)
}

// lastOffset is where a node's own text ends, or where the text of its last
// positioned descendant ends.
func lastOffset(node ast.Node) (int, bool) {
	if lines := node.Lines(); lines != nil && lines.Len() > 0 {
		return lines.At(lines.Len() - 1).Stop, true
	}
	if t, ok := node.(*ast.Text); ok {
		return t.Segment.Stop, true
	}
	for child := node.LastChild(); child != nil; child = child.PreviousSibling() {
		if offset, ok := lastOffset(child); ok {
			return offset, true
		}
	}
	return 0, false
}

// inlineLine is the line an inline node sits on. An inline node carries no
// lines of its own, so the position comes from the text inside it, and a node
// with no text at all falls back to the paragraph's own first line.
func (w *blockWalk) inlineLine(node ast.Node) int {
	if child := firstText(node); child != nil {
		return w.lineAt(child.Segment.Start)
	}
	for up := node.Parent(); up != nil; up = up.Parent() {
		if lines := up.Lines(); lines != nil && lines.Len() > 0 {
			return w.lineAt(lines.At(0).Start)
		}
	}
	return 1
}

// itemLine is the line a list item's refusal names. A ListItem carries no
// source position of its own: its Offset is a column inside the line, and
// goldmark appends lines to leaf blocks only, so the line comes from something
// inside the item, and from the nearest sibling when the item holds nothing.
func (w *blockWalk) itemLine(item ast.Node) int {
	if line, ok := w.descendantLine(item); ok {
		return line
	}
	for sibling := item.PreviousSibling(); sibling != nil; sibling = sibling.PreviousSibling() {
		if line, ok := w.descendantLine(sibling); ok {
			return line
		}
	}
	for sibling := item.NextSibling(); sibling != nil; sibling = sibling.NextSibling() {
		if line, ok := w.descendantLine(sibling); ok {
			return line
		}
	}
	return 1
}

// descendantLine is the line the node starts on, or the line of the first
// descendant that carries one.
func (w *blockWalk) descendantLine(node ast.Node) (int, bool) {
	if lines := node.Lines(); lines != nil && lines.Len() > 0 {
		return w.lineAt(lines.At(0).Start), true
	}
	if t, ok := node.(*ast.Text); ok {
		return w.lineAt(t.Segment.Start), true
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if line, ok := w.descendantLine(child); ok {
			return line, true
		}
	}
	return 0, false
}

// lineAt turns a source offset into the 1-based line holding it.
func (w *blockWalk) lineAt(offset int) int {
	if offset < 0 || offset > len(w.source) {
		return 1
	}
	return 1 + strings.Count(string(w.source[:offset]), "\n")
}

// firstText finds a node's first text leaf, which is the only thing on a
// heading or a link that carries a source position.
func firstText(node ast.Node) *ast.Text {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			return t
		}
		if found := firstText(child); found != nil {
			return found
		}
	}
	return nil
}
