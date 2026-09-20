// The round trip a sub-list makes: internal/view writes the file and goldmark,
// the parser this package reads a note back with, has to see the same list.
// The indent rule lives in view and the reader of it lives here, so the two
// are only one rule while a test asks goldmark rather than stating an answer.

package body

import (
	"fmt"
	"testing"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"gdoc/internal/docs"
	"gdoc/internal/view"
)

// listItem is one item of a list, at a nesting level.
func listItem(ordered bool, level int, listID, s string, at int) docs.Block {
	return docs.Block{Paragraph: &docs.Paragraph{
		Style:  "NORMAL_TEXT",
		Bullet: &docs.Bullet{Ordered: ordered, NestingLevel: level, ListID: listID},
		Runs:   []docs.Run{{Kind: docs.KindText, Text: s, StartIndex: at, EndIndex: at + len([]rune(s))}},
	}}
}

// TestASubListViewWritesNestsWhenBodyReadsIt is the guarantee the export rests
// on for a list: the shape of the document survives the file.
//
// The three shapes are the ones a fixed indent per level loses. A bullet under
// a numbered item and a sub-list under a wide marker both used to read as a
// second list beside the first rather than inside it, so a document came home
// with its nesting flattened and, in the second shape, with its numbering
// wrong as well. Nothing warned, because nothing was lost: the text was all
// there and only its shape had changed.
func TestASubListViewWritesNestsWhenBodyReadsIt(t *testing.T) {
	for _, c := range []struct {
		name  string
		body  []docs.Block
		depth int
	}{
		{
			name: "a bullet under a numbered item",
			body: []docs.Block{
				listItem(true, 0, "l1", "one", 1),
				listItem(false, 1, "l1", "sub", 10),
			},
			depth: 2,
		},
		{
			name:  "a sub-list under the eleventh item, whose marker is wider",
			body:  append(elevenItems("l2"), listItem(true, 1, "l2", "sub", 400)),
			depth: 2,
		},
		{
			name: "three levels",
			body: []docs.Block{
				listItem(true, 0, "l3", "one", 1),
				listItem(true, 1, "l3", "two", 10),
				listItem(false, 2, "l3", "three", 20),
			},
			depth: 3,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			source, _ := view.Text(&docs.Document{
				Tabs: []docs.Tab{{ID: "t.0", Body: c.body}},
			})
			root := parse().Parser().Parse(text.NewReader([]byte(source)))

			if got := listDepth(root); got != c.depth {
				t.Errorf("goldmark reads %d levels of list in\n%s\nwant %d", got, source, c.depth)
			}
		})
	}
}

// elevenItems is a numbered list long enough that its marker grows a character.
func elevenItems(listID string) []docs.Block {
	var out []docs.Block
	for i := 1; i <= 11; i++ {
		out = append(out, listItem(true, 0, listID, fmt.Sprintf("item %d", i), i*20))
	}
	return out
}

// listDepth is the deepest run of lists inside one another. Two lists side by
// side are one level; a list inside a list's item is two.
func listDepth(root ast.Node) int {
	deepest := 0
	var walk func(n ast.Node, depth int)
	walk = func(n ast.Node, depth int) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			next := depth
			if _, ok := c.(*ast.List); ok {
				next++
				if next > deepest {
					deepest = next
				}
			}
			walk(c, next)
		}
	}
	walk(root, 0)
	return deepest
}
