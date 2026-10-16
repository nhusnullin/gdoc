package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/frontmatter"
	"gdoc/internal/house"
	"gdoc/internal/prelude"
)

// fixture parses a recorded Docs answer. The prelude fixtures are this
// package's own, under testdata; everything else is read from internal/docs's,
// so a golden here says what the hub gets for a document the walk already has
// an assertion about.
func fixture(t *testing.T, name string) *docs.Document {
	t.Helper()
	dir := "testdata"
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		dir = filepath.Join("..", "docs", "testdata")
	}
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("docs.Parse(%s): %v", name, err)
	}
	return d
}

// file is the one tab of a one-tab fixture as the bytes export would write,
// the gdoc: block included.
func file(t *testing.T, name string, pics PictureNames) (string, []Piece, []string) {
	t.Helper()
	files, pieces, _, warnings := Project(fixture(t, name), pics, nil)
	if len(files) != 1 {
		t.Fatalf("%s gave %d files, want 1", name, len(files))
	}
	out, err := File(files[0].Body, block(fixture(t, name).ID))
	if err != nil {
		t.Fatal(err)
	}
	return string(out), pieces, warnings
}

// block is the front matter export writes for a file of its own: one entry,
// the document it came from, and the date it was read.
func block(id string) *frontmatter.Block {
	return &frontmatter.Block{Schema: frontmatter.Schema, Documents: []frontmatter.Entry{{
		ID:       id,
		Exported: &frontmatter.Exported{At: time.Date(2026, 9, 19, 14, 0, 0, 0, time.UTC)},
	}}}
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if os.Getenv("GDOC_RECORD") != "" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s =\n%s\nwant\n%s", name, got, want)
	}
}

// kinds is the pieces as a list of their kinds, for a test that is about what
// was recognised rather than about the words it held.
func kinds(ps []Piece) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.Kind)
	}
	return out
}

func joined(ps []Piece) string {
	var sb strings.Builder
	for _, p := range ps {
		sb.WriteString(p.Kind + ": " + p.Text + "\n")
	}
	return sb.String()
}

// TestAPlainDocumentIsTheTextAndTheBlock is the file for a document with no
// house prelude: the text read prints, with its markers, and a front matter
// holding the gdoc: block and nothing else.
func TestAPlainDocumentIsTheTextAndTheBlock(t *testing.T) {
	got, pieces, _ := file(t, "elements.json", nil)
	golden(t, "plain.md", got)
	if len(pieces) != 0 {
		t.Errorf("a document with no prelude stripped %v", kinds(pieces))
	}
}

// TestAPublishedPreludeIsStrippedByLayout is the document publish made: the
// file opens at its first body heading, and every piece of the prelude is
// listed with the text it held.
func TestAPublishedPreludeIsStrippedByLayout(t *testing.T) {
	got, pieces, _ := file(t, "publish-prelude.json", nil)
	golden(t, "publish-prelude.md", got)

	want := []string{
		PieceCover, PieceTable, PieceTable, PieceLegend, PieceTable, PieceContents,
		PieceHeadingNumber, PieceHeadingNumber, PieceHeadingNumber,
	}
	if strings.Join(kinds(pieces), ",") != strings.Join(want, ",") {
		t.Errorf("stripped:\n%swant kinds %v", joined(pieces), want)
	}
	if !strings.Contains(pieces[0].Text, "The Payments Policy") {
		t.Errorf("the cover piece does not hold the title: %q", pieces[0].Text)
	}
	if !strings.Contains(pieces[1].Text, "Nail Khusnullin") {
		t.Errorf("the first table piece does not hold its cells: %q", pieces[1].Text)
	}
	if !strings.Contains(pieces[5].Text, "Contents") {
		t.Errorf("the contents piece does not hold the label above it: %q", pieces[5].Text)
	}
}

// TestARestyledPreludeIsStrippedOverItsNamedRange is the other route in: a
// document gdoc styled where it stood carries its own marker, and the marker
// is the answer.
func TestARestyledPreludeIsStrippedOverItsNamedRange(t *testing.T) {
	got, pieces, _ := file(t, "restyle-prelude.json", nil)
	golden(t, "restyle-prelude.md", got)

	want := []string{PieceCover, PieceTable, PieceTable, PieceLegend, PieceTable}
	if strings.Join(kinds(pieces), ",") != strings.Join(want, ",") {
		t.Errorf("stripped:\n%swant kinds %v", joined(pieces), want)
	}
	if strings.Contains(got, "Altery Group") {
		t.Errorf("the cover is still in the file:\n%s", got)
	}
	if !strings.Contains(got, "# Scope") {
		t.Errorf("the body is not in the file:\n%s", got)
	}
}

