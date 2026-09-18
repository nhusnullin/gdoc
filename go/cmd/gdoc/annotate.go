// The comment on quoted words, and the two ways a caller names one.
//
// `gdoc annotate <url> --from annotations.json` is the file form, which is what
// a review session uses: it read the document, decided in the hub, and wrote
// down the words and the reason for each comment it wants left. `gdoc annotate
// <url> --quote "..." --body-file why.txt` is the same thing for one comment
// typed by hand.
//
// It takes no folder and no note, and neither is an omission.
//
// No folder, because there is nothing to probe. propose creates a throwaway
// document every run to ask whether Docs honours SUGGEST today, since a SUGGEST
// that is quietly ignored turns a proposal into a direct edit of somebody's
// prose. The batch this command sends holds one insertComment and nothing else,
// and no insertComment can move a character whatever the write mode does, so
// there is no question for a probe to answer and no reason to litter a folder
// asking it.
//
// No note, because there is nothing to take back. The note exists so withdraw
// can recognise gdoc's own pending suggestions later, and a comment is not a
// suggestion: it is in the thread, signed with the robot, and a person deletes
// it in the browser in one gesture. Recording it would be provenance for a
// permission nothing uses.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gdoc/internal/annotate"
	"gdoc/internal/emit"
)

// annotationReport is one annotation as the envelope carries it. Sent is the
// fact the skill reads first, and it means the same narrow thing propose's does:
// gdoc got no answer saying the comment landed. A guard refusal and a 4xx never
// touched the document, and a dropped connection is the third case, where the
// request was written and gdoc cannot tell. The envelope's own error names
// which, so read it beside the flag.
type annotationReport struct {
	Quoted             string          `json:"quoted"`
	Sent               bool            `json:"sent"`
	CommentID          string          `json:"comment_id,omitempty"`
	CommentUpdateState string          `json:"comment_update_state,omitempty"`
	Verified           bool            `json:"verified"`
	Checks             annotate.Checks `json:"checks"`
}

// annotateData is what `gdoc annotate` prints.
type annotateData struct {
	DocumentID  string             `json:"document_id"`
	Annotations []annotationReport `json:"annotations"`
}

func cmdAnnotate(a *args) emit.Result {
	list, err := annotationsFor(a)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	r, err := open(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	return runAnnotate(r, list)
}

// annotationsFor turns the flags into the list this run will place, and refuses
// every shape that is not one of the two calls.
//
// Everything here happens before a session is opened, so a call gdoc does not
// understand costs no request and reads nobody's document. The entries are
// checked as a batch as well: a third entry refused after the first two have
// landed is a run that half happened in somebody's document.
func annotationsFor(a *args) ([]annotate.Annotation, error) {
	switch {
	case a.has("--from") && a.has("--quote"):
		return nil, fmt.Errorf("annotate takes --quote with --body-file, or --from <file>, and this run gave both: " +
			"one names a single comment on the command line and the other names a file of them")
	// --body-file beside --from is the same mistake as --quote beside it, and
	// it has to be named here: the --body-file arm below would otherwise tell
	// this caller to add --quote, which is the one flag that makes the run
	// worse. A refusal that names the wrong mistake costs two attempts.
	case a.has("--from") && a.has("--body-file"):
		return nil, fmt.Errorf("--from carries the reason for every comment in the file, and --body-file belongs to --quote: " +
			"drop --body-file to leave the comments the file names")
	case a.has("--quote") && !a.has("--body-file"):
		return nil, fmt.Errorf("--quote names the words to comment on, and the reason goes in --body-file <file>: " +
			"a comment that says nothing is one Nail has to guess at")
	case a.has("--body-file") && !a.has("--quote"):
		return nil, fmt.Errorf("--body-file holds the reason for a comment, and --quote <text> says which words it goes on")
	case a.has("--from"):
		return readAnnotations(a.flags["--from"])
	case a.has("--quote"):
		why, err := readWhy(a.flags["--body-file"])
		if err != nil {
			return nil, err
		}
		one := annotate.Annotation{Quoted: a.flags["--quote"], Why: why}
		if err := one.Check(); err != nil {
			return nil, err
		}
		return []annotate.Annotation{one}, nil
	}
	return nil, fmt.Errorf("annotate needs --quote <text> with --body-file <file> to leave one comment, " +
		"or --from <file> to leave the comments a file names")
}

// readWhy reads the reason out of the file the caller wrote.
//
// The trailing newline a text file ends with is not part of what was said, and
// it is dropped, for the reason reply's own reader drops it: the read-backs
// compare the words Drive and the export carry against the words that were
// sent, and a newline the thread trimmed would report a comment that is plainly
// there as unverified.
func readWhy(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("the annotation body file could not be read: %w", err)
	}
	return strings.TrimRight(string(raw), " \t\r\n"), nil
}

