package prelude

import (
	"encoding/json"
	"net/url"
	"testing"

	"gdoc/internal/cover"
	"gdoc/internal/guard"
)

// bulletRemovals is every deleteParagraphBullets in the requests, as its range,
// with the position of each in the list.
func bulletRemovals(requests []map[string]any) (spans [][2]int, at []int) {
	for i, r := range requests {
		body, ok := r["deleteParagraphBullets"].(map[string]any)
		if !ok {
			continue
		}
		spans = append(spans, styleRange(body))
		at = append(at, i)
	}
	return spans, at
}

// lastInsert is the position of the last request that adds a character.
func lastInsert(requests []map[string]any) int {
	last := -1
	for i, r := range requests {
		for _, kind := range []string{"insertText", "insertTable", "insertPageBreak"} {
			if _, ok := r[kind]; ok {
				last = i
			}
		}
	}
	return last
}

// A prelude proposed in front of a list item arrives bulleted, because every
// paragraph inserted at a list item's start joins its list. One
// deleteParagraphBullets over exactly what the prelude inserted takes that off,
// under the insertion's own suggestion id (MEASURED.md, "A block of new
// paragraphs, proposed in one SUGGEST batch"). It goes after every insert,
// because a range names characters that must already be there.
func TestThePreludeTakesOffTheListMarkerItInherits(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}

	// Assert
	spans, at := bulletRemovals(got.Requests)
	if len(spans) != 1 {
		t.Fatalf("the front matter carries %d deleteParagraphBullets, want exactly 1", len(spans))
	}
	if spans[0] != [2]int{got.Start, got.End} {
		t.Errorf("the bullets come off %v, want exactly the prelude's own range %v", spans[0], [2]int{got.Start, got.End})
	}
	if at[0] < lastInsert(got.Requests) {
		t.Errorf("deleteParagraphBullets is request %d, before the last insert at %d", at[0], lastInsert(got.Requests))
	}
	body := got.Requests[at[0]]["deleteParagraphBullets"].(map[string]any)
	if len(body) != 1 {
		t.Errorf("deleteParagraphBullets carries %v, want its range and nothing else", body)
	}
}

// On a second run the old prelude is proposed for deletion behind the new one,
// and that text is not the new prelude's to touch: the removal names the new
// prelude's range and stops at its end.
func TestAReplaceRunTakesTheMarkerOffOnlyTheNewPrelude(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	d := markedDocument(1, 964, false)

	// Act
	got, err := Propose(cfg, cover.Fields{Title: "A Policy"}, d)
	if err != nil {
		t.Fatalf("Propose() = %v", err)
	}

	// Assert
	spans, _ := bulletRemovals(got.Requests)
	if len(spans) != 1 {
		t.Fatalf("a replace run carries %d deleteParagraphBullets, want exactly 1", len(spans))
	}
	if spans[0] != [2]int{got.Start, got.End} {
		t.Errorf("the bullets come off %v, want the new prelude's range %v and not the old one behind it", spans[0], [2]int{got.Start, got.End})
	}
}

// The prelude goes out on the policy propose uses, a handed-in document at the
// suggest level with no grant, and the removal carries there in a SUGGEST
// batch. The real policy judges the real bytes, so the test fails if the guard
// ever stops carrying it rather than if a copy of its shape drifts.
func TestThePreludeBatchCarriesAtTheSuggestLevel(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	res, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}
	p := guard.NewPolicy()
	p.AllowFile("DOC1", guard.LevelSuggest)
	at, err := url.Parse("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if err != nil {
		t.Fatal(err)
	}
	spans, idx := bulletRemovals(res.Requests)
	if len(spans) != 1 {
		t.Fatalf("the front matter carries %d deleteParagraphBullets, want exactly 1", len(spans))
	}
	suggest := func(reqs []map[string]any) []byte {
		body, err := json.Marshal(map[string]any{
			"requests":     reqs,
			"writeControl": map[string]any{"writeMode": "SUGGEST"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}

	// Act, Assert
	if err := p.Judge("POST", at, suggest([]map[string]any{res.Requests[idx[0]]})); err != nil {
		t.Fatalf("the guard refused the bullet removal in a SUGGEST batch: %v", err)
	}
	direct, _ := json.Marshal(map[string]any{"requests": []map[string]any{res.Requests[idx[0]]}})
	if p.Judge("POST", at, direct) == nil {
		t.Fatal("without SUGGEST the removal is a direct edit of somebody's document and must be refused")
	}
}
