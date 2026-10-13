// The guard over the words a refusal says to a colleague's session.

package boundary

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// person is the name no refusal may carry. A refusal reaches whoever runs the
// binary, and that is a colleague's session as often as it is Nail's.
const person = "Nail"

// TestNoRefusalNamesAPerson reads every production file under go/ and fails on
// a string that builds an error, through errors.New or fmt.Errorf, or that is
// the Error of an emit.Result, and names a person.
func TestNoRefusalNamesAPerson(t *testing.T) {
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		for _, lit := range refusalStrings(f) {
			if strings.Contains(lit, person) {
				t.Errorf("%s: a refusal names a person: %q", path, lit)
			}
		}
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
}

// TestTheRefusalScannerFindsAllThreeShapes keeps the scanner honest: a scanner
// that finds nothing passes every tree.
func TestTheRefusalScannerFindsAllThreeShapes(t *testing.T) {
	src := `package p
func a() error { return errors.New("one Nail has to guess at") }
func b() error { return fmt.Errorf("%s is one Nail reads", "x") }
func c() emit.Result { return emit.Result{OK: false, Error: "Nail again"} }
func d() error { return fmt.Errorf("two halves: " + "the second names Nail") }
func e() string { return "Nail in prose is not a refusal" }
`
	f, err := parser.ParseFile(token.NewFileSet(), "p.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	got := refusalStrings(f)
	want := []string{"one Nail has to guess at", "%s is one Nail reads", "Nail again", "two halves: the second names Nail"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("refusal strings = %q, want %q", got, want)
	}
}

// refusalStrings is the message of every errors.New and fmt.Errorf, and the
// value of every Error field in a composite literal, where that is a string
// literal or literals joined with +.
func refusalStrings(f *ast.File) []string {
	var out []string
	add := func(e ast.Expr) {
		if s, ok := literal(e); ok {
			out = append(out, s)
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if len(n.Args) > 0 && ((pkg.Name == "errors" && sel.Sel.Name == "New") || (pkg.Name == "fmt" && sel.Sel.Name == "Errorf")) {
				add(n.Args[0])
			}
		case *ast.KeyValueExpr:
			if k, ok := n.Key.(*ast.Ident); ok && k.Name == "Error" {
				add(n.Value)
			}
		}
		return true
	})
	return out
}

// literal is the string an expression spells out, when it is a string literal
// or string literals joined with +.
func literal(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		return s, err == nil
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		l, lok := literal(e.X)
		r, rok := literal(e.Y)
		return l + r, lok && rok
	case *ast.ParenExpr:
		return literal(e.X)
	}
	return "", false
}
