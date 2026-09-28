// This file is the block's content, read strictly: the markdown subset a block
// proposal carries, as the paragraphs the write is built from, and every
// construct outside that subset refused by name. lines.go holds the line a
// refusal names, propose.go holds the proposal itself, span.go finds the words in
// the document, and doc.go states the subset and names the test that pins each
// refusal.

package propose

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"gdoc/internal/docsreq"
)

// NormalStyle is the named style a paragraph of body text takes, and the style
// a list item takes too: a bullet is a list membership in Docs, never a style.
const NormalStyle = "NORMAL_TEXT"

// ListKind is whether a new paragraph is a list item, and which kind of list it
// belongs to. The two kinds take different bullet presets, so they are two
// values rather than a flag.
type ListKind string

const (
	NotAList ListKind = ""
	Bulleted ListKind = "bullet"
	Numbered ListKind = "number"
)

// Run is one stretch of a new paragraph's text and the marks on it. The marks
// are the three the subset carries: bold, italic, and a link.
type Run struct {
	Text   string
	Bold   bool
	Italic bool
	// Link is the address the words open, empty otherwise. It is always an
	// absolute address, because a relative one opens nothing in a document.
	Link string
}

// Para is one new paragraph the block writes: the named style it takes, the
// list it belongs to, and its runs.
type Para struct {
	Style string
	List  ListKind
	Runs  []Run
}

// Text is what the paragraph reads as once the marks are dropped, which is what
// the insert carries.
func (p Para) Text() string {
	var b strings.Builder
	for _, r := range p.Runs {
		b.WriteString(r.Text)
	}
	return b.String()
}

// ParseContent reads a block's content into the paragraphs the write is built
// from, and refuses by name everything the subset does not hold.
//
// The subset is paragraphs, headings one to six, bulleted and numbered lists one
// level deep, and the three marks bold, italic and a link. Everything else is a
// refusal naming the line, because a block that quietly lost its table is a
// suggestion nobody can read and nobody can explain.
//
// gdoc's own markers are not asked about here. Every route into a document asks,
// and for the proposals file that is cmd/gdoc, which asks it of every field at
// once rather than of this one string.
func ParseContent(content string) ([]Para, error) {
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("the block carries no content, and a block proposes paragraphs")
	}
	source := []byte(content)
	w := &blockWalk{source: source}
	if err := w.blocks(parseBlock().Parser().Parse(text.NewReader(source)), NotAList); err != nil {
		return nil, err
	}
	if len(w.paras) == 0 {
		return nil, errors.New("the block carries no content, and a block proposes paragraphs")
	}
	return w.paras, nil
}

// parseBlock is goldmark with the GFM extensions and nothing else.
//
// GFM is on so that a table, a struck-out word and a task list are parsed and
// refused by name. With it off each of them is ordinary text, and a table would
// arrive in the document as rows of pipes.
//
// The typographer internal/body enables is off, and so are footnotes. The words
// here are a review's own, written to land in somebody's document as they were
// written, and a straight quote turned curly is a change gdoc made and nobody
// asked for. internal/body substitutes because it is publishing a whole note in
// the house look, which is a different job.
func parseBlock() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.GFM))
}

// blockWalk is one content string being read: the source, for the lines a
// refusal names, and the paragraphs so far.
type blockWalk struct {
	source []byte
	paras  []Para
}

// blocks reads every block in a subtree. list is the list the blocks belong to,
// which is NotAList at the top of the content and a kind inside a list item.
func (w *blockWalk) blocks(parent ast.Node, list ListKind) error {
	for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
		if err := w.block(node, list); err != nil {
			return err
		}
	}
	return nil
}

