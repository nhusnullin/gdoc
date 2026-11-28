package edit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gdoc/internal/docs"
	"gdoc/internal/docsreq"
	"gdoc/internal/propose"
)

// Yellow is the colour an edit paints its words. The guard states the same
// colour as its own literal, and TestTheEditYellowIsTheGuardsYellow in
// cmd/gdoc holds the two together.
const Yellow = "#FFF59D"

// MaxUnits is the most one edit replaces and the most it writes, in UTF-16
// code units. The guard holds the same ceiling.
const MaxUnits = 300

// Session is what this package needs of a session: the read and the one write.
type Session interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	PostJSON(ctx context.Context, rawURL string, body any, into any) error
}

// Span is one half-open range of body indexes.
type Span struct{ Start, End int }

// Grant is how the caller tells its guard which spans the next batch may
// touch, and at which revision. This package never names the guard.
type Grant func(revision string, spans []Span)

// Checks is what the read-back found, one fact each.
type Checks struct {
	Words   bool `json:"words"`   // the new words stand where the quote stood, and nothing there is pending
	Painted bool `json:"painted"` // every one of them carries the edit yellow
	Look    bool `json:"look"`    // and the look the old words had
}

func (c Checks) all() bool { return c.Words && c.Painted && c.Look }

// Result is what one edit left behind.
type Result struct {
	Quoted      string   `json:"quoted"`
	Replacement string   `json:"replacement"`
	Verified    bool     `json:"verified"`
	Checks      Checks   `json:"checks"`
	Warnings    []string `json:"warnings,omitempty"`
	// Outcome is OutcomeUnknown when the batch went out and no answer came
	// back, and empty everywhere else.
	Outcome string `json:"outcome,omitempty"`
}

// OutcomeUnknown is the one outcome there is, as in internal/propose.
const OutcomeUnknown = "unknown"

// Check is the shape rule for one edit, answered from the entry alone so a
// caller can ask it of every entry before the first one leaves the machine.
func Check(p propose.Proposal, mark string) error {
	if p.Kind != propose.KindWords {
		return fmt.Errorf("an edit replaces words with words, and this entry is the %q kind; propose it in a run without --edit, where it is a suggestion", p.Kind)
	}
	if err := p.Check(mark); err != nil {
		return err
	}
	if n := docsreq.Len(p.Replacement); n > MaxUnits {
		return fmt.Errorf("the replacement is %d units long, and one edit writes at most %d; split it, or propose it as a suggestion", n, MaxUnits)
	}
	if n := docsreq.Len(p.Quoted); n > MaxUnits {
		return fmt.Errorf("the quoted text is %d units long, and one edit replaces at most %d; quote less, or propose it as a suggestion", n, MaxUnits)
	}
	return nil
}

// Apply writes one edit: read, find the quote, check the ground, grant the
// span, write once, read back. held is a read the caller already has, which
// must have just come back; nil reads.
//
// It fails only before the write. After the batch has gone out the change
// exists, and everything from there is reported rather than raised.
func Apply(ctx context.Context, s Session, docID string, p propose.Proposal, held *docs.Document, mark string, grant Grant) (Result, error) {
	out := Result{Quoted: p.Quoted, Replacement: p.Replacement}
	if err := Check(p, mark); err != nil {
		return out, err
	}
	d, err := readOneTab(ctx, s, docID, held)
	if err != nil {
		return out, err
	}
	if d.RevisionID == "" {
		return out, errors.New("the read named no revision, and an edit is sent only against the revision its span was read in")
	}
	r, err := propose.FindSpanNear(d, p.Quoted, p.Near)
	if err != nil {
		return out, err
	}
	look, err := ground(d, r)
	if err != nil {
		return out, err
	}
	grant(d.RevisionID, []Span{{Start: r.Start, End: r.End}})
	if err := post(ctx, s, docID, Batch(r, p.Replacement, d.RevisionID), &out.Outcome, &out.Warnings); err != nil {
		return out, err
	}

	after, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("the document could not be read back, so the edit could not be confirmed: %v", err))
		return out, nil
	}
	var why []string
	out.Checks, why = readBack(after, r, p.Replacement, look)
	out.Warnings = append(out.Warnings, why...)
	out.Verified = out.Checks.all()
	return out, nil
}

// Batch is the whole write: delete the quote, insert the words where it
// started, paint exactly those words. One batch, so the indexes agree, sent
// against the revision the span was read in.
func Batch(r docs.Range, replacement, revision string) []byte {
	colour, _ := docsreq.Color(Yellow)
	body := map[string]any{
		"requests": []any{
			map[string]any{"deleteContentRange": map[string]any{
				"range": map[string]any{"startIndex": r.Start, "endIndex": r.End},
			}},
			map[string]any{"insertText": map[string]any{
				"location": map[string]any{"index": r.Start},
				"text":     replacement,
			}},
			map[string]any{"updateTextStyle": map[string]any{
				"range":     map[string]any{"startIndex": r.Start, "endIndex": r.Start + docsreq.Len(replacement)},
				"textStyle": map[string]any{"backgroundColor": colour},
				"fields":    "backgroundColor",
			}},
		},
		"writeControl": map[string]any{"requiredRevisionId": revision},
	}
	raw, _ := json.Marshal(body)
	return raw
}

func readOneTab(ctx context.Context, s Session, docID string, held *docs.Document) (*docs.Document, error) {
	d := held
	if d == nil {
		var err error
		if d, err = docs.Fetch(ctx, s, docID); err != nil {
			return nil, fmt.Errorf("the document could not be read before editing: %w", err)
		}
	}
	if d.MultiTab() {
		return nil, propose.MoreThanOneTab("the document has %d tabs, and an edit is written into a document with one", len(d.Tabs))
	}
	return d, nil
}

