package body

import (
	"strings"
	"testing"
)

// numIDs is every w:numId a walk wrote, in the order the paragraphs carry
// them, so a test can say which list each item named.
func numIDs(t *testing.T, out Result) []string {
	t.Helper()
	var ids []string
	for _, block := range out.Blocks {
		for _, e := range findAll(block, "w:numId") {
			ids = append(ids, attr(t, e, "w:val"))
		}
	}
	return ids
}

// TestASecondNumberedListStartsAgain: each top-level numbered list has its own
// definition in numbering.xml, so the second one opens at 1 rather than
// carrying on from the first.
func TestASecondNumberedListStartsAgain(t *testing.T) {
	out := walk(t, "1. a\n2. b\n\ntext\n\n1. c\n2. d\n")

	want := []string{"2", "2", "3", "3"}
	got := numIDs(t, out)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("the items name lists %v, want %v", got, want)
	}
	if out.NumberedLists != 2 {
		t.Errorf("the walk counted %d numbered lists, want 2", out.NumberedLists)
	}
	for _, warning := range out.Warnings {
		if strings.Contains(warning, "carries on") {
			t.Errorf("a second numbered list still warns: %q", warning)
		}
	}
}

// TestANestedNumberedListNamesItsParent: a numbered list inside a numbered
// item is one list as Word counts them, because an absent w:lvlRestart already
// restarts the inner level whenever the outer one moves.
func TestANestedNumberedListNamesItsParent(t *testing.T) {
	out := walk(t, "1. a\n   1. inner\n2. b\n")

	for i, id := range numIDs(t, out) {
		if id != "2" {
			t.Errorf("item %d names list %s, want 2", i, id)
		}
	}
	if out.NumberedLists != 1 {
		t.Errorf("the walk counted %d numbered lists, want 1", out.NumberedLists)
	}
}

// TestANumberedListUnderABulletOpensItsOwn: a bullet is not a number, so a
// numbered list nested in one is the first numbered list of the document and a
// top-level numbered list after it is the second.
func TestANumberedListUnderABulletOpensItsOwn(t *testing.T) {
	out := walk(t, "- a\n   1. inner\n\ntext\n\n1. c\n")

	want := []string{"1", "2", "3"}
	got := numIDs(t, out)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("the items name lists %v, want %v", got, want)
	}
	if out.NumberedLists != 2 {
		t.Errorf("the walk counted %d numbered lists, want 2", out.NumberedLists)
	}
}

// TestAListThatStartsElsewhereStillSaysSo: every list's own w:num already
// states w:startOverride 1, and honouring the author's own start number is that
// same override carrying their number, which gdoc does not write, so a list
// that opens at 5 still opens at 1 in the document and still says so.
func TestAListThatStartsElsewhereStillSaysSo(t *testing.T) {
	out := walk(t, "5. a\n")

	var got string
	for _, warning := range out.Warnings {
		if strings.Contains(warning, "numbered list") {
			got = warning
		}
	}
	if !strings.Contains(got, "starts at 5 in the note and at 1 in the document") {
		t.Fatalf("the warning is %q, warnings %q", got, out.Warnings)
	}
}
