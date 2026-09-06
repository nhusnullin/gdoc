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

// holdsWire reports whether e is a wire type or a container that holds one by
// value. `var c [1]http.Client` is a whole client, and so is every element of
// `make([]http.Client, 1)`, a map value and a channel's element type. A scanner
// that reads only the bare type is walked around by writing one pair of
// brackets in front of it.
//
// A generic instantiation is the same brackets under a name the scanner has
// never heard of. `type Box[T any] struct{ V T }` names no wire, so the type
// declaration is clean, and `var c Box[http.Client]` then builds a whole
// zero-value client out of it. The type arguments are walked for that reason.
//
// A pointer ends the walk. `[]*http.Client` is a list of clients somebody else
// made, which is the same as taking one as a parameter, and that is not
// building the wire.
//
// A function's RESULTS are walked, and its parameters are not. `func Build() (c
// http.Client) { return }` hands back a whole zero-value client that nobody
// gave it, so the function is a factory for the wire; a parameter is a client
// its caller built, which is `handed` again.
func holdsWire(e ast.Expr, names map[string]bool, dot bool) bool {
	switch v := e.(type) {
	case *ast.ArrayType: // [N]http.Client and []http.Client
		return holdsWire(v.Elt, names, dot)
	case *ast.MapType: // map[k]http.Client
		return holdsWire(v.Key, names, dot) || holdsWire(v.Value, names, dot)
	case *ast.ChanType: // chan http.Client
		return holdsWire(v.Value, names, dot)
	case *ast.IndexExpr: // Box[http.Client], one type argument
		return holdsWire(v.Index, names, dot)
	case *ast.IndexListExpr: // Pair[string, http.Client], several
		for _, arg := range v.Indices {
			if holdsWire(arg, names, dot) {
				return true
			}
		}
		return false
	case *ast.FuncType: // func() http.Client, and func() (c http.Client)
		if v.Results == nil {
			return false
		}
		for _, fld := range v.Results.List {
			if holdsWire(fld.Type, names, dot) {
				return true
			}
		}
		return false
	}
	return isWireType(e, names, dot)
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
// Eight ways to build one, and a scanner that knows only the first is a scanner
// that can be walked around: a composite literal, `new(http.Client)`, a
// zero-value declaration `var c http.Client`, the package-level dialers, a type
// declaration that renames the wire (`type C = http.Client`, or the same
// without the equals sign), a struct that holds one by value, a container that
// holds one by value, which is `make([]http.Client, 1)` and every shape
// holdsWire walks, and a function that hands one back by value, which is
// `func Build() (c http.Client) { return }`. Each is checked under every import
// spelling.
//
// A struct field counts whether it is embedded or named. `struct{ C http.Client }`
// is the same zero-value client as `struct{ http.Client }`, reached through one
// extra word, and a scanner that reads only the embedded shape is walked around
// by naming the field.
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
			case *ast.CompositeLit: // http.Client{...}, []http.Client{...}
				if holdsWire(v.Type, names, dot) {
					found[rel] = true
				}
			case *ast.CallExpr: // new(http.Client), make([]http.Client, 1)
				id, ok := v.Fun.(*ast.Ident)
				if ok && (id.Name == "new" || id.Name == "make") && len(v.Args) > 0 && holdsWire(v.Args[0], names, dot) {
					found[rel] = true
				}
			case *ast.ValueSpec: // var c http.Client, var c [1]http.Client
				if v.Type != nil && holdsWire(v.Type, names, dot) {
					found[rel] = true
				}
			case *ast.TypeSpec: // type C = http.Client, type C http.Client
				if holdsWire(v.Type, names, dot) {
					found[rel] = true
				}
			case *ast.StructType: // struct{ http.Client }, struct{ C http.Client }
				for _, fld := range v.Fields.List {
					if holdsWire(fld.Type, names, dot) {
						found[rel] = true
					}
				}
			case *ast.FuncType: // func Build() (c http.Client), and the signature of a func value
				if holdsWire(v, names, dot) {
					found[rel] = true
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

// allowedModules names the third-party modules this tree may require, each
// against the reason it is here. It was empty at M1, and it grows one line per
// milestone that first needs a module.
//
// SPEC.md agreed three, each with its reason written there:
//
//	github.com/beevik/etree   OOXML, because encoding/xml corrupts it
//	github.com/yuin/goldmark  the hub markdown, likely first needed at M5
//	github.com/goccy/go-yaml  the gdoc: front matter and house.yaml
//
// M2 added the third. The milestone that first needs one of the others adds its
// path here and nothing else. It does not delete this test, and it does not
// widen it to "whatever go.mod says". A fourth module needs its reason in
// SPEC.md before its line in this map; the open candidate is sergi/go-diff at
// M8.
var allowedModules = map[string]string{
	// M2 needs it for the gdoc: front-matter block, and M5 for house.yaml. It
	// decodes strictly, which is what a block that must be refused rather than
	// half-read needs, and it carries no transitive modules of its own.
	"github.com/goccy/go-yaml": "the gdoc: front matter and house.yaml; reason in SPEC.md",
}

// TestNoThirdPartyDependencies keeps the module list a property of the tree
// rather than a sentence in a plan. A require line naming something outside
// allowedModules is where that stops being true, and so is a go.sum entry: the
// sum file is what proves something was really fetched.
func TestNoThirdPartyDependencies(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range unlistedModules(requiredModules(string(b))) {
		t.Errorf("go.mod requires %q, which allowedModules does not name. Add it there, with its reason in SPEC.md, or drop the dependency", path)
	}
	sum, err := os.ReadFile(filepath.Join("..", "go.sum"))
	if err != nil {
		return // no go.sum means nothing was fetched
	}
	for _, path := range unlistedModules(summedModules(string(sum))) {
		t.Errorf("go.sum names %q, so it was fetched, and allowedModules does not name it", path)
	}
}

// TestAllowedModulesAreReallyRequired is the disappearance half. A module that
// stops being used has to leave this map too: an allowlist naming something the
// tree no longer requires is a door standing open for no reason.
func TestAllowedModulesAreReallyRequired(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	required := map[string]bool{}
	for _, path := range requiredModules(string(b)) {
		required[path] = true
	}
	for path := range allowedModules {
		if !required[path] {
			t.Errorf("allowedModules names %q, which go.mod no longer requires. Drop it from the map", path)
		}
	}
}

// unlistedModules names the paths allowedModules does not carry, in the order
// they were read and with each path named once. The judgement lives here rather
// than inside the test above so a canary can put scratch text through it: once
// a module is both listed and required, a test that only reads the real files
// passes whether the refusal still works or not.
func unlistedModules(paths []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, path := range paths {
		if _, ok := allowedModules[path]; ok {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}

// summedModules reads the module paths out of a go.sum. Every module has two
// lines there, the zip hash and the go.mod hash, so a path is returned once.
func summedModules(sum string) []string {
	var paths []string
	seen := map[string]bool{}
	for _, line := range strings.Split(sum, "\n") {
		path, _, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok || path == "" {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths
}

// requiredModules reads the module paths out of a go.mod, in both spellings:
// the one-line `require path version` and the parenthesised block. It reads the
// text rather than running `go list`, so the test says what the file says even
// when nothing was ever downloaded.
func requiredModules(mod string) []string {
	var paths []string
	inBlock := false
	for _, raw := range strings.Split(mod, "\n") {
		line := strings.TrimSpace(raw)
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if path, _, ok := strings.Cut(line, " "); ok && path != "" {
				paths = append(paths, path)
			}
			continue
		}
		rest, ok := strings.CutPrefix(line, "require")
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)
		if rest == "(" {
			inBlock = true
			continue
		}
		if path, _, ok := strings.Cut(rest, " "); ok && path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

// TestRequiredModulesReadsBothSpellings is the canary for the parser above.
// Without it an allowlist that never sees a require line reads exactly like a
// tree that has no dependencies.
func TestRequiredModulesReadsBothSpellings(t *testing.T) {
	mod := "module gdoc\n\ngo 1.27\n\nrequire github.com/one/alpha v1.0.0\n\nrequire (\n\tgithub.com/two/beta v2.0.0 // indirect\n\tgithub.com/three/gamma v3.0.0\n)\n"
	got := requiredModules(mod)
	want := []string{"github.com/one/alpha", "github.com/two/beta", "github.com/three/gamma"}
	if len(got) != len(want) {
		t.Fatalf("requiredModules read %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("requiredModules[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if len(requiredModules("module gdoc\n\ngo 1.27\n")) != 0 {
		t.Error("a go.mod with no require line reads as a dependency")
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
	// A container of POINTERS is a list of clients somebody else made, which is
	// `handed` again with one more layer.
	write(t, filepath.Join(root, "ptrslice", "ptrslice.go"),
		"package ptrslice\n\nimport \"net/http\"\n\nvar C []*http.Client\n")
	// A pointer type argument is a client somebody else made, so the walk stops
	// at it here exactly as it does in ptrslice.
	write(t, filepath.Join(root, "genericptr", "genericptr.go"),
		"package genericptr\n\nimport \"net/http\"\n\ntype Box[T any] struct{ V T }\n\nvar C Box[*http.Client]\n")
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
	// Naming the field changes nothing: a value of T still holds a whole client
	// that the guard never made.
	write(t, filepath.Join(root, "namedfield", "namedfield.go"),
		"package namedfield\n\nimport \"net/http\"\n\ntype T struct{ C http.Client }\n")
	// A container holds the client by value the same way a struct field does.
	// `var c [1]http.Client` is one whole client, and make() builds as many as
	// it is asked for, so a scanner that reads only the bare type is walked
	// around by wrapping the type in one bracket.
	write(t, filepath.Join(root, "arrayvar", "arrayvar.go"),
		"package arrayvar\n\nimport \"net/http\"\n\nvar C [1]http.Client\n")
	write(t, filepath.Join(root, "slicemake", "slicemake.go"),
		"package slicemake\n\nimport \"net/http\"\n\nvar C = make([]http.Client, 1)\n")
	write(t, filepath.Join(root, "slicelit", "slicelit.go"),
		"package slicelit\n\nimport \"net/http\"\n\nvar C = []http.Client{{}}\n")
	write(t, filepath.Join(root, "mapvalue", "mapvalue.go"),
		"package mapvalue\n\nimport \"net/http\"\n\nvar C map[string]http.Client\n")
	write(t, filepath.Join(root, "chanelem", "chanelem.go"),
		"package chanelem\n\nimport \"net/http\"\n\nvar C chan http.Transport\n")
	write(t, filepath.Join(root, "nestedfield", "nestedfield.go"),
		"package nestedfield\n\nimport \"net/http\"\n\ntype T struct{ C []http.Client }\n")
	write(t, filepath.Join(root, "containertype", "containertype.go"),
		"package containertype\n\nimport \"net/http\"\n\ntype C []http.Client\n")
	// A function that produces a wire by value is a factory for one, whether the
	// result is named or not. `func Build() (c http.Client) { return }` returns a
	// whole zero-value client that nobody handed over, and it is the shape a
	// scanner reading only declarations and literals never sees.
	write(t, filepath.Join(root, "namedresult", "namedresult.go"),
		"package namedresult\n\nimport \"net/http\"\n\nfunc Build() (c http.Client) { return }\n")
	write(t, filepath.Join(root, "bareresult", "bareresult.go"),
		"package bareresult\n\nimport \"net/http\"\n\nfunc Build() http.Transport { panic(\"unused\") }\n")
	write(t, filepath.Join(root, "resultslice", "resultslice.go"),
		"package resultslice\n\nimport \"net/http\"\n\nfunc Build() (cs []http.Client) { return }\n")
	// A method result is the same factory reached through a receiver.
	write(t, filepath.Join(root, "methodresult", "methodresult.go"),
		"package methodresult\n\nimport \"net/http\"\n\ntype F struct{}\n\nfunc (F) Build() (c http.Client) { return }\n")
	// A generic instantiation hides the wire behind one pair of brackets, the
	// same move as the array above. `Box[http.Client]` is a Box holding a whole
	// zero-value client, and the type argument is where that client is named.
	write(t, filepath.Join(root, "generic", "generic.go"),
		"package generic\n\nimport \"net/http\"\n\ntype Box[T any] struct{ V T }\n\nvar C Box[http.Client]\n")
	// Several type arguments is the same shape under a different AST node.
	write(t, filepath.Join(root, "genericlist", "genericlist.go"),
		"package genericlist\n\nimport \"net/http\"\n\ntype Pair[A any, B any] struct {\n\tFirst  A\n\tSecond B\n}\n\nvar C Pair[string, http.Client]\n")

	found, err := httpBuilders(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"aliased", "aliasedcall", "anonembed", "arrayvar", "bareresult", "chanelem",
		"containertype", "definedtype", "dotted", "embedded", "generic", "genericlist", "maker",
		"mapvalue", "methodresult", "namedfield", "namedresult", "nestedfield", "newed",
		"resultslice", "roundtripper", "shortcut", "slicelit", "slicemake", "typealias", "zero"}
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

// TestAllowedModulesStillRefusesAnUnlistedPath is the canary for the allowlist
// itself. TestNoThirdPartyDependencies reads the real files, so once a module
// is listed and required, that test passes whether the refusal still works or
// not. This one judges scratch text: the listed path is carried, and an
// unlisted one beside it is still named as a refusal.
func TestAllowedModulesStillRefusesAnUnlistedPath(t *testing.T) {
	mod := "module gdoc\n\ngo 1.27\n\nrequire (\n\tgithub.com/goccy/go-yaml v1.19.2\n\tgithub.com/sergi/go-diff v1.3.1\n)\n"
	got := unlistedModules(requiredModules(mod))
	if !reflect.DeepEqual(got, []string{"github.com/sergi/go-diff"}) {
		t.Errorf("unlistedModules read %v, want [github.com/sergi/go-diff]", got)
	}

	sum := "github.com/goccy/go-yaml v1.19.2 h1:abc=\ngithub.com/sergi/go-diff v1.3.1/go.mod h1:def=\n"
	if got := unlistedModules(summedModules(sum)); !reflect.DeepEqual(got, []string{"github.com/sergi/go-diff"}) {
		t.Errorf("unlistedModules over go.sum read %v, want [github.com/sergi/go-diff]", got)
	}

	if len(unlistedModules(requiredModules("module gdoc\n\ngo 1.27\n"))) != 0 {
		t.Error("a go.mod with no require line reads as an unlisted dependency")
	}
}
