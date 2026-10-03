package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// This file is the bound on the context the commands are handed. A caller that
// stops the call stops the reads, and a write is never cut from its read-backs:
// a suggestion in somebody's document that gdoc did not read back is a change
// nobody can account for.
//
// The two halves are one rule. Before an item, the context decides whether to
// go on. Inside an item, it does not: the send and the three read-backs run on
// a context nothing cancels, and what the caller's deadline costs is the items
// that were not reached.

// cancelledEnvelope is the sentence a run that ran out of time prints. It is a
// literal here, as every value this package pins is, so a reword is a test
// somebody has to change on purpose.
const (
	proposeRanOut  = "the time for this call ran out after 1 of 3; nothing after that was sent. Call again with the rest"
	annotateRanOut = "the time for this call ran out after 1 of 2; nothing after that was sent. Call again with the rest"
	replyRanOut    = "the time for this call ran out before the reply was sent, so nothing was written. Call again"
)

// done is a context that has already been cancelled, which is what a command
// is handed when the caller stopped the call before it started.
func done(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// TestACancelledContextCancelsTheRead is the read half. The three read commands
// are handed a context that is already done: nothing reaches the wire, and the
// envelope fails naming the cancellation rather than blaming Google for a read
// that was never made.
func TestACancelledContextCancelsTheRead(t *testing.T) {
	for _, words := range [][]string{
		{"read", fixtureDocID},
		{"suggestions", fixtureDocID},
		{"comments", fixtureDocID},
	} {
		t.Run(words[0], func(t *testing.T) {
			f := stubSession(t, docsAndComments(t))

			got, code := runJSONCtx(t, done(t), words...)
			if code == 0 || got["ok"] != false {
				t.Fatalf("a cancelled call must fail: %v (exit %d)", got, code)
			}
			if msg, _ := got["error"].(string); !strings.Contains(msg, "context canceled") {
				t.Errorf("error = %q, and the cancellation is what stopped the read", msg)
			}
			if len(f.urls) != 0 {
				t.Errorf("%v reached the wire on a call that was already stopped", f.urls)
			}
		})
	}
}

// proposeRun is a propose run of n proposals, with a hook on the first one's
// write and a hook on its first read-back. The hooks are how a test stands
// where the caller's deadline lands in the middle of one proposal.
//
// The preview and the export come first in the list and are never spent,
// because find takes the first unused answer whose match string is in the URL.
// Everything else is once, so the list is the order: the command's own read,
// then a read, a write and a read-back per proposal.
func proposeRun(t *testing.T, n int, onWrite, onReadBack func()) []*answer {
	t.Helper()
	out := []*answer{
		{method: "GET", match: "PREVIEW_WITHOUT_SUGGESTIONS", json: readFixture(t, "propose-preview.json")},
		{method: "GET", match: "/export?", bytes: exportWithComment(t, true)},
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-before.json"), once: true},
	}
	for i := 0; i < n; i++ {
		write := &answer{method: "POST", match: proposeDocID + ":batchUpdate",
			json: readFixture(t, "propose-batch.json"), once: true}
		readBack := &answer{method: "GET", match: proposeDocID + "?includeTabsContent",
			json: readFixture(t, "propose-after.json"), once: true}
		if i == 0 {
			write.before = onWrite
			readBack.before = onReadBack
		}
		out = append(out,
			&answer{method: "GET", match: proposeDocID + "?includeTabsContent",
				json: readFixture(t, "propose-before.json"), once: true},
			write, readBack)
	}
	return out
}

// batchesOn counts the writes that reached the wire, which is how far the loop
// really got whatever the envelope says.
func batchesOn(f *fakeWire) int {
	var n int
	for _, c := range f.writes() {
		if strings.Contains(c.URL, ":batchUpdate") {
			n++
		}
	}
	return n
}

// TestProposeStopsBetweenProposalsWhenTimeRunsOut is the write half. The
// caller's time runs out while the first proposal is being read back: that
// proposal finishes all three read-backs and reports them, and the two behind
// it never leave the machine. The sentence says how far the run got, so the
// next call knows what is left rather than sending the file again.
func TestProposeStopsBetweenProposalsWhenTimeRunsOut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := stubWire(t, &fakeWire{answers: proposeRun(t, 3, nil, cancel)})
	from := tempFile(t, "proposals.json", threeProposals)

	got, code := runJSONCtx(t, ctx, "propose", proposeDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a run that ran out of time must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); msg != proposeRanOut {
		t.Errorf("error = %q, want %q", msg, proposeRanOut)
	}
	list, _ := dataOf(t, got)["proposals"].([]any)
	if len(list) != 3 {
		t.Fatalf("proposals = %v, want one entry per proposal in the file", list)
	}
	first, _ := list[0].(map[string]any)
	if first["sent"] != true {
		t.Errorf("the first proposal was written and reports sent = %v", first["sent"])
	}
	// All three read-backs ran after the cancellation, which is the whole of
	// the rule: a proposal gdoc sent is a proposal gdoc read back.
	checks, _ := first["checks"].(map[string]any)
	if checks["suggestions_inline"] != true || checks["preview_without_suggestions"] != true ||
		checks["docx_anchored"] != true {
		t.Errorf("checks = %v, and the cancellation may not cut a read-back", checks)
	}
	for i, entry := range list[1:] {
		one, _ := entry.(map[string]any)
		if one["sent"] != false {
			t.Errorf("proposal %d was sent after the time ran out: %v", i+2, one)
		}
	}
	if n := batchesOn(f); n != 1 {
		t.Errorf("%d batches reached the wire, and only the first proposal may be sent: %v", n, f.writes())
	}
}

