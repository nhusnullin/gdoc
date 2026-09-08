// Package probe asks Google, on a document gdoc made for the purpose, whether
// a suggestion written today is honoured as a suggestion.
//
// The reason it exists is in docs/v2/BLOCKED-BY-API.md. `writeMode` is absent
// from the public Docs discovery document, and one morning a batchUpdate
// carrying SUGGEST answered 200 and made a direct edit instead. The guard
// refuses a write on a handed-in document unless the body says SUGGEST, but the
// guard reads gdoc's own words: what the server did with them is a different
// question, and only a read-back answers it.
//
// So the probe is that read-back, made where being wrong costs nothing. It
// creates a throwaway document in the folder the command was given, writes one
// sentence into it directly, suggests one word inside that sentence, reads the
// document back and looks for the word carrying a suggestion id. Then it puts
// the document in the trash and confirms it went.
//
// Nothing here decides anything. Enrolled is a fact about what Google answered,
// and what to do about a false one is the caller's.
package probe

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/drive"
	"gdoc/internal/frontmatter"
	"gdoc/internal/suggestions"
)

// probeSentence is what the direct write puts in the empty document. It is
// ASCII on purpose: the Docs API counts indexes in UTF-16 code units, and on
// ASCII the byte offset of a word is that offset, so the index below needs no
// conversion to be right.
const probeSentence = "The supplier register is reviewed annually by the operations team.\n"

// probeWord is what the SUGGEST write inserts, and it must not already occur in
// the sentence: the read-back tells the two apart by the suggestion id a run
// carries, and a word that was there before is a run without one.
const probeWord = "critical "

// probeAnchor is the word the suggested insert goes in front of. Naming the
// place rather than the number is the same rule propose is built on: an index
// is computed from a read, never carried as a constant.
const probeAnchor = "annually"

// bodyStart is where a Docs document's body text begins. Index 0 is the
// document itself and 1 is the first character position, which is where the
// direct insert puts the sentence.
const bodyStart = 1

// mimeDocument is the Drive type of a Google Doc. A create naming it makes a
// document rather than a file with a document's name.
const mimeDocument = "application/vnd.google-apps.document"

