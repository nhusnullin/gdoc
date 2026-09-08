// Package drift is the measurement that makes the 2026-08-29 decision safe
// rather than brave.
//
// The house style became a config file, and the master .docx became provenance.
// What answers the risk in that trade is one comparison: read the same list of
// values out of a document built from house.yaml and out of the master, and
// fail when a value moves. DECISIONS.md says it plainly: "the 160-item
// comparison stays as a test. It renders from the config, renders from the
// master, and fails when any value drifts."
//
// The list is written once, in items.go, and it is read two ways. FromDocx
// reads a value out of the docx XML, which is what the offline gate in
// `make test` compares. FromDoc reads the same value out of a Docs API answer,
// which is the live gate: Google's import is part of the result, so the
// measurement that means something happens after an upload. The names are the
// same in both, so a row that fails in one is findable in the other, and both
// match compare.py's names in the 2026-08-29 report.
//
// Nothing here judges. A Row carries what each side said and a verdict about
// the two numbers; whether a difference matters is Nail's, in Word or in Drive.
// Known is the list of differences already explained, and each entry carries
// the explanation rather than the word "expected".
package drift

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
)

// Verdict is what comparing one value on both sides said. The four are
// compare.py's, kept by name so a row here reads like a row there.
type Verdict string

const (
	// Identical: the two sides said the same thing.
	Identical Verdict = "IDENTICAL"
	// Close: two numbers inside the item's tolerance. Below the tolerance
	// nothing is visible on the page, which is what the tolerance is for.
	Close Verdict = "CLOSE"
	// Different: the two sides disagree.
	Different Verdict = "DIFFERENT"
	// Missing: one side has the value and the other has nothing. It is not
	// Different because the answer is not "another value", it is "no value",
	// and those two send somebody to look in different places.
	Missing Verdict = "MISSING"
)

// DefaultTol is the tolerance in points a numeric item carries unless it names
// its own. compare.py's comment is the reason: below three quarters of a point
// nothing is visible.
const DefaultTol = 0.75

// equal is the distance under which two numbers are the same number rather than
// a close one. It is float noise, not a tolerance: a point read as twips and
// divided back does not land on the same bits it started from.
const equal = 1e-6

// Value is one measured value: the item's name, the tolerance that item
// carries, and what was read. A nil V means the document does not carry it,
// which is an answer rather than a failure to look.
type Value struct {
	Name string
	Tol  float64
	V    any
}

// Row is one line of the report: what each side said, and the verdict.
type Row struct {
	Item    string
	A       any
	B       any
	Verdict Verdict
	// Note carries the one thing a verdict cannot say, the worst difference in
	// a list of numbers. It is empty on every other row.
	Note string
	// Bug says the fault is in this package rather than in either document: two
	// readings that did not line up, or two halves of one row answering in
	// different types. Known explains a difference between the two documents,
	// so it must never explain one of these away.
	Bug bool
}