// TestAReadBackIsNeverCutByTheDeadline is the same rule with nothing behind it.
// The caller's time runs out between the write and the first read-back, and all
// three read-backs are still made: the suggestion is in the document, and what
// it says about it is the only account there is.
func TestAReadBackIsNeverCutByTheDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := stubWire(t, &fakeWire{answers: proposeRun(t, 1, cancel, nil)})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSONCtx(t, ctx, "propose", proposeDocID, "--from", from)
	if code != 0 || got["ok"] != true {
		t.Fatalf("the one proposal was sent and read back: %v (exit %d)", got, code)
	}
	list, _ := dataOf(t, got)["proposals"].([]any)
	if len(list) != 1 {
		t.Fatalf("proposals = %v, want the one in the file", list)
	}
	one, _ := list[0].(map[string]any)
	if one["sent"] != true || one["verified"] != true {
		t.Errorf("proposals[0] = %v, want it sent and verified", one)
	}
	checks, _ := one["checks"].(map[string]any)
	if checks["suggestions_inline"] != true || checks["preview_without_suggestions"] != true ||
		checks["docx_anchored"] != true {
		t.Errorf("checks = %v, want all three routes read after the cancellation", checks)
	}
	if n := batchesOn(f); n != 1 {
		t.Errorf("%d batches reached the wire, want the one: %v", n, f.writes())
	}
}

// annotateRun is annotateAnswers with a hook on the first batch, so a test can
// stand where the caller's time runs out with one comment in the document.
func annotateRun(t *testing.T, onWrite func()) []*answer {
	t.Helper()
	second := strings.Replace(readFixture(t, "annotate-batch.json"), "AAACOne", "AAACTwo", 1)
	return []*answer{
		{method: "POST", match: annotateDocID + ":batchUpdate",
			json: readFixture(t, "annotate-batch.json"), once: true, before: onWrite},
		{method: "POST", match: annotateDocID + ":batchUpdate", json: second, once: true},
		{method: "GET", match: annotateDocID + "?includeTabsContent", json: readFixture(t, "annotate-before.json")},
		{method: "GET", match: "/comments?", json: readFixture(t, "annotate-comments.json")},
		{method: "GET", match: "/export?", bytes: annotateExport(t)},
	}
}