func (w *blockWalk) block(node ast.Node, list ListKind) error {
	switch typed := node.(type) {
	case *ast.Heading:
		if list != NotAList {
			return w.refuse(node, "a heading inside a list item is not something a block proposes; write the heading above the list")
		}
		runs, err := w.runs(node)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			return w.refuse(node, "this heading carries no text, and a block proposes words")
		}
		w.paras = append(w.paras, Para{Style: fmt.Sprintf("HEADING_%d", typed.Level), Runs: runs})
		return nil

	case *ast.Paragraph, *ast.TextBlock:
		runs, err := w.runs(node)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			return w.refuse(node, "this paragraph carries no text, and a block proposes words")
		}
		w.paras = append(w.paras, Para{Style: NormalStyle, List: list, Runs: runs})
		return nil

	case *ast.List:
		// One level deep. Nesting in a suggestion goes in through leading tabs,
		// which is unmeasured in SUGGEST mode, so a nested list is refused
		// rather than flattened: a flattened list is a change nobody asked for
		// and nobody would see until the suggestion was accepted.
		if list != NotAList {
			return w.refuse(node, "a nested list is not something a block proposes; nesting a suggested list is unmeasured, so write the list one level deep")
		}
		kind := Bulleted
		if typed.IsOrdered() {
			kind = Numbered
		}
		for item := typed.FirstChild(); item != nil; item = item.NextSibling() {
			before := len(w.paras)
			if err := w.blocks(item, kind); err != nil {
				return err
			}
			switch len(w.paras) - before {
			case 1:
			case 0:
				return w.refuseAt(w.itemLine(item), "this list item carries no text, and a block proposes words")
			default:
				// Docs numbers each paragraph of a list, so a second paragraph
				// in one item is a second item: the author's "2." prints as
				// "3.". internal/body carries the same failure as a warning,
				// because a publish rewrites a whole note and a refusal there
				// costs the document. Here the content is three lines a review
				// wrote and rewriting them is cheap.
				return w.refuseAt(w.itemLine(item), "a list item with more than one paragraph is not something a block proposes; write the item as one paragraph")
			}
		}
		return nil

	case *east.Table:
		return w.refuse(node, "a table is not something a block proposes; propose the paragraphs around it, and add the table by hand")

	case *ast.Blockquote:
		return w.refuse(node, "a block quote is not something a block proposes; write the words as a paragraph")

	case *ast.FencedCodeBlock:
		// The fence is the line the author sees. A fenced block's own lines
		// start at its first line of code, one further down.
		return w.refuseAt(max(w.line(node)-1, 1), "a code block is not something a block proposes; it is not in the house style either")

	case *ast.CodeBlock:
		return w.refuse(node, "a code block is not something a block proposes; it is not in the house style either")

	case *ast.HTMLBlock:
		return w.refuse(node, "HTML is not something a block proposes; it would arrive in the document as words")

	case *ast.ThematicBreak:
		return w.refuse(node, "a horizontal rule is not something a block proposes")

	default:
		// A container this walk does not name is read through rather than
		// dropped, and a leaf it does not name is refused: losing a block in
		// silence is the failure that only shows up once somebody reads the
		// suggestion.
		if node.HasChildren() {
			return w.blocks(node, list)
		}
		return w.refuse(node, fmt.Sprintf("%s is not something a block proposes", node.Kind().String()))
	}
}

// maxReference is the longest an entity reference can be, in bytes, counted
// from its "&" to its ";". The longest name HTML5 has is the thirty-one
// characters of "CounterClockwiseContourIntegral", and a numeric reference is
// shorter than that. A "&" whose name runs past this many bytes without a ";"
// is an ampersand somebody wrote, and it is left alone.
const maxReference = 33

// decodeText is one text segment as markdown says it reads, rather than as the
// author had to type it.
//
// goldmark leaves a backslash escape and an entity reference in the source and
// resolves both in its HTML renderer, because there they are already correct.
// Written into a Docs insertText they are not: "R&amp;D" would arrive as five
// characters nobody wrote, and "snake\_case" would keep its backslash. Every
// read-back compares against this same text, so nothing downstream would catch
// it.
//
// It is one left-to-right pass rather than the order goldmark's util.URLEscape
// chains the three resolvers in, which unescapes the whole segment first and
// reads the references out of what is left. Chained that way the backslash
// comes off "\&amp;" and the five characters behind it are then read as an
// entity, so an author who escaped an entity on purpose gets one ampersand.
// CommonMark says an escaped character is literal and never the start of a
// reference, and goldmark's own text writer makes the single pass this one
// makes. A link's destination is read through here too, because the spec reads
// an escape the same way on both sides of the brackets.
//
// internal/body does the entity half for the same reason, one file over. The
// escape half is this package's alone: body renders through goldmark's own
// walker for the marks, and this one reads the segments itself.
func decodeText(segment []byte) string {
	var b strings.Builder
	for at := 0; at < len(segment); {
		c := segment[at]
		switch {
		case c == '\\' && at+1 < len(segment) && util.IsPunct(segment[at+1]):
			b.WriteByte(segment[at+1])
			at += 2
		case c == '&':
			decoded, width := reference(segment[at:])
			if width == 0 {
				b.WriteByte(c)
				at++
				continue
			}
			b.Write(decoded)
			at += width
		default:
			b.WriteByte(c)
			at++
		}
	}
	return b.String()
}

