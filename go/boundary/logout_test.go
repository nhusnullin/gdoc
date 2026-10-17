// The guard over the one file that is somebody's credential.
//
// The token is the whole of what gdoc can reach Google with. One room may take
// it away, by name, and every other room reads it. A second remover would be a
// second rule about when a person is signed out, written somewhere nobody
// looks.

package boundary

import (
	"go/ast"
	"go/parser"
	"path/filepath"
	"strings"
	"testing"
)

// tokenRemover is the one function in the tree that may remove the token:
// internal/auth's Logout, which `gdoc auth logout` is the one caller of.
const tokenRemover = "internal/auth.Logout"

// TestOnlyLogoutRemovesTheToken reads every function of the tree for the two
// things a remover does: it asks where the token is, and it removes a file.
//
// The granularity is the function and not the file, because internal/auth's
// pending.go removes the waiting sign-in's lock and asks where the token is,
// in two functions that have nothing to do with each other. A function doing
// both is a function that can sign somebody out.
//
// It fails in both directions: a second function that does both, and Logout
// no longer doing it, because then this test guards nothing.
func TestOnlyLogoutRemovesTheToken(t *testing.T) {
	found := map[string]bool{}
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			var asksWhere, removes bool
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if !ok {
					return true
				}
				switch id.Name {
				case "TokenPath":
					asksWhere = true
				case "Remove", "RemoveAll", "Rename", "Truncate":
					removes = true
				}
				return true
			})
			if asksWhere && removes {
				found[filepath.ToSlash(rel)+"."+fn.Name.Name] = true
			}
		}
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	for name := range found {
		if name != tokenRemover {
			t.Errorf("%s asks where the token is and takes a file away; only %s may", name, tokenRemover)
		}
	}
	if !found[tokenRemover] {
		t.Errorf("%s no longer removes the token, and these are the names that do: %v",
			tokenRemover, names(found))
	}
}

// TestNothingInTheChatSignsAnybodyOut is the other direction of the same rule.
// internal/chat is what a session adds to a command, and a model that could
// take the credential away could stop every tool of its own session working.
func TestNothingInTheChatSignsAnybodyOut(t *testing.T) {
	err := walkGo("..", func(rel, path string, f *ast.File) error {
		if rel != "internal/chat" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == "Logout" {
				t.Errorf("%s names Logout; a chat signs nobody out", filepath.ToSlash(path))
			}
			return true
		})
		return nil
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
}
