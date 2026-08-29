package main

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func typoTest() {
	src := `Say "quoted" and 'single' -- dash.`
	for _, tc := range []struct{ name string; exts []goldmark.Extender }{
		{"plain", nil},
		{"typographer", []goldmark.Extender{extension.Typographer}},
	} {
		gm := goldmark.New(goldmark.WithExtensions(tc.exts...))
		doc := gm.Parser().Parse(text.NewReader([]byte(src)))
		var parts []string
		_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering { return ast.WalkContinue, nil }
			switch v := n.(type) {
			case *ast.Text:
				parts = append(parts, fmt.Sprintf("Text%q", string(v.Value([]byte(src)))))
			case *ast.String:
				parts = append(parts, fmt.Sprintf("String%q", string(v.Value)))
			}
			return ast.WalkContinue, nil
		})
		fmt.Printf("%-12s %s\n", tc.name, strings.Join(parts, " "))
	}
}
