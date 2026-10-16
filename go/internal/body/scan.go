package body

// The source scan both line checks read: the lines of a note with its code
// left out, and the footnote definitions marked.
//
// Two checks are asked of the source rather than of the tree, for the reason
// footnote.go and mark.go each give, and both need the same lines. Nothing
// inside a code block is markdown, so the syntax an author writes inside one
// defines nothing and refuses nothing, and a definition's own continuation
// lines are a note's words rather than the document's prose. The rule is here
// once because two copies of it drifted: a fence closed on a fence of the other
// character, and an indented code block was read as prose.

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// fence opens or shuts a code block, inside which nothing is markdown. The run
// is captured because a block shuts only on its own character, in a run at
// least as long as the one that opened it: a "```" block may hold a "~~~" line
// and a four-backtick block may hold a three-backtick one. What follows the run
// is captured because a block shuts on a fence carrying nothing else: the
// "```js" line inside a "```" block is the block's own content.
//
// The indent is not bounded here. A fence is indented up to three spaces from
// the content indent of the block it sits in, which is four inside a footnote
// definition and the item's own inside a list, so the bound is the caller's and
// is measured against base.
var fence = regexp.MustCompile("^[ \t]*(`{3,}|~{3,})(.*)$")

// listOpener opens a list item, whose own indented lines are its content. The
// marker is captured because the item's content indent is measured from it: a
// "- " item carries its lines at two, a "1. " item at three.
//
// The indent is not bounded here either, because a nested item sits at its
// parent's content indent. The bound is base, the same one the fence is
// measured against.
var listOpener = regexp.MustCompile(`^[ \t]*([-*+]|[0-9]{1,9}[.)])(\s|$)`)

// thematicBreak is the `---` line, which a note writes for a page break. It
// closes the block above it the way a heading does, so an indented code block
// may open on the line after one. RE2 has no backreference, so each character
// is its own alternative.
var thematicBreak = regexp.MustCompile(`^ {0,3}(\*[ \t]*){3,}$|^ {0,3}(-[ \t]*){3,}$|^ {0,3}(_[ \t]*){3,}$`)

// setextUnderline is the underline under a setext heading, which closes the
// paragraph it underlines. It is only an underline where a paragraph is open:
// "===" with a blank line above it is a paragraph of its own, and the indented
// line after that one continues it rather than opening a code block.
var setextUnderline = regexp.MustCompile(`^ {0,3}(=+|-+)[ \t]*$`)

// sourceLine is one line of the note as a check reads it.
type sourceLine struct {
	// at is the 1-based line number, counted the one way internal/markers
	// counts it, so a refusal names the line an editor shows.
	at int
	// text is the line with its code spans taken out.
	text string
	// code says a code span was taken out of this line, so text is shorter
	// than the line the author wrote. A check reading an empty stretch of
	// text cannot tell whether the author left it empty or the code came out
	// of it, and this is how it asks.
	code bool
	// definition says this line opens a footnote definition, and label is the
	// label it opens.
	definition bool
	label      string
	// inside says this line is a definition's own, its opener or one of its
	// continuation lines, so a footnote mark on it is inside a note rather
	// than in the document's own words.
	inside bool
}