// post sends one batch. It returns an error only when nothing was written, or
// when nobody can say whether it was.
func post(ctx context.Context, s Session, docID string, body []byte, outcome *string, warns *[]string) error {
	err := s.PostJSON(ctx, propose.BatchURL(docID), json.RawMessage(body), nil)
	if err == nil {
		return nil
	}
	var lost interface{ Unknown() bool }
	if errors.As(err, &lost) && lost.Unknown() {
		*outcome = OutcomeUnknown
		return fmt.Errorf("the change was written and its answer was lost, so it may or may not be in the document: %w", err)
	}
	var sent interface{ Sent() bool }
	if errors.As(err, &sent) && sent.Sent() {
		*warns = append(*warns, "the change was accepted by Docs and its answer could not be read: "+err.Error())
		return nil
	}
	return fmt.Errorf("the change could not be written: %w", err)
}

// ground says whether the quoted words stand on plain ground, and returns the
// look they carry.
func ground(d *docs.Document, r docs.Range) (string, error) {
	runs := paragraphAt(d.Tabs[0].Body, r.Start)
	var inside []docs.Run
	var before, behind *docs.Run
	for i, run := range runs {
		switch {
		case run.EndIndex <= r.Start:
			before = &runs[i]
		case run.StartIndex >= r.End:
			if behind == nil {
				behind = &runs[i]
			}
		default:
			inside = append(inside, run)
		}
	}
	if len(inside) == 0 {
		return "", fmt.Errorf("the quoted text at [%d,%d) is in no paragraph this read holds", r.Start, r.End)
	}
	look := inside[0].Look
	for _, run := range inside {
		switch {
		case run.Background != "":
			return "", fmt.Errorf("the quoted words already carry the highlight %s; an edit paints its own words, and it leaves a mark somebody made alone. Propose this one as a suggestion", run.Background)
		case run.Link != nil:
			return "", errors.New("the quoted words are a link, or part of one; new words written there would take the link with them. Propose this one as a suggestion")
		case run.Look != look:
			return "", errors.New("the quoted words change style part way, bold or italic or a colour; new words would take one of the two. Quote the words of one style, or propose this one as a suggestion")
		}
	}
	// Docs gives inserted text the style of its neighbour, so the neighbour
	// has to be styled like the words that go. The neighbour is the character
	// beside the quote: where the quote starts part way into a run, that
	// character is in the run and already has its look, and an earlier run
	// has nothing to say. The run behind is asked only where nothing stands
	// in front, and only where the quote reaches the end of its own run.
	var neighbour *docs.Run
	switch {
	case inside[0].StartIndex < r.Start:
	case before != nil:
		neighbour = before
	case inside[len(inside)-1].EndIndex <= r.End:
		neighbour = behind
	}
	if neighbour != nil && neighbour.Kind == docs.KindText && (neighbour.Look != look || neighbour.Link != nil) {
		return "", errors.New("the quoted words stand at a style edge: the text beside them is a link, or bold, italic or coloured where they are not. New words would take that style. Quote one more word, or propose this one as a suggestion")
	}
	return look, nil
}

// readBack is the three facts about an edit, read out of the document as it
// stands after the write.
func readBack(d *docs.Document, r docs.Range, replacement, look string) (Checks, []string) {
	var c Checks
	end := r.Start + docsreq.Len(replacement)
	var got strings.Builder
	pending, painted, sameLook := false, true, true
	if len(d.Tabs) == 1 {
		for _, run := range paragraphAt(d.Tabs[0].Body, r.Start) {
			if run.EndIndex <= r.Start || run.StartIndex >= end || run.Kind != docs.KindText {
				continue
			}
			// A run may reach past the insertion: Docs joins the new words to
			// a neighbour of the same style, a passage an earlier edit painted
			// among them. So only the part inside the insertion is asked about.
			got.WriteString(clip(run, r.Start, end))
			pending = pending || len(run.InsertionIDs) > 0 || len(run.DeletionIDs) > 0
			painted = painted && strings.EqualFold(run.Background, Yellow)
			sameLook = sameLook && run.Look == look
		}
	}
	var warns []string
	c.Words = got.String() == replacement && !pending
	if !c.Words {
		warns = append(warns, fmt.Sprintf("the read-back carries %q where %q was written, or carries it as a pending suggestion", got.String(), replacement))
		return c, warns
	}
	c.Painted = painted
	if !painted {
		warns = append(warns, "the new words are in the document and the edit yellow is not exactly on them, so gdoc seen may not find them")
	}
	c.Look = sameLook
	if !sameLook {
		warns = append(warns, "the new words took another style than the words they replaced; look at them in the document")
	}
	return c, warns
}

// clip is the part of a run's text between two document indexes.
func clip(run docs.Run, start, end int) string {
	var b strings.Builder
	at := run.StartIndex
	for _, c := range run.Text {
		if at >= start && at < end {
			b.WriteRune(c)
		}
		at += docsreq.Len(string(c))
	}
	return b.String()
}

// paragraphAt is the runs of the paragraph holding index at, tables walked
// into, and nil where no paragraph does.
func paragraphAt(bs []docs.Block, at int) []docs.Run {
	for _, b := range bs {
		if b.Paragraph != nil {
			runs := b.Paragraph.Runs
			if len(runs) > 0 && runs[0].StartIndex <= at && at < runs[len(runs)-1].EndIndex {
				return runs
			}
			continue
		}
		if b.Table == nil {
			continue
		}
		for _, row := range b.Table.Rows {
			for _, c := range row {
				if runs := paragraphAt(c.Blocks, at); runs != nil {
					return runs
				}
			}
		}
	}
	return nil
}
