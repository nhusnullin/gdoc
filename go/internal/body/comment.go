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
	"fmt"
	"regexp"
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
//
// The words come back as one line, because a comment inside a sentence is one
// run of the paragraph it sits in. goldmark hands a comment the author's
// editor wrapped back with its newlines inside it, and oneLine says why they
// go.
func commentWords(node *ast.RawHTML, source []byte) (words, opener string, ok bool) {
	var b strings.Builder
	for i := 0; i < node.Segments.Len(); i++ {
		segment := node.Segments.At(i)
		b.Write(segment.Value(source))
	}
	words, opener, ok = insideComment(b.String())
	if !ok {
		return "", "", false
	}
	return oneLine(words), opener, true
}

// insideComment is one written comment with its delimiters taken off, which
// delimiter opened it, and whether it is a comment at all.
//
// One space inside each delimiter is the delimiter's, the way the space after
// a highlight's colour emoji is: an author writes "<!-- words -->" to separate
// the words from the syntax, and a grey run opening with a space is not what
// was typed. Both readings above then hand the words to commentParagraphs,
// which trims each line, so no space on either edge of them survives: a grey
// run never opens or shuts on one.
// TestACommentsWordsCarryNoSpaceOnEitherEdge is the pin.
//
// A pair whose closer is not its opener holds no closer inside it. The words
// of an HTML comment end at the first "-->" a reader meets, so a piece of text
// opening with "<!--" and ending with "-->" and holding another one between
// them is two comments and whatever stands between them, never one: reading it
// as one put both delimiters into the document as grey words.
// TestTwoCommentsOnOneLineAreTwoPencils is the pin. Obsidian's fence is the
// other case, and it is delimited by whole lines, so "%%" inside it is words.
func insideComment(raw string) (words, opener string, ok bool) {
	for _, pair := range commentDelimiters {
		open, shut := pair[0], pair[1]
		if len(raw) < len(open)+len(shut) ||
			!strings.HasPrefix(raw, open) ||
			!strings.HasSuffix(raw, shut) {
			continue
		}
		inner := raw[len(open) : len(raw)-len(shut)]
		if open != shut && strings.Contains(inner, shut) {
			continue
		}
		return strings.TrimSuffix(strings.TrimPrefix(inner, " "), " "), open, true
	}
	return "", "", false
}

// oneLine is a comment's words as one line, which is what a comment inside a
// sentence reaches the document as.
//
// goldmark hands a comment that wraps over two lines back with the newline
// inside it, and a newline in a run is a line break in the published
// paragraph: an aside broke where the author's editor wrapped. The lines are
// joined the way commentParagraphs joins the lines of a stretch, by one space,
// because a line break inside a comment means no more than it does in prose.
// TestAWrappedCommentInsideASentenceIsOneLine is the pin.
func oneLine(words string) string {
	return strings.Join(commentParagraphs(words), " ")
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
// What a block of HTML is written as is blockText's to say, because body.go
// asks the same question of the same node and two readings of one block are
// one rule with a copy that can drift.
func blockComment(node *ast.HTMLBlock, source []byte) ([]string, string, bool) {
	// A comment the note never closed is asked first, because Obsidian's
	// opener is also its closer: "%%" on a fence line with "note 50%%" under
	// it opens a comment and closes none, and insideComment, which reads a
	// prefix and a suffix, said the block was a comment that shut. So the
	// fence swallowed every line after it as one grey aside with nothing
	// named. TestAnUnclosedFenceEndingInTheDelimiterIsStillUnclosed is the
	// pin.
	if _, unclosed := unclosedComment(node, source); unclosed {
		return nil, "", false
	}
	words, opener, comment := insideComment(blockText(node, source))
	if !comment {
		return nil, "", false
	}
	return commentParagraphs(words), opener, true
}

// unclosedComment is the delimiter a block of HTML opened a comment with and
// never closed, and whether it is one at all.
//
// It is a loss either way and it is named either way, but naming it as a block
// of HTML would send the author looking for a tag they never wrote, and the
// HTML spelling is the one the guide teaches and the one export writes. Both
// run to the end of the note, so the lines after the opener go with it, which
// is what the warning says.
func unclosedComment(node *ast.HTMLBlock, source []byte) (string, bool) {
	written := blockText(node, source)
	if obsidianFenceBlock(node, source) {
		// obsidian.go shuts a fence on a fence and on nothing else, so the
		// closing line is the whole of the answer here.
		return obsidianDelimiter, !node.HasClosure()
	}
	if !strings.HasPrefix(strings.TrimLeft(written, " \t"), commentOpener) {
		return "", false
	}
	// The text holding a closer is what closed, rather than goldmark's own
	// ClosureLine: goldmark shuts a comment on the first "-->" it meets
	// wherever on the line it is, and a comment opening and shutting on one
	// line of its own carries no closure line at all.
	return commentOpener, !strings.Contains(written, commentCloser)
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

// oldPencilSpelling is the attribute the grey mark was a Pandoc span with
// until milestone 20, "[words]{.pencil}". The spelling is gone from the
// parser and from the house style, and this is the one place it is still
// written down: a note that carries it is refused by name.
const oldPencilSpelling = ".pencil"

// oldPencilSpan is that spelling as a line carries it: the words in brackets
// where the bracket is on this line, and the attribute that shut them in
// either case. The bracket is optional because a span never had to open and
// close on one line for the attribute to reach the document as words.
var oldPencilSpan = regexp.MustCompile(
	`(?:\[([^\[\]]*))?\]\{` + regexp.QuoteMeta(oldPencilSpelling) + `\}`)

// checkOldPencilSpan refuses the old spelling by line, naming the comment to
// write instead.
//
// Nothing parses the span now, so the brackets and the braces would reach the
// document as the characters the author typed, with the build saying ok. That
// is the failure every line check in this package exists to prevent, and the
// refusal is read off the same lines the others are: nothing inside backticks,
// inside a code block or inside a block of HTML is markdown, so a note
// explaining the change is not refused for explaining it.
// TestTheOldSpanIsRefusedByLineNamingTheComment and
// TestTheOldSpanWhereNothingIsMarkdownIsNotRefused are the pins.
func checkOldPencilSpan(source []byte, root ast.Node) error {
	var refusal error
	checkLines(source, root, func(l sourceLine) {
		if refusal != nil {
			return
		}
		m := oldPencilSpan.FindStringSubmatch(l.text)
		if m == nil {
			return
		}
		// A closer with no bracket in front of it names no words, so the
		// refusal shows the shape to write rather than the author's own
		// aside.
		words := strings.TrimSpace(m[1])
		if words == "" {
			words = "words"
		}
		refusal = fmt.Errorf("line %d: %s is how the grey mark was written before milestone 20, and a pencil is a comment now: write %s %s %s, or %s %s %s, which Obsidian hides while somebody reads the note; left as it is, the brackets and the braces would reach the document as words",
			l.at, m[0], commentOpener, words, commentCloser,
			obsidianDelimiter, words, obsidianDelimiter)
	})
	return refusal
}

// openersRead counts, by line, the comment openers goldmark did read: every
// one inside a piece of raw HTML, counted against the line that piece of it
// sits on.
//
// Every opener the parse consumed is inside raw HTML, in both spellings and
// whether or not the raw HTML is a comment: goldmark's own parser claims the
// whole of "<!-- ... -->" and obsidian.go hands its pair over as the same
// node, and a piece of raw HTML that is no comment is warned about and left
// out rather than published. An opener the walk reads as a comment's own words
// is counted too, because "<!-- an aside about <!-- itself -->" is one comment
// holding two of them, and a count that missed the inner one would refuse an
// author for quoting the syntax inside their own aside.
//
// The count is per line rather than per node because a comment wraps: an
// author writes one aside and their editor breaks it, so goldmark hands the
// node over as one segment per line and each line is asked about its own.
func openersRead(source []byte, root ast.Node) map[int]int {
	out := map[int]int{}
	if root == nil {
		return out
	}
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		raw, ok := node.(*ast.RawHTML)
		if !ok {
			return ast.WalkContinue, nil
		}
		for i := 0; i < raw.Segments.Len(); i++ {
			segment := raw.Segments.At(i)
			out[lineOf(source, segment.Start)] +=
				strings.Count(string(segment.Value(source)), commentOpener)
		}
		return ast.WalkContinue, nil
	})
	return out
}

