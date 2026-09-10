package live

// M7c's acceptance: the house template goes into somebody's document as a
// proposal, and not as text.
//
// That sentence is the whole milestone. Every character of the cover, the three
// front-matter tables and the legend goes out in writeMode SUGGEST, so Nail
// accepts it in the browser the way he accepts any suggestion, or rejects it
// and the document is as it was. Nothing offline can say Google honoured the
// field: CLAUDE.md records the morning it did not, and the guard's refusal is
// about gdoc's own words rather than about what Docs does with them. So the
// claim is measured here, against a real document, and the read-back the
// binary prints is the thing doing the measuring.
//
// Four things about how it runs are the point of it rather than plumbing.
//
// It writes to a copy, never to the document it was named. The copy is made
// through the API with copyComments=true under the per-run AllowCopy grant,
// which is the shape M7b's acceptance already uses, and the original is
// re-read on every path out of this test to say its revision never moved.
//
// The prelude phase runs on a policy that granted nothing at all. That is the
// milestone's shape stated as a live fact: a SUGGEST batchUpdate carrying
// insertText, insertTable and deleteContentRange on a handed-in document is
// what internal/propose has sent every day since M3, so the prelude needed no
// permission this milestone added. The marker is the one exception, and the
// refusal before its grant is asserted rather than assumed: without that line a
// policy that had quietly kept the copy at LevelFull would run the whole
// acceptance through a door this milestone is about keeping shut.
//
// The marker goes out in a batch of its own, ahead of anything else, because a
// prelude nothing records is a prelude the next run proposes a second cover in
// front of.
//
// It leaves the copy behind on purpose. The numbers below say every piece came
// back carrying a suggestion id and the author's text is character for
// character what it was; whether the cover reads right is Nail's, in the
// document, which is where this milestone always said that answer lives.

import (
	"context"
	"os"
	"strings"
	"testing"

	"gdoc/internal/cover"
	"gdoc/internal/docs"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/house"
	"gdoc/internal/prelude"
	"gdoc/internal/restyle"
)

// preludeVar names the document to copy and propose the prelude into. No
// default, for GDOC_LIVE_IDEAL_DOC_ID's reason: this run copies the whole of
// somebody's document, comments included, into gdoc's folder.
const preludeVar = "GDOC_LIVE_PRELUDE_DOC_ID"

