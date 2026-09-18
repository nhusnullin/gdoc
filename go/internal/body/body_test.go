package body

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/house"
	"github.com/beevik/etree"
)

// Every house value in this file is a literal on purpose. A test that reads
// the constant it checks is a mirror: change the config and the assertion
// follows it and still passes. See CLAUDE.md, "A house-style test must never
// read the constant it tests".

var update = flag.Bool("update", false, "rewrite the golden bodies under testdata")

func config(t *testing.T) *house.Config {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("house.Load: %v", err)
	}
	return cfg
}

// render walks one piece of markdown with heading numbering on and the test's
// own directory as the place a picture is looked for.
func walk(t *testing.T, markdown string) Result {
	t.Helper()
	out, err := Render(config(t), []byte(markdown), "testdata/docs", true)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return out
}

// text serialises the blocks so a test can look for what reached the part.
func serialise(t *testing.T, blocks []*etree.Element) string {
	t.Helper()
	doc := etree.NewDocument()
	root := doc.CreateElement("w:body")
	for _, block := range blocks {
		root.AddChild(block.Copy())
	}
	doc.Indent(2)
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatalf("serialise: %v", err)
	}
	return buf.String()
}

// stripFrontMatter drops a note's YAML block. Reading it is internal/cover's,
// and this package is given the body alone.
func stripFrontMatter(src string) string {
	if !strings.HasPrefix(src, "---\n") {
		return src
	}
	if end := strings.Index(src[4:], "\n---\n"); end >= 0 {
		return src[4+end+5:]
	}
	return src
}

// find returns the first descendant with the given tag, or nil.
func find(root *etree.Element, tag string) *etree.Element {
	if root.Tag == tag || root.FullTag() == tag {
		return root
	}
	for _, child := range root.ChildElements() {
		if found := find(child, tag); found != nil {
			return found
		}
	}
	return nil
}

// findAll collects every descendant with the given tag, the element itself
// included.
func findAll(root *etree.Element, tag string) []*etree.Element {
	var out []*etree.Element
	var walk func(*etree.Element)
	walk = func(e *etree.Element) {
		if e.FullTag() == tag {
			out = append(out, e)
		}
		for _, child := range e.ChildElements() {
			walk(child)
		}
	}
	walk(root)
	return out
}

func attr(t *testing.T, e *etree.Element, name string) string {
	t.Helper()
	if e == nil {
		t.Fatalf("no element to read %s from", name)
	}
	a := e.SelectAttr(name)
	if a == nil {
		t.Fatalf("<%s> carries no %s", e.FullTag(), name)
	}
	return a.Value
}

