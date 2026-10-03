// The one room that writes an escape byte. The reason is in doc.go.

package boundary

import (
	"go/ast"
	"go/parser"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// theEscapeRoom is the package directory, relative to the module root, whose
// files may hold an escape byte. internal/tty holds the palette and every
// cursor code; internal/panel takes a Depth from it and draws boxes without
// naming a byte.
const theEscapeRoom = "internal/tty"

// escapeSpellings are the ways a Go literal writes the escape byte, as a reader
// meets them in source. The test does not match on these: it unquotes the
// literal and looks for the byte itself, which catches every spelling and every
// letter case at once, these three included. They are here so the message can
// name what a person should go and look for.
var escapeSpellings = []string{`\x1b`, `\033`, `\u001b`}

// TestNoEscapeLiteralOutsideTTY holds that one package writes to the terminal
// in the terminal's own language.
//
// A screen that moves the cursor, paints a title or turns auto-wrap off is a
// stream of escape sequences, and a sequence written in the wrong place is the
// bug that is hardest to see: a stray colour leaks onto the shell prompt under
// the screen, a miscounted row makes a redraw crawl down the screen, and a byte
// that reaches a pipe reaches a skill's parser. Keeping every one of them in
// internal/tty means the palette has one definition, NoColour really is no
// escape byte anywhere, and a boundary test rather than a reviewer holds it.
//
// It reads the syntax tree and not the text, so the prose in this file and in
// progress.go may spell a code out: a comment is not a literal. Test files are
// not judged either, because a test that pins the bytes a role paints with has
// to state them, which is the house rule that a test states its value as a
// literal rather than reading the constant it checks.
//
// It fails in both directions. A literal outside internal/tty fails, and so
// does internal/tty holding none, because then the codes have moved and this
// test is guarding an empty rule.
func TestNoEscapeLiteralOutsideTTY(t *testing.T) {
	holding := map[string]bool{}
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || !strings.ContainsRune(unquoted(lit.Value), 0x1b) {
				return true
			}
			holding[filepath.ToSlash(rel)] = true
			return true
		})
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}

	for pkg := range holding {
		if pkg != theEscapeRoom {
			t.Errorf("%s holds an escape byte (%s); only %s may, and every other room asks it for a role or a code",
				pkg, strings.Join(escapeSpellings, ", "), theEscapeRoom)
		}
	}
	if !holding[theEscapeRoom] {
		t.Errorf("%s holds no escape byte any more; the codes moved and this test did not", theEscapeRoom)
	}
}

// unquoted is what a literal's bytes are. A literal Go cannot unquote, which is
// an untyped number or an imaginary, has its source text read instead, so the
// walk always has something to search and never passes a byte through on an
// error.
func unquoted(text string) string {
	if s, err := strconv.Unquote(text); err == nil {
		return s
	}
	return text
}
