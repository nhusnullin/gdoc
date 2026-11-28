// Edit mode: `propose --edit`, which writes words directly and paints them
// yellow, and `seen`, which takes the yellow off. Nail's decision of
// 2026-11-28, DECISIONS.md. internal/edit holds the rules of the write, and
// internal/guard the two grammars the level carries.

package main

import (
	"context"
	"fmt"

	"gdoc/internal/docs"
	"gdoc/internal/edit"
	"gdoc/internal/emit"
	"gdoc/internal/guard"
	"gdoc/internal/propose"
	"gdoc/internal/publish"
)

// editFlag is the one word that turns a propose run into direct edits. It is
// on the command line or it is nowhere: no setting, no note and no proposals
// file turns it on, and `gdoc mcp` never offers it.
// TestEditModeIsOnlyTheFlagOnTheLine and TestEditAndSeenAreNeverOfferedInChat.
const editFlag = "--edit"

// editReport is one entry of an edit run, sent or not.
type editReport struct {
	Quoted      string       `json:"quoted"`
	Replacement string       `json:"replacement"`
	Sent        bool         `json:"sent"`
	Verified    bool         `json:"verified"`
	Checks      *edit.Checks `json:"checks,omitempty"`
	Outcome     string       `json:"outcome,omitempty"`
}

type editData struct {
	DocumentID string       `json:"document_id"`
	Mode       string       `json:"mode"`
	Colour     string       `json:"colour"`
	Tabs       int          `json:"tabs"`
	MultiTab   bool         `json:"multi_tab"`
	Edits      []editReport `json:"edits"`
}

// openEdit is the one place a document is raised to the edit level: the id
// from the command line, handed in at the suggest level and then granted.
// cmdProposeEdit and cmdSeen are its two callers, and
// TestGrantEditHasTwoCallers in go/boundary holds the count.
func openEdit(docID string) (*reach, *guard.Policy, error) {
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)
	p.GrantEdit(docID)
	s, err := openSession(p)
	if err != nil {
		return nil, nil, err
	}
	return &reach{id: docID, session: s}, p, nil
}

// spansTo hands a read's spans to the guard, which is the only thing that
// lets the next batch name them.
func spansTo(p *guard.Policy, kind guard.EditKind) edit.Grant {
	return func(revision string, spans []edit.Span) {
		out := make([]guard.Span, 0, len(spans))
		for _, s := range spans {
			out = append(out, guard.Span{Start: s.Start, End: s.End})
		}
		p.AllowEditSpans(kind, revision, out)
	}
}

// cmdProposeEdit is `propose --edit`: every entry of the file written directly,
// one at a time, each after its own fresh read.
//
// Every entry is checked before the first one leaves the machine, so a block in
// the file refuses the run rather than stopping it half way. The run stops at
// the first entry that cannot be written and at the first the read-back cannot
// confirm, as a propose run does.
func cmdProposeEdit(ctx context.Context, a *args, docID string, proposals []propose.Proposal) emit.Result {
	if a.has("--md") {
		return emit.Result{OK: false, Error: "--edit takes no --md: a direct edit is in the document, and there is no suggestion to record or to take back later"}
	}
	for i, p := range proposals {
		if err := edit.Check(p, a.set.Mark); err != nil {
			return emit.Result{OK: false, Error: fmt.Sprintf("entry %d: %s. Nothing was sent", i+1, err)}
		}
	}
	r, policy, err := openEdit(docID)
	if err != nil {
		return failed(err)
	}
	d, err := docs.Fetch(ctx, r.session, r.id)
	if err != nil {
		return withStop(emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}, err)
	}
	data := editData{DocumentID: r.id, Mode: "edit", Colour: edit.Yellow, Tabs: len(d.Tabs), MultiTab: d.MultiTab()}
	for _, p := range proposals {
		data.Edits = append(data.Edits, editReport{Quoted: p.Quoted, Replacement: p.Replacement})
	}

	var warns []string
	for i, one := range proposals {
		if ctx.Err() != nil {
			return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...),
				Error: fmt.Sprintf("the time for this call ran out after %d of %d; nothing after that was sent. Call again with the rest", i, len(proposals))}
		}
		var held *docs.Document
		if i == 0 {
			held = d
		}
		res, err := edit.Apply(context.WithoutCancel(ctx), r.session, r.id, one, held, a.set.Mark, spansTo(policy, guard.EditWords))
		if err != nil {
			data.Edits[i].Outcome = res.Outcome
			if res.Outcome == edit.OutcomeUnknown {
				return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...),
					Error: fmt.Sprintf("edit %d (%q) was written and its answer was lost, so it may or may not be in the document. Read the document before sending it again", i+1, one.Quoted)}
			}
			return withStop(emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...),
				Error: fmt.Sprintf("edit %d (%q): %s", i+1, one.Quoted, err)}, err)
		}
		checks := res.Checks
		data.Edits[i] = editReport{Quoted: res.Quoted, Replacement: res.Replacement, Sent: true, Verified: res.Verified, Checks: &checks}
		warns = append(warns, about(res.Quoted, res.Warnings)...)
		if !res.Verified {
			return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...),
				Error: fmt.Sprintf("gdoc could not confirm that edit %d (%q) landed as written and painted, so nothing after it was sent. Look at it in the browser: %s",
					i+1, one.Quoted, publish.DocumentURL(r.id))}
		}
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(warns...)}
}

type seenData struct {
	DocumentID string `json:"document_id"`
	Colour     string `json:"colour"`
	Cleared    int    `json:"cleared"`
	Left       int    `json:"left"`
	Pending    int    `json:"pending"`
	Verified   bool   `json:"verified"`
}

// cmdSeen is `gdoc seen`: the edit yellow taken off every passage that carries
// it. It takes no quote and no range, so what it clears is what its own read
// found and nothing a caller named.
func cmdSeen(ctx context.Context, a *args) emit.Result {
	docID, err := documentID(a.target())
	if err != nil {
		return failed(err)
	}
	r, policy, err := openEdit(docID)
	if err != nil {
		return failed(err)
	}
	res, err := edit.Clear(context.WithoutCancel(ctx), r.session, r.id, spansTo(policy, guard.EditClear))
	data := seenData{DocumentID: r.id, Colour: edit.Yellow, Cleared: res.Cleared, Left: res.Left, Pending: res.Pending, Verified: res.Verified}
	if err != nil {
		if res.Outcome == edit.OutcomeUnknown {
			return emit.Result{OK: false, Data: data, Warnings: r.warnings(res.Warnings...), Error: err.Error()}
		}
		return withStop(emit.Result{OK: false, Data: data, Warnings: r.warnings(res.Warnings...), Error: err.Error()}, err)
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(res.Warnings...)}
}
