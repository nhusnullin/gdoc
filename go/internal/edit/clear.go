package edit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gdoc/internal/docs"
)

// ClearResult is what one clear did.
type ClearResult struct {
	Cleared  int      `json:"cleared"`  // passages whose yellow this run took off
	Left     int      `json:"left"`     // passages still yellow in the read-back
	Pending  int      `json:"pending"`  // yellow runs inside a pending suggestion, which a clear leaves alone
	Verified bool     `json:"verified"` // none left, and the text is the same character for character
	Warnings []string `json:"warnings,omitempty"`
	Outcome  string   `json:"outcome,omitempty"`
}

// Painted is every passage of body text carrying exactly the edit yellow, with
// runs that touch joined into one. pending counts the yellow runs that sit
// inside a suggestion, which are not in the list.
func Painted(d *docs.Document) (spans []Span, pending int) {
	if len(d.Tabs) != 1 {
		return nil, 0
	}
	for _, run := range textRuns(d.Tabs[0].Body) {
		if !strings.EqualFold(run.Background, Yellow) {
			continue
		}
		if len(run.InsertionIDs) > 0 || len(run.DeletionIDs) > 0 {
			pending++
			continue
		}
		if n := len(spans); n > 0 && spans[n-1].End == run.StartIndex {
			spans[n-1].End = run.EndIndex
			continue
		}
		spans = append(spans, Span{Start: run.StartIndex, End: run.EndIndex})
	}
	return spans, pending
}

// ClearBatch sets the background of every span to no colour, against the
// revision the spans were read in. It names no character.
func ClearBatch(spans []Span, revision string) []byte {
	reqs := make([]any, 0, len(spans))
	for _, s := range spans {
		reqs = append(reqs, map[string]any{"updateTextStyle": map[string]any{
			"range":     map[string]any{"startIndex": s.Start, "endIndex": s.End},
			"textStyle": map[string]any{"backgroundColor": map[string]any{}},
			"fields":    "backgroundColor",
		}})
	}
	raw, _ := json.Marshal(map[string]any{
		"requests":     reqs,
		"writeControl": map[string]any{"requiredRevisionId": revision},
	})
	return raw
}

// Clear takes the edit yellow off every passage that carries it: read, grant
// those spans, write once, read back. A document with no yellow is read and
// not written.
func Clear(ctx context.Context, s Session, docID string, grant Grant) (ClearResult, error) {
	var out ClearResult
	d, err := readOneTab(ctx, s, docID, nil)
	if err != nil {
		return out, err
	}
	spans, pending := Painted(d)
	out.Pending = pending
	if pending > 0 {
		out.Warnings = append(out.Warnings, fmt.Sprintf("%d yellow passages sit inside a pending suggestion and were left as they are", pending))
	}
	if len(spans) == 0 {
		out.Verified = true
		return out, nil
	}
	if d.RevisionID == "" {
		return out, fmt.Errorf("the read named no revision, and a clear is sent only against the revision its spans were read in")
	}
	before := text(d)
	grant(d.RevisionID, spans)
	if err := post(ctx, s, docID, ClearBatch(spans, d.RevisionID), &out.Outcome, &out.Warnings); err != nil {
		return out, err
	}
	out.Cleared = len(spans)

	after, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("the document could not be read back, so the clear could not be confirmed: %v", err))
		return out, nil
	}
	left, _ := Painted(after)
	out.Left = len(left)
	same := text(after) == before
	if !same {
		out.Warnings = append(out.Warnings, "the text read back is not the text read before the clear; somebody may have typed in between, so look at the document")
	}
	out.Verified = out.Left == 0 && same
	return out, nil
}

// text is the body's words in reading order, for the one question a clear
// asks of them: are they the same afterwards.
func text(d *docs.Document) string {
	var b strings.Builder
	if len(d.Tabs) == 1 {
		for _, run := range textRuns(d.Tabs[0].Body) {
			b.WriteString(run.Text)
		}
	}
	return b.String()
}

func textRuns(bs []docs.Block) []docs.Run {
	var out []docs.Run
	for _, b := range bs {
		if b.Paragraph != nil {
			for _, r := range b.Paragraph.Runs {
				if r.Kind == docs.KindText {
					out = append(out, r)
				}
			}
			continue
		}
		if b.Table == nil {
			continue
		}
		for _, row := range b.Table.Rows {
			for _, c := range row {
				out = append(out, textRuns(c.Blocks)...)
			}
		}
	}
	return out
}
