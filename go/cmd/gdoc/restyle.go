// The survey, and the restyle it makes safe.
//
// `gdoc restyle <url> --dry-run` reports what a document holds before anything
// is done to it: its threads with a witness for each, what is pending, its
// chips, its tabs, its named ranges and the revision the reads were made
// against. It writes to no document and to no file.
//
// `gdoc restyle <url> --from survey.json` is the other half. It reads that
// survey back, reads the document fresh, refuses it unless the survey is of
// this document and the document has not moved since, and only then opens the
// one grant in this binary that permits a direct edit. What it sends is
// typography: the page geometry, a paragraph's spacing and indent, a run's
// face, size and colour, and a table cell's padding and borders. None of the
// four request kinds the in-place level carries can change a character.
//
// The two halves are two runs, and a run naming both flags is refused. The
// survey is what makes the write safe, so it has to be a thing a person read
// before the write was asked for, rather than something the same run produced
// a moment earlier and never showed anybody.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/emit"
	"gdoc/internal/guard"
	"gdoc/internal/house"
	"gdoc/internal/restyle"
)

func cmdRestyle(raw []string) emit.Result {
	a, err := parseArgs(raw, flagSet{"--dry-run": false, "--from": true})
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	switch {
	case a.has("--dry-run") && a.has("--from"):
		return emit.Result{OK: false, Error: "restyle takes --dry-run or --from, and this run gave both: " +
			"the survey and the restyle are two runs, and the survey is what a person reads before the restyle is asked for"}
	case a.has("--dry-run"):
		return surveyRestyle(a)
	case a.has("--from"):
		return applyRestyle(a)
	}
	return emit.Result{OK: false, Error: "restyle needs --dry-run to survey a document, " +
		"or --from <survey.json> to apply the house style to it using a survey taken earlier"}
}

func surveyRestyle(a *args) emit.Result {
	r, err := open(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	ctx := context.Background()

	// The listing before the Docs read, which is the order every poll in this
	// binary makes its two reads in. A comment written between them is in
	// whichever ran second: listed first, it is a comment the Docs read has not
	// got to yet and a later read places it; read first, it is a comment in the
	// listing with no anchor in a document read a moment before it existed, so
	// the survey reports a thread placed nowhere and says the document holds
	// something it cannot point at.
	//
	// The cursor is nil, because a survey is the whole document rather than a
	// window on it.
	raws, err := comments.Fetch(ctx, r.session, r.id, nil)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}
	d, err := docs.Fetch(ctx, r.session, r.id)
	if err != nil {
		// A failed Docs read is a failed survey, and unlike the export it has no
		// honest partial answer: the chips, the pending suggestions, the named
		// ranges, the tab count and the revision id are all in that one read.
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}

	// The named ranges come out of the read above rather than out of a read of
	// their own. The whole document answers with them, so docs.NamedRangesURL
	// is for the caller that wants only them, which is the apply rechecking the
	// ranges rather than the prose. A second read here would be one more
	// request for facts already in hand.
	//
	// The export is the last of the three, and an export that failed is a
	// warning on a survey that still carries its threads. Survey turns the
	// error into that warning and reports every thread unmatched, exactly as
	// `comments --witness` behaves: the threads are the answer and the witness
	// is a second read on top of them.
	f, exportErr := exportFile(ctx, r)
	report, notes := restyle.Survey(restyle.Input{
		Document:  d,
		Comments:  raws,
		Export:    f,
		ExportErr: exportErr,
	})
	return emit.Result{OK: true, Data: report, Warnings: r.warnings(notes...)}
}

// restyleData is what the apply prints: the document it styled, the revision it
// left it on, what it planned to send and what Docs took. Every field is a
// count, an id or a list, and none of them says whether the document now looks
// right. That is read in the document, and past that it is Nail's.
type restyleData struct {
	DocumentID string `json:"document_id"`
	// SurveyRevisionID is the revision the survey was taken at, and RevisionID
	// the one the run ended on. Two fields rather than one, because the whole
	// safety of this command is that they started out equal.
	SurveyRevisionID string          `json:"survey_revision_id"`
	RevisionID       string          `json:"revision_id"`
	Tabs             int             `json:"tabs"`
	Planned          plannedCounts   `json:"planned"`
	Applied          restyle.Applied `json:"applied"`
	// ReadBack is the document read again once the batches landed: what
	// survived, and whether the style is really there. It is absent when no
	// batch was applied, because a document nothing was written to has nothing
	// to read back.
	ReadBack *restyle.ReadBack `json:"read_back,omitempty"`
	// Verified is the read-back's checks together. False is never a failure:
	// the batches Docs took are in the document either way, and a caller told
	// the run failed is a caller that runs it again.
	Verified bool `json:"verified"`
}

