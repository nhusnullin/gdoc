// The four write commands, and the arguments they take.
//
// They are the read commands with one thing added: something leaves the
// machine. So the shape is the same, and two rules are added to it. Every write
// into a document is a suggestion, which the guard holds by refusing a
// batchUpdate on a handed-in id whose body does not say SUGGEST. And every
// write is read back through another route before the envelope calls it
// verified, because a status code is Google agreeing with itself.
//
// Nothing here decides what to write. The body of a reply, the words of a
// proposal and the reason for it arrive written, in a file the skill wrote.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gdoc/internal/docs"
	"gdoc/internal/emit"
	"gdoc/internal/frontmatter"
	"gdoc/internal/guard"
	"gdoc/internal/probe"
	"gdoc/internal/propose"
	"gdoc/internal/reply"
	"gdoc/internal/withdraw"
)

// probeData is what `gdoc probe` prints, and what rides inside a propose run.
// The report's own warnings are hoisted onto the envelope instead, where every
// other warning in this binary lives.
type probeData struct {
	Enrolled        bool     `json:"enrolled"`
	ProbeDocumentID string   `json:"probe_document_id,omitempty"`
	Trashed         bool     `json:"trashed"`
	SuggestionIDs   []string `json:"suggestion_ids,omitempty"`
}

func probeReport(r probe.Report) probeData {
	return probeData{
		Enrolled:        r.Enrolled,
		ProbeDocumentID: r.ProbeDocumentID,
		Trashed:         r.Trashed,
		SuggestionIDs:   r.SuggestionIDs,
	}
}

// openFolder is open's twin for a command whose reach is a folder to create in
// rather than a document to read. The policy has one door open, and it is the
// create: the probe document is learned from the answer, never handed in.
func openFolder(target string) (string, session, error) {
	id, err := folderID(target)
	if err != nil {
		return "", nil, err
	}
	p := guard.NewPolicy()
	p.AllowCreateIn(id)
	s, err := openSession(p)
	if err != nil {
		return "", nil, err
	}
	return id, s, nil
}

// required reads a flag the command cannot run without, naming it when it is
// missing. A command that guessed a default here would write somewhere nobody
// pointed it at.
func required(a *args, name string) (string, error) {
	if !a.has(name) {
		return "", fmt.Errorf("this command needs %s", name)
	}
	return a.flags[name], nil
}

func cmdProbe(a *args) emit.Result {
	folder, err := required(a, "--folder")
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	id, s, err := openFolder(folder)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	report, err := probe.Run(context.Background(), s, id)
	warns := append(append([]string{}, s.Warnings()...), report.Warnings...)
	if err != nil {
		// The report goes out beside the error: a create that succeeded and a
		// step that then failed is exactly the run where the document id is
		// worth most, because somebody may have to go and look at it.
		return emit.Result{OK: false, Error: err.Error(), Data: probeReport(report), Warnings: warns}
	}
	return emit.Result{OK: true, Data: probeReport(report), Warnings: warns}
}

// replyData is what `gdoc reply` prints. Verified is the read-back rather than
// the status code: Drive answering 200 is Drive agreeing with itself.
type replyData struct {
	DocumentID string `json:"document_id"`
	CommentID  string `json:"comment_id"`
	ReplyID    string `json:"reply_id,omitempty"`
	Created    string `json:"created,omitempty"`
	Verified   bool   `json:"verified"`
}

func cmdReply(a *args) emit.Result {
	path, err := required(a, "--body-file")
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	commentID := a.at(1)
	if commentID == "" {
		return emit.Result{OK: false, Error: "this command needs the comment id of the thread to reply in"}
	}
	body, err := readBody(path)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	// Before the session, and so before anything reaches Drive: a body Docs
	// would render literally is a mistake in the call, not something a write
	// could fix.
	if err := reply.Check(body); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	r, err := open(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	res, err := reply.Post(context.Background(), r.session, r.id, commentID, body)
	data := replyData{
		DocumentID: r.id,
		CommentID:  commentID,
		ReplyID:    res.ReplyID,
		Created:    res.Created,
		Verified:   res.Verified,
	}
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Data: data, Warnings: r.warnings()}
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(res.Warnings...)}
}

