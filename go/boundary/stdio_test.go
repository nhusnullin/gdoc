// The one room that names the real streams. The reason is in doc.go.

package boundary

import (
	"go/ast"
	"go/parser"
	"path/filepath"
	"strings"
	"testing"
)

// theStdioRoom is the one file under go/ that may name os.Stdin or os.Stdout,
// written as a path a reader can open.
const theStdioRoom = "cmd/gdoc/main.go"

// stdioNames are the two streams this test is about. os.Stderr is not here:
// human words go to stderr from several rooms by design, the login URL among
// them, and nothing a caller parses arrives on it.
var stdioNames = map[string]bool{"Stdin": true, "Stdout": true}

// TestOnlyMainNamesStdinAndStdout holds two rules at once, and closes the
// TODO(test) that stood in cmd/gdoc/doc.go until gdoc mcp arrived.
//
// The first is the older one: the binary never reads stdin. It takes facts as
// arguments and prints one object, so nothing it does waits on a pipe nobody
// filled, and no command can start asking a question. Until now that held by
// absence, since no file named os.Stdin at all.
//
// The second is what gdoc mcp needs. A JSON-RPC session is a stream rather
// than one object, and it reads stdin and writes stdout for as long as Claude
// Desktop is open. One room names those two streams and hands every other room
// an io.Reader and an io.Writer, so internal/mcp runs a whole session against
// strings in memory and no package below cmd/gdoc can print a line that lands
// in the middle of somebody's protocol.
//
// It reads the syntax tree rather than the text, so a comment naming os.Stdin
// is prose and not a use, which is what lets doc.go explain the rule. Test
// files are not judged: a test that captures os.Stdout to watch a writer is
// about that writer and reaches nobody's session.
//
// It fails in both directions. A second room naming either stream fails, and
// so does the one room no longer naming them, because then the session has
// moved and this test is guarding an empty rule.
func TestOnlyMainNamesStdinAndStdout(t *testing.T) {
	named := map[string]bool{}
	err := walkGo("..", func(_, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		osNames, dot := importRefs(f, "os")
		if len(osNames) == 0 && !dot {
			return nil
		}
		rel, rerr := filepath.Rel("..", path)
		if rerr != nil {
			return rerr
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.SelectorExpr: // os.Stdin, os.Stdout
				if x, ok := v.X.(*ast.Ident); ok && osNames[x.Name] && stdioNames[v.Sel.Name] {
					named[filepath.ToSlash(rel)] = true
				}
			case *ast.Ident: // Stdin, Stdout, under a dot import
				if dot && stdioNames[v.Name] {
					named[filepath.ToSlash(rel)] = true
				}
			}
			return true
		})
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}

	for file := range named {
		if file != theStdioRoom {
			t.Errorf("%s names os.Stdin or os.Stdout; only %s may, and every other room is handed a reader and a writer",
				file, theStdioRoom)
		}
	}
	if !named[theStdioRoom] {
		t.Errorf("%s no longer names os.Stdin or os.Stdout; the streams moved and this test did not", theStdioRoom)
	}
}

// importRefs is the name or names a file can reach one standard package by,
// and whether it dot-imported it. It is httpRefs over any path, so a scanner
// that assumed the import name is always the last element of the path is not
// walked around by import o "os".
func importRefs(f *ast.File, path string) (names map[string]bool, dot bool) {
	names = map[string]bool{}
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != path {
			continue
		}
		if imp.Name == nil {
			names[path[strings.LastIndex(path, "/")+1:]] = true
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