// TestAnnotateStopsBetweenItemsWhenTimeRunsOut is propose's rule on the other
// writer. The first comment is in the document and read back through both
// routes; the second is not sent, and the sentence says so.
func TestAnnotateStopsBetweenItemsWhenTimeRunsOut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := stubWire(t, &fakeWire{answers: annotateRun(t, cancel)})
	from := tempFile(t, "annotations.json", readFixture(t, "annotations.json"))

	got, code := runJSONCtx(t, ctx, "annotate", annotateDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a run that ran out of time must fail: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); msg != annotateRanOut {
		t.Errorf("error = %q, want %q", msg, annotateRanOut)
	}
	list := annotationsOf(t, got)
	if len(list) != 2 {
		t.Fatalf("annotations = %v, want one entry per annotation in the file", list)
	}
	if list[0]["sent"] != true || list[0]["verified"] != true {
		t.Errorf("annotations[0] = %v, want it placed and read back", list[0])
	}
	checks, _ := list[0]["checks"].(map[string]any)
	if checks["drive_listing"] != true || checks["docx_anchored"] != true {
		t.Errorf("checks = %v, and the cancellation may not cut a read-back", checks)
	}
	if list[1]["sent"] != false {
		t.Errorf("annotations[1] was sent after the time ran out: %v", list[1])
	}
	if len(f.writes()) != 1 {
		t.Errorf("%d writes reached the wire, and only the first comment may be sent: %v",
			len(f.writes()), f.writes())
	}
}

// TestReplyNeverCutsItsReadBack is the one-write command under the same rule.
// A call that is already over posts nothing. A call whose time runs out as the
// reply goes out still reads the thread back, because a reply in somebody's
// document reported as unverified is a reply nobody can find again.
func TestReplyNeverCutsItsReadBack(t *testing.T) {
	t.Run("a call that is already over posts nothing", func(t *testing.T) {
		f := stubWire(t, &fakeWire{})
		path := tempFile(t, "reply.txt", "🤖 The 2026 register.\n")

		got, code := runJSONCtx(t, done(t), "reply", fixtureDocID, "AAAA1111", "--body-file", path)
		if code == 0 || got["ok"] != false {
			t.Fatalf("a cancelled call must fail: %v (exit %d)", got, code)
		}
		if msg, _ := got["error"].(string); msg != replyRanOut {
			t.Errorf("error = %q, want %q", msg, replyRanOut)
		}
		if len(f.calls) != 0 {
			t.Errorf("%v reached the wire on a call that was already stopped", f.calls)
		}
	})

	t.Run("a reply that went out is read back", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		f := stubWire(t, &fakeWire{answers: []*answer{
			{method: "POST", match: "/comments/AAAA1111/replies", before: cancel,
				json: `{"id":"R1","createdTime":"2026-09-06T10:45:00Z","content":"🤖 The 2026 register."}`},
			{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
		}})
		path := tempFile(t, "reply.txt", "🤖 The 2026 register.\n")

		got, code := runJSONCtx(t, ctx, "reply", fixtureDocID, "AAAA1111", "--body-file", path)
		if code != 0 || got["ok"] != true {
			t.Fatalf("the reply was posted and read back: %v (exit %d)", got, code)
		}
		data := dataOf(t, got)
		if data["reply_id"] != "R1" || data["verified"] != true {
			t.Errorf("data = %v, want the reply id and a read-back that found it", data)
		}
		posts := f.writes()
		if len(posts) != 1 {
			t.Fatalf("one reply is one write: %v", posts)
		}
		var body map[string]any
		if err := json.Unmarshal(posts[0].Body, &body); err != nil {
			t.Fatal(err)
		}
		if body["content"] != "🤖 The 2026 register." {
			t.Errorf("content = %q", body["content"])
		}
	})
}
