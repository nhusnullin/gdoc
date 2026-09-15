// The three guards over the docs' shape.
//
// The wire checks in boundary_test.go keep one promise about the code. These
// keep three about the documentation, and they are here for the same reason:
// the rule is only true if something asks it on every commit. A prose rule
// nobody measures is a rule that decays quietly, which is how CLAUDE.md grew
// to 3,311 lines of per-package essay that no reader asked for and every
// session paid for.
//
// The first guard is a ceiling on the file every session loads. The second
// says each package carries exactly one package comment, in a file `go doc`
// reads. The third says the task map in CLAUDE.md names files that are really
// there.

package boundary

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// repoRoot is where the module sits: go/boundary is two levels down.
const repoRoot = "../.."

// claudeCeiling is the line count CLAUDE.md may not pass. It is today's count,
// so the guard is live from this commit and measures a file nobody has rewritten
// yet. Task 24 of the docs restructure rewrites that file whole and lowers this
// to 300. Raising it after that is a decision somebody explains in the commit
// message, not a number somebody nudges to make a test green: this file is
// loaded into every session unasked, so its length is a cost paid by every
// reader whether or not they needed the words.
const claudeCeiling = 3311

func TestCLAUDEmdIsUnderTheCeiling(t *testing.T) {
	path := filepath.Join(repoRoot, "CLAUDE.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := len(strings.Split(strings.TrimRight(string(b), "\n"), "\n"))
	if lines > claudeCeiling {
		t.Errorf("CLAUDE.md is %d lines, over the %d line ceiling; it is loaded into every session, so length is a cost every reader pays", lines, claudeCeiling)
	}
}

// docRoots are the trees walked for package comments. Everything the binary is
// made of is under one of the three.
var docRoots = []string{"cmd/gdoc", "internal", "boundary"}

// known lists the package directories expected to fail the one-comment rule
// today, with the reason. It works in both directions, the way drift.Known and
// TestEveryKnownDifferenceStillDiffers do: a package off this list that breaks
// the rule fails, and a package on it that already holds the rule fails too, so
// the task that fixes a package has to delete its row rather than leave a name
// here that stopped meaning anything.
//
// The one entry left is a package whose only files are tests, so its package
// comment sits where `go doc` does not look. Task 6 gave boundary a doc.go and
// deleted its row; Task 19 gives internal/live one and deletes the last.
//
// The docs restructure plan named four here, adding cmd/gdoc and internal/render
// on the belief that cmd/gdoc carried no package comment and internal/render
// carried two. Measured on 2026-09-15, neither is so: cmd/gdoc carries one, on
// main.go, opening "Command gdoc", and internal/render carries one, on xml.go.
// Both already hold the rule, so listing them would fail this test on its own
// second assertion.
var known = map[string]string{
	"internal/live": "its package comment is on live_test.go, and go doc does not read test files",
}

// pkgComment reports whether a parsed file carries the package's own comment.
//
// Any comment block sitting directly above the package clause is a Doc to
// go/parser, and most files here carry one: a paragraph saying what that file
// is for. Those are not package comments and must not be counted, or the rule
// would read as "one commented file per package" and refuse the repo's own
// convention. The package comment is the one written in Go's documented shape,
// opening with the package's name, and for a main package with the command's,
// which is what `go doc` prints and what the convention in the plan asks each
// package to hold exactly one of.
func pkgComment(f *ast.File) bool {
	if f.Doc == nil {
		return false
	}
	text := f.Doc.Text()
	opening := "Package " + f.Name.Name + " "
	if f.Name.Name == "main" {
		opening = "Command "
	}
	return strings.HasPrefix(text, opening)
}

// packageComments walks a tree and reports, per package directory, which files
// carry the package comment. Test files are counted separately because where
// the comment sits is half the rule.
func packageComments(root string) (prod, tests map[string][]string, err error) {
	prod, tests = map[string][]string{}, map[string][]string{}
	err = filepath.WalkDir(filepath.Join(repoRoot, "go", root), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return walkErr
		}
		f, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if perr != nil {
			return perr
		}
		rel, rerr := filepath.Rel(filepath.Join(repoRoot, "go"), filepath.Dir(path))
		if rerr != nil {
			return rerr
		}
		pkg := filepath.ToSlash(rel)
		if _, seen := prod[pkg]; !seen {
			prod[pkg] = nil
		}
		if !pkgComment(f) {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasSuffix(base, "_test.go") {
			tests[pkg] = append(tests[pkg], base)
			return nil
		}
		prod[pkg] = append(prod[pkg], base)
		return nil
	})
	return prod, tests, err
}