// TestTheSixDocumentsRenderToTheirGoldens is the whole walker at once. The
// goldens are the specification: the same markdown gives the same bytes, so a
// change to any builder shows up as a diff rather than as a silent difference
// in a published document.
func TestTheSixDocumentsRenderToTheirGoldens(t *testing.T) {
	names, err := filepath.Glob("testdata/docs/*.md")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(names) != 6 {
		t.Fatalf("testdata/docs holds %d documents, want the spike's six", len(names))
	}
	for _, name := range names {
		t.Run(filepath.Base(name), func(t *testing.T) {
			src, err := os.ReadFile(name)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			out, err := Render(config(t), []byte(stripFrontMatter(string(src))),
				"testdata/docs", true)
			if err != nil {
				t.Fatalf("Render %s: %v", name, err)
			}
			got := serialise(t, out.Blocks)
			golden := filepath.Join("testdata/golden",
				strings.TrimSuffix(filepath.Base(name), ".md")+".xml")
			if *update {
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatalf("write %s: %v", golden, err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read %s: %v (run go test -update to write it)", golden, err)
			}
			if got != string(want) {
				t.Errorf("%s does not match %s. Run go test -update and read the diff",
					name, golden)
			}
		})
	}
}

// TestAPipeTableCarriesTheHouseRecipe states the house table recipe as
// literals.
func TestAPipeTableCarriesTheHouseRecipe(t *testing.T) {
	out := walk(t, "| Field | Value |\n|---|---|\n| One | 1 |\n| Two | 2 |\n")

	if out.Counts.Tables != 1 {
		t.Fatalf("counted %d tables, want 1", out.Counts.Tables)
	}
	var tbl *etree.Element
	for _, block := range out.Blocks {
		if block.FullTag() == "w:tbl" {
			tbl = block
		}
	}
	if tbl == nil {
		t.Fatalf("the blocks carry no w:tbl:\n%s", serialise(t, out.Blocks))
	}

	if got := attr(t, find(tbl, "w:tblLayout"), "w:type"); got != "fixed" {
		t.Errorf("table layout is %q, want fixed", got)
	}
	// A4 less the two 51.05pt margins is 493.18pt, which is 9864 twips.
	if got := attr(t, find(tbl, "w:tblW"), "w:w"); got != "9864" {
		t.Errorf("table width is %s twips, want 9864", got)
	}
	widths := 0
	for _, col := range findAll(tbl, "w:gridCol") {
		w := attr(t, col, "w:w")
		n := 0
		for _, r := range w {
			n = n*10 + int(r-'0')
		}
		widths += n
	}
	if widths != 9864 {
		t.Errorf("the columns sum to %d twips, want the usable width 9864", widths)
	}
	for _, borders := range append(findAll(tbl, "w:tblBorders"), findAll(tbl, "w:tcBorders")...) {
		for _, edge := range borders.ChildElements() {
			if got := attr(t, edge, "w:sz"); got != "4" {
				t.Errorf("the %s border is sz %s, want 4", edge.FullTag(), got)
			}
			if got := attr(t, edge, "w:color"); got != "c9c9c9" {
				t.Errorf("the %s border is %s, want the grey grid c9c9c9", edge.FullTag(), got)
			}
		}
	}
	rows := findAll(tbl, "w:tr")
	if len(rows) != 3 {
		t.Fatalf("the table has %d rows, want 3", len(rows))
	}
	for i, row := range rows {
		if find(row, "w:cantSplit") == nil {
			t.Errorf("row %d may split across a page", i)
		}
	}
	if got := attr(t, find(rows[0], "w:tblHeader"), "w:val"); got != "1" {
		t.Errorf("the header row repeats as %q, want 1", got)
	}
	if got := attr(t, find(rows[1], "w:tblHeader"), "w:val"); got != "0" {
		t.Errorf("a body row repeats as %q, want 0", got)
	}
	fills := []string{}
	for _, row := range rows {
		fills = append(fills, attr(t, find(row, "w:shd"), "w:fill"))
	}
	if fills[0] != "bdcdd2" {
		t.Errorf("the header fill is %s, want bdcdd2", fills[0])
	}
	if fills[1] != "ffffff" {
		t.Errorf("the first body row is filled %s, want ffffff", fills[1])
	}
	if fills[2] != "f3f8f9" {
		t.Errorf("the banded row is filled %s, want f3f8f9", fills[2])
	}
	if got := attr(t, find(rows[0], "w:trHeight"), "w:val"); got != "390" {
		t.Errorf("the header row is %s twips high, want 390", got)
	}
	if got := attr(t, find(rows[1], "w:trHeight"), "w:val"); got != "300" {
		t.Errorf("a body row is %s twips high, want 300", got)
	}
}

// TestAnAuthoredHeadingNumberIsKept is the rule that the author's numbers win.
func TestAnAuthoredHeadingNumberIsKept(t *testing.T) {
	out := walk(t, "# Introduction\n\n## 2. Scope\n\n# Controls\n")
	got := serialise(t, out.Blocks)
	for _, want := range []string{">1-Introduction<", ">2. Scope<", ">2-Controls<"} {
		if !strings.Contains(got, want) {
			t.Errorf("the body carries no %s:\n%s", want, got)
		}
	}
	if strings.Contains(got, "1.1-2. Scope") {
		t.Errorf("the authored number was doubled:\n%s", got)
	}
}

// TestNumberingOffNumbersNothing is the front matter's heading_numbering key.
func TestNumberingOffNumbersNothing(t *testing.T) {
	out, err := Render(config(t), []byte("# Introduction\n\n## Scope\n"), "", false)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	got := serialise(t, out.Blocks)
	if !strings.Contains(got, ">Introduction<") || strings.Contains(got, "1-Introduction") {
		t.Errorf("a heading was numbered with numbering off:\n%s", got)
	}
	if out.Counts.Headings != 2 {
		t.Errorf("counted %d headings, want 2", out.Counts.Headings)
	}
}

// TestNestedBulletsUseTheirOwnLevel walks three levels down.
func TestNestedBulletsUseTheirOwnLevel(t *testing.T) {
	out := walk(t, "- one\n    - two\n        - three\n")
	var levels []string
	for _, block := range out.Blocks {
		if lvl := find(block, "w:ilvl"); lvl != nil {
			levels = append(levels, attr(t, lvl, "w:val"))
		}
	}
	if strings.Join(levels, ",") != "0,1,2" {
		t.Errorf("the bullets sit at levels %v, want 0,1,2", levels)
	}
	for _, block := range out.Blocks {
		if id := find(block, "w:numId"); id != nil {
			if got := attr(t, id, "w:val"); got != "1" {
				t.Errorf("a bullet names list %s, want the bulleted list 1", got)
			}
		}
	}
	if out.Counts.Lists != 1 {
		t.Errorf("counted %d lists, want 1", out.Counts.Lists)
	}
}

// TestANumberedListNamesTheNumberedList keeps the two lists apart.
func TestANumberedListNamesTheNumberedList(t *testing.T) {
	out := walk(t, "1. first\n2. second\n")
	for _, block := range out.Blocks {
		if id := find(block, "w:numId"); id != nil {
			if got := attr(t, id, "w:val"); got != "2" {
				t.Errorf("a numbered item names list %s, want 2", got)
			}
		}
	}
}

// TestMarkIsAYellowHighlight is the ==mark== syntax the house markdown uses to
// say a human still has to fill something in.
func TestMarkIsAYellowHighlight(t *testing.T) {
	out := walk(t, "A sentence with ==a gap== in it.\n")
	got := serialise(t, out.Blocks)
	if !strings.Contains(got, `<w:highlight w:val="yellow"/>`) {
		t.Errorf("the marked span carries no yellow highlight:\n%s", got)
	}
	if !strings.Contains(got, ">a gap<") {
		t.Errorf("the marked words are missing:\n%s", got)
	}
	if strings.Contains(got, "==") {
		t.Errorf("the delimiters reached the document:\n%s", got)
	}
}

// TestALinkIsAHyperlinkWithARelationship: the destination is a package
// relationship, so a reader can follow it.
func TestALinkIsAHyperlinkWithARelationship(t *testing.T) {
	out := walk(t, "See [the policy](https://example.com/p?a=1&b=2) for more.\n")
	got := serialise(t, out.Blocks)
	if !strings.Contains(got, "<w:hyperlink") {
		t.Errorf("the link is not a w:hyperlink:\n%s", got)
	}
	if len(out.Media) != 1 {
		t.Fatalf("the result carries %d relationships, want the link's one", len(out.Media))
	}
	link := out.Media[0]
	if link.Target != "https://example.com/p?a=1&b=2" {
		t.Errorf("the relationship points at %q", link.Target)
	}
	if link.RelID != "rId8" {
		t.Errorf("the link took %s, and the shell's own seven end at rId7", link.RelID)
	}
	if !strings.Contains(got, `r:id="rId8"`) {
		t.Errorf("the hyperlink names no relationship:\n%s", got)
	}
}

// TestAnImageIsReadAndSizedToTheColumn: a picture wider than the text column is
// scaled down, and never up.
func TestAnImageIsReadAndSizedToTheColumn(t *testing.T) {
	out := walk(t, "![A wide banner](wide-banner.png)\n")
	if out.Counts.Images != 1 {
		t.Fatalf("counted %d images, want 1", out.Counts.Images)
	}
	if len(out.Media) != 1 {
		t.Fatalf("the result carries %d media, want 1", len(out.Media))
	}
	if out.Media[0].Name != "image1.png" || len(out.Media[0].Data) == 0 {
		t.Errorf("the image part is %q with %d bytes", out.Media[0].Name, len(out.Media[0].Data))
	}
	got := serialise(t, out.Blocks)
	// 493.18pt of usable width is 6263330 EMU: a picture wider than the column
	// is scaled to exactly that.
	if !strings.Contains(got, `cx="6263330"`) {
		t.Errorf("the picture was not sized to the text column:\n%s", got)
	}
}

// TestASmallImageKeepsItsOwnSize is the other half of that rule.
func TestASmallImageKeepsItsOwnSize(t *testing.T) {
	out := walk(t, "![A badge](badge.png)\n")
	got := serialise(t, out.Blocks)
	if strings.Contains(got, `cx="6263330"`) {
		t.Errorf("a small picture was stretched to the column:\n%s", got)
	}
}

// TestARemoteImageIsRefusedNamingTheLine: nothing here downloads.
func TestARemoteImageIsRefusedNamingTheLine(t *testing.T) {
	_, err := Render(config(t), []byte("Words.\n\n![A picture](https://example.com/p.png)\n"),
		"testdata/docs", true)
	if err == nil {
		t.Fatalf("a remote picture was accepted")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("the refusal does not name line 3: %v", err)
	}
}

// TestAMissingImageIsRefusedNamingTheFile keeps a lost picture loud.
func TestAMissingImageIsRefusedNamingTheFile(t *testing.T) {
	_, err := Render(config(t), []byte("![Gone](no-such-file.png)\n"), "testdata/docs", true)
	if err == nil {
		t.Fatalf("a missing picture was accepted")
	}
	if !strings.Contains(err.Error(), "no-such-file.png") {
		t.Errorf("the refusal does not name the file: %v", err)
	}
}

// TestACodeBlockWarnsAndIsLeftOut: code blocks stay out of the house style,
// and a note that carries one says so on the envelope.
func TestACodeBlockWarnsAndIsLeftOut(t *testing.T) {
	out := walk(t, "Words.\n\n```go\nfmt.Println(1)\n```\n\nMore words.\n")
	if len(out.Warnings) != 1 {
		t.Fatalf("the result carries %d warnings, want 1: %v", len(out.Warnings), out.Warnings)
	}
	if !strings.Contains(out.Warnings[0], "line 3") {
		t.Errorf("the warning does not name line 3: %s", out.Warnings[0])
	}
	got := serialise(t, out.Blocks)
	if strings.Contains(got, "fmt.Println") {
		t.Errorf("the code block reached the document:\n%s", got)
	}
	if out.Counts.Paragraphs != 2 {
		t.Errorf("counted %d paragraphs, want the two around the block", out.Counts.Paragraphs)
	}
}

// TestSmartQuotesAreTheCharacters: goldmark's typographer writes HTML entities
// by default, and those land in a w:t as the literal text "&ldquo;".
func TestSmartQuotesAreTheCharacters(t *testing.T) {
	out := walk(t, "He said \"hello\" -- and left...\n")
	got := serialise(t, out.Blocks)
	for _, want := range []string{"\u201c", "\u201d", "\u2013", "\u2026"} {
		if !strings.Contains(got, want) {
			t.Errorf("the text carries no %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "&ldquo;") || strings.Contains(got, "ldquo") {
		t.Errorf("an HTML entity reached the document:\n%s", got)
	}
}

// TestAnAmpersandIsEscapedOnce is what etree is here for.
func TestAnAmpersandIsEscapedOnce(t *testing.T) {
	out := walk(t, "Risk & Control, AT&T, and the entity &amp; written out.\n")
	got := serialise(t, out.Blocks)
	if !strings.Contains(got, "Risk &amp; Control, AT&amp;T, and the entity &amp; written out.") {
		t.Errorf("the ampersand is not escaped once:\n%s", got)
	}
	if strings.Contains(got, "&amp;amp;") {
		t.Errorf("the ampersand was escaped twice:\n%s", got)
	}
}

// TestInlineHTMLWarnsAndIsLeftOut: goldmark reads <angle brackets> as HTML, and
// the house style carries none. Dropping it in silence loses the author's own
// words, so the line is named.
func TestInlineHTMLWarnsAndIsLeftOut(t *testing.T) {
	out := walk(t, "Words.\n\nRisk & Control <ampersands> in a heading.\n")
	if len(out.Warnings) != 1 {
		t.Fatalf("the result carries %d warnings, want 1: %v", len(out.Warnings), out.Warnings)
	}
	if !strings.Contains(out.Warnings[0], "line 3") {
		t.Errorf("the warning does not name line 3: %s", out.Warnings[0])
	}
}

// TestInlineHTMLInATableCellOrAQuoteWarnsToo. warnRawHTML was called from the
// heading and the paragraph only, so a cell reading "one<br>two" published as
// "onetwo" with an empty warnings list. The same words matter more in a cell:
// goldmark reads <ampersands> as HTML, so an author's word disappears and
// nothing says so.
func TestInlineHTMLInATableCellOrAQuoteWarnsToo(t *testing.T) {
	cell := walk(t, "| Field | Value |\n| --- | --- |\n| A | one<br>two |\n")
	if len(cell.Warnings) != 1 {
		t.Fatalf("a table cell holding HTML carries %d warnings, want 1: %v",
			len(cell.Warnings), cell.Warnings)
	}
	if !strings.Contains(cell.Warnings[0], "line 3") {
		t.Errorf("the warning does not name line 3: %s", cell.Warnings[0])
	}

	quote := walk(t, "Words.\n\n> quoted <b>text</b>\n")
	if len(quote.Warnings) != 1 {
		t.Fatalf("a block quote holding HTML carries %d warnings, want 1: %v",
			len(quote.Warnings), quote.Warnings)
	}
	if !strings.Contains(quote.Warnings[0], "line 3") {
		t.Errorf("the warning does not name line 3: %s", quote.Warnings[0])
	}
}

// TestAPictureTheHouseStyleCannotPlaceIsNamed. A figure is a centred line of
// its own, which it cannot be inside a bullet, a quote or a table cell. The
// walker collected the images on those three paths and threw them away, so
// "- Text ![alt](one.png)" published as a bullet with no picture, no warning
// and an image count of nought.
func TestAPictureTheHouseStyleCannotPlaceIsNamed(t *testing.T) {
	for _, c := range []struct {
		what     string
		markdown string
	}{
		{"a list item", "- Text ![alt](one.png)\n"},
		{"a block quote", "> Text ![alt](one.png)\n"},
		{"a table cell", "| A | B |\n| --- | --- |\n| x | ![alt](one.png) |\n"},
	} {
		out := walk(t, c.what+"\n\n"+c.markdown)
		if len(out.Warnings) != 1 {
			t.Errorf("%s holding a picture carries %d warnings, want 1: %v",
				c.what, len(out.Warnings), out.Warnings)
			continue
		}
		if !strings.Contains(out.Warnings[0], c.what) {
			t.Errorf("the warning does not name %s: %s", c.what, out.Warnings[0])
		}
		if out.Counts.Images != 0 {
			t.Errorf("%s reports %d images, and none was placed", c.what, out.Counts.Images)
		}
	}
}

// TestALooseListItemIsOneItem. goldmark gives a loose item one paragraph per
// block, and every one of them used to take the list marker: a two-paragraph
// item read as two items, so the author's "2." printed as "3.".
func TestALooseListItemIsOneItem(t *testing.T) {
	out := walk(t, strings.Join([]string{
		"1. The provider is assessed annually.",
		"",
		"   The assessment covers financial standing.",
		"",
		"2. The register is reviewed quarterly.",
		"",
	}, "\n"))
	var marked, plain int
	for _, block := range out.Blocks {
		if find(block, "w:numPr") != nil {
			marked++
			continue
		}
		plain++
	}
	if marked != 2 {
		t.Errorf("%d paragraphs carry the list marker, want the two items the note wrote", marked)
	}
	if plain != 1 {
		t.Errorf("%d paragraphs carry no marker, want the one continuation paragraph", plain)
	}
}

// TestALooseContinuationKeepsTheIndentAndNotTheHang. The hanging indent is the
// marker's own column. Written on a paragraph with no marker to fill it, Word
// starts that paragraph's first line 18pt to the left of the item's own words.
func TestALooseContinuationKeepsTheIndentAndNotTheHang(t *testing.T) {
	out := walk(t, strings.Join([]string{
		"1. The provider is assessed annually.",
		"",
		"   The assessment covers financial standing.",
		"",
	}, "\n"))
	if len(out.Blocks) != 2 {
		t.Fatalf("%d blocks, want the item and its continuation", len(out.Blocks))
	}
	item, continuation := find(out.Blocks[0], "w:ind"), find(out.Blocks[1], "w:ind")
	// 36pt of indent, in twips, which is what house.yaml states for a numbered
	// list at the first level.
	if got := attr(t, item, "w:left"); got != "720" {
		t.Errorf("the item is indented %s twips, want 720", got)
	}
	if got := attr(t, continuation, "w:left"); got != "720" {
		t.Errorf("the continuation is indented %s twips, want the item's 720", got)
	}
	if got := attr(t, item, "w:hanging"); got != "360" {
		t.Errorf("the item hangs %s twips, want 360", got)
	}
	if a := continuation.SelectAttr("w:hanging"); a != nil {
		t.Errorf("the continuation hangs %s twips, want no hanging indent at all", a.Value)
	}
}

// TestAnItemTakesNoMoreThanOneMarkerWhateverItHolds. The marker goes on the first
// paragraph the item emits, and every case here is a way of getting that wrong
// in one direction or the other.
//
// A quote hands its own paragraphs back to block, so a quote carrying the
// item's marker numbered itself as an item and the author's "2." printed as
// "3.". Marking only the item's *ast.Paragraph children fixes that and breaks
// its mirror, an item that is nothing but a quote, which then has no
// *ast.Paragraph child and takes no number at all. Marking the item's first
// child instead breaks the third, an item opening with a fenced code block,
// which renders nothing and would swallow the marker.
//
// No more than one, rather than exactly one: an item holding nothing that can
// carry a marker takes none, and says so on the envelope. That is
// TestAnItemThatSpendsNoMarkerSaysSo.
func TestAnItemTakesNoMoreThanOneMarkerWhateverItHolds(t *testing.T) {
	cases := []struct {
		name     string
		markdown []string
		want     int
	}{
		{
			name: "a quote after the item's own words",
			markdown: []string{
				"1. The provider is assessed annually.",
				"",
				"   > Assessment covers financial standing.",
				"",
				"2. The register is reviewed quarterly.",
			},
			want: 2,
		},
		{
			name: "a quote as the whole of an item",
			markdown: []string{
				"1. The provider is assessed annually.",
				"",
				"2. > Assessment covers financial standing.",
				"",
				"3. The register is reviewed quarterly.",
			},
			want: 3,
		},
		{
			name: "a code block before the item's own words",
			markdown: []string{
				"1. The provider is assessed annually.",
				"",
				"2. ```",
				"   assess(provider)",
				"   ```",
				"",
				"   The register is reviewed quarterly.",
			},
			want: 2,
		},
		{
			name: "a continuation paragraph",
			markdown: []string{
				"1. The provider is assessed annually.",
				"",
				"   Assessment covers financial standing.",
				"",
				"2. The register is reviewed quarterly.",
			},
			want: 2,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := walk(t, strings.Join(append(c.markdown, ""), "\n"))
			var marked int
			for _, block := range out.Blocks {
				if find(block, "w:numPr") != nil {
					marked++
				}
			}
			if marked != c.want {
				t.Errorf("%d paragraphs carry the list marker, want the %d items the note wrote",
					marked, c.want)
			}
		})
	}
}

// TestANestedListTakesItsOwnMarkersAndLeavesTheOuterItemsAlone. The marker is
// the renderer's now, so a sub-list walked from inside an item has to put the
// outer item's back on the way out.
func TestANestedListTakesItsOwnMarkersAndLeavesTheOuterItemsAlone(t *testing.T) {
	out := walk(t, strings.Join([]string{
		"1. The provider is assessed annually.",
		"",
		"   - Financial standing.",
		"   - Operational history.",
		"",
		"2. The register is reviewed quarterly.",
		"",
	}, "\n"))
	var marked int
	for _, block := range out.Blocks {
		if find(block, "w:numPr") != nil {
			marked++
		}
	}
	if marked != 4 {
		t.Errorf("%d paragraphs carry a list marker, want the two items and the two nested ones", marked)
	}
}

// TestAnOuterItemWhoseWordsFollowItsSubListStillTakesItsMarker is the case the
// restore actually exists for, and the fixture above cannot reach it.
//
// There the outer item's paragraph runs first and spends the marker, so by the
// time the sub-list is walked there is nothing left to put back: deleting the
// defer keeps that test green. Here the sub-list comes first. Its two items arm
// and spend their own markers, and without the restore the outer item's own
// paragraph finds pendingMark false, takes no w:numPr, and Word then numbers
// the author's item 2 as "1.".
func TestAnOuterItemWhoseWordsFollowItsSubListStillTakesItsMarker(t *testing.T) {
	out := walk(t, strings.Join([]string{
		"1. - Financial standing.",
		"   - Operational history.",
		"",
		"   The provider is assessed annually.",
		"",
		"2. The register is reviewed quarterly.",
		"",
	}, "\n"))
	var marked int
	for _, block := range out.Blocks {
		if find(block, "w:numPr") != nil {
			marked++
		}
	}
	if marked != 4 {
		t.Errorf("%d paragraphs carry a list marker, want the two outer items and the two nested ones", marked)
	}
}

// TestAQuoteInsideAQuoteIsIndentedTwice. The depth used to be a parameter with
// one call site and one value, so "> >" was indented exactly as far as ">".
func TestAQuoteInsideAQuoteIsIndentedTwice(t *testing.T) {
	out := walk(t, "Words.\n\n> one\n\n> > two\n")
	var indents []string
	for _, block := range out.Blocks {
		if ind := find(block, "w:ind"); ind != nil {
			indents = append(indents, attr(t, ind, "w:left"))
		}
	}
	if len(indents) != 2 {
		t.Fatalf("%d quoted paragraphs carry an indent, want 2", len(indents))
	}
	if indents[0] != "720" {
		t.Errorf("one step of quoting indents %s twips, want 720 (36pt)", indents[0])
	}
	if indents[1] != "1440" {
		t.Errorf("two steps of quoting indent %s twips, want 1440 (72pt)", indents[1])
	}
}

// TestAQuotesPropertiesAreInSchemaOrder. w:ind used to be patched into a
// finished w:pPr by looking for w:jc, and a house file that states no body
// alignment has none: the fallback appended it after the paragraph mark's
// w:rPr, which is the last element the schema allows, and Word reads a w:pPr
// out of order as a repair.
func TestAQuotesPropertiesAreInSchemaOrder(t *testing.T) {
	cfg := config(t)
	cfg.Body.Align = ""
	out, err := Render(cfg, []byte("Words.\n\n> quoted\n"), "testdata/docs", true)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	var pPr *etree.Element
	for _, block := range out.Blocks {
		if find(block, "w:ind") != nil {
			pPr = find(block, "w:pPr")
		}
	}
	if pPr == nil {
		t.Fatal("no quoted paragraph carries an indent")
	}
	var tags []string
	for _, e := range pPr.ChildElements() {
		tags = append(tags, e.FullTag())
	}
	if len(tags) == 0 || tags[len(tags)-1] != "w:rPr" {
		t.Errorf("the quote's properties run %v, and w:rPr is last in the schema", tags)
	}
}

// TestATablesPropertiesAreInSchemaOrder is TestAQuotesPropertiesAreInSchemaOrder
// for a table. CT_TblPrBase is a sequence too, and tblCellMar used to be
// written before tblLayout, which is positions 14 then 13.
func TestATablesPropertiesAreInSchemaOrder(t *testing.T) {
	out := walk(t, "| Control | Owner |\n| --- | --- |\n| Screening | MLRO |\n")
	var tblPr *etree.Element
	for _, block := range out.Blocks {
		if block.FullTag() == "w:tbl" {
			tblPr = find(block, "w:tblPr")
		}
	}
	if tblPr == nil {
		t.Fatal("the note rendered no table")
	}
	var got []string
	for _, e := range tblPr.ChildElements() {
		got = append(got, e.FullTag())
	}
	want := []string{
		"w:tblStyle", "w:tblW", "w:jc", "w:tblInd",
		"w:tblBorders", "w:tblLayout", "w:tblCellMar", "w:tblLook",
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("the table's properties run %v, want %v", got, want)
	}
}

// TestAHeadingCarriesTheHouseStyleAndItsColour reads the house literals off a
// rendered heading.
func TestAHeadingCarriesTheHouseStyleAndItsColour(t *testing.T) {
	out := walk(t, "# Introduction\n\n## Scope\n\nWords.\n")
	first, second := out.Blocks[0], out.Blocks[1]
	if got := attr(t, find(first, "w:pStyle"), "w:val"); got != "Heading1" {
		t.Errorf("the first heading is styled %q, want Heading1", got)
	}
	if got := attr(t, find(second, "w:pStyle"), "w:val"); got != "Heading2" {
		t.Errorf("the second heading is styled %q, want Heading2", got)
	}
	if got := attr(t, find(first, "w:color"), "w:val"); got != "22265f" {
		t.Errorf("the heading colour is %q, want the house navy 22265f", got)
	}
	// The body starts on a clean page after the contents list, and only the
	// first heading carries the break.
	if find(first, "w:pageBreakBefore") == nil {
		t.Errorf("the first heading has no page break before it")
	}
	if find(second, "w:pageBreakBefore") != nil {
		t.Errorf("a later heading carries a page break")
	}
}

// TestABodyParagraphIsJustifiedAtTheHouseSize states the body's own literals.
func TestABodyParagraphIsJustifiedAtTheHouseSize(t *testing.T) {
	out := walk(t, "One ordinary paragraph.\n")
	p := out.Blocks[0]
	if got := attr(t, find(p, "w:jc"), "w:val"); got != "both" {
		t.Errorf("the paragraph is aligned %q, want both", got)
	}
	// 12pt body text is 24 half-points.
	if got := attr(t, find(p, "w:sz"), "w:val"); got != "24" {
		t.Errorf("the body size is %q half-points, want 24", got)
	}
	if out.Counts.Paragraphs != 1 {
		t.Errorf("counted %d paragraphs, want 1", out.Counts.Paragraphs)
	}
}

// TestAHeadingOfNothingButAPictureIsAFigure is Drive's "# ![][image1]".
func TestAHeadingOfNothingButAPictureIsAFigure(t *testing.T) {
	out := walk(t, "# **![](badge.png)**\n\n# Introduction\n")
	got := serialise(t, out.Blocks)
	if strings.Contains(got, "Heading1\"/>\n") && strings.Contains(got, "2-Introduction") {
		t.Errorf("the figure took a heading number:\n%s", got)
	}
	if !strings.Contains(got, ">1-Introduction<") {
		t.Errorf("the real heading is not numbered 1:\n%s", got)
	}
	if out.Counts.Images != 1 {
		t.Errorf("counted %d images, want the figure's one", out.Counts.Images)
	}
}

// TestNothingHereReachesTheNetwork is the render path's own rule, stated as a
// test rather than left to review.
func TestNothingHereReachesTheNetwork(t *testing.T) {
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, banned := range []string{`"net/http"`, `"os/exec"`, "internal/gapi"} {
			if bytes.Contains(src, []byte(banned)) {
				t.Errorf("%s imports %s, and a build reaches no network", name, banned)
			}
		}
	}
}

// TestAnItemThatSpendsNoMarkerSaysSo. An item whose children never emit a list
// paragraph takes no marker. In a numbered list Word numbers what is left, so
// the author's "3." prints as "2."; in a bulleted list nothing counts, so the
// item loses its own bullet and indent and nothing after it moves. A table is
// the case that reaches this in silence, because the table itself renders and
// nothing else warns.
//
// The warning is the walker's own rule, the one a picture in a list item and a
// code block already follow: what did not reach the document in the shape the
// author wrote is named on the envelope, never left for somebody to find by
// reading the published policy.
//
// The line is asserted as well as the words. A ListItem carries no source
// position, so an item holding only a code block resolved through nothing and
// named line 1, which in a note is the front matter's own delimiter: an author
// sent to the top of the file for a list further down is an author who cannot
// act on the warning.
func TestAnItemThatSpendsNoMarkerSaysSo(t *testing.T) {
	cases := []struct {
		name     string
		markdown []string
		want     string
		line     int
	}{
		{
			name: "an item that is only a table",
			markdown: []string{
				"1. Collect the provider's financial standing.",
				"",
				"2. | Control | Owner |",
				"   | --- | --- |",
				"   | Screening | MLRO |",
				"",
				"3. Record the outcome in the register.",
			},
			want: "takes no number",
			line: 3,
		},
		{
			name: "an item that is only a code block",
			markdown: []string{
				"1. The provider is assessed annually.",
				"",
				"2. ```",
				"   assess(provider)",
				"   ```",
				"",
				"3. The register is reviewed quarterly.",
			},
			want: "takes no number",
			line: 3,
		},
		{
			// A bullet is not a count, so nothing after this item moves and
			// the sentence may not say the numbering slipped.
			name: "a bulleted item that is only a table",
			markdown: []string{
				"- Collect the provider's financial standing.",
				"",
				"- | Control | Owner |",
				"  | --- | --- |",
				"  | Screening | MLRO |",
				"",
				"- Record the outcome in the register.",
			},
			want: "takes no bullet and no indent",
			line: 3,
		},
		{
			// The item spends its marker on its own words, so the table costs
			// the list nothing and there is nothing to say about numbering.
			name: "a table after the item's own words",
			markdown: []string{
				"1. Collect the provider's financial standing.",
				"",
				"   | Control | Owner |",
				"   | --- | --- |",
				"   | Screening | MLRO |",
				"",
				"2. Record the outcome in the register.",
			},
			want: "",
		},
		{
			name: "an ordinary list, which says nothing",
			markdown: []string{
				"1. The provider is assessed annually.",
				"",
				"2. The register is reviewed quarterly.",
			},
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := walk(t, strings.Join(append(c.markdown, ""), "\n"))
			var got string
			for _, warning := range out.Warnings {
				if strings.Contains(warning, "has no text of its own") {
					got = warning
				}
			}
			if c.want == "" {
				if got != "" {
					t.Fatalf("a list that costs nothing warned: %q", got)
				}
				return
			}
			if !strings.Contains(got, c.want) {
				t.Fatalf("the warning is %q, want one containing %q, warnings %q",
					got, c.want, out.Warnings)
			}
			if prefix := fmt.Sprintf("line %d:", c.line); !strings.HasPrefix(got, prefix) {
				t.Errorf("the warning is %q, want it to open with %q", got, prefix)
			}
		})
	}
}

