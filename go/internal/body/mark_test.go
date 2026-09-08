package body

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// TestAMarkedSpanIsItsOwnNodeKind states what ==marked== parses into. The two
// methods here are ast.Node's, so goldmark calls them rather than this package,
// and without a test they are an interface satisfied by nobody's measurement.
func TestAMarkedSpanIsItsOwnNodeKind(t *testing.T) {
	source := []byte("a ==marked== word\n")
	root := parse().Parser().Parse(text.NewReader(source))

	var found ast.Node
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == KindHighlight {
			found = n
		}
		return ast.WalkContinue, nil
	})
	if found == nil {
		t.Fatal("==marked== parsed into no Highlight node")
	}
	if got := found.Kind().String(); got != "Highlight" {
		t.Errorf("the node kind is %q, want Highlight", got)
	}

	// Dump is goldmark's debug printer, and it writes to stdout. What is worth
	// stating is that the node answers it, because a node that panics there is
	// one nobody can print while working out why a document came out wrong.
	if got := dumped(t, found, source); !strings.Contains(got, "Highlight") {
		t.Errorf("Dump printed %q, and it does not name the node", got)
	}
}

// dumped runs n.Dump with stdout redirected, and hands back what it wrote.
func dumped(t *testing.T, n ast.Node, source []byte) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		done <- string(out)
	}()
	n.Dump(source, 0)
	os.Stdout = saved
	if err := w.Close(); err != nil {
		t.Fatalf("closing the pipe: %v", err)
	}
	out := <-done
	if err := r.Close(); err != nil {
		t.Fatalf("closing the read end: %v", err)
	}
	return out
}
