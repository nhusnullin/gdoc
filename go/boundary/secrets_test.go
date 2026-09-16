package boundary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The OAuth client secret left the source on 2026-09-16, the day the
// repository went public: GitHub reports a Google client secret it finds in a
// public commit to Google, who may revoke the client and break every
// colleague's login at once. The secret is injected at build time instead.
// This test refuses the shape of a Google client secret anywhere in the tree,
// so the next person who "fixes" a failing local login by pasting one in
// finds out before the commit does. The prefix is built from two halves so
// this file does not match itself.
func TestNoGoogleClientSecretInTheTree(t *testing.T) {
	prefix := "GOC" + "SPX-"
	// Hidden directories at the root are tool state, the review logs among
	// them, and bin/ is build output. Neither is the tree, and .github is
	// the one hidden directory that is.
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			atRoot := filepath.Dir(path) == root
			hidden := strings.HasPrefix(d.Name(), ".") && d.Name() != ".github"
			if path == root {
				return nil
			}
			if (atRoot && (hidden || d.Name() == "bin")) || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), prefix) {
			t.Errorf("%s carries a string shaped like a Google client secret; the secret is injected at build time and never committed", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