// A footnote warns naming the author's own line and reaches no paragraph.
//
// The path is not hypothetical: v2's own `read` writes a document's footnotes
// as "[^1]" in the prose and "[^1]: the text" after a rule, so a note pulled
// out of a Google Doc and built back into one carried the markers as published
// prose. The extension is what turns that into a warning.
func TestAFootnoteWarnsAndIsLeftOut(t *testing.T) {
	out := walk(t, "A claim.[^1]\n\nMore words.\n\n[^1]: The supporting detail.\n")

	if len(out.Warnings) != 1 {
		t.Fatalf("the result carries %d warnings, want 1: %v", len(out.Warnings), out.Warnings)
	}
	if !strings.Contains(out.Warnings[0], "line 5") {
		t.Errorf("the warning does not name line 5, where the note is: %s", out.Warnings[0])
	}
	if !strings.Contains(out.Warnings[0], "footnote") {
		t.Errorf("the warning does not say what was left out: %s", out.Warnings[0])
	}
	got := serialise(t, out.Blocks)
	for _, unwanted := range []string{"supporting detail", "[^1]"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("%q reached the document:\n%s", unwanted, got)
		}
	}
	if out.Counts.Paragraphs != 2 {
		t.Errorf("counted %d paragraphs, want the two the author wrote", out.Counts.Paragraphs)
	}
}

