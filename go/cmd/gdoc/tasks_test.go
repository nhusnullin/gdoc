package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// taskNote is one note whose list holds a bare owner. It is the note both
// tests below are about: `build` writes it to a file and `publish` would send
// it, and the owner is the one thing the two commands answer differently.
const taskNote = `---
title: Supplier Questions
---

# Questions

- [ ] Which bank holds the safeguarding account @anna
`

// A bare owner is an address gdoc cannot finish without the domain in effect,
// and a publish that sent one would put a chip on nobody. So the refusal comes
// before the upload: no document is made, no request leaves the machine, and
// the sentence names the setting and the flag.
func TestABareOwnerWithNoDomainIsRefusedBeforeTheUpload(t *testing.T) {
	noSession(t)
	md := writeNote(t, "questions.md", taskNote)

	got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID)

	if code == 0 || got["ok"] == true {
		t.Fatalf("a publish of a note with a bare owner and no domain answered ok: %v", got)
	}
	text, _ := got["error"].(string)
	if !strings.Contains(text, "@anna") {
		t.Errorf("the refusal does not name the token it could not finish: %q", text)
	}
	if !strings.Contains(text, "domain") {
		t.Errorf("the refusal does not name the setting that would finish it: %q", text)
	}
}

// A build writes a file. Nothing in it is sent to anybody, and the docx keeps
// the token as the note wrote it, so there is nothing to resolve: the same
// note that stops a publish builds.
func TestBuildResolvesNoOwner(t *testing.T) {
	noSession(t)
	md := writeNote(t, "questions.md", taskNote)
	out := filepath.Join(filepath.Dir(md), "questions.docx")

	got, code := runJSON(t, "build", "--md", md, "--out", out)

	if code != 0 || got["ok"] != true {
		t.Fatalf("a build of a note with a bare owner and no domain failed: %v", got)
	}
}
