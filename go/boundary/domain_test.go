package boundary

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// companyHostSHA256 is the SHA-256 of the firm's own email domain, lowercased,
// with no scheme and no trailing dot. It is a hash and not the domain because
// the rule is that the domain is not in the repository, and a test naming it
// would be the first file to break that rule. It must never be written out,
// here or in a failure message.
const companyHostSHA256 = "5fa5b421eb18f7a84987c3c36a6640571defd76bcd8278f5893730c3354fb5bb"

// hostSHA256 is what the scan compares against: the hash of one host.
func hostSHA256(host string) [32]byte {
	return sha256.Sum256([]byte(strings.ToLower(host)))
}

// wantHost reads a hex hash into the shape scanForHost takes.
func wantHost(t *testing.T, hexHash string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(hexHash)
	if err != nil || len(b) != 32 {
		t.Fatalf("the hash constant is not 32 hex-encoded bytes")
	}
	var want [32]byte
	copy(want[:], b)
	return want
}

// A .gitignore file, read as patterns relative to the directory holding it.
type ignoreFile struct {
	dir      string // absolute
	patterns []pattern
}

// pattern is one .gitignore line, read once rather than at every path: the
// text as written for the failure message, whether it names a path back into
// the scan, whether it names a directory, and the body the match is made on.
type pattern struct {
	text     string
	negated  bool
	dirOnly  bool
	body     string
	anchored bool // the body holds a /, so it is a path rather than a name
}

// readIgnore reads one .gitignore and refuses every pattern it cannot read.
// Refusing rather than skipping is the point: a pattern this helper quietly
// misunderstood would widen what the scan never looks at, and nobody would
// find out from a passing test.
func readIgnore(path string) ([]pattern, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var patterns []pattern
	for n, line := range strings.Split(string(b), "\n") {
		text := strings.TrimSpace(line)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		p, err := readPattern(text)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %q: %w", path, n+1, text, err)
		}
		patterns = append(patterns, p)
	}
	return patterns, nil
}

// readPattern reads one line into the shape ignores matches on, and refuses
// the shapes the four rules below do not cover. A negation is read by taking
// the ! off and reading what is left, so a negation of a shape the scan cannot
// read is refused like the shape itself.
func readPattern(text string) (pattern, error) {
	p := pattern{text: text}
	body := text
	if strings.HasPrefix(body, "!") {
		p.negated = true
		body = body[1:]
	}
	if strings.Contains(body, "**") {
		return p, errors.New("a ** pattern is not one this scan reads")
	}
	if strings.HasSuffix(body, "/") {
		p.dirOnly = true
		body = strings.TrimSuffix(body, "/")
	}
	if body == "" {
		return p, errors.New("a pattern with nothing in it")
	}
	p.body = body
	p.anchored = strings.Contains(body, "/")
	probe := body
	if !p.anchored {
		probe = filepath.Base(body)
	}
	if _, err := filepath.Match(probe, "x"); err != nil {
		return p, fmt.Errorf("a pattern filepath.Match refuses: %w", err)
	}
	return p, nil
}

// ignores says whether a .gitignore's patterns leave rel out of the scan, rel
// being a path relative to that file's directory which is a directory when
// isDir.
//
// Four rules, and they are the four git shapes this tree actually uses: a
// pattern ending in / names a directory; a pattern holding a / is a path
// relative to the .gitignore's own directory, with a * in it matching inside
// one segment; a pattern holding a * and no / is matched against the base
// name; anything else names a file or directory at any depth.
//
// The last pattern that matches decides, which is git's own rule, and it is
// what makes a ! line mean anything: everything under a directory is ignored
// and one file in it is named back in. Every pattern is tried rather than the
// first match winning.
func ignores(patterns []pattern, rel string, isDir bool) bool {
	base := filepath.Base(rel)
	out := false
	for _, p := range patterns {
		if p.dirOnly && !isDir {
			continue
		}
		if !p.matches(rel, base) {
			continue
		}
		out = !p.negated
	}
	return out
}

// matches is the one path rule, by pattern shape. A path pattern matches the
// path itself, and a directory it names carries everything under it.
func (p pattern) matches(rel, base string) bool {
	switch {
	case p.anchored:
		if ok, _ := filepath.Match(p.body, rel); ok {
			return true
		}
		return rel == p.body || strings.HasPrefix(rel, p.body+"/")
	case strings.Contains(p.body, "*"):
		ok, _ := filepath.Match(p.body, base)
		return ok
	default:
		return base == p.body
	}
}