// readBody reads the reply body out of the file the skill wrote.
//
// The trailing newline a text file ends with is not part of what was said, and
// it is dropped: the read-back compares the words Drive stored against the
// words that were sent, and a newline Drive trimmed would report a reply that
// is plainly in the thread as unverified.
func readBody(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("the reply body file could not be read: %w", err)
	}
	return strings.TrimRight(string(raw), " \t\r\n"), nil
}

// proposalReport is one proposal as the envelope carries it. Sent is the fact
// the skill reads first: a run stopped by the probe reports every proposal, and
// each one says gdoc got no answer saying it landed.
//
// That is what the field means, and it is narrower than "it never left the
// machine". A guard refusal and a 4xx never changed the document, and a 5xx or
// a dropped connection is the third case: the request was written and may have
// been applied, and gdoc cannot tell. The envelope's own error names it on that
// path, so read the error beside the flag rather than the flag alone, and read
// the document before proposing the same words again.
type proposalReport struct {
	Quoted             string         `json:"quoted"`
	Replacement        string         `json:"replacement"`
	Sent               bool           `json:"sent"`
	SuggestionIDs      []string       `json:"suggestion_ids,omitempty"`
	CommentID          string         `json:"comment_id,omitempty"`
	CommentUpdateState string         `json:"comment_update_state,omitempty"`
	Verified           bool           `json:"verified"`
	Checks             propose.Checks `json:"checks"`
}

// proposeData is what `gdoc propose` prints.
type proposeData struct {
	DocumentID   string           `json:"document_id"`
	Tabs         int              `json:"tabs"`
	MultiTab     bool             `json:"multi_tab"`
	Probe        *probeData       `json:"probe,omitempty"`
	Proposals    []proposalReport `json:"proposals"`
	FilesChanged []string         `json:"files_changed,omitempty"`
}

func cmdPropose(a *args) emit.Result {
	from, err := required(a, "--from")
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	folder, err := required(a, "--folder")
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	proposals, err := readProposals(from)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	docID, err := documentID(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	probeFolder, err := folderID(folder)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	// The note is checked before the session is opened. A note paired with
	// another document would be handed this document's provenance, and
	// provenance is the permission withdraw reads.
	note, err := pairedNote(a, docID)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}

	// One policy, two doors: the document at LevelSuggest, and the probe's
	// folder as the one place a create may land. The probe document itself is
	// learned from the create the guard carried.
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)
	p.AllowCreateIn(probeFolder)
	s, err := openSession(p)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	r := &reach{id: docID, session: s}
	return runPropose(r, probeFolder, proposals, note)
}

// runPropose is the run itself: read, probe, then one proposal at a time.
func runPropose(r *reach, probeFolder string, proposals []propose.Proposal, note *notePath) emit.Result {
	ctx := context.Background()
	var warns []string

	d, err := docs.Fetch(ctx, r.session, r.id)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}
	data := proposeData{
		DocumentID: r.id,
		Tabs:       len(d.Tabs),
		MultiTab:   d.MultiTab(),
		Proposals:  notSent(proposals),
	}
	if d.MultiTab() {
		// A proposal names one range, and a range means nothing without saying
		// which tab it is in. Nothing is sent, the probe included.
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(),
			Error: fmt.Sprintf("the document has %d tabs, and a proposal is written into a document with one", len(d.Tabs))}
	}

	report, err := probe.Run(ctx, r.session, probeFolder)
	shown := probeReport(report)
	data.Probe = &shown
	warns = append(warns, report.Warnings...)
	if err != nil {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...),
			Error: fmt.Sprintf("the capability probe could not be run, so whether SUGGEST is honoured today is unknown and nothing was proposed: %v", err)}
	}
	if !report.Enrolled {
		return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...),
			Error: "the probe document came back with the suggested word as plain text, so SUGGEST is not honoured for this project today and nothing was proposed"}
	}

	// One proposal at a time, each after its own fresh read, because the first
	// one moves the ground under the second. The run stops at the first one that
	// cannot be sent, and the report still carries one entry per proposal in the
	// file: each entry answers `sent` for itself, so a stop in the middle names
	// what landed and what never left rather than shortening the list.
	results := make([]propose.Result, 0, len(proposals))
	for i, one := range proposals {
		res, err := propose.Apply(ctx, r.session, r.id, one)
		if err != nil {
			data.FilesChanged, warns = record(note, results, warns)
			return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...), Error: err.Error()}
		}
		results = append(results, res)
		data.Proposals[i] = sent(res)
		warns = append(warns, about(one.Quoted, res.Warnings)...)
	}
	data.FilesChanged, warns = record(note, results, warns)
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(warns...)}
}