// reference is the entity or numeric character reference at the front of s and
// how many bytes of s it took, or no bytes at all when what is there is not
// one.
//
// The shape is read here and the meaning is goldmark's: referenceWidth says
// where the reference at the front ends, and the two resolvers say whether
// that is a reference the spec knows. The shape has to be read first, because
// both resolvers scan whatever they are handed for every "&" in it rather than
// the one at its front. Handed the whole span up to the next ";", they would
// resolve a reference further along it and hand back the bytes in between as
// they stand, escapes and all, which is the single left-to-right pass gone: in
// "R&D costs\* are &lt; 5%" the "&" of "R&D" would swallow the backslash and
// the "&lt;" behind it.
func reference(s []byte) ([]byte, int) {
	width := referenceWidth(s)
	if width == 0 {
		return nil, 0
	}
	token := s[:width]
	decoded := util.ResolveEntityNames(util.ResolveNumericReferences(token))
	if bytes.Equal(decoded, token) {
		return nil, 0
	}
	return decoded, len(token)
}

// referenceWidth is how many bytes the reference shape at the front of s takes,
// its "&" and its ";" counted, and none when what is at the front is not that
// shape. It reads the shape the spec allows and nothing about the meaning: a
// name that is one HTML5 has, and a code point that is one a character can be,
// are the resolvers' own answers.
func referenceWidth(s []byte) int {
	if len(s) == 0 || s[0] != '&' {
		return 0
	}
	at, inside := 1, util.IsAlphaNumeric
	if at < len(s) && s[at] == '#' {
		at, inside = at+1, util.IsNumeric
		if at < len(s) && (s[at] == 'x' || s[at] == 'X') {
			at, inside = at+1, util.IsHexDecimal
		}
	}
	start := at
	for at < len(s) && at < maxReference && inside(s[at]) {
		at++
	}
	if at == start || at >= len(s) || at >= maxReference || s[at] != ';' {
		return 0
	}
	return at + 1
}

// marks is what an inline node hands down to the text inside it.
type marks struct {
	bold   bool
	italic bool
	link   string
}

// runs flattens a paragraph's inline tree into runs, one stretch per set of
// marks, and refuses every inline construct the subset does not carry.
//
// Two characters are refused here rather than written, and both are decoded
// ones: a reference or an escape puts into a run what the author could not type
// into it, so "&#12;" is five characters in the file and a form feed in the
// document.
//
// A character the Docs API strips out of an inserted text is the first.
// docsreq.Strippable says which they are and what they cost, and BlockBatch is
// the caller that pays it: every index it names is counted from the length of
// this text, so a unit the server drops puts each of them one place out.
//
// A line break is the second. Docs makes a paragraph of a newline and a soft
// break of a vertical tab, so one written this way would arrive as a paragraph
// the block never proposed, styled and counted as part of the one in front of
// it. appendRuns refuses the markdown spelling of the same thing a few lines
// up, and this is that rule held over the spelling it cannot see.
func (w *blockWalk) runs(node ast.Node) ([]Run, error) {
	var out []Run
	if err := w.appendRuns(&out, node, marks{}); err != nil {
		return nil, err
	}
	out = mergeRuns(out)
	for _, r := range out {
		if at := strings.IndexAny(r.Text, "\n\r\v"); at >= 0 {
			return nil, w.refuseAt(w.inlineLine(node), fmt.Sprintf(
				"the words carry U+%04X, a line break: a line break inside a paragraph is not something a block proposes, and an empty line makes the next paragraph",
				[]rune(r.Text[at:])[0]))
		}
		if bad, found := docsreq.Strippable(r.Text); found {
			return nil, w.refuseAt(w.inlineLine(node), fmt.Sprintf(
				"the words carry U+%04X, which the Docs API strips out of an inserted text: "+
					"gdoc counts the characters it sends to place everything after them, so take it out and write the block again", bad))
		}
	}
	return out, nil
}

