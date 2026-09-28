package prelude

import (
	"encoding/json"
	"net/url"
	"reflect"
	"testing"

	"gdoc/internal/cover"
	"gdoc/internal/docs"
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

// openingWith is a document with no prelude whose first paragraph is the
// author's own, a list item or not.
func openingWith(listItem bool) *docs.Document {
	para := &docs.Paragraph{Style: "NORMAL_TEXT", StartIndex: 1, EndIndex: 12, Runs: []docs.Run{
		{Kind: docs.KindText, Text: "First item\n", StartIndex: 1, EndIndex: 12},
	}}
	if listItem {
		para.Bullet = &docs.Bullet{ListID: "kix.list1"}
	}
	return &docs.Document{ID: "DOC1", Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{{Paragraph: para}}}}}
}

// assertOneRemovalOverThePrelude says the requests carry exactly one bullet
// removal, over exactly the new prelude, and last.
func assertOneRemovalOverThePrelude(t *testing.T, got Result) {
	t.Helper()
	spans, at := bulletRemovals(got.Requests)
	if len(spans) != 1 {
		t.Fatalf("the batch carries %d deleteParagraphBullets, want exactly 1", len(spans))
	}
	if spans[0] != [2]int{got.Start, got.End} {
		t.Errorf("the bullets come off %v, want exactly the new prelude's range %v", spans[0], [2]int{got.Start, got.End})
	}
	if at[0] != len(got.Requests)-1 || at[0] < lastInsert(got.Requests) {
		t.Errorf("deleteParagraphBullets is request %d of %d, want it last, behind every insert", at[0], len(got.Requests))
	}
	body := got.Requests[at[0]]["deleteParagraphBullets"].(map[string]any)
	if len(body) != 1 {
		t.Errorf("deleteParagraphBullets carries %v, want its range and nothing else", body)
	}
}

// A prelude proposed in front of a list item would arrive bulleted, because
// every paragraph inserted at a list item's start joins its list. One
// deleteParagraphBullets over exactly what the prelude inserted takes that off,
// under the insertion's own suggestion id (MEASURED.md, "A block of new
// paragraphs, proposed in one SUGGEST batch").
func TestAPreludeInFrontOfAListItemTakesOffTheMarker(t *testing.T) {
	// Arrange
	cfg := testConfig(t)

	// Act
	got, err := Propose(cfg, testFields(), openingWith(true))
	if err != nil {
		t.Fatalf("Propose() = %v", err)
	}

	// Assert
	assertOneRemovalOverThePrelude(t, got)
}

// On a replace run the old prelude is proposed for deletion behind the new one,
// and that text is not the new prelude's to touch: the removal names the new
// prelude's range and stops at its end.
func TestAReplaceRunInFrontOfAListItemClearsOnlyTheNewPrelude(t *testing.T) {
	// Arrange: the paragraph the new prelude lands in front of is a list item.
	cfg := testConfig(t)
	d := markedDocument(1, 964, false)
	d.Tabs[0].Body[0].Paragraph.Bullet = &docs.Bullet{ListID: "kix.list1"}

	// Act
	got, err := Propose(cfg, cover.Fields{Title: "A Policy"}, d)
	if err != nil {
		t.Fatalf("Propose() = %v", err)
	}

	// Assert
	if got.Replaces == nil {
		t.Fatal("the document carries a settled marker, so this run replaces it")
	}
	assertOneRemovalOverThePrelude(t, got)
}

// The common case stays the batch that was measured. A prelude in front of
// anything that is not a list item sends no bullet removal, and its requests
// are the front matter's own, request for request.
func TestAPreludeInFrontOfAPlainParagraphSendsTheMeasuredBatch(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	want, err := FrontMatter(cfg, testFields(), 1)
	if err != nil {
		t.Fatalf("FrontMatter() = %v", err)
	}
	cases := map[string]*docs.Document{
		"a plain first paragraph": openingWith(false),
		"an empty body":           {ID: "DOC1", Tabs: []docs.Tab{{ID: "t.0"}}},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			got, err := Propose(cfg, testFields(), d)
			if err != nil {
				t.Fatalf("Propose() = %v", err)
			}

			// Assert
			if spans, _ := bulletRemovals(got.Requests); len(spans) != 0 {
				t.Fatalf("the batch carries deleteParagraphBullets over %v in front of a paragraph with no bullet", spans)
			}
			if !reflect.DeepEqual(got.Requests, want.Requests) {
				t.Error("the requests differ from the front matter's own, and in front of a plain paragraph they are that and nothing else")
			}
		})
	}

	t.Run("and the front matter itself sends none", func(t *testing.T) {
		if spans, _ := bulletRemovals(want.Requests); len(spans) != 0 {
			t.Fatalf("FrontMatter carries deleteParagraphBullets over %v; the decision is Propose's, which has the document", spans)
		}
	})
}

// The prelude goes out on the policy propose uses, a handed-in document at the
// suggest level with no grant, and the removal carries there in a SUGGEST
// batch. The real policy judges the real bytes, so the test fails if the guard
// ever stops carrying it rather than if a copy of its shape drifts.
func TestThePreludeBatchCarriesAtTheSuggestLevel(t *testing.T) {
	// Arrange
	cfg := testConfig(t)
	res, err := Propose(cfg, testFields(), openingWith(true))
	if err != nil {
		t.Fatalf("Propose() = %v", err)
	}
	p := guard.NewPolicy()
	p.AllowFile("DOC1", guard.LevelSuggest)
	at, err := url.Parse("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if err != nil {
		t.Fatal(err)
	}
	spans, idx := bulletRemovals(res.Requests)
	if len(spans) != 1 {
		t.Fatalf("the prelude carries %d deleteParagraphBullets, want exactly 1", len(spans))
	}
	suggest, err := json.Marshal(map[string]any{
		"requests":     []map[string]any{res.Requests[idx[0]]},
		"writeControl": map[string]any{"writeMode": "SUGGEST"},
	})
	if err != nil {
		t.Fatal(err)
	}
	direct, err := json.Marshal(map[string]any{"requests": []map[string]any{res.Requests[idx[0]]}})
	if err != nil {
		t.Fatal(err)
	}

	// Act, Assert
	if err := p.Judge("POST", at, suggest); err != nil {
		t.Fatalf("the guard refused the bullet removal in a SUGGEST batch: %v", err)
	}
	if p.Judge("POST", at, direct) == nil {
		t.Fatal("without SUGGEST the removal is a direct edit of somebody's document and must be refused")
	}
}
