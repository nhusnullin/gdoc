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

// `gdoc help publish` and `gdoc help propose` are the two screens read before
// a task is written, by a person and by a skill, so each carries the notation
// as a line of its own. They are notes rather than flag sentences, because a
// task is syntax the command reads out of a file and belongs to no flag, and
// they reach the object as well as the screen, because the other reader of
// this help is a session building a note.
func TestHelpPublishAndProposeNameTheTaskNotation(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	for _, c := range []struct {
		command string
		wants   []string
	}{
		{"publish", []string{"- [ ]", "chip", "emails nobody"}},
		{"propose", []string{"- [ ]", "chip"}},
	} {
		got, prose, code := runHelp(t, "help", c.command, "--json")
		if code != 0 || got["ok"] != true {
			t.Fatalf("gdoc help %s must be answered: %v (exit %d)", c.command, got, code)
		}
		list := helpCommands(t, got)
		if len(list) != 1 {
			t.Fatalf("gdoc help %s is one entry and %d came back: %v", c.command, len(list), list)
		}
		entry, ok := list[0].(map[string]any)
		if !ok {
			t.Fatalf("the entry is not an object: %v", list[0])
		}
		notes, ok := entry["notes"].([]any)
		if !ok || len(notes) == 0 {
			t.Fatalf("%s's entry carries no notes, and the task notation is one: %v", c.command, entry)
		}
		joined := ""
		for _, n := range notes {
			line, _ := n.(string)
			joined += line + "\n"
			// The words a person reads carry every note, so a
			// colleague who typed the help without --json is not
			// sent to the object for it.
			if !strings.Contains(prose, line) {
				t.Errorf("the words gdoc help %s prints leave out %q", c.command, line)
			}
		}
		for _, want := range c.wants {
			if !strings.Contains(joined, want) {
				t.Errorf("gdoc help %s never says %q in its notes: %q", c.command, want, joined)
			}
		}
	}
}