// TestAnEditedPreludeStaysAndWarns is the refusal to guess. A heading somebody
// added inside the cover breaks the layout the strip reads, so the whole
// document stays in the file and the warning names what stood there.
func TestAnEditedPreludeStaysAndWarns(t *testing.T) {
	files, pieces, _, warnings := Project(fixture(t, "edited-prelude.json"), nil, nil)

	if len(pieces) != 0 {
		t.Errorf("a prelude nothing recognised stripped %v", kinds(pieces))
	}
	if !strings.Contains(files[0].Body, "Altery Group") {
		t.Errorf("the cover is gone from a document nothing recognised:\n%s", files[0].Body)
	}
	// The numbers stay with the cover, because the shape that recognises them
	// as gdoc's own is the shape that did not match.
	if !strings.Contains(files[0].Body, "# 1-Scope") {
		t.Errorf("the heading number came off a prelude nothing recognised:\n%s", files[0].Body)
	}
	kept := ""
	for _, w := range warnings {
		if strings.Contains(w, "headings opening with digits") {
			kept = w
		}
	}
	if kept == "" {
		t.Fatalf("nothing named the heading numbers that stayed: %v", warnings)
	}
	for _, want := range []string{"1-Scope", "1.1-Out of scope", "2-Who decides"} {
		if !strings.Contains(kept, want) {
			t.Errorf("the warning does not name %q: %s", want, kept)
		}
	}
	// The same warning is raised on documents gdoc never published, so it may
	// not call the digits gdoc's own: it names what would make them so.
	if !strings.Contains(kept, "if gdoc publish wrote this document") {
		t.Errorf("the warning calls the numbers the house's without asking where the document came from: %s", kept)
	}
	found := ""
	for _, w := range warnings {
		if strings.Contains(w, "nothing was stripped") {
			found = w
		}
	}
	if found == "" {
		t.Fatalf("nothing warned about the prelude that stayed: %v", warnings)
	}
	for _, want := range []string{"The Payments Policy", "Nail Khusnullin", "Contents", "0 of the 3 house tables"} {
		if !strings.Contains(found, want) {
			t.Errorf("the warning does not name %q: %s", want, found)
		}
	}
}

// TestAHeadingLinkRoundTripsToItsSlug is the link into the document as the
// note writes one. The two headings are the shapes 05-edge-cases.md holds: an
// ampersand between two spaces, and accents. Both are goldmark's answer and
// not the file-name slug's: goldmark writes one hyphen per space and drops a
// letter it does not call ASCII, so these two are the headings that tell the
// rules apart. TestTheAnchorViewWritesIsTheIDBodyCollects is the other half,
// which asks goldmark itself rather than a literal.
func TestAHeadingLinkRoundTripsToItsSlug(t *testing.T) {
	files, _, _, _ := Project(fixture(t, "headings.json"), nil, nil)

	for _, want := range []string{
		"[Risk & Control <ampersands>](#risk--control-ampersands)",
		"[the changes](#rsum-of-nderungen-nave-caf)",
	} {
		if !strings.Contains(files[0].Body, want) {
			t.Errorf("%s is not in the file:\n%s", want, files[0].Body)
		}
	}
}

// TestRepointDoesNotChainRewrites is two headings whose numbers come off into
// each other: "1-2024-Report" becomes "2024-Report", which is what the second
// heading was called before its own number came off. A rewrite per entry of the
// map would run the second rule over what the first wrote, and Go's map order
// would decide which answer the file carried, so the same document exported
// twice would differ. One pass answers each link once.
func TestRepointDoesNotChainRewrites(t *testing.T) {
	md := "# 1-2024-Report\n\nSee [the report](#1-2024-report).\n\n# 2024-Report\n\nSee [the other](#2024-report).\n"

	for i := 0; i < 50; i++ {
		got, _ := stripHeadingNumbers(md)
		if !strings.Contains(got, "[the report](#2024-report)") {
			t.Fatalf("run %d moved the first link somewhere else:\n%s", i, got)
		}
		if !strings.Contains(got, "[the other](#report)") {
			t.Fatalf("run %d moved the second link somewhere else:\n%s", i, got)
		}
	}
}

// TestAHeadingLosesItsHouseNumberAndKeepsItsLinks is the number publish wrote
// into the heading itself, taken back out with the links that pointed at it.
func TestAHeadingLosesItsHouseNumberAndKeepsItsLinks(t *testing.T) {
	files, pieces, _, _ := Project(fixture(t, "publish-prelude.json"), nil, nil)
	body := files[0].Body

	for _, gone := range []string{"# 1-Scope", "## 1.1-Out of scope", "# 2-Who decides"} {
		if strings.Contains(body, gone) {
			t.Errorf("%q is still in the file:\n%s", gone, body)
		}
	}
	if !strings.Contains(body, "[Scope](#scope)") {
		t.Errorf("the link into the renumbered heading did not move:\n%s", body)
	}
	var numbers []string
	for _, p := range pieces {
		if p.Kind == PieceHeadingNumber {
			numbers = append(numbers, p.Text)
		}
	}
	want := []string{"1-Scope", "1.1-Out of scope", "2-Who decides"}
	if strings.Join(numbers, ",") != strings.Join(want, ",") {
		t.Errorf("heading number pieces = %v, want %v", numbers, want)
	}
}