// plannedCounts is the plan as counts. The requests themselves are not printed:
// a batch of several hundred style objects is not something a reader reads, and
// what a caller has to know is how much was styled and what was left alone.
type plannedCounts struct {
	Requests   int `json:"requests"`
	Paragraphs int `json:"paragraphs"`
	Text       int `json:"text"`
	Cells      int `json:"cells"`
	// Bulleted is the paragraphs whose lists were left exactly as they are, and
	// Tables the tables whose column widths and row heights were not touched.
	// Both need a request kind the in-place level does not carry.
	Bulleted int `json:"bulleted"`
	Tables   int `json:"tables"`
	// Unstyled names the named styles the house has no look for. Those
	// paragraphs keep the look they had: gdoc maps none of them to the nearest
	// style it does know, because that would be inferring structure.
	Unstyled []string `json:"unstyled,omitempty"`
}

// applyRestyle is the write half. The order below is the milestone's, and every
// refusal in the first part of it happens before the grant is opened.
func applyRestyle(a *args) emit.Result {
	saved, err := readSurvey(a.flags["--from"])
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	docID, err := documentID(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	// A survey of another document is refused before a session is opened. The
	// survey is the only record of what this document held before the run, and
	// one taken of a different document says nothing about this one: rechecking
	// against it would be rechecking against somebody else's facts.
	if saved.DocumentID != docID {
		return emit.Result{OK: false, Error: fmt.Sprintf(
			"the survey was taken of document %s, and this run is of %s", saved.DocumentID, docID)}
	}
	if saved.Tabs > 1 {
		return emit.Result{OK: false, Error: fmt.Sprintf(
			"the survey found %d tabs, and a style request names a range, which means nothing without saying which tab it is in", saved.Tabs)}
	}
	cfg, err := house.Load()
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}

	// The document is handed in at the level every handed-in document gets:
	// read, comment and suggest, and no direct edit. The grant comes later, and
	// only if the recheck below holds.
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)
	s, err := openSession(p)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	r := &reach{id: docID, session: s}
	ctx := context.Background()

	d, err := docs.Fetch(ctx, r.session, r.id)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}
	data := restyleData{
		DocumentID:       docID,
		SurveyRevisionID: saved.RevisionID,
		RevisionID:       d.RevisionID,
		Tabs:             len(d.Tabs),
	}
	// A document that moved is refused rather than restyled against a survey
	// that no longer describes it. requiredRevisionId would refuse the batch
	// itself a moment later, and this refusal is the same rule said earlier and
	// in words: what the survey reported is what a person read before asking
	// for the write.
	if d.RevisionID != saved.RevisionID {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(), Error: fmt.Sprintf(
			"the survey was taken at revision %s and the document is on %s, so it moved after the survey: survey it again and read what changed before restyling it",
			saved.RevisionID, d.RevisionID)}
	}
	if d.MultiTab() {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(), Error: fmt.Sprintf(
			"the document has %d tabs, and a restyle styles a document with one: a style request names a range, and a range means nothing without saying which tab it is in",
			len(d.Tabs))}
	}

	// This is the line. It upgrades one document, for this run, from suggest to
	// direct edit, and it is the widest thing gdoc can be asked to do. Three
	// things hold it in: the id is the one the caller named and the survey
	// agrees with, the document is provably the one the survey described, and
	// the guard carries only the four styling request kinds at that level, none
	// of which can change a character. The grant dies with the process.
	p.GrantInPlace(docID)

	plan := restyle.TabRequests(d.Tabs[0], cfg)
	// The page first, because it names no range and a reader comparing a batch
	// against a log should find the document's own geometry at the top of it.
	requests := append([]map[string]any{restyle.PageRequest(cfg)}, plan.Requests...)
	data.Planned = plannedCounts{
		Requests:   len(requests),
		Paragraphs: plan.Paragraphs,
		Text:       plan.Text,
		Cells:      plan.Cells,
		Bulleted:   plan.Bulleted,
		Tables:     plan.Tables,
		Unstyled:   plan.Unstyled,
	}

	applied, applyErr := restyle.Apply(ctx, r.session, docID, requests, d.RevisionID)
	warns := applied.Warnings
	// The warnings go on the envelope and nowhere else. A caller reading them
	// in two places has two chances to read a different list, and the envelope
	// is where every other command puts them.
	applied.Warnings = nil
	data.Applied = applied
	data.RevisionID = applied.RevisionID

	// The read-back, and it runs on a failed run too. A run that stopped at
	// batch twelve is exactly the run somebody needs the preservation facts
	// for, and the half-styled document is still a document whose comments
	// either survived or did not. What is skipped is a run that wrote nothing:
	// there the document is as it was, and three reads would answer a question
	// nobody asked.
	if applied.Batches > 0 {
		rb, notes := readBack(ctx, r, *saved, requests, plan)
		data.ReadBack = rb
		if rb != nil {
			data.Verified = rb.Verified
		}
		warns = append(warns, notes...)
	}
	if applyErr != nil {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...), Error: applyErr.Error()}
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(warns...)}
}

