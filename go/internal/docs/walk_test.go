package docs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture parses a testdata file the way Fetch parses a body off the wire, so
// every test below runs the code the wire feeds.
func fixture(t *testing.T, name string) *Document {
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

// paragraphs is every paragraph of a tab's body in reading order, tables aside.
func paragraphs(blocks []Block) []*Paragraph {
	var out []*Paragraph
	for _, b := range blocks {
		if b.Paragraph != nil {
			out = append(out, b.Paragraph)
		}
	}
	return out
}

// runText is a paragraph's runs joined, which is the text before any projection.
func runText(p *Paragraph) string {
	s := ""
	for _, r := range p.Runs {
		s += r.Text
	}
	return s
}

func TestDocumentHeaderFields(t *testing.T) {
	d := fixture(t, "single-tab.json")
	if d.ID != "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd" {
		t.Errorf("ID = %q", d.ID)
	}
	if d.Title != "Supplier register policy" {
		t.Errorf("Title = %q", d.Title)
	}
	if d.RevisionID != "ALm37BXsingleTab" {
		t.Errorf("RevisionID = %q", d.RevisionID)
	}
	if d.MultiTab() {
		t.Error("MultiTab() = true for a one-tab document")
	}
}

func TestTwoTabsAreTwoTabs(t *testing.T) {
	d := fixture(t, "two-tabs.json")
	if len(d.Tabs) != 2 {
		t.Fatalf("len(Tabs) = %d, want 2", len(d.Tabs))
	}
	if !d.MultiTab() {
		t.Error("MultiTab() = false with two tabs")
	}
	if d.Tabs[0].ID != "t.0" || d.Tabs[0].Title != "Overview" {
		t.Errorf("first tab = %q %q", d.Tabs[0].ID, d.Tabs[0].Title)
	}
	// A child tab is a tab. It comes after its parent, which is the order the
	// document shows it in.
	if d.Tabs[1].ID != "t.1" || d.Tabs[1].Title != "Detail" {
		t.Errorf("second tab = %q %q", d.Tabs[1].ID, d.Tabs[1].Title)
	}
	if got := runText(paragraphs(d.Tabs[1].Body)[1]); got != "The second tab's text.\n" {
		t.Errorf("second tab body = %q", got)
	}
}

func TestPreTabsShapeIsOneTab(t *testing.T) {
	d := fixture(t, "pre-tabs.json")
	if len(d.Tabs) != 1 {
		t.Fatalf("len(Tabs) = %d, want 1", len(d.Tabs))
	}
	if d.Tabs[0].ID != "t.0" || d.Tabs[0].Title != "" {
		t.Errorf("tab = %q %q, want t.0 with no title", d.Tabs[0].ID, d.Tabs[0].Title)
	}
	if d.MultiTab() {
		t.Error("MultiTab() = true for the pre-tabs shape")
	}
	ps := paragraphs(d.Tabs[0].Body)
	if len(ps) != 2 {
		t.Fatalf("len(paragraphs) = %d, want 2", len(ps))
	}
	if ps[0].Style != "TITLE" {
		t.Errorf("first paragraph style = %q", ps[0].Style)
	}
}

func TestHeadingsAndBulletsCarryTheirStyle(t *testing.T) {
	ps := paragraphs(fixture(t, "single-tab.json").Tabs[0].Body)
	if len(ps) != 6 {
		t.Fatalf("len(paragraphs) = %d, want 6", len(ps))
	}
	if ps[0].Style != "HEADING_1" || runText(ps[0]) != "Scope\n" {
		t.Errorf("first paragraph = %q %q", ps[0].Style, runText(ps[0]))
	}
	if ps[0].StartIndex != 1 || ps[0].EndIndex != 7 {
		t.Errorf("first paragraph indexes = %d..%d, want 1..7", ps[0].StartIndex, ps[0].EndIndex)
	}
	if ps[2].Style != "HEADING_2" {
		t.Errorf("third paragraph style = %q, want HEADING_2", ps[2].Style)
	}
	if ps[3].Bullet == nil || ps[3].Bullet.NestingLevel != 0 {
		t.Errorf("first bullet = %+v, want nesting level 0", ps[3].Bullet)
	}
	if ps[4].Bullet == nil || ps[4].Bullet.NestingLevel != 1 {
		t.Errorf("second bullet = %+v, want nesting level 1", ps[4].Bullet)
	}
	if ps[1].Bullet != nil {
		t.Error("a plain paragraph carries a bullet")
	}
}

func TestTableTextLandsInCellsInReadingOrder(t *testing.T) {
	var tbl Table
	for _, b := range fixture(t, "single-tab.json").Tabs[0].Body {
		if b.Table != nil {
			tbl = b.Table
		}
	}
	if len(tbl) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(tbl))
	}
	want := [][]string{{"Control\n", "Owner\n"}, {"Register review\n", "Operations\n"}}
	for r, row := range tbl {
		if len(row) != 2 {
			t.Fatalf("row %d has %d cells, want 2", r, len(row))
		}
		for c, cell := range row {
			ps := paragraphs(cell.Blocks)
			if len(ps) != 1 {
				t.Fatalf("cell %d,%d has %d paragraphs, want 1", r, c, len(ps))
			}
			if got := runText(ps[0]); got != want[r][c] {
				t.Errorf("cell %d,%d = %q, want %q", r, c, got, want[r][c])
			}
		}
	}
}