// TestTheHeadingNumberIsTheHouseSeparator states the house value as a literal,
// so a house style that numbered its headings another way fails here rather
// than leaving export taking a prefix off nothing.
func TestTheHeadingNumberIsTheHouseSeparator(t *testing.T) {
	cfg, err := house.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HeadingNumbering.Level1Format != "{n}-{title}" {
		t.Errorf("heading_numbering.level1_format = %q, and export takes %q off a heading",
			cfg.HeadingNumbering.Level1Format, headingNumberSeparator)
	}
}

// TestAChipExportsLabelAndTarget is the chip in the file: the label the
// document shows, and where it points, which the read leaves to --structure.
func TestAChipExportsLabelAndTarget(t *testing.T) {
	files, _, _, _ := Project(fixture(t, "elements.json"), nil, nil)
	body := files[0].Body

	for _, want := range []string{
		"[link: A placeholder calendar entry](https://calendar.example.com/event/placeholder)",
		"[person: A Placeholder](mailto:placeholder@example.com)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s is not in the file:\n%s", want, body)
		}
	}
}

// TestAFootnoteIsWrittenAndWarned is the footnote carried and the two things
// it still costs named. The third warning went when publish stopped refusing a
// note holding a footnote: a note built from this file carries the footnote
// into the document it publishes, so nothing has to be resolved first.
func TestAFootnoteIsWrittenAndWarned(t *testing.T) {
	files, _, _, warnings := Project(fixture(t, "objects.json"), nil, nil)
	body := files[0].Body

	if !strings.Contains(body, "[^1]") || !strings.Contains(body, "[^1]: Measured on 2026-09-06.") {
		t.Errorf("the footnote is not in the file:\n%s", body)
	}
	for _, want := range []string{
		"docs/backlog/suggestions-inside-footnotes.md",
		"docs/backlog/comment-anchors-in-headers-and-footnotes.md",
	} {
		found := false
		for _, w := range warnings {
			if strings.Contains(w, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("no warning names %q: %v", want, warnings)
		}
	}
	for _, gone := range []string{"publish refuses", "footnote reference(s)"} {
		for _, w := range warnings {
			if strings.Contains(w, gone) {
				t.Errorf("a warning still says %q, and publish carries a footnote now: %q", gone, w)
			}
		}
	}
}

// TestAMarkInsideAFootnoteIsNamedAsALoss. The read keeps a footnote's text
// alone, so a mark the author wrote inside one has no colour left to read by
// the time the projection runs: the words come home plain, no envelope counts
// them, and the next build would report a debt fewer than the document holds.
// The read records which notes carried a colour, and that is the sentence.
func TestAMarkInsideAFootnoteIsNamedAsALoss(t *testing.T) {
	d := fixture(t, "coloured-footnote.json")
	if len(d.FootnoteColours) != 1 {
		t.Fatalf("the read names %v as the coloured notes, want kix.fn1 alone", d.FootnoteColours)
	}
	files, _, other, warnings := Project(d, nil, palette(t))

	if !strings.Contains(files[0].Body, "[^1]: to fill in") {
		t.Errorf("the note is not in the file:\n%s", files[0].Body)
	}
	if strings.Contains(files[0].Body, "==to fill in==") {
		t.Errorf("the mark came home after all, so this warning is the wrong fact:\n%s", files[0].Body)
	}
	if len(other) != 0 {
		t.Errorf("other = %v, want nothing: the colour is inside a footnote", other)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "docs/backlog/marks-inside-footnotes.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("no warning names the mark lost inside the footnote: %v", warnings)
	}
}

// TestAPencilMarkInsideAFootnoteIsNamedAsALoss is the text-colour half of the
// same question, and it is the reason the read hands over the hexes rather
// than a yes or a no. The pencil is a run's text colour and nothing else, so a
// note carrying it carries no background at all: a rule that counted
// backgrounds alone would let that mark come home as plain words with nothing
// said.
//
// #9A9A9A is the pencil's hex, stated as a literal, and the fixture carries it
// as the colour Docs returns, 0.6039216 on each channel.
func TestAPencilMarkInsideAFootnoteIsNamedAsALoss(t *testing.T) {
	d := fixture(t, "pencil-footnote.json")
	c, ok := d.FootnoteColours["kix.fn1"]
	if !ok || len(c.Backgrounds) != 0 {
		t.Fatalf("the read names %v, want kix.fn1 with no background: the pencil is a text colour", d.FootnoteColours)
	}
	if len(c.Colours) != 1 || !strings.EqualFold(c.Colours[0], "#9A9A9A") {
		t.Fatalf("the note's text colours are %v, want the pencil's #9A9A9A alone", c.Colours)
	}

	_, _, other, warnings := Project(d, nil, palette(t))

	if len(other) != 0 {
		t.Errorf("other = %v, want nothing: the colour is inside a footnote", other)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "docs/backlog/marks-inside-footnotes.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("no warning names the pencil mark lost inside the footnote: %v", warnings)
	}
}

// TestAFootnoteWithNoColourIsNotWarnedAboutAMark is the other half: the
// sentence is a fact about this document and not a line every footnote carries.
func TestAFootnoteWithNoColourIsNotWarnedAboutAMark(t *testing.T) {
	_, _, _, warnings := Project(fixture(t, "objects.json"), nil, palette(t))

	for _, w := range warnings {
		if strings.Contains(w, "marks-inside-footnotes") {
			t.Errorf("a note carrying no colour was warned about: %q", w)
		}
	}
}

// TestTwoTabsGiveTwoProjections is one file per tab, each naming the tab it
// came from, and neither holding the other's text or the tab line the read
// prints.
func TestTwoTabsGiveTwoProjections(t *testing.T) {
	files, _, _, _ := Project(fixture(t, "two-tabs.json"), nil, nil)

	if len(files) != 2 {
		t.Fatalf("two tabs gave %d files", len(files))
	}
	if files[0].TabID != "t.0" || files[0].TabTitle != "Overview" {
		t.Errorf("first file = %+v", files[0])
	}
	if files[1].TabID != "t.1" || files[1].TabTitle != "Detail" {
		t.Errorf("second file = %+v", files[1])
	}
	for i, f := range files {
		if strings.Contains(f.Body, "<!-- tab ") {
			t.Errorf("file %d carries a tab line, and a file is one tab:\n%s", i, f.Body)
		}
	}
	if strings.Contains(files[0].Body, "second tab") || strings.Contains(files[1].Body, "first tab") {
		t.Errorf("a file holds the other tab's text:\n%s\n---\n%s", files[0].Body, files[1].Body)
	}
}

// TestAPictureIsWrittenWhereItStands is the caller's own file name reaching
// the text, with nothing invented for a picture that has no file.
func TestAPictureIsWrittenWhereItStands(t *testing.T) {
	names := func(id string) string {
		if id == "kix.draw1" {
			return ""
		}
		return "![](assets/note-1.png)"
	}
	files, _, _, warnings := Project(fixture(t, "objects.json"), names, nil)
	body := files[0].Body

	if !strings.Contains(body, "![](assets/note-1.png)") {
		t.Errorf("the picture's file is not in the text:\n%s", body)
	}
	if !strings.Contains(body, "[drawing]") {
		t.Errorf("a drawing with no file lost its placeholder:\n%s", body)
	}
	for _, w := range warnings {
		if strings.Contains(w, "an image at index") {
			t.Errorf("a picture that was written is still a warning: %s", w)
		}
	}
}

// para is one paragraph of one style, holding one run of text.
func para(style, text string) docs.Block {
	return docs.Block{Paragraph: &docs.Paragraph{
		Style: style,
		Runs:  []docs.Run{{Kind: docs.KindText, Text: text + "\n", StartIndex: 1, EndIndex: 1 + len([]rune(text)) + 1}},
	}}
}

// TestAPlainDocumentKeepsAHeadingThatOpensWithANumber is the heading number
// rule held to the documents it is about. "2024-2025 Budget" is a heading
// somebody wrote, not a number publish put there, and taking the "2024-" off
// it would delete the author's own words with nothing to put them back.
func TestAPlainDocumentKeepsAHeadingThatOpensWithANumber(t *testing.T) {
	d := &docs.Document{ID: "doc", Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{
		para("HEADING_1", "2024-2025 Budget"),
		para("HEADING_2", "1-on-1 meetings"),
		para("NORMAL_TEXT", "Prose."),
	}}}}
	files, pieces, _, _ := Project(d, nil, nil)
	for _, want := range []string{"# 2024-2025 Budget", "## 1-on-1 meetings"} {
		if !strings.Contains(files[0].Body, want) {
			t.Errorf("%q lost its own words:\n%s", want, files[0].Body)
		}
	}
	for _, p := range pieces {
		t.Errorf("a document with no house prelude was stripped of a %s (%q)", p.Kind, p.Text)
	}
}

