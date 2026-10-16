package body

// A pencil is a comment.
//
// An aside the hub's reader never pays for still reaches the Doc's reader,
// grey, where it is cheapest. Obsidian dims a comment while editing and hides
// it while reading, so the one spelling that costs the note nothing is the one
// gdoc carries.
//
// goldmark already parses HTML's comment, inside a sentence as an ast.RawHTML
// holding the whole of it and on lines of its own as an ast.HTMLBlock. So
// there is no parser here for that spelling, only the reading: is this raw
// HTML a comment, and what are the words inside it. Obsidian's own %%
// spelling has no node in goldmark, and obsidian.go parses it into those same
// two nodes so that everything below reads one shape and every rule a comment
// has holds for both spellings without being written twice.

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// commentOpener and commentCloser are what an HTML comment is written with.
//
// They are constants here and not values read off the house style because the
// walk reads a comment before any house file is in hand; the house file states
// the opener as one of the pencil mark's own syntaxes, and
// TestThePencilSyntaxIsTheHouseStyles pins the two together.
const (
	commentOpener = "<!--"
	commentCloser = "-->"
)

// obsidianDelimiter opens and shuts the second spelling. Obsidian dims it
// while editing and hides it while reading, which is the whole reason a pencil
// is a comment.
const obsidianDelimiter = "%%"

// commentDelimiters is every pair a comment may be written with, in the order
// they are tried. The pair that matched is what the mark is looked up by, so
// the house style says which mark each spelling is and this package says which
// spellings there are.
var commentDelimiters = [2][2]string{
	{commentOpener, commentCloser},
	{obsidianDelimiter, obsidianDelimiter},
}

// commentWords is the words inside a piece of raw HTML, the delimiter it was
// written with, and whether that raw HTML is a comment at all. Raw HTML that
// is not one is the caller's to warn about and leave out, which is the rule it
// has always followed.
func commentWords(node *ast.RawHTML, source []byte) (words, opener string, ok bool) {
	var b strings.Builder
	for i := 0; i < node.Segments.Len(); i++ {
		segment := node.Segments.At(i)
		b.Write(segment.Value(source))
	}
	return insideComment(b.String())
}

// insideComment is one written comment with its delimiters taken off, which
// delimiter opened it, and whether it is a comment at all.
//
// One space inside each delimiter is the delimiter's, the way the space after
// a highlight's colour emoji is: an author writes "<!-- words -->" to separate
// the words from the syntax, and a grey run opening with a space is not what
// was typed. A second space is the author's own.
func insideComment(raw string) (words, opener string, ok bool) {
	for _, pair := range commentDelimiters {
		open, shut := pair[0], pair[1]
		if len(raw) < len(open)+len(shut) ||
			!strings.HasPrefix(raw, open) ||
			!strings.HasSuffix(raw, shut) {
			continue
		}
		inner := raw[len(open) : len(raw)-len(shut)]
		return strings.TrimSuffix(strings.TrimPrefix(inner, " "), " "), open, true
	}
	return "", "", false
}

// isComment says whether a piece of raw HTML is a comment.
func isComment(node *ast.RawHTML, source []byte) bool {
	_, _, comment := commentWords(node, source)
	return comment
}

// blockComment is the paragraphs inside a comment written on lines of its own,
// the delimiter it was written with, and whether this block of HTML is a
// comment at all. A block that is not one is the caller's to name and leave
// out, which is the rule it has always followed.
//
// goldmark holds the closing line of such a block apart from the rest, so the
// written comment is its lines and then its closure.
func blockComment(node *ast.HTMLBlock, source []byte) ([]string, string, bool) {
	var b strings.Builder
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		b.Write(line.Value(source))
	}
	if node.HasClosure() {
		closure := node.ClosureLine
		b.Write(closure.Value(source))
	}
	words, opener, comment := insideComment(strings.TrimSpace(b.String()))
	if !comment {
		return nil, "", false
	}
	return commentParagraphs(words), opener, true
}

// commentParagraphs gathers the lines inside a comment into paragraphs.
//
// One stretch of lines is one aside, and its lines are joined by a single
// space, because a comment is wrapped where the author's editor wrapped it and
// a line break there means no more than it does in prose. A blank line inside
// the comment opens a second aside, for the same reason a blank line between
// two paragraphs does. A comment holding no words is no paragraph at all.
func commentParagraphs(words string) []string {
	var paragraphs []string
	var stretch []string
	end := func() {
		if len(stretch) > 0 {
			paragraphs = append(paragraphs, strings.Join(stretch, " "))
			stretch = nil
		}
	}
	for _, line := range strings.Split(words, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			end()
			continue
		}
		stretch = append(stretch, line)
	}
	end()
	return paragraphs
}

// isPencil says whether a node is a comment that marks words, which is the
// third shape a mark is written in.
//
// A comment holding no words is not one. It marks none, so no run of its own
// reaches the document and nothing a refusal exists to protect is at risk:
// "empty is nothing" holds wherever a mark is refused, the same as it holds in
// a paragraph. TestAPencilGoesWhereAMarkGoes is the pin over where a comment
// that does mark words is refused.
func isPencil(node ast.Node, source []byte) bool {
	raw, ok := node.(*ast.RawHTML)
	if !ok {
		return false
	}
	words, _, comment := commentWords(raw, source)
	return comment && words != ""
}

// isCommentBlock says whether a block of HTML is a comment, closed or not. A
// fence the note never closes is one too: naming it a block of HTML would send
// the author looking for a tag they never wrote.
func isCommentBlock(node *ast.HTMLBlock, source []byte) bool {
	if _, _, comment := blockComment(node, source); comment {
		return true
	}
	return unclosedObsidianComment(node, source)
}
