// The author's page break where a caller meets it: the two envelopes that turn
// a note into a document carry the number of breaks the document got.

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// breakNote holds three `---` lines and gets two pages out of them: the one at
// the foot of the note opens a page holding nothing, so it is left out. The
// title is the one the publish read-back fixture carries, so both commands can
// be run over the same note without a title warning.
const breakNote = `---
title: Supplier Register Policy
---

# Purpose

The first page.

---

The second page.

---

The third page.

---
`

// theBreaksTheDocumentGot is what both envelopes must say for that note,
// written out as a literal: the paragraphs the walk emitted, not the lines the
// note held.
const theBreaksTheDocumentGot = float64(2)

// Both commands print what the walker counted, and a page break is one of the
// facts a note's author has to be able to check: three dividers in the note and
// two pages in the document is a difference nobody can see from the file. The
// number is a number and not a refusal, because whether a document of 216
// pages is what somebody wanted is the skill's to say.
func TestBuildAndPublishCarryThePageBreakCount(t *testing.T) {
	t.Run("build", func(t *testing.T) {
		noSession(t)
		md := writeNote(t, "register.md", breakNote)
		out := filepath.Join(filepath.Dir(md), "register.docx")

		got, code := runJSON(t, "build", "--md", md, "--out", out)
		if code != 0 || got["ok"] != true {
			t.Fatalf("build: %v (exit %d)", got, code)
		}
		assertPageBreaks(t, got)
	})

	t.Run("publish", func(t *testing.T) {
		stubWire(t, &fakeWire{answers: publishAnswers(t)})
		md := writeNote(t, "register.md", breakNote)

		got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID)
		if code != 0 || got["ok"] != true {
			t.Fatalf("publish: %v (exit %d)", got, code)
		}
		assertPageBreaks(t, got)
	})
}

// assertPageBreaks reads body.page_breaks off an envelope, by the key a skill
// reads it by.
func assertPageBreaks(t *testing.T, got map[string]any) {
	t.Helper()
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	counts, ok := data["body"].(map[string]any)
	if !ok {
		t.Fatalf("no body counts: %v", data)
	}
	if counts["page_breaks"] != theBreaksTheDocumentGot {
		t.Errorf("body.page_breaks is %v, want %v: the pages the document got, not the lines the note held",
			counts["page_breaks"], theBreaksTheDocumentGot)
	}
}

// gdoc help build is where somebody reads what a note may hold without leaving
// the terminal, so the line saying `---` starts a new page belongs there and
// not only in the guide. The note carries the count's key beside the syntax,
// because the pages the document got are the only check on the lines the note
// held.
func TestHelpBuildSaysAThematicBreakIsAPageBreak(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, prose, code := runHelp(t, "help", "build", "--json")
	if code != 0 || got["ok"] != true {
		t.Fatalf("gdoc help build must be answered: %v (exit %d)", got, code)
	}
	list := helpCommands(t, got)
	if len(list) != 1 {
		t.Fatalf("gdoc help build is one entry and %d came back: %v", len(list), list)
	}
	entry, ok := list[0].(map[string]any)
	if !ok {
		t.Fatalf("the entry is not an object: %v", list[0])
	}
	notes, ok := entry["notes"].([]any)
	if !ok {
		t.Fatalf("build's entry carries no notes: %v", entry)
	}
	var breakLine string
	for _, n := range notes {
		line, _ := n.(string)
		if strings.Contains(line, "page break") {
			breakLine = line
			break
		}
	}
	if breakLine == "" {
		t.Fatalf("no line of gdoc help build says what a page break is: %v", notes)
	}
	for _, want := range []string{"---", "page_breaks"} {
		if !strings.Contains(breakLine, want) {
			t.Errorf("the page break line leaves out %q: %q", want, breakLine)
		}
	}
	if !strings.Contains(prose, breakLine) {
		t.Errorf("the words a person reads leave out %q: %q", breakLine, prose)
	}
}
