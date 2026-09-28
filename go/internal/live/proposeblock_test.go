package live

// M14's live test: the block kind end to end, in a document it creates and
// trashes.
//
// TestLiveBlockProposalProbe measured the shape on 2026-09-27, one request list
// at a time, written by hand. This one asks the other question: the production
// writer, over the production placement, verified by the production read-backs,
// on a real document. It is here rather than in live_test.go because that file
// is M3's and M4's and is already over 700 lines.
//
// Two things nothing else has measured run through it. A numbered list in
// SUGGEST mode was never sent before this test, and the bullet the new text
// inherits is cleared here rather than in a hand-written batch: the replace
// begins at a list item, so the paragraphs going in front of it arrive
// bulleted unless deleteParagraphBullets holds.
//
// Everything is asserted rather than logged. The unit tests already cover a
// route that did not hold, and what this run is for is Google agreeing with
// all three, twice, and the document coming back the way it started once both
// proposals are withdrawn.

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/drive"
	"gdoc/internal/frontmatter"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/probe"
	"gdoc/internal/propose"
	"gdoc/internal/view"
	"gdoc/internal/withdraw"
)

// The five paragraphs the subject document holds, in order and each with its
// own paragraph mark. The third is made a list item by the setup batch, and it
// is the first paragraph the replace covers: the block's text goes in at that
// paragraph's start, so it inherits the bullet, and the clearing this test is
// here for is what takes it off again.
//
// Every quote below occurs exactly once in the whole document, which is what
// FindSpan asks for, and no quote carries a line break.
var blockSubjectParagraphs = []string{
	"The supplier register is reviewed annually by the operations team.\n",
	"Every exception to the review is recorded in the register itself.\n",
	"The old section opens with this list item.\n",
	"The old section closes with this paragraph.\n",
	"The closing paragraph is the one nothing in this test touches.\n",
}

// blockSubjectItem is which of those paragraphs is a list item, counted from
// zero.
const blockSubjectItem = 2

const (
	// The after block: a heading, a paragraph with a mark on it, a bulleted
	// list and a numbered list. Its first line is in the document nowhere
	// else, which is what the preview read-back needs to answer at all.
	blockAfterQuoted  = "reviewed annually by the operations team"
	blockAfterContent = "## Escalation path\n" +
		"\n" +
		"An exception nobody closes in five working days is **escalated** to the risk team.\n" +
		"\n" +
		"- The reviewer names the exception in the register.\n" +
		"- The owner answers in the same row.\n" +
		"\n" +
		"1. The risk team reads the register every Monday.\n" +
		"2. The board sees whatever is open at the end of the month.\n"
	blockAfterWhy = "The policy says exceptions are escalated and the document never says to whom, so this is the missing section."

	// The replace: two whole paragraphs out, three new ones in, starting at
	// the list item.
	blockReplaceFrom    = "The old section opens"
	blockReplaceTo      = "The old section closes"
	blockReplaceContent = "The register section is rewritten here as plain paragraphs.\n" +
		"\n" +
		"- It opens with a paragraph rather than with a list item.\n" +
		"- It says the same thing in fewer words.\n"
	blockReplaceWhy = "The two paragraphs it stands in for say the same thing twice, and one of them is a list item for no reason."
)