// TestARestyledDocumentKeepsAHeadingThatOpensWithANumber is the heading number
// rule held to the other route in. A document restyle made wears gdoc's marker
// over its prelude and none of its headings were numbered by gdoc: heading
// numbering is out of the restyle route, which internal/prelude's doc.go
// states. So "2024-2025 Budget" there is somebody's own words, the same as in a
// document gdoc never touched.
func TestARestyledDocumentKeepsAHeadingThatOpensWithANumber(t *testing.T) {
	at := func(style, text string, start int) docs.Block {
		return docs.Block{Paragraph: &docs.Paragraph{
			Style:      style,
			StartIndex: start,
			Runs:       []docs.Run{{Kind: docs.KindText, Text: text + "\n", StartIndex: start, EndIndex: start + len([]rune(text)) + 1}},
		}}
	}
	d := &docs.Document{ID: "doc", Tabs: []docs.Tab{{
		ID: "t.0",
		NamedRanges: []docs.NamedRange{{
			ID: "kix.marker1", Name: prelude.MarkerName, Tab: "t.0",
			Ranges: []docs.Range{{Tab: "t.0", Start: 1, End: 20}},
		}},
		Body: []docs.Block{
			at("TITLE", "Altery Group", 1),
			at("HEADING_1", "2024-2025 Budget", 40),
			at("HEADING_2", "1-on-1 meetings", 80),
			at("NORMAL_TEXT", "Prose.", 120),
		},
	}}}
	files, pieces, _, _ := Project(d, nil, nil)
	if strings.Contains(files[0].Body, "Altery Group") {
		t.Errorf("the marked prelude is still in the file:\n%s", files[0].Body)
	}
	for _, want := range []string{"# 2024-2025 Budget", "## 1-on-1 meetings"} {
		if !strings.Contains(files[0].Body, want) {
			t.Errorf("%q lost its own words:\n%s", want, files[0].Body)
		}
	}
	for _, p := range pieces {
		if p.Kind == PieceHeadingNumber {
			t.Errorf("a restyled document was stripped of a heading number (%q)", p.Text)
		}
	}
}