// notSent is every proposal as it stands before anything has left the machine.
func notSent(proposals []propose.Proposal) []proposalReport {
	out := make([]proposalReport, 0, len(proposals))
	for _, p := range proposals {
		out = append(out, proposalReport{Quoted: p.Quoted, Replacement: p.Replacement})
	}
	return out
}

// sent is one applied proposal as the envelope carries it.
func sent(r propose.Result) proposalReport {
	return proposalReport{
		Quoted:             r.Quoted,
		Replacement:        r.Replacement,
		Sent:               true,
		SuggestionIDs:      r.SuggestionIDs,
		CommentID:          r.CommentID,
		CommentUpdateState: r.CommentUpdateState,
		Verified:           r.Verified,
		Checks:             r.Checks,
	}
}

// about names which proposal a warning belongs to. The envelope carries one
// list, and a run with two proposals in it would otherwise report a route that
// did not hold without saying which change it was about.
func about(quoted string, warns []string) []string {
	out := make([]string, 0, len(warns))
	for _, w := range warns {
		out = append(out, fmt.Sprintf("proposal %q: %s", quoted, w))
	}
	return out
}

// record writes the provenance of everything that landed, and says which file
// changed. A proposal whose batch was accepted is remembered whether or not the
// read-backs held: it is in the document either way, and a proposal gdoc has
// forgotten is one it will refuse to withdraw. What cannot be remembered at all,
// because an id an entry needs is not in hand, is named in a warning rather than
// dropped: that is the same loss, and silence about it reads as a verification
// gap instead of a permission thrown away. missingID says which id and which
// route it would have come from.
//
// The note is read again here rather than reused from the pairing check. Between
// the two sits the probe, a read and a write per proposal and three read-backs
// each, which is seconds to tens of seconds; these notes live in a synced vault,
// and writing the bytes this run started with would throw away whatever landed
// in that window. A note that has stopped naming this document is left alone and
// said so, because the proposals belong to a pairing it no longer records.
func record(note *notePath, results []propose.Result, warns []string) ([]string, []string) {
	if note == nil || len(results) == 0 {
		return nil, warns
	}
	src, _, err := freshNote(note)
	if err != nil {
		return nil, append(warns, fmt.Sprintf(
			"the proposals were written into the document and %s could not be recorded against them, so gdoc cannot withdraw them later: %v", note.path, err))
	}
	out, missed, err := propose.Record(src, results, now())
	if err != nil {
		return nil, append(warns, fmt.Sprintf(
			"the proposals were written into the document and %s could not be updated to record them, so gdoc cannot withdraw them later: %v", note.path, err))
	}
	for _, m := range missed {
		// The change is in the document and there is no id to remember it by, so
		// withdraw will refuse it for ever. That is the one loss this file exists
		// to prevent, and it is said out loud rather than left to the reader of a
		// verification warning to work out.
		warns = append(warns, fmt.Sprintf(
			"the proposal %q was written into the document and %s does not record it, because %s; gdoc cannot withdraw it later",
			m.Quoted, note.path, missingID(m)))
	}
	if len(missed) == len(results) {
		// Nothing was added, so nothing is written and the note is not named as
		// changed: a file listed in files_changed that gdoc did not touch is a
		// false fact in the field a caller reads first.
		return nil, warns
	}
	if err := writeFile(note.path, out); err != nil {
		return nil, append(warns, fmt.Sprintf(
			"the proposals were written into the document and %s could not be written, so gdoc cannot withdraw them later: %v", note.path, err))
	}
	return []string{note.path}, warns
}

