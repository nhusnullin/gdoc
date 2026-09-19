// The four facts M13 added to the walk: a run's link target, a bullet's list
// and glyph, a paragraph's heading id and its floating objects, and the
// contents element as a block. Each fixture beside these tests is written from
// the Docs API reference, the way elements.json is, and stays a hypothesis
// until a live read agrees with it.

package docs

import (
	"encoding/json"
	"testing"
)

// linked is the run whose text is exactly this, in the tab's paragraph p. A
// test that reads runs by index moves every time the fixture gains a word.
func linked(t *testing.T, p *Paragraph, text string) Run {
	t.Helper()
	for _, r := range p.Runs {
		if r.Text == text {
			return r
		}
	}
	t.Fatalf("no run saying %q in %q", text, runText(p))
	return Run{}
}

func TestARunCarriesItsLinkTarget(t *testing.T) {
	ps := paragraphs(fixture(t, "links.json").Tabs[0].Body)
	p := ps[1]

	if got := linked(t, p, "guide").Link; got == nil || got.URL != "https://example.com/guide" {
		t.Errorf("the url link = %+v", got)
	}
	if got := linked(t, p, "the section").Link; got == nil || got.HeadingID != "h.thesection" {
		t.Errorf("the heading link = %+v", got)
	}
	if got := linked(t, p, "a bookmark").Link; got == nil || got.BookmarkID != "id.bookmarkone" {
		t.Errorf("the bookmark link = %+v", got)
	}
	if got := linked(t, p, "the plan").Link; got == nil || got.TabID != "t.1" {
		t.Errorf("the tab link = %+v", got)
	}
	// A run with no link carries no Link at all, so a reader asking whether
	// this text points anywhere gets no for an answer rather than an empty
	// target it has to know is empty.
	if got := linked(t, p, "Read the ").Link; got != nil {
		t.Errorf("a run with no link carried %+v", got)
	}
	// One form at a time: the url link says url and nothing else.
	if got := linked(t, p, "guide").Link; got.HeadingID != "" || got.BookmarkID != "" || got.TabID != "" {
		t.Errorf("the url link also carried %+v", got)
	}
}

func TestALinkInItsNestedFormIsTheSameFact(t *testing.T) {
	// The reference documents heading and bookmark as objects carrying an id
	// and the tab the target is in, with headingId and bookmarkId as the older
	// flat spelling of the same fact. Both decode into the same two fields.
	p := paragraphs(fixture(t, "links.json").Tabs[0].Body)[2]

	got := linked(t, p, "the nested heading").Link
	if got == nil || got.HeadingID != "h.nested" {
		t.Fatalf("the nested heading link = %+v", got)
	}
	if got.TabID != "t.1" {
		t.Errorf("the nested heading link named tab %q, want t.1", got.TabID)
	}
	mark := linked(t, p, "the bookmark").Link
	if mark == nil || mark.BookmarkID != "id.nested" || mark.TabID != "t.0" {
		t.Errorf("the nested bookmark link = %+v", mark)
	}
}

func TestABulletCarriesItsListAndGlyph(t *testing.T) {
	d := fixture(t, "lists.json")
	ps := paragraphs(d.Tabs[0].Body)
	if len(ps) != 4 {
		t.Fatalf("len(paragraphs) = %d, want 4", len(ps))
	}

	first := ps[0].Bullet
	if first == nil {
		t.Fatal("the first item carries no bullet")
	}
	if first.ListID != "kix.listone" || first.NestingLevel != 0 {
		t.Errorf("the first item = %+v", first)
	}
	if !first.Ordered || first.Glyph != "DECIMAL" {
		t.Errorf("the first item's glyph = %+v", first)
	}

	nested := ps[1].Bullet
	if nested == nil || nested.NestingLevel != 1 || !nested.Ordered || nested.Glyph != "ALPHA" {
		t.Errorf("the nested item = %+v", nested)
	}

	// A glyph type of GLYPH_TYPE_UNSPECIFIED is Docs saying this list is not
	// numbered, so it is neither ordered nor a glyph.
	bullet := ps[2].Bullet
	if bullet == nil || bullet.ListID != "kix.listtwo" || bullet.Ordered || bullet.Glyph != "" {
		t.Errorf("the unordered item = %+v", bullet)
	}

	if ps[3].Bullet != nil {
		t.Errorf("a paragraph in no list carried %+v", ps[3].Bullet)
	}

	// The lists map is the tab's own. Two tabs may name one list id and mean
	// two different lists, and a map read from the document would number this
	// second tab's bullets.
	other := paragraphs(d.Tabs[1].Body)[0].Bullet
	if other == nil || other.ListID != "kix.listone" {
		t.Fatalf("the second tab's item = %+v", other)
	}
	if other.Ordered || other.Glyph != "" {
		t.Errorf("the second tab's item took the first tab's glyph: %+v", other)
	}
}

