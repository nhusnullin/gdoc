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
//
// `gdoc restyle <url> --from survey.json --fields fields.json` adds the house
// template to that, and it adds it as a proposal. M7c, and the whole of its
// design is that the run has two permissions rather than one:
//
//   - Phase 1 proposes the prelude, which is the cover, the three front-matter
//     tables and the legend. It goes out on a policy that granted nothing at
//     all, in writeMode SUGGEST, which is exactly what `propose` sends every
//     day. Nail accepts it in the browser, or rejects it and the document is as
//     it was.
//   - Phase 2 is M7b's styling, unchanged: a second policy, GrantInPlace, and
//     the four request kinds none of which can change a character.
//
// They are two policies because the guard's rule is right and stays: at
// LevelInPlace the allowlist gates every batchUpdate whatever writeMode says,
// so a granted document cannot take an insertText and a document with no grant
// cannot take a direct edit. Neither phase can do the other's job, and that is
// the point rather than an inconvenience. The one permission this milestone
// added is AllowMarker, for the named range over what phase 1 proposed, because
// createNamedRange is the one request Docs refuses to apply as a suggestion.
//
// Phase 1 goes first and phase 2 reads the document again before it builds
// anything. The prelude is text, so every index a styling request names moved
// when it landed, and the styling then walks past the prelude's own span: those
// paragraphs are gdoc's own, stating the cover's sizes and colours in full, and
// giving them the house body look would leave Nail accepting a cover that had
// already been turned into prose.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"gdoc/internal/comments"
	"gdoc/internal/cover"
	"gdoc/internal/docs"
	"gdoc/internal/emit"
	"gdoc/internal/guard"
	"gdoc/internal/house"
	"gdoc/internal/prelude"
	"gdoc/internal/restyle"
)