// TestATabWithNoLevelOneHeadingIsNotAPrelude is the layout rule's other edge.
// The span in front of the first level-one heading is the whole tab when there
// is no such heading, so every table in the document would be counted as a
// house table and somebody's own front page would come out of their note.
func TestATabWithNoLevelOneHeadingIsNotAPrelude(t *testing.T) {
	table := docs.Block{Table: &docs.Table{Rows: [][]docs.Cell{{{Blocks: []docs.Block{
		para("NORMAL_TEXT", "a cell"),
	}}}}}}
	d := &docs.Document{ID: "doc", Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{
		para("TITLE", "Somebody's own front page"),
		docs.Block{TOC: &docs.TOC{Blocks: []docs.Block{para("NORMAL_TEXT", "Scope")}}},
		para("HEADING_2", "Scope"),
		table, table, table,
	}}}}
	files, pieces, _, warnings := Project(d, nil, nil)
	if !strings.Contains(files[0].Body, "Somebody's own front page") {
		t.Errorf("the front page is gone from a document nothing recognised:\n%s", files[0].Body)
	}
	if len(pieces) != 0 {
		t.Errorf("pieces = %v, want nothing stripped", pieces)
	}
	found := ""
	for _, w := range warnings {
		if strings.Contains(w, "nothing was stripped") {
			found = w
		}
	}
	if found == "" {
		t.Fatalf("nothing warned about the prelude that stayed: %v", warnings)
	}
	if !strings.Contains(found, "no level-one heading") {
		t.Errorf("the warning does not say why: %s", found)
	}
}

// TestAPlainDocumentWithAContentsListIsNotToldItsNumbersAreTheHouses holds the
// wording of the kept-numbers warning to the documents that reach it. A tab
// with a contents list and no house tables is the common foreign shape, so the
// warning may not tell the session reading it that "2024-2025 Budget" opens
// with a number gdoc wrote: the words are the author's until the session says
// the document came from publish.
func TestAPlainDocumentWithAContentsListIsNotToldItsNumbersAreTheHouses(t *testing.T) {
	d := &docs.Document{ID: "doc", Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{
		para("TITLE", "Somebody's own budget"),
		{TOC: &docs.TOC{Blocks: []docs.Block{para("NORMAL_TEXT", "2024-2025 Budget")}}},
		para("HEADING_1", "2024-2025 Budget"),
		para("HEADING_2", "1-on-1 meetings"),
	}}}}
	files, pieces, _, warnings := Project(d, nil, nil)

	for _, want := range []string{"# 2024-2025 Budget", "## 1-on-1 meetings"} {
		if !strings.Contains(files[0].Body, want) {
			t.Errorf("%q lost its own words:\n%s", want, files[0].Body)
		}
	}
	if len(pieces) != 0 {
		t.Errorf("a document with no house prelude was stripped of %v", kinds(pieces))
	}
	for _, w := range warnings {
		if strings.Contains(w, "house number") && !strings.Contains(w, "if gdoc publish wrote this document") {
			t.Errorf("the author's own heading was called a house number: %s", w)
		}
	}
}

