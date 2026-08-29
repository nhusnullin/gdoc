// Package emit is the one output contract: every command prints exactly one
// JSON object to stdout and exits 0 iff ok. Spec: "Output contract".
package emit

import (
	"encoding/json"
	"io"
)

type Result struct {
	OK       bool     `json:"ok"`
	Data     any      `json:"data,omitempty"`
	Error    string   `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func Print(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}

func ExitCode(r Result) int {
	if r.OK {
		return 0
	}
	return 1
}
