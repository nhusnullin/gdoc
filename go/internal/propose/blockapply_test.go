package propose

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// testBlock is the everyday block the apply tests send: the four paragraphs of
// testContent, after the anchor block-body.json holds.
var testBlock = Proposal{
	Kind:    KindBlock,
	After:   "reviewed annually",
	Content: testContent,
	Why:     testWhy,
}

// blockApplyScript is Google for a whole block run: the read the placement is
// computed from, the answer to the batch, and the three read-backs behind it.
//
// The inline read is a queue because ApplyBlock reads that URL twice, once to
// place the block and once to see what the write did. One fixture for both
// would verify the document as it stood before the write.
func blockApplyScript(t *testing.T, before, after, preview, batch string, export []byte) *fakeSession {
	t.Helper()
	return &fakeSession{
		inline:  [][]byte{fixture(t, before), fixture(t, after)},
		preview: fixture(t, preview),
		batch:   fixture(t, batch),
		export:  export,
		failAt:  map[int]error{},
	}
}

// posts is every batchUpdate the fake was asked to make.
func posts(f *fakeSession) []call {
	var out []call
	for _, c := range f.calls {
		if c.method == "POST" {
			out = append(out, c)
		}
	}
	return out
}

func TestApplyBlockReadsPlacesWritesOnceAndVerifies(t *testing.T) {
	f := blockApplyScript(t, "block-body.json", "block-after-inline.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	res, err := ApplyBlock(context.Background(), f, testDocID, testBlock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Verified {
		t.Errorf("Verified = false, checks = %+v, warnings = %v", res.Checks, res.Warnings)
	}
	if res.Checks != (Checks{SuggestionsInline: true, PreviewWithoutSuggestions: true, DocxAnchored: true}) {
		t.Errorf("checks = %+v, want all three", res.Checks)
	}
	if res.CommentID != "AAAC" || res.CommentUpdateState != "ALL_SAVED" {
		t.Errorf("comment = %q, state = %q", res.CommentID, res.CommentUpdateState)
	}
	if len(res.SuggestionIDs) != 1 || res.SuggestionIDs[0] != "suggest.block" {
		t.Errorf("suggestion ids = %v, want the one id the block folded into", res.SuggestionIDs)
	}
	if f.calls[0].method != "GET" {
		t.Errorf("the first call is %s %s, and it should be the read the placement is computed from", f.calls[0].method, f.calls[0].url)
	}
	if len(posts(f)) != 1 {
		t.Errorf("POSTs = %d, want exactly one batchUpdate", len(posts(f)))
	}
}

