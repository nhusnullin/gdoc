// The author's page break where a caller meets it: the two envelopes that turn
// a note into a document carry the number of breaks the document got.

package main

import (
	"path/filepath"
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
