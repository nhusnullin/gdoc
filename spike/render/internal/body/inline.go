package body

// Flattening goldmark's inline tree into runs.
//
// A run is a stretch of text sharing one set of marks. goldmark hands back a
// tree of nested emphasis, links and code spans; Word wants a flat list. The
// recursion carries the marks down, which is all there is to it.

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/util"
)

// decodeEntities turns "&amp;" into "&" and "&#8212;" into an em dash.
//
// goldmark leaves entity references in the source text and resolves them in its
// HTML renderer, because in HTML they are already correct. Written into a w:t
// they are not: the XML escape then doubles them, and a document explaining
// escaping ends up saying "&amp;amp;".
func decodeEntities(source []byte) string {
	return string(util.ResolveEntityNames(util.ResolveNumericReferences(source)))
}

// Run is one stretch of text and the marks on it.
type Run struct {
	Text      string
	Bold      bool
	Italic    bool
	Mono      bool
	Highlight string
	// Link is the destination when this run sits inside a hyperlink. Empty
	// otherwise. The Python renderer drops it; see Options.Hyperlinks.
	Link string
}

type marks struct {
	bold      bool
	italic    bool
	mono      bool
	highlight string
	link      string
}

// ImageRef is a picture found in an inline tree, kept with its alt text.
type ImageRef struct {
	Target string
	Alt    []Run
}

// inlineRuns flattens an inline subtree. Images are skipped: a caller that
// wants them asks collectImages for them, so nothing depends on where in the
// tree they were found.
func inlineRuns(node ast.Node, source []byte) []Run {
	var out []Run
	appendInline(&out, node, source, marks{})
	return merge(out)
}

func appendInline(out *[]Run, node ast.Node, source []byte, m marks) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch typed := child.(type) {
		case *ast.Text:
			*out = append(*out, Run{Text: decodeEntities(typed.Segment.Value(source)),
				Bold: m.bold, Italic: m.italic, Mono: m.mono,
				Highlight: m.highlight, Link: m.link})
			// A soft break is a space, a hard break is a line of its own.
			if typed.SoftLineBreak() {
				*out = append(*out, Run{Text: " ", Bold: m.bold, Italic: m.italic,
					Mono: m.mono, Highlight: m.highlight, Link: m.link})
			}
			if typed.HardLineBreak() {
				*out = append(*out, Run{Text: "\n", Bold: m.bold, Italic: m.italic,
					Mono: m.mono, Highlight: m.highlight, Link: m.link})
			}
		case *ast.String:
			// What the typographer substitutes in: curly quotes, dashes, an
			// ellipsis. Held as bytes on the node rather than in the source.
			*out = append(*out, Run{Text: string(typed.Value), Bold: m.bold,
				Italic: m.italic, Mono: m.mono, Highlight: m.highlight, Link: m.link})
		case *ast.CodeSpan:
			next := m
			next.mono = true
			*out = append(*out, Run{Text: nodeText(typed, source), Bold: next.bold,
				Italic: next.italic, Mono: true, Highlight: next.highlight, Link: next.link})
		case *ast.Emphasis:
			next := m
			if typed.Level >= 2 {
				next.bold = true
			} else {
				next.italic = true
			}
			appendInline(out, typed, source, next)
		case *Highlight:
			next := m
			next.highlight = "yellow"
			appendInline(out, typed, source, next)
		case *east.Strikethrough:
			appendInline(out, typed, source, m)
		case *east.TaskCheckBox:
			// The reference renderer takes these from pandoc, which writes the
			// ballot characters into the text rather than marking the item.
			glyph := "\u2610 "
			if typed.IsChecked {
				glyph = "\u2612 "
			}
			*out = append(*out, Run{Text: glyph, Bold: m.bold, Italic: m.italic,
				Mono: m.mono, Highlight: m.highlight, Link: m.link})
		case *ast.Link:
			next := m
			next.link = string(typed.Destination)
			appendInline(out, typed, source, next)
		case *ast.AutoLink:
			label := string(typed.URL(source))
			*out = append(*out, Run{Text: label, Bold: m.bold, Italic: m.italic,
				Mono: m.mono, Highlight: m.highlight, Link: label})
		case *ast.Image:
			// Handled by collectImages. Its alt text is not rendered here,
			// which matches the reference renderer: an image inside a
			// paragraph is lifted out and captioned instead.
			continue
		case *ast.RawHTML, *ast.HTMLBlock:
			continue
		default:
			appendInline(out, child, source, m)
		}
	}
}

// nodeText reads the literal text of a leaf such as a code span.
func nodeText(node ast.Node, source []byte) string {
	var b strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			b.Write(text.Segment.Value(source))
		}
	}
	return b.String()
}

// merge joins adjacent runs that carry the same marks. goldmark splits text at
// every syntactic boundary, and one w:r per word would bloat the document
// without changing a pixel.
func merge(runs []Run) []Run {
	out := runs[:0:0]
	for _, run := range runs {
		if run.Text == "" {
			continue
		}
		if len(out) > 0 {
			last := &out[len(out)-1]
			if last.Bold == run.Bold && last.Italic == run.Italic &&
				last.Mono == run.Mono && last.Highlight == run.Highlight &&
				last.Link == run.Link {
				last.Text += run.Text
				continue
			}
		}
		out = append(out, run)
	}
	return out
}

// collectImages finds every picture in an inline subtree, however deeply it is
// nested.
//
// The search is recursive, and that is not a refinement. Drive writes a picture
// on its own line as a bold heading, `# **![][image1]**`, so the image sits
// inside a Strong node. Looking only at the top level of the inline list found
// nothing and dropped the picture in silence. Formatting can nest arbitrarily,
// so nothing may assume a depth.
func collectImages(node ast.Node, source []byte) []ImageRef {
	var found []ImageRef
	var walk func(ast.Node)
	walk = func(parent ast.Node) {
		for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
			if image, ok := child.(*ast.Image); ok {
				found = append(found, ImageRef{
					Target: string(image.Destination),
					Alt:    inlineRuns(image, source),
				})
				continue
			}
			walk(child)
		}
	}
	walk(node)
	return found
}

// plainText is what a heading reads as once its marks are dropped. The numberer
// works on this, not on the marked-up form.
func plainText(runs []Run) string {
	var b strings.Builder
	for _, run := range runs {
		b.WriteString(run.Text)
	}
	return b.String()
}
