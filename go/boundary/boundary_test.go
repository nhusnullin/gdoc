// The wire checks, the external-program ban, and the dependency list. Each one
// fails in both directions, on spread and on silent disappearance. The reasons
// are in doc.go.

package boundary

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
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
	// internal/gapi builds the requests every read goes out as and sets the
	// bearer on them. It names *http.Request and *http.Client and makes
	// neither: the client comes in as a parameter, and builders below is what
	// proves it.
	"internal/gapi": true,
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

// TestMcpImportsNoNetHTTP holds the one rule that keeps internal/mcp the
// protocol and nothing else. It speaks JSON-RPC over a reader and a writer it
// was handed, knows tools as names and functions, and never learns what a
// document or a request is. It fails in both directions: if the package starts
// importing net/http, and if the package disappears, because then this test
// guards nothing.
func TestMcpImportsNoNetHTTP(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "internal", "mcp")); err != nil {
		t.Fatalf("internal/mcp is not there, so this test guards nothing: %v", err)
	}
	all, _, err := httpImporters("..")
	if err != nil {
		t.Fatal(err)
	}
	if all["internal/mcp"] {
		t.Error("internal/mcp imports net/http; the protocol layer takes a reader and a writer and opens nothing")
	}
	if allowed["internal/mcp"] {
		t.Error("internal/mcp is on the net/http allowlist; take it off, the server has no wire of its own")
	}
}

// markWriters are the three files that may test a text against the robot mark
// with a prefix check of their own, because each is refusing or requiring the
// mark on a caller's own input rather than reading who wrote a thread entry.
// internal/propose and internal/annotate refuse a comment the caller opened
// with the mark; internal/reply requires it on the body it was handed.
var markWriters = map[string]bool{
	"internal/propose/propose.go":   true,
	"internal/annotate/annotate.go": true,
	"internal/reply/reply.go":       true,
}

// markReaders are the package directories that ask who wrote a thread entry.
// Each one asks internal/plaintext, so the day the mark changes there is one
// line to change rather than three answers to one question.
var markReaders = map[string]bool{
	"internal/comments": true,
	"internal/chat":     true,
	"cmd/gdoc":          true,
}

// TestTheRobotMarkIsReadInOnePlace holds the rule internal/plaintext's doc.go
// states: every reader of the mark asks that package rather than writing the
// prefix test out again. The mark is the only record of authorship a thread
// itself carries, so three copies of the check would be three answers to one
// question the day the mark changes.
//
// It fails in both directions. A prefix check against the mark anywhere but the
// three writers above fails, and a listed reader that stops calling
// plaintext.OpensWithRobot fails too, because then the rule has moved and this
// test is guarding nothing.
//
// What counts as naming the mark is the value and never the name, because the
// copy this rule took out of internal/comments was a lowercase robot constant
// of its own: a test matching the names Robot and Prefix would let that same
// line back in, and would fail on an unrelated prefix somebody happens to call
// Prefix. So every constant in the tree whose value is the mark is found first,
// and a bare "🤖" in the call is the mark as much as a constant is.
func TestTheRobotMarkIsReadInOnePlace(t *testing.T) {
	const owner = "internal/plaintext"
	marks, err := markConstants("..")
	if err != nil {
		t.Fatal(err)
	}
	asks := map[string]bool{}
	err = walkGo("..", func(rel, path string, f *ast.File) error {
		where := filepath.ToSlash(filepath.Join(rel, filepath.Base(path)))
		if strings.HasSuffix(path, "_test.go") || rel == owner {
			return nil
		}
		imports := importedPackages(f)
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if pkg.Name == "plaintext" && sel.Sel.Name == "OpensWithRobot" {
				asks[rel] = true
				return true
			}
			if pkg.Name != "strings" || !prefixChecks[sel.Sel.Name] || len(call.Args) != 2 {
				return true
			}
			if !marks.names(call.Args[1], rel, imports) || markWriters[where] {
				return true
			}
			t.Errorf("%s tests a text against the robot mark itself; ask plaintext.OpensWithRobot, "+
				"or add the file to markWriters with the reason", where)
			return true
		})
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	for reader := range markReaders {
		if !asks[reader] {
			t.Errorf("%s no longer calls plaintext.OpensWithRobot; the one check moved and this test did not", reader)
		}
	}
	for where := range markWriters {
		if _, serr := os.Stat(filepath.Join("..", where)); serr != nil {
			t.Errorf("markWriters names %s, which is not there: %v", where, serr)
		}
	}
}