func TestAParagraphCarriesItsHeadingID(t *testing.T) {
	ps := paragraphs(fixture(t, "links.json").Tabs[0].Body)
	if ps[0].Style != "HEADING_1" {
		t.Fatalf("the first paragraph's style = %q", ps[0].Style)
	}
	if ps[0].HeadingID != "h.thesection" {
		t.Errorf("the heading's id = %q", ps[0].HeadingID)
	}
	if ps[1].HeadingID != "" {
		t.Errorf("an ordinary paragraph carried the heading id %q", ps[1].HeadingID)
	}
}

func TestATableOfContentsIsABlock(t *testing.T) {
	body := fixture(t, "toc.json").Tabs[0].Body
	if len(body) != 3 {
		t.Fatalf("len(blocks) = %d, want 3", len(body))
	}
	toc := body[1].TOC
	if toc == nil {
		t.Fatal("the contents element did not land as a block")
	}
	if body[1].Paragraph != nil || body[1].Table != nil {
		t.Error("the contents block is also a paragraph or a table")
	}
	entries := paragraphs(toc.Blocks)
	if len(entries) != 2 {
		t.Fatalf("len(contents paragraphs) = %d, want 2", len(entries))
	}
	if got := runText(entries[0]); got != "1-Scope\n" {
		t.Errorf("the first entry = %q", got)
	}
	// An entry is a link to the heading it names, walked like any other run.
	if got := entries[1].Runs[0].Link; got == nil || got.HeadingID != "h.whodecides" {
		t.Errorf("the second entry's link = %+v", got)
	}
	// plainText is what a footnote's text is joined from, and it reads a
	// contents block the way it reads a paragraph.
	if got := plainText(body); got != "The policy\n1-Scope\n2-Who decides\n1-Scope" {
		t.Errorf("plainText = %q", got)
	}
}

func TestAPositionedObjectIsCarriedOnItsParagraph(t *testing.T) {
	tab := fixture(t, "positioned.json").Tabs[0]
	ps := paragraphs(tab.Body)

	if len(ps[0].Positioned) != 2 {
		t.Fatalf("the first paragraph's floating objects = %v", ps[0].Positioned)
	}
	if ps[0].Positioned[0] != "kix.posone" || ps[0].Positioned[1] != "kix.postwo" {
		t.Errorf("the ids came back as %v, out of the order the paragraph named them", ps[0].Positioned)
	}
	if ps[1].Positioned != nil {
		t.Errorf("a paragraph anchoring nothing carried %v", ps[1].Positioned)
	}

	if len(tab.Positioned) != 2 {
		t.Fatalf("the tab's objects = %+v", tab.Positioned)
	}
	pic := tab.Positioned["kix.posone"]
	if pic.ID != "kix.posone" || pic.Kind != KindImage {
		t.Errorf("the floating picture = %+v", pic)
	}
	if draw := tab.Positioned["kix.postwo"]; draw.Kind != KindDrawing {
		t.Errorf("the floating drawing = %+v", draw)
	}
}

// TestAnInlineObjectCarriesItsObjectID is the pin for the id an inline picture
// or drawing is paired with its bytes by. The run carries no label with it, so
// the placeholder internal/view prints is unchanged.
func TestAnInlineObjectCarriesItsObjectID(t *testing.T) {
	d := fixture(t, "objects.json")

	runs := d.Tabs[0].Body[0].Paragraph.Runs
	want := map[string]string{"kix.img1": KindImage, "kix.draw1": KindDrawing, "kix.obj1": KindObject}
	got := map[string]string{}
	for _, r := range runs {
		if r.Kind != KindImage && r.Kind != KindDrawing && r.Kind != KindObject {
			continue
		}
		if r.Detail == nil {
			t.Fatalf("the %s run carries no detail, so it names no object", r.Kind)
		}
		if r.Detail.Label != "" {
			t.Errorf("the %s run carries the label %q, and an object shows none", r.Kind, r.Detail.Label)
		}
		got[r.Detail.ID] = r.Kind
	}
	if len(got) != len(want) {
		t.Fatalf("object ids = %v, want %v", got, want)
	}
	for id, kind := range want {
		if got[id] != kind {
			t.Errorf("object %s is %q, want %q", id, got[id], kind)
		}
	}
}

// NONE is the Docs enum for a level drawn with an empty glyph: the list is
// there and its items show no marker at all. It is not an absence and not
// GLYPH_TYPE_UNSPECIFIED, so the check that reads "is this level numbered" has
// to name it, or a list that draws nothing reads back as 1., 2., 3. and is
// exported that way into the hub.
func TestAGlyphOfNONEIsNotANumberedList(t *testing.T) {
	var blank rawList
	if err := json.Unmarshal([]byte(`{"listProperties":{"nestingLevels":[{"glyphType":"NONE"}]}}`), &blank); err != nil {
		t.Fatal(err)
	}

	got := bullet("kix.blank", 0, map[string]rawList{"kix.blank": blank})
	if got == nil {
		t.Fatal("a level with a NONE glyph carries no bullet")
	}
	if got.Ordered || got.Glyph != "" {
		t.Errorf("a level with a NONE glyph = %+v, want neither ordered nor a glyph", got)
	}
}
