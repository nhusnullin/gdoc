// This file is the block's content, read strictly: the markdown subset a block
// proposal carries, as the paragraphs the write is built from, and every
// construct outside that subset refused by name. lines.go holds the line a
// refusal names, propose.go holds the proposal itself, span.go finds the words in
// the document, and doc.go states the subset and names the test that pins each
// refusal.

package propose

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
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

// marks is what an inline node hands down to the text inside it.
type marks struct {
	bold   bool
	italic bool
	link   string
}

// runs flattens a paragraph's inline tree into runs, one stretch per set of
// marks, and refuses every inline construct the subset does not carry.
func (w *blockWalk) runs(node ast.Node) ([]Run, error) {
	var out []Run
	if err := w.appendRuns(&out, node, marks{}); err != nil {
		return nil, err
	}
	return mergeRuns(out), nil
}

func (w *blockWalk) appendRuns(out *[]Run, node ast.Node, m marks) error {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch typed := child.(type) {
		case *ast.Text:
			*out = append(*out, Run{Text: string(typed.Segment.Value(w.source)),
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
			dest := string(typed.Destination)
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
