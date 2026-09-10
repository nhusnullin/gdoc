package cover

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// full is a fields file naming every one of the thirteen values, so a test
// about one of them states only what it is about by editing this.
const full = `{
  "title": "Third Party Risk",
  "alt_title": "Supplier Risk",
  "doc_type": "Policy",
  "version": "3.1",
  "date": "2026-09-08",
  "owner": "Head of Risk",
  "last_approval": "2026-06-01",
  "review_frequency": "Annual",
  "board_ratification": "2026-06-15",
  "distribution": "All staff",
  "classification": "restricted",
  "heading_numbering": false,
  "revisions": [
    {"version": "3.1", "date": "2026-09-08", "author": "N Khusnullin",
     "approved_by": "Board", "approval_date": "2026-06-15",
     "section": "4.2", "change": "Reworded the escalation path"}
  ]
}`

func readFields(t *testing.T, src string) Fields {
	t.Helper()
	f, err := ReadFields([]byte(src))
	if err != nil {
		t.Fatalf("ReadFields: %v", err)
	}
	return f
}

func TestAFieldsFileNamingEveryValueReadsThemAll(t *testing.T) {
	f := readFields(t, full)

	if f.Title != "Third Party Risk" {
		t.Errorf("title = %q, want Third Party Risk", f.Title)
	}
	if f.AltTitle != "Supplier Risk" {
		t.Errorf("alt title = %q, want Supplier Risk", f.AltTitle)
	}
	if f.DocType != "Policy" {
		t.Errorf("doc type = %q, want Policy", f.DocType)
	}
	if f.Version != "3.1" {
		t.Errorf("version = %q, want 3.1", f.Version)
	}
	if f.Owner != "Head of Risk" {
		t.Errorf("owner = %q, want Head of Risk", f.Owner)
	}
	if f.ReviewFrequency != "Annual" {
		t.Errorf("review frequency = %q, want Annual", f.ReviewFrequency)
	}
	if f.Distribution != "All staff" {
		t.Errorf("distribution = %q, want All staff", f.Distribution)
	}
	if f.Classification != "Restricted (R)" {
		t.Errorf("classification = %q, want Restricted (R)", f.Classification)
	}
	if f.HeadingNumbering {
		t.Error("heading numbering is on, and the file said false")
	}
	if len(f.Revisions) != 1 {
		t.Fatalf("revisions = %d, want 1", len(f.Revisions))
	}
	if f.Revisions[0].Change != "Reworded the escalation path" {
		t.Errorf("change = %q, want Reworded the escalation path", f.Revisions[0].Change)
	}
	if f.Revisions[0].Section != "4.2" {
		t.Errorf("section = %q, want 4.2", f.Revisions[0].Section)
	}
	// The cover title joins the title to its type, the same as a note's, so the
	// two readers give the render one shape and not two.
	if f.CoverTitle() != "Third Party Risk Policy" {
		t.Errorf("cover title = %q, want Third Party Risk Policy", f.CoverTitle())
	}
	if f.RunningHead() != "Altery - Supplier Risk Policy" {
		t.Errorf("running head = %q, want Altery - Supplier Risk Policy", f.RunningHead())
	}
}

// TestADateInTheFieldsFileReadsInUKLongForm. The note's reader rewrites an ISO
// date into the form the cover states, and a fields file naming the same date
// has to reach the cover as the same words.
func TestADateInTheFieldsFileReadsInUKLongForm(t *testing.T) {
	f := readFields(t, full)

	if f.Date != "8 September 2026" {
		t.Errorf("date = %q, want 8 September 2026", f.Date)
	}
	if f.LastApproval != "1 June 2026" {
		t.Errorf("last approval = %q, want 1 June 2026", f.LastApproval)
	}
	if f.BoardRatification != "15 June 2026" {
		t.Errorf("board ratification = %q, want 15 June 2026", f.BoardRatification)
	}
	if f.Revisions[0].ApprovalDate != "15 June 2026" {
		t.Errorf("approval date = %q, want 15 June 2026", f.Revisions[0].ApprovalDate)
	}
}