func TestLivePreludeIsProposedNotWritten(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 and %s=<document id> to copy a document into the Drive test folder and propose the house prelude into the copy; %s names another folder",
			liveVar, writeVar, preludeVar, folderVar)
	}
	source := strings.TrimSpace(os.Getenv(preludeVar))
	if source == "" {
		t.Fatalf("%s=1 with %s=1 needs %s=<document id>: the copy grant names exactly one source, and there is no default", liveVar, writeVar, preludeVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	ctx := context.Background()

	// The source is handed in to be read and copied, and nothing more. It is at
	// LevelSuggest with no grant of any kind, so nothing in this run could
	// write to it even by mistake.
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

	copyID := copyDocumentNamed(t, ctx, s, source, folder, "gdoc M7c prelude acceptance copy")
	t.Cleanup(func() {
		t.Logf("the copy is left for a person to look at: https://docs.google.com/document/d/%s/edit", copyID)
		t.Logf("accept or reject the proposed prelude, then trash it: File > Move to bin, or leave it in folder %s", folder)
	})

	before, span := proposePrelude(t, ctx, copyID)

	// The read-back, on a session that can read the copy and nothing else. It
	// is the production one, because what the binary prints is what the skill
	// reads out to Nail, and a second reader written for this test would be a
	// second rule that drifts from it.
	rp := guard.NewPolicy()
	rp.AllowFile(copyID, guard.LevelSuggest)
	rs, err := gapi.Open(rp, nil)
	if err != nil {
		t.Fatalf("the read-back session could not be opened: %v", err)
	}
	after, err := docs.Fetch(ctx, rs, copyID)
	if err != nil {
		t.Fatalf("the copy %q could not be read back, so nothing here says whether the prelude is a suggestion or text: %v", copyID, err)
	}
	check, notes := prelude.Verify(before, after, span.Start, span.End)
	for _, n := range notes {
		t.Logf("read-back warning: %s", n)
	}
	t.Logf("proposed: %d paragraphs, %d tables, %d cells, %d runs",
		check.Proposed.Paragraphs, check.Proposed.Tables, check.Proposed.Cells, check.Proposed.Runs)
	t.Logf("written:  %d paragraphs, %d tables, %d cells, %d runs",
		check.Written.Paragraphs, check.Written.Tables, check.Written.Cells, check.Written.Runs)

	// The one fact the milestone is named after. A piece carrying no suggestion
	// id is a character in somebody's document on gdoc's own authority.
	if check.Written.Runs != 0 {
		t.Errorf("%d run(s) of the prelude are text in the document rather than proposed into it: %v. This is the promise M7c makes, and Google honouring writeMode is what keeps it",
			check.Written.Runs, check.Settled)
	}
	if check.Proposed.Runs == 0 {
		t.Errorf("the prelude was proposed into [%d,%d) and the read back found nothing carrying a suggestion id there, so nothing says it is in the document",
			span.Start, span.End)
	}
	// The milestone's own claim about the author's words, character for
	// character. Not a hope: a run that took a paragraph of somebody's prose
	// with it would pass every other check here.
	if !check.BodyUnchanged {
		t.Errorf("the author's own text is not what it was before the prelude was proposed, and this run changed nobody's words")
	}
	if !check.Marked {
		t.Errorf("no named range %q covers [%d,%d), so gdoc has no record of having written the prelude and the next run proposes a second cover in front of it",
			prelude.MarkerName, span.Start, span.End)
	}
	if !check.Verified {
		t.Errorf("the read-back does not verify the prelude: proposed %+v, written %+v, marked %v, body unchanged %v",
			check.Proposed, check.Written, check.Marked, check.BodyUnchanged)
	}
}

// proposePrelude is phase 1 and the marker, on the copy, and it is the shape
// cmd/gdoc's proposeThenStyle sends them in. It hands back the author's own
// text as it stood before a word was proposed, which is the before side of the
// one check this milestone's own claim rests on, and the span the marker
// covers.
//
// The policy holds the copy at the level every handed-in document gets, and
// that is the whole point of it: the copy is at LevelFull on the caller's
// policy, because the guard learned it from the create it carried, and a
// prelude sent there would exercise no bar at all and pass with the milestone
// broken.
func proposePrelude(t *testing.T, ctx context.Context, copyID string) (string, restyle.Span) {
	t.Helper()
	p := guard.NewPolicy()
	p.AllowFile(copyID, guard.LevelSuggest)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the prelude session could not be opened: %v", err)
	}
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the house style could not be loaded: %v", err)
	}
	d, err := docs.Fetch(ctx, s, copyID)
	if err != nil {
		t.Fatalf("the copy %q could not be read before the prelude: %v", copyID, err)
	}
	if d.MultiTab() {
		t.Fatalf("the copy has %d tabs, and a prelude names a range, which means nothing without saying which tab it is in", len(d.Tabs))
	}
	// Taken here because this is the last moment it can be taken: every read
	// after this one carries the prelude.
	before := prelude.AuthorText(d)

	res, err := prelude.Propose(cfg, liveFields(), d)
	if err != nil {
		t.Fatalf("the prelude could not be built for %q: %v", copyID, err)
	}
	span := restyle.Span{Start: res.Start, End: res.End}
	t.Logf("proposing %d requests into [%d,%d): %d paragraphs, %d tables, %d cells",
		len(res.Requests), res.Start, res.End, res.Paragraphs, res.Tables, res.Cells)
	for _, m := range res.Manual {
		t.Logf("manual step: %s (%s)", m.What, m.Where)
	}

	// Phase 1, on a policy that granted nothing. It carries because a SUGGEST
	// batchUpdate on a handed-in document is what internal/propose has sent
	// since M3, and that is the fact this milestone rests on.
	proposed, err := restyle.Suggest(ctx, s, copyID, res.Requests, d.RevisionID)
	for _, w := range proposed.Warnings {
		t.Logf("prelude warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the prelude phase stopped after %d batches: %v", proposed.Batches, err)
	}
	t.Logf("proposed: %d requests in %d batches, revision %s", proposed.Requests, proposed.Batches, proposed.RevisionID)

	// The marker is the one thing written rather than proposed, because Docs
	// answered "createNamedRange: Request does not support application as
	// suggestion" on 2026-09-10. The refusal before its two grants is asserted
	// rather than assumed: it is what says the copy is still a handed-in
	// document at this point and not one the guard learned at LevelFull.
	mark := prelude.MarkerRequest(res.Start, res.End)
	if _, err := restyle.Apply(ctx, s, copyID, []map[string]any{mark}, proposed.RevisionID); err == nil ||
		!strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("a createNamedRange on a handed-in document must be refused until this run grants it, and it was not: %v", err)
	}
	p.GrantInPlace(copyID)
	p.AllowMarker(prelude.MarkerName, res.Start, res.End)

	marked, err := restyle.Apply(ctx, s, copyID, []map[string]any{mark}, proposed.RevisionID)
	for _, w := range marked.Warnings {
		t.Logf("marker warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the prelude was proposed and the marker over [%d,%d) did not land, so gdoc has no record of having written it: %v",
			res.Start, res.End, err)
	}
	t.Logf("marker %q created over [%d,%d), revision %s", prelude.MarkerName, res.Start, res.End, marked.RevisionID)
	return before, span
}

// liveFields is the fields file this run would be given, as the struct it is
// read into. The values are a test document's, and they are written out here
// rather than read from a file so the run needs nothing on disk beside the
// binary.
func liveFields() cover.Fields {
	return cover.Fields{
		Title:             "M7c Prelude Acceptance",
		DocType:           "Policy",
		Version:           "1.0",
		Date:              "September 2026",
		Owner:             "Nail Khusnullin",
		LastApproval:      "September 2026",
		ReviewFrequency:   "Annually",
		BoardRatification: "Pending",
		Distribution:      "Internal",
		Classification:    "Internal",
		HeadingNumbering:  true,
		Revisions: []cover.Revision{
			{
				Version:      "1.0",
				Date:         "September 2026",
				Author:       "gdoc",
				ApprovedBy:   "Nail Khusnullin",
				ApprovalDate: "September 2026",
				Section:      "All",
				Change:       "The house prelude, proposed rather than written.",
			},
		},
	}
}