// TestARefusedMarkerStripsNothing is the restyle rule held where gdoc's own
// record of the prelude cannot be read. Two ranges wearing the marker's name
// is prelude's refusal, and falling back to the layout would strip the span in
// doubt and take the heading numbers off a document restyle made, which never
// wrote one.
func TestARefusedMarkerStripsNothing(t *testing.T) {
	marker := func(id string, start, end int) docs.NamedRange {
		return docs.NamedRange{
			ID: id, Name: prelude.MarkerName, Tab: "t.0",
			Ranges: []docs.Range{{Tab: "t.0", Start: start, End: end}},
		}
	}
	table := docs.Block{Table: &docs.Table{Rows: [][]docs.Cell{{{Blocks: []docs.Block{
		para("NORMAL_TEXT", "a cell"),
	}}}}}}
	d := &docs.Document{ID: "doc", Tabs: []docs.Tab{{
		ID:          "t.0",
		NamedRanges: []docs.NamedRange{marker("kix.one", 1, 20), marker("kix.two", 1, 20)},
		Body: []docs.Block{
			para("TITLE", "Altery Group"),
			table, table, table,
			para("NORMAL_TEXT", "Contents"),
			{TOC: &docs.TOC{Blocks: []docs.Block{para("NORMAL_TEXT", "1-Scope")}}},
			para("HEADING_1", "1-Scope"),
		},
	}}}
	files, pieces, _, warnings := Project(d, nil, nil)

	if !strings.Contains(files[0].Body, "Altery Group") {
		t.Errorf("a prelude whose marker was refused was stripped by the layout:\n%s", files[0].Body)
	}
	if !strings.Contains(files[0].Body, "# 1-Scope") {
		t.Errorf("the heading number came off under a refused marker:\n%s", files[0].Body)
	}
	if len(pieces) != 0 {
		t.Errorf("pieces = %v, want nothing stripped", kinds(pieces))
	}
	found := ""
	for _, w := range warnings {
		if strings.Contains(w, "two "+prelude.MarkerName+" markers") {
			found = w
		}
	}
	if found == "" {
		t.Fatalf("nothing named the two markers: %v", warnings)
	}
	if !strings.Contains(found, "nothing was stripped") {
		t.Errorf("the warning does not say what was done: %s", found)
	}
}

// TestACommentInTheStrippedPreludeIsNotMarkedInTheBody holds the rule that a
// comment anchored in words the prelude strip took out leaves no marker in the
// file. The markers are armed for the whole tab, so without this the first
// surviving paragraph collects every marker the cover held, and the file says a
// comment covers text it does not contain. It is named instead, through the
// route a range naming a tab the document lacks already takes.
func TestACommentInTheStrippedPreludeIsNotMarkedInTheBody(t *testing.T) {
	d := fixture(t, "publish-prelude.json")
	d.CommentRanges = map[string]docs.Range{
		"CoverComment": {Tab: "t.0", Start: 15, End: 34},
	}

	files, _, _, warnings := Project(d, nil, nil)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if strings.Contains(files[0].Body, "CoverComment") {
		t.Errorf("a comment on the stripped cover left a marker in the body:\n%s", files[0].Body)
	}
	if !strings.HasPrefix(files[0].Body, "# Scope") {
		t.Errorf("the body does not open at its first heading:\n%s", files[0].Body)
	}
	named := false
	for _, w := range warnings {
		if strings.Contains(w, "CoverComment") {
			named = true
		}
	}
	if !named {
		t.Fatalf("nothing named the comment that was stripped with the prelude: %v", warnings)
	}
}

// TestACommentNamingAMissingTabIsNamed holds that a range whose tab is not in
// the document is warned about here too. The projection is handed one tab at a
// time, and a range naming no tab at all would otherwise be filtered out of
// every one of them and reach no warning on any route.
func TestACommentNamingAMissingTabIsNamed(t *testing.T) {
	d := fixture(t, "publish-prelude.json")
	d.CommentRanges = map[string]docs.Range{
		"Ghost": {Tab: "t.9", Start: 300, End: 310},
	}

	_, _, _, warnings := Project(d, nil, nil)
	named := false
	for _, w := range warnings {
		if strings.Contains(w, "Ghost") && strings.Contains(w, "t.9") {
			named = true
		}
	}
	if !named {
		t.Fatalf("nothing named the comment whose tab the document does not have: %v", warnings)
	}
}

// TestABodyOpeningWithTheDelimiterKeepsItsFirstParagraph: a document whose
// first paragraph is the three characters that open front matter is a document
// like any other. The file export writes carries every word of it, and reads
// back with the gdoc: block as its front matter and nothing else.
func TestABodyOpeningWithTheDelimiterKeepsItsFirstParagraph(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
	}{
		{name: "a second delimiter further down", body: "---\n\nThe first words.\n\n---\n\nMore words.\n"},
		{name: "no second delimiter at all", body: "---\n\nThe only words.\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			out, err := File(c.body, block(testDocID))
			if err != nil {
				t.Fatalf("File: %v", err)
			}
			for _, want := range []string{"The first words.", "More words.", "The only words."} {
				if strings.Contains(c.body, want) && !strings.Contains(string(out), want) {
					t.Errorf("the file lost %q:\n%s", want, out)
				}
			}
			got, err := frontmatter.Read(out)
			if err != nil {
				t.Fatalf("the file does not read back: %v\n%s", err, out)
			}
			if got == nil || len(got.Documents) != 1 || got.Documents[0].ID != testDocID {
				t.Errorf("the block read back is %+v, want the one entry for the document:\n%s", got, out)
			}
		})
	}
}