// TestLiveProposeBlock proposes a block after a paragraph and a block in place
// of two paragraphs, asserts both landed as one pending suggestion with all
// three read-backs holding, withdraws each, and reads the document back
// character for character what it was before.
//
// Every document it touches is one it created. The policy has one door open,
// AllowCreateIn on the folder, so the subject document is reachable only
// because the guard carried the create that made it, and no document of Nail's
// is in the reachable set at all.
func TestLiveProposeBlock(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create a document in the Drive test folder and propose blocks into it; %s names another folder", liveVar, writeVar, folderVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	t.Logf("creating in folder %s", folder)

	p := guard.NewPolicy()
	p.AllowCreateIn(folder)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	ctx := context.Background()

	// The probe first, because it is what the command runs before it writes:
	// an unenrolled project makes every assertion below meaningless, and the
	// probe is the one that says so.
	report, err := probe.Run(ctx, s, folder)
	for _, w := range report.Warnings {
		t.Logf("probe warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the probe failed, and its document is %q: %v", report.ProbeDocumentID, err)
	}
	if !report.Enrolled {
		t.Fatalf("the probe says SUGGEST is not honoured today, so nothing below can be a suggestion; probe document %q", report.ProbeDocumentID)
	}
	if !report.Trashed {
		t.Errorf("the probe document %q was left in the folder", report.ProbeDocumentID)
	}

	docID := blockSubject(t, ctx, s, folder)
	before := blockBodyText(t, ctx, s, docID, "before the proposals")

	after := proposeBlock(t, ctx, s, docID, "the after block", propose.Proposal{
		Kind:    propose.KindBlock,
		After:   blockAfterQuoted,
		Content: blockAfterContent,
		Why:     blockAfterWhy,
	})
	replace := proposeBlock(t, ctx, s, docID, "the replace block", propose.Proposal{
		Kind:        propose.KindBlock,
		ReplaceFrom: blockReplaceFrom,
		ReplaceTo:   blockReplaceTo,
		Content:     blockReplaceContent,
		Why:         blockReplaceWhy,
	})

	// The replace goes back first. Nothing requires that order, and it is the
	// one a person would take: the last change made is the first one undone.
	withdrawBlocks(t, ctx, p, s, docID, []propose.Result{replace, after})

	if got := blockBodyText(t, ctx, s, docID, "after both withdrawals"); got != before {
		t.Errorf("the document does not read as it did before the two blocks were proposed:\nbefore %q\nafter  %q", before, got)
	}
}

// blockSubject makes the document the two proposals are written into: the five
// paragraphs above, the third of them a real list item, and the trash that runs
// whatever happens next.
//
// The setup writes are direct edits rather than suggestions, and that is the
// point of the level: the document was learned from a create the guard carried,
// so the policy holds it at LevelFull and Docs is asked plainly. A handed-in
// document would be refused here, which is the bar the whole milestone stands
// on.
func blockSubject(t *testing.T, ctx context.Context, s *gapi.Session, folder string) string {
	t.Helper()
	docID, err := createDoc(ctx, s, folder, "gdoc live propose block "+time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("the subject document could not be created in folder %q: %v", folder, err)
	}
	t.Logf("subject document %s", docID)
	t.Cleanup(func() {
		if err := drive.Trash(ctx, s, docID); err != nil {
			t.Errorf("the subject document %q is still in the folder: %v", docID, err)
			return
		}
		t.Logf("subject document %s trashed", docID)
	})

	body := strings.Join(blockSubjectParagraphs, "")
	item := blockSubjectParagraphs[blockSubjectItem]
	start := blockParagraphStart(blockSubjectItem)
	if err := batch(ctx, s, docID,
		req("insertText", map[string]any{
			"location": map[string]any{"index": 1},
			"text":     body,
		}),
		// The bullet covers the item without its paragraph mark, which is the
		// range TestLiveBlockProposalProbe made its own list item with.
		req("createParagraphBullets", map[string]any{
			"range":        rng(start, start+len(item)-1),
			"bulletPreset": "BULLET_DISC_CIRCLE_SQUARE",
		}),
	); err != nil {
		t.Fatalf("the five paragraphs could not be written into %q: %v", docID, err)
	}
	return docID
}

// blockParagraphStart is where paragraph n begins, counted from zero. Docs
// numbers the body from index 1 and every character in the setup is ASCII, so
// the arithmetic is the length of everything in front of it.
func blockParagraphStart(n int) int {
	at := 1
	for _, p := range blockSubjectParagraphs[:n] {
		at += len(p)
	}
	return at
}

// blockBodyText is the document as the text projection prints it, which is the
// one comparison that says a withdrawal really put everything back: it carries
// every character of the body and no index, no revision and no id.
func blockBodyText(t *testing.T, ctx context.Context, s *gapi.Session, docID, when string) string {
	t.Helper()
	d, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		t.Fatalf("the document %q could not be read %s: %v", docID, when, err)
	}
	text, notes := view.Text(d)
	t.Logf("%s: %d characters, %d projection warnings", when, len(text), len(notes))
	return text
}