func TestRunsCarryTheirSuggestionIDs(t *testing.T) {
	ps := paragraphs(fixture(t, "single-tab.json").Tabs[0].Body)
	runs := ps[1].Runs
	if len(runs) != 7 {
		t.Fatalf("len(runs) = %d, want 7", len(runs))
	}
	if got := runs[1].DeletionIDs; len(got) != 1 || got[0] != "suggest.a1" {
		t.Errorf("deletion run ids = %v", got)
	}
	if len(runs[1].InsertionIDs) != 0 {
		t.Errorf("deletion run carries insertion ids %v", runs[1].InsertionIDs)
	}
	// The insertion is two runs because the second half is bold. Both carry the
	// one id, and joining them is the reader's job, not the walk's.
	for _, i := range []int{2, 3} {
		if got := runs[i].InsertionIDs; len(got) != 1 || got[0] != "suggest.a1" {
			t.Errorf("run %d insertion ids = %v", i, got)
		}
	}
	if len(runs[0].InsertionIDs) != 0 || len(runs[0].DeletionIDs) != 0 {
		t.Error("a plain run carries suggestion ids")
	}
	if runs[0].Kind != KindText {
		t.Errorf("plain run kind = %q", runs[0].Kind)
	}
	if runs[1].StartIndex != 41 || runs[1].EndIndex != 49 {
		t.Errorf("deletion run indexes = %d..%d, want 41..49", runs[1].StartIndex, runs[1].EndIndex)
	}
}

func TestObjectRunsCarryTheirKindsAndTheFootnoteText(t *testing.T) {
	d := fixture(t, "objects.json")
	runs := paragraphs(d.Tabs[0].Body)[0].Runs
	// KindObject is in this list on purpose. An embedded object carrying
	// neither imageProperties nor embeddedDrawingProperties is a thing gdoc
	// cannot name, and calling it an image would be the guess this package
	// refuses to make. Without a case here that guess passes every test.
	want := []string{KindText, KindImage, KindText, KindDrawing, KindObject, KindEquation, KindFootnoteRef, KindText}
	if len(runs) != len(want) {
		t.Fatalf("len(runs) = %d, want %d", len(runs), len(want))
	}
	for i, k := range want {
		if runs[i].Kind != k {
			t.Errorf("run %d kind = %q, want %q", i, runs[i].Kind, k)
		}
	}
	if runs[6].FootnoteID != "kix.fn1" {
		t.Errorf("footnote reference id = %q", runs[6].FootnoteID)
	}
	if got := d.Footnotes["kix.fn1"]; got != "Measured on 2026-09-06." {
		t.Errorf("footnote text = %q", got)
	}
	if len(d.Footnotes) != 1 {
		t.Errorf("len(Footnotes) = %d, want 1", len(d.Footnotes))
	}
}

func TestCommentRangesPlaceOneAndListTheOther(t *testing.T) {
	d := fixture(t, "single-tab.json")
	got, ok := d.CommentRanges["AAAA1111"]
	if !ok {
		t.Fatalf("CommentRanges = %v, want the ranged comment", d.CommentRanges)
	}
	if got != (Range{Tab: "t.0", Start: 66, End: 85}) {
		t.Errorf("range = %+v", got)
	}
	if len(d.CommentRanges) != 1 {
		t.Errorf("len(CommentRanges) = %d, want 1", len(d.CommentRanges))
	}
	if len(d.Unplaced) != 1 || d.Unplaced[0] != "BBBB2222" {
		t.Errorf("Unplaced = %v, want [BBBB2222]", d.Unplaced)
	}
}

