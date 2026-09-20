// This file is the note as a list of documents, from the command side: which
// entry a run acts on, what it refuses, and what it says once when it rewrites
// a block written before 2026-09-19. read_test.go and write_test.go hold the
// tests that were there when a note named one document.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/frontmatter"
)

// thirdDocID is a document no note in this file names, so a run of it is the
// refusal every writer makes.
const thirdDocID = "1ThIrD00000000000000000000000000000000000"

// otherDocID is the second document a note names, alongside the one the command
// under test is of.
const otherDocID = "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz"

// twoEntryNote writes a note naming two documents, the run's own second, so a
// test that passes cannot be a test that reads the first entry and stops.
func twoEntryNote(t *testing.T, first, second string, body string) string {
	t.Helper()
	src := "---\ntitle: Supplier register policy\ngdoc:\n  schema: 2\n  documents:\n    - id: " + first +
		"\n    - id: " + second + "\n" + body + "author: Nail\n---\n\n# Scope\n\nThe supplier register is reviewed annually by the operations team.\n"
	return tempFile(t, "two-documents.md", src)
}

// blockOf reads the note back.
func blockOf(t *testing.T, path string) *frontmatter.Block {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := frontmatter.Read(src)
	if err != nil {
		t.Fatalf("the note does not read back: %v", err)
	}
	return b
}

// entryOf is the entry for one document, or a failed test.
func entryOf(t *testing.T, path, id string) *frontmatter.Entry {
	t.Helper()
	e, err := blockOf(t, path).Entry(id)
	if err != nil {
		t.Fatalf("the note does not name %s: %v", id, err)
	}
	return e
}

