package view

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// fixture parses one of internal/docs's testdata files. The projection is
// tested against the same fixtures the tree is, so a golden here says what the
// AI reads for a document the walk already has an assertion about.
func fixture(t *testing.T, name string) *docs.Document {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "docs", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("docs.Parse(%s): %v", name, err)
	}
	return d
}

func golden(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestGolden is the whole projection, one fixture at a time. Every rule under
// "The read text" in the plan is visible in one of these four files: headings,
// bullets and their nesting, the pipe table, the joined suggestion spans, the
// comment anchor, the escaped literal markers, the tab lines, the placeholders
// and the footnote.
func TestGolden(t *testing.T) {
	for _, name := range []string{"single-tab", "two-tabs", "pre-tabs", "objects"} {
		t.Run(name, func(t *testing.T) {
			got, _ := Text(fixture(t, name+".json"))
			if want := golden(t, name+".golden"); got != want {
				t.Errorf("Text() =\n%s\nwant\n%s", got, want)
			}
			if !strings.HasSuffix(got, "\n") || strings.HasSuffix(got, "\n\n") {
				t.Errorf("Text() does not end with exactly one newline: %q", got[max(0, len(got)-8):])
			}
		})
	}
}

func TestTheProjectionIsDeterministic(t *testing.T) {
	d := fixture(t, "single-tab.json")
	first, _ := Text(d)
	second, _ := Text(d)
	if first != second {
		t.Error("two reads of one document gave two strings")
	}
}

func TestOnlyAMultiTabDocumentCarriesTabLines(t *testing.T) {
	multi, _ := Text(fixture(t, "two-tabs.json"))
	if n := strings.Count(multi, "<!-- tab "); n != 2 {
		t.Errorf("two-tab document has %d tab lines, want 2", n)
	}
	for _, name := range []string{"single-tab.json", "pre-tabs.json"} {
		if one, _ := Text(fixture(t, name)); strings.Contains(one, "<!-- tab ") {
			t.Errorf("%s carries a tab line", name)
		}
	}
}

func TestAPlaceholderIsAWarning(t *testing.T) {
	_, warnings := Text(fixture(t, "objects.json"))
	want := []string{"[image]", "[drawing]", "[equation]"}
	for _, w := range want {
		found := false
		for _, got := range warnings {
			if strings.Contains(got, w) {
				found = true
			}
		}
		if !found {
			t.Errorf("no warning names %s: %v", w, warnings)
		}
	}
}

// A range the reader cannot place is not printed, and its id is a warning. The
// alternative is an opening marker with no close, which is worse than saying so.
func TestARangeOutsideTheTextIsAWarningAndIsNotPrinted(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":6,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":6,"textRun":{"content":"Text\n"}}]}}]},
		"comments":[{"id":"FARAWAY","range":{"startIndex":900,"endIndex":950}}]}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, warnings := Text(d)
	if strings.Contains(text, "[[c:") {
		t.Errorf("an unplaceable range was marked: %q", text)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "FARAWAY") {
		t.Errorf("warnings = %v, want one naming FARAWAY", warnings)
	}
}

func TestARangeNamingAnAbsentTabIsAWarning(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":6,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":6,"textRun":{"content":"Text\n"}}]}}]},
		"comments":[{"id":"OTHERTAB","range":{"startIndex":1,"endIndex":5,"tabId":"t.9"}}]}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, warnings := Text(d)
	if strings.Contains(text, "[[c:") {
		t.Errorf("a range naming an absent tab was marked: %q", text)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "OTHERTAB") {
		t.Errorf("warnings = %v, want one naming OTHERTAB", warnings)
	}
}

// structure is the shape the envelope carries under `structure`. Decoding into
// it is the round trip: a field renamed in docs shows up here as a zero.
type structure struct {
	Tabs []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Blocks []struct {
			Paragraph *struct {
				Style      string `json:"style"`
				StartIndex int    `json:"start_index"`
				EndIndex   int    `json:"end_index"`
				Runs       []struct {
					Kind       string `json:"kind"`
					Text       string `json:"text"`
					StartIndex int    `json:"start_index"`
					EndIndex   int    `json:"end_index"`
				} `json:"runs"`
			} `json:"paragraph"`
		} `json:"blocks"`
	} `json:"tabs"`
}

func TestStructureRoundTripsWithItsIndexes(t *testing.T) {
	raw, err := json.Marshal(Structure(fixture(t, "single-tab.json")))
	if err != nil {
		t.Fatal(err)
	}
	var got structure
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tabs) != 1 || got.Tabs[0].ID != "t.0" || got.Tabs[0].Title != "Policy" {
		t.Fatalf("tabs = %+v", got.Tabs)
	}
	p := got.Tabs[0].Blocks[0].Paragraph
	if p == nil {
		t.Fatal("the first block carries no paragraph")
	}
	if p.Style != "HEADING_1" || p.StartIndex != 1 || p.EndIndex != 7 {
		t.Errorf("first paragraph = %+v", *p)
	}
	if len(p.Runs) != 1 {
		t.Fatalf("len(runs) = %d, want 1", len(p.Runs))
	}
	r := p.Runs[0]
	if r.Kind != docs.KindText || r.Text != "Scope\n" || r.StartIndex != 1 || r.EndIndex != 7 {
		t.Errorf("first run = %+v", r)
	}
}

// Text suggested and then suggested away carries both id lists on one run, and
// Docs shows it as a deletion followed by an insertion. Both copies print, and
// the second one carries no comment marker: the characters exist once in the
// document, so opening a range around them twice would describe a range the
// document does not have.
func TestTextSuggestedAndThenSuggestedAwayPrintsTwiceAndIsMarkedOnce(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":10,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":10,"textRun":{"content":"draftier\n",
		   "suggestedInsertionIds":["INS1"],"suggestedDeletionIds":["DEL1"]}}]}}]},
		"comments":[{"id":"C1","range":{"startIndex":1,"endIndex":9}}]}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, _ := Text(d)

	if n := strings.Count(text, "draftier"); n != 2 {
		t.Errorf("the text prints %d times, want 2 (a deletion then an insertion): %q", n, text)
	}
	if !strings.Contains(text, "{-") || !strings.Contains(text, "[s:DEL1]") {
		t.Errorf("no deletion span: %q", text)
	}
	if !strings.Contains(text, "{+") || !strings.Contains(text, "[s:INS1]") {
		t.Errorf("no insertion span: %q", text)
	}
	if n := strings.Count(text, "[[c:C1]]"); n != 1 {
		t.Errorf("the comment opens %d times, want exactly 1: %q", n, text)
	}
	if n := strings.Count(text, "[[/c]]"); n != 1 {
		t.Errorf("the comment closes %d times, want exactly 1: %q", n, text)
	}
	// The deletion is the first copy, so the marker belongs to it.
	del := strings.Index(text, "{-")
	ins := strings.Index(text, "{+")
	if del < 0 || ins < 0 || del > ins {
		t.Fatalf("want the deletion before the insertion: %q", text)
	}
	if at := strings.Index(text, "[[c:C1]]"); at > ins {
		t.Errorf("the comment marker is on the second copy: %q", text)
	}
}
