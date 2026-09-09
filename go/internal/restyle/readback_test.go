package restyle

import (
	"strings"
	"testing"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/docx"
)

// before is a survey of a document with two threads, one anchored and one
// detached, two pending suggestions and one person chip. It is written out
// rather than surveyed, because what a test of the preservation half states is
// the before, and taking it from a fixture would hide it.
func before() Report {
	return Report{
		DocumentID: "DOC1",
		RevisionID: "ALm37BXrev1",
		Threads: ThreadCounts{
			Open:    2,
			Witness: WitnessCounts{Anchored: 1, Detached: 1},
			Witnessed: []ThreadWitness{
				{ID: "c1", Witness: docx.WitnessAnchored},
				{ID: "c2", Witness: docx.WitnessDetached},
			},
		},
		Suggestions: SuggestionCounts{Pending: 2, IDs: []string{"suggest.aaa", "suggest.bbb"}},
		Chips:       ChipCounts{Person: 1},
	}
}

// after is the same survey with the changes a test asks for applied to it.
func after(change func(*Report)) Report {
	r := before()
	r.RevisionID = "ALm37BXrev2"
	change(&r)
	return r
}

func TestNothingMovedIsReportedAsNothingMoved(t *testing.T) {
	// Act
	got := Preserve(before(), after(func(*Report) {}))

	// Assert
	if !got.Intact {
		t.Errorf("intact = false on a document that did not change: %+v", got)
	}
	if len(got.Threads.Gone) != 0 || len(got.Threads.WitnessChanged) != 0 {
		t.Errorf("threads = %+v, want nothing gone and no witness moved", got.Threads)
	}
	if got.Threads.Before != 2 || got.Threads.After != 2 {
		t.Errorf("threads before and after = %d and %d, want 2 and 2", got.Threads.Before, got.Threads.After)
	}
	if len(got.Suggestions.Gone) != 0 || got.Chips.Fewer {
		t.Errorf("suggestions %+v and chips %+v, want neither moved", got.Suggestions, got.Chips)
	}
}

// The measurement this whole milestone rests on: replacing a document's body
// destroyed every comment anchor. A thread that was anchored and now reads
// detached is that failure, and it is named rather than counted.
func TestAThreadThatLostItsAnchorIsNamed(t *testing.T) {
	got := Preserve(before(), after(func(r *Report) {
		r.Threads.Witnessed[0].Witness = docx.WitnessDetached
		r.Threads.Witness = WitnessCounts{Detached: 2}
	}))

	if got.Intact {
		t.Error("intact = true on a run that detached a comment")
	}
	if len(got.Threads.LostAnchor) != 1 || got.Threads.LostAnchor[0] != "c1" {
		t.Errorf("lost_anchor = %v, want c1", got.Threads.LostAnchor)
	}
	if len(got.Threads.WitnessChanged) != 1 || got.Threads.WitnessChanged[0].Before != docx.WitnessAnchored {
		t.Errorf("witness_changed = %+v, want c1 anchored before and detached now", got.Threads.WitnessChanged)
	}
}

// A thread that was already detached before the run is not damage the run did.
// That is why M7's survey carries a witness per thread rather than totals.
func TestAThreadDetachedBeforeTheRunIsNotDamage(t *testing.T) {
	got := Preserve(before(), after(func(*Report) {}))

	if len(got.Threads.LostAnchor) != 0 {
		t.Errorf("lost_anchor = %v, want nothing: c2 was detached before the run too", got.Threads.LostAnchor)
	}
}

// Unmatched is the export giving no answer, not a destroyed anchor. Calling
// absence of evidence damage is the cry-wolf warning this tool avoids
// everywhere else, so it is reported as its own fact.
func TestAWitnessThatStoppedAnsweringIsNotAnchorDamage(t *testing.T) {
	got := Preserve(before(), after(func(r *Report) {
		r.Threads.Witnessed[0].Witness = docx.WitnessUnmatched
	}))

	if len(got.Threads.LostAnchor) != 0 {
		t.Errorf("lost_anchor = %v, want nothing: unmatched is no answer, not a lost anchor", got.Threads.LostAnchor)
	}
	if len(got.Threads.Unwitnessed) != 1 || got.Threads.Unwitnessed[0] != "c1" {
		t.Errorf("unwitnessed = %v, want c1", got.Threads.Unwitnessed)
	}
	if !got.Intact {
		t.Error("intact = false: a witness with no answer is not a loss, and verified reads it separately")
	}
}

func TestAThreadThatIsGoneIsNamedAndOneThatArrivedIsToo(t *testing.T) {
	got := Preserve(before(), after(func(r *Report) {
		r.Threads.Witnessed = []ThreadWitness{
			{ID: "c1", Witness: docx.WitnessAnchored},
			{ID: "c3", Witness: docx.WitnessAnchored},
		}
	}))

	if len(got.Threads.Gone) != 1 || got.Threads.Gone[0] != "c2" {
		t.Errorf("gone = %v, want c2", got.Threads.Gone)
	}
	if len(got.Threads.Arrived) != 1 || got.Threads.Arrived[0] != "c3" {
		t.Errorf("arrived = %v, want c3", got.Threads.Arrived)
	}
	if got.Intact {
		t.Error("intact = true on a run a thread went missing across")
	}
}

