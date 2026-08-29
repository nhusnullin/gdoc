// Package boundary enforces that gdoc has exactly one wire, runs no external
// programs, and depends on nothing outside the standard library. This is the v1
// allowlist test, ported: the wire checks fail in BOTH directions, on spread
// and on silent disappearance.
//
// Two wire checks, because importing net/http and dialing with it are not the
// same thing. The import allowlist says which packages may name the type at
// all. The builder allowlist says who may make a client out of it, and that is
// the single room the guard occupies.
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
// below proves it never makes one. `internal/auth/loopback` is here because the
// login flow parks the browser redirect on a localhost listener; it serves and
// never dials, so it is not a builder. A room is listed only once it exists,
// because this test has to be green at every commit.
var allowed = map[string]bool{
	"internal/guard":         true,
	"internal/auth":          true,
	"internal/auth/loopback": true,
}

// builders lists the package directories that may construct an HTTP client or
// reach the package-level dialing helpers. Exactly one, and it stays one.
// Serving is not building: an http.Server answers requests somebody else made,
// so the loopback listener is deliberately absent from this set.
var builders = map[string]bool{
	"internal/guard": true,
}

// dialers are the net/http names that reach the network on their own, without
// a client anybody handed over.
var dialers = map[string]bool{
	"DefaultClient": true, "DefaultTransport": true,
	"Get": true, "Post": true, "PostForm": true, "Head": true,
}

// wireTypes are the net/http types that are a wire once a value of one exists.
var wireTypes = map[string]bool{"Client": true, "Transport": true}

// httpRefs reports the identifiers that stand for net/http in this file, and
// whether the file dot-imported it. Assuming the name is always "http" is how a
// scanner misses `import nh "net/http"` and `import . "net/http"`, which build
// a wire just as well as the canonical spelling does.
func httpRefs(f *ast.File) (names map[string]bool, dot bool) {
	names = map[string]bool{}
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != "net/http" {
			continue
		}
		if imp.Name == nil {
			names["http"] = true
			continue
		}
		switch imp.Name.Name {
		case ".":
			dot = true
		case "_": // a blank import cannot name anything
		default:
			names[imp.Name.Name] = true
		}
	}
	return names, dot
}

// isWireType reports whether e names http.Client or http.Transport, under any
// spelling this file's imports allow.
func isWireType(e ast.Expr, names map[string]bool, dot bool) bool {
	switch v := e.(type) {
	case *ast.SelectorExpr:
		x, ok := v.X.(*ast.Ident)
		return ok && names[x.Name] && wireTypes[v.Sel.Name]
	case *ast.Ident:
		return dot && wireTypes[v.Name]
	}
	return false
}

// httpImporters returns the package directories under root, slash separated
// and relative to root, that import net/http. Two sets, and the difference is
// the point. `all` counts every file, because a stray import is worth flagging
// wherever it sits. `prod` counts only non-test files, because a test file that
// fakes the wire must never stand in for the room that owns it: without the
// split, a package could quietly stop dialing and its own test file would keep
// the allowlist looking satisfied.
func httpImporters(root string) (all, prod map[string]bool, err error) {
	all, prod = map[string]bool{}, map[string]bool{}
	err = walkGo(root, func(rel, path string, f *ast.File) error {
		names, dot := httpRefs(f)
		if len(names) == 0 && !dot {
			return nil
		}
		all[rel] = true
		if !strings.HasSuffix(path, "_test.go") {
			prod[rel] = true
		}
		return nil
	}, parser.ImportsOnly)
	if err != nil {
		return nil, nil, err
	}
	return all, prod, nil
}

