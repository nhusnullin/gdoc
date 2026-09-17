package update

import (
	"strings"
	"testing"
)

func TestAVersionIsThreeNumbersAfterAV(t *testing.T) {
	cases := []struct {
		in    string
		major int
		minor int
		patch int
	}{
		{"v2.1.3", 2, 1, 3},
		{"v2.0.0", 2, 0, 0},
		{"v10.20.30", 10, 20, 30},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.in, err)
		}
		if got.Major != c.major || got.Minor != c.minor || got.Patch != c.patch {
			t.Errorf("Parse(%q) = %d.%d.%d, want %d.%d.%d",
				c.in, got.Major, got.Minor, got.Patch, c.major, c.minor, c.patch)
		}
		if got.String() != c.in {
			t.Errorf("Parse(%q).String() = %q, want the same text back", c.in, got.String())
		}
	}
}

func TestAVersionThatIsNotThreeNumbersIsRefusedByName(t *testing.T) {
	cases := []struct {
		in   string
		says string
	}{
		{"", "is empty"},
		{"dev", "opens with a v"},
		{"2.1.3", "opens with a v"},
		{"V2.1.3", "opens with a v"},
		{"vv2.1.3", "three numbers"},
		{"v2.1", "three numbers"},
		{"v2.1.3.4", "three numbers"},
		{"v2.1.x", "three numbers"},
		{"v2..3", "three numbers"},
		{"v-1.2.3", "three numbers"},
		{"v2.-1.3", "three numbers"},
		{"v 2.1.3", "three numbers"},
		{"v2.1.3 ", "three numbers"},
		{"v+2.1.3", "three numbers"},
		{"v01.2.3", "leading zero"},
		{"v2.1.03", "leading zero"},
		{"v2.1.3-", "no pre-release"},
		{"v2.0.0-rc1", "no pre-release"},
		{"v2.0.0-dirty", "no pre-release"},
		{"v2.1.3-rc 1", "no pre-release"},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err == nil {
			t.Errorf("Parse(%q) = %v, want a refusal", c.in, got)
			continue
		}
		if !strings.Contains(err.Error(), c.says) {
			t.Errorf("Parse(%q) said %q, want it to name %q", c.in, err, c.says)
		}
	}
}

func TestCompareOrdersTheVersionsThePolicyReadsAbout(t *testing.T) {
	cases := []struct {
		a    string
		b    string
		want int
	}{
		{"v2.0.0", "v2.0.0", 0},
		{"v2.0.0", "v2.0.1", -1},
		{"v2.0.1", "v2.0.0", 1},
		{"v2.0.3", "v2.1.0", -1},
		{"v2.1.0", "v3.0.0", -1},
		{"v3.0.0", "v2.9.9", 1},
		{"v10.0.0", "v9.0.0", 1},
		{"v2.0.10", "v2.0.9", 1},
	}
	for _, c := range cases {
		a, b := mustParse(t, c.a), mustParse(t, c.b)
		if got := Compare(a, b); got != c.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestTheZeroVersionIsHowNothingFoundIsSaid(t *testing.T) {
	var none Version
	if !none.IsZero() {
		t.Error("the zero Version does not report itself as zero")
	}
	if mustParse(t, "v2.0.0").IsZero() {
		t.Error("v2.0.0 reports itself as the zero Version")
	}
	if none.String() != "" {
		t.Errorf("the zero Version prints %q, want the empty string", none.String())
	}
	// v0.0.0 parses to the same value, and this pins that it does. gdoc's
	// first release is v2.0.0 and its tags only go up, so the one thing a
	// caller could confuse with nothing found is a tag that will never exist.
	if !mustParse(t, "v0.0.0").IsZero() {
		t.Error("v0.0.0 and the zero Version have stopped being the same value; say which is which in the doc comment")
	}
}

func TestAStableVersionIsOneWhosePatchIsZero(t *testing.T) {
	if !mustParse(t, "v2.1.0").IsStable() {
		t.Error("v2.1.0 is not reported as stable")
	}
	if mustParse(t, "v2.1.3").IsStable() {
		t.Error("v2.1.3, a nightly, is reported as stable")
	}
}

func mustParse(t *testing.T, s string) Version {
	t.Helper()
	v, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return v
}
