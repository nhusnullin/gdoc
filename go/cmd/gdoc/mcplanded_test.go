package main

import (
	"testing"

	"gdoc/internal/chat"
	"gdoc/internal/emit"
	"gdoc/internal/propose"
)

// A write that reached the document is kept, though the call it was part of
// answered not ok.
//
// Three writers answer `ok: false` over a change that is already in somebody's
// document: a proposal whose read-back could not confirm it, a proposal whose
// answer was lost, and an item that landed before a later one stopped the run.
// Nothing trusts a success, so the envelope says what landed in `sent` and in
// `outcome`, and that is what the ledger must read. Reading `ok` instead would
// under-count the burst, leave the Focus rule's reset point behind, and report a
// comment gdoc itself wrote as `robot_not_ours`.
func TestAWriteThatLandedIsRecordedThoughTheCallFailed(t *testing.T) {
	confirmed := propose.Checks{}
	cases := []struct {
		name string
		tool string
		data any
		own  string
	}{
		{
			name: "a proposal gdoc could not confirm",
			tool: "propose",
			data: proposeData{DocumentID: fixtureDocID, Proposals: []proposalReport{
				{Quoted: "one", Sent: true, CommentID: "C9", Checks: confirmed},
			}},
			own: "C9",
		},
		{
			name: "a proposal whose answer was lost",
			tool: "propose",
			data: proposeData{DocumentID: fixtureDocID, Proposals: []proposalReport{
				{Quoted: "one", Outcome: propose.OutcomeUnknown},
			}},
		},
		{
			name: "an annotation that landed before the run stopped",
			tool: "annotate",
			data: annotateData{DocumentID: fixtureDocID, Annotations: []annotationReport{
				{Quoted: "one", Sent: true, CommentID: "C8"},
				{Quoted: "two"},
			}},
			own: "C8",
		},
		{
			name: "a reply Drive named and gdoc could not verify",
			tool: "reply",
			data: replyData{DocumentID: fixtureDocID, CommentID: "AAAA1111", ReplyID: "R9"},
			own:  "R9",
		},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			led := chat.NewLedger()
			mcpRecordWrite(led, one.tool, emit.Result{OK: false, Data: one.data, Error: "something"}, ledgerClock)

			writes := led.Writes()
			if len(writes) != 1 {
				t.Fatalf("the ledger kept %d writes, want one", len(writes))
			}
			if writes[0].Tool != one.tool || writes[0].DocID != fixtureDocID {
				t.Errorf("the write is kept as %+v", writes[0])
			}
			if one.own != "" && !led.Wrote(one.own) {
				t.Errorf("the ledger does not know gdoc wrote %s", one.own)
			}
		})
	}
}

// And a call that wrote nothing is still not a write. Every entry says `sent:
// false` with no outcome beside it, which is what a refusal before the wire
// leaves, and counting it would hold the next call for a burst that never
// reached a document.
func TestACallThatWroteNothingIsNotRecorded(t *testing.T) {
	cases := []struct {
		name string
		tool string
		data any
	}{
		{"propose", "propose", proposeData{DocumentID: fixtureDocID, Proposals: []proposalReport{
			{Quoted: "one"}, {Quoted: "two"},
		}}},
		{"annotate", "annotate", annotateData{DocumentID: fixtureDocID, Annotations: []annotationReport{
			{Quoted: "one"},
		}}},
		{"reply", "reply", replyData{DocumentID: fixtureDocID, CommentID: "AAAA1111"}},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			led := chat.NewLedger()
			mcpRecordWrite(led, one.tool, emit.Result{OK: false, Data: one.data, Error: "refused"}, ledgerClock)
			if writes := led.Writes(); len(writes) != 0 {
				t.Errorf("the ledger kept %d writes for a call that wrote nothing: %+v", len(writes), writes)
			}
		})
	}
}