// TestAFootnoteWhoseOnlyColourIsALinksIsNotWarnedAboutAMark. A link inserted
// in Google Docs carries the link blue as a direct text colour on its run, and
// a footnote is where a citation lives, so most footnotes in a document gdoc
// did not write carry one. A text colour is how a document is designed unless
// it is the one the house claims, which is the rule otherColours already
// states: a note coloured only by its own link lost no mark, and asking the
// person whether to mark those words again asks about a debt that never
// existed.
func TestAFootnoteWhoseOnlyColourIsALinksIsNotWarnedAboutAMark(t *testing.T) {
	d := fixture(t, "linked-footnote.json")

	_, _, _, warnings := Project(d, nil, palette(t))

	for _, w := range warnings {
		if strings.Contains(w, "docs/backlog/marks-inside-footnotes.md") {
			t.Errorf("the link's own blue was named as a mark that came home plain: %q", w)
		}
	}
}

// TestWordsThatWouldReadAsAFootnoteMarkAreNamedAsALoss. The projection writes a
// real note's reference as "[^n]", so the document's own words holding that
// shape are read by the next build as a mark of gdoc's. A label the file does
// not define refuses the file by line, blaming the file rather than the export;
// a label it does define grows a second reference to a note the document never
// had, and the author's own words go with it. Neither is something a session
// can see, so the loss is named.
func TestWordsThatWouldReadAsAFootnoteMarkAreNamedAsALoss(t *testing.T) {
	// The words are the document's own: no footnote run, so nothing gdoc wrote
	// is in this paragraph at all.
	_, _, _, warnings := Project(oneMarked(docs.Run{Text: "see [^1] and [^note] here"}), nil, palette(t))

	for _, want := range []string{"[^1]", "[^note]"} {
		found := false
		for _, w := range warnings {
			if strings.Contains(w, want) && strings.Contains(w, "footnote mark") {
				found = true
			}
		}
		if !found {
			t.Errorf("no warning names %s as a footnote mark the next build would read: %v", want, warnings)
		}
	}
}

// The other half: a document whose prose holds no such shape is not warned
// about one, so the sentence is a fact about this document rather than a line
// every export carries.
func TestProseWithNoFootnoteMarkIsNotWarnedAboutOne(t *testing.T) {
	_, _, _, warnings := Project(oneMarked(docs.Run{Text: "see [the note] and ^1 here"}), nil, palette(t))

	for _, w := range warnings {
		if strings.Contains(w, "footnote mark") {
			t.Errorf("a warning names a footnote mark in words that hold none: %q", w)
		}
	}
}

// headed is a document of one heading paragraph holding the runs passed, so a
// test can put a mark or a footnote reference in a heading and read back what
// the file holds. The runs carry their own indexes, which is what the warnings
// name.
func headed(rs ...docs.Run) *docs.Document {
	at := 1
	for i := range rs {
		if rs[i].Kind == "" {
			rs[i].Kind = docs.KindText
		}
		rs[i].StartIndex = at
		at += len([]rune(rs[i].Text))
		if rs[i].Kind != docs.KindText {
			at = rs[i].StartIndex + 1
		}
		rs[i].EndIndex = at
	}
	rs = append(rs, docs.Run{Kind: docs.KindText, Text: "\n", StartIndex: at, EndIndex: at + 1})
	return &docs.Document{
		ID: "1head", Title: "One heading",
		Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{{Paragraph: &docs.Paragraph{
			Style: "HEADING_2", Runs: rs,
		}}}}},
	}
}

// TestAMarkInsideAHeadingComesHomeAsItsWords. internal/body refuses a mark
// inside a heading by line, because Word copies a heading into the contents
// field and the mark would be in the contents list as well as on the page. So
// a highlighted heading written home as "## Scope ==to fill in==" is a file
// the next build refuses, blaming the file rather than the export: the words
// go home alone and the loss is named, which is what every other mark that
// cannot be written does.
func TestAMarkInsideAHeadingComesHomeAsItsWords(t *testing.T) {
	files, _, _, warnings := Project(headed(
		docs.Run{Text: "Scope "},
		docs.Run{Text: "to fill in", Background: "#FFD500"},
	), nil, palette(t))

	if len(files) != 1 {
		t.Fatalf("the document gave %d files, want 1", len(files))
	}
	if body := files[0].Body; !strings.Contains(body, "## Scope to fill in") || strings.Contains(body, "==") {
		t.Errorf("the heading came home as\n%q\nwant the words alone", body)
	}
	named := false
	for _, w := range warnings {
		if strings.Contains(w, "heading") && strings.Contains(w, "contents") {
			named = true
		}
	}
	if !named {
		t.Errorf("no warning names the mark a heading cannot carry: %v", warnings)
	}
}