// httpBuilders returns the package directories under root that construct an
// http.Client or http.Transport, or use a package-level dialer. Test files are
// skipped: faking the wire is what a test is for.
//
// Six ways to build one, and a scanner that knows only the first is a scanner
// that can be walked around: a composite literal, `new(http.Client)`, a
// zero-value declaration `var c http.Client`, the package-level dialers, a type
// declaration that renames the wire (`type C = http.Client`, or the same
// without the equals sign), and a struct that embeds one by value. Each is
// checked under every import spelling.
//
// The two type shapes are flagged on the declaration rather than on the use.
// Following an alias to the values built from it would mean resolving names
// across a package, and a package outside the guard has no reason to give the
// wire a second name in the first place. Embedding a POINTER is not flagged:
// that is holding a client somebody else made, which is what `handed` does.
func httpBuilders(root string) (map[string]bool, error) {
	found := map[string]bool{}
	err := walkGo(root, func(rel, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		names, dot := httpRefs(f)
		if len(names) == 0 && !dot {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.CompositeLit: // http.Client{...}
				if isWireType(v.Type, names, dot) {
					found[rel] = true
				}
			case *ast.CallExpr: // new(http.Client)
				id, ok := v.Fun.(*ast.Ident)
				if ok && id.Name == "new" && len(v.Args) == 1 && isWireType(v.Args[0], names, dot) {
					found[rel] = true
				}
			case *ast.ValueSpec: // var c http.Client
				if v.Type != nil && isWireType(v.Type, names, dot) {
					found[rel] = true
				}
			case *ast.TypeSpec: // type C = http.Client, type C http.Client
				if isWireType(v.Type, names, dot) {
					found[rel] = true
				}
			case *ast.StructType: // struct{ http.Client }
				for _, fld := range v.Fields.List {
					if len(fld.Names) == 0 && isWireType(fld.Type, names, dot) {
						found[rel] = true
					}
				}
			case *ast.SelectorExpr: // http.Get, http.DefaultClient
				if x, ok := v.X.(*ast.Ident); ok && names[x.Name] && dialers[v.Sel.Name] {
					found[rel] = true
				}
			case *ast.Ident: // Get, DefaultClient, under a dot import
				if dot && dialers[v.Name] {
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
	all, prod, err := httpImporters("..")
	if err != nil {
		t.Fatal(err)
	}
	for pkg := range all {
		if !allowed[pkg] {
			t.Errorf("%s imports net/http; only %v may", pkg, names(allowed))
		}
	}
	for pkg := range allowed {
		if !prod[pkg] {
			t.Errorf("allowlisted package %s no longer imports net/http outside its tests; update the allowlist deliberately", pkg)
		}
	}
}

func TestOnlyTheGuardBuildsTheWire(t *testing.T) {
	found, err := httpBuilders("..")
	if err != nil {
		t.Fatal(err)
	}
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

// TestNothingRunsAnExternalProgram is v1's rule, made true rather than stated.
// v2 runs no external programs at all: that is what lets the login flow print a
// URL instead of opening a browser, and it is why the binary is one file.
func TestNothingRunsAnExternalProgram(t *testing.T) {
	banned := map[string]bool{"os/exec": true, "syscall/js": true}
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		for _, imp := range f.Imports {
			if p := strings.Trim(imp.Path.Value, `"`); banned[p] {
				t.Errorf("%s imports %s; v2 runs no external programs", path, p)
			}
		}
		return nil
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
}

// TestEveryTargetThatWritesIntoBinMakesIt: bin/ is ignored and nothing tracks
// it, so a fresh clone does not have one. A target that writes a binary there
// without making the directory first fails before it produces anything, and it
// fails only on the machine that has never built before.
func TestEveryTargetThatWritesIntoBinMakesIt(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	target, writes, makes := "", false, false
	check := func() {
		if writes && !makes {
			t.Errorf("the %s target writes into bin/ without creating it; a fresh clone has no bin/", target)
		}
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "\t") { // a recipe line
			if strings.Contains(line, "bin/") {
				writes = true
			}
			if strings.Contains(line, "mkdir") && strings.Contains(line, "bin") {
				makes = true
			}
			continue
		}
		check()
		target, writes, makes = strings.TrimSpace(strings.SplitN(line, ":", 2)[0]), false, false
	}
	check()
}

// TestNoThirdPartyDependencies keeps "standard library only" a property of the
// tree rather than a sentence in a plan. A require block is where that stops
// being true.
func TestNoThirdPartyDependencies(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "require") {
			t.Errorf("go.mod requires something: %q. v2 is standard library only", line)
		}
	}
	if _, err := os.Stat(filepath.Join("..", "go.sum")); err == nil {
		t.Error("go.sum exists, so something outside the standard library was fetched")
	}
}

// TestScannerFindsAStrayImport is the canary. Without it the tests above pass
// just as well when the walk finds nothing at all.
func TestScannerFindsAStrayImport(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "stray", "dial.go"), "package stray\n\nimport \"net/http\"\n\nvar _ = http.DefaultClient\n")
	write(t, filepath.Join(root, "clean", "quiet.go"), "package clean\n\nimport \"fmt\"\n\nvar _ = fmt.Sprint\n")
	write(t, filepath.Join(root, "notes.txt"), "net/http\n")
	// A package whose only net/http import is in a test file. It counts as a
	// stray, and it must not count as a room that still owns the wire.
	write(t, filepath.Join(root, "faker", "faker_test.go"), "package faker\n\nimport \"net/http\"\n\nvar _ http.RoundTripper\n")
	// Renamed and dot imports are the same import.
	write(t, filepath.Join(root, "aliased", "aliased.go"), "package aliased\n\nimport nh \"net/http\"\n\nfunc Use(c *nh.Client) {}\n")

	all, prod, err := httpImporters(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(all); !reflect.DeepEqual(got, []string{"aliased", "faker", "stray"}) {
		t.Errorf("httpImporters all found %v, want [aliased faker stray]", got)
	}
	if got := names(prod); !reflect.DeepEqual(got, []string{"aliased", "stray"}) {
		t.Errorf("httpImporters prod found %v, want [aliased stray]; a test file must not stand in for production code", got)
	}
}

// TestScannerTellsNamingFromBuilding is the second canary: a package that only
// names *http.Client in a signature is not a builder, one that serves is not a
// builder either, and one that makes a client by any spelling is.
//
// The five build cases below are not decoration. Every one of them was checked
// against this scanner and, before the fix, the aliased import, the dot import,
// new(http.Client) and the zero-value declaration all came back clean while
// building a wire outside the guard.
func TestScannerTellsNamingFromBuilding(t *testing.T) {
	root := t.TempDir()
	// Not builders.
	write(t, filepath.Join(root, "handed", "handed.go"),
		"package handed\n\nimport \"net/http\"\n\nfunc Use(c *http.Client) *http.Response { return nil }\n")
	write(t, filepath.Join(root, "server", "server.go"),
		"package server\n\nimport \"net/http\"\n\nvar S = &http.Server{Handler: http.NewServeMux()}\n")
	write(t, filepath.Join(root, "pointer", "pointer.go"),
		"package pointer\n\nimport \"net/http\"\n\nvar C *http.Client\n")
	// Embedding a POINTER is being handed one, the same as the field above.
	write(t, filepath.Join(root, "embedptr", "embedptr.go"),
		"package embedptr\n\nimport \"net/http\"\n\ntype T struct{ *http.Client }\n")
	write(t, filepath.Join(root, "faketest", "wire_test.go"),
		"package faketest\n\nimport \"net/http\"\n\nvar C = &http.Client{}\n")
	// Builders, one spelling each.
	write(t, filepath.Join(root, "maker", "maker.go"),
		"package maker\n\nimport \"net/http\"\n\nvar C = &http.Client{}\n")
	write(t, filepath.Join(root, "shortcut", "shortcut.go"),
		"package shortcut\n\nimport \"net/http\"\n\nfunc Fetch() { http.Get(\"https://x\") }\n")
	write(t, filepath.Join(root, "aliased", "aliased.go"),
		"package aliased\n\nimport nh \"net/http\"\n\nvar C = &nh.Client{}\n")
	write(t, filepath.Join(root, "aliasedcall", "aliasedcall.go"),
		"package aliasedcall\n\nimport nh \"net/http\"\n\nfunc Fetch() { nh.Get(\"https://x\") }\n")
	write(t, filepath.Join(root, "dotted", "dotted.go"),
		"package dotted\n\nimport . \"net/http\"\n\nvar C = &Client{}\n")
	write(t, filepath.Join(root, "newed", "newed.go"),
		"package newed\n\nimport \"net/http\"\n\nvar C = new(http.Client)\n")
	write(t, filepath.Join(root, "zero", "zero.go"),
		"package zero\n\nimport \"net/http\"\n\nvar C http.Client\n")
	write(t, filepath.Join(root, "roundtripper", "roundtripper.go"),
		"package roundtripper\n\nimport \"net/http\"\n\nvar T = &http.Transport{}\n")
	// A type alias renames the wire, and every spelling above then works again
	// under a name the scanner has never heard of.
	write(t, filepath.Join(root, "typealias", "typealias.go"),
		"package typealias\n\nimport \"net/http\"\n\ntype C = http.Client\n\nvar _ = &C{}\n")
	// A defined type is the same move without the equals sign.
	write(t, filepath.Join(root, "definedtype", "definedtype.go"),
		"package definedtype\n\nimport \"net/http\"\n\ntype C http.Client\n")
	// Embedding a VALUE puts a whole client inside another type, and a value of
	// that type is a wire nobody handed over.
	write(t, filepath.Join(root, "embedded", "embedded.go"),
		"package embedded\n\nimport \"net/http\"\n\ntype T struct{ http.Client }\n")
	write(t, filepath.Join(root, "anonembed", "anonembed.go"),
		"package anonembed\n\nimport \"net/http\"\n\nvar V = struct{ http.Transport }{}\n")

	found, err := httpBuilders(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"aliased", "aliasedcall", "anonembed", "definedtype", "dotted", "embedded",
		"maker", "newed", "roundtripper", "shortcut", "typealias", "zero"}
	if got := names(found); !reflect.DeepEqual(got, want) {
		t.Errorf("httpBuilders found %v, want %v", got, want)
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