// proseLines calls f for every line of the source a check should read, in
// order, with the code left out.
//
// A fenced block is left out whole, and so is an indented code block. Both are
// measured from the content indent of the block they sit in rather than from
// the margin: a footnote definition carries its lines at four and a list item
// at its marker's width, so a fence inside one is indented four or more and
// four spaces inside one is the note's own prose rather than code. Measuring
// from the margin instead read a code sample in a nested list item as prose
// and refused a note nothing was wrong with.
//
// What an indented code block cannot do is interrupt a paragraph, which is the
// rest of the rule: it opens after a blank line, at the start of the file, and
// after any line that closes a block of its own. A heading is one, a horizontal
// rule and a setext underline are two more, and a fence that shut is the last.
// mayOpen is false only after a line that could be continued.
func proseLines(source []byte, f func(sourceLine)) {
	// opener is the fence run that opened the block in hand, empty when none
	// is open, and fenceBase the content indent it opened at: the closer is
	// indented up to three spaces from there and anything deeper is content.
	opener, fenceBase := "", 0
	openDefinition := false
	inCode := false
	// base is the content indent of the block still running on: four inside a
	// footnote definition, the marker's width inside a list item, and zero at
	// the margin. A fence opens within three spaces of it and an indented code
	// block four beyond it.
	base := 0
	// openParagraph says the line before this one was a line of prose, so a
	// setext underline on this line underlines it. It is not the negation of
	// mayOpen: after a line of an indented code block neither is true.
	openParagraph := false
	// mayOpen says a new block may open on this line: there was no line before
	// it, or the line before it was blank, or that line closed a block of its
	// own.
	mayOpen := true
	for i, raw := range strings.Split(string(source), "\n") {
		indent := indentWidth(raw)
		if opener != "" {
			// Inside a fenced block nothing is markdown and no state moves.
			// The block shuts on its own character, in a run at least as long
			// as the opener, with nothing but whitespace after it, indented no
			// more than three spaces past the block's own content indent.
			if m := fence.FindStringSubmatch(raw); m != nil && indent <= fenceBase+3 {
				run, rest := m[1], m[2]
				if run[0] == opener[0] && len(run) >= len(opener) && strings.TrimSpace(rest) == "" {
					opener, mayOpen, openParagraph = "", true, false
				}
			}
			continue
		}
		if m := fence.FindStringSubmatch(raw); m != nil && indent <= base+3 && opensFence(m[1], m[2]) {
			opener, fenceBase = m[1], base
			// A fence shallower than the content indent of the block in hand
			// is outside that block, so the block ends here.
			if indent < base {
				openDefinition, base, fenceBase = false, 0, 0
			}
			mayOpen, openParagraph = true, false
			continue
		}
		blank := strings.TrimSpace(raw) == ""
		if !blank && indent >= base+4 && (inCode || mayOpen) {
			inCode = true
			mayOpen, openParagraph = false, false
			continue
		}
		if !blank {
			inCode = false
		}
		stripped := withoutCode(raw)
		line := sourceLine{at: i + 1, text: stripped, code: stripped != raw}
		if m := footnoteDefinition.FindStringSubmatch(line.text); m != nil {
			if label := m[1]; strings.TrimSpace(label) != "" {
				line.definition, line.label = true, label
			}
		}
		// A heading, a thematic break and the underline of a setext heading
		// each end on their own line, so the line after one opens a new block,
		// an indented code block among them.
		closes := atxHeading.MatchString(line.text) ||
			thematicBreak.MatchString(line.text) ||
			(openParagraph && setextUnderline.MatchString(line.text))
		// A definition and a list item each run on through their blank and
		// indented lines, which is goldmark's own rule for continuing one, and
		// end at the first line of prose at the margin. A line indented inside
		// the block in hand is one of its own lines and ends nothing.
		switch {
		case line.definition:
			openDefinition, base = true, definitionIndent
		case indent <= base+3 && !closes && listOpener.MatchString(line.text):
			// A list item indented inside the definition still running is one
			// of that definition's own blocks, which is goldmark's reading of
			// it too: the item is lifted into the note. Ending the definition
			// here read the item and every line after it as the document's
			// own prose, so a mark in one answered for a note nothing in the
			// document marks, and that note's words reached no page with the
			// build saying ok.
			// TestAMarkInAListInsideADefinitionDoesNotAnswerForThatNote is the
			// pin.
			if indent < definitionIndent {
				openDefinition = false
			}
			base = itemIndent(line.text, indent)
		case !blank && atMargin(raw):
			openDefinition, base = false, 0
		}
		line.inside = line.definition || (openDefinition && indent >= definitionIndent)
		mayOpen = blank || closes
		openParagraph = !blank && !closes
		f(line)
	}
}

// opensFence says a line matching fence opens a block rather than being prose.
//
// CommonMark forbids a backtick in the info string of a backtick fence, so
// "```x``` is a span" opens nothing: goldmark reads a code span there and the
// rest of the line as words. Read as an opener, every line after it was left
// out of both checks until something closed the block, so a mark naming no
// note and the grey mark's old spelling reached the document as the words the
// author wrote; and a real fenced block under such a line was read as prose
// instead, which refused a note nothing was wrong with. A tilde fence's info
// string may hold anything, backticks included.
// TestABacktickSpanIsNotACodeFence is the pin.
func opensFence(run, info string) bool {
	return run[0] != '`' || !strings.Contains(info, "`")
}

// definitionIndent is the content indent of a footnote definition, which is
// goldmark's own: four spaces, whatever the label's width.
const definitionIndent = 4

// itemIndent is the content indent of the list item this line opens: its own
// indent, the marker, and the one space after it. An author writing more than
// one space there carries their content further in, which this does not follow;
// the answer is only ever too shallow, and too shallow reads a line as the
// item's prose rather than as its code.
func itemIndent(text string, indent int) int {
	m := listOpener.FindStringSubmatch(text)
	if m == nil {
		return indent
	}
	return indent + len(m[1]) + 1
}

// indentWidth is how far the line is indented, counting a tab to the next
// multiple of four, which is how CommonMark reads one.
func indentWidth(raw string) int {
	n := 0
	for _, r := range raw {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4 - n%4
		default:
			return n
		}
	}
	return n
}

// atMargin says the line starts in column 0, so it is no continuation line of
// the block above it.
func atMargin(raw string) bool {
	return !strings.HasPrefix(raw, " ") && !strings.HasPrefix(raw, "\t")
}

