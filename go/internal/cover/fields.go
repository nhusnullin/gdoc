package cover

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
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
// heading_numbering is a raw message because JSON has a boolean and a note's
// front matter carries the words "auto" and "none". Both are read, and
// anything else is refused naming the key.
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

	if err := checkStrippable(file); err != nil {
		return Fields{}, err
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

// fieldsNumbering reads heading_numbering as a boolean or as one of the words
// a note's front matter carries. An absent key is the note reader's answer to
// an absent key, which is on.
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

// checkStrippable refuses a value carrying a character the Docs API removes
// from an insert, naming the key it is in.
//
// InsertTextRequest says which ones in its own words: "Some control characters
// (U+0000-U+0008, U+000C-U+001F) and characters from the Unicode Basic
// Multilingual Plane Private Use Area (U+E000-U+F8FF) will be stripped out of
// the inserted text." U+0009, the tab, is not among them, which is why the
// house legend can write one.
//
// It is refused here rather than left to the writer because of what a stripped
// character costs. internal/prelude computes every index it names itself, from
// the length of the string it is about to send, so a unit Docs drops puts every
// later insert one place out. The loud outcome is Docs refusing the whole batch
// for an index that is inside no paragraph, which is what a missing table unit
// did on 2026-09-10. The quiet one is worse: where the wrong index is still
// valid the prelude ends one short, the marker is written one character into
// the author's own text, and the next run proposes deleting a character they
// wrote.
//
// The live route is an escape rather than a raw byte. A raw control character
// is invalid JSON and the decoder above refuses it, while "\u001f" is what
// json.Marshal writes for text pasted out of a word processor.
//
// This is the fields file's rule and not the note reader's. A note builds a
// docx, where nothing computes a Docs index, and a note is the author's file
// where gdoc carries what it does not read.
func checkStrippable(file fieldsFile) error {
	v := reflect.ValueOf(file)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		key := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		switch value := v.Field(i).Interface().(type) {
		case string:
			if err := strippable(key, value); err != nil {
				return err
			}
		case []revisionRow:
			for n, row := range value {
				if err := checkStrippableRow(fmt.Sprintf("%s[%d]", key, n+1), row); err != nil {
					return err
				}
			}
		case json.RawMessage:
			// heading_numbering, and it is read to a boolean that reaches no
			// insert. There is no text in it to strip.
		default:
			return unreadableField(key, t.Field(i).Type.String())
		}
	}
	return nil
}

// unreadableField is what both walks say about a field they cannot read.
//
// The reflection is here so the check covers every value without an edit, and a
// reader takes it that way. A field of a kind neither walk knows is one they
// have stopped covering, so it fails rather than passing in silence: that is
// the default arm internal/docs' own decoder learned to report from, and the
// cost of the silence here is the quiet one the doc comment above names, a
// prelude one character short and a marker written into the author's own text.
func unreadableField(key, kind string) error {
	return fmt.Errorf(
		"%s is a %s, and the fields file's check for the characters the Docs API strips cannot read one: "+
			"give internal/cover a case for it before this key reaches an inserted text", key, kind)
}

// checkStrippableRow is the same question of one revision row's own columns.
//
// Every column is a string today, and the kind is asked rather than assumed:
// reflect.Value.String() does not panic on another kind, it hands back a
// placeholder such as "<int Value>", so a column added later would be checked
// against something that is not its value and would pass.
func checkStrippableRow(where string, row revisionRow) error {
	v := reflect.ValueOf(row)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		key := where + "." + strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if v.Field(i).Kind() != reflect.String {
			return unreadableField(key, t.Field(i).Type.String())
		}
		if err := strippable(key, v.Field(i).String()); err != nil {
			return err
		}
	}
	return nil
}

// strippable is one value, and the refusal names the key and the code point.
func strippable(key, written string) error {
	for _, r := range written {
		if (r <= 0x08) || (r >= 0x0C && r <= 0x1F) || (r >= 0xE000 && r <= 0xF8FF) {
			return fmt.Errorf(
				"%s carries U+%04X, which the Docs API strips out of an inserted text: "+
					"gdoc counts the characters it sends to place everything after them, so take it out and run this again", key, r)
		}
	}
	return nil
}