func cmdRestyle(raw []string) emit.Result {
	a, err := parseArgs(raw, flagSet{"--dry-run": false, "--from": true, "--fields": true})
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	switch {
	case a.has("--dry-run") && a.has("--from"):
		return emit.Result{OK: false, Error: "restyle takes --dry-run or --from, and this run gave both: " +
			"the survey and the restyle are two runs, and the survey is what a person reads before the restyle is asked for"}
	case a.has("--dry-run") && a.has("--fields"):
		return emit.Result{OK: false, Error: "restyle takes --dry-run or --fields, and this run gave both: " +
			"--dry-run surveys a document and writes nothing to it, while --fields proposes the house template into one"}
	case a.has("--fields") && !a.has("--from"):
		return emit.Result{OK: false, Error: "--fields proposes the house template into a document, and that needs --from <survey.json>: " +
			"the survey is what says the document has not moved since a person read what it held"}
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
	SurveyRevisionID string `json:"survey_revision_id"`
	RevisionID       string `json:"revision_id"`
	Tabs             int    `json:"tabs"`
	// Prelude is phase 1: the house template proposed into the document, and
	// the marker written over what was proposed. It is absent on a run that
	// named no --fields, which is M7b's styling-only restyle, because a field
	// reporting nothing proposed on a run that proposed nothing says the
	// command has a phase it does not.
	Prelude *preludeData    `json:"prelude,omitempty"`
	Planned plannedCounts   `json:"planned"`
	Applied restyle.Applied `json:"applied"`
	// ReadBack is the document read again once the batches landed: what
	// survived, and whether the style is really there. It is absent when
	// nothing reached the document, because a document nothing was written to
	// has nothing to read back. A batch Docs accepted whose answer could not be
	// read is not that case: it may be in the document, so it is read back.
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
	// Skipped is the paragraphs and tables phase 2 walked past because phase 1
	// proposed them, or proposed deleting them: on a replace run the prelude
	// an earlier run left is still real text, behind the words replacing it.
	// Either way they are gdoc's own words, stating the cover's own sizes, and
	// the house body look is not what they are meant to wear.
	Skipped int `json:"skipped"`
	// Unstyled names the named styles the house has no look for. Those
	// paragraphs keep the look they had: gdoc maps none of them to the nearest
	// style it does know, because that would be inferring structure.
	Unstyled []string `json:"unstyled,omitempty"`
}

// preludeData is phase 1 as facts: what the house template came to, where it
// landed, what Docs took, and whether the marker over it was written.
//
// Nothing here says whether the cover is right. That is read in the document,
// by the person who is being asked to accept it, and past that it is Nail's.
type preludeData struct {
	// Requests is how many requests the prelude came to, Paragraphs how many
	// paragraphs it writes, Tables how many tables it inserts and Cells how
	// many of their cells it fills.
	Requests   int `json:"requests"`
	Paragraphs int `json:"paragraphs"`
	Tables     int `json:"tables"`
	Cells      int `json:"cells"`
	// Start and End are the span the prelude was proposed into, which is the
	// range the marker covers and the range the read-back counts suggestion
	// ids inside. It is not the range phase 2 walks past: on a replace run
	// that one is wider, and prelude.Result.Occupies is where the arithmetic
	// and the reason for it live.
	Start int `json:"start"`
	End   int `json:"end"`
	// Applied is what Docs took of the prelude batches.
	Applied restyle.Applied `json:"applied"`
	// MarkerCreated says the named range over the prelude was written. It is
	// gdoc's whole memory of having been here, so a run that proposed a prelude
	// and could not mark it stops rather than styling: a second run over an
	// unmarked prelude would propose a second cover on top of the first.
	MarkerCreated bool `json:"marker_created"`
	// MarkerMaybeCreated says Docs accepted the marker batch and its answer
	// could not be read, so the marker may be in the document and nothing here
	// can say. It is a second flag rather than a truer MarkerCreated for the
	// reason restyle.Applied keeps MaybeApplied beside Batches: what Docs
	// confirmed and what may have happened are two facts, and a field that
	// folded them could not say which it was. Both are printed on every run,
	// because a flag that vanishes when it is false cannot be read the same way
	// twice.
	MarkerMaybeCreated bool `json:"marker_maybe_created"`
	// Replaced is the marker of a prelude a run before this one left, which
	// this run proposed deleting. It is absent on a document carrying none,
	// which is either a document gdoc has never touched or one whose prelude
	// was rejected.
	Replaced *prelude.Marker `json:"replaced,omitempty"`
	// Manual is what the prelude could not propose at all, each with the menu
	// path a person takes instead.
	Manual []prelude.ManualStep `json:"manual,omitempty"`
	// ReadBack is phase 1 read out of the document once both phases had run:
	// whether every piece of the prelude carries a suggestion id, whether the
	// marker is over the span it was proposed into, and whether the author's
	// own text is character for character what it was. It is absent when the
	// run could not read the document back at all, which is a warning rather
	// than a claim that nothing is there.
	ReadBack *prelude.Check `json:"read_back,omitempty"`
}

// phaseOne is what phase 2 and the read-back need to know about phase 1: the
// span the prelude was proposed into, the wider span phase 2 walks past, the
// author's own text as it stood before a word of it was proposed, and where the
// answer goes.
//
// The two spans are two fields on purpose. They are the same on a first run and
// they are not on a replace run, and the comments on each of them say what each
// one answers.
//
// It is nil on a run that named no --fields, which is M7b's styling-only
// restyle, and every use of it below reads that nil as "there was no phase 1"
// rather than as "phase 1 found nothing".
type phaseOne struct {
	// span is what phase 1 proposed: the marker's range, and the range the
	// read-back counts suggestion ids inside.
	span restyle.Span
	// skip is what phase 2 walks past, which is span and, on a replace run, the
	// prelude behind it that phase 1 proposed deleting. A suggested delete
	// marks text rather than removing it, so those words are still in the
	// document and still gdoc's own, and styled they would flatten a cover Nail
	// may yet reject the deletion of. It is prelude.Result.Occupies', which is
	// where the arithmetic and the reason for it live.
	//
	// It is not span, and the read-back is the reason the two are separate: the
	// replaced prelude carries a deletion id rather than an insertion one, so
	// asked about skip the read-back would report gdoc's own replaced words as
	// text somebody wrote.
	skip   restyle.Span
	before string
	data   *preludeData
}

// readFields reads the cover's values out of the file --fields names.
//
// A restyle has no note behind it, so there is no front matter to read the
// cover from. The read is internal/cover's, strict the way readSurvey and
// readProposals read theirs, and it happens here rather than after a session is
// opened: a file gdoc half understands must never reach a document, and a
// misspelled key is a cover line that would silently never print.
func readFields(path string) (cover.Fields, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return cover.Fields{}, fmt.Errorf("the fields file could not be read: %w", err)
	}
	f, err := cover.ReadFields(raw)
	if err != nil {
		return cover.Fields{}, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
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
	// The cover's values, and they are read before a session is opened. A run
	// that named --fields is a run that will propose words into somebody's
	// document, so the file those words come out of is checked while nothing
	// has left the machine.
	var fields *cover.Fields
	if path := a.flags["--fields"]; path != "" {
		read, err := readFields(path)
		if err != nil {
			return emit.Result{OK: false, Error: err.Error()}
		}
		fields = &read
	}

	// The document is handed in at the level every handed-in document gets:
	// read, comment and suggest, and no direct edit. The grant comes later, and
	// only if the recheck below holds. On a run that proposes a prelude the
	// grant never comes at all on this policy: phase 1 sends its suggestions
	// here, and phase 2 opens a second policy of its own.
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

	if fields != nil {
		return proposeThenStyle(ctx, r, *saved, cfg, *fields, d, data)
	}

	// This is the line. It upgrades one document, for this run, from suggest to
	// direct edit, and it is the widest thing gdoc can be asked to do. Three
	// things hold it in: the id is the one the caller named and the survey
	// agrees with, the document is provably the one the survey described, and
	// the guard carries only the four styling request kinds at that level, none
	// of which can change a character. The grant dies with the process.
	p.GrantInPlace(docID)
	return styleDocument(ctx, r, *saved, cfg, d.Tabs[0], d.RevisionID, nil, data)
}

// proposeThenStyle is the two-phase run: the prelude proposed, the marker
// written, and then M7b's styling on a second policy.
//
// The order is the milestone's and so is the split. Phase 1 sends on the policy
// it is handed, which granted nothing and never will: a SUGGEST batchUpdate on
// a handed-in document is what internal/propose has sent every day since M3, so
// the prelude needs no permission this milestone added. Phase 2 opens a policy
// of its own, and that second policy is the only one in the run that ever holds
// a grant. A caller that collapsed the two into one policy would have undone
// the whole shape of this milestone, and the guard would refuse it in both
// directions: at LevelInPlace an insertText is not on the allowlist, and with no
// grant a direct styling batch is not a suggestion.
//
// It takes no policy. Phase 1 sends on r's own, which granted nothing and never
// will, and phase 2 builds its own below: a parameter naming the caller's
// policy would read as though this function still had a hand on it.
func proposeThenStyle(ctx context.Context, r *reach, saved restyle.Report,
	cfg *house.Config, fields cover.Fields, d *docs.Document, data restyleData) emit.Result {
	// What to propose, and what a run before this one left. Decide refuses a
	// document whose prelude is still pending: replacing it would propose
	// deleting text that has never been written, and the answer to that is the
	// suggestion already in front of Nail.
	res, err := prelude.Propose(cfg, fields, d)
	if err != nil {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(), Error: err.Error()}
	}
	pre := &preludeData{
		Requests:   len(res.Requests),
		Paragraphs: res.Paragraphs,
		Tables:     res.Tables,
		Cells:      res.Cells,
		Start:      res.Start,
		End:        res.End,
		Replaced:   res.Replaces,
		Manual:     res.Manual,
	}
	data.Prelude = pre

	proposed, proposeErr := restyle.Suggest(ctx, r.session, r.id, res.Requests, d.RevisionID)
	warns := proposed.Warnings
	proposed.Warnings = nil
	pre.Applied = proposed
	data.RevisionID = proposed.RevisionID
	if proposeErr != nil {
		// A failed phase 1 does not run phase 2. What the document carries is
		// whatever of the prelude Docs took, and every character of that is a
		// suggestion, so rejecting it puts the document back.
		//
		// A run that stopped part way through the prelude left words in the
		// document with no marker over them, which is the same hazard the
		// marker failure below names and the same sentence answers it: the
		// next run finds no marker, proposes at index 1, and puts a second
		// cover in front of the first. MaybeApplied is read beside the count
		// because the live prelude goes out in one batch, so the path where
		// Docs accepted it and the answer could not be read is the path where
		// the whole unmarked prelude may be there.
		if proposed.Batches > 0 || proposed.MaybeApplied {
			warns = append(warns, "the prelude was proposed in part and nothing marks it, so gdoc has no record of having written it: "+
				"accept or reject what is in the document before running this again, or the next run proposes a second prelude in front of it")
		}
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...), Error: fmt.Sprintf(
			"the prelude phase stopped, so the body was not styled: %v", proposeErr)}
	}

	// Phase 2's policy, and it is a second one rather than a grant added to the
	// first. Two sessions go with it, because a session is built from a policy
	// and the first request in its history is already judged against it.
	p2 := guard.NewPolicy()
	p2.AllowFile(r.id, guard.LevelSuggest)
	s2, err := openSession(p2)
	if err != nil {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...), Error: err.Error()}
	}
	r2 := &reach{id: r.id, session: s2}

	// The fresh read. The prelude is text, so every index a styling request
	// names moved when it landed, and a plan built from the read phase 1 was
	// computed from would name ranges that are now somebody else's words.
	fresh, err := docs.Fetch(ctx, r2.session, r2.id)
	if err != nil {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(append(warns, r2.warnings()...)...), Error: fmt.Sprintf(
			"the prelude was proposed and the document could not be read again, so nothing was styled: %v", err)}
	}
	// The prelude phase's own last-batch warning, if it made one, said the
	// revision reported is not the one the document carries. This read has just
	// answered that, so the sentence goes rather than standing beside a
	// revision Docs named.
	warns = dropWarning(warns, restyle.RevisionUnconfirmedWarning)
	data.RevisionID = fresh.RevisionID
	if fresh.MultiTab() {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(append(warns, r2.warnings()...)...), Error: fmt.Sprintf(
			"the document has %d tabs when read again, and a restyle styles a document with one", len(fresh.Tabs))}
	}

	// The grant, and the one permission this milestone added beside it. The
	// marker is a named range over exactly what phase 1 proposed, and it is
	// written directly because Docs refuses to apply a createNamedRange as a
	// suggestion. It adds and removes no character.
	p2.GrantInPlace(r.id)
	p2.AllowMarker(prelude.MarkerName, res.Start, res.End)

	// The revision the marker batch is sent against, and it is phase 1's own
	// answer rather than the read a moment ago.
	//
	// This is the one place the run stops chaining a revision through, and
	// unchained it opens the window requiredRevisionId exists to close.
	// Somebody editing between phase 1's last answer and the fresh read above
	// is carried by every batch that follows: the marker requires the revision
	// their edit made, so it lands, the styling chains off it, and phase 2
	// then rewrites their paragraph by direct edit at LevelInPlace. M7b has no
	// such window, because every batch requires the revision the one before it
	// ended on, and the phase boundary must not be the gap in that chain.
	//
	// The refusal is left inside Docs rather than made here as a comparison of
	// two strings. That a batch answer's revision is one a later write accepts
	// is measured, in every multi-batch run M7b has made; that it is spelled
	// the way documents.get spells it is not, so a run refused on a comparison
	// gdoc made itself could cry wolf on every document. Sent this revision,
	// Docs takes the marker when the document has not moved and refuses it as
	// stale when it has, which is the failure that already reports itself.
	//
	// RevisionUnconfirmed is the path this cannot cover: phase 1's last batch
	// named no revision, so what Applied carries is the one that batch was
	// sent against and gdoc's own prelude has moved the document past it.
	// There the fresh read is the only revision in hand, and a third party's
	// edit inside that window is indistinguishable from gdoc's own.
	markerRevision := fresh.RevisionID
	if !proposed.RevisionUnconfirmed {
		markerRevision = proposed.RevisionID
	}

	// Its own batch, ahead of the styling. A styling batch that does not land
	// still leaves a prelude Nail can accept, and a marked one is a prelude the
	// next run can find; folded into the styling it would be lost with it.
	marked, markErr := restyle.Apply(ctx, r2.session, r.id,
		[]map[string]any{prelude.MarkerRequest(res.Start, res.End)}, markerRevision)
	warns = append(warns, marked.Warnings...)
	pre.MarkerCreated = marked.Batches > 0
	// MaybeApplied is read beside the count, and it is the rule every other
	// writer here holds: a write whose answer could not be read is not a write
	// that never happened. Reported from the count alone the run said the
	// marker did not land one line under a warning saying the batch may be in
	// the document, which is the contradiction restyle's own leftBehind exists
	// to avoid.
	pre.MarkerMaybeCreated = marked.MaybeApplied
	data.RevisionID = marked.RevisionID
	if markErr != nil {
		// The prelude is in the document and nothing records that gdoc put it
		// there, so the run stops rather than styling on top of it. A second
		// run over an unmarked prelude proposes a second cover in front of the
		// first, and saying so here is what stops somebody discovering it then.
		//
		// r2's warnings go on the envelope like they do on the two returns
		// above: this is phase 2's session, and dropping what its policy and
		// its own two requests had to say would be the one path in this
		// function that reports half the run.
		//
		// What the error says about the marker is what the run can honestly
		// say. Docs accepted a batch whose answer could not be read means the
		// marker may be there, and calling that a marker that did not land
		// would contradict the warning standing beside it.
		what := "the marker over it did not land"
		if marked.MaybeApplied {
			what = "the marker over it was accepted by Docs and its answer could not be read, so it may or may not be there"
		}
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(append(append(warns, r2.warnings()...),
			"the prelude was proposed and could not be marked, so gdoc may have no record of having written it: "+
				"accept or reject the prelude in the browser before running this again, or the next run proposes a second one in front of it")...),
			Error: fmt.Sprintf("the prelude was proposed and %s, so nothing was styled: %v", what, markErr)}
	}

	// The revision the styling phase is sent against, and it has to be one Docs
	// named. A marker batch is the last batch of its own run, so an answer that
	// carried no revision id leaves Apply reporting the revision that batch was
	// sent against, which the marker has already moved the document past. Sent
	// on, the first styling batch is refused as stale and isStale reports that
	// as somebody having edited the document after the survey: a third party
	// named for a revision gdoc itself moved. So the read is made here instead,
	// on the rare path that needs it.
	revision := marked.RevisionID
	if marked.RevisionUnconfirmed {
		read, err := restyle.RevisionOf(ctx, r2.session, r.id)
		if err != nil {
			return emit.Result{OK: false, Data: data, Warnings: r.warnings(append(warns, r2.warnings()...)...), Error: fmt.Sprintf(
				"the prelude was proposed and marked, its answer named no revision id, and the document could not be read for one, so nothing was styled: %v", err)}
		}
		revision = read
		data.RevisionID = read
		// And the warning that said the revision reported is not the one the
		// document carries goes with it. It was true when Apply said it and
		// this read is what made it false, so leaving it on the envelope beside
		// a revision Docs named is the contradiction leftBehind's own comment
		// exists to avoid.
		warns = dropWarning(warns, restyle.RevisionUnconfirmedWarning)
	}

	// The author's own text as it stood before a word of the prelude was
	// proposed, read off the document phase 1 was computed from. It is the
	// before side of the one check this milestone's own claim rests on, and it
	// is taken here because this is the last moment it can be: every read after
	// this one carries the prelude.
	skipStart, skipEnd := res.Occupies()
	// Phase 1's own session warnings go with them, and this is the only place
	// they can. styleDocument builds the envelope from the reach it is handed,
	// which is phase 2's, so anything phase 1's session had to say reaches
	// nobody unless it is carried in here: today that is gapi's receipt for a
	// token refreshed and saved, and phase 2 opens after it and warns about
	// nothing. Every failing return above carries both sessions, and the run
	// that worked must not be the one path that reports half of it.
	return styleDocument(ctx, r2, saved, cfg, fresh.Tabs[0], revision,
		&phaseOne{
			span:   restyle.Span{Start: res.Start, End: res.End},
			skip:   restyle.Span{Start: skipStart, End: skipEnd},
			before: prelude.AuthorText(d),
			data:   pre,
		}, data, append(r.warnings(), warns...)...)
}

