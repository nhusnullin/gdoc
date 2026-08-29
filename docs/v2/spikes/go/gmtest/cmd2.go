package main

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
)

func runMarkTest() {
	src := "---\ntitle: T\ngdoc: 1abc\n---\n\n# H\n\nPara ==highlight me== and ==two words== plus a = sign and x==y.\n"
	gm := goldmark.New(goldmark.WithExtensions(
		extension.GFM, &frontmatter.Extender{}, &markExt{},
	))
	pctx := parser.NewContext()
	doc := gm.Parser().Parse(text.NewReader([]byte(src)), parser.WithContext(pctx))
	d := 0
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering { d--; return ast.WalkContinue, nil }
		extra := ""
		if t, ok := n.(*ast.Text); ok { extra = fmt.Sprintf("%q", string(t.Value([]byte(src)))) }
		fmt.Printf("%s%s %s\n", strings.Repeat("  ", d), n.Kind(), extra)
		d++
		return ast.WalkContinue, nil
	})
	var meta struct {
		Title string `yaml:"title"`
		Gdoc  string `yaml:"gdoc"`
	}
	if fm := frontmatter.Get(pctx); fm != nil {
		if err := fm.Decode(&meta); err != nil { fmt.Println("fm err:", err) }
	}
	fmt.Printf("FRONTMATTER: title=%q gdoc=%q\n", meta.Title, meta.Gdoc)
}
