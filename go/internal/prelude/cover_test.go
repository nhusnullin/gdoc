package prelude

import (
	"strings"
	"testing"

	"gdoc/internal/cover"
	"gdoc/internal/house"
)

// The house values below are written out as numbers on purpose. A test that
// reads cfg.Cover.Lines[2].SizePt is a mirror: move the constant and the
// assertion follows it and still passes. These say what the Altery cover is.

func testFields() cover.Fields {
	return cover.Fields{
		Title: "Third Party Risk", DocType: "Policy",
		Version: "1.0", Date: "May 2026", Owner: "The Board",
		Classification: "Internal (I)",
	}
}

func testConfig(t *testing.T) *house.Config {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("house.Load() = %v", err)
	}
	return cfg
}

// texts is the paragraphs the requests insert, in the order they insert them.
func texts(requests []map[string]any) []string {
	var out []string
	for _, r := range requests {
		insert, ok := r["insertText"].(map[string]any)
		if !ok {
			continue
		}
		out = append(out, strings.TrimSuffix(insert["text"].(string), "\n"))
	}
	return out
}

// styleAt is the one style request of a kind covering the given range, and
// whether there was one.
func styleAt(requests []map[string]any, kind string, start, end int) map[string]any {
	for _, r := range requests {
		body, ok := r[kind].(map[string]any)
		if !ok {
			continue
		}
		span, ok := body["range"].(map[string]any)
		if !ok {
			continue
		}
		if span["startIndex"] == start && span["endIndex"] == end {
			return body
		}
	}
	return nil
}

// paragraphSpans is every paragraph the requests insert, as start and end.
func paragraphSpans(requests []map[string]any) [][2]int {
	var out [][2]int
	for _, r := range requests {
		body, ok := r["updateParagraphStyle"].(map[string]any)
		if !ok {
			continue
		}
		span := body["range"].(map[string]any)
		out = append(out, [2]int{span["startIndex"].(int), span["endIndex"].(int)})
	}
	return out
}

func TestTheCoverIsTheHouseCoverLineByLine(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := Cover(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("Cover() = %v", err)
	}

	// Assert
	want := []string{
		"", "", "", "", "", // the five leading blanks
		"Altery Group ",
		"", // the blank line under it, which the master sets at 29pt
		"Third Party Risk Policy",
		"Version: 1.0",
		"May 2026",
		"", "", "", "", "", "", "", "", // the eight trailing blanks
		"", // the page break's own paragraph
	}
	if got := texts(got.Requests); !equal(got, want) {
		t.Errorf("the cover reads\n%q\nwant\n%q", got, want)
	}
	if got.Paragraphs != len(want) {
		t.Errorf("Paragraphs = %d, want %d", got.Paragraphs, len(want))
	}
}

func TestTheCoverLinesAreCentredAtTheHouseSizes(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := Cover(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("Cover() = %v", err)
	}

	// Assert: the title line, which is the ninth paragraph the cover writes.
	spans := paragraphSpans(got.Requests)
	title := spans[7]
	style := styleAt(got.Requests, "updateParagraphStyle", title[0], title[1])
	if style == nil {
		t.Fatalf("the title line states no paragraph style")
	}
	set := style["paragraphStyle"].(map[string]any)
	if set["alignment"] != "CENTER" {
		t.Errorf("the title line is aligned %v, want CENTER", set["alignment"])
	}
	if set["namedStyleType"] != "NORMAL_TEXT" {
		t.Errorf("the title line is %v, want NORMAL_TEXT", set["namedStyleType"])
	}
	text := styleAt(got.Requests, "updateTextStyle", title[0], title[1]-1)
	if text == nil {
		t.Fatalf("the title line states no text style")
	}
	look := text["textStyle"].(map[string]any)
	if magnitude(look["fontSize"]) != 29 {
		t.Errorf("the title is %v pt, want 29", magnitude(look["fontSize"]))
	}
	if look["bold"] != true {
		t.Errorf("the title is not bold")
	}
	if look["weightedFontFamily"].(map[string]any)["fontFamily"] != "Calibri" {
		t.Errorf("the title face is %v, want Calibri", look["weightedFontFamily"])
	}

	// The version and the date are 20pt, and the two blanks under the third
	// leading one are 18pt.
	version := spans[8]
	label := styleAt(got.Requests, "updateTextStyle", version[0], version[0]+len("Version: "))
	if label == nil || magnitude(label["textStyle"].(map[string]any)["fontSize"]) != 20 {
		t.Errorf("the version label is not 20pt: %v", label)
	}
	fourth := spans[3]
	blank := styleAt(got.Requests, "updateTextStyle", fourth[0], fourth[1])
	if blank == nil || magnitude(blank["textStyle"].(map[string]any)["fontSize"]) != 18 {
		t.Errorf("the fourth leading blank is not 18pt: %v", blank)
	}
	// A blank the file states no size for takes the house default, which is
	// 11pt, rather than whatever the paragraph it was inserted beside carries.
	first := spans[0]
	blank = styleAt(got.Requests, "updateTextStyle", first[0], first[1])
	if blank == nil || magnitude(blank["textStyle"].(map[string]any)["fontSize"]) != 11 {
		t.Errorf("the first leading blank is not 11pt: %v", blank)
	}
}