// scanForHost walks root and returns path:line for every line holding a host
// whose SHA-256 is want. It skips .git and whatever the .gitignore files under
// root name, which it reads itself: reading a file is not running git, and
// nothing under go/ may run a program.
func scanForHost(root string, want [32]byte) ([]string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var found []string
	var files []ignoreFile
	ignored := func(path string, isDir bool) bool {
		for _, f := range files {
			rel, err := filepath.Rel(f.dir, path)
			if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
				continue
			}
			if ignores(f.patterns, filepath.ToSlash(rel), isDir) {
				return true
			}
		}
		return false
	}
	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != abs && (d.Name() == ".git" || ignored(path, true)) {
				return filepath.SkipDir
			}
			// A directory's own .gitignore is read on the way in, not when the
			// walk reaches it as a file: WalkDir visits a directory's entries
			// in lexical order, and .claude sorts before .gitignore.
			ignorePath := filepath.Join(path, ".gitignore")
			if _, err := os.Stat(ignorePath); err == nil {
				patterns, err := readIgnore(ignorePath)
				if err != nil {
					return err
				}
				files = append(files, ignoreFile{dir: path, patterns: patterns})
			}
			return nil
		}
		if ignored(path, false) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			rel = path
		}
		for n, line := range strings.Split(string(b), "\n") {
			if lineHasHost(line, want) {
				found = append(found, fmt.Sprintf("%s:%d", filepath.ToSlash(rel), n+1))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// lineHasHost takes every dotted token out of one line and hashes every
// dot-boundary suffix of it that still holds a dot, so a subdomain of the host
// is caught as well as the host itself.
func lineHasHost(line string, want [32]byte) bool {
	for _, tok := range dottedTokens(line) {
		rest := tok
		for {
			if strings.Contains(rest, ".") && hostSHA256(rest) == want {
				return true
			}
			i := strings.Index(rest, ".")
			if i < 0 {
				break
			}
			rest = rest[i+1:]
		}
	}
	return false
}

// dottedTokens finds every [a-z0-9-]+(\.[a-z0-9-]+)+ in the lowercased line.
func dottedTokens(line string) []string {
	s := strings.ToLower(line)
	hostByte := func(c byte) bool {
		return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.'
	}
	var out []string
	for i := 0; i < len(s); {
		if !hostByte(s[i]) {
			i++
			continue
		}
		j := i
		for j < len(s) && hostByte(s[j]) {
			j++
		}
		tok := strings.Trim(s[i:j], ".-")
		if strings.Contains(tok, ".") && !strings.Contains(tok, "..") {
			out = append(out, tok)
		}
		i = j
	}
	return out
}

// The canary. It plants a host in a temp tree and asks for that host's hash, so
// the scanner is watched finding something before it is trusted to find
// nothing, and no test in this file names the firm's domain, whole or in parts.
func TestTheDomainScanFindsAPlantedHost(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "notes", "note.md"), "ask\nwrite to person@example.net today\n")
	write(t, filepath.Join(root, "sub", "deep.go"), "// mail.example.net is the relay\n")
	write(t, filepath.Join(root, "clean.txt"), "nothing here but example.org\n")

	found, err := scanForHost(root, wantHost(t, "3daab7cff97925bbd07d11df5dc3b0e37e2d965520175ada0ec62ce72cda5ed2"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"notes/note.md:2", "sub/deep.go:1"}
	if strings.Join(found, " ") != strings.Join(want, " ") {
		t.Errorf("found = %v, want %v", found, want)
	}
}

// What git ignores is not the tree. The three .gitignore files here are shaped
// like the three real ones, and the planted host sits only under what they
// name, except for the one file no pattern covers.
func TestTheDomainScanSkipsWhatGitIgnores(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".gitignore"), "*.py[cod]\n\n# build output\nbin/\n.claude/worktrees/\n")
	write(t, filepath.Join(root, ".ralphex", ".gitignore"), ".gitignore\nprogress/\nworktrees/\n")
	write(t, filepath.Join(root, ".revmux", ".gitignore"), "tasks/\n")

	planted := "person@example.net\n"
	write(t, filepath.Join(root, "bin", "gdoc.log"), planted)
	write(t, filepath.Join(root, ".ralphex", "progress", "run.txt"), planted)
	write(t, filepath.Join(root, ".revmux", "tasks", "finding.md"), planted)
	write(t, filepath.Join(root, "go", "stale.pyc"), planted)
	write(t, filepath.Join(root, ".claude", "worktrees", "a", "copy.md"), planted)

	want := wantHost(t, "3daab7cff97925bbd07d11df5dc3b0e37e2d965520175ada0ec62ce72cda5ed2")
	found, err := scanForHost(root, want)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("the scan read what git ignores: %v", found)
	}

	write(t, filepath.Join(root, "docs", "guide.md"), planted)
	found, err = scanForHost(root, want)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(found, " ") != "docs/guide.md:1" {
		t.Errorf("found = %v, want [docs/guide.md:1]", found)
	}
}