// missingID names which of the two ids an entry needs is not in hand, and it
// names the route each one comes from rather than blaming the write for both.
// The comment id is the batch's own answer; the suggestion id is read out of the
// inline read-back afterwards, so a read-back that failed leaves an entry that
// cannot be written down over a write that came back perfectly well. Saying the
// write came back without an id there sends somebody to look at Docs while the
// envelope's other warning is already saying the re-read is what broke.
func missingID(r propose.Result) string {
	switch {
	case len(r.SuggestionIDs) == 0 && r.CommentID == "":
		return "the write came back without a comment id and the read-back could not confirm a suggestion id"
	case r.CommentID == "":
		return "the write came back without a comment id"
	default:
		return "the read-back could not confirm a suggestion id"
	}
}

// freshNote reads the note again and hands back its new bytes and the block in
// them. A file that changed under the run is fine, and its new bytes are what
// the block is written into.
//
// Four things are refused rather than written, and each one drops the
// provenance the caller was about to record: a file that cannot be read again,
// one whose front matter no longer parses, one whose gdoc: block has gone, and
// one that now names another document. Every caller warns with the reason, so
// which of the four it was is on the envelope.
func freshNote(note *notePath) ([]byte, *frontmatter.Block, error) {
	src, err := os.ReadFile(note.path)
	if err != nil {
		return nil, nil, fmt.Errorf("the markdown file could not be read again: %w", err)
	}
	block, err := frontmatter.Read(src)
	if err != nil {
		return nil, nil, err
	}
	if block == nil {
		return nil, nil, fmt.Errorf("%s no longer carries a gdoc: front matter block", note.path)
	}
	if block.DocumentID != note.block.DocumentID {
		return nil, nil, fmt.Errorf("%s is now paired with document %s, and this run was of %s", note.path, block.DocumentID, note.block.DocumentID)
	}
	return src, block, nil
}

// notePath is the paired note: where it is, and the block the pairing check
// read. The bytes that check ran over are deliberately not kept. Both writers
// read the file again through freshNote, for the reason record gives, so a copy
// held here would only be the stale bytes somebody later wrote back.
type notePath struct {
	path  string
	block *frontmatter.Block
}

// pairedNote reads the note --md named and checks it belongs to this document.
// A command given no --md has no note, which is not an error: propose without
// one still writes the suggestion, and only the memory of it is lost.
func pairedNote(a *args, docID string) (*notePath, error) {
	if !a.has("--md") {
		return nil, nil
	}
	return readNote(a.flags["--md"], docID)
}

func readNote(path, docID string) (*notePath, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("the markdown file could not be read: %w", err)
	}
	block, err := frontmatter.Read(src)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("%s carries no gdoc: front matter, so it is not paired with a document", path)
	}
	if block.DocumentID != docID {
		return nil, fmt.Errorf("%s is paired with document %s, and this run is of %s", path, block.DocumentID, docID)
	}
	return &notePath{path: path, block: block}, nil
}

// readProposals reads the list the skill wrote. An empty list is refused rather
// than run: a probe document would be created and trashed for a run with
// nothing to propose.
//
// The read is strict, for the reason parseArgsN refuses an unknown flag and
// frontmatter reads with yaml.Strict(). A misspelled `quoted`, `replacement` or
// `why` is caught a few lines down, because Check refuses their empty values;
// `assignee` is optional, so a dropped one landed a comment with nobody
// assigned, reported verified: true, and warned about nothing.
func readProposals(path string) ([]propose.Proposal, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("the proposals file could not be read: %w", err)
	}
	var out []propose.Proposal
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("%s is not a list of proposals: %w", path, err)
	}
	// Decode stops at the end of the first value, where Unmarshal refused a
	// file with anything behind it. A second list after the first is a file
	// somebody edited wrongly, and running the first half of it silently is the
	// same mistake as dropping a key.
	if dec.More() {
		return nil, fmt.Errorf("%s carries more than one list of proposals", path)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s carries no proposals, so there is nothing to write", path)
	}
	// Every proposal is checked here, before the probe and before the first
	// write, for the reason the empty list is refused here: all of them are in
	// hand, and a third entry refused after the first two have landed is a run
	// that half happened in somebody's document.
	for i, p := range out {
		if err := p.Check(); err != nil {
			return nil, fmt.Errorf("%s proposals[%d]: %w", path, i, err)
		}
	}
	return out, nil
}