// The shape Google actually returns, measured on 2026-09-06 against a real
// document with six comments: each `comments[]` entry carries `commentId` and
// an `anchorId`, and the range lives in the tab, under
// `documentTab.commentAnchors[anchorId].ranges`. The fixture is that shape with
// placeholder text; the recording it was modelled on holds a real document and
// stays out of the tree.
func TestCommentRangesComeFromTheTabsCommentAnchors(t *testing.T) {
	d := fixture(t, "anchors.json")
	want := map[string]Range{
		"C-FIRST":  {Tab: "t.0", Start: 7, End: 15},
		"C-TWO":    {Tab: "t.0", Start: 22, End: 31}, // an anchor with several ranges places on its first
		"C-SECOND": {Tab: "t.1", Start: 13, End: 27}, // the tab is the one whose anchors named it
	}
	for id, w := range want {
		got, ok := d.CommentRanges[id]
		if !ok || got != w {
			t.Errorf("%s: range = %+v (placed %v), want %+v", id, got, ok, w)
		}
	}
	if len(d.CommentRanges) != len(want) {
		t.Errorf("len(CommentRanges) = %d, want %d: %v", len(d.CommentRanges), len(want), d.CommentRanges)
	}
	// An anchorId no tab knows is a comment the read did not place: reported,
	// never guessed.
	if len(d.Unplaced) != 1 || d.Unplaced[0] != "C-LOST" {
		t.Errorf("Unplaced = %v, want [C-LOST]", d.Unplaced)
	}
}

func TestCommentRangeShapesTheDecoderAccepts(t *testing.T) {
	// The shape of the Docs read's `comments` key is measured, not documented,
	// so the decoder tries the three places a range has been seen. Each case is
	// one of them, and the last is an entry with no range at all.
	cases := []struct {
		name  string
		entry string
		want  *Range
	}{
		{"range", `{"id":"C1","range":{"startIndex":4,"endIndex":9}}`, &Range{Tab: "t.0", Start: 4, End: 9}},
		{"anchor.range", `{"id":"C1","anchor":{"range":{"startIndex":4,"endIndex":9}}}`, &Range{Tab: "t.0", Start: 4, End: 9}},
		{"top level", `{"id":"C1","startIndex":4,"endIndex":9}`, &Range{Tab: "t.0", Start: 4, End: 9}},
		{"anchor is a string", `{"id":"C1","anchor":"kix.abc","startIndex":4,"endIndex":9}`, &Range{Tab: "t.0", Start: 4, End: 9}},
		{"no range", `{"id":"C1"}`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw := `{"documentId":"D","body":{"content":[]},"comments":[` + c.entry + `]}`
			d, err := Parse([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			got, ok := d.CommentRanges["C1"]
			if c.want == nil {
				if ok {
					t.Fatalf("placed %+v, want unplaced", got)
				}
				if len(d.Unplaced) != 1 || d.Unplaced[0] != "C1" {
					t.Fatalf("Unplaced = %v", d.Unplaced)
				}
				return
			}
			if !ok || got != *c.want {
				t.Fatalf("range = %+v (placed %v), want %+v", got, ok, *c.want)
			}
		})
	}
}

func TestACommentInAMultiTabDocumentNeedsItsTab(t *testing.T) {
	// One tab is the tab. With more than one, a range without a tabId names no
	// position gdoc can print, so it is reported unplaced rather than guessed.
	raw := `{"documentId":"D","tabs":[
		{"tabProperties":{"tabId":"t.0"},"documentTab":{"body":{"content":[]}}},
		{"tabProperties":{"tabId":"t.1"},"documentTab":{"body":{"content":[]}}}],
		"comments":[{"id":"C1","range":{"startIndex":4,"endIndex":9}},
		            {"id":"C2","range":{"startIndex":4,"endIndex":9,"tabId":"t.1"}}]}`
	d, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Unplaced) != 1 || d.Unplaced[0] != "C1" {
		t.Errorf("Unplaced = %v, want [C1]", d.Unplaced)
	}
	if got := d.CommentRanges["C2"]; got != (Range{Tab: "t.1", Start: 4, End: 9}) {
		t.Errorf("C2 range = %+v", got)
	}
}