// proposeBlock runs the production writer for one block and asserts everything
// the milestone promised: one suggestion id, a comment, all three read-backs,
// and verified.
//
// One id is the promise the note rests on. withdraw takes back the id the note
// records, so a block that came back as two suggestions is one gdoc cannot take
// back whole, and VerifyBlock already reports that as unverified. This says the
// measured shape still holds today.
func proposeBlock(t *testing.T, ctx context.Context, s *gapi.Session, docID, name string, p propose.Proposal) propose.Result {
	t.Helper()
	result, err := propose.Apply(ctx, s, docID, p)
	for _, w := range result.Warnings {
		t.Logf("%s warning: %s", name, w)
	}
	if err != nil {
		t.Fatalf("%s could not be written: %v", name, err)
	}
	if len(result.SuggestionIDs) != 1 {
		t.Fatalf("%s came back as %d suggestion ids (%v), and a block is one", name, len(result.SuggestionIDs), result.SuggestionIDs)
	}
	if result.CommentID == "" {
		t.Fatalf("%s landed with no comment id, so the reason is not in the document", name)
	}
	if !result.Checks.SuggestionsInline {
		t.Errorf("%s does not read back as a pending suggestion in the inline view", name)
	}
	if !result.Checks.PreviewWithoutSuggestions {
		t.Errorf("%s is in the preview view, so the write was an edit rather than a suggestion", name)
	}
	if !result.Checks.DocxAnchored {
		t.Errorf("%s has no anchored comment in the docx export", name)
	}
	if !result.Verified {
		t.Errorf("%s is not verified: state %q, checks %+v", name, result.CommentUpdateState, result.Checks)
	}
	t.Logf("%s: suggestion %v, comment %s, state %s", name, result.SuggestionIDs, result.CommentID, result.CommentUpdateState)
	return result
}

// withdrawBlocks takes both proposals back, through the same provenance the
// command uses: each id is recorded into a note's front matter by propose.Record
// and read back by frontmatter.Read, so the run proves the memory as well as the
// write. A note that never named an id is a withdrawal the package refuses, and
// the guard refuses the reject beside it unless the run granted that one id.
func withdrawBlocks(t *testing.T, ctx context.Context, p *guard.Policy, s *gapi.Session, docID string, results []propose.Result) {
	t.Helper()
	note := []byte("---\ngdoc:\n  schema: 1\n  document_id: " + docID + "\n---\n\n# Live block test\n")
	recorded, missed, err := propose.Record(note, docID, results, time.Now().UTC())
	if err != nil {
		t.Fatalf("the proposals could not be recorded in the note: %v", err)
	}
	if len(missed) != 0 {
		t.Fatalf("a live block proposal could not be remembered: %+v", missed)
	}
	block, err := frontmatter.Read(recorded)
	if err != nil {
		t.Fatalf("the note propose wrote could not be read back: %v", err)
	}
	for _, result := range results {
		entry, err := block.Entry(docID)
		if err != nil {
			t.Fatalf("the note propose wrote does not name the document: %v", err)
		}
		id := result.SuggestionIDs[0]
		if !withdraw.Mine(entry, id) {
			t.Fatalf("the note does not record %q, so nothing may be granted", id)
		}
		p.AllowReject(id)
		gone, err := withdraw.Run(ctx, s, docID, id, entry)
		for _, w := range gone.Warnings {
			t.Logf("withdraw warning: %s", w)
		}
		if err != nil {
			t.Fatalf("the block %q (%s) could not be withdrawn: %v", id, result.Quote(), err)
		}
		if !gone.Verified {
			t.Errorf("the withdrawal of %q is not verified: rejected ids %v", id, gone.RejectedSuggestionIDs)
		}
		block = withdraw.Forget(block, docID, id)
		t.Logf("withdrawn: %v", gone.RejectedSuggestionIDs)
	}
}