func TestTheAlternativeTitleLinesArriveOnlyWithAnAlternativeTitle(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	f := testFields()
	f.AltTitle = "Supplier Risk"

	// Act
	got, err := Cover(cfg, f, 1)
	if err != nil {
		t.Fatalf("Cover() = %v", err)
	}

	// Assert
	lines := texts(got.Requests)
	if !contains(lines, "or") || !contains(lines, "Supplier Risk Policy") {
		t.Errorf("the alternative title is not on the cover: %q", lines)
	}
	// And without one, neither line is written: the master offers the title
	// twice for a person to pick one, and printing both published a cover
	// reading the title, then "or", then the template's own placeholder.
	got, err = Cover(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("Cover() = %v", err)
	}
	if contains(texts(got.Requests), "or") {
		t.Errorf("a cover with one title carries the master's \"or\": %q", texts(got.Requests))
	}
}

func TestTheVersionLabelIsARunAndOnlyTheNumberIsTheFields(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	f := testFields()
	f.Version = "2.3"

	// Act
	got, err := Cover(cfg, f, 1)
	if err != nil {
		t.Fatalf("Cover() = %v", err)
	}

	// Assert
	if !contains(texts(got.Requests), "Version: 2.3") {
		t.Errorf("the version line reads %q, want a label and the number", texts(got.Requests))
	}
	span := paragraphSpans(got.Requests)[8]
	number := styleAt(got.Requests, "updateTextStyle", span[0]+len("Version: "), span[1]-1)
	if number == nil {
		t.Fatalf("the version number is not a run of its own")
	}
	look := number["textStyle"].(map[string]any)
	// A filled placeholder loses the template's marks: the yellow means "a
	// person fills this in", and once the number is there it is misleading.
	if _, marked := look["backgroundColor"].(map[string]any)["color"]; marked {
		t.Errorf("the version number kept the template's highlight: %v", look)
	}
}

func TestAnUnfilledPlaceholderKeepsTheTemplatesMarks(t *testing.T) {
	// Arrange: one line whose placeholder the fields leave empty.
	cfg := markedConfig()

	// Act
	got, err := Cover(cfg, cover.Fields{Title: "A Policy"}, 1)
	if err != nil {
		t.Fatalf("Cover() = %v", err)
	}

	// Assert
	if !contains(texts(got.Requests), "(Name of)Framework") {
		t.Fatalf("the template's own words are not on the cover: %q", texts(got.Requests))
	}
	span := paragraphSpans(got.Requests)[0]
	first := styleAt(got.Requests, "updateTextStyle", span[0], span[0]+len("(Name of)"))
	if first == nil {
		t.Fatalf("the highlighted run states no text style")
	}
	rgb := first["textStyle"].(map[string]any)["backgroundColor"].(map[string]any)
	if channel(rgb, "red") != 1 || channel(rgb, "green") != 1 || channel(rgb, "blue") != 0 {
		t.Errorf("the run is highlighted %v, want the yellow the master carries", rgb)
	}
	second := styleAt(got.Requests, "updateTextStyle", span[0]+len("(Name of)"), span[1]-1)
	if second == nil {
		t.Fatalf("the red run states no text style")
	}
	fg := second["textStyle"].(map[string]any)["foregroundColor"].(map[string]any)
	if channel(fg, "red") != 1 || channel(fg, "green") != 0 || channel(fg, "blue") != 0 {
		t.Errorf("the guidance run is %v, want the red the master carries", fg)
	}
}

