// The four guards over the docs' shape.
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
// reads. The third says nothing but that comment reaches `go doc`, since a file
// paragraph touching the package clause is joined into it. The fourth says the
// task map in CLAUDE.md names files that are really there.

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

// claudeCeiling is the line count CLAUDE.md may not pass. Raising it is a
// decision somebody explains in the commit message, not a number somebody
// nudges to make a test green: the file is loaded into every session unasked,
// so its length is a cost paid by every reader whether or not they needed the
// words. What belongs in a longer file is a package's own doc.go, which a
// reader opens when the task reaches that package.
const claudeCeiling = 300

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

// known lists the package directories expected to fail the one-comment rule,
// with the reason. It works in both directions, the way drift.Known and
// TestEveryKnownDifferenceStillDiffers do: a package off this list that breaks
// the rule fails, and a package on it that already holds the rule fails too, so
// a task that fixes a package has to delete its row rather than leave a name
// here that stopped meaning anything.
//
// It is empty. Every package under the three roots carries exactly one package
// comment, in a file that is not a test, so the second assertion has nothing to
// check today. The map stays rather than the check being written as a plain
// "every package holds the rule", because what it holds is a way to defer one
// package with its reason written down, and the second assertion is what stops
// a deferral outliving the work it was waiting for.
//
// The two packages that were listed were the ones whose only files are tests,
// so their package comment sat where `go doc` does not look: boundary, which
// Task 6 gave a doc.go, and internal/live, which Task 19 gave one.
//
// The docs restructure plan named four here, adding cmd/gdoc and internal/render
// on the belief that cmd/gdoc carried no package comment and internal/render
// carried two. Measured on 2026-09-15, neither was so: each already carried
// exactly one, so listing them would have failed this test on its own second
// assertion. Both comments have since moved into that package's doc.go, which
// is where a reader should look rather than at the files this note used to
// name.
var known = map[string]string{}

// pkgComment reports whether a parsed file carries the package's own comment.
//
// Any comment block sitting directly above the package clause is a Doc to
// go/parser, and most files here carry one: a paragraph saying what that file
// is for. Those are not package comments and must not be counted, or the rule
// would read as "one commented file per package" and refuse the repo's own
// convention. The package comment is the one written in Go's documented shape:
// the word "Package" and the package's own name, or the bare word "Command" for
// a main package, as CLAUDE.md states the invariant. That is what `go doc`
// prints and what the convention asks each package to hold exactly one of.
func pkgComment(f *ast.File) bool {
	if f.Doc == nil {
		return false
	}
	text := f.Doc.Text()
	opening := "Package " + f.Name.Name
	if f.Name.Name == "main" {
		opening = "Command"
	}
	rest, ok := strings.CutPrefix(text, opening)
	if !ok {
		return false
	}
	// What follows the name has to end the word, or "Package emitter" would
	// count as package emit's comment. A period ends it, because
	// "// Package emit." is the shortest comment Go documents and go doc prints
	// it. TestPkgCommentReadsTheShortestPackageComment is the pin.
	return strings.HasPrefix(rest, " ") ||
		strings.HasPrefix(rest, ".") ||
		strings.HasPrefix(rest, "\n")
}

// TestPkgCommentReadsTheShortestPackageComment pins the helper the guard rests
// on. A one-line comment, "// Package emit.", is the shape Go's own
// documentation gives as the minimum, and `go doc` prints it. A helper that
// missed it would tell the next person their package carries no package comment
// while the comment is right there, which sends them looking in the wrong file.
func TestPkgCommentReadsTheShortestPackageComment(t *testing.T) {
	for _, src := range []string{
		"// Package emit.\npackage emit\n",
		"// Package emit is the envelope.\npackage emit\n",
		"// Command gdoc.\npackage main\n",
	} {
		f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		if !pkgComment(f) {
			t.Errorf("go doc prints this as the package comment, so the guard must read it too: %q", src)
		}
	}

	// The other direction: a file paragraph is not a package comment, and a
	// package whose name only starts the same way is not one either.
	for _, src := range []string{
		"// This file holds the writer's own vocabulary.\npackage render\n",
		"// Package emitter is another package.\npackage emit\n",
	} {
		f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		if pkgComment(f) {
			t.Errorf("this is not the package's own comment: %q", src)
		}
	}
}

// packageComments walks a tree and reports, per package directory, which files
// carry the package comment, and which files carry a paragraph that go/doc
// joins into it anyway. Test files are counted separately because where the
// comment sits is half the rule.
func packageComments(root string) (prod, tests, joined map[string][]string, err error) {
	prod, tests, joined = map[string][]string{}, map[string][]string{}, map[string][]string{}
	err = filepath.WalkDir(filepath.Join(repoRoot, "go", root), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// testdata is not Go the toolchain reads, so a fixture in it may be
		// malformed or carry no package clause at all. Parsing one would fail
		// this guard for a reason that has nothing to do with documentation.
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
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
		base := filepath.Base(path)
		if !pkgComment(f) {
			if f.Doc != nil {
				joined[pkg] = append(joined[pkg], base)
			}
			return nil
		}
		if strings.HasSuffix(base, "_test.go") {
			tests[pkg] = append(tests[pkg], base)
			return nil
		}
		prod[pkg] = append(prod[pkg], base)
		return nil
	})
	return prod, tests, joined, err
}

// TestEveryPackageHasExactlyOnePackageComment is the guard behind the doc.go
// convention. One package comment, in a file that is not a test, because a
// package comment on a test file is one `go doc` never prints: the reader who
// went looking for the reasons finds nothing and writes them down a third time.
// Two comments are the other failure, and it is the quieter one, since go/doc
// joins them and neither author sees the pair.
func TestEveryPackageHasExactlyOnePackageComment(t *testing.T) {
	for _, root := range docRoots {
		prod, tests, _, err := packageComments(root)
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

// TestOnlyThePackageCommentReachesGoDoc is the other half of the one-comment
// rule, and it is the half a reader notices first. A comment block sitting
// directly above the package clause is that file's Doc to go/parser, and go/doc
// concatenates every file's Doc into the package's documentation, in filename
// order. So a paragraph saying what one file is for does not stay in that file:
// it is printed as part of the package comment, and if its filename sorts
// before doc.go it is printed first.
//
// That is the failure TestEveryPackageHasExactlyOnePackageComment was written
// to catch and cannot, because pkgComment deliberately does not count a file
// paragraph. Counting them there would refuse the convention outright. The rule
// is not that a file may carry no paragraph; it is that the paragraph must be
// detached from the package clause by a blank line, which leaves it as an
// ordinary comment in the file and leaves `go doc` printing the package comment
// alone.
func TestOnlyThePackageCommentReachesGoDoc(t *testing.T) {
	for _, root := range docRoots {
		_, _, joined, err := packageComments(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range sorted(joined) {
			if len(joined[pkg]) == 0 {
				continue
			}
			t.Errorf("in %s, %s open with a paragraph touching the package clause, so go doc joins each one into the package comment; put a blank line between the paragraph and the package clause", pkg, strings.Join(joined[pkg], ", "))
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
