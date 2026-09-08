package body

import "testing"

func TestAuthoredNumberNeedsASeparatorOrADot(t *testing.T) {
	cases := []struct {
		heading string
		want    []int
	}{
		{"1. Key terms", []int{1}},
		{"1) Key terms", []int{1}},
		{"3.1 Sub-section", []int{3, 1}},
		{"10 things we learned", nil},
		{"2026 plan", nil},
		{"2.0 release notes", []int{2, 0}}, // a dotted number reads as one, and cannot be helped
		{"1.5x throughput", nil},           // no space after it, so it is not a section number
		{"Introduction", nil},
	}
	for _, c := range cases {
		got := authoredNumber(c.heading)
		if len(got) != len(c.want) {
			t.Errorf("authoredNumber(%q) = %v, want %v", c.heading, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("authoredNumber(%q) = %v, want %v", c.heading, got, c.want)
				break
			}
		}
	}
}

func TestPrefixesRunOnePerLevel(t *testing.T) {
	n := newHeadingNumberer(true, "-", 1)

	if got := n.prefix(1, "Purpose"); got != "1-" {
		t.Errorf("first heading = %q, want %q", got, "1-")
	}
	if got := n.prefix(2, "Scope"); got != "1.1-" {
		t.Errorf("second level = %q, want %q", got, "1.1-")
	}
	if got := n.prefix(2, "Exclusions"); got != "1.2-" {
		t.Errorf("second level again = %q, want %q", got, "1.2-")
	}
	if got := n.prefix(1, "Governance"); got != "2-" {
		t.Errorf("back to top = %q, want %q", got, "2-")
	}
	if got := n.prefix(2, "The register"); got != "2.1-" {
		t.Errorf("deeper counter must have restarted, got %q", got)
	}
}

func TestAnAuthoredNumberIsFollowedRatherThanDoubled(t *testing.T) {
	n := newHeadingNumberer(true, "-", 1)
	n.prefix(1, "Purpose") // 1-

	if got := n.prefix(1, "7. A heading that numbers itself"); got != "" {
		t.Errorf("a self-numbered heading must take no prefix, got %q", got)
	}
	if got := n.prefix(1, "The next one"); got != "8-" {
		t.Errorf("the sequence must carry on from the author, got %q, want %q", got, "8-")
	}
}

func TestAnAuthoredNumberAtTheWrongDepthMovesNothing(t *testing.T) {
	n := newHeadingNumberer(true, "-", 1)
	n.prefix(1, "Purpose") // 1-

	// Two parts at the top level is not this document's sequence.
	n.prefix(1, "4.2 A self-numbered heading at an odd depth")

	if got := n.prefix(1, "The last section"); got != "2-" {
		t.Errorf("got %q, want %q: an ill-fitting number must move nothing", got, "2-")
	}
}

func TestHeadingsThatNameThemselvesTakeNoNumber(t *testing.T) {
	n := newHeadingNumberer(true, "-", 1)
	for _, heading := range []string{"Appendix A", "Appendices", "Annexe 1", "Glossary",
		"Contents", "Schedule 2", "Addendum 1"} {
		if got := n.prefix(1, heading); got != "" {
			t.Errorf("prefix(%q) = %q, want none", heading, got)
		}
	}
}

func TestNumberingOffMeansNoPrefixAtAll(t *testing.T) {
	n := newHeadingNumberer(false, "-", 1)

	if got := n.prefix(1, "Purpose"); got != "" {
		t.Errorf("got %q, want no prefix when numbering is off", got)
	}
}

func TestTheShallowestHeadingBecomesHeadingOne(t *testing.T) {
	// A body lifted out of a note that already had its own title starts at "##".
	n := newHeadingNumberer(true, "-", 2)

	if got := n.styleLevel(2); got != 1 {
		t.Errorf("styleLevel(2) = %d, want 1 so it gets the 16pt Heading1", got)
	}
	if got := n.styleLevel(3); got != 2 {
		t.Errorf("styleLevel(3) = %d, want 2", got)
	}
	if got := n.prefix(2, "Purpose"); got != "1-" {
		t.Errorf("prefix = %q, want %q rather than %q", got, "1-", "0.1-")
	}
}

// TestTheSeparatorComesFromTheHouseFile: "{n}-{title}" is a house value, so
// the dash between the number and the title is read rather than assumed.
func TestTheSeparatorComesFromTheHouseFile(t *testing.T) {
	got, err := separator("{n}-{title}")
	if err != nil {
		t.Fatalf("separator: %v", err)
	}
	if got != "-" {
		t.Errorf("separator = %q, want %q", got, "-")
	}
	if got, err := separator("{n}. {title}"); err != nil || got != ". " {
		t.Errorf("separator = %q, %v, want %q", got, err, ". ")
	}
}

// A format that does not name both parts is refused: guessing at it would
// number every heading in a shape nobody wrote down.
func TestAFormatMissingItsPartsIsRefused(t *testing.T) {
	for _, format := range []string{"", "{n}", "{title}", "{title}-{n}"} {
		if _, err := separator(format); err == nil {
			t.Errorf("separator(%q) was accepted", format)
		}
	}
}
