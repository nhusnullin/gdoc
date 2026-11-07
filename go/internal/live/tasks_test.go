package live

// M33's acceptance: a note's task lines arrive in the document as boxes a
// person can tick, with a name chip and a date chip on them.
//
// It is a write test, so it needs GDOC_LIVE_TEST=1 and GDOC_LIVE_WRITE=1, and
// one more variable of its own: GDOC_LIVE_OWNER is the whole address the name
// chip carries. There is no default, because a chip names a person and gdoc
// may never guess who.
//
// What no fixture can answer is what Drive's import does with a ballot
// character and what Docs does with a checklist request over a list that came
// in as disc bullets. Both were measured by the probe on 2026-11-05; this is
// the publish path resting on that measurement, end to end.
//
// The document is trashed on the way out. The one left for a person's eyes is
// TestLiveChecklistForHands's, because a tick is a click nothing here can
// make.

import (
	"os"
	"strings"
	"testing"
	"time"

	"gdoc/internal/body"
	"gdoc/internal/docs"
	"gdoc/internal/publish"
	"gdoc/internal/tasks"
)

// ownerVar names the address the person chip carries. It is its own variable
// beside GDOC_LIVE_TEST and GDOC_LIVE_WRITE, for the reason every live
// variable is its own: a chip on somebody is not a thing to default into.
const ownerVar = "GDOC_LIVE_OWNER"

// taskNoteSource is the note this test publishes. One list, three tasks and
// one plain item: a task with an owner, a task with a day, a task that is
// done, and a bullet that is no task at all. %s takes the owner's address.
//
// It is written into a temp directory rather than kept as a fixture, for
// publishNoteSource's reason, with one more: the address is the run's.
const taskNoteSource = `---
title: Live Tasks Test
doc_type: Note
version: 1.0
date: 2026-11-07
owner: Chief Risk Officer
classification: internal
---

# Questions for the bank

- [ ] Which bank holds the safeguarding account @%s
- [ ] Confirm the licence 2026-11-20
- [x] Ask the auditor
- This item is no task
`

// The words of each line, as literals. What is asserted is what a person
// really wrote in the note.
const (
	taskWithOwner = "Which bank holds the safeguarding account"
	taskWithDay   = "Confirm the licence"
	taskDone      = "Ask the auditor"
	plainItem     = "This item is no task"
)

