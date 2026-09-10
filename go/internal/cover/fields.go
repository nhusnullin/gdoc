package cover

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// fieldsFileWhere names the file ReadFields looked for a title in, so the one
// refusal reads truthfully whichever reader made it.
const fieldsFileWhere = "the fields file"

// numberingWords are the values heading_numbering may be written as, listed in
// a refusal so somebody reads what to write instead.
var numberingWords = []string{"auto", "none", "true", "false"}

// fieldsFile is the fields file's own shape: the same thirteen values Fields
// holds, spelled the way a note's front matter spells them, each one as the
// author wrote it and before the two normalisations below.
//
// The keys are the note's keys on purpose. A restyle has no note to read front
// matter from, so somebody writing this file writes it beside a note they have
// already written, and one value must not have two names.
//
// heading_numbering is a raw message because JSON has a boolean and v1's notes
// have words. Both are read, and anything else is refused naming the key.
type fieldsFile struct {
	Title             string          `json:"title"`
	AltTitle          string          `json:"alt_title"`
	DocType           string          `json:"doc_type"`
	Version           string          `json:"version"`
	Date              string          `json:"date"`
	Owner             string          `json:"owner"`
	LastApproval      string          `json:"last_approval"`
	ReviewFrequency   string          `json:"review_frequency"`
	BoardRatification string          `json:"board_ratification"`
	Distribution      string          `json:"distribution"`
	Classification    string          `json:"classification"`
	HeadingNumbering  json.RawMessage `json:"heading_numbering"`
	Revisions         []revisionRow   `json:"revisions"`
}

// revisionRow is one row of the revision-history table, named by the seven
// columns readRevisions already reads out of a note.
type revisionRow struct {
	Version      string `json:"version"`
	Date         string `json:"date"`
	Author       string `json:"author"`
	ApprovedBy   string `json:"approved_by"`
	ApprovalDate string `json:"approval_date"`
	Section      string `json:"section"`
	Change       string `json:"change"`
}

// ReadFields reads the cover's values out of a JSON file rather than out of a
// note's front matter.
//
// `gdoc restyle` styles a document that has no note behind it, so there is no
// front matter to read the cover from and the values arrive in a file instead.
// What comes back is the same Fields the note reader returns, defaults and all:
// two readers of one shape, and a value left out means here exactly what it
// means there, or one cover would be built two ways.
//
// The read is strict, the way readSurvey and readProposals read theirs. An
// unknown key is refused by name and so is a second object behind the first.
// That is the opposite of the note reader, which carries a key it does not
// read, and the difference is whose file it is: a note is the author's and gdoc
// owns one key in it, while this file is gdoc's own shape written for one
// command, so a key it does not know is a misspelling, and a misspelled value
// is a cover line that silently never prints.
//
// A missing title is the note reader's refusal, MissingTitle, carrying no
// candidate: this file has no body and no name to draw one from. Nothing here
// invents a title, and the skill proposes one for a person to confirm.
func ReadFields(raw []byte) (Fields, error) {
	var file fieldsFile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return Fields{}, fmt.Errorf("the fields file is not the JSON object `gdoc restyle --fields` reads: %w", err)
	}
	// Decode stops at the end of the first value, where Unmarshal refused a
	// file with anything behind it. Two objects is a file somebody appended a
	// second cover to, and reading the first half of it silently is the same
	// mistake as reading past an unknown key.
	if dec.More() {
		return Fields{}, fmt.Errorf(
			"the fields file carries more than one JSON object, and which cover counts is not decided here")
	}

	title := strings.TrimSpace(file.Title)
	if title == "" {
		return Fields{}, &MissingTitle{Where: fieldsFileWhere}
	}
	label, err := classificationLabel(file.Classification)
	if err != nil {
		return Fields{}, err
	}
	numbers, err := fieldsNumbering(file.HeadingNumbering)
	if err != nil {
		return Fields{}, err
	}
	revisions, err := fieldsRevisions(file.Revisions)
	if err != nil {
		return Fields{}, err
	}

	f := Fields{
		Title:             title,
		AltTitle:          value(file.AltTitle),
		DocType:           value(file.DocType),
		Version:           value(file.Version),
		Date:              value(file.Date),
		Owner:             value(file.Owner),
		LastApproval:      value(file.LastApproval),
		ReviewFrequency:   value(file.ReviewFrequency),
		BoardRatification: value(file.BoardRatification),
		Distribution:      value(file.Distribution),
		Classification:    label,
		HeadingNumbering:  numbers,
		Revisions:         revisions,
	}
	if f.Version == "" {
		f.Version = DefaultVersion
	}
	if f.Date == "" {
		// The note reader's rule, and it is here for the same reason: a date
		// left empty reaches the cover as the template's own highlighted "May
		// 2025", which is when the master was captured.
		f.Date = now().Format(defaultDateLayout)
	}
	return f, nil
}

// value is one written value read the way the note reader reads a scalar: an
// ISO date becomes the UK long form the cover states, and everything else is
// the author's own words with the spaces trimmed.
func value(written string) string {
	return longDate(strings.TrimSpace(written))
}

// fieldsNumbering reads heading_numbering as a boolean or as one of v1's words.
// An absent key is the note reader's answer to an absent key, which is on.
func fieldsNumbering(raw json.RawMessage) (bool, error) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return numbering("")
	}
	var written bool
	if err := json.Unmarshal(raw, &written); err == nil {
		return written, nil
	}
	var word string
	if err := json.Unmarshal(raw, &word); err == nil {
		return numbering(word)
	}
	return false, fmt.Errorf("heading_numbering %s is not true, false, or one of %v", raw, numberingWords)
}

// fieldsRevisions checks the rows the file states. A row needs its version,
// which is the row's own name; every other cell may be blank. An unknown column
// was refused by the strict decode above, because a misspelled column is a cell
// that silently never prints.
func fieldsRevisions(rows []revisionRow) ([]Revision, error) {
	var out []Revision
	for index, row := range rows {
		if strings.TrimSpace(row.Version) == "" {
			return nil, fmt.Errorf("revision %d has no 'version', so the row has no name", index+1)
		}
		out = append(out, Revision{
			Version:      value(row.Version),
			Date:         value(row.Date),
			Author:       value(row.Author),
			ApprovedBy:   value(row.ApprovedBy),
			ApprovalDate: value(row.ApprovalDate),
			Section:      value(row.Section),
			Change:       value(row.Change),
		})
	}
	return out, nil
}