// A negation names a file back into the scan, and what a path pattern holding
// a star names stays out of it.
//
// Both halves are the root .gitignore's own shape since M18: the live
// recordings are somebody's real document, so everything under
// internal/live/testdata is ignored and the two probe fixtures are named back
// in one at a time. A scan that read the negation as an ordinary pattern would
// ignore the fixtures and read the recordings, which is the one direction this
// file exists to prevent.
func TestTheDomainScanReadsBackWhatANegationNames(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".gitignore"), "recordings/*\n!recordings/fixture.md\n")

	planted := "person@example.net\n"
	write(t, filepath.Join(root, "recordings", "somebodys-document.json"), planted)
	write(t, filepath.Join(root, "recordings", "deep", "also-theirs.json"), planted)
	write(t, filepath.Join(root, "recordings", "fixture.md"), planted)

	found, err := scanForHost(root, wantHost(t, "3daab7cff97925bbd07d11df5dc3b0e37e2d965520175ada0ec62ce72cda5ed2"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(found, " ") != "recordings/fixture.md:1" {
		t.Errorf("found = %v, want [recordings/fixture.md:1]", found)
	}
}

// The last pattern that matches decides, which is git's rule and not a
// convenience: a negation above the pattern that excludes a file leaves that
// file excluded.
func TestTheLastPatternThatMatchesDecides(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".gitignore"), "!recordings/fixture.md\nrecordings/*\n")
	write(t, filepath.Join(root, "recordings", "fixture.md"), "person@example.net\n")

	found, err := scanForHost(root, wantHost(t, "3daab7cff97925bbd07d11df5dc3b0e37e2d965520175ada0ec62ce72cda5ed2"))
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("the negation above the pattern re-included the file: %v", found)
	}
}

// A pattern the helper cannot read fails by name. A new .gitignore line that
// this scan would misunderstand must not quietly widen what it never looks at.
//
// A negation is read now, so what has to fail is a negation whose own body is
// a shape the scan cannot read.
func TestTheDomainScanRefusesAPatternItCannotRead(t *testing.T) {
	for _, pattern := range []string{"a/**/b", "!a/**/b", "*.py[cod", "!*.py[cod"} {
		root := t.TempDir()
		write(t, filepath.Join(root, ".gitignore"), pattern+"\n")
		_, err := scanForHost(root, wantHost(t, "3daab7cff97925bbd07d11df5dc3b0e37e2d965520175ada0ec62ce72cda5ed2"))
		if err == nil {
			t.Fatalf("%q was accepted; a pattern the scan cannot read has to fail", pattern)
		}
		if !strings.Contains(err.Error(), pattern) || !strings.Contains(err.Error(), ".gitignore:1") {
			t.Errorf("%q: error does not name the line: %v", pattern, err)
		}
	}
}

// The rule itself. No real company domain appears in this repository, as a
// default, an example or a fixture: examples use the RFC 2606 domains, which
// exist so nobody has to borrow a real one.
func TestNoCompanyDomainInTheTree(t *testing.T) {
	found, err := scanForHost(repoRoot, wantHost(t, companyHostSHA256))
	if err != nil {
		t.Fatal(err)
	}
	for _, where := range found {
		t.Errorf("%s holds the firm's own domain; use example.com, example.org or example.net", where)
	}
}
