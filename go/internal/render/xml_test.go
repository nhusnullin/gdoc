package render

import "testing"

// The units are the whole reason house.yaml holds points and nothing else.
// One value in the file has one meaning, and every conversion happens here.

func TestPointsBecomeTwipsHalfPointsEighthsAndEMU(t *testing.T) {
	if got := twips(51.05); got != "1021" {
		t.Errorf("twips(51.05) = %s, want 1021", got)
	}
	if got := twips(841.89); got != "16838" {
		t.Errorf("twips(841.89) = %s, want 16838", got)
	}
	if got := halfPoints(16); got != "32" {
		t.Errorf("halfPoints(16) = %s, want 32", got)
	}
	if got := eighths(0.5); got != "4" {
		t.Errorf("eighths(0.5) = %s, want 4", got)
	}
	if got := emu(146.25); got != "1857375" {
		t.Errorf("emu(146.25) = %s, want 1857375", got)
	}
	if got := emu(-5.15); got != "-65405" {
		t.Errorf("emu(-5.15) = %s, want -65405", got)
	}
	if got := lineTwentyFourths(1.15); got != "276" {
		t.Errorf("lineTwentyFourths(1.15) = %s, want 276", got)
	}
}

func TestAColourIsWrittenWithoutItsHash(t *testing.T) {
	if got := hexColor("#22265F"); got != "22265f" {
		t.Errorf("hexColor(#22265F) = %s, want 22265f", got)
	}
	if got := hexColor(""); got != "000000" {
		t.Errorf("hexColor(\"\") = %s, want 000000", got)
	}
}

func TestAnAlignmentTheHouseDoesNotHaveIsRefusedByName(t *testing.T) {
	b := &builder{}
	if got := b.align("justify"); got != "both" {
		t.Errorf("align(justify) = %q, want both", got)
	}
	if b.err != nil {
		t.Fatalf("a house alignment was refused: %v", b.err)
	}
	b.align("sideways")
	if b.err == nil {
		t.Fatal("an alignment that is not a Word alignment was accepted")
	}
	if got := b.err.Error(); got == "" || !contains(got, "sideways") {
		t.Errorf("the error does not name the alignment: %v", b.err)
	}
}

func TestTheFirstFailureIsTheOneReported(t *testing.T) {
	b := &builder{}
	b.fail("the first thing")
	b.fail("the second thing")
	if got := b.err.Error(); got != "the first thing" {
		t.Errorf("the reported failure is %q, want the first thing", got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
