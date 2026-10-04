package main

import (
	"strings"
	"testing"
)

// TestARunTimeRefusalNamesTheProposalByNumber: a refusal raised while the run
// is out, after the file was checked, says which proposal it is about, the way
// a refusal of the file itself names its entry. The arrangement is
// TestProposeReportsEveryProposalWhenOneOfThemCannotBeSent's: the first quote
// lands, the second is not in the document.
func TestARunTimeRefusalNamesTheProposalByNumber(t *testing.T) {
	answers := append(proposeAnswers(t, true),
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-after.json"), once: true})
	stubWire(t, &fakeWire{answers: answers})
	from := tempFile(t, "proposals.json",
		`[{"quoted":"reviewed annually","replacement":"reviewed every six months","why":"the policy says twice a year"},`+
			`{"quoted":"reviewed monthly","replacement":"reviewed weekly","why":"the policy says twice a year"}]`)

	got, _ := runJSON(t, "propose", proposeDocID, "--from", from)
	msg, _ := got["error"].(string)
	if want := `proposal 2 ("reviewed monthly"): `; !strings.HasPrefix(msg, want) {
		t.Errorf("error = %q, want it to open with %q", msg, want)
	}
	first, _ := dataOf(t, got)["proposals"].([]any)[0].(map[string]any)
	if first["sent"] != true {
		t.Errorf("the first proposal landed and reports sent = %v", first["sent"])
	}
}