// TestEveryWriterActsOnTheURLAndRefusesAnIDOutsideTheList is the list rule at
// the door. A note names every document it has been published to, and the URL
// is what says which one a run means: a run of a document the note does not
// name is refused, and the refusal says what the note does name, because the
// answer is always to open one of those instead.
//
// annotate is not here because it takes no --md: it leaves a comment on quoted
// words and records nothing in a note.
func TestEveryWriterActsOnTheURLAndRefusesAnIDOutsideTheList(t *testing.T) {
	t.Run("suggestions", func(t *testing.T) {
		stubSession(t, docsAndComments(t))
		stubNow(t, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
		note := twoEntryNote(t, otherDocID, fixtureDocID, "")

		got, code := runJSON(t, "suggestions", fixtureDocID, "--md", note)
		if code != 0 || got["ok"] != true {
			t.Fatalf("a run of the second document the note names must stand: %v (exit %d)", got, code)
		}
		if e := entryOf(t, note, fixtureDocID); e.SuggestionsSeen == nil {
			t.Error("the snapshot did not go under the document the URL named")
		}
		if e := entryOf(t, note, otherDocID); e.SuggestionsSeen != nil {
			t.Error("the snapshot went under the other document too")
		}
	})

	t.Run("suggestions refuses a document the note does not name", func(t *testing.T) {
		stubSession(t, docsAndComments(t))
		note := twoEntryNote(t, otherDocID, fixtureDocID, "")
		before := mustRead(t, note)

		got, code := runJSON(t, "suggestions", thirdDocID, "--md", note)
		if code == 0 || got["ok"] != false {
			t.Fatalf("a document the note does not name must be refused: %v (exit %d)", got, code)
		}
		msg, _ := got["error"].(string)
		for _, want := range []string{otherDocID, fixtureDocID, thirdDocID} {
			if !strings.Contains(msg, want) {
				t.Errorf("the refusal %q does not name %s", msg, want)
			}
		}
		if mustRead(t, note) != before {
			t.Error("a refused run wrote to the note")
		}
	})

	t.Run("propose", func(t *testing.T) {
		stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
		from := tempFile(t, "proposals.json", oneProposal)
		note := twoEntryNote(t, otherDocID, proposeDocID, "")

		got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID, "--md", note)
		if code != 0 || got["ok"] != true {
			t.Fatalf("a run of the second document the note names must stand: %v (exit %d)", got, code)
		}
		if len(entryOf(t, note, proposeDocID).Proposals) != 1 {
			t.Error("the proposal did not go under the document the URL named")
		}
	})

	t.Run("propose refuses a document the note does not name", func(t *testing.T) {
		f := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
		from := tempFile(t, "proposals.json", oneProposal)
		note := twoEntryNote(t, otherDocID, proposeDocID, "")

		got, code := runJSON(t, "propose", thirdDocID, "--from", from, "--folder", testFolderID, "--md", note)
		if code == 0 || got["ok"] != false {
			t.Fatalf("a document the note does not name must be refused: %v (exit %d)", got, code)
		}
		msg, _ := got["error"].(string)
		if !strings.Contains(msg, otherDocID) || !strings.Contains(msg, proposeDocID) {
			t.Errorf("the refusal %q does not name what the note holds", msg)
		}
		if len(f.writes()) != 0 {
			t.Errorf("nothing may be written before the note is checked: %v", f.writes())
		}
	})

	t.Run("withdraw", func(t *testing.T) {
		stubWire(t, &fakeWire{answers: []*answer{
			{method: "GET", match: withdrawDocID + "?includeTabsContent", json: readFixture(t, "withdraw-pending.json"), once: true},
			{method: "POST", match: withdrawDocID + ":batchUpdate", json: readFixture(t, "withdraw-rejected.json")},
			{method: "GET", match: withdrawDocID + "?includeTabsContent", json: readFixture(t, "withdraw-gone.json")},
		}})
		note := twoEntryNote(t, otherDocID, withdrawDocID, proposalsFor(t))

		got, code := runJSON(t, "withdraw", withdrawDocID, "suggest.abc", "--md", note)
		if code != 0 || got["ok"] != true {
			t.Fatalf("a run of the second document the note names must stand: %v (exit %d)", got, code)
		}
		if len(entryOf(t, note, withdrawDocID).Proposals) != 0 {
			t.Error("the proposal was not forgotten under the document the URL named")
		}
	})

	t.Run("withdraw refuses a document the note does not name", func(t *testing.T) {
		f := stubWire(t, &fakeWire{})
		note := twoEntryNote(t, otherDocID, withdrawDocID, proposalsFor(t))

		got, code := runJSON(t, "withdraw", thirdDocID, "suggest.abc", "--md", note)
		if code == 0 || got["ok"] != false {
			t.Fatalf("a document the note does not name must be refused: %v (exit %d)", got, code)
		}
		msg, _ := got["error"].(string)
		if !strings.Contains(msg, otherDocID) || !strings.Contains(msg, withdrawDocID) {
			t.Errorf("the refusal %q does not name what the note holds", msg)
		}
		if len(f.calls) != 0 {
			t.Errorf("nothing may be sent before the note is checked: %v", f.calls)
		}
	})
}

// proposalsFor is the proposals block the withdraw tests need under the second
// entry, indented for a note whose documents are a list.
func proposalsFor(t *testing.T) string {
	t.Helper()
	return "      proposals:\n        - id: suggest.abc\n          comment_id: AAAC\n" +
		"          at: 2026-09-07T10:00:00Z\n          quoted: reviewed annually\n"
}

// TestAProposalIsRecordedUnderItsOwnDocument is the other half of the URL rule:
// the entry that grows is the one the run was of, and the other document's list
// is left exactly as it was. A proposal recorded under the wrong document is a
// suggestion withdraw would refuse to take back.
func TestAProposalIsRecordedUnderItsOwnDocument(t *testing.T) {
	stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	from := tempFile(t, "proposals.json", oneProposal)
	note := twoEntryNote(t, otherDocID, proposeDocID,
		"      proposals:\n        - id: suggest.older\n          comment_id: AAAB\n          at: 2026-09-07T09:00:00Z\n")

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID, "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("propose: %v (exit %d)", got, code)
	}

	grown := entryOf(t, note, proposeDocID).Proposals
	if len(grown) != 2 || grown[0].ID != "suggest.older" || grown[1].ID != "suggest.abc" {
		t.Errorf("documents[1].proposals = %+v, want the older one and the new one in that order", grown)
	}
	if left := entryOf(t, note, otherDocID).Proposals; len(left) != 0 {
		t.Errorf("the other document's proposals grew: %+v", left)
	}
}