// Ids rather than counts, because one suggestion destroyed and another created
// is two counts that do not move.
func TestAPendingSuggestionIsComparedByID(t *testing.T) {
	got := Preserve(before(), after(func(r *Report) {
		r.Suggestions.IDs = []string{"suggest.aaa", "suggest.ccc"}
	}))

	if len(got.Suggestions.Gone) != 1 || got.Suggestions.Gone[0] != "suggest.bbb" {
		t.Errorf("gone = %v, want suggest.bbb", got.Suggestions.Gone)
	}
	if len(got.Suggestions.Arrived) != 1 || got.Suggestions.Arrived[0] != "suggest.ccc" {
		t.Errorf("arrived = %v, want suggest.ccc", got.Suggestions.Arrived)
	}
	if got.Intact {
		t.Error("intact = true on a run a pending suggestion went missing across")
	}
}

func TestAChipThatIsGoneIsReported(t *testing.T) {
	got := Preserve(before(), after(func(r *Report) { r.Chips = ChipCounts{} }))

	if !got.Chips.Fewer || got.Intact {
		t.Errorf("chips = %+v, intact = %v, want the lost chip reported", got.Chips, got.Intact)
	}
}

// The three the Docs API cannot create at all, each with the menu path a person
// follows. SPEC has gdoc write this list into the document; Nail's decision of
// 2026-09-09 is that it reports them and the skill reads them out.
func TestTheManualStepsNameTheMenuPathForEach(t *testing.T) {
	got := ManualSteps(Plan{})

	if len(got) != 3 {
		t.Fatalf("steps = %+v, want the three the API cannot create", got)
	}
	for _, want := range []string{"header", "contents", "page numbers"} {
		if !anyStep(got, want) {
			t.Errorf("no step names %q: %+v", want, got)
		}
	}
	for _, s := range got {
		if !strings.Contains(s.Where, ">") {
			t.Errorf("step %q has no menu path: %q", s.What, s.Where)
		}
	}
}

func TestTheManualStepsAlsoNameWhatTheRunLeftAlone(t *testing.T) {
	got := ManualSteps(Plan{Bulleted: 2, Tables: 1, Unstyled: []string{"HEADING_7"}})

	if len(got) != 6 {
		t.Fatalf("steps = %+v, want the three fixed ones and the three this run left alone", got)
	}
	for _, want := range []string{"bulleted", "column widths", "HEADING_7"} {
		if !anyStep(got, want) {
			t.Errorf("no step names %q: %+v", want, got)
		}
	}
}

func anyStep(steps []ManualStep, want string) bool {
	for _, s := range steps {
		if strings.Contains(s.What, want) || strings.Contains(s.Where, want) {
			return true
		}
	}
	return false
}

// verified is the two halves together. A run that preserved everything and
// landed nothing did no work, and a run that landed everything and lost the
// anchors is the failure this milestone exists to avoid.
func TestVerifiedIsBothHalvesTogether(t *testing.T) {
	in := Input{
		Document: doc(para(1, text(1, "The supplier\n"))),
		Comments: []comments.RawComment{{ID: "c1", Content: "ai? this",
			Author:            comments.Author{DisplayName: "Nail"},
			QuotedFileContent: &comments.Quoted{Value: "The supplier"}}},
		Export: &docx.File{Comments: []docx.Comment{{Author: "Nail", Text: "ai? this", Anchored: true}}},
	}
	in.Document.CommentRanges = map[string]docs.Range{"c1": {Tab: "t.0", Start: 1, End: 5}}
	was := Report{
		DocumentID: "DOC1",
		Threads: ThreadCounts{Open: 1, Witnessed: []ThreadWitness{
			{ID: "c1", Witness: docx.WitnessAnchored}}},
		Suggestions: SuggestionCounts{IDs: []string{}},
	}
	sent := []map[string]any{
		{"updateDocumentStyle": map[string]any{
			"documentStyle": map[string]any{"marginTop": dim(62.35)}, "fields": "marginTop"}},
	}
	read := `{"tabs": [{"documentTab": {"documentStyle": {"marginTop": {"magnitude": 62.35, "unit": "PT"}},
	           "body": {"content": []}}}]}`

	// Act
	got, warnings := Verify(was, in, []byte(read), Sent{Confirmed: sent}, Plan{})

	// Assert
	if !got.Verified {
		t.Errorf("verified = false: preservation %+v, landing %+v, warnings %v",
			got.Preservation, got.Landing, warnings)
	}
	if got.After.DocumentID != "DOC1" {
		t.Errorf("after = %+v, want the survey taken again", got.After)
	}
	if len(got.Manual) != 3 {
		t.Errorf("manual = %+v, want the three the API cannot create", got.Manual)
	}
}

