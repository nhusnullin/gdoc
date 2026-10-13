package main

import (
	"strings"
	"testing"
)

// readsBefore counts the Docs reads of the test document that the fake saw
// before the n-th write.
func readsBefore(f *fakeWire, n int) int {
	reads, writes := 0, 0
	for _, c := range f.calls {
		if c.Method != "GET" {
			writes++
			if writes == n {
				return reads
			}
			continue
		}
		if strings.Contains(c.URL, proposeDocID+"?includeTabsContent") {
			reads++
		}
	}
	return reads
}

// TestProposeReadsTheDocumentOnceBeforeTheFirstWrite: the run reads the
// document to count its tabs, and that read is fresh enough to place the first
// proposal from. A second read of the same document a moment later is a call
// to Google that tells gdoc nothing new.
func TestProposeReadsTheDocumentOnceBeforeTheFirstWrite(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from)
	if code != 0 || got["ok"] != true {
		t.Fatalf("the run must succeed: %v (exit %d)", got, code)
	}
	if n := readsBefore(f, 1); n != 1 {
		t.Errorf("%d reads of the document before the first write, want 1", n)
	}
}

// TestEveryLaterProposalStillReadsFresh: only the first proposal borrows the
// run's read. The second is placed from a read made after the first one
// landed, because the first one moved the ground under it.
func TestEveryLaterProposalStillReadsFresh(t *testing.T) {
	answers := append(proposeAnswers(t, true),
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-after.json"), once: true})
	f := stubWire(t, &fakeWire{answers: answers})
	from := tempFile(t, "proposals.json",
		`[{"quoted":"reviewed annually","replacement":"reviewed every six months","why":"the policy says twice a year"},`+
			`{"quoted":"reviewed monthly","replacement":"reviewed weekly","why":"the policy says twice a year"}]`)

	runJSON(t, "propose", proposeDocID, "--from", from)
	first := readsBefore(f, 1)
	total := 0
	for _, c := range f.calls {
		if c.Method == "GET" && strings.Contains(c.URL, proposeDocID+"?includeTabsContent") {
			total++
		}
	}
	// After the first write: its inline read-back, its preview read, and the
	// second proposal's own read.
	if total-first < 3 {
		t.Errorf("%d reads after the first write, want the two read-backs and a fresh read for the second proposal", total-first)
	}
}