// TestAFieldsFileWithOnlyATitleTakesTheSameDefaultsANoteTakes. A value left out
// of the file must mean what it means in a note, or one document built two ways
// carries two covers.
func TestAFieldsFileWithOnlyATitleTakesTheSameDefaultsANoteTakes(t *testing.T) {
	frozen := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return frozen }
	t.Cleanup(func() { now = time.Now })

	f := readFields(t, `{"title": "A Policy"}`)

	if f.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", f.Version)
	}
	if f.Date != "September 2026" {
		t.Errorf("date = %q, want September 2026", f.Date)
	}
	if f.Classification != "Internal (I)" {
		t.Errorf("classification = %q, want Internal (I)", f.Classification)
	}
	if !f.HeadingNumbering {
		t.Error("heading numbering is off, want on when the file says nothing")
	}
	if len(f.Revisions) != 0 {
		t.Errorf("revisions = %d, want none", len(f.Revisions))
	}
}

// TestAnUnknownKeyInTheFieldsFileIsRefusedByName. The note is the author's file
// and a key gdoc does not read there belongs to somebody else's tool. This file
// is gdoc's own shape, written for this one command, so a key it does not know
// is a misspelling, and a misspelled value is a cover line that silently never
// prints.
func TestAnUnknownKeyInTheFieldsFileIsRefusedByName(t *testing.T) {
	_, err := ReadFields([]byte(`{"title": "A Policy", "tittle": "A Policy"}`))
	if err == nil {
		t.Fatal("an unknown key was accepted")
	}
	if !strings.Contains(err.Error(), "tittle") {
		t.Errorf("the refusal %q does not name the key", err)
	}
}

func TestASecondObjectBehindTheFieldsFileIsRefused(t *testing.T) {
	_, err := ReadFields([]byte(`{"title": "A Policy"}` + "\n" + `{"title": "Another Policy"}`))
	if err == nil {
		t.Fatal("a file carrying two objects was accepted, and which cover counts was decided by the reader")
	}
	if !strings.Contains(err.Error(), "more than one JSON object") {
		t.Errorf("the refusal %q does not say what is wrong with the file", err)
	}
}

func TestAFieldsFileThatIsNotJSONIsRefused(t *testing.T) {
	if _, err := ReadFields([]byte("title: A Policy\n")); err == nil {
		t.Fatal("a YAML file was read as a fields file")
	}
}

// TestAFieldsFileWithNoTitleIsRefusedAsAMissingTitle. Nothing here invents a
// title: it is the same refusal the note reader makes, so one caller reads one
// error type and the skill proposes the words to Nail.
func TestAFieldsFileWithNoTitleIsRefusedAsAMissingTitle(t *testing.T) {
	_, err := ReadFields([]byte(`{"owner": "Head of Risk"}`))
	var missing *MissingTitle
	if !errors.As(err, &missing) {
		t.Fatalf("ReadFields = %v, want a MissingTitle", err)
	}
	if !strings.Contains(err.Error(), "fields file") {
		t.Errorf("the refusal %q does not name the file the title was looked for in", err)
	}
	if missing.Candidate != "" {
		t.Errorf("candidate = %q, and a fields file carries no body and no name to draw one from", missing.Candidate)
	}
}

// TestANoteWithNoTitleStillSaysFrontMatter is the other half of the sentence
// above: the two readings of one failure name two different files, and neither
// one names the other's.
func TestANoteWithNoTitleStillSaysFrontMatter(t *testing.T) {
	_, _, err := Read(note("owner: Head of Risk", ""))
	if !strings.Contains(err.Error(), "front matter") {
		t.Errorf("the refusal %q no longer names the front matter", err)
	}
	if strings.Contains(err.Error(), "fields file") {
		t.Errorf("the refusal %q names the fields file, and a note has none", err)
	}
}

func TestAnUnknownClassificationInTheFieldsFileIsRefusedNamingWhatWasWritten(t *testing.T) {
	_, err := ReadFields([]byte(`{"title": "A Policy", "classification": "Top Secret"}`))
	if err == nil {
		t.Fatal("an unknown classification was accepted, so it would shade no row at all")
	}
	if !strings.Contains(err.Error(), "Top Secret") {
		t.Errorf("the refusal %q does not name what was written", err)
	}
}

