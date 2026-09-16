package prelude

import (
	"testing"
)

// The house values below are written out as words and numbers on purpose, for
// the reason cover_test.go states: a test that reads the constant it checks
// follows that constant wherever somebody moves it.

// pageBreakIndexes is where each insertPageBreak request writes, in the order
// the requests write them.
func pageBreakIndexes(requests []map[string]any) []int {
	var out []int
	for _, r := range requests {
		body, ok := r["insertPageBreak"].(map[string]any)
		if !ok {
			continue
		}
		out = append(out, body["location"].(map[string]any)["index"].(int))
	}
	return out
}

// paragraphSpanAt is the paragraph the requests style over a span beginning at
// this index, and whether there is one.
func paragraphSpanAt(requests []map[string]any, at int) ([2]int, bool) {
	for _, span := range paragraphSpans(requests) {
		if span[0] == at {
			return span, true
		}
	}
	return [2]int{}, false
}

// The front matter turns the page where house.yaml says it does, and the second
// of its two breaks comes after the classification table, so the contents
// heading starts a page of its own.
func TestTheFrontMatterEndsWithAPageBreak(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert: two breaks, which is what house.yaml's front_matter states.
	breaks := pageBreakIndexes(got.Requests)
	if len(breaks) != 2 {
		t.Fatalf("the front matter carries %d page breaks at %v, want 2", len(breaks), breaks)
	}

	// Assert: the second break comes after the last table the front matter
	// inserts, which is the classification table, and nothing inserts a table
	// behind it.
	lastTable := -1
	for _, r := range got.Requests {
		body, ok := r["insertTable"].(map[string]any)
		if !ok {
			continue
		}
		at := body["location"].(map[string]any)["index"].(int)
		if at > lastTable {
			lastTable = at
		}
		if at > breaks[1] {
			t.Errorf("a table is inserted at %d, behind the last page break at %d", at, breaks[1])
		}
	}
	if lastTable < 0 || breaks[1] <= lastTable {
		t.Errorf("the last page break is at %d and the last table at %d, want the break after the table", breaks[1], lastTable)
	}

	// Assert: the heading behind it is the contents label, so the break is the
	// one that puts Contents at the top of its own page.
	after := ""
	for _, r := range got.Requests {
		body, ok := r["insertText"].(map[string]any)
		if !ok {
			continue
		}
		if body["location"].(map[string]any)["index"].(int) > breaks[1] {
			after = body["text"].(string)
			break
		}
	}
	if after != "Contents\n" {
		t.Errorf("the first words behind the last break are %q, want %q", after, "Contents\n")
	}

	// Assert: the break lives in a paragraph of gdoc's own, two index units
	// wide, and nothing writes a newline beside it. insertPageBreak writes the
	// break and the newline behind it, so an insertText at the same index would
	// leave a stray empty paragraph the count does not know about.
	span, ok := paragraphSpanAt(got.Requests, breaks[1])
	if !ok {
		t.Fatalf("no paragraph begins at the break's own index %d", breaks[1])
	}
	if span[1] != span[0]+2 {
		t.Errorf("the break's paragraph is %v, want two index units", span)
	}
	for _, r := range got.Requests {
		body, ok := r["insertText"].(map[string]any)
		if !ok {
			continue
		}
		if body["location"].(map[string]any)["index"].(int) == breaks[1] {
			t.Errorf("an insertText writes %q at %d, where the page break already writes its own newline",
				body["text"], breaks[1])
		}
	}
}