// TestApplyBlockSendsTheBatchTheBuilderBuilt is the one thing this room adds
// over the builder's own golden tests: the batch that goes out is built from
// the read that just came back, at the placement computed from it.
func TestApplyBlockSendsTheBatchTheBuilderBuilt(t *testing.T) {
	f := blockApplyScript(t, "block-body.json", "block-after-inline.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	if _, err := ApplyBlock(context.Background(), f, testDocID, testBlock); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sent := posts(f)
	if len(sent) != 1 {
		t.Fatalf("POSTs = %d, want one", len(sent))
	}
	place, content := afterPlace(t)
	want := BlockBatch(place, content, testBlock.Why, testBlock.Assignee)
	if !bytes.Equal(bytes.TrimSpace(sent[0].body), bytes.TrimSpace(want)) {
		t.Errorf("the batch sent is not the one BlockBatch builds.\n--- sent ---\n%s\n--- built ---\n%s", sent[0].body, want)
	}
	if sent[0].url != BatchURL(testDocID) {
		t.Errorf("the batch went to %q", sent[0].url)
	}
}

// TestApplyBlockCarriesThePlacementItWasAskedFor is what the envelope reports.
// A block names its place with words, so the result says which words, and the
// words kind's two fields stay empty on it.
func TestApplyBlockCarriesThePlacementItWasAskedFor(t *testing.T) {
	f := blockApplyScript(t, "block-body.json", "block-after-inline.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	res, err := ApplyBlock(context.Background(), f, testDocID, testBlock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.After != testBlock.After {
		t.Errorf("after = %q, want %q", res.After, testBlock.After)
	}
	if res.Quoted != "" || res.Replacement != "" {
		t.Errorf("result = %+v, and the words kind's fields are empty on a block", res)
	}
	if res.ReplaceFrom != "" || res.ReplaceTo != "" {
		t.Errorf("result = %+v, and this block named no replace", res)
	}
}

func TestApplyBlockCarriesBothQuotesOfAReplace(t *testing.T) {
	f := blockApplyScript(t, "block-body.json", "block-replace-inline.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))
	p := Proposal{Kind: KindBlock, ReplaceFrom: "reviewed annually", ReplaceTo: "risk matrix",
		Content: testContent, Why: testWhy}

	res, err := ApplyBlock(context.Background(), f, testDocID, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Verified {
		t.Errorf("Verified = false, checks = %+v, warnings = %v", res.Checks, res.Warnings)
	}
	if res.ReplaceFrom != p.ReplaceFrom || res.ReplaceTo != p.ReplaceTo {
		t.Errorf("result = %+v, want both quotes of the replace", res)
	}
	if res.After != "" {
		t.Errorf("after = %q on a replace", res.After)
	}
}

// TestApplyBlockRefusesContentItCannotReadBeforeAnyRequest is the shape rule
// ahead of the wire. Content outside the subset is refused by name, and the
// document is never even read for it.
func TestApplyBlockRefusesContentItCannotReadBeforeAnyRequest(t *testing.T) {
	f := blockApplyScript(t, "block-body.json", "block-after-inline.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))
	p := testBlock
	p.Content = "| a | b |\n| - | - |\n| c | d |\n"

	_, err := ApplyBlock(context.Background(), f, testDocID, p)
	if err == nil {
		t.Fatal("a block carrying a table was written")
	}
	if !strings.Contains(err.Error(), "table") {
		t.Errorf("error = %q, and it should name the table", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("the content was read after %d requests, and it needs none at all", len(f.calls))
	}
}

// TestApplyBlockRefusesAPlacementItCannotMakeBeforeAnyWrite is the other half:
// the placement needs the document, so the read happens, and nothing is written
// when the answer is a refusal.
func TestApplyBlockRefusesAPlacementItCannotMakeBeforeAnyWrite(t *testing.T) {
	f := blockApplyScript(t, "block-body.json", "block-after-inline.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))
	p := testBlock
	p.After = "reviewed monthly"

	_, err := ApplyBlock(context.Background(), f, testDocID, p)
	if err == nil {
		t.Fatal("a block whose anchor is not in the document was written")
	}
	if len(posts(f)) != 0 {
		t.Errorf("a batch reached the wire: %v", f.calls)
	}
}

func TestApplyBlockRefusesAMultiTabDocumentBeforeAnyWrite(t *testing.T) {
	f := blockApplyScript(t, "two-tabs.json", "block-after-inline.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	_, err := ApplyBlock(context.Background(), f, testDocID, testBlock)
	if err == nil {
		t.Fatal("a two-tab document was written to")
	}
	if !strings.Contains(err.Error(), "2 tabs") {
		t.Errorf("error = %q, and it should name the tab count", err)
	}
	if len(posts(f)) != 0 {
		t.Errorf("a batch reached the wire: %v", f.calls)
	}
}

// TestApplyBlockReportsALostAnswerRatherThanRaising is the rule everything
// after the post follows. The server took the batch, so the block is in the
// document, and a run told it failed is a run that proposes the same block
// again.
func TestApplyBlockReportsALostAnswerRatherThanRaising(t *testing.T) {
	f := blockApplyScript(t, "block-body.json", "block-after-inline.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))
	f.failAt[1] = acceptedError{errors.New("the answer could not be read: unexpected EOF")}

	res, err := ApplyBlock(context.Background(), f, testDocID, testBlock)
	if err != nil {
		t.Fatalf("a batch the server accepted was reported as never sent: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on a write whose answer was never read")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "accepted by Docs") {
		t.Errorf("warnings = %v, and one should say the write was accepted", res.Warnings)
	}
	if !res.Checks.SuggestionsInline {
		t.Errorf("checks = %+v, and the read-backs still ran", res.Checks)
	}
}

// TestApplyBlockReportsMoreThanOneSuggestionAsUnverified is the decision that
// one id is what a verified block means: withdraw takes back the id the note
// records, so a block that came back as two is one gdoc cannot take back whole.
func TestApplyBlockReportsMoreThanOneSuggestionAsUnverified(t *testing.T) {
	f := blockApplyScript(t, "block-body.json", "block-after-inline-two-ids.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	res, err := ApplyBlock(context.Background(), f, testDocID, testBlock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Verified {
		t.Error("Verified = true on a block that came back as two suggestions")
	}
	if len(res.SuggestionIDs) != 2 {
		t.Errorf("suggestion ids = %v, want both of them", res.SuggestionIDs)
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "withdraw") {
		t.Errorf("warnings = %v, and one should say withdraw takes back the first id only", res.Warnings)
	}
}

// TestCheckAcceptsABlockInEitherPlacementForm is the shape rule saying yes.
// Both forms are whole, and the words kind's own fields are absent from both.
func TestCheckAcceptsABlockInEitherPlacementForm(t *testing.T) {
	blocks := []Proposal{
		testBlock,
		{Kind: KindBlock, ReplaceFrom: "a", ReplaceTo: "b", Content: testContent, Why: testWhy},
	}
	for _, p := range blocks {
		if err := p.Check(); err != nil {
			t.Errorf("Check(%+v) refused it: %v", p, err)
		}
	}
}

// TestCheckRefusesEachBadBlockByName is the block's own shape rule, asked of
// the proposal alone. It runs before the probe and before the first write, so a
// bad third entry stops a run that has changed nothing.
//
// The placement form is checked here and nowhere else: PlaceAfter and
// PlaceReplace each take the quotes they need, so a block naming neither form
// would reach one of them as an empty quote and be refused in words about a
// quote that is not in the document.
func TestCheckRefusesEachBadBlockByName(t *testing.T) {
	cases := []struct {
		name string
		p    Proposal
		says string
	}{
		{"no placement", Proposal{Kind: KindBlock, Content: testContent, Why: testWhy}, "after"},
		{"both placements", Proposal{Kind: KindBlock, After: "a", ReplaceFrom: "b", ReplaceTo: "c",
			Content: testContent, Why: testWhy}, "one of the two"},
		{"after with a half replace", Proposal{Kind: KindBlock, After: "a", ReplaceTo: "c",
			Content: testContent, Why: testWhy}, "one of the two"},
		{"replace_from alone", Proposal{Kind: KindBlock, ReplaceFrom: "b", Content: testContent, Why: testWhy}, "replace_to"},
		{"replace_to alone", Proposal{Kind: KindBlock, ReplaceTo: "c", Content: testContent, Why: testWhy}, "replace_from"},
		// The words kind's fields do nothing in a block, and a block that
		// carried them would have its quoted silently ignored while the
		// placement it really used came from somewhere else.
		{"carrying quoted", Proposal{Kind: KindBlock, After: "a", Quoted: "x", Content: testContent, Why: testWhy}, "words kind"},
		{"carrying replacement", Proposal{Kind: KindBlock, After: "a", Replacement: "x", Content: testContent, Why: testWhy}, "words kind"},
		{"no content", Proposal{Kind: KindBlock, After: "a", Why: testWhy}, "no content"},
		{"whitespace content", Proposal{Kind: KindBlock, After: "a", Content: "  \n\t\n", Why: testWhy}, "no content"},
		{"content over the ceiling", Proposal{Kind: KindBlock, After: "a", Why: testWhy,
			Content: strings.Repeat("a paragraph of words.\n\n", MaxContent/20)}, "bytes"},
		{"no reason", Proposal{Kind: KindBlock, After: "a", Content: testContent}, "no reason"},
		{"markdown reason", Proposal{Kind: KindBlock, After: "a", Content: testContent,
			Why: "use **six months**"}, "markdown"},
		{"reason already signed", Proposal{Kind: KindBlock, After: "a", Content: testContent,
			Why: Prefix + "the section was missing"}, "already opens with"},
		// An unknown kind is refused rather than read as the words kind. The
		// dispatch would otherwise send a misspelled "blcok" down the words
		// path, where it is refused for quoting no text, which names nothing
		// the author did wrong.
		{"unknown kind", Proposal{Kind: "paragraphs", After: "a", Content: testContent, Why: testWhy}, "paragraphs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.p.Check()
			if err == nil {
				t.Fatalf("Check(%+v) accepted it", c.p)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("error = %q, and it should say %q", err, c.says)
			}
		})
	}
}

// TestApplyDispatchesOnTheKind is why the command has one call. A proposals
// file holds both kinds in one list, and the loop that walks it does not branch.
func TestApplyDispatchesOnTheKind(t *testing.T) {
	f := blockApplyScript(t, "block-body.json", "block-after-inline.json", "block-body.json",
		"batch-saved.json", blockExport(t, Prefix+testWhy, "3.6 Limits"))

	res, err := Apply(context.Background(), f, testDocID, testBlock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.After != testBlock.After || !res.Verified {
		t.Errorf("result = %+v, want the block ApplyBlock would have returned", res)
	}
	sent := posts(f)
	if len(sent) != 1 {
		t.Fatalf("POSTs = %d, want one", len(sent))
	}
	var body map[string]any
	if err := json.Unmarshal(sent[0].body, &body); err != nil {
		t.Fatalf("the batch body is not JSON: %v", err)
	}
	if reqs, ok := body["requests"].([]any); !ok || len(reqs) <= 3 {
		t.Errorf("requests = %v, and the block's batch is longer than the words kind's three", body["requests"])
	}
}

// TestAnEmptyKindIsStillTheWordsKind is what keeps every proposals file written
// before this milestone readable: a words proposal names no kind, and gains no
// field it has to set.
func TestAnEmptyKindIsStillTheWordsKind(t *testing.T) {
	if testProposal.Kind != "" {
		t.Fatalf("the words proposal names the kind %q", testProposal.Kind)
	}
	f := script(t, "before.json", "after.json", "preview.json", "batch-saved.json", withComment)

	res, err := Apply(context.Background(), f, testDocID, testProposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Verified || res.Quoted != testProposal.Quoted {
		t.Errorf("result = %+v, want the words kind's own", res)
	}
	if res.After != "" || res.ReplaceFrom != "" || res.ReplaceTo != "" {
		t.Errorf("result = %+v, and a words proposal names no placement of its own", res)
	}
}
