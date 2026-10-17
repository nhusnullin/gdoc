package boundary

import (
	"crypto/sha256"
	"encoding/hex"
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

// scanForHost returns path:line for every line in the tree holding a host
// whose SHA-256 is want. What the tree is is walkTracked's answer.
func scanForHost(root string, want [32]byte) ([]string, error) {
	var found []string
	err := walkTracked(root, func(rel, path string) error {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for n, line := range strings.Split(string(b), "\n") {
			if lineHasHost(line, want) {
				found = append(found, fmt.Sprintf("%s:%d", rel, n+1))
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