// readBack is the document read again once the styling landed, and the two
// halves of the verification made from it.
//
// The three reads are the survey's three, in the survey's order: the comment
// listing, the Docs read, the docx export. The listing goes first for the
// reason every poll in this binary puts it first, and the export is last
// because it is the witness on top of the threads rather than a read the
// answer needs.
//
// The Docs read is made here rather than through docs.Fetch because both halves
// want it: the preservation half wants the decoded tree, and the landing half
// wants the bytes, which carry the styling internal/docs deliberately does not
// decode. One read, two readers, so the two halves cannot be looking at
// different documents.
//
// A read that failed is a warning and no read-back at all. Reporting a
// preservation half made from a listing that never arrived would name every
// thread in the survey as gone, which is the one warning that must never cry
// wolf.
func readBack(ctx context.Context, r *reach, before restyle.Report, sent []map[string]any, plan restyle.Plan) (*restyle.ReadBack, []string) {
	raws, err := comments.Fetch(ctx, r.session, r.id, nil)
	if err != nil {
		return nil, []string{fmt.Sprintf(
			"the comment listing could not be read back, so nothing here says what survived the restyle: %v", err)}
	}
	var raw json.RawMessage
	if err := r.session.GetJSON(ctx, docs.URL(r.id), &raw); err != nil {
		return nil, []string{fmt.Sprintf(
			"the document could not be read back, so nothing here says what survived the restyle or whether the style landed: %v", err)}
	}
	d, err := docs.Parse(raw)
	if err != nil {
		return nil, []string{fmt.Sprintf(
			"the document was read back and did not decode, so nothing here says what survived the restyle: %v", err)}
	}
	f, exportErr := exportFile(ctx, r)
	rb, notes := restyle.Verify(before, restyle.Input{
		Document:  d,
		Comments:  raws,
		Export:    f,
		ExportErr: exportErr,
	}, raw, sent, plan)
	return &rb, notes
}

// savedSurvey is the envelope the survey run printed, read back. Data is the
// same struct the survey printed, which is what makes a strict read possible:
// a key the survey never wrote is a file somebody edited or a file from another
// tool, and either way it is not the survey this recheck rests on.
type savedSurvey struct {
	OK       bool           `json:"ok"`
	Data     restyle.Report `json:"data"`
	Error    string         `json:"error,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
}

// readSurvey reads what `restyle --dry-run` printed and the caller saved.
//
// The read is strict, for the reason parseArgs refuses an unknown flag and
// frontmatter reads with yaml.Strict(). This file is the only record of what
// the document held before the run and the only thing standing between a
// direct-edit grant and a document nobody looked at, so a file gdoc half
// understands is not one to open that grant on.
//
// A survey that failed is refused by name. It carries no revision id and no
// counts, so there is nothing to recheck against, and reading its zero fields
// as facts would be the one false fact in the file this command trusts most.
func readSurvey(path string) (*restyle.Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("the survey file could not be read: %w", err)
	}
	var saved savedSurvey
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&saved); err != nil {
		return nil, fmt.Errorf("%s is not a survey `gdoc restyle --dry-run` printed: %w", path, err)
	}
	// Decode stops at the end of the first value, where Unmarshal refused a
	// file with anything behind it. Two objects is a file somebody appended a
	// second run to, and rechecking against the first half of it silently is
	// the same mistake as reading past an unknown key.
	if dec.More() {
		return nil, fmt.Errorf("%s carries more than one JSON object, and which survey counts is not decided here", path)
	}
	if !saved.OK {
		return nil, fmt.Errorf("%s is a survey that did not succeed, so there is nothing to recheck the document against: %s",
			path, saved.Error)
	}
	if saved.Data.DocumentID == "" {
		return nil, fmt.Errorf("%s names no document, so it cannot be a survey of the one being restyled", path)
	}
	if saved.Data.RevisionID == "" {
		return nil, fmt.Errorf(
			"%s carries no revision id, and a restyle sends nothing without one: it is what refuses a batch on a document that moved after the survey",
			path)
	}
	return &saved.Data, nil
}