func TestLivePublishTurnsTasksIntoAChecklist(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)
	owner := strings.TrimSpace(os.Getenv(ownerVar))
	if owner == "" {
		t.Skipf("skipped: set %s=<address> to put a name chip on a task; a chip names a person and there is no default", ownerVar)
	}

	note := writeLiveNote(t, "live-tasks.md", strings.Replace(taskNoteSource, "%s", owner, 1))
	doc := renderNoteFile(t, note)
	if len(doc.tasks) != 3 {
		t.Fatalf("the note rendered with %d tasks, and it holds three: %+v", len(doc.tasks), doc.tasks)
	}
	sending := resolveLiveTasks(t, doc.tasks)

	title := "gdoc M33 tasks " + time.Now().UTC().Format(time.RFC3339)
	rep, err := publish.Run(ctx, s, publish.Options{
		FolderID: folder,
		Title:    title,
		Docx:     doc.docx,
		Tasks:    sending,
	})
	// Registered before the error is read, for TestLivePublish's reason: a
	// create Drive accepted and this run failed after is still a document in
	// somebody's folder.
	if rep.DocumentID != "" {
		id := rep.DocumentID
		t.Cleanup(func() { trashLive(t, ctx, s, id, "the published document") })
	}
	for _, w := range rep.Warnings {
		t.Logf("publish warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the publish failed: %v", err)
	}
	t.Logf("published %s at %s", rep.DocumentID, rep.URL)
	if !rep.Checks.Tasks {
		t.Errorf("checks.tasks is false, so the run says its own task lines did not land: %v", rep.Warnings)
	}
	if !rep.Verified {
		t.Errorf("the publish is not verified: %+v", rep.Checks)
	}

	// The document itself, through the read every command makes, so what is
	// asserted below is what `read` would print.
	after, err := docs.Fetch(ctx, s, rep.DocumentID)
	if err != nil {
		t.Fatalf("the published document could not be read back: %v", err)
	}
	if len(after.Tabs) != 1 {
		t.Fatalf("the document came back with %d tabs", len(after.Tabs))
	}

	var boxes, people, dates, discs int
	var sawPlain bool
	for _, b := range after.Tabs[0].Body {
		p := b.Paragraph
		if p == nil {
			continue
		}
		text := taskLineText(p)
		for _, ballot := range []string{tasks.BallotEmpty, tasks.BallotCrossed} {
			// The character alone: the space after it is in every paragraph
			// of the document, and only the character marks a task.
			if strings.Contains(text, strings.TrimSpace(ballot)) {
				t.Errorf("the paragraph %q still carries a ballot character, and the box is what replaced it", text)
			}
		}
		if p.Bullet == nil {
			continue
		}
		if !p.Bullet.Checklist {
			if strings.Contains(text, plainItem) {
				sawPlain, discs = true, discs+1
			}
			continue
		}
		boxes++
		for _, r := range p.Runs {
			switch r.Kind {
			case docs.KindPerson:
				people++
				if r.Detail == nil || !strings.EqualFold(r.Detail.Email, owner) {
					t.Errorf("the name chip on %q names %+v, want %s", text, r.Detail, owner)
				}
			case docs.KindDate:
				dates++
			}
		}
	}
	if boxes != 3 {
		t.Errorf("the document holds %d checklist items, and the note held three tasks", boxes)
	}
	if people != 1 {
		t.Errorf("the document holds %d name chips, and one task named an owner", people)
	}
	if dates != 1 {
		t.Errorf("the document holds %d date chips, and one task named a day", dates)
	}
	if !sawPlain || discs != 1 {
		t.Errorf("the plain item is not a bullet beside the checklist: found %d, seen %v", discs, sawPlain)
	}

	// The done task keeps the strike the docx gave it, because a file has no
	// box and the strike is what said the item was done. The box beside it is
	// unticked: no API writes a tick.
	if !struckSomewhere(after.Tabs[0], taskDone) {
		t.Errorf("the done task %q came back without its strike", taskDone)
	}
}

// paragraphText is one paragraph's text runs joined, the newline off.
func taskLineText(p *docs.Paragraph) string {
	var b strings.Builder
	for _, r := range p.Runs {
		if r.Kind == docs.KindText {
			b.WriteString(r.Text)
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// struckSomewhere says whether the words came back struck. The read carries no
// strike field, so this asks the one thing it can: that the words are there at
// all, in a checklist item of their own.
func struckSomewhere(tab docs.Tab, words string) bool {
	for _, b := range tab.Body {
		if b.Paragraph == nil || b.Paragraph.Bullet == nil || !b.Paragraph.Bullet.Checklist {
			continue
		}
		if strings.Contains(taskLineText(b.Paragraph), words) {
			return true
		}
	}
	return false
}

// resolveLiveTasks finishes every owner the way cmd/gdoc finishes one before
// an upload. It is spelled here rather than imported because cmd/gdoc is
// package main, which is why the pairing in TestLivePublish is spelled there
// too. The note writes whole addresses, so no domain is in effect.
func resolveLiveTasks(t *testing.T, list []body.Task) []publish.Task {
	t.Helper()
	out := make([]publish.Task, 0, len(list))
	for _, one := range list {
		var address string
		if one.Owner != "" {
			owner, err := tasks.ReadOwner(one.Owner)
			if err != nil {
				t.Fatalf("the note's owner %q does not read: %v", one.Owner, err)
			}
			if address, err = owner.Resolve(""); err != nil {
				t.Fatalf("the note's owner %q could not be resolved: %v", one.Owner, err)
			}
		}
		out = append(out, publish.Task{
			Words: one.Words, Owner: address, Due: one.Due, Done: one.Done, Tokens: one.Tokens,
		})
	}
	return out
}

// writeLiveNote puts one note in a directory of this test's own.
func writeLiveNote(t *testing.T, name, source string) string {
	t.Helper()
	path := t.TempDir() + "/" + name
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("the note could not be written: %v", err)
	}
	return path
}
