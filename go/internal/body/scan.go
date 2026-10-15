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
)

// fence opens or shuts a code block, inside which nothing is markdown. The run
// is captured because a block shuts only on its own character, in a run at
// least as long as the one that opened it: a "```" block may hold a "~~~" line
// and a four-backtick block may hold a three-backtick one. What follows the run
// is captured because a block shuts on a fence carrying nothing else: the
// "```js" line inside a "```" block is the block's own content.
var fence = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")

// listOpener opens a list item, whose own indented lines are its content. It
// bounds the indent the way footnoteDefinition does, because four spaces is a
// continuation line rather than a new item.
var listOpener = regexp.MustCompile(`^ {0,3}([-*+]|[0-9]{1,9}[.)])(\s|$)`)

// thematicBreak is a horizontal rule. It closes the block above it the way a
// heading does, so an indented code block may open on the line after one. RE2
// has no backreference, so each character is its own alternative.
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
// A fenced block is left out whole, and so is an indented code block: four
// spaces where a new block could open, with no footnote definition and no list
// item still running on. Inside either of those an indented line is that
// block's own content and may hold a mark the author meant, which is why the
// indent alone is not enough.
//
// What an indented code block cannot do is interrupt a paragraph, which is the
// whole of the rule: it opens after a blank line, at the start of the file, and
// after any line that closes a block of its own. A heading is one, a horizontal
// rule and a setext underline are two more, and a fence that shut is the last.
// mayOpen is false only after a line that could be continued.
func proseLines(source []byte, f func(sourceLine)) {
	// opener is the fence run that opened the block in hand, empty when none
	// is open.
	opener := ""
	openDefinition := false
	openList := false
	inCode := false
	// openParagraph says the line before this one was a line of prose, so a
	// setext underline on this line underlines it. It is not the negation of
	// mayOpen: after a line of an indented code block neither is true.
	openParagraph := false
	// mayOpen says a new block may open on this line: there was no line before
	// it, or the line before it was blank, or that line closed a block of its
	// own.
	mayOpen := true
	for i, raw := range strings.Split(string(source), "\n") {
		if opener != "" {
			// Inside a fenced block nothing is markdown and no state moves.
			// The block shuts on its own character, in a run at least as long
			// as the opener, with nothing but whitespace after it.
			if m := fence.FindStringSubmatch(raw); m != nil {
				run, rest := m[1], m[2]
				if run[0] == opener[0] && len(run) >= len(opener) && strings.TrimSpace(rest) == "" {
					opener, mayOpen, openParagraph = "", true, false
				}
			}
			continue
		}
		if m := fence.FindStringSubmatch(raw); m != nil {
			opener = m[1]
			// A fence this regexp matches is indented under four spaces, so it
			// is no definition's continuation line and the definition ends
			// here. A list item's content indent is the item's own, which this
			// scan does not measure, so only a fence at the margin ends one.
			openDefinition = false
			if atMargin(raw) {
				openList = false
			}
			mayOpen, openParagraph = true, false
			continue
		}
		blank := strings.TrimSpace(raw) == ""
		indented := strings.HasPrefix(raw, "    ") || strings.HasPrefix(raw, "\t")
		if indented && !openDefinition && !openList && (inCode || mayOpen) {
			inCode = true
			mayOpen, openParagraph = false, false
			continue
		}
		if !blank {
			inCode = false
		}
		line := sourceLine{at: i + 1, text: withoutCode(raw)}
		if m := footnoteDefinition.FindStringSubmatch(line.text); m != nil {
			if label := m[1]; strings.TrimSpace(label) != "" {
				line.definition, line.label = true, label
			}
		}
		// A heading, a horizontal rule and the underline of a setext heading
		// each end on their own line, so the line after one opens a new block,
		// an indented code block among them.
		closes := atxHeading.MatchString(line.text) ||
			thematicBreak.MatchString(line.text) ||
			(openParagraph && setextUnderline.MatchString(line.text))
		// A definition and a list item each run on through their blank and
		// indented lines, which is goldmark's own rule for continuing one, and
		// end at the first line of prose at the margin. A line indented one,
		// two or three spaces is a continuation line of the item it sits in,
		// whose content indent this scan does not measure, so it ends nothing:
		// ending the list there read the item's own later paragraph, indented
		// four, as a code block.
		switch {
		case line.definition:
			openDefinition, openList = true, false
		case listOpener.MatchString(line.text) && !closes:
			openDefinition, openList = false, true
		case !blank && atMargin(raw):
			openDefinition, openList = false, false
		}
		line.inside = line.definition || (indented && openDefinition)
		mayOpen = blank || closes
		openParagraph = !blank && !closes
		f(line)
	}
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
// A backtick run with no closer on the line is text, which is what goldmark
// makes of it, so the scan carries on from just after it rather than giving up
// on the rest of the line and missing a real mark.
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
