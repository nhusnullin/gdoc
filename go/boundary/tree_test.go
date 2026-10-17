package boundary

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The walk two rules share: every file in this tree that git does not ignore.
// The domain scan reads each as lines, the logo scan hashes each as bytes, and
// both have to agree about what the tree is. Reading the .gitignore files here
// rather than asking git is what keeps the rule in CLAUDE.md true: nothing
// under go/ runs a program, and that includes the tests.

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

// walkTracked walks root and hands visit every file git does not ignore, by
// its slash-separated path relative to root and by its path on disk. It skips
// .git and whatever the .gitignore files under root name, which it reads
// itself: reading a file is not running git.
func walkTracked(root string, visit func(rel, path string) error) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
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
	return filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
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
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			rel = path
		}
		return visit(filepath.ToSlash(rel), path)
	})
}
