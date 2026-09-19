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
	files, pieces, warnings := Project(fixture(t, name), pics)
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
	files, pieces, warnings := Project(fixture(t, "edited-prelude.json"), nil)

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
// ampersand between two spaces, and accents.
func TestAHeadingLinkRoundTripsToItsSlug(t *testing.T) {
	files, _, _ := Project(fixture(t, "headings.json"), nil)

	for _, want := range []string{
		"[Risk & Control <ampersands>](#risk-control-ampersands)",
		"[the changes](#résumé-of-änderungen-naïve-café)",
	} {
		if !strings.Contains(files[0].Body, want) {
			t.Errorf("%s is not in the file:\n%s", want, files[0].Body)
		}
	}
}

// TestAHeadingLosesItsHouseNumberAndKeepsItsLinks is the number publish wrote
// into the heading itself, taken back out with the links that pointed at it.
func TestAHeadingLosesItsHouseNumberAndKeepsItsLinks(t *testing.T) {
	files, pieces, _ := Project(fixture(t, "publish-prelude.json"), nil)
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
	files, _, _ := Project(fixture(t, "elements.json"), nil)
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

// TestAFootnoteIsWrittenAndWarned is the footnote carried and the three things
// it costs named, because publish refuses a note that holds one.
func TestAFootnoteIsWrittenAndWarned(t *testing.T) {
	files, _, warnings := Project(fixture(t, "objects.json"), nil)
	body := files[0].Body

	if !strings.Contains(body, "[^1]") || !strings.Contains(body, "[^1]: Measured on 2026-09-06.") {
		t.Errorf("the footnote is not in the file:\n%s", body)
	}
	for _, want := range []string{
		"publish refuses a note that holds a footnote",
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
}

// TestTwoTabsGiveTwoProjections is one file per tab, each naming the tab it
// came from, and neither holding the other's text or the tab line the read
// prints.
func TestTwoTabsGiveTwoProjections(t *testing.T) {
	files, _, _ := Project(fixture(t, "two-tabs.json"), nil)

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
	files, _, warnings := Project(fixture(t, "objects.json"), names)
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
	files, pieces, _ := Project(d, nil)
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
	files, pieces, _ := Project(d, nil)
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
	files, pieces, warnings := Project(d, nil)
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
	files, pieces, warnings := Project(d, nil)

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
	files, pieces, warnings := Project(d, nil)

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
