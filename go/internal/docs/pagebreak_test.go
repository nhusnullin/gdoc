package docs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// read is one of this package's fixtures as a document.
func read(t *testing.T, name string) *Document {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%s): %v", name, err)
	}
	return d
}

// TestAParagraphKeepsWhetherThePageBreaksBeforeIt is M19's half of the read.
//
// The fixture is the shape the live measurement of 2026-10-16 found
// (MEASURED.md, "How read prints the two kinds of page break"): Drive's import
// of a docx states a page break as pageBreakBefore on the paragraph's own
// style and not as a pageBreak element, so the flag is the only fact in the
// answer that says where the page turns. Two paragraphs carry it here, the
// empty NORMAL_TEXT one the body's own `---` became and the HEADING_1 the
// house breaks before, and the rest of the document carries it nowhere.
func TestAParagraphKeepsWhetherThePageBreaksBeforeIt(t *testing.T) {
	d := read(t, "page-breaks.json")
	if len(d.Tabs) != 1 {
		t.Fatalf("the fixture read as %d tabs, want 1", len(d.Tabs))
	}
	body := d.Tabs[0].Body

	want := map[int]bool{0: true, 2: true}
	for i, b := range body {
		if b.Paragraph == nil {
			t.Fatalf("block %d is not a paragraph", i)
		}
		if got := b.Paragraph.PageBreakBefore; got != want[i] {
			t.Errorf("paragraph %d (%s) carries PageBreakBefore %t, want %t",
				i, b.Paragraph.Style, got, want[i])
		}
	}
	if body[0].Paragraph.Style != "HEADING_1" {
		t.Errorf("the first paragraph is styled %q, and the heading's own break is the one it carries", body[0].Paragraph.Style)
	}
	if runs := body[2].Paragraph.Runs; len(runs) != 1 || runs[0].Text != "\n" {
		t.Errorf("the break paragraph holds %d runs, want the one newline the measurement found", len(runs))
	}
}

// TestTheStructurePrintCarriesThePageBreakFlag holds the field to the print
// `read --structure` makes. The structure is docs's own tags, so a field with
// no tag reaches no caller, and a field without omitempty says "false" on
// every paragraph in the document.
func TestTheStructurePrintCarriesThePageBreakFlag(t *testing.T) {
	d := read(t, "page-breaks.json")
	out, err := json.Marshal(d.Tabs[0])
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), `"page_break_before":true`); n != 2 {
		t.Errorf("the structure print names page_break_before %d times, want the 2 paragraphs that carry it:\n%s", n, out)
	}
	if strings.Contains(string(out), `"page_break_before":false`) {
		t.Error("the structure print says page_break_before on a paragraph the page does not break before")
	}
}