func (w *blockWalk) appendRuns(out *[]Run, node ast.Node, m marks) error {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch typed := child.(type) {
		case *ast.Text:
			*out = append(*out, Run{Text: decodeText(typed.Segment.Value(w.source)),
				Bold: m.bold, Italic: m.italic, Link: m.link})
			// A wrapped line is a space: a paragraph is a paragraph because of
			// the empty line between two of them.
			if typed.SoftLineBreak() {
				*out = append(*out, Run{Text: " ", Bold: m.bold, Italic: m.italic, Link: m.link})
			}
			// A hard break is a break inside one paragraph, which Docs writes
			// as its own character and no read-back here counts. An empty line
			// is the way a block makes a second paragraph.
			if typed.HardLineBreak() {
				return w.refuseAt(w.lineAt(typed.Segment.Start),
					"a line break inside a paragraph is not something a block proposes; an empty line makes the next paragraph")
			}
		case *ast.String:
			*out = append(*out, Run{Text: string(typed.Value),
				Bold: m.bold, Italic: m.italic, Link: m.link})
		case *ast.Emphasis:
			next := m
			if typed.Level >= 2 {
				next.bold = true
			} else {
				next.italic = true
			}
			if err := w.appendRuns(out, typed, next); err != nil {
				return err
			}
		case *ast.Link:
			// The destination is a source segment like the words are, so it
			// carries the same entity references and backslash escapes, and a
			// link written raw would open an address nobody typed. It is read
			// by the same pass, because CommonMark reads an escape the same way
			// inside a destination as it does in the words.
			dest := decodeText(typed.Destination)
			if !isAddress(dest) {
				return w.refuseAt(w.inlineLine(typed), fmt.Sprintf(
					"the link to %q is not an address a document can open; write the full address, or drop the link and keep the words", dest))
			}
			next := m
			next.link = dest
			if err := w.appendRuns(out, typed, next); err != nil {
				return err
			}
		case *ast.AutoLink:
			address := string(typed.URL(w.source))
			// goldmark hands back an email address bare, and a relationship
			// without the scheme opens nothing.
			if typed.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(address), "mailto:") {
				address = "mailto:" + address
			}
			*out = append(*out, Run{Text: string(typed.Label(w.source)),
				Bold: m.bold, Italic: m.italic, Link: address})
		case *ast.CodeSpan:
			return w.refuseAt(w.inlineLine(typed), "code in a sentence is not one of the marks a block carries, which are bold, italic and a link")
		case *east.Strikethrough:
			return w.refuseAt(w.inlineLine(typed), "struck-out words are not one of the marks a block carries, which are bold, italic and a link")
		case *east.TaskCheckBox:
			return w.refuseAt(w.inlineLine(typed), "a task list is not something a block proposes; write the items as a plain list")
		case *ast.Image:
			return w.refuseAt(w.inlineLine(typed), "a picture is not something a block proposes; propose the words and add the picture by hand")
		case *ast.RawHTML:
			return w.refuseAt(w.inlineLine(typed), "HTML is not something a block proposes; it would arrive in the document as words")
		default:
			if err := w.appendRuns(out, child, m); err != nil {
				return err
			}
		}
	}
	return nil
}

// mergeRuns joins neighbours carrying the same marks and drops the empty ones.
// goldmark splits text at every syntactic boundary, and one request per word
// would be a batch nobody can read.
func mergeRuns(runs []Run) []Run {
	var out []Run
	for _, run := range runs {
		if run.Text == "" {
			continue
		}
		if len(out) > 0 {
			last := &out[len(out)-1]
			if last.Bold == run.Bold && last.Italic == run.Italic && last.Link == run.Link {
				last.Text += run.Text
				continue
			}
		}
		out = append(out, run)
	}
	return out
}

// isAddress is a link destination a document can open: it has a scheme, or it
// has a host. A hub path and a "#heading" jump have neither, and in a Google Doc
// both open nothing, so they are refused rather than written as dead links.
func isAddress(dest string) bool {
	if dest == "" || strings.HasPrefix(dest, "#") {
		return false
	}
	u, err := url.Parse(dest)
	if err != nil {
		return false
	}
	return u.Scheme != "" || u.Host != ""
}