// TestHeadingNumberingReadsABooleanAndTheWordsV1Wrote. JSON has a boolean, so
// that is what somebody writes; v1's notes say auto and none, and somebody
// copying from one of those must not silently get the other answer.
func TestHeadingNumberingReadsABooleanAndTheWordsV1Wrote(t *testing.T) {
	for _, tc := range []struct {
		written string
		want    bool
	}{
		{"true", true},
		{"false", false},
		{`"auto"`, true},
		{`"none"`, false},
		{`"yes"`, true},
		{`"off"`, false},
	} {
		f := readFields(t, `{"title": "A Policy", "heading_numbering": `+tc.written+`}`)
		if f.HeadingNumbering != tc.want {
			t.Errorf("heading_numbering %s = %v, want %v", tc.written, f.HeadingNumbering, tc.want)
		}
	}
}

func TestAHeadingNumberingValueThatIsNeitherIsRefused(t *testing.T) {
	_, err := ReadFields([]byte(`{"title": "A Policy", "heading_numbering": 3}`))
	if err == nil {
		t.Fatal("a number was read as heading numbering")
	}
	if !strings.Contains(err.Error(), "heading_numbering") {
		t.Errorf("the refusal %q does not name the key", err)
	}
}

// TestAnUnknownFieldInARevisionIsRefusedByName. A misspelled column is a cell
// that silently never prints, which is the note reader's rule and the reason
// the strict read reaches inside the list.
func TestAnUnknownFieldInARevisionIsRefusedByName(t *testing.T) {
	_, err := ReadFields([]byte(`{"title": "A Policy", "revisions": [{"version": "1.0", "auther": "N"}]}`))
	if err == nil {
		t.Fatal("an unknown revision field was accepted")
	}
	if !strings.Contains(err.Error(), "auther") {
		t.Errorf("the refusal %q does not name the field", err)
	}
}

func TestARevisionWithNoVersionIsRefused(t *testing.T) {
	_, err := ReadFields([]byte(`{"title": "A Policy", "revisions": [{"change": "Reworded it"}]}`))
	if err == nil {
		t.Fatal("a revision with no version was accepted, so the row has no name")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("the refusal %q does not say what the row is missing", err)
	}
}

// TestTheFieldsFileNamesTheSameThirteenValuesTheNoteDoes states the claim the
// plan makes in words: one shape, two readers. A field added to Fields without
// a line in the fields file is a value a restyle can never state.
func TestTheFieldsFileNamesTheSameThirteenValuesTheNoteDoes(t *testing.T) {
	fromFile := readFields(t, full)
	fromNote, _, err := Read(note(strings.Join([]string{
		"title: Third Party Risk",
		"alt_title: Supplier Risk",
		"doc_type: Policy",
		"version: '3.1'",
		"date: 2026-09-08",
		"owner: Head of Risk",
		"last_approval: 2026-06-01",
		"review_frequency: Annual",
		"board_ratification: 2026-06-15",
		"distribution: All staff",
		"classification: restricted",
		"heading_numbering: none",
		"revisions:",
		"  - version: '3.1'",
		"    date: 2026-09-08",
		"    author: N Khusnullin",
		"    approved_by: Board",
		"    approval_date: 2026-06-15",
		"    section: '4.2'",
		"    change: Reworded the escalation path",
	}, "\n"), ""))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if fromFile.Title != fromNote.Title || fromFile.AltTitle != fromNote.AltTitle ||
		fromFile.DocType != fromNote.DocType || fromFile.Version != fromNote.Version ||
		fromFile.Date != fromNote.Date || fromFile.Owner != fromNote.Owner ||
		fromFile.LastApproval != fromNote.LastApproval ||
		fromFile.ReviewFrequency != fromNote.ReviewFrequency ||
		fromFile.BoardRatification != fromNote.BoardRatification ||
		fromFile.Distribution != fromNote.Distribution ||
		fromFile.Classification != fromNote.Classification ||
		fromFile.HeadingNumbering != fromNote.HeadingNumbering {
		t.Errorf("the fields file read %+v and the note read %+v", fromFile, fromNote)
	}
	if len(fromFile.Revisions) != len(fromNote.Revisions) {
		t.Fatalf("revisions = %d from the file and %d from the note",
			len(fromFile.Revisions), len(fromNote.Revisions))
	}
	for i, want := range fromNote.Revisions {
		if fromFile.Revisions[i] != want {
			t.Errorf("revision %d = %+v from the file, want %+v", i+1, fromFile.Revisions[i], want)
		}
	}
}
