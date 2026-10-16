// The guard over the one file gdoc appends to and never reads.
//
// internal/presence writes a record for another program. The whole safety of
// that design is that it is one way: the binary appends to it, nothing in the
// binary reads it back, and no decision gdoc makes depends on a line in it.
// These tests are where the one way is measured, because a reader added later
// would be a second source of truth for holds and cursors and nothing inside
// the package would notice.

package boundary

import (
	"go/ast"
	"go/parser"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// presencePkg is the one room, written out rather than derived, so a package
// that moves is a failure somebody reads.
const presencePkg = "internal/presence"

// presenceClosedTo are the imports that would turn the record into something
// other than a file: a wire, a program, the chat session's own state, or the
// authenticated session. None of them has anything to say to an append.
var presenceClosedTo = []string{
	"net",
	"net/http",
	"os/exec",
	"gdoc/internal/chat",
	"gdoc/internal/gapi",
}

// presenceReaders are the two calls that would read a file's contents. The
// package may still call os.OpenFile, which is the append itself, and
// os.ReadDir, which is the sweep over names: neither reads a line.
var presenceReaders = []string{"Open", "ReadFile"}

// TestOnlyPresenceAppends holds that the append-only file has one writer.
//
// O_APPEND is the whole mechanism: it is what lets a terminal run and a chat
// session write the same file with no lock and no torn line. A second room
// opening a file that way would be a second record nobody agreed on, written
// in a shape nobody pinned, so the flag belongs to internal/presence and to
// nothing else. It fails in both directions: a flag that appears elsewhere,
// and a package that stops using it, because then this test guards nothing.
func TestOnlyPresenceAppends(t *testing.T) {
	found := map[string]bool{}
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.SelectorExpr:
				if v.Sel.Name == "O_APPEND" {
					found[rel] = true
				}
			case *ast.Ident:
				if v.Name == "O_APPEND" {
					found[rel] = true
				}
			}
			return true
		})
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	for pkg := range found {
		if pkg != presencePkg {
			t.Errorf("%s opens a file with O_APPEND; only %s may, because the record has one writer", pkg, presencePkg)
		}
	}
	if !found[presencePkg] {
		t.Errorf("%s no longer appends; the record moved and this test did not", presencePkg)
	}
}

// TestPresenceOpensNoWireAndRunsNothing holds what the package is allowed to
// know. It is a file and a clock. It reaches no network, starts no program,
// and does not know what a chat hold or an authenticated session is: the
// callers hand it the few words an event carries and it writes them down.
func TestPresenceOpensNoWireAndRunsNothing(t *testing.T) {
	dir := filepath.Join("..", filepath.FromSlash(presencePkg))
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("%s is not there, so this test guards nothing: %v", presencePkg, err)
	}
	closed := map[string]bool{}
	for _, name := range presenceClosedTo {
		closed[name] = true
	}
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		if rel != presencePkg || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		for _, spec := range f.Imports {
			name := strings.Trim(spec.Path.Value, `"`)
			if closed[name] {
				t.Errorf("%s imports %s; the record is a file and a clock, and nothing else", filepath.ToSlash(path), name)
			}
		}
		return nil
	}, parser.ImportsOnly|parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
}

// TestPresenceExportsNoReader holds the one way of the channel.
//
// A reader here would be read by somebody, and the moment a hold rule, a
// receipt or a cursor asked the file a question, a line somebody edited or a
// file half written by a disk that filled would change what gdoc does. There
// is no exported Read or Open, and nothing inside reads a file's contents
// either, so the rule holds for the package and not only for its callers.
func TestPresenceExportsNoReader(t *testing.T) {
	var readers, opens []string
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		if rel != presencePkg || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		where := filepath.ToSlash(strings.TrimPrefix(path, "../"))
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() {
				continue
			}
			if strings.HasPrefix(fn.Name.Name, "Read") || strings.HasPrefix(fn.Name.Name, "Open") {
				readers = append(readers, fn.Name.Name+" ("+where+")")
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			x, ok := sel.X.(*ast.Ident)
			if !ok || x.Name != "os" {
				return true
			}
			for _, name := range presenceReaders {
				if sel.Sel.Name == name {
					opens = append(opens, "os."+name+" ("+where+")")
				}
			}
			return true
		})
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(readers)
	sort.Strings(opens)
	if len(readers) != 0 {
		t.Errorf("%s exports %v; the binary never reads the record it writes", presencePkg, readers)
	}
	if len(opens) != 0 {
		t.Errorf("%s calls %v; the record is appended to and never read back", presencePkg, opens)
	}
}

// presenceBegin is the hook, and the room and function the one call sits in.
const (
	presenceBegin = "Begin"
	beginRoom     = "cmd/gdoc/main.go"
	beginCaller   = "dispatch"
)

// TestBeginHasOneCaller holds that the hook is one hook.
//
// Every command's start and end is written from dispatch, after the parse and
// on a deferred end, so a command that is added is recorded without its author
// remembering to record it, and a panic that safeDispatch recovers still ends
// its run. A second call anywhere would mean one command writing two starts or
// a command writing one the table never asked for, and the pairing of a start
// with its end is what the app reads. It sits beside TestAccountOfHasTwoCallers
// for the same reason: a call that moves is a failure a person reads.
func TestBeginHasOneCaller(t *testing.T) {
	found, err := callsOf(presenceBegin)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].file != beginRoom || found[0].fn != beginCaller {
		t.Errorf("presence.%s is called from %s's %s and nowhere else, and this tree calls it from %v",
			presenceBegin, beginRoom, beginCaller, found)
	}
}

// TestNoSkillPassesThePresenceFlag holds the flag out of the five skills.
//
// --presence is experimental: one person turns it on for their own machine,
// in their own settings file or on their own command line. A skill that typed
// it would turn it on for every colleague the plugin reaches, which is the
// opposite of what experimental means, and it would do it in a file nobody
// reads before the session runs it.
func TestNoSkillPassesThePresenceFlag(t *testing.T) {
	root := filepath.Join(repoRoot, "skills")
	seen := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		seen++
		if strings.Contains(string(b), "--presence") {
			t.Errorf("%s passes --presence; an experimental flag is one person's own, and a skill is every colleague's", filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen == 0 {
		t.Fatal("no files under skills/. A moved skills directory is a failure, not an empty pass")
	}
}

// swiftFixture and goFixture are the two copies of the one scripted run, each
// written out rather than derived, so a file that moves is a failure a person
// reads rather than a test that quietly passes over nothing.
const (
	goFixture    = "go/internal/presence/testdata/events-v1.jsonl"
	swiftFixture = "presence/Tests/PresenceCoreTests/Fixtures/events-v1.jsonl"
)

// TestTheSwiftFixtureIsTheGoFixture holds the schema still across two
// languages.
//
// The Go writer writes the bytes and a Go test pins them; the presence app
// decodes the same bytes in Swift. Two copies of a format is how a field
// quietly means one thing on one side and another on the other, so the copies
// are one file: byte for byte, including the trailing newline and the order of
// the keys. A change to the event is then a change both sides see at once.
func TestTheSwiftFixtureIsTheGoFixture(t *testing.T) {
	want, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(goFixture)))
	if err != nil {
		t.Fatalf("%s: %v", goFixture, err)
	}
	got, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(swiftFixture)))
	if err != nil {
		t.Fatalf("%s: %v", swiftFixture, err)
	}
	if string(got) != string(want) {
		t.Errorf("%s is not %s byte for byte; the schema would drift between Go and Swift.\nswift: %q\ngo:    %q",
			swiftFixture, goFixture, got, want)
	}
}
