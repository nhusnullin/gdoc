package main

import (
	"fmt"
	"os"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

const src = `---
title: Test
version: "1.0"
---

# Heading One

Some **bold** and *emph* and ` + "`code`" + ` and ~~struck~~ and ==marked== text.

- bullet one
- bullet two
  - nested
    - deeper

3. ordered starting at three
4. next

| Col A | Col B |
|-------|------:|
| x     | y     |

Term
:   Definition of term

> quoted

![alt text][img1]

Paragraph with words then ![d](data:image/png;base64,iVBORw0KGgo=) inline.

# **![][img1]**

[img1]: data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==
`

func main() {
	md := goldmark.New(goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.DefinitionList,
		extension.TaskList,
		extension.Footnote,
		MarkExtension,
	))
	reader := text.NewReader([]byte(src))
	doc := md.Parser().Parse(reader)
	depth := 0
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			depth--
			return ast.WalkContinue, nil
		}
		extra := ""
		switch v := n.(type) {
		case *ast.Heading:
			extra = fmt.Sprintf("level=%d", v.Level)
		case *ast.List:
			extra = fmt.Sprintf("ordered=%v start=%d marker=%q tight=%v", v.IsOrdered(), v.Start, string(v.Marker), v.IsTight)
		case *ast.Image:
			extra = fmt.Sprintf("dest=%.40q titlelen=%d", string(v.Destination), len(v.Title))
		case *ast.Link:
			extra = fmt.Sprintf("dest=%.40q", string(v.Destination))
		case *ast.Emphasis:
			extra = fmt.Sprintf("level=%d", v.Level)
		case *ast.FencedCodeBlock:
			extra = fmt.Sprintf("lang=%q", string(v.Language([]byte(src))))
		case *east.TableCell:
			extra = fmt.Sprintf("align=%v", v.Alignment)
		case *Mark:
			extra = "HIGHLIGHT"
		case *ast.Text:
			extra = fmt.Sprintf("%q", string(v.Segment.Value([]byte(src))))
		}
		fmt.Printf("%*s%s %s\n", depth*2, "", n.Kind(), extra)
		depth++
		return ast.WalkContinue, nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "walk error:", err)
	}
}