func TestACommentWithNoIDIsReportedByPosition(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[]},"comments":[{"range":{"startIndex":1,"endIndex":2}}]}`
	d, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Unplaced) != 1 || d.Unplaced[0] != "comments[0] (no id)" {
		t.Errorf("Unplaced = %v", d.Unplaced)
	}
}

// TestTheDroppedElementsFixtureNamesAllSeven states what elements.json is for.
// ParagraphElement is a union of eleven and run() reads four, so seven members
// are dropped at decode today. The fixture holds one of each, with its own two
// suggestion id lists, and it is what the decoder is taught against.
func TestTheDroppedElementsFixtureNamesAllSeven(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "elements.json"))
	if err != nil {
		t.Fatal(err)
	}
	// It parses the way a body off the wire parses, whatever the decoder does
	// with the members afterwards.
	if _, err := Parse(raw); err != nil {
		t.Fatalf("Parse(elements.json): %v", err)
	}

	found := map[string]map[string]bool{}
	for _, el := range fixtureParaElements(t, raw) {
		for key, val := range el {
			if key == "startIndex" || key == "endIndex" {
				continue
			}
			var member map[string]json.RawMessage
			if err := json.Unmarshal(val, &member); err != nil {
				t.Fatalf("member %q is not an object: %v", key, err)
			}
			keys := map[string]bool{}
			for k := range member {
				keys[k] = true
			}
			found[key] = keys
		}
	}

	for _, member := range []string{
		"person", "richLink", "dateElement",
		"autoText", "pageBreak", "columnBreak", "horizontalRule",
	} {
		keys, ok := found[member]
		if !ok {
			t.Errorf("elements.json holds no %s", member)
			continue
		}
		for _, ids := range []string{"suggestedInsertionIds", "suggestedDeletionIds"} {
			if !keys[ids] {
				t.Errorf("%s carries no %s", member, ids)
			}
		}
	}

	// The fields the reference names for the two chips that carry data, and
	// the field that says which auto text this is. A fixture missing one of
	// them cannot teach the decoder to read it.
	for member, field := range map[string]string{
		"person":         "personProperties",
		"richLink":       "richLinkProperties",
		"dateElement":    "dateElementProperties",
		"autoText":       "type",
		"horizontalRule": "textStyle",
	} {
		if !found[member][field] {
			t.Errorf("%s carries no %s", member, field)
		}
	}
}

// fixtureParaElements is every paragraph element in a fixture, raw, so a test
// can ask what the JSON names rather than what the decoder kept.
func fixtureParaElements(t *testing.T, raw []byte) []map[string]json.RawMessage {
	t.Helper()
	var doc struct {
		Tabs []struct {
			DocumentTab struct {
				Body struct {
					Content []struct {
						Paragraph *struct {
							Elements []map[string]json.RawMessage `json:"elements"`
						} `json:"paragraph"`
					} `json:"content"`
				} `json:"body"`
			} `json:"documentTab"`
		} `json:"tabs"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var out []map[string]json.RawMessage
	for _, tab := range doc.Tabs {
		for _, el := range tab.DocumentTab.Body.Content {
			if el.Paragraph != nil {
				out = append(out, el.Paragraph.Elements...)
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("the fixture holds no paragraph elements")
	}
	return out
}

// TestTheSevenElementsDecodeIntoRuns is the fix Task 1's fixture was built for.
// ParagraphElement is a union of eleven and run() read four, so a person chip,
// a date chip, a calendar link, an auto text, a page break, a column break and
// a horizontal rule each vanished at decode: no run, no placeholder, no
// warning. Every review since M2 read those documents with holes in them.
func TestTheSevenElementsDecodeIntoRuns(t *testing.T) {
	d := fixture(t, "elements.json")
	var runs []Run
	for _, p := range paragraphs(d.Tabs[0].Body) {
		runs = append(runs, p.Runs...)
	}
	want := []string{
		KindText, KindPerson, KindText, KindDate, KindText, KindRichLink, KindText,
		KindHorizontalRule, KindText,
		KindText, KindAutoText, KindText,
		KindPageBreak, KindText,
		KindColumnBreak, KindText,
	}
	if len(runs) != len(want) {
		t.Fatalf("len(runs) = %d, want %d: %+v", len(runs), len(want), runs)
	}
	for i, k := range want {
		if runs[i].Kind != k {
			t.Errorf("run %d kind = %q, want %q", i, runs[i].Kind, k)
		}
	}

	// The fields the reference names for each member, and the two id lists
	// every element in a paragraph carries.
	byKind := map[string]Run{}
	for _, r := range runs {
		byKind[r.Kind] = r
	}
	person := byKind[KindPerson]
	if person.Detail == nil {
		t.Fatal("the person chip carries no detail")
	}
	if person.Detail.ID != "kix.person1" || person.Detail.Label != "A Placeholder" || person.Detail.Email != "placeholder@example.com" {
		t.Errorf("person detail = %+v", *person.Detail)
	}
	if got := person.InsertionIDs; len(got) != 1 || got[0] != "suggest.person1" {
		t.Errorf("person insertion ids = %v", got)
	}
	if len(person.DeletionIDs) != 0 {
		t.Errorf("person deletion ids = %v", person.DeletionIDs)
	}

	// A chip that shows the address carries no name at all: the reference
	// documents name as what is shown "instead of the person's email address",
	// and email as always present. A bare [person] tells a reader somebody is
	// there and not who, which for a policy naming its owner is the fact that
	// mattered, so the label falls back to the address.
	t.Run("a person chip shown as an address is labelled with it", func(t *testing.T) {
		raw := `{"documentId":"D","body":{"content":[
			{"startIndex":1,"endIndex":3,"paragraph":{"elements":[
			  {"startIndex":1,"endIndex":2,"person":{"personId":"kix.person2",
			    "personProperties":{"email":"placeholder@example.com"}}},
			  {"startIndex":2,"endIndex":3,"textRun":{"content":"\n"}}]}}]}}`
		d, err := Parse([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		got := d.Tabs[0].Body[0].Paragraph.Runs[0]
		if got.Detail == nil || got.Detail.Label != "placeholder@example.com" {
			t.Errorf("person detail = %+v, want the address as the label", got.Detail)
		}
	})

	date := byKind[KindDate]
	if date.Detail == nil || date.Detail.ID != "kix.date1" || date.Detail.Label != "Sep 9, 2026" {
		t.Errorf("date detail = %+v", date.Detail)
	}
	if got := date.DeletionIDs; len(got) != 1 || got[0] != "suggest.date1" {
		t.Errorf("date deletion ids = %v", got)
	}

	link := byKind[KindRichLink]
	if link.Detail == nil {
		t.Fatal("the rich link carries no detail")
	}
	if link.Detail.ID != "kix.link1" || link.Detail.Label != "A placeholder calendar entry" ||
		link.Detail.URI != "https://calendar.example.com/event/placeholder" ||
		link.Detail.MimeType != "application/vnd.google-apps.calendar-event" {
		t.Errorf("rich link detail = %+v", *link.Detail)
	}
	// One element carrying both lists is one that was suggested and then
	// suggested away, and it is a replacement whatever kind of element it is.
	if got := link.InsertionIDs; len(got) != 1 || got[0] != "suggest.link1" {
		t.Errorf("rich link insertion ids = %v", got)
	}
	if got := link.DeletionIDs; len(got) != 1 || got[0] != "suggest.link2" {
		t.Errorf("rich link deletion ids = %v", got)
	}

	auto := byKind[KindAutoText]
	if auto.Detail == nil || auto.Detail.Type != "PAGE_NUMBER" {
		t.Errorf("auto text detail = %+v", auto.Detail)
	}
	if got := auto.InsertionIDs; len(got) != 1 || got[0] != "suggest.auto1" {
		t.Errorf("auto text insertion ids = %v", got)
	}

	// The three that carry nothing but their position still carry their ids.
	for kind, ids := range map[string][]string{
		KindHorizontalRule: {"suggest.rule1", "suggest.rule2"},
		KindPageBreak:      {"", "suggest.break1"},
		KindColumnBreak:    {"suggest.break2", ""},
	} {
		r := byKind[kind]
		if got := strings.Join(r.InsertionIDs, ","); got != ids[0] {
			t.Errorf("%s insertion ids = %q, want %q", kind, got, ids[0])
		}
		if got := strings.Join(r.DeletionIDs, ","); got != ids[1] {
			t.Errorf("%s deletion ids = %q, want %q", kind, got, ids[1])
		}
	}
}

// TestAnElementTheWalkCannotNameIsReportedRatherThanDropped is the wider fix,
// and it is the point of the task. Seven kinds went missing because the default
// arm vanished rather than reported; the eighth must not. A decoder that drops
// what it does not recognise makes every reader downstream confidently wrong.
func TestAnElementTheWalkCannotNameIsReportedRatherThanDropped(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":9,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[
		   {"startIndex":1,"endIndex":5,"textRun":{"content":"See "}},
		   {"startIndex":5,"endIndex":6,"tomorrowsElement":{"suggestedInsertionIds":["suggest.x1"]}},
		   {"startIndex":6,"endIndex":9,"textRun":{"content":".\n"}}]}}]}}`
	d, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	runs := paragraphs(d.Tabs[0].Body)[0].Runs
	if len(runs) != 3 {
		t.Fatalf("len(runs) = %d, want 3: the unnamed element was dropped", len(runs))
	}
	if runs[1].Kind != KindUnknown {
		t.Errorf("run 1 kind = %q, want %q", runs[1].Kind, KindUnknown)
	}
	// The member name is what the warning downstream names, so the run has to
	// carry it: "an element gdoc does not read" says nothing a person can act
	// on, and "tomorrowsElement" says exactly what Google added.
	if runs[1].Detail == nil || runs[1].Detail.Member != "tomorrowsElement" {
		t.Errorf("run 1 detail = %+v, want the member name", runs[1].Detail)
	}
	if runs[1].StartIndex != 5 || runs[1].EndIndex != 6 {
		t.Errorf("run 1 indexes = %d..%d, want 5..6", runs[1].StartIndex, runs[1].EndIndex)
	}
}

// An element naming no member at all is still a position in the document, so it
// is still a run. There is nothing to name in the warning, and saying so is the
// honest answer.
func TestAnElementWithNoMemberIsStillARun(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":2,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":2}]}}]}}`
	d, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	runs := paragraphs(d.Tabs[0].Body)[0].Runs
	if len(runs) != 1 || runs[0].Kind != KindUnknown {
		t.Fatalf("runs = %+v, want one unknown run", runs)
	}
	if runs[0].Detail != nil && runs[0].Detail.Member != "" {
		t.Errorf("detail = %+v, want no member named", *runs[0].Detail)
	}
}