// TestWithdrawNeverSendsAnIDFromAnotherDocument is the permission rule under the
// list. A suggestion id recorded under one document says nothing about another:
// reading the whole block would let a proposal in document A be the permission
// to reject an id in document B, in somebody else's document.
func TestWithdrawNeverSendsAnIDFromAnotherDocument(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	// The proposal sits under the first entry, and the run is of the second.
	src := "---\ngdoc:\n  schema: 2\n  documents:\n    - id: " + otherDocID +
		"\n      proposals:\n        - id: suggest.abc\n          comment_id: AAAC\n          at: 2026-09-07T10:00:00Z\n" +
		"    - id: " + withdrawDocID + "\n---\n\n# Scope\n"
	note := tempFile(t, "split-proposals.md", src)

	got, code := runJSON(t, "withdraw", withdrawDocID, "suggest.abc", "--md", note)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a suggestion recorded under another document must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "suggest.abc") || !strings.Contains(msg, note) {
		t.Errorf("the refusal %q must name the suggestion and the note", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may be sent for a suggestion this document does not record: %v", f.calls)
	}
	// And the record under the other document is untouched, because the refusal
	// is about which document it belongs to rather than about the id.
	if len(entryOf(t, note, otherDocID).Proposals) != 1 {
		t.Error("the other document's proposal was taken out of the note")
	}
}

// TestGoneSinceReadsTheSnapshotOfTheDocumentRead is what the snapshot is for,
// under the list. Each document has its own history, and comparing this read
// against another document's snapshot would report that document's suggestions
// as gone from this one.
func TestGoneSinceReadsTheSnapshotOfTheDocumentRead(t *testing.T) {
	stubSession(t, docsAndComments(t))
	stubNow(t, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	seen := func(id string) string {
		return "      suggestions_seen:\n        at: 2026-09-01T09:00:00Z\n        items:\n" +
			"          - id: suggest.a1\n            kind: deletion\n            section: Scope\n            text: annually\n" +
			"          - id: " + id + "\n            kind: insertion\n            section: Controls\n            text: \"quarterly \"\n"
	}
	src := "---\ngdoc:\n  schema: 2\n  documents:\n    - id: " + otherDocID + "\n" + seen("suggest.gone-from-a") +
		"    - id: " + fixtureDocID + "\n" + seen("suggest.gone-from-b") + "---\n\n# Scope\n"
	note := tempFile(t, "two-snapshots.md", src)

	got, code := runJSON(t, "suggestions", fixtureDocID, "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("suggestions: %v (exit %d)", got, code)
	}
	gone, _ := dataOf(t, got)["gone_since_last_look"].([]any)
	if len(gone) != 1 {
		t.Fatalf("gone = %v, want the one suggestion this document's snapshot named", gone)
	}
	first, _ := gone[0].(map[string]any)
	if first["id"] != "suggest.gone-from-b" {
		t.Errorf("gone[0] = %v, want the entry read from the document the URL named", first)
	}
	// The other document's snapshot is where it was, because this read says
	// nothing about it.
	other := entryOf(t, note, otherDocID).SuggestionsSeen
	if other == nil || len(other.Items) != 2 || other.Items[1].ID != "suggest.gone-from-a" {
		t.Errorf("the other document's snapshot moved: %+v", other)
	}
}

// TestEveryWriterRefusesACopyThatNamesANote is the copy rule. A note gdoc wrote
// beside somebody else's document carries exported.note, which names the note
// the document was exported next to. It is a copy rather than a source: there
// is nothing in it to record, nothing that would ever be published from it, and
// a snapshot written into it would be read next time as that document's own
// history.
func TestEveryWriterRefusesACopyThatNamesANote(t *testing.T) {
	const theNote = "notes/supplier-register.md"
	copyNote := func(t *testing.T, id string) string {
		t.Helper()
		src := "---\ngdoc:\n  schema: 2\n  documents:\n    - id: " + id +
			"\n      exported:\n        at: 2026-09-19T08:00:00Z\n        note: " + theNote + "\n---\n\n# Scope\n"
		return tempFile(t, "exported-copy.md", src)
	}

	t.Run("suggestions", func(t *testing.T) {
		stubSession(t, docsAndComments(t))
		note := copyNote(t, fixtureDocID)
		before := mustRead(t, note)

		got, code := runJSON(t, "suggestions", fixtureDocID, "--md", note)
		if code == 0 || got["ok"] != false {
			t.Fatalf("a copy must be refused: %v (exit %d)", got, code)
		}
		if msg, _ := got["error"].(string); !strings.Contains(msg, theNote) {
			t.Errorf("the refusal %q does not name the note the copy came from", msg)
		}
		if mustRead(t, note) != before {
			t.Error("a refused run wrote to the copy")
		}
	})

	t.Run("propose", func(t *testing.T) {
		f := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
		from := tempFile(t, "proposals.json", oneProposal)
		note := copyNote(t, proposeDocID)

		got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID, "--md", note)
		if code == 0 || got["ok"] != false {
			t.Fatalf("a copy must be refused: %v (exit %d)", got, code)
		}
		if msg, _ := got["error"].(string); !strings.Contains(msg, theNote) {
			t.Errorf("the refusal %q does not name the note the copy came from", msg)
		}
		if len(f.writes()) != 0 {
			t.Errorf("nothing may be written before the note is checked: %v", f.writes())
		}
	})

	t.Run("withdraw", func(t *testing.T) {
		f := stubWire(t, &fakeWire{})
		note := copyNote(t, withdrawDocID)

		got, code := runJSON(t, "withdraw", withdrawDocID, "suggest.abc", "--md", note)
		if code == 0 || got["ok"] != false {
			t.Fatalf("a copy must be refused: %v (exit %d)", got, code)
		}
		if msg, _ := got["error"].(string); !strings.Contains(msg, theNote) {
			t.Errorf("the refusal %q does not name the note the copy came from", msg)
		}
		if len(f.calls) != 0 {
			t.Errorf("nothing may be sent before the note is checked: %v", f.calls)
		}
	})
}