// readAnnotations reads the list the skill wrote. An empty list is refused
// rather than run: a session would be opened and somebody's document read for a
// run that writes nothing.
//
// The read is strict, for the reason parseArgsN refuses an unknown flag. A
// misspelled `quoted` or `why` is caught a few lines down, because Check refuses
// their empty values; `assignee` is optional, so a dropped one lands a comment
// assigned to nobody, reported verified and warned about by nothing.
func readAnnotations(path string) ([]annotate.Annotation, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("the annotations file could not be read: %w", err)
	}
	var out []annotate.Annotation
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("%s is not a list of annotations: %w", path, err)
	}
	// Decode stops at the end of the first value, where Unmarshal refused a file
	// with anything behind it. A second list after the first is a file somebody
	// edited wrongly, and writing the first half of it silently is the same
	// mistake as dropping a key.
	if dec.More() {
		return nil, fmt.Errorf("%s carries more than one list of annotations", path)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s carries no annotations, so there is nothing to write", path)
	}
	for i, one := range out {
		if err := one.Check(); err != nil {
			return nil, fmt.Errorf("%s annotations[%d]: %w", path, i, err)
		}
	}
	return out, nil
}

// runAnnotate is the run itself: one annotation at a time, each with its own
// read, and the report carrying one entry per annotation whatever happens.
//
// The run stops at the first one that cannot be sent, and the list is not
// shortened: each entry answers `sent` for itself, so a stop in the middle says
// what landed and what never left rather than leaving the skill to match the
// envelope back against the file it wrote.
func runAnnotate(r *reach, list []annotate.Annotation) emit.Result {
	ctx := context.Background()
	var warns []string
	data := annotateData{DocumentID: r.id, Annotations: notPlaced(list)}

	for i, one := range list {
		res, err := annotate.Apply(ctx, r.session, r.id, one)
		if err != nil {
			return emit.Result{OK: false, Data: data, Warnings: r.warnings(warns...), Error: err.Error()}
		}
		data.Annotations[i] = placed(res)
		warns = append(warns, aboutAnnotation(one.Quoted, res.Warnings)...)
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(warns...)}
}

// notPlaced is every annotation as it stands before anything has left the
// machine.
func notPlaced(list []annotate.Annotation) []annotationReport {
	out := make([]annotationReport, 0, len(list))
	for _, one := range list {
		out = append(out, annotationReport{Quoted: one.Quoted})
	}
	return out
}

// placed is one comment that landed, as the envelope carries it.
func placed(r annotate.Result) annotationReport {
	return annotationReport{
		Quoted:             r.Quoted,
		Sent:               true,
		CommentID:          r.CommentID,
		CommentUpdateState: r.CommentUpdateState,
		Verified:           r.Verified,
		Checks:             r.Checks,
	}
}

// aboutAnnotation names which comment a warning belongs to. The envelope carries
// one list of warnings, and a run of two annotations would otherwise report a
// route that did not hold without saying which comment it was about. It is
// propose's `about` on this command's own word, rather than a shared helper,
// because the word is the whole of what it does.
func aboutAnnotation(quoted string, warns []string) []string {
	out := make([]string, 0, len(warns))
	for _, w := range warns {
		out = append(out, fmt.Sprintf("annotation %q: %s", quoted, w))
	}
	return out
}
