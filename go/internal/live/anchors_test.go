package live

// The cheap half of M7b's acceptance: does an in-place restyle keep a comment
// attached to its words, and a pending suggestion pending.
//
// TestLiveRestylePreservesTenFeatures is the full acceptance and needs a
// document holding all ten of SPEC item 5's features, which is a document
// somebody has to build by hand and keep intact. This one needs a document with
// one anchored comment and one pending suggestion, which is any document
// somebody has reviewed. It answers the question the whole milestone rests on,
// and it can be run against a real document on any day.
//
// The question is worth stating. The 2026-08-29 measurement is why in-place
// styling exists at all: replacing a document's body destroyed 100% of comment
// anchors, 355 of 355 characters across three anchors, and recreated every
// suggestion id. In-place batchUpdate preserved them. Nothing in this code has
// asked Google whether that still holds, and at LevelInPlace the guard and the
// read-back are the only two bars there are.
//
// It writes only to a copy, and it leaves the copy behind on purpose: the
// numbers below say the anchors survived, and a person opening the copy is what
// says the document looks right. Trash it when you have looked at it.

import (
	"context"

	"os"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/restyle"
)

// anchorVar names the document to copy and restyle. No default, for
// GDOC_LIVE_IDEAL_DOC_ID's reason: this run copies the whole of somebody's
// document, comments included, into gdoc's folder.
const anchorVar = "GDOC_LIVE_ANCHOR_DOC_ID"

func TestLiveRestyleKeepsAnchorsAndSuggestions(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 and %s=<document id> to copy a document into the Drive test folder and restyle the copy",
			liveVar, writeVar, anchorVar)
	}
	source := strings.TrimSpace(os.Getenv(anchorVar))
	if source == "" {
		t.Fatalf("%s=1 with %s=1 needs %s=<document id>: the copy grant names exactly one source, and there is no default", liveVar, writeVar, anchorVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	ctx := context.Background()

	// The source is handed in to be read and copied, and nothing more. It is at
	// LevelSuggest with no in-place grant, so a restyle of it would be refused
	// by the guard before it left the machine.
	p := guard.NewPolicy()
	p.AllowFile(source, guard.LevelSuggest)
	p.AllowCreateIn(folder)
	p.AllowCopy(source)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}

	original, err := docs.Fetch(ctx, s, source)
	if err != nil {
		t.Fatalf("the source document %q could not be read: %v", source, err)
	}
	t.Logf("source %s (%q), revision %s, %d tabs", source, original.Title, original.RevisionID, len(original.Tabs))
	// Registered before anything is created, so it runs on every path out of
	// this test, the failing ones included.
	t.Cleanup(func() { assertOriginalUnmoved(t, ctx, s, source, original.RevisionID) })

	copyID := copyDocument(t, ctx, s, source, folder)
	t.Cleanup(func() {
		t.Logf("the copy is left for a person to look at: https://docs.google.com/document/d/%s/edit", copyID)
		t.Logf("trash it with: File > Move to bin, or leave it in folder %s", folder)
	})

	before := readEverything(t, ctx, s, copyID)
	beforeAnchored, beforeThreads := anchoredThreads(before.report)
	beforePending := sortedIDs(before.report.Suggestions.IDs)
	t.Logf("before: %d threads (%d anchored), %d pending %v",
		beforeThreads, beforeAnchored, len(beforePending), beforePending)

	// The setup gate, and it is a gate rather than an assertion for the reason
	// the ten-feature run has one: a copy that carried nothing would report the
	// copy's own limits as damage the restyle did.
	if beforeAnchored == 0 {
		t.Fatalf("the copy carries no anchored comment, so a restyle of it would measure the copy rather than the write. Check that %s has a comment attached to text", source)
	}
	if len(beforePending) == 0 {
		t.Fatalf("the copy carries no pending suggestion, so nothing here would say a suggestion survived. Check that %s has one pending", source)
	}

	requests, plan := restyleCopy(t, ctx, copyID)
	t.Logf("sent %d requests: %d paragraphs, %d runs, %d cells, %d tables",
		len(requests), plan.Paragraphs, plan.Text, plan.Cells, plan.Tables)

	after := readEverything(t, ctx, s, copyID)
	afterAnchored, afterThreads := anchoredThreads(after.report)
	afterPending := sortedIDs(after.report.Suggestions.IDs)
	t.Logf("after:  %d threads (%d anchored), %d pending %v",
		afterThreads, afterAnchored, len(afterPending), afterPending)

	// The two facts the milestone rests on. The witness is the docx export, in
	// both reads, because comments.list reports a destroyed anchor as healthy.
	if afterThreads != beforeThreads {
		t.Errorf("threads went from %d to %d: a restyle changes no thread", beforeThreads, afterThreads)
	}
	if afterAnchored != beforeAnchored {
		t.Errorf("anchored comments went from %d to %d. This is the 2026-08-29 failure, and in-place styling exists to avoid it",
			beforeAnchored, afterAnchored)
	}
	if strings.Join(afterPending, ",") != strings.Join(beforePending, ",") {
		t.Errorf("the pending suggestions changed from %v to %v: a restyle accepts, rejects and recreates none of them",
			beforePending, afterPending)
	}

	// The production read-back on top of those, because what the binary reports
	// to the skill is what a person acts on.
	for _, w := range after.report.NamedRanges {
		t.Logf("named range still there: %s (%s)", w.Name, w.ID)
	}
}

// anchoredThreads counts the threads the export still finds attached to text,
// and the threads there are. Both are facts read out of the survey, and neither
// is a verdict about whether the restyle was good.
func anchoredThreads(rep restyle.Report) (anchored, total int) {
	for _, w := range rep.Threads.Witnessed {
		total++
		if w.Witness == "anchored" {
			anchored++
		}
	}
	return anchored, total
}

// sortedIDs is a copy in a fixed order, so two reads of one document compare as
// strings rather than depending on the order Docs happened to answer in.
func sortedIDs(ids []string) []string {
	out := append([]string{}, ids...)
	sort.Strings(out)
	return out
}