// Known names the differences that have already been looked at, with the reason
// each one is there. The offline gate fails on a DIFFERENT or a MISSING row
// whose name is not in here, so adding a name is a decision somebody writes
// down rather than a test somebody loosens.
//
// Three groups, and they are three different kinds of thing.
//
// The first is the master saying one thing in a style and the opposite on every
// paragraph that uses it. DECISIONS.md measured this in 2026-08-29 and counted
// 21 rows of it: "the template declares a heading colour and then overrides it
// on every paragraph while the config states the effective one". The XML
// carries both statements, so the offline gate sees the declared one and the
// live gate, which reads what Docs resolved, sees the effective one. house.yaml
// states the effective one, which is what a reader sees.
//
// The second is the two documents holding different words. The master is the
// template, with "xxx" where a title goes and its own body text; the built
// document is a real note. Every row that reads text rather than a measurement
// differs for that reason and only that reason.
//
// The third is one colour, one unit apart. Google's docx export writes the
// house navy as #222660 where the Docs API reports #22265F, which is the value
// house.yaml states. One unit of blue in 255 is not a colour anybody can see,
// and the live gate reads them as the same value.
var Known = map[string]string{
	// The master declares the heading colour on the style and overrides it to
	// the house navy on every heading paragraph.
	"HEADING_1 colour": "the master's style declares #06436E and every heading paragraph overrides it to the house navy; house.yaml states the effective one",
	"HEADING_2 colour": "the master's style declares #06436E and every heading paragraph overrides it to the house navy; house.yaml states the effective one",

	// The master's heading styles carry Word's list indents, and every heading
	// paragraph overrides them back to zero.
	"HEADING_1 indentStart": "the master's style indents the heading and every heading paragraph sets the indent back to zero; house.yaml states the effective one",
	"HEADING_2 indentStart": "the master's style indents the heading and every heading paragraph sets the indent back to zero; house.yaml states the effective one",
	"HEADING_3 indentStart": "the master's style indents the heading and every heading paragraph sets the indent back to zero; house.yaml states the effective one",
	"HEADING_4 indentStart": "the master's style indents the heading and every heading paragraph sets the indent back to zero; house.yaml states the effective one",
	"HEADING_5 indentStart": "the master's style indents the heading and every heading paragraph sets the indent back to zero; house.yaml states the effective one",
	"HEADING_6 indentStart": "the master's style indents the heading and every heading paragraph sets the indent back to zero; house.yaml states the effective one",

	// The two documents hold different words.
	"default header text": "the master is the template and its running head reads \"Altery - xxx Policy\"; the built document carries the note's own title",
	"body H1 text":        "the master's body is the template's own text and the built document is the note's",
	"body H2 text":        "the master's body is the template's own text and the built document is the note's",
	"table count":         "the three front matter tables are compared one by one below; the built document carries the note's own tables after them",
	"bulleted paragraphs": "the master's body is the template's own text and the built document is the note's",

	// The master's body has no level-three heading in it at all, so there is
	// nothing on that side to read. It is MISSING rather than DIFFERENT, and
	// the two send somebody to look in different places.
	"body H3 text":        "the master's body carries no level-three heading, so there is nothing on that side to read",
	"body H3 indentStart": "the master's body carries no level-three heading, so there is nothing on that side to read",
	"body H3 run colour":  "the master's body carries no level-three heading, so there is nothing on that side to read",

	// The front matter tables the note fills in. The master is the template,
	// with its own blanks and its "xx" prototype row and Internal marked; the
	// built document carries the note's own owner, revisions and class. These
	// are the rows that would go IDENTICAL again if the generator stopped
	// placing what the note declares, which is what M5 was fixing.
	"table Version Control cell text":          "the master leaves the owner and the approval dates blank for a person to fill in; the built document carries the note's own",
	"table Revision History size":              "the master keeps its \"xx\" prototype row and a blank row behind it; the built document carries one row per revision the note declares",
	"table Revision History cell fills":        "the master keeps its \"xx\" prototype row and a blank row behind it, so it has one more row of fills than the note's revisions produce",
	"table Revision History cell text":         "the master's rows read \"xx\"; the built document carries the note's own revisions",
	"table Revision History cell borders":      "the master keeps its \"xx\" prototype row and a blank row behind it, so it has one more row of borders than the note's revisions produce",
	"table Revision History row heights":       "the master keeps its \"xx\" prototype row and a blank row behind it, so it has one more row height than the note's revisions produce",
	"table Document Classification cell fills": "the master was captured with Internal marked; the built document shades the class the note declares, which is Confidential",

	// One unit of blue.
	"body H1 run colour": "Google's docx export writes the house navy as #222660 where the Docs API reports the #22265F house.yaml states",
	"body H2 run colour": "Google's docx export writes the house navy as #222660 where the Docs API reports the #22265F house.yaml states",
}

// Compare reads the two sides row by row. The lists come from one Items list,
// so a name that does not line up is a programming error rather than a
// difference, and it is reported as one instead of being compared.
func Compare(a, b []Value) []Row {
	rows := make([]Row, 0, len(a))
	for i, av := range a {
		if i >= len(b) {
			rows = append(rows, Row{Item: av.Name, A: av.V, Verdict: Missing, Bug: true,
				Note: "the two readings are different lengths, which is a bug in this package rather than a difference between the documents"})
			continue
		}
		bv := b[i]
		if av.Name != bv.Name {
			rows = append(rows, Row{Item: av.Name, A: av.V, B: bv.V, Verdict: Different, Bug: true,
				Note: fmt.Sprintf("this row was read as %q on the other side, which is a bug in this package", bv.Name)})
			continue
		}
		v, note, bug := verdict(av.V, bv.V, tolerance(av.Tol))
		rows = append(rows, Row{Item: av.Name, A: av.V, B: bv.V, Verdict: v, Note: note, Bug: bug})
	}
	for _, bv := range b[min(len(a), len(b)):] {
		rows = append(rows, Row{Item: bv.Name, B: bv.V, Verdict: Missing, Bug: true,
			Note: "the two readings are different lengths, which is a bug in this package rather than a difference between the documents"})
	}
	return rows
}

// tolerance fills in the default for an item that named none. Zero is a real
// tolerance, which line spacing uses, so an item that wants it says -1 and
// nothing else can spell it by accident.
func tolerance(tol float64) float64 {
	if tol == 0 {
		return DefaultTol
	}
	if tol < 0 {
		return 0
	}
	return tol
}