// TestAnElementTheWalkCannotNameStillCarriesItsSuggestionIDs is the same rule
// one field in. Every named member carries suggestedInsertionIds and
// suggestedDeletionIds, so the twelfth Google adds will carry them too. Reading
// the member name and dropping the ids leaves a pending change that read prints
// with no markers and that restyle --dry-run counts in neither number, which is
// the confidently-wrong reader the default arm exists to prevent.
func TestAnElementTheWalkCannotNameStillCarriesItsSuggestionIDs(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":9,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[
		   {"startIndex":1,"endIndex":5,"textRun":{"content":"See "}},
		   {"startIndex":5,"endIndex":6,"tomorrowsElement":{"suggestedInsertionIds":["suggest.x1"],
		    "suggestedDeletionIds":["suggest.x2"]}},
		   {"startIndex":6,"endIndex":9,"textRun":{"content":".\n"}}]}}]}}`
	d, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	r := paragraphs(d.Tabs[0].Body)[0].Runs[1]
	if got := strings.Join(r.InsertionIDs, ","); got != "suggest.x1" {
		t.Errorf("insertion ids = %q, want %q", got, "suggest.x1")
	}
	if got := strings.Join(r.DeletionIDs, ","); got != "suggest.x2" {
		t.Errorf("deletion ids = %q, want %q", got, "suggest.x2")
	}
}

// A member whose value is not an object carries no ids, and it must not end the
// decode either: the element's position and its name are still facts, and a
// read that failed over a member gdoc has never seen would take the whole
// document with it.
func TestAnUnnamedMemberThatIsNotAnObjectStillDecodes(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":1,"endIndex":2,"paragraph":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},
		 "elements":[{"startIndex":1,"endIndex":2,"tomorrowsFlag":true}]}}]}}`
	d, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	r := paragraphs(d.Tabs[0].Body)[0].Runs[0]
	if r.Detail == nil || r.Detail.Member != "tomorrowsFlag" {
		t.Fatalf("detail = %+v, want the member name", r.Detail)
	}
	if len(r.InsertionIDs) != 0 || len(r.DeletionIDs) != 0 {
		t.Errorf("ids = %v/%v, want none", r.InsertionIDs, r.DeletionIDs)
	}
}