// TestEveryPackageHasExactlyOnePackageComment is the guard behind the doc.go
// convention. One package comment, in a file that is not a test, because a
// package comment on a test file is one `go doc` never prints: the reader who
// went looking for the reasons finds nothing and writes them down a third time.
// Two comments are the other failure, and it is the quieter one, since go/doc
// joins them and neither author sees the pair.
func TestEveryPackageHasExactlyOnePackageComment(t *testing.T) {
	for _, root := range docRoots {
		prod, tests, err := packageComments(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range sorted(prod) {
			why, listed := known[pkg]
			problem := packageProblem(prod[pkg], tests[pkg])
			switch {
			case problem == "" && listed:
				t.Errorf("%s is on known (%s) and already holds the rule; delete its row", pkg, why)
			case problem != "" && !listed:
				t.Errorf("%s %s; a package carries exactly one package comment, in a file that is not a test", pkg, problem)
			}
		}
	}
}

// packageProblem names what is wrong with a package's comments, or returns an
// empty string when the package holds the rule.
func packageProblem(prod, tests []string) string {
	switch {
	case len(prod) == 1 && len(tests) == 0:
		return ""
	case len(prod) == 0 && len(tests) == 0:
		return "carries no package comment"
	case len(prod) == 0:
		return "carries its package comment on a test file (" + strings.Join(tests, ", ") + "), where go doc does not read it"
	case len(prod) > 1:
		return "carries " + plural(len(prod)) + " package comments (" + strings.Join(prod, ", ") + ")"
	default:
		return "carries a second package comment on a test file (" + strings.Join(tests, ", ") + ")"
	}
}

func plural(n int) string {
	switch n {
	case 2:
		return "two"
	case 3:
		return "three"
	default:
		return "several"
	}
}

func sorted(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// taskMapHeading is the heading CLAUDE.md's task map sits under. The map is the
// one thing in that file a reader uses to decide where to go next, so a row
// naming a file that has moved sends them nowhere.
const taskMapHeading = "## If you touch"

// backticked finds every span between backticks. A candidate path is one with
// no spaces in it that carries a slash or a dot, which is what separates
// `go/internal/guard/doc.go` from a backticked word such as `read`.
var backticked = regexp.MustCompile("`([^`]+)`")

// TestTheTaskMapNamesFilesThatExist reads the task map and stats every path in
// it, relative to the repo root. A missing heading is a failure rather than an
// empty pass: a map that is not there is the same silence as a map full of dead
// rows, and this test exists to make that audible.
func TestTheTaskMapNamesFilesThatExist(t *testing.T) {
	t.Skip("fails until Task 24 writes the heading; Task 24 removes this skip")

	b, err := os.ReadFile(filepath.Join(repoRoot, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	paths, found := taskMapPaths(string(b))
	if !found {
		t.Fatalf("CLAUDE.md has no %q heading; the task map is what tells a reader which file to open", taskMapHeading)
	}
	if len(paths) == 0 {
		t.Fatalf("the %q table names no path; a map with no rows is not a map", taskMapHeading)
	}
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(repoRoot, p)); err != nil {
			t.Errorf("the task map names %s, which is not there: %v", p, err)
		}
	}
}

// taskMapPaths returns every path the task map names, and whether the heading
// was there at all. The section ends at the next heading of any level.
func taskMapPaths(doc string) (paths []string, found bool) {
	sc := bufio.NewScanner(strings.NewReader(doc))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	in := false
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, taskMapHeading) {
			in, found = true, true
			continue
		}
		if in && strings.HasPrefix(line, "#") {
			break
		}
		if !in {
			continue
		}
		for _, m := range backticked.FindAllStringSubmatch(line, -1) {
			if p := strings.TrimSuffix(m[1], "/"); isPath(p) {
				paths = append(paths, p)
			}
		}
	}
	return paths, found
}

func isPath(s string) bool {
	if s == "" || strings.ContainsAny(s, " \t") {
		return false
	}
	return strings.Contains(s, "/") || strings.Contains(s, ".")
}