// robotMark is the mark, written out. It is a literal rather than
// plaintext.Robot read back, because what this test asks is which files hold a
// copy of this character, and a test reading the constant would follow it
// wherever somebody moved it.
const robotMark = "🤖"

// prefixChecks are the strings functions that test or take a prefix off a
// text. All three are the same question about the mark asked a different way,
// so all three are the question internal/plaintext owns.
var prefixChecks = map[string]bool{"HasPrefix": true, "CutPrefix": true, "TrimPrefix": true}

// marks is every constant in the tree whose value is the mark: local by package
// directory, so a file can name its package's own constant, and qualified by
// package name, so a file can name another package's exported one.
type marks struct {
	local     map[string]bool // "internal/propose.Robot"
	qualified map[string]bool // "plaintext.Robot"
}

// names answers whether an expression is the mark, from the package directory it
// is written in and that file's imports. A string literal carrying the mark is
// one, a constant whose value is the mark is one, and anything else is not,
// whatever it is called.
func (m marks) names(arg ast.Expr, dir string, imports map[string]string) bool {
	switch v := arg.(type) {
	case *ast.ParenExpr:
		return m.names(v.X, dir, imports)
	case *ast.BinaryExpr:
		return m.names(v.X, dir, imports) || m.names(v.Y, dir, imports)
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return false
		}
		text, err := strconv.Unquote(v.Value)
		return err == nil && strings.Contains(text, robotMark)
	case *ast.Ident:
		return m.local[dir+"."+v.Name]
	case *ast.SelectorExpr:
		q, ok := v.X.(*ast.Ident)
		if !ok {
			return false
		}
		pkg := q.Name
		if named, ok := imports[q.Name]; ok {
			pkg = named
		}
		return m.qualified[pkg+"."+v.Sel.Name]
	}
	return false
}

// markConstants finds every constant and variable in the tree whose value is
// the mark, following one constant to another until nothing new is found: the
// writers hold `const Robot = plaintext.Robot` and `const Prefix = Robot + " "`,
// and both of those are the mark.
func markConstants(root string) (marks, error) {
	type spec struct {
		dir, pkg, name string
		imports        map[string]string
		value          ast.Expr
	}
	var specs []spec
	found := marks{local: map[string]bool{}, qualified: map[string]bool{}}
	err := walkGo(root, func(rel, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		imports := importedPackages(f)
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
				continue
			}
			for _, s := range gen.Specs {
				value, ok := s.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for at, name := range value.Names {
					if at >= len(value.Values) {
						continue
					}
					specs = append(specs, spec{
						dir: rel, pkg: f.Name.Name, name: name.Name,
						imports: imports, value: value.Values[at],
					})
				}
			}
		}
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		return marks{}, err
	}
	for again := true; again; {
		again = false
		for _, s := range specs {
			key := s.dir + "." + s.name
			if found.local[key] || !found.names(s.value, s.dir, s.imports) {
				continue
			}
			found.local[key] = true
			found.qualified[s.pkg+"."+s.name] = true
			again = true
		}
	}
	return found, nil
}

// importedPackages maps the name a file uses for each import onto the package's
// own name, which in this tree is the last element of its path. An import with
// an alias is under the alias, so a file that renames a package is still read.
func importedPackages(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		name := path.Base(p)
		if imp.Name != nil {
			out[imp.Name.Name] = name
			continue
		}
		out[name] = name
	}
	return out
}

// desktopFile is the one file under go/ that may start another program: the
// Claude Desktop extension is installed by handing the file to the application
// that owns it, and on macOS that is /usr/bin/open. Nail's call of 2026-10-03,
// decision 18 of the M14 specification.
const desktopFile = "cmd/gdoc/desktop.go"

// TestNothingRunsAnExternalProgram makes the rule true rather than stating it.
// gdoc runs no external programs but the one above: that is what lets the login
// flow print a URL instead of opening a browser, and it is why the binary is one
// file.
func TestNothingRunsAnExternalProgram(t *testing.T) {
	banned := map[string]bool{"os/exec": true, "syscall/js": true}
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if !banned[p] {
				continue
			}
			if p == "os/exec" && filepath.ToSlash(filepath.Join(rel, filepath.Base(path))) == desktopFile {
				continue
			}
			t.Errorf("%s imports %s; v2 runs no external programs but the one %s runs", path, p, desktopFile)
		}
		return nil
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
}

