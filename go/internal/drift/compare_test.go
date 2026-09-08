package drift

import (
	"strings"
	"testing"
)

// TestVerdictIsCompareDotPysRule states the four verdicts as literals rather
// than as the constants they are compared against. A test that reads the
// tolerance it is testing is a mirror: raise DefaultTol to ten points and the
// assertion follows it.
func TestVerdictIsCompareDotPysRule(t *testing.T) {
	cases := []struct {
		name string
		a, b any
		tol  float64
		want Verdict
	}{
		{"two nils are the same answer", nil, nil, 0, Identical},
		{"one nil is missing, not different", 11.0, nil, 0, Missing},
		{"the other nil is missing too", nil, 11.0, 0, Missing},
		{"equal numbers", 11.0, 11.0, 0, Identical},
		{"a quarter of a point apart is close", 11.0, 11.25, 0, Close},
		{"three quarters of a point apart is still close", 11.0, 11.75, 0, Close},
		{"a whole point apart is different", 11.0, 12.0, 0, Different},
		{"a per-item tolerance widens it", 11.0, 12.0, 2, Close},
		{"a per-item tolerance of zero narrows it", 11.0, 11.25, -1, Different},
		{"equal words", "START", "START", 0, Identical},
		{"different words", "START", "CENTER", 0, Different},
		{"true is not one", true, true, 0, Identical},
		{"true against false is different, never close", true, false, 0, Different},
		{"a list is judged on its worst member", []float64{1, 2, 3}, []float64{1, 2, 3.5}, 0, Close},
		{"a list beyond the tolerance is different", []float64{1, 2, 3}, []float64{1, 2, 9}, 0, Different},
		{"two lists of different lengths are different", []float64{1, 2}, []float64{1, 2, 3}, 0, Different},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _, _ := verdict(c.a, c.b, tolerance(c.tol))
			if got != c.want {
				t.Errorf("%v against %v under tolerance %v gave %s, and it is %s", c.a, c.b, c.tol, got, c.want)
			}
		})
	}
}

// TestDefaultToleranceIsThreeQuartersOfAPoint states the house number as a
// literal. compare.py's reason is the assertion: below this nothing is visible.
func TestDefaultToleranceIsThreeQuartersOfAPoint(t *testing.T) {
	if DefaultTol != 0.75 {
		t.Errorf("the default tolerance is %v, and it is 0.75 of a point", DefaultTol)
	}
}

// TestCompareLinesUpTheRowsByName is the guarantee the whole package rests on:
// one item list read twice gives two lists in the same order, so a row is the
// same question on both sides.
func TestCompareLinesUpTheRowsByName(t *testing.T) {
	a := []Value{{Name: "page width (pt)", V: 595.28}, {Name: "page height (pt)", V: 841.89}}
	b := []Value{{Name: "page width (pt)", V: 595.28}, {Name: "page height (pt)", V: 841.0}}
	rows := Compare(a, b)
	if len(rows) != 2 {
		t.Fatalf("two values against two gave %d rows", len(rows))
	}
	if rows[0].Verdict != Identical {
		t.Errorf("the same page width came back %s", rows[0].Verdict)
	}
	if rows[1].Verdict != Different {
		t.Errorf("a page height 0.89 of a point out came back %s, and it is DIFFERENT", rows[1].Verdict)
	}
}

// TestCompareSaysSoWhenTheTwoReadingsDoNotLineUp. Two lists out of one Items
// list cannot disagree, so a disagreement is a bug in this package and the row
// says that rather than reporting a difference between two documents.
func TestCompareSaysSoWhenTheTwoReadingsDoNotLineUp(t *testing.T) {
	rows := Compare(
		[]Value{{Name: "page width (pt)", V: 1.0}},
		[]Value{{Name: "page height (pt)", V: 1.0}})
	if rows[0].Verdict != Different || !strings.Contains(rows[0].Note, "bug in this package") {
		t.Errorf("a row read under two names came back %s with note %q", rows[0].Verdict, rows[0].Note)
	}

	short := Compare([]Value{{Name: "one", V: 1.0}, {Name: "two", V: 1.0}}, []Value{{Name: "one", V: 1.0}})
	if len(short) != 2 || short[1].Verdict != Missing {
		t.Errorf("a short reading gave %d rows ending %v", len(short), short[len(short)-1].Verdict)
	}
	long := Compare([]Value{{Name: "one", V: 1.0}}, []Value{{Name: "one", V: 1.0}, {Name: "two", V: 1.0}})
	if len(long) != 2 || long[1].Verdict != Missing {
		t.Errorf("a long reading gave %d rows ending %v", len(long), long[len(long)-1].Verdict)
	}
}

