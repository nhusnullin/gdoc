package view

import (
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// TestProjectSkipsAndNamesPictures is the one door beside Text: a caller may
// drop a block before it is projected and say what a picture is written as, and
// a caller asking for neither gets exactly what Text gives.
func TestProjectSkipsAndNamesPictures(t *testing.T) {
	t.Run("the zero options are Text", func(t *testing.T) {
		for _, name := range []string{"single-tab", "two-tabs", "pre-tabs", "objects", "elements", "links", "lists", "positioned"} {
			d := fixture(t, name+".json")
			want, wantWarnings := Text(d)
			got, gotWarnings := Project(d, Options{})
			if got != want {
				t.Errorf("%s: Project with no options =\n%s\nwant\n%s", name, got, want)
			}
			if len(gotWarnings) != len(wantWarnings) {
				t.Errorf("%s: Project raised %d warnings, Text raised %d", name, len(gotWarnings), len(wantWarnings))
			}
		}
	})

	t.Run("skip drops a block", func(t *testing.T) {
		d := fixture(t, "positioned.json")
		first := d.Tabs[0].Body[0]
		got, _ := Project(d, Options{Skip: func(b docs.Block) bool { return b.Paragraph == first.Paragraph }})
		if strings.Contains(got, "The chart sits beside this.") {
			t.Errorf("the skipped paragraph is still in the text:\n%s", got)
		}
		if !strings.Contains(got, "Plain prose.") {
			t.Errorf("the blocks that were not skipped are gone too:\n%s", got)
		}
	})

	t.Run("a floating picture is named after its placeholder", func(t *testing.T) {
		d := fixture(t, "positioned.json")
		got, warnings := Project(d, Options{Picture: func(id string) string { return "![](assets/n-" + id + ".png)" }})
		want := "<!-- image: floating, kix.posone -->\n\n![](assets/n-kix.posone.png)"
		if !strings.Contains(got, want) {
			t.Errorf("the floating picture's line is not under its placeholder:\n%s", got)
		}
		if len(warnings) != 2 {
			t.Errorf("warnings = %v, want one per floating object", warnings)
		}
	})

	t.Run("an inline picture is written raw and raises no warning", func(t *testing.T) {
		d := fixture(t, "objects.json")
		got, warnings := Project(d, Options{Picture: func(id string) string {
			if id == "kix.obj1" {
				return ""
			}
			return "![](assets/n-" + id + ".png)"
		}})
		for _, want := range []string{"![](assets/n-kix.img1.png)", "![](assets/n-kix.draw1.png)"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s is not in the text:\n%s", want, got)
			}
		}
		if strings.Contains(got, "[image]") || strings.Contains(got, "[drawing]") {
			t.Errorf("a picture that was named still prints its placeholder:\n%s", got)
		}
		if !strings.Contains(got, "[object]") {
			t.Errorf("the object nothing was named for lost its placeholder:\n%s", got)
		}
		for _, w := range warnings {
			if strings.Contains(w, "an image") || strings.Contains(w, "a drawing") {
				t.Errorf("a picture that was written is still a warning: %s", w)
			}
		}
	})
}

// TestAChipCarriesItsTargetWhenAskedFor is the other half of the door: the read
// prints a chip's label alone, and a projection that becomes a file in the hub
// carries the address the chip points at as well.
func TestAChipCarriesItsTargetWhenAskedFor(t *testing.T) {
	d := fixture(t, "elements.json")

	plain, _ := Project(d, Options{})
	if strings.Contains(plain, "](") {
		t.Errorf("read prints a chip's target:\n%s", plain)
	}

	got, _ := Project(d, Options{ChipTargets: true})
	for _, want := range []string{
		"[link: A placeholder calendar entry](https://calendar.example.com/event/placeholder)",
		"[person: A Placeholder](mailto:placeholder@example.com)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%s is not in the text:\n%s", want, got)
		}
	}
	if strings.Contains(got, "[date: Sep 9, 2026](") {
		t.Errorf("a date chip points somewhere:\n%s", got)
	}
}
