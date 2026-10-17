// The room that reads one file's bytes out of Drive, and the one path through
// the binary that reaches it. The reason is in doc.go.

package boundary

import (
	"go/ast"
	"go/parser"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The room, the call into it, and the chain of four names between that call
// and the two commands allowed to make it. Spelled out here rather than read
// from the source, so a call that moves is a failure a person reads rather
// than an allowlist that followed it.
const (
	assetRoom  = "gdoc/internal/asset"
	assetFetch = "Fetch"
	assetFile  = "cmd/gdoc/logo.go"
	logoFunc   = "logoFor"
	renderFunc = "renderNote"
	tableFunc  = "commands"
)

// assetCommands are the two commands that render a note, which is the whole
// reason either of them wants a logo.
var assetCommands = map[string]bool{"cmdBuild": true, "cmdPublish": true}

// TestOnlyBuildAndPublishCallTheAssetRoom reads the chain from the room up to
// the two commands, one link at a time.
//
// The room fetches one Drive file's bytes under a grant the caller makes, so
// what bounds it is not the request but the path to it: one call in one file,
// in a function one function calls, in a function two commands call. A fifth
// caller anywhere along that chain is a run reaching Drive that nobody
// reviewed, and this is where it is noticed. The grant itself is held narrow
// by internal/guard's own asset_test.go; this holds who may open it.
func TestOnlyBuildAndPublishCallTheAssetRoom(t *testing.T) {
	fetch, err := roomCalls(assetRoom, assetFetch)
	if err != nil {
		t.Fatal(err)
	}
	if len(fetch) != 1 {
		t.Fatalf("%s.%s is called in one place, and it is called from %v", assetRoom, assetFetch, fetch)
	}
	if fetch[0].file != assetFile || fetch[0].fn != logoFunc {
		t.Errorf("%s.%s is called inside %s in %s, and this call is in %s (%s)",
			assetRoom, assetFetch, logoFunc, assetFile, fetch[0].fn, fetch[0].file)
	}

	for _, link := range []struct {
		name string
		want map[string]bool
	}{
		{logoFunc, map[string]bool{renderFunc: true}},
		{renderFunc, assetCommands},
	} {
		callers, err := callsOf(link.name)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, c := range callers {
			got[c.fn] = true
		}
		if len(got) != len(link.want) {
			t.Errorf("%s is called by %v, and this tree calls it from %v", link.name, names(link.want), names(got))
			continue
		}
		for fn := range link.want {
			if !got[fn] {
				t.Errorf("%s is called by %v, and this tree calls it from %v", link.name, names(link.want), names(got))
				break
			}
		}
	}
}

// TestReadExportRestyleAndHelpNeverReachTheAssetRoom holds the other half:
// every other command is exactly as offline as it was.
//
// export, restyle and help marks read the house style for their own reasons,
// and read, comments and suggestions read a document. None of them renders a
// note, so none of them has a logo to fetch, and a path from any of them into
// the fetch would be a read command that made a request to Drive nobody asked
// for. The walk is over the call graph of cmd/gdoc rather than over one file,
// because the path that would appear is a helper two of them share.
//
// The one node left out of that walk is the command table. It names every
// command by construction, in a closure that is the command's own body, so a
// path through it is not one command reaching another: it is the dispatcher,
// and following it would make every command reach every other.
func TestReadExportRestyleAndHelpNeverReachTheAssetRoom(t *testing.T) {
	graph, err := callGraph("../cmd/gdoc")
	if err != nil {
		t.Fatal(err)
	}
	for fn := range graph {
		if !strings.HasPrefix(fn, "cmd") || assetCommands[fn] {
			continue
		}
		if path := reaches(graph, fn, logoFunc); path != "" {
			t.Errorf("%s reaches the asset room: %s. Only %v may", fn, path, names(assetCommands))
		}
	}
	// And the two that may, still do: a test that holds a rule nothing
	// exercises holds nothing.
	for fn := range assetCommands {
		if reaches(graph, fn, logoFunc) == "" {
			t.Errorf("%s no longer reaches %s; the fetch moved and this test did not", fn, logoFunc)
		}
	}
}

// roomCalls is every call of name qualified through an import of that package,
// outside the package itself and outside the tests. The import's local name is
// read per file, because an alias would otherwise make the call invisible.
func roomCalls(importPath, name string) ([]callSite, error) {
	var found []callSite
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		local := localName(f, importPath)
		if local == "" {
			return nil
		}
		for _, decl := range f.Decls {
			for _, in := range declNames(decl) {
				ast.Inspect(in.node, func(n ast.Node) bool {
					if qualifiedCall(n, local, name) {
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

// localName is the name importPath is known by in this file, or empty when the
// file does not import it. A blank or dot import answers empty: neither can
// carry a qualified call.
func localName(f *ast.File, importPath string) string {
	for _, spec := range f.Imports {
		got, err := strconv.Unquote(spec.Path.Value)
		if err != nil || got != importPath {
			continue
		}
		if spec.Name == nil {
			return importPath[strings.LastIndex(importPath, "/")+1:]
		}
		if spec.Name.Name == "_" || spec.Name.Name == "." {
			return ""
		}
		return spec.Name.Name
	}
	return ""
}

// qualifiedCall says whether this node is pkg.name(...).
func qualifiedCall(n ast.Node, pkg, name string) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}

// callGraph is every declaration in one directory and the names it calls, read
// by name the way callsOf reads them: a plain call by its identifier and a
// call through a receiver by the selected name. It is an approximation of the
// real graph and deliberately the generous one, because a test that asks
// whether a path exists must not miss one.
func callGraph(dir string) (map[string]map[string]bool, error) {
	graph := map[string]map[string]bool{}
	err := walkGo(dir, func(_, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		for _, decl := range f.Decls {
			for _, in := range declNames(decl) {
				if graph[in.name] == nil {
					graph[in.name] = map[string]bool{}
				}
				if in.name == tableFunc {
					// The table, not a caller. The comment on
					// TestReadExportRestyleAndHelpNeverReachTheAssetRoom says why.
					continue
				}
				ast.Inspect(in.node, func(n ast.Node) bool {
					if called := calleeName(n); called != "" {
						graph[in.name][called] = true
					}
					return true
				})
			}
		}
		return nil
	}, parser.SkipObjectResolution)
	return graph, err
}

// calleeName is the name this node calls, or empty when it is not a call.
func calleeName(n ast.Node) string {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return ""
	}
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		return fun.Sel.Name
	}
	return ""
}

// reaches is the path from one name to another through the graph, written as a
// person would read it, or empty when there is none. The path matters: a
// failure naming only the two ends sends somebody looking through every
// function in between.
func reaches(graph map[string]map[string]bool, from, to string) string {
	seen := map[string]bool{from: true}
	type step struct {
		name string
		path string
	}
	queue := []step{{from, from}}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		next := make([]string, 0, len(graph[at.name]))
		for name := range graph[at.name] {
			next = append(next, name)
		}
		sort.Strings(next)
		for _, name := range next {
			if name == to {
				return at.path + " -> " + to
			}
			if seen[name] || graph[name] == nil {
				continue
			}
			seen[name] = true
			queue = append(queue, step{name, at.path + " -> " + name})
		}
	}
	return ""
}