func TestVerifiedIsFalseWhenTheStyleIsNotThere(t *testing.T) {
	in := Input{Document: doc(para(1, text(1, "The supplier\n")))}
	sent := []map[string]any{
		{"updateDocumentStyle": map[string]any{
			"documentStyle": map[string]any{"marginTop": dim(62.35)}, "fields": "marginTop"}},
	}
	read := `{"tabs": [{"documentTab": {"documentStyle": {"marginTop": {"magnitude": 72, "unit": "PT"}},
	           "body": {"content": []}}}]}`

	got, warnings := Verify(Report{Suggestions: SuggestionCounts{IDs: []string{}}}, in, []byte(read), Sent{Confirmed: sent}, Plan{})

	if got.Verified {
		t.Error("verified = true over a margin the document does not carry")
	}
	if !anyWarning(warnings, "updateDocumentStyle") {
		t.Errorf("warnings = %v, want the route that did not hold named", warnings)
	}
}

func TestVerifiedIsFalseWhenAThreadLostItsAnchor(t *testing.T) {
	in := Input{
		Document: doc(para(1, text(1, "The supplier\n"))),
		Comments: []comments.RawComment{{ID: "c1", Content: "ai? this",
			Author:            comments.Author{DisplayName: "Nail"},
			QuotedFileContent: &comments.Quoted{Value: "The supplier"}}},
		Export: &docx.File{Comments: []docx.Comment{{Author: "Nail", Text: "ai? this", Anchored: false}}},
	}
	in.Document.CommentRanges = map[string]docs.Range{"c1": {Tab: "t.0", Start: 1, End: 5}}
	was := Report{Threads: ThreadCounts{Open: 1, Witnessed: []ThreadWitness{
		{ID: "c1", Witness: docx.WitnessAnchored}}}, Suggestions: SuggestionCounts{IDs: []string{}}}
	sent := []map[string]any{
		{"updateDocumentStyle": map[string]any{
			"documentStyle": map[string]any{"marginTop": dim(62.35)}, "fields": "marginTop"}},
	}
	read := `{"tabs": [{"documentTab": {"documentStyle": {"marginTop": {"magnitude": 62.35, "unit": "PT"}},
	           "body": {"content": []}}}]}`

	got, warnings := Verify(was, in, []byte(read), Sent{Confirmed: sent}, Plan{})

	if got.Verified {
		t.Error("verified = true on a run that detached a comment")
	}
	if !anyWarning(warnings, "detached") {
		t.Errorf("warnings = %v, want the detached thread named", warnings)
	}
}

// A batch Docs accepted whose answer could not be read is read back and never
// verified. The landing half reads the first request of each kind, so a batch of
// hundreds can hold one request that landed and the rest that did not, and the
// run has no answer for the ones it never read.
func TestAnUnconfirmedBatchIsCheckedAndNeverVerified(t *testing.T) {
	// Arrange: everything the run sent is in the document, and nothing was
	// preserved wrongly. The only thing wrong is that Docs never said so.
	in := Input{Document: doc(para(1, text(1, "The supplier\n")))}
	sent := []map[string]any{
		{"updateDocumentStyle": map[string]any{
			"documentStyle": map[string]any{"marginTop": dim(62.35)}, "fields": "marginTop"}},
	}
	read := `{"tabs": [{"documentTab": {"documentStyle": {"marginTop": {"magnitude": 62.35, "unit": "PT"}},
	           "body": {"content": []}}}]}`
	was := Report{Suggestions: SuggestionCounts{IDs: []string{}}}

	// Act
	got, _ := Verify(was, in, []byte(read), Sent{Unconfirmed: sent}, Plan{})

	// Assert
	if len(got.Landing.Checks) == 0 {
		t.Error("no check over a batch Docs accepted: reading it back is the only way to know whether it landed")
	}
	if got.Verified {
		t.Error("verified = true over a batch whose answer could not be read")
	}
	// And the same requests confirmed do verify, so the flag is what decides
	// and not the checks.
	if again, _ := Verify(was, in, []byte(read), Sent{Confirmed: sent}, Plan{}); !again.Verified {
		t.Errorf("verified = false on the confirmed half of the same run: landing %+v", again.Landing)
	}
}

// A run that sent nothing verifies nothing. Reporting verified over no checks
// at all would be the same false fact as reporting it over a check that could
// not be made.
func TestVerifiedIsFalseWhenNothingWasChecked(t *testing.T) {
	in := Input{Document: doc(para(1, text(1, "The supplier\n")))}

	got, _ := Verify(Report{Suggestions: SuggestionCounts{IDs: []string{}}}, in, []byte(`{}`), Sent{}, Plan{})

	if got.Verified {
		t.Error("verified = true on a run with no check in it")
	}
}

func anyWarning(warnings []string, want string) bool {
	for _, w := range warnings {
		if strings.Contains(w, want) {
			return true
		}
	}
	return false
}
