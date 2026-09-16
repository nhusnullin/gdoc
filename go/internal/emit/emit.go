// Package emit is the one output contract: every command prints exactly one
// JSON object to stdout and exits 0 iff ok. Spec: "Output contract".
package emit

import (
	"encoding/json"
	"io"
)

// Result is the one object every command prints. Error carries what went
// wrong, Warnings what went right in a way the reader still has to know about,
// Version names the release that printed it, and the optional fields are
// absent rather than empty so a skill can tell "nothing to say" from "said
// nothing".
//
// Version is absent for a binary built from a checkout, because such a binary
// belongs to no release and naming one would be a guess. The caller sets it:
// emit knows the shape of the envelope and nothing about how gdoc was built.
type Result struct {
	OK       bool     `json:"ok"`
	Data     any      `json:"data,omitempty"`
	Error    string   `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Version  string   `json:"version,omitempty"`
}

// Print writes the result as one JSON object and a newline. HTML escaping is
// off because the strings here are paths, URLs and Drive ids, and `<` in a
// path helps nobody read it.
func Print(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}

// ExitCode is the other half of the output contract: the process exits 0 if and
// only if the object it printed says ok.
func ExitCode(r Result) int {
	if r.OK {
		return 0
	}
	return 1
}
