// The strict read of a --from file, one entry at a time.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// strictEntries reads a file that holds one JSON list, and decodes each entry
// of it strictly, on its own.
//
// Decoding the whole list in one strict pass refused an unknown field with Go's
// own words, `json: unknown field "reason"`, which names neither the entry nor
// what gdoc takes instead. A session with no memory file then guesses again.
// One entry at a time, the refusal names the entry, the field, and the fields
// that entry takes, which is the list a caller hands in as fields:
// TestAnUnknownFieldIsRefusedWithTheFieldsTheEntryTakes and
// TestAnUnknownFieldNamesTheEntryItIsIn. The read stays as strict as it was.
func strictEntries[T any](path string, raw []byte, noun, fields string) ([]T, error) {
	var list []json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&list); err != nil {
		return nil, fmt.Errorf("%s is not a list of %s: %w", path, noun, err)
	}
	// Decode stops at the end of the first value, where Unmarshal refused a
	// file with anything behind it. A second list after the first is a file
	// somebody edited wrongly, and running the first half of it silently is the
	// same mistake as dropping a key.
	if dec.More() {
		return nil, fmt.Errorf("%s carries more than one list of %s", path, noun)
	}
	out := make([]T, 0, len(list))
	for i, entry := range list {
		var one T
		ed := json.NewDecoder(bytes.NewReader(entry))
		ed.DisallowUnknownFields()
		if err := ed.Decode(&one); err != nil {
			if name, ok := unknownField(err); ok {
				return nil, fmt.Errorf("%s %s[%d] has a field gdoc does not take: %q. %s",
					path, noun, i, name, fields)
			}
			return nil, fmt.Errorf("%s %s[%d]: %w", path, noun, i, err)
		}
		out = append(out, one)
	}
	return out, nil
}

// unknownField is the field name in the error encoding/json gives for a field
// the struct does not have, and false for every other error.
func unknownField(err error) (string, bool) {
	rest, ok := strings.CutPrefix(err.Error(), "json: unknown field ")
	if !ok {
		return "", false
	}
	return strings.Trim(rest, `"`), true
}