func TestAFilledPlaceholderLosesTheTemplatesMarks(t *testing.T) {
	// Arrange
	cfg := markedConfig()
	f := cover.Fields{Title: "A Policy", Distribution: "All staff"}

	// Act
	got, err := Cover(cfg, f, 1)
	if err != nil {
		t.Fatalf("Cover() = %v", err)
	}

	// Assert
	if !contains(texts(got.Requests), "All staff") {
		t.Fatalf("the fields' own words are not on the cover: %q", texts(got.Requests))
	}
	span := paragraphSpans(got.Requests)[0]
	run := styleAt(got.Requests, "updateTextStyle", span[0], span[1]-1)
	if run == nil {
		t.Fatalf("the filled line states no text style")
	}
	look := run["textStyle"].(map[string]any)
	if _, marked := look["backgroundColor"].(map[string]any)["color"]; marked {
		t.Errorf("the filled line kept the template's highlight: %v", look)
	}
	fg := look["foregroundColor"].(map[string]any)
	if channel(fg, "red") != 0 || channel(fg, "green") != 0 || channel(fg, "blue") != 0 {
		t.Errorf("the filled line is %v, want the black the normal style states", fg)
	}
}

func TestAClassificationCellIsShadedOnlyWhenTheFieldsDeclareThatClass(t *testing.T) {
	// Arrange
	internal := house.Cell{Fill: "#D9D9D9", Classification: "Internal (I)"}
	public := house.Cell{Fill: "#D9D9D9", Classification: "Public (P)"}
	plain := house.Cell{Fill: "#F2F2F2"}

	// Act and assert
	if got := cellFill(internal, testFields()); got != "#D9D9D9" {
		t.Errorf("the declared class is shaded %q, want #D9D9D9", got)
	}
	if got := cellFill(public, testFields()); got != "" {
		t.Errorf("a class the fields do not declare is shaded %q, want no fill", got)
	}
	if got := cellFill(plain, testFields()); got != "#F2F2F2" {
		t.Errorf("a cell describing no class is shaded %q, want #F2F2F2", got)
	}
}

func TestTheCoverEndsWithAPageBreak(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := Cover(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("Cover() = %v", err)
	}

	// Assert: the break is inside gdoc's own last paragraph, so what follows
	// starts at the top of the next page and the author's own paragraph is
	// left as it was.
	breaks := 0
	var at int
	for _, r := range got.Requests {
		if body, ok := r["insertPageBreak"].(map[string]any); ok {
			breaks++
			at = body["location"].(map[string]any)["index"].(int)
		}
	}
	if breaks != 1 {
		t.Fatalf("the cover carries %d page breaks, want 1", breaks)
	}
	last := paragraphSpans(got.Requests)[len(paragraphSpans(got.Requests))-1]
	if at != last[0] || last[1] != last[0]+2 {
		t.Errorf("the break sits at %d and its paragraph is %v, want the break first in the last paragraph", at, last)
	}
	if got.End != last[1] {
		t.Errorf("End = %d, want %d, one past the last character the prelude inserts", got.End, last[1])
	}
}

func TestTheRequestsInsertOneAfterAnotherFromWhereTheyWereTold(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act: a prelude that starts at 1, which is where a document's body
	// begins, and one that starts further in.
	for _, start := range []int{1, 42} {
		got, err := Cover(cfg, testFields(), start)
		if err != nil {
			t.Fatalf("Cover(start=%d) = %v", start, err)
		}

		// Assert
		if got.Start != start {
			t.Errorf("Start = %d, want %d", got.Start, start)
		}
		at := start
		for _, span := range paragraphSpans(got.Requests) {
			if span[0] != at {
				t.Fatalf("a paragraph opens at %d, want %d: the indexes are not contiguous", span[0], at)
			}
			if span[1] <= span[0] {
				t.Fatalf("a paragraph spans %v, which ends where it starts or before it", span)
			}
			at = span[1]
		}
		if got.End != at {
			t.Errorf("End = %d, want %d", got.End, at)
		}
	}
}

