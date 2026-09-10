package docsreq

import (
	"reflect"
	"testing"
)

func TestPointsIsTheDocsDimension(t *testing.T) {
	got := Points(29)
	want := map[string]any{"magnitude": 29.0, "unit": "PT"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Points(29) = %v, want %v", got, want)
	}
}

func TestPercentRoundsTheBinaryTail(t *testing.T) {
	// 1.15 times 100 is 114.99999999999999 in binary floating point, and a
	// request nobody can match to the value in the file is one nobody can
	// check.
	if got := Percent(1.15); got != 115 {
		t.Errorf("Percent(1.15) = %v, want 115", got)
	}
	if got := Percent(1.0); got != 100 {
		t.Errorf("Percent(1.0) = %v, want 100", got)
	}
}

func TestColorIsTheHouseHexAsThreeChannels(t *testing.T) {
	got, ok := Color("#FFFF00")
	if !ok {
		t.Fatalf("Color(#FFFF00) refused a house colour")
	}
	want := map[string]any{"color": map[string]any{"rgbColor": map[string]any{
		"red": 1.0, "green": 1.0, "blue": 0.0,
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Color(#FFFF00) = %v, want %v", got, want)
	}
}

func TestColorRefusesWhatIsNotSixDigitsOfHex(t *testing.T) {
	for _, hex := range []string{"", "#FFF", "yellow", "#GGGGGG", "#FFFF000"} {
		if _, ok := Color(hex); ok {
			t.Errorf("Color(%q) was carried, and Docs would reject the batch", hex)
		}
	}
}

func TestAlignmentReadsTheHouseWords(t *testing.T) {
	for word, want := range map[string]string{
		"left": "START", "center": "CENTER", "centre": "CENTER",
		"CENTER": "CENTER", "right": "END", "justify": "JUSTIFIED",
	} {
		got, ok := Alignment(word)
		if !ok || got != want {
			t.Errorf("Alignment(%q) = %q %v, want %q true", word, got, ok, want)
		}
	}
	if _, ok := Alignment("middle"); ok {
		t.Errorf("Alignment(middle) was carried, and Docs has no such alignment")
	}
	if _, ok := Alignment(""); ok {
		t.Errorf("Alignment(empty) was carried, and an unstated alignment is not written")
	}
}

func TestLenCountsTheUnitsTheDocsAPICounts(t *testing.T) {
	// The Docs API counts UTF-16 code units, so a non-BMP character is two.
	if got := Len("abc"); got != 3 {
		t.Errorf("Len(abc) = %d, want 3", got)
	}
	if got := Len("a\U0001F600b"); got != 4 {
		t.Errorf("Len of a string carrying an emoji = %d, want 4", got)
	}
}

func TestFieldsCarryTheMaskInTheOrderTheyWereSet(t *testing.T) {
	f := NewFields()
	if !f.Empty() {
		t.Errorf("a new Fields is not empty")
	}
	f.Put("fontSize", Points(11))
	f.Put("bold", true)
	if f.Mask() != "fontSize,bold" {
		t.Errorf("Mask() = %q, want %q", f.Mask(), "fontSize,bold")
	}
	if f.Empty() {
		t.Errorf("Fields carrying two values reads as empty")
	}
	want := map[string]any{"fontSize": Points(11), "bold": true}
	if !reflect.DeepEqual(f.Set, want) {
		t.Errorf("Set = %v, want %v", f.Set, want)
	}
}

func TestAMaskNamesExactlyWhatTheFieldsSet(t *testing.T) {
	f := NewFields()
	f.Put("alignment", "CENTER")
	f.Put("spaceAbove", Points(0))
	for _, name := range []string{"alignment", "spaceAbove"} {
		if _, ok := f.Set[name]; !ok {
			t.Errorf("the mask names %q and the object does not set it", name)
		}
	}
	if len(f.Set) != 2 {
		t.Errorf("the object sets %d values and the mask names 2", len(f.Set))
	}
}