// Every footnote is named, and by its own line rather than the list's: goldmark
// collects the definitions into one list at the end of the document, so the
// list's own position says nothing about where the author wrote them.
func TestEveryFootnoteIsNamedByItsOwnLine(t *testing.T) {
	out := walk(t, "A claim.[^a] Another.[^b]\n\n[^a]: The first detail.\n\n[^b]: The second.\n")

	if len(out.Warnings) != 2 {
		t.Fatalf("the result carries %d warnings, want 2: %v", len(out.Warnings), out.Warnings)
	}
	lines := strings.Join(out.Warnings, "\n")
	for _, want := range []string{"line 3", "line 5"} {
		if !strings.Contains(lines, want) {
			t.Errorf("no warning names %s:\n%s", want, lines)
		}
	}
}

// An empty fence names the line the author sees, and never line 0. The fenced
// arm steps one line back from the block's first line of code, and an empty
// fence has no lines at all.
func TestAnEmptyFenceNamesARealLine(t *testing.T) {
	out := walk(t, "Words.\n\n```\n```\n")

	if len(out.Warnings) != 1 {
		t.Fatalf("the result carries %d warnings, want 1: %v", len(out.Warnings), out.Warnings)
	}
	if strings.Contains(out.Warnings[0], "line 0") {
		t.Errorf("the warning names line 0, which is no line in any file: %s", out.Warnings[0])
	}
}

