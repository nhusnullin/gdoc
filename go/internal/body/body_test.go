package body

import (
	"bytes"
	"flag"
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

// TestAPipeTableCarriesTheHouseRecipe states v1's table recipe as literals.
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
	// 493.18pt of usable width is 6263386 EMU: a picture wider than the column
	// is scaled to exactly that.
	if !strings.Contains(got, `cx="6263386"`) {
		t.Errorf("the picture was not sized to the text column:\n%s", got)
	}
}

// TestASmallImageKeepsItsOwnSize is the other half of that rule.
func TestASmallImageKeepsItsOwnSize(t *testing.T) {
	out := walk(t, "![A badge](badge.png)\n")
	got := serialise(t, out.Blocks)
	if strings.Contains(got, `cx="6263386"`) {
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