// Report is what the run saw. Enrolled is the answer; the other three are what
// the run left behind, so a document that could not be trashed is named rather
// than lost.
type Report struct {
	Enrolled        bool     `json:"enrolled"`
	ProbeDocumentID string   `json:"probe_document_id,omitempty"`
	Trashed         bool     `json:"trashed"`
	SuggestionIDs   []string `json:"suggestion_ids,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

// Session is what this package needs of a session: a JSON read, a JSON write
// and the one PATCH Drive spells trashing as. A *gapi.Session satisfies it, and
// so does a fake in a test. Naming the interface here keeps net/http out of
// this room, which is what the boundary test asks of every package but the four
// that build requests.
type Session interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	PostJSON(ctx context.Context, rawURL string, body any, into any) error
	PatchJSON(ctx context.Context, rawURL string, body any, into any) error
}

// Run is the whole probe: create, write, suggest, read, trash, confirm.
//
// The policy it is given is AllowCreateIn(folderID) and nothing else. The probe
// document is learned from the create the guard itself carried, which is the
// second of the policy's two doors, so no handed-in document is reachable while
// this runs and the probe cannot touch the document being reviewed.
//
// The report comes back on every path, error included. A create that succeeded
// and a step that then failed is exactly the case where the id matters most: it
// names the document somebody may have to go and look at.
func Run(ctx context.Context, s Session, folderID string) (Report, error) {
	var rep Report
	id, err := create(ctx, s, folderID)
	if err != nil {
		return rep, err
	}
	rep.ProbeDocumentID = id

	measured, err := measure(ctx, s, id)
	rep.Enrolled = measured.Enrolled
	rep.SuggestionIDs = measured.SuggestionIDs

	// The trash runs whatever the measurement did. A document gdoc created and
	// then left in the folder is litter in somebody's Drive, and the step that
	// failed is not a reason to add to it.
	trashed, warns := trash(ctx, s, id)
	rep.Trashed = trashed
	rep.Warnings = warns
	return rep, err
}

// create makes the throwaway document. The name carries the moment, so a
// document that survives a failed trash says what left it there.
func create(ctx context.Context, s Session, folderID string) (string, error) {
	body := map[string]any{
		"name":     "gdoc probe " + time.Now().UTC().Format(time.RFC3339),
		"mimeType": mimeDocument,
		"parents":  []string{folderID},
	}
	var answer struct {
		ID string `json:"id"`
	}
	if err := s.PostJSON(ctx, createURL(), body, &answer); err != nil {
		// A failure raised after Drive accepted the create is not a create that
		// did not happen. The document is in the folder, and its id was in the
		// answer nothing could read, so gdoc can neither name it in
		// probe_document_id nor trash it. Saying it could not be created sends
		// somebody to look for a failure while the litter sits in their Drive.
		if sentAnyway(err) {
			return "", fmt.Errorf("Drive accepted the create and its answer could not be read, so a probe document may be in folder %q with no id for gdoc to name or trash it: %w", folderID, err)
		}
		// Everything else says the create failed and claims nothing about what
		// is in the folder. A guard refusal never left the machine and a 4xx is
		// Drive turning the create down, but a 5xx or a dropped connection is
		// neither, and gdoc cannot tell it apart from them. The wrapped error
		// carries the status, which is the only thing here that can say which.
		return "", fmt.Errorf("the create of the probe document in folder %q failed: %w", folderID, err)
	}
	if answer.ID == "" {
		// Without an id nothing further is reachable: the guard learns the new
		// file from this answer, so a create it could not read is a create with
		// no document behind it as far as the rest of the run is concerned. It
		// is not one as far as Drive is concerned, which is why this says the
		// document may be there rather than that it is not.
		return "", fmt.Errorf("the create in folder %q answered with no document id, so the probe has nothing to write to and a document may be sitting there unnamed", folderID)
	}
	return answer.ID, nil
}

// measure writes the sentence, suggests the word, and reads the document back.
// It returns what it found, so a failure part-way still leaves the caller with
// what was true before it.
func measure(ctx context.Context, s Session, id string) (Report, error) {
	var rep Report
	if err := s.PostJSON(ctx, batchURL(id), directInsert(), nil); err != nil {
		return rep, fmt.Errorf("the probe sentence could not be written: %w", err)
	}
	if err := s.PostJSON(ctx, batchURL(id), suggestInsert(), nil); err != nil {
		// A refusal here is itself an answer to a narrower question, and the
		// caller reports it: the write did not happen at all, which is not the
		// same as a write that happened as a direct edit.
		return rep, fmt.Errorf("the suggested word could not be written: %w", err)
	}
	d, err := docs.Fetch(ctx, s, id)
	if err != nil {
		return rep, fmt.Errorf("the probe document could not be read back: %w", err)
	}
	rep.SuggestionIDs = insertionIDs(d)
	rep.Enrolled = len(rep.SuggestionIDs) > 0
	return rep, nil
}

// directInsert is the sentence, written as gdoc's own edit. No writeControl:
// the document is one gdoc created, so a direct edit is what it is entitled to,
// and suggesting the sentence too would leave nothing to suggest inside.
func directInsert() map[string]any {
	return map[string]any{
		"requests": []any{
			map[string]any{"insertText": map[string]any{
				"location": map[string]any{"index": bodyStart},
				"text":     probeSentence,
			}},
		},
	}
}

// suggestInsert is the one word, written the way every document write gdoc
// makes is written. The index is computed from the sentence rather than stored:
// the sentence starts at bodyStart, so the word's place is that plus the offset
// of the anchor inside it.
func suggestInsert() map[string]any {
	return map[string]any{
		"requests": []any{
			map[string]any{"insertText": map[string]any{
				"location": map[string]any{"index": suggestIndex()},
				"text":     probeWord,
			}},
		},
		"writeControl": map[string]any{"writeMode": "SUGGEST"},
	}
}

// suggestIndex is where the word goes: in front of the anchor word, inside the
// sentence the direct write just made.
func suggestIndex() int {
	return bodyStart + strings.Index(probeSentence, probeAnchor)
}

// insertionIDs is every pending suggested insertion in the document, in reading
// order. suggestions.All is the same walk read and comments stand on, joining
// the runs Docs cuts one edit into, so the probe and the reader cannot disagree
// about what a suggestion is.
//
// Only insertions count. The probe made one insertion and no deletion, so a
// deletion id here would be somebody else's work in a document nobody else has,
// and reading it as the probe's answer would be a fact that is not true.
func insertionIDs(d *docs.Document) []string {
	var out []string
	for _, p := range suggestions.All(d) {
		if p.Kind == frontmatter.KindInsertion {
			out = append(out, p.ID)
		}
	}
	return out
}

// trash puts the document away and confirms it went. It answers with the fact
// and the warnings rather than an error: the probe's question is whether
// SUGGEST is honoured, and a document left in the folder does not change that
// answer. It does need saying out loud, which is what the warning is for.
//
// The three steps are internal/drive's, because publish takes its document back
// the same way and a second copy of them is a second chance for the two to
// disagree about whether an unconfirmed trash counts as a trash. What stays here
// is what the document is and what a failure costs.
//
// The warning says the document **may** still be in the folder, and never that
// it is, because it is one string over drive.Trash's three failures and it takes
// the weakest of the three. Only one of them knows where the file is: a
// read-back answering trashed: false is Drive saying the document is there, and
// this prefix understates it by one word. The other two do not. A failed PATCH
// covers a 5xx and a dropped connection, which Drive may have applied, and an
// unconfirmed read-back says nothing either way: a warning opening with "is
// still in the folder" in front of "Drive took the trash and could not be asked
// to confirm it" contradicts itself in one sentence and sends somebody to delete
// a document that is almost certainly already trashed. Understating the one is
// the cheaper mistake, because not knowing must never resolve to a fact, in
// either direction.
func trash(ctx context.Context, s Session, id string) (bool, []string) {
	if err := drive.Trash(ctx, s, id); err != nil {
		return false, []string{fmt.Sprintf("the probe document %q may still be in the folder: %v", id, err)}
	}
	return true, nil
}

// createURL is files.create asking for the one field the run needs. fields=id
// is not a saving: it is what the guard reads the new id out of, and a create
// whose id is not in the answer is one the policy cannot learn.
func createURL() string {
	return "https://www.googleapis.com/drive/v3/files?fields=id&supportsAllDrives=true"
}

// batchURL is the one write path the Docs API has.
func batchURL(id string) string {
	return "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
}

// sentAnyway says whether the request reached Drive in spite of the error. The
// session marks the failures raised after the server accepted a request, and
// this room asks by behaviour rather than by importing that package: naming a
// Session interface here is what keeps net/http out, and an imported sentinel
// would bring it back through the side door.
func sentAnyway(err error) bool {
	var sent interface{ Sent() bool }
	return errors.As(err, &sent) && sent.Sent()
}