// TestOnlyDesktopRunsAProgram holds the whole of that one exception by reading
// the syntax tree rather than by trusting the comment above it.
//
// Three things, each of them the narrowness itself. os/exec is imported by that
// one file and by no other, tests included, so no second call site can appear
// anywhere. That file holds exactly one exec.Command call. And that call names
// the opener by its full path, through the file's own constant, with one
// argument after it: a program gdoc starts with a list of words a caller chose
// would be a different thing entirely.
func TestOnlyDesktopRunsAProgram(t *testing.T) {
	const opener = "/usr/bin/open"
	calls := 0
	seen := false
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		where := filepath.ToSlash(filepath.Join(rel, filepath.Base(path)))
		imports := false
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "os/exec" {
				imports = true
			}
		}
		if where != desktopFile {
			if imports {
				t.Errorf("%s imports os/exec; only %s may", where, desktopFile)
			}
			return nil
		}
		seen = true
		if !imports {
			t.Errorf("%s no longer imports os/exec; the one program moved and this test did not", where)
		}
		consts := stringConsts(f)
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "exec" {
				return true
			}
			calls++
			if sel.Sel.Name != "Command" {
				t.Errorf("%s calls exec.%s; the one program is started by exec.Command", where, sel.Sel.Name)
				return true
			}
			if len(call.Args) != 2 || call.Ellipsis.IsValid() {
				t.Errorf("%s runs a program with %d arguments; it is one program and one argument", where, len(call.Args))
				return true
			}
			if got := literalOrConst(call.Args[0], consts); got != opener {
				t.Errorf("%s runs %q; the one program gdoc runs is %s", where, got, opener)
			}
			return true
		})
		return nil
	}, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatalf("%s is not there, so this test guards nothing", desktopFile)
	}
	if calls != 1 {
		t.Errorf("%s holds %d exec calls; one program is started in one place", desktopFile, calls)
	}
}

// stringConsts is the file's own string constants, so a call that names one can
// be read back to the value it holds. The test states the value as a literal and
// the code keeps its constant.
func stringConsts(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			v, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range v.Names {
				if i >= len(v.Values) {
					continue
				}
				if lit, ok := v.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if s, err := strconv.Unquote(lit.Value); err == nil {
						out[name.Name] = s
					}
				}
			}
		}
	}
	return out
}

// literalOrConst is the string a call argument names, written out or held in one
// of the file's constants. Anything else answers the empty string, which fails
// the check above, because a program name this test cannot read is a program
// name nobody reviewing it can read either.
func literalOrConst(arg ast.Expr, consts map[string]string) string {
	switch v := arg.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			if s, err := strconv.Unquote(v.Value); err == nil {
				return s
			}
		}
	case *ast.Ident:
		return consts[v.Name]
	}
	return ""
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
// M2 added the yaml one, and M5 the other two, which is all three SPEC.md
// agreed. A milestone adds its path here and nothing else. It does not delete
// this test, and it does not widen it to "whatever go.mod says". A fourth
// module needs its reason in SPEC.md before its line in this map; the open
// candidate is sergi/go-diff at M8.
var allowedModules = map[string]string{
	// M2 needs it for the gdoc: front-matter block, and M5 for house.yaml. It
	// decodes strictly, which is what a block that must be refused rather than
	// half-read needs, and it carries no transitive modules of its own.
	"github.com/goccy/go-yaml": "the gdoc: front matter and house.yaml; reason in SPEC.md",

	// M5 writes OOXML, and encoding/xml corrupts it: it rewrites namespace
	// prefixes and drops the attribute order Word reads, so a part that went
	// through it comes back as a document Word repairs. etree keeps the tree as
	// it was written.
	"github.com/beevik/etree": "OOXML, because encoding/xml corrupts it; reason in SPEC.md",

	// M5 parses the hub markdown with it: nested lists, bold, italic, links,
	// images in headings, and ==mark== as an extension in about 55 lines. The
	// alternative is a parser written here, in the room where the body's
	// fidelity lives.
	"github.com/yuin/goldmark": "the hub markdown, images in headings included; reason in SPEC.md",
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
