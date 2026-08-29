// Package boundary enforces that gdoc has exactly one wire. This is the v1
// allowlist test, ported: it fails in BOTH directions, on spread and on silent
// disappearance.
//
// Two checks, because importing net/http and dialing with it are not the same
// thing. The import allowlist says which packages may name the type at all.
// The builder allowlist says who may make a client out of it, and that is the
// single room the guard occupies.
package boundary

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// allowed lists the package directories, relative to the module root, that may
// import net/http. `internal/auth` is here because its Refresh and Login take
// the guard's *http.Client as a parameter: it names the type, and builders
// below proves it never makes one. `internal/auth/loopback` joins this set in
// Task 7, which adds the login listener. A room is listed only once it exists,
// because this test has to be green at every commit.
var allowed = map[string]bool{
	"internal/guard": true,
	"internal/auth":  true,
}

// builders lists the package directories that may construct an HTTP client or
// reach the package-level dialing helpers. Exactly one, and it stays one.
var builders = map[string]bool{
	"internal/guard": true,
}

// dialers are the net/http names that reach the network on their own, without
// a client anybody handed over.
var dialers = map[string]bool{
	"DefaultClient": true, "DefaultTransport": true,
	"Get": true, "Post": true, "PostForm": true, "Head": true,
}

// httpImporters returns the package directories under root, slash separated
// and relative to root, that import net/http.
func httpImporters(root string) (map[string]bool, error) {
	found := map[string]bool{}
	err := walkGo(root, func(rel, _ string, f *ast.File) error {
		if importsHTTP(f) {
			found[rel] = true
		}
		return nil
	}, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	return found, nil
}

// httpBuilders returns the package directories under root that construct an
// http.Client or use a package-level dialer. Test files are skipped: faking the
// wire is what a test is for.
func httpBuilders(root string) (map[string]bool, error) {
	found := map[string]bool{}
	err := walkGo(root, func(rel, path string, f *ast.File) error {
		if !importsHTTP(f) || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.CompositeLit:
				if isHTTPName(v.Type, "Client") || isHTTPName(v.Type, "Transport") {
					found[rel] = true
				}
			case *ast.SelectorExpr:
				if x, ok := v.X.(*ast.Ident); ok && x.Name == "http" && dialers[v.Sel.Name] {
					found[rel] = true
				}
			}
			return true
		})
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	return found, nil
}

func isHTTPName(e ast.Expr, name string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "http"
}

func importsHTTP(f *ast.File) bool {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) == "net/http" {
			return true
		}
	}
	return false
}

// walkGo parses every .go file under root and hands the visitor the file's
// package directory, relative to root and slash separated.
func walkGo(root string, visit func(rel, path string, f *ast.File) error, mode parser.Mode) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, perr := parser.ParseFile(token.NewFileSet(), path, nil, mode)
		if perr != nil {
			return perr
		}
		rel, rerr := filepath.Rel(root, filepath.Dir(path))
		if rerr != nil {
			return rerr
		}
		return visit(filepath.ToSlash(rel), path, f)
	})
}

func TestNetHTTPStaysInItsRooms(t *testing.T) {
	found, err := httpImporters("..")
	if err != nil {
		t.Fatal(err)
	}
	delete(found, "boundary") // this package's own canary, below

	for pkg := range found {
		if !allowed[pkg] {
			t.Errorf("%s imports net/http; only %v may", pkg, names(allowed))
		}
	}
	for pkg := range allowed {
		if !found[pkg] {
			t.Errorf("allowlisted package %s no longer imports net/http; update the allowlist deliberately", pkg)
		}
	}
}

func TestOnlyTheGuardBuildsTheWire(t *testing.T) {
	found, err := httpBuilders("..")
	if err != nil {
		t.Fatal(err)
	}
	delete(found, "boundary") // the canary again

	for pkg := range found {
		if !builders[pkg] {
			t.Errorf("%s builds an HTTP client or dials directly; only %v may", pkg, names(builders))
		}
	}
	for pkg := range builders {
		if !found[pkg] {
			t.Errorf("%s no longer builds the client; the wire moved and this test did not", pkg)
		}
	}
}

// TestScannerFindsAStrayImport is the canary. Without it the tests above pass
// just as well when the walk finds nothing at all.
func TestScannerFindsAStrayImport(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "stray", "dial.go"), "package stray\n\nimport \"net/http\"\n\nvar _ = http.DefaultClient\n")
	write(t, filepath.Join(root, "clean", "quiet.go"), "package clean\n\nimport \"fmt\"\n\nvar _ = fmt.Sprint\n")
	write(t, filepath.Join(root, "notes.txt"), "net/http\n")

	found, err := httpImporters(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(found); !reflect.DeepEqual(got, []string{"stray"}) {
		t.Errorf("httpImporters found %v, want [stray]", got)
	}
}

// TestScannerTellsNamingFromBuilding is the second canary: a package that only
// names *http.Client in a signature is not a builder, and one that makes a
// client or calls http.Get is.
func TestScannerTellsNamingFromBuilding(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "handed", "handed.go"),
		"package handed\n\nimport \"net/http\"\n\nfunc Use(c *http.Client) *http.Response { return nil }\n")
	write(t, filepath.Join(root, "maker", "maker.go"),
		"package maker\n\nimport \"net/http\"\n\nvar C = &http.Client{}\n")
	write(t, filepath.Join(root, "shortcut", "shortcut.go"),
		"package shortcut\n\nimport \"net/http\"\n\nfunc Fetch() { http.Get(\"https://x\") }\n")

	found, err := httpBuilders(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(found); !reflect.DeepEqual(got, []string{"maker", "shortcut"}) {
		t.Errorf("httpBuilders found %v, want [maker shortcut]", got)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func names(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