// checkUnclosedComment refuses an opener inside a sentence that the parse
// never read, naming the line and the closer to write.
//
// An opener on a line of its own opens a block of HTML, which the walk names
// as the comment it is and leaves out with every line under it. An opener in
// the middle of a sentence opens nothing at all: goldmark reads raw HTML only
// once it has the closer, so the aside stays prose and reaches the document
// black, with the typographer rewriting the two dashes of the opener into an
// en dash on the way. The author wrote words nobody was meant to read, and the
// build said ok. That is the failure every line check in this package exists
// to prevent, so this is read off the same lines the others are and refuses
// nothing inside backticks, inside a code block or inside a block of HTML.
//
// Only the HTML spelling is asked about. A lone "%%" is two characters an
// author may well have typed for themselves, and obsidian.go puts the reader
// back where it found it when no closer is reached, so those characters stay
// the author's own words rather than becoming a delimiter.
// TestAnUnclosedCommentInsideASentenceIsRefusedByLine and
// TestAnOpenerGoldmarkReadIsNotRefused are the pins.
func checkUnclosedComment(source []byte, root ast.Node) error {
	var refusal error
	read := openersRead(source, root)
	checkLines(source, root, func(l sourceLine) {
		if refusal != nil {
			return
		}
		if strings.Count(l.text, commentOpener) <= read[l.at] {
			return
		}
		refusal = fmt.Errorf("line %d: %s inside a sentence opens a comment this line never closes, and an opener nothing closes is not read as one: the aside and the %s in front of it would reach the document as words rather than grey; shut it with %s, or write the aside as %s words %s, which Obsidian hides while somebody reads the note",
			l.at, commentOpener, commentOpener, commentCloser,
			obsidianDelimiter, obsidianDelimiter)
	})
	return refusal
}

// isCommentBlock says whether a block of HTML is a comment, closed or not. A
// comment the note never closes is one too, in either spelling: naming it a
// block of HTML would send the author looking for a tag they never wrote.
func isCommentBlock(node *ast.HTMLBlock, source []byte) bool {
	if _, _, comment := blockComment(node, source); comment {
		return true
	}
	_, unclosed := unclosedComment(node, source)
	return unclosed
}

// blockText is every line of a block of HTML, its closing line included, with
// the whitespace around the whole taken off. It is what the reading functions
// above are asked about, and goldmark is why the closing line is named on its
// own: it holds that line apart from the rest, so the written comment is a
// block's lines and then its closure.
func blockText(node *ast.HTMLBlock, source []byte) string {
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
	return strings.TrimSpace(b.String())
}
