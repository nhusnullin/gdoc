// The one read that names no file, and the two rooms that ask for it. The
// reason is in doc.go.

package boundary

import (
	"go/ast"
	"go/parser"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The grant, the room that builds it, and the two files that read an account.
// Spelled out here rather than read from the source, so a call that moves is a
// failure a person reads rather than an allowlist that followed it.
const (
	accountGrant  = "AllowAccountRead"
	accountReader = "accountOf"
	accountRoom   = "cmd/gdoc/mcplogin.go"
	statusRoom    = "cmd/gdoc/main.go"
)

// TestOnlyAccountOfCallsAllowAccountRead holds that the grant is built in one
// place.
//
// The grant opens the one Drive read that names no file, so what bounds it is
// not the request but the room: accountOf builds a policy for that read alone
// and lets it die with the call. A second caller anywhere would be a second
// policy nobody reviewed, and the grant would stop being one object for one
// run. The read itself now has two callers, which is why this is the test that
// stays narrow while the next one counts to two.
func TestOnlyAccountOfCallsAllowAccountRead(t *testing.T) {
	found, err := callsOf(accountGrant)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("%s is granted in one place, and it is called from %v", accountGrant, found)
	}
	if found[0].fn != accountReader {
		t.Errorf("%s is granted inside %s, and this call is in %s (%s)",
			accountGrant, accountReader, found[0].fn, found[0].file)
	}
}

// TestAccountOfHasTwoCallers holds the move this milestone made: the read the
// MCP login tool answered with is now also what `gdoc auth status` names, and
// that is all. The guard did not widen by a request; the binary widened by one
// caller, and this test is where a third one is noticed.
func TestAccountOfHasTwoCallers(t *testing.T) {
	found, err := callsOf(accountReader)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]bool{}
	for _, c := range found {
		files[c.file] = true
	}
	if len(found) != 2 || !files[accountRoom] || !files[statusRoom] {
		t.Errorf("%s is read by %s and %s, and this tree calls it from %v",
			accountReader, accountRoom, statusRoom, found)
	}
}

// callSite is one call and the declaration it sits in, which is what a failure
// has to name for somebody to go and look at it.
type callSite struct {
	file string
	fn   string
}

// callsOf is every call of that name in the tree, outside the tests. A
// function literal assigned to a package-level variable counts as that
// variable, because that is how accountOf and the stubbable rooms beside it
// are written.
func callsOf(name string) ([]callSite, error) {
	var found []callSite
	err := walkGo("..", func(_, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		for _, decl := range f.Decls {
			for _, in := range declNames(decl) {
				ast.Inspect(in.node, func(n ast.Node) bool {
					if calls(n, name) {
						found = append(found, callSite{file: filepath.ToSlash(strings.TrimPrefix(path, "../")), fn: in.name})
					}
					return true
				})
			}
		}
		return nil
	}, parser.SkipObjectResolution)
	sort.Slice(found, func(i, j int) bool { return found[i].file < found[j].file })
	return found, err
}

// named is a declaration's body and the name a reader knows it by.
type named struct {
	name string
	node ast.Node
}

// declNames is the declarations a call can sit in: a function, and a
// package-level variable holding a function literal.
func declNames(decl ast.Decl) []named {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Body == nil {
			return nil
		}
		return []named{{name: d.Name.Name, node: d.Body}}
	case *ast.GenDecl:
		var out []named
		for _, spec := range d.Specs {
			v, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, value := range v.Values {
				if i < len(v.Names) {
					out = append(out, named{name: v.Names[i].Name, node: value})
				}
			}
		}
		return out
	}
	return nil
}

// calls says whether this node is a call of that name, written plainly or
// through a receiver.
func calls(n ast.Node, name string) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name == name
	case *ast.SelectorExpr:
		return fun.Sel.Name == name
	}
	return false
}