func TestEveryMaskNamesExactlyWhatTheRequestSets(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := Cover(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("Cover() = %v", err)
	}

	// Assert: the Docs reference says a field named in a mask and left unset
	// is a property reset to its default, so a mask naming more than the
	// request sets destroys formatting, and one naming less is a value the
	// server ignores.
	kinds := map[string]string{
		"updateParagraphStyle": "paragraphStyle",
		"updateTextStyle":      "textStyle",
	}
	seen := 0
	for _, r := range got.Requests {
		for kind, key := range kinds {
			body, ok := r[kind].(map[string]any)
			if !ok {
				continue
			}
			seen++
			set := body[key].(map[string]any)
			mask := strings.Split(body["fields"].(string), ",")
			if len(mask) != len(set) {
				t.Errorf("%s names %d fields and sets %d: %v against %v", kind, len(mask), len(set), mask, set)
			}
			for _, name := range mask {
				if _, ok := set[name]; !ok {
					t.Errorf("%s names %q in its mask and does not set it", kind, name)
				}
			}
			if body["fields"] == "" || body["fields"] == "*" {
				t.Errorf("%s carries the mask %q, which Docs reads as every field", kind, body["fields"])
			}
		}
	}
	if seen == 0 {
		t.Fatalf("no style request was checked")
	}
}

func TestAPlaceholderThatIsNotACoverFieldStopsTheCover(t *testing.T) {
	// Arrange
	cfg := markedConfig()
	cfg.Cover.Lines[0].Placeholder = "approver"

	// Act
	_, err := Cover(cfg, testFields(), 1)

	// Assert
	if err == nil {
		t.Fatalf("a cover naming a placeholder gdoc does not hold was built")
	}
	if !strings.Contains(err.Error(), "approver") {
		t.Errorf("the refusal does not name the placeholder: %v", err)
	}
}

func TestAnAlignmentTheDocsAPIDoesNotHaveStopsTheCover(t *testing.T) {
	// Arrange
	cfg := markedConfig()
	cfg.Cover.Align = "middle"

	// Act
	_, err := Cover(cfg, testFields(), 1)

	// Assert
	if err == nil {
		t.Fatalf("a cover aligned in a way Docs has no word for was built")
	}
	if !strings.Contains(err.Error(), "middle") {
		t.Errorf("the refusal does not name the alignment: %v", err)
	}
}

// markedConfig is one cover line carrying the master's own two marks: the
// yellow a person fills in and the red that is guidance rather than content.
func markedConfig() *house.Config {
	size := 20.0
	return &house.Config{
		Defaults: house.Defaults{Font: "Calibri", SizePt: 11, LineSpacing: 1.15},
		Styles:   map[string]house.Style{"normal": {Color: "#000000"}},
		Cover: house.Cover{
			Align: "center",
			Lines: []house.CoverLine{{
				SizePt:      &size,
				Placeholder: "distribution",
				Runs: []house.Run{
					{Text: "(Name of)", Highlight: "yellow"},
					{Text: "Framework", Color: "#FF0000"},
				},
			}},
		},
	}
}

func magnitude(v any) float64 {
	dimension, ok := v.(map[string]any)
	if !ok {
		return -1
	}
	size, ok := dimension["magnitude"].(float64)
	if !ok {
		return -1
	}
	return size
}

func channel(optional map[string]any, name string) float64 {
	color, ok := optional["color"].(map[string]any)
	if !ok {
		return -1
	}
	rgb, ok := color["rgbColor"].(map[string]any)
	if !ok {
		return -1
	}
	v, ok := rgb[name].(float64)
	if !ok {
		return 0 // Docs leaves a channel of zero out, and so does the builder
	}
	return v
}

func contains(list []string, want string) bool {
	for _, got := range list {
		if got == want {
			return true
		}
	}
	return false
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