// TestUnexplainedIsWhatAGateFailsOn. A named difference is already understood
// and does not fail the gate; anything else does, and it is named.
func TestUnexplainedIsWhatAGateFailsOn(t *testing.T) {
	rows := []Row{
		{Item: "page width (pt)", Verdict: Identical},
		{Item: "logo topOffset (pt)", Verdict: Close},
		{Item: "table count", Verdict: Different},
		{Item: "HEADING_2 spaceAbove", Verdict: Different},
		{Item: "logo width (pt)", Verdict: Missing},
	}
	got := Unexplained(rows)
	if len(got) != 2 {
		t.Fatalf("two rows are unexplained and %d came back: %v", len(got), got)
	}
	if got[0].Item != "HEADING_2 spaceAbove" || got[1].Item != "logo width (pt)" {
		t.Errorf("the unexplained rows are %q and %q", got[0].Item, got[1].Item)
	}
}

// TestEveryKnownDifferenceCarriesItsReason. Known is a decision somebody wrote
// down, so an entry with no reason in it is an assertion loosened rather than a
// difference explained.
func TestEveryKnownDifferenceCarriesItsReason(t *testing.T) {
	for name, why := range Known {
		if len(strings.Fields(why)) < 5 {
			t.Errorf("known difference %q carries %q, which is not a reason", name, why)
		}
	}
}

// TestEveryKnownDifferenceNamesARealItem. A name that no item produces is a
// row that will never be compared, so the entry silently protects nothing.
func TestEveryKnownDifferenceNamesARealItem(t *testing.T) {
	names := map[string]bool{}
	for _, it := range Items {
		names[it.Name] = true
	}
	for name := range Known {
		if !names[name] {
			t.Errorf("known difference %q is not an item this package compares", name)
		}
	}
}

// TestTableAndSummaryPrintEveryRow. The report is what a failure prints, so a
// row missing from it is a difference nobody can read.
func TestTableAndSummaryPrintEveryRow(t *testing.T) {
	rows := Compare(
		[]Value{{Name: "page width (pt)", V: 595.28}, {Name: "body font", V: nil}},
		[]Value{{Name: "page width (pt)", V: 500.0}, {Name: "body font", V: "Calibri"}})
	out := Table(rows)
	for _, want := range []string{"page width (pt)", "DIFFERENT", "body font", "MISSING", "<none>", "595.28"} {
		if !strings.Contains(out, want) {
			t.Errorf("the report does not carry %q:\n%s", want, out)
		}
	}
	if got := Summary(rows); !strings.Contains(got, "2 items") || !strings.Contains(got, "DIFFERENT 1") {
		t.Errorf("the summary reads %q", got)
	}
}

// TestATypeMismatchIsABugRatherThanAnAnswer. The %v fallback compared a number
// against its own decimal spelling and called them the same: verdict("1", 1.0)
// read IDENTICAL. Two halves of one row that answer in different types is a bug
// in this package, the way a name that does not line up is, and it has to be
// reported as one rather than hidden behind a string comparison.
func TestATypeMismatchIsABugRatherThanAnAnswer(t *testing.T) {
	got, note, bug := verdict("1", 1.0, tolerance(0))
	if got == Identical {
		t.Errorf("a string against a number came back %s, and the two are not one answer", got)
	}
	if !strings.Contains(note, "bug in this package") {
		t.Errorf("a string against a number carried note %q, and it names a bug here", note)
	}
	if !bug {
		t.Error("a string against a number is not marked as a bug in this package, so Known can explain it away")
	}
}

// TestABugRowIsNeverExplainedAway. Known explains a difference between the two
// documents. A row saying the fault is in this package is not that, so landing
// on a name in Known must not drop it: ten of the twenty-five names carry rows
// that can go wrong this way, and the gate would pass in silence.
func TestABugRowIsNeverExplainedAway(t *testing.T) {
	rows := []Row{
		{Item: "table count", Verdict: Different, Bug: true,
			Note: "read under two names, which is a bug in this package"},
		{Item: "table count", Verdict: Different,
			Note: "the two documents hold different tables"},
	}
	got := Unexplained(rows)
	if len(got) != 1 {
		t.Fatalf("one row is a bug here and %d came back: %v", len(got), got)
	}
	if !got[0].Bug {
		t.Errorf("the row that came back is %q, and it is the document difference Known explains", got[0].Note)
	}
}