// TestTheReplySaysOnceWhenTheBlockWasRewritten is the migration told to the
// person running it. A note published before 2026-09-19 carries schema 1, and
// the first run that writes to it rewrites the block: that is a change to
// somebody's file, so it is said out loud. The next run reads schema 2 and has
// nothing to say.
func TestTheReplySaysOnceWhenTheBlockWasRewritten(t *testing.T) {
	stubSession(t, docsAndComments(t))
	stubNow(t, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	note := copyToTemp(t, "paired.md")

	got, code := runJSON(t, "suggestions", fixtureDocID, "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("suggestions: %v (exit %d)", got, code)
	}
	warns := warningsOf(t, got)
	if !hasWarning(warns, "rewritten") || !hasWarning(warns, note) {
		t.Fatalf("the first run must say the block was rewritten, naming the note: %v", warns)
	}
	if b := blockOf(t, note); b.Schema != frontmatter.Schema {
		t.Fatalf("the note is still schema %d after the write", b.Schema)
	}

	got, code = runJSON(t, "suggestions", fixtureDocID, "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("the second run: %v (exit %d)", got, code)
	}
	if warns := warningsOf(t, got); hasWarning(warns, "rewritten") {
		t.Errorf("the second run said it again: %v", warns)
	}
}

// A schema 1 note that nothing changes in is a note nothing is written to, so
// nothing says it was rewritten either. build reads a note and never writes to
// it, which is the case that proves the warning follows the write rather than
// the read.
func TestAReadThatWritesNothingNeverSaysTheBlockWasRewritten(t *testing.T) {
	stubSession(t, docsAndComments(t))
	note := copyToTemp(t, "paired.md")
	before := mustRead(t, note)

	got, code := runJSON(t, "suggestions", fixtureDocID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("suggestions: %v (exit %d)", got, code)
	}
	if warns := warningsOf(t, got); hasWarning(warns, "rewritten") {
		t.Errorf("a run with no --md said a note was rewritten: %v", warns)
	}
	if mustRead(t, note) != before {
		t.Error("a run with no --md wrote to a note")
	}
	if filepath.Base(note) != "paired.md" {
		t.Errorf("the note moved: %s", note)
	}
}