// A numbered list that opens on a number the author did not write says so on
// the envelope.
//
// Every level of the numbered abstract list states w:start 1 and every list's
// own w:num states w:startOverride 1 over it, so an author's "5." opens at 1
// whatever depth it sits at. Honouring it is that same override carrying the
// author's number, which gdoc does not write. The silence is not deferred, because
// the prose around a list cross-references the numbers the author wrote.
//
// A second numbered list is no longer one of these shapes: it opens its own
// w:num and starts again at 1. See TestASecondNumberedListStartsAgain.
func TestANumberedListWhoseNumbersAreNotTheAuthorsSaysSo(t *testing.T) {
	cases := []struct {
		name     string
		markdown []string
		want     string
		line     int
	}{
		{
			name: "a numbered list that starts at five",
			markdown: []string{
				"5. five",
				"6. six",
			},
			want: "starts at 5 in the note and at 1 in the document",
			line: 1,
		},
		{
			name: "one numbered list, which says nothing",
			markdown: []string{
				"1. one",
				"2. two",
			},
			want: "",
		},
		{
			// A bullet is not a count, so two bulleted lists cost nothing.
			name: "two bulleted lists",
			markdown: []string{
				"- one",
				"",
				"Prose between.",
				"",
				"- alpha",
			},
			want: "",
		},
		{
			// An absent w:lvlRestart restarts a level whenever the level
			// above it moves, so a nested list is not in the count.
			name: "a nested numbered list under each of two items",
			markdown: []string{
				"1. one",
				"   1. inner",
				"2. two",
				"   1. inner",
			},
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := walk(t, strings.Join(append(c.markdown, ""), "\n"))
			var got string
			for _, warning := range out.Warnings {
				if strings.Contains(warning, "numbered list") {
					got = warning
				}
			}
			if c.want == "" {
				if got != "" {
					t.Fatalf("a list whose numbers are the author's warned: %q", got)
				}
				return
			}
			if !strings.Contains(got, c.want) {
				t.Fatalf("the warning is %q, want one containing %q, warnings %q",
					got, c.want, out.Warnings)
			}
			if prefix := fmt.Sprintf("line %d:", c.line); !strings.HasPrefix(got, prefix) {
				t.Errorf("the warning is %q, want it to open with %q", got, prefix)
			}
		})
	}
}