// dropWarning is one sentence taken back out of a list, by value.
//
// It is here rather than in internal/restyle because the package that says a
// thing is not the one that can know it stopped being true: Apply is right to
// warn that the revision it reports is not the one the document carries, and
// only the caller that goes and reads knows the sentence no longer holds.
func dropWarning(warns []string, drop string) []string {
	out := warns[:0:0]
	for _, w := range warns {
		if w != drop {
			out = append(out, w)
		}
	}
	return out
}

// styleDocument is phase 2, and on a run with no --fields it is the whole
// command: the plan, the batches, and the read-back over both phases.
//
// one is phase 1, nil on a run that proposed no prelude.
func styleDocument(ctx context.Context, r *reach, saved restyle.Report, cfg *house.Config,
	tab docs.Tab, revisionID string, one *phaseOne, data restyleData, warns ...string) emit.Result {
	var skip *restyle.Span
	if one != nil {
		skip = &one.skip
	}
	plan := restyle.TabRequestsExcept(tab, cfg, skip)
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
		Skipped:    plan.Skipped,
		Unstyled:   plan.Unstyled,
	}

	applied, applyErr := restyle.Apply(ctx, r.session, r.id, requests, revisionID)
	warns = append(warns, applied.Warnings...)
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
	//
	// MaybeApplied is read beside the count for that reason. A batch Docs
	// accepted whose answer could not be read is not a batch that never
	// happened: it may be in the document, which under this grant means the
	// document may have been directly edited, and that is the run the
	// preservation facts are needed for most.
	//
	// A batch that failed on the request itself is not in the gate, and that is
	// a decision rather than an oversight. A guard refusal, a 4xx, a 5xx and a
	// dropped connection arrive here as one thing, so opening the gate to them
	// would pay three reads on every refusal to answer the one case in four
	// where the batch may have landed. Apply's own warning says so instead, and
	// the caller reads the document.
	//
	// The landing half is given the requests that reached Docs rather than the
	// whole plan. landing.go asks whether the first request of each kind is in
	// the document, and a request from a batch that never left the machine would
	// be reported as a style that did not land on a run that never tried to
	// write it.
	//
	// A run that proposed a prelude always reads back, whatever phase 2 did.
	// Phase 1 wrote, so there is something in the document to ask about: every
	// piece of the prelude either carries a suggestion id or does not, and that
	// is the one question this milestone exists to answer.
	if applied.Batches > 0 || applied.MaybeApplied || one != nil {
		rb, pre, notes := readBack(ctx, r, saved, reached(requests, applied), plan, one)
		data.ReadBack = rb
		if one != nil {
			one.data.ReadBack = pre
		}
		// Both phases, and fewer than all of the checks is verified: false with
		// the route named in the warnings. A read the run could not make leaves
		// both halves nil, which is not verified either: nothing then says what
		// is in the document.
		data.Verified = rb != nil && rb.Verified && (one == nil || (pre != nil && pre.Verified))
		warns = append(warns, notes...)
	}
	if applyErr != nil {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...), Error: applyErr.Error()}
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(warns...)}
}

