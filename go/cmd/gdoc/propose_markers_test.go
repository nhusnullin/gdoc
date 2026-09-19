package main

import (
	"strings"
	"testing"
)

// TestProposeRefusesAMarkerInTheFile is decision 3's last door. Document text
// never passes through internal/plaintext: the quote and the replacement go
// into the document itself, not into a thread, so the marker rule has to be
// asked here, in the read of the --from file.
//
// A skill that proposes from a note it has just exported, without resolving the
// markers first, would otherwise send "{-annually-}[s:AAA]" into the document
// as words. The refusal names the field, because the file has two of them and
// the person has to know which one to fix.
func TestProposeRefusesAMarkerInTheFile(t *testing.T) {
	for _, tc := range []struct{ name, file, says string }{
		{"the replacement carries a deletion marker",
			`[{"quoted":"a","replacement":"the team {-met monthly-}[s:AAA]","why":"c"}]`,
			"replacement"},
		{"the quote carries an insertion marker",
			`[{"quoted":"the team {+meets weekly+}[s:AAA]","replacement":"b","why":"c"}]`,
			"quoted"},
		{"the reason carries a comment anchor",
			`[{"quoted":"a","replacement":"b","why":"see [[c:AAA]]the team[[/c]]"}]`,
			"marker"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
			from := tempFile(t, "proposals.json", tc.file)

			got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
			if code == 0 || got["ok"] != false {
				t.Fatalf("a proposal carrying a marker must stop the run: %v (exit %d)", got, code)
			}
			msg, _ := got["error"].(string)
			if !strings.Contains(msg, tc.says) {
				t.Errorf("the error must say %q: %q", tc.says, msg)
			}
			if !strings.Contains(msg, "marker") {
				t.Errorf("the error must call it a marker: %q", msg)
			}
			if len(f.calls) != 0 {
				t.Errorf("nothing may reach Google, the probe document included: %v", f.calls)
			}
		})
	}
}

// TestProposeAcceptsAnEscapedMarker is the other half. A proposal whose words
// are about the markers is written the way `read` writes them, escaped, and
// that is text a document may carry.
func TestProposeAcceptsAnEscapedMarkerInTheFile(t *testing.T) {
	stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	from := tempFile(t, "proposals.json",
		`[{"quoted":"a","replacement":"the export writes \\{+words\\+} for an insertion","why":"c"}]`)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if msg, _ := got["error"].(string); strings.Contains(msg, "marker") {
		t.Fatalf("an escaped marker is the author's own text: %q (exit %d)", msg, code)
	}
}