// withdrawData is what `gdoc withdraw` prints.
type withdrawData struct {
	DocumentID            string   `json:"document_id"`
	SuggestionID          string   `json:"suggestion_id"`
	RejectedSuggestionIDs []string `json:"rejected_suggestion_ids,omitempty"`
	Verified              bool     `json:"verified"`
	FilesChanged          []string `json:"files_changed,omitempty"`
}

func cmdWithdraw(a *args) emit.Result {
	suggestionID := a.at(1)
	if suggestionID == "" {
		return emit.Result{OK: false, Error: "this command needs the id of the suggestion to withdraw"}
	}
	// The note is not optional here, and the message says why: it is the only
	// record of which suggestions are gdoc's own, and gdoc retracts nothing
	// else.
	if !a.has("--md") {
		return emit.Result{OK: false,
			Error: "this command needs --md: the note's proposals are the only record of which suggestions gdoc wrote, and gdoc withdraws only those"}
	}
	docID, err := documentID(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	note, err := readNote(a.flags["--md"], docID)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if !withdraw.Mine(note.block, suggestionID) {
		return emit.Result{OK: false, Error: fmt.Sprintf(
			"%s does not record %q as one of gdoc's own proposals, and gdoc withdraws only what it proposed", note.path, suggestionID)}
	}
	// The grant is the note's answer handed to the guard: Mine said above that
	// this id is gdoc's own, and AllowReject is how the guard learns it for
	// this run. The guard refuses a rejectSuggestion naming any other id, so
	// the permission read from the note is also the permission on the wire.
	r, err := open(a.target(), func(p *guard.Policy) { p.AllowReject(suggestionID) })
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	res, err := withdraw.Run(context.Background(), r.session, r.id, suggestionID, note.block)
	data := withdrawData{
		DocumentID:            r.id,
		SuggestionID:          suggestionID,
		RejectedSuggestionIDs: res.RejectedSuggestionIDs,
		Verified:              res.Verified,
	}
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Data: data, Warnings: r.warnings()}
	}
	warns := res.Warnings
	if res.Verified {
		// The entry leaves the note only when the suggestion has provably left
		// the document. Forgetting it while it is still pending would leave
		// gdoc refusing to withdraw its own work.
		changed, err := forget(note, suggestionID)
		if err != nil {
			warns = append(warns, err.Error())
		} else {
			data.FilesChanged = changed
		}
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(warns...)}
}

// forget takes the withdrawn proposal out of the note.
//
// The note is read again first, for the reason record gives: between the
// pairing check and here sit two whole-document reads and a batchUpdate, which
// is seconds on the network, and these notes live in a synced vault. The block
// is re-parsed from those bytes too, so a proposal recorded into the gdoc: key
// in that window survives; writing the block this run started with would take
// the author's prose across and still drop that entry.
func forget(note *notePath, suggestionID string) ([]string, error) {
	src, block, err := freshNote(note)
	if err != nil {
		return nil, fmt.Errorf("the suggestion was withdrawn and %s still records it: %v", note.path, err)
	}
	out, err := frontmatter.Write(src, withdraw.Forget(block, suggestionID))
	if err != nil {
		return nil, fmt.Errorf("the suggestion was withdrawn and %s still records it: %v", note.path, err)
	}
	if err := writeFile(note.path, out); err != nil {
		return nil, fmt.Errorf("the suggestion was withdrawn and %s could not be written, so it still records it: %v", note.path, err)
	}
	return []string{note.path}, nil
}