// A heading that skips a level is numbered with a zero in it, and says so.
//
// The number is built from every counter down to the heading's own level, so a
// "###" under a "#" reads "1.0.1-". The number itself is the house format and
// is left as it is, so what the note gets here is the line to look at.
func TestASkippedHeadingLevelSaysSo(t *testing.T) {
	out := walk(t, "# Alpha\n\n### Gamma\n")

	var got string
	for _, warning := range out.Warnings {
		if strings.Contains(warning, "skips a level") {
			got = warning
		}
	}
	if got == "" {
		t.Fatalf("a heading numbered 1.0.1- said nothing, warnings %q", out.Warnings)
	}
	if !strings.HasPrefix(got, "line 3:") {
		t.Errorf("the warning is %q, want it to name line 3", got)
	}
	if !strings.Contains(got, "1.0.1-") {
		t.Errorf("the warning is %q, want it to carry the number it wrote", got)
	}

	quiet := walk(t, "# Alpha\n\n## Beta\n\n### Gamma\n")
	for _, warning := range quiet.Warnings {
		if strings.Contains(warning, "skips a level") {
			t.Errorf("a document that skips nothing warned: %q", warning)
		}
	}
}

// TestAHeadingInsideAFootnoteSetsNoHeadingDepth: a footnote definition is
// dropped whole, so a heading written inside one is not a heading of this
// document and must not set the depth every other heading numbers from.
// Otherwise a stray "# " in a footnote numbers the real headings "0.1-".
func TestAHeadingInsideAFootnoteSetsNoHeadingDepth(t *testing.T) {
	out := walk(t, "## Real heading\n\nBody.[^1]\n\n[^1]: the detail\n\n    # Hidden heading\n")

	body := serialise(t, out.Blocks)
	if !strings.Contains(body, "1-Real heading") || strings.Contains(body, "0.1-Real heading") {
		t.Errorf("the only heading is not numbered 1, so a heading inside a footnote set the depth:\n%s", body)
	}
}