// withoutCode is the line with its code spans taken out, because nothing
// inside backticks is markdown. goldmark reads no footnote mark in a span, and
// a note explaining a regular expression writes "[^abc]" in one: read as a
// mark, it would refuse a note nothing is wrong with.
//
// A backtick run with no closer on the line is text as far as this line goes,
// so the scan carries on from just after it rather than giving up on the rest
// of the line and missing a real mark. Where goldmark shut that run on a later
// line of the same paragraph, the span wrapped and the words inside it are
// code: blankCodeSpans has already taken them out of the line this reads, and
// holds why one line cannot answer that on its own.
func withoutCode(line string) string {
	src := []byte(line)
	var b strings.Builder
	for i := 0; i < len(src); {
		if src[i] != '`' {
			b.WriteByte(src[i])
			i++
			continue
		}
		shut := codeSpanEnd(src, i)
		if shut < 0 {
			open := ticks(src, i)
			b.Write(src[i : i+open])
			i += open
			continue
		}
		i = shut
	}
	return b.String()
}

// codeSpanEnd is the index just after the code span that opens at i, or -1
// where the backtick run at i has no closer of its own length on the line.
func codeSpanEnd(line []byte, i int) int {
	open := ticks(line, i)
	for j := i + open; j < len(line); {
		if line[j] != '`' {
			j++
			continue
		}
		n := ticks(line, j)
		if n == open {
			return j + n
		}
		j += n
	}
	return -1
}

// ticks is the length of the run of backticks starting at i.
func ticks(line []byte, i int) int {
	n := 0
	for i+n < len(line) && line[i+n] == '`' {
		n++
	}
	return n
}

// checkLines calls f for every source line a refusal may be read off: the
// prose lines, with the lines of a block of HTML left out.
//
// A block of HTML that is not a comment is named on the envelope and left out
// of the document, and nothing inside one is markdown: goldmark reads the
// whole block as one leaf, so a footnote definition written there defines
// nothing and the grey mark's old spelling marks nothing. That is the rule a
// code block already has, and the lines come off the parse rather than off a
// second reading of the source, because CommonMark opens a block of HTML on
// seven different conditions and shuts it on as many.
// A code span that wrapped comes off the parse for the same reason, and
// blankCodeSpans holds it.
// TestADefinitionInsideABlockOfHTMLRefusesNothing and
// TestTheOldSpanWhereNothingIsMarkdownIsNotRefused are the pins.
func checkLines(source []byte, root ast.Node, f func(sourceLine)) {
	html := htmlBlockLines(source, root)
	proseLines(blankCodeSpans(source, root), func(l sourceLine) {
		if html[l.at] {
			return
		}
		f(l)
	})
}

// blankCodeSpans is the source with the inside of every code span the parse
// read replaced by spaces, byte for byte and never over a newline, so every
// line keeps its number, its length and its indent.
//
// withoutCode reads one line at a time, and a code span wraps: a backtick run
// with no closer on its own line is text as far as that line goes, while
// goldmark reads the paragraph to its end and shuts the span on the line
// below. So an opener, a footnote mark or the grey mark's old spelling written
// inside backticks the author's editor wrapped was read as the document's own
// words, and all three line checks refused a note for the one place this
// package promises nothing is markdown. Which lines a line cannot answer, so
// the spans come off the parse, the way the lines of a block of HTML do.
// TestAWrappedCodeSpanIsStillCode is the pin.
//
// Spaces rather than nothing at all, because every structural reading in
// proseLines is measured in columns: taking the characters out moved the
// fence, the list marker and the indent of a code block that a line is read
// by. The source is copied rather than written over, because Render hands the
// note's own bytes in and the walk reads them after this.
func blankCodeSpans(source []byte, root ast.Node) []byte {
	if root == nil {
		return source
	}
	var out []byte
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if _, ok := node.(*ast.CodeSpan); !ok {
			return ast.WalkContinue, nil
		}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			text, ok := child.(*ast.Text)
			if !ok {
				continue
			}
			if out == nil {
				out = append([]byte(nil), source...)
			}
			segment := text.Segment
			for i := segment.Start; i < segment.Stop && i < len(out); i++ {
				if out[i] != '\n' {
					out[i] = ' '
				}
			}
		}
		return ast.WalkSkipChildren, nil
	})
	if out == nil {
		return source
	}
	return out
}

// htmlBlockLines is every line a block of HTML holds, the line it shuts on
// included. goldmark appends one segment per line of a leaf block, so the
// lines are the segments' own.
func htmlBlockLines(source []byte, root ast.Node) map[int]bool {
	out := map[int]bool{}
	if root == nil {
		return out
	}
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		block, ok := node.(*ast.HTMLBlock)
		if !ok {
			return ast.WalkContinue, nil
		}
		lines := block.Lines()
		for i := 0; i < lines.Len(); i++ {
			out[lineOf(source, lines.At(i).Start)] = true
		}
		if block.HasClosure() {
			out[lineOf(source, block.ClosureLine.Start)] = true
		}
		return ast.WalkContinue, nil
	})
	return out
}