// reached is the plan split by what Docs answered for, which is what the
// read-back is asked about.
//
// Apply sends the batches in order and counts the requests inside the ones it
// confirmed, so those are the first Requests entries of the plan, and the
// requests of a batch Docs accepted whose answer could not be read are the
// MaybeRequests standing right behind them. Both left the machine. What is
// behind them never did, and asking about those would name a style as not landed
// on a run that never tried to write it.
func reached(requests []map[string]any, applied restyle.Applied) restyle.Sent {
	end := min(applied.Requests, len(requests))
	stop := min(end+applied.MaybeRequests, len(requests))
	return restyle.Sent{Confirmed: requests[:end], Unconfirmed: requests[end:stop]}
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
// A read that failed is a warning and no read-back at all, and that is both
// halves of it. Reporting a preservation half made from a listing that never
// arrived would name every thread in the survey as gone, which is the one
// warning that must never cry wolf, and the prelude half is read out of the
// same Docs answer the landing half is.
//
// The prelude half is phase 1's, and it is asked here because this is the one
// read that carries the finished document: whether every piece of what was
// proposed is a suggestion, whether the marker is over it, and whether the
// author's own text is what it was before any of it. one is nil on a run that
// proposed no prelude, and so is the answer.
func readBack(ctx context.Context, r *reach, before restyle.Report, sent restyle.Sent,
	plan restyle.Plan, one *phaseOne) (*restyle.ReadBack, *prelude.Check, []string) {
	raws, err := comments.Fetch(ctx, r.session, r.id, nil)
	if err != nil {
		return nil, nil, []string{fmt.Sprintf(
			"the comment listing could not be read back, so nothing here says what survived the restyle: %v", err)}
	}
	var raw json.RawMessage
	if err := r.session.GetJSON(ctx, docs.URL(r.id), &raw); err != nil {
		return nil, nil, []string{fmt.Sprintf(
			"the document could not be read back, so nothing here says what survived the restyle or whether the style landed: %v", err)}
	}
	d, err := docs.Parse(raw)
	if err != nil {
		return nil, nil, []string{fmt.Sprintf(
			"the document was read back and did not decode, so nothing here says what survived the restyle: %v", err)}
	}
	f, exportErr := exportFile(ctx, r)
	rb, notes := restyle.Verify(before, restyle.Input{
		Document:  d,
		Comments:  raws,
		Export:    f,
		ExportErr: exportErr,
	}, raw, sent, plan)
	if one == nil {
		return &rb, nil, notes
	}
	check, preNotes := prelude.Verify(one.before, d, one.span.Start, one.span.End)
	return &rb, &check, append(notes, preNotes...)
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
	// The version, and it is checked before any of the facts are read. A strict decoder
	// refuses a key it does not know and says nothing about a key that is
	// absent, so a survey printed by an older gdoc reads here as a survey whose
	// missing evidence is evidence of nothing lost: the suggestion ids arrived
	// at M7b, and without them the read-back reports every suggestion that was
	// destroyed as one that was never pending. That is the one false fact in the
	// file this command trusts most, and a version is what refuses it by name.
	switch saved.Data.Schema {
	case restyle.Schema:
	case 0:
		return nil, fmt.Errorf(
			"%s carries no survey schema, so it was printed by a gdoc older than this one and the evidence this recheck rests on may not be in it: "+
				"take the survey again with `gdoc restyle <url> --dry-run`", path)
	default:
		return nil, fmt.Errorf(
			"%s states survey schema %d and this gdoc reads %d, so what its fields mean is not decided here: "+
				"take the survey again with `gdoc restyle <url> --dry-run`", path, saved.Data.Schema, restyle.Schema)
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