// verdict is compare.py's rule, with one addition: a list of numbers is judged
// on its worst member, which is what compare.py did by hand for the column
// widths. The note is that worst difference, so a CLOSE row says how close.
//
// The third return says the fault is in this package. Two halves of one row are
// one question asked two ways, so they answer in one type; when they do not,
// the last arm used to compare their two spellings and call "1" and 1.0 the
// same answer. That is the same kind of thing as a name that does not line up,
// and it is reported the same way.
func verdict(a, b any, tol float64) (Verdict, string, bool) {
	if a == nil && b == nil {
		return Identical, "", false
	}
	if a == nil || b == nil {
		return Missing, "", false
	}
	an, aok := number(a)
	bn, bok := number(b)
	if aok && bok {
		return numberVerdict(math.Abs(an-bn), tol), "", false
	}
	al, aok := numbers(a)
	bl, bok := numbers(b)
	if aok && bok {
		if len(al) != len(bl) {
			return Different, fmt.Sprintf("%d values against %d", len(al), len(bl)), false
		}
		worst := 0.0
		for i := range al {
			worst = math.Max(worst, math.Abs(al[i]-bl[i]))
		}
		// Three decimals, because the differences this catches are the ones two
		// writers make out of the same number: 0.00pt reads as no difference at
		// all on a row that is not identical.
		return numberVerdict(worst, tol), fmt.Sprintf("worst difference %.3fpt", worst), false
	}
	if ta, tb := reflect.TypeOf(a), reflect.TypeOf(b); ta != tb {
		return Different, fmt.Sprintf("this row was read as %s on one side and %s on the other, which is a bug in this package", ta, tb), true
	}
	if fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b) {
		return Identical, "", false
	}
	return Different, "", false
}

// numberVerdict turns one distance into a verdict.
func numberVerdict(diff, tol float64) Verdict {
	switch {
	case diff < equal:
		return Identical
	case diff <= tol:
		return Close
	default:
		return Different
	}
}

// number reads one value as a number, and says whether it was one. A bool is
// not: true and false compare as words, so a bool read as 1 and 0 would come
// back CLOSE under a tolerance of 0.75.
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	}
	return 0, false
}

// numbers reads a list of numbers, which is what a table's column widths are.
func numbers(v any) ([]float64, bool) {
	switch l := v.(type) {
	case []float64:
		return l, true
	case nil:
		return nil, false
	}
	return nil, false
}

// Table is the report, one row per item, in the order the items are listed.
// It is printed on a failure, so the row that failed is read beside the rows
// that held.
func Table(rows []Row) string {
	var b strings.Builder
	width := len("item")
	for _, r := range rows {
		if len(r.Item) > width {
			width = len(r.Item)
		}
	}
	fmt.Fprintf(&b, "%-*s  %-10s  %s\n", width, "item", "verdict", "built vs master")
	for _, r := range rows {
		line := fmt.Sprintf("%-*s  %-10s  %s vs %s", width, r.Item, r.Verdict, show(r.A), show(r.B))
		if r.Note != "" {
			line += "  (" + r.Note + ")"
		}
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}
	return b.String()
}

// Summary counts the verdicts, which is the one line the 2026-08-29 report
// opened with.
func Summary(rows []Row) string {
	counts := map[Verdict]int{}
	for _, r := range rows {
		counts[r.Verdict]++
	}
	names := make([]string, 0, len(counts))
	for v := range counts {
		names = append(names, string(v))
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, fmt.Sprintf("%s %d", n, counts[Verdict(n)]))
	}
	return fmt.Sprintf("%d items: %s", len(rows), strings.Join(parts, ", "))
}

// Unexplained is what a gate fails on: the rows that disagree and are not in
// Known. It is a list rather than a count, because the failure has to name what
// moved.
func Unexplained(rows []Row) []Row {
	out := []Row{}
	for _, r := range rows {
		if r.Verdict != Different && r.Verdict != Missing {
			continue
		}
		// Known explains a difference between the two documents. A row whose
		// fault is in this package is not one, and ten of the names in Known
		// carry rows that can go wrong that way, so filtering on the name alone
		// dropped them and the gate passed in silence.
		if _, ok := Known[r.Item]; ok && !r.Bug {
			continue
		}
		out = append(out, r)
	}
	return out
}

// show prints one side of a row. A nil is spelled out, because an empty column
// reads as a value nobody wrote down.
func show(v any) string {
	if v == nil {
		return "<none>"
	}
	if s, ok := v.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%v", v)
}
