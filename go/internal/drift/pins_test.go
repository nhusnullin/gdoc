package drift

import (
	"fmt"
	"strings"
	"testing"
)

// pair is the two sides of one row as show prints them.
type pair struct{ A, B string }

// knownOffline pins the values every Known difference reads today.
//
// Known says a difference has been looked at and gives the reason. On its own
// that exempts the row for ever: the gate stopped reading it, so the row could
// move to any other pair of values and still pass. This map is the second
// half. A Known row must still read the pair that was measured when its reason
// was written, or the offline gate fails naming the row, what it reads now and
// what it read then.
//
// A failure here is never a pin to bring up to date on the way past. It says
// one of two things happened: the generator or the house style moved, which is
// what the gate exists to catch, or the difference really is a new one, in
// which case re-recording the pair is a decision somebody writes down with the
// reason beside the Known entry, the same rule Known already states.
//
// Every value is a literal, printed by show, never read out of house.yaml or
// computed from the master.
var knownOffline = map[string]pair{
	"HEADING_1 colour":                         {`"#22265F"`, `"#06436E"`},
	"HEADING_1 indentStart":                    {`0`, `36`},
	"HEADING_2 colour":                         {`"#22265F"`, `"#06436E"`},
	"HEADING_2 indentStart":                    {`0`, `72`},
	"HEADING_3 indentStart":                    {`0`, `108`},
	"HEADING_4 indentStart":                    {`0`, `144`},
	"HEADING_5 indentStart":                    {`0`, `180`},
	"HEADING_6 indentStart":                    {`0`, `216`},
	"body H1 run colour":                       {`"#22265F"`, `"#222660"`},
	"body H1 text":                             {`"1-Third Party and Outsourcing Policy"`, `"1-Purpose"`},
	"body H2 run colour":                       {`"#22265F"`, `"#222660"`},
	"body H2 text":                             {`"1.1-Purpose and scope"`, `"Appendix 1 – Associated Documents"`},
	"body H3 indentStart":                      {`0`, `<none>`},
	"body H3 run colour":                       {`"#549F99"`, `<none>`},
	"body H3 text":                             {`"1.1.1-What counts as outsourcing"`, `<none>`},
	"bulleted paragraphs":                      {`17`, `27`},
	"default header text":                      {`"Altery - Supplier Management Policy"`, `"Altery - xxx Policy"`},
	"table Document Classification cell fills": {`"#BDCDD2\x1f#BDCDD2\x1f#F4CCCC\x1f#F4CCCC\x1f#FCE5CD\x1f\x1f#FFF2CC\x1f\x1f#D9EAD3\x1f"`, `"#BDCDD2\x1f#BDCDD2\x1f#F4CCCC\x1f\x1f#FCE5CD\x1f\x1f#FFF2CC\x1f#FFF2CC\x1f#D9EAD3\x1f"`},
	"table Revision History cell borders":      {`"0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000"`, `"0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000\x1f0.500pt #000000"`},
	"table Revision History cell fills":        {`"#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9"`, `"#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#BDCDD2\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9\x1f#F3F8F9"`},
	"table Revision History cell text":         {`"Version No\x1fDate\x1fVersion Author\x1fApproved By\x1fApproval Date\x1fSection Updated\x1fRevisions/Changes\x1f3.1\x1f29 August 2026\x1fN Khusnullin\x1fBoard\x1f20 August 2026\x1f4, 6\x1fAligned the exit plan with the operational resilience framework"`, `"Version No\x1fDate\x1fVersion Author\x1fApproved By\x1fApproval Date\x1fSection Updated\x1fRevisions/Changes\x1fxx\x1fxx\x1fxx\x1f[Relevant Board Level Committee]\x1fxx\x1fNew document\x1f\x1f\x1f\x1f\x1f\x1f\x1f\x1f"`},
	"table Revision History row heights":       {`[32.4 28.35]`, `[32.4 28.35 28.35]`},
	"table Revision History size":              {`"2x7"`, `"3x7"`},
	"table Version Control cell text":          {`"Document Owner\x1fChief Risk Officer\x1fDate of Last Approval\x1f14 August 2026\x1fReview Frequency\x1fAnnually\x1fBoard Ratification Date\x1f20 August 2026\x1fPolicy Distribution\x1fAll staff, all contractors, and the Board"`, `"Document Owner\x1f\x1fDate of Last Approval\x1f\x1fReview Frequency\x1fAnnually\x1fBoard Ratification Date\x1f\x1fPolicy Distribution\x1f"`},
	"table count":                              {`6`, `3`},
}

// unpinned checks every Known row against its pin and returns one sentence per
// row that does not match. It is the gate's check, out here so a test can hand
// it rows built by hand.
func unpinned(rows []Row) []string {
	out := []string{}
	for _, r := range rows {
		if _, ok := Known[r.Item]; !ok {
			continue
		}
		p, ok := knownOffline[r.Item]
		if !ok {
			out = append(out, fmt.Sprintf("known difference %q reads %s | %s and has no pinned pair: record it in knownOffline beside the reason in Known",
				r.Item, show(r.A), show(r.B)))
			continue
		}
		if show(r.A) != p.A || show(r.B) != p.B {
			out = append(out, fmt.Sprintf("known difference %q now reads %s | %s, and the pinned pair is %s | %s: re-record it in knownOffline with the reason, or fix what moved",
				r.Item, show(r.A), show(r.B), p.A, p.B))
		}
	}
	return out
}

// TestEveryKnownDifferenceHasItsPairPinned is the two directions between the
// two maps. A Known name with no pin is a row the gate stopped reading, and a
// pin naming nothing in Known is a pin nothing checks.
func TestEveryKnownDifferenceHasItsPairPinned(t *testing.T) {
	for name := range Known {
		if _, ok := knownOffline[name]; !ok {
			t.Errorf("known difference %q has no pinned pair, so the gate lets that row read anything", name)
		}
	}
	for name := range knownOffline {
		if _, ok := Known[name]; !ok {
			t.Errorf("pinned pair %q names no entry in Known, so nothing checks it", name)
		}
	}
}

// TestAKnownRowWhoseValuesMovedFailsTheGate runs the gate's own check over
// rows built by hand: one Known row that moved away from its pin, and one that
// still reads it.
func TestAKnownRowWhoseValuesMovedFailsTheGate(t *testing.T) {
	moved := unpinned([]Row{{Item: "table count", A: 9, B: 3, Verdict: Different}})
	if len(moved) != 1 {
		t.Fatalf("a known row reading 9 against a pin of 6 gave %d sentences, and it is one: %v", len(moved), moved)
	}
	for _, want := range []string{`"table count"`, "9", "6", "3"} {
		if !strings.Contains(moved[0], want) {
			t.Errorf("the sentence for a moved row does not name %s: %s", want, moved[0])
		}
	}

	if still := unpinned([]Row{{Item: "table count", A: 6, B: 3, Verdict: Different}}); len(still) != 0 {
		t.Errorf("a known row still reading its pinned pair gave %d sentences, and it is none: %v", len(still), still)
	}
}
