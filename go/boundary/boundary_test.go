// Package boundary enforces that gdoc has exactly one wire. Only the guard
// (and, from Task 7 on, the login loopback listener, which serves and never
// dials) may import net/http. This is the v1 allowlist test, ported: it fails
// in BOTH directions, on spread and on silent disappearance.
package boundary

import (
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
// import net/http. `internal/auth/loopback` joins this set in Task 7, which
// adds the login listener. The room is listed only once it exists, because
// this test has to be green at every commit.
var allowed = map[string]bool{
	"internal/guard": true,
}

// httpImporters returns the package directories under root, slash separated
// and relative to root, that import net/http.
func httpImporters(root string) (map[string]bool, error) {
	found := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) != "net/http" {
				continue
			}
			rel, rerr := filepath.Rel(root, filepath.Dir(path))
			if rerr != nil {
				return rerr
			}
			found[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
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

// TestScannerFindsAStrayImport is the canary. Without it the test above passes
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
