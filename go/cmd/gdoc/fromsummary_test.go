package main

import (
	"strings"
	"testing"
)

// TestTheFromSummaryNamesEveryRequiredField: a session that learns propose from
// help alone writes the file the summary describes. Leaving out why, the one
// field every entry needs, is how a session guesses reason and is refused once.
func TestTheFromSummaryNamesEveryRequiredField(t *testing.T) {
	var summary string
	for _, c := range commands() {
		if c.name != "propose" {
			continue
		}
		for _, f := range c.flags {
			if f.name == "--from" {
				summary = f.summary
			}
		}
	}
	for _, want := range []string{"quoted, replacement", "why"} {
		if !strings.Contains(summary, want) {
			t.Errorf("propose --from summary must name %q: %q", want, summary)
		}
	}
}