// The other half: a mark in a paragraph that is not a heading still comes home
// as its syntax, so the rule is about the heading and not about every mark.
func TestAMarkOutsideAHeadingIsNotNamedAsAHeadingLoss(t *testing.T) {
	files, _, _, warnings := Project(oneMarked(docs.Run{Text: "to fill in", Background: "#FFD500"}), nil, palette(t))

	if body := files[0].Body; !strings.Contains(body, "==to fill in==") {
		t.Errorf("the mark came home as\n%q\nwant its syntax", body)
	}
	for _, w := range warnings {
		if strings.Contains(w, "heading") {
			t.Errorf("a warning names a heading in a document that has none: %q", w)
		}
	}
}

// TestAFootnoteMarkInsideAHeadingIsNamedAsALoss. The reference is written where
// the document holds it, because the note's own words are written under the
// body and a definition nothing marks is refused as well. So the file the
// export writes is one the next build refuses either way, and the one thing
// this package can do is say which words have to move.
func TestAFootnoteMarkInsideAHeadingIsNamedAsALoss(t *testing.T) {
	d := headed(docs.Run{Text: "Scope "}, docs.Run{Kind: docs.KindFootnoteRef, FootnoteID: "f1", Text: "1"})
	d.Footnotes = map[string]string{"f1": "The note's words."}

	_, _, _, warnings := Project(d, nil, palette(t))

	named := false
	for _, w := range warnings {
		if strings.Contains(w, "footnote mark") && strings.Contains(w, "heading") {
			named = true
		}
	}
	if !named {
		t.Errorf("no warning names the footnote mark inside the heading: %v", warnings)
	}
}

// TestAFootnoteWhoseOwnWordsWouldReadAsAFootnoteMarkIsNamedAsALoss. A note's
// text is written under the body as "[^n]: words", and those words are the
// document's own: one of them holding "[^9]" is read by the next build as a
// mark of gdoc's, which refuses the file by line and blames the file rather
// than the export. The scan over the prose did not reach them, so the one
// route that writes a note's words said nothing.
func TestAFootnoteWhoseOwnWordsWouldReadAsAFootnoteMarkIsNamedAsALoss(t *testing.T) {
	d := oneMarked(docs.Run{Text: "words "})
	d.Tabs[0].Body[0].Paragraph.Runs = append(d.Tabs[0].Body[0].Paragraph.Runs,
		docs.Run{Kind: docs.KindFootnoteRef, FootnoteID: "f1", Text: "1", StartIndex: 8, EndIndex: 9})
	d.Footnotes = map[string]string{"f1": "see [^9] in the note"}

	_, _, _, warnings := Project(d, nil, palette(t))

	named := false
	for _, w := range warnings {
		if strings.Contains(w, "[^9]") && strings.Contains(w, "footnote mark") {
			named = true
		}
	}
	if !named {
		t.Errorf("no warning names the shape inside the note's own words: %v", warnings)
	}
}

// TestWordsThatWouldReadAsAMarksSpanAreNamedAsALoss. The pencil comes home as
// Pandoc's span, so the attribute it closes with is syntax of gdoc's in the
// file: the document's own words holding it are read by the next build as that
// syntax and not as the words the document shows. Two of the three shapes
// refuse the file by line and the third publishes a grey phrase the document
// never carried, with the brackets and the attribute dropped from the words.
func TestWordsThatWouldReadAsAMarksSpanAreNamedAsALoss(t *testing.T) {
	for _, prose := range []string{
		"the {.pencil} attribute",
		"ends with ]{.pencil} here",
		"write [x]{.pencil} to mark",
	} {
		_, _, _, warnings := Project(oneMarked(docs.Run{Text: prose}), nil, palette(t))

		named := false
		for _, w := range warnings {
			if strings.Contains(w, "{.pencil}") {
				named = true
			}
		}
		if !named {
			t.Errorf("%q: no warning names the span syntax the next build would read: %v", prose, warnings)
		}
	}
}

// The other half: prose holding no such shape is not warned about one.
func TestProseWithNoSpanSyntaxIsNotWarnedAboutOne(t *testing.T) {
	_, _, _, warnings := Project(oneMarked(docs.Run{Text: "a pencil note about .pencil here"}), nil, palette(t))

	for _, w := range warnings {
		if strings.Contains(w, "{.pencil}") {
			t.Errorf("a warning names span syntax in words that hold none: %q", w)
		}
	}
}
