package house

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gdoc/internal/asset"
)

// Every value in this file is written as a literal on purpose. A test that
// reads the constant it checks is a mirror: change the config and the
// assertion follows it. See CLAUDE.md, "A house-style test must never read the
// constant it tests".

func TestLoadReadsTheEmbeddedFile(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Meta.Name != "altery-group-policy" {
		t.Errorf("meta.name = %q, want altery-group-policy", cfg.Meta.Name)
	}
	if cfg.Meta.Version != "1.0" {
		t.Errorf("meta.version = %q, want 1.0", cfg.Meta.Version)
	}
}

func TestPageIsA4InPoints(t *testing.T) {
	cfg := load(t)
	if cfg.Page.WidthPt != 595.2755905511811 {
		t.Errorf("page.width_pt = %v, want A4's 595.2755905511811", cfg.Page.WidthPt)
	}
	if cfg.Page.HeightPt != 841.8897637795275 {
		t.Errorf("page.height_pt = %v, want A4's 841.8897637795275", cfg.Page.HeightPt)
	}
	if cfg.Page.MarginTopPt != 62.35 {
		t.Errorf("page.margin_top_pt = %v, want 62.35", cfg.Page.MarginTopPt)
	}
	if cfg.Page.MarginLeftPt != 51.05 {
		t.Errorf("page.margin_left_pt = %v, want 51.05", cfg.Page.MarginLeftPt)
	}
	if !cfg.Page.DifferentFirstPage {
		t.Error("page.different_first_page = false, want true")
	}
	if cfg.Page.PageNumberStart != 1 {
		t.Errorf("page.page_number_start = %d, want 1", cfg.Page.PageNumberStart)
	}
}

func TestHeading1IsTheHouseHeading(t *testing.T) {
	cfg := load(t)
	s, ok := cfg.Style("heading_1")
	if !ok {
		t.Fatal("styles.heading_1 is missing")
	}
	if s.SizePt == nil || *s.SizePt != 16 {
		t.Errorf("heading_1.size_pt = %v, want 16", s.SizePt)
	}
	if !s.Bold {
		t.Error("heading_1.bold = false, want true")
	}
	if s.Color != "#22265F" {
		t.Errorf("heading_1.color = %q, want #22265F", s.Color)
	}
	if s.SpaceBeforePt == nil || *s.SpaceBeforePt != 12 {
		t.Errorf("heading_1.space_before_pt = %v, want 12", s.SpaceBeforePt)
	}
	if s.SpaceAfterPt == nil || *s.SpaceAfterPt != 6 {
		t.Errorf("heading_1.space_after_pt = %v, want 6", s.SpaceAfterPt)
	}
	if !s.KeepWithNext {
		t.Error("heading_1.keep_with_next = false, want true")
	}
}

func TestHeading2HasSixPointsAfter(t *testing.T) {
	cfg := load(t)
	s, ok := cfg.Style("heading_2")
	if !ok {
		t.Fatal("styles.heading_2 is missing")
	}
	if s.SpaceAfterPt == nil || *s.SpaceAfterPt != 6 {
		t.Errorf("heading_2.space_after_pt = %v, want 6", s.SpaceAfterPt)
	}
}

func TestTheTenStylesAreThere(t *testing.T) {
	cfg := load(t)
	want := []string{
		"normal", "heading_1", "heading_2", "heading_3", "heading_4",
		"heading_5", "heading_6", "title", "subtitle", "footnote_text",
	}
	if len(cfg.Styles) != 10 {
		t.Errorf("styles has %d entries, want 10", len(cfg.Styles))
	}
	for _, name := range want {
		if _, ok := cfg.Style(name); !ok {
			t.Errorf("styles.%s is missing", name)
		}
	}
}

// TestFootnoteTextIsTenPointSingleSpacedAndTight. The tenth style, added with
// the footnotes. A note under the page is smaller than the prose it hangs off,
// sits in the body face, and carries no space above or below, because the
// notes are a block of lines rather than a run of paragraphs.
func TestFootnoteTextIsTenPointSingleSpacedAndTight(t *testing.T) {
	cfg := load(t)
	s, ok := cfg.Style("footnote_text")
	if !ok {
		t.Fatal("styles.footnote_text is missing")
	}
	if s.SizePt == nil || *s.SizePt != 10 {
		t.Errorf("footnote_text.size_pt = %v, want 10", s.SizePt)
	}
	if s.LineSpacing == nil || *s.LineSpacing != 1.0 {
		t.Errorf("footnote_text.line_spacing = %v, want 1.0", s.LineSpacing)
	}
	if s.SpaceBeforePt == nil || *s.SpaceBeforePt != 0 {
		t.Errorf("footnote_text.space_before_pt = %v, want 0", s.SpaceBeforePt)
	}
	if s.SpaceAfterPt == nil || *s.SpaceAfterPt != 0 {
		t.Errorf("footnote_text.space_after_pt = %v, want 0", s.SpaceAfterPt)
	}
	if s.Font != "Calibri" {
		t.Errorf("footnote_text.font = %q, want Calibri", s.Font)
	}
	// Subtitle, the style above this one in the file, keeps with the next
	// paragraph and keeps its own lines together. A note under the page does
	// neither: Word places the notes, and keepNext on one of them is a rule
	// about a page this style has no say over. Asked because the first draft
	// of this entry landed between subtitle's values and those two keys, and
	// both styles came out wrong with nothing else to name it.
	if s.KeepWithNext {
		t.Error("footnote_text.keep_with_next is true, want false")
	}
	if s.KeepLinesTogether {
		t.Error("footnote_text.keep_lines_together is true, want false")
	}
}

func TestHeadingNumberingIsTheLiteralFormat(t *testing.T) {
	cfg := load(t)
	if cfg.HeadingNumbering.Mechanism != "literal" {
		t.Errorf("heading_numbering.mechanism = %q, want literal", cfg.HeadingNumbering.Mechanism)
	}
	if cfg.HeadingNumbering.Level1Format != "{n}-{title}" {
		t.Errorf("heading_numbering.level1_format = %q, want {n}-{title}", cfg.HeadingNumbering.Level1Format)
	}
}

func TestTOCIsALiveFieldOverThreeHeadingLevels(t *testing.T) {
	cfg := load(t)
	if !cfg.TOC.Live {
		t.Error("toc.live = false, want true")
	}
	if !strings.Contains(cfg.TOC.Instr, "Heading 1,1,Heading 2,2,Heading 3,3") {
		t.Errorf("toc.instr = %q, want it to carry Heading 1,1,Heading 2,2,Heading 3,3", cfg.TOC.Instr)
	}
	if cfg.TOC.SizePt != 11 {
		t.Errorf("toc.size_pt = %v, want 11", cfg.TOC.SizePt)
	}
}

func TestFrontMatterIsFourteenBlocksInOrder(t *testing.T) {
	cfg := load(t)
	type want struct {
		kind string
		ref  string
	}
	order := []want{
		{"cover", ""},
		{"page_break", ""},
		{"label", "version_control"},
		{"table", "version_control"},
		{"blank", ""},
		{"table", "revision_history"},
		{"blank", ""},
		{"legend", ""},
		{"blank", ""},
		{"label", "document_classification"},
		{"table", "document_classification"},
		{"page_break", ""},
		{"label", "contents"},
		{"toc", ""},
	}
	if len(cfg.FrontMatter) != 14 {
		t.Fatalf("front_matter has %d blocks, want 14", len(cfg.FrontMatter))
	}
	for i, w := range order {
		got := cfg.FrontMatter[i]
		if got.Block != w.kind || got.Ref != w.ref {
			t.Errorf("front_matter[%d] = {%s %s}, want {%s %s}", i, got.Block, got.Ref, w.kind, w.ref)
		}
	}
}

func TestTheBulletGlyphsAndTheNumberedFormat(t *testing.T) {
	cfg := load(t)
	glyphs := cfg.Body.Bullet.Glyphs
	if len(glyphs) != 3 || glyphs[0] != "●" || glyphs[1] != "○" || glyphs[2] != "■" {
		t.Errorf("body.bullet.glyphs = %q, want [● ○ ■]", glyphs)
	}
	if cfg.Body.Numbered.Format != "%1." {
		t.Errorf("body.numbered.format = %q, want %%1.", cfg.Body.Numbered.Format)
	}
	if cfg.Body.Bullet.IndentStartPt != 36 {
		t.Errorf("body.bullet.indent_start_pt = %v, want 36", cfg.Body.Bullet.IndentStartPt)
	}
	if cfg.Body.Bullet.HangingPt != 18 {
		t.Errorf("body.bullet.hanging_pt = %v, want 18", cfg.Body.Bullet.HangingPt)
	}
}

func TestTheThreeTablesAreThereWithTheirColumns(t *testing.T) {
	cfg := load(t)
	for _, name := range []string{"version_control", "revision_history", "document_classification"} {
		if _, ok := cfg.Tables[name]; !ok {
			t.Errorf("tables.%s is missing", name)
		}
	}
	vc := cfg.Tables["version_control"]
	if len(vc.ColumnsPt) != 2 || vc.ColumnsPt[0] != 150.0 || vc.ColumnsPt[1] != 359.25 {
		t.Errorf("tables.version_control.columns_pt = %v, want [150 359.25]", vc.ColumnsPt)
	}
	if vc.Border.WidthPt != 1.0 {
		t.Errorf("tables.version_control.border.width_pt = %v, want 1", vc.Border.WidthPt)
	}
	if len(vc.Rows) == 0 || len(vc.Rows[0].Cells) == 0 {
		t.Fatal("tables.version_control has no first cell")
	}
	if fill := vc.Rows[0].Cells[0].Fill; fill != "#F5D1AE" {
		t.Errorf("first cell fill = %q, want #F5D1AE", fill)
	}
}

// TestTheLogoStatesItsMimeWidthAndPlacement reads the copy under testdata
// rather than the embedded file, because the firm's logo is one file in Drive
// and no test in this tree carries its bytes. The copy is the embedded file
// with the placeholder's base64 in place of them, so what is under test is
// still the house's own statement of the mark and the decode of it.
func TestTheLogoStatesItsMimeWidthAndPlacement(t *testing.T) {
	cfg := loadBase64House(t)
	if cfg.Logo.Mime != "image/png" {
		t.Errorf("logo.mime = %q, want image/png", cfg.Logo.Mime)
	}
	if cfg.Logo.WidthPt != 111.75 {
		t.Errorf("logo.width_pt = %v, want 111.75", cfg.Logo.WidthPt)
	}
	raw, err := cfg.LogoBytes()
	if err != nil {
		t.Fatalf("LogoBytes: %v", err)
	}
	img, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("the logo is not a PNG: %v", err)
	}
	if img.Width != 509 || img.Height != 185 {
		t.Errorf("logo is %dx%d px, want 509x185", img.Width, img.Height)
	}
	if cfg.Logo.Anchor.OffsetXPt != 381.43 || cfg.Logo.Anchor.OffsetYPt != -11.15 {
		t.Errorf("logo anchor offsets = %v, %v, want 381.43, -11.15",
			cfg.Logo.Anchor.OffsetXPt, cfg.Logo.Anchor.OffsetYPt)
	}
}

// TestTheMarkEndsInLineWithTheText reads the mark's width and its left offset
// out of the house that ships, and states every other number as a literal.
// Nail's decision of 2026-10-29: the mark ends in line with the text instead
// of reaching past it into the right margin, which read as a slip on the first
// page of every document. Two things stay true of that pair. The mark's right
// side is the column's right edge, and the mark is wholly on the paper with
// the body's own margin to its right.
func TestTheMarkEndsInLineWithTheText(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	width, offset := cfg.Logo.WidthPt, cfg.Logo.Anchor.OffsetXPt

	// The offset, then the mark: 381.43 + 111.75 is 493.18 points, the
	// column's own width, so the mark ends where the text ends.
	right := offset + width
	if math.Abs(right-493.18) > 0.005 {
		t.Errorf("the mark's right side is %v points from the column's left edge, want the column's 493.18", right)
	}

	// The left margin, then the offset, then the mark: 51.05 + 493.18 is
	// 544.23 points, which leaves the body's own 51.05 pt margin to the
	// mark's right, inside A4's 595.28.
	onPaper := 51.05 + right
	if math.Abs(onPaper-544.23) > 0.005 {
		t.Errorf("the mark's right side is %v points from the paper's left edge, want 544.23", onPaper)
	}
	if onPaper >= 595.2755905511811 {
		t.Errorf("the mark reaches %v points, past A4's 595.2755905511811, so the paper's edge cuts it", onPaper)
	}
}

// TestTheLabelDropsSixPointsAndTheMarkStays is the other half of Nail's
// decision of 2026-10-29: "For internal use only" sits level with the mark's
// middle instead of six points above it. The label is a run in the first
// paragraph of the first-page header, and the mark is anchored from that same
// paragraph, so the paragraph gains six points before it and the mark's own
// offset gives the same six back. The sum is the offset the mark had from the
// header margin before this run, which is the sentence "the label moves and
// the mark does not" written as a number.
func TestTheLabelDropsSixPointsAndTheMarkStays(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Header.First.Paragraphs) == 0 {
		t.Fatal("the first page header carries no paragraphs")
	}
	stated := cfg.Header.First.Paragraphs[0].SpaceBeforePt
	if stated == nil {
		t.Fatal("the label's paragraph states no space_before_pt, and it states six points")
	}
	before := *stated
	if before != 6 {
		t.Errorf("the label's paragraph carries space_before_pt %v, want 6", before)
	}
	top := cfg.Logo.Anchor.OffsetYPt
	if top != -11.15 {
		t.Errorf("the logo anchor's offset_y_pt is %v, want -11.15", top)
	}
	if math.Abs(before+top-(-5.15)) > 0.005 {
		t.Errorf("the label's six points and the mark's offset sum to %v, want -5.15, the mark's offset from the header margin before this run", before+top)
	}
}

func TestLoadFileReadsACopyOfTheEmbeddedFile(t *testing.T) {
	path := write(t, embedded)
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if cfg.Page.WidthPt != 595.2755905511811 {
		t.Errorf("page.width_pt = %v, want A4's 595.2755905511811", cfg.Page.WidthPt)
	}
	if len(cfg.FrontMatter) != 14 {
		t.Errorf("front_matter has %d blocks, want 14", len(cfg.FrontMatter))
	}
}

func TestLoadFileOnAMissingPathNamesThePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-here.yaml")
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile on a missing path returned no error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the path %q", err, path)
	}
}

func TestAnUnknownKeyIsRefusedNamingIt(t *testing.T) {
	path := write(t, append(append([]byte{}, embedded...), []byte("\nnonsense_key: 1\n")...))
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("an unknown key was accepted")
	}
	if !strings.Contains(err.Error(), "nonsense_key") {
		t.Errorf("error %q does not name nonsense_key", err)
	}
}

func TestABlockNamingATableThatIsNotThereIsRefused(t *testing.T) {
	src := bytes.Replace(embedded,
		[]byte("- block: table\n  ref: revision_history\n"),
		[]byte("- block: table\n  ref: no_such_table\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the front_matter table reference was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a front_matter block naming a missing table was accepted")
	}
	if !strings.Contains(err.Error(), "no_such_table") {
		t.Errorf("error %q does not name no_such_table", err)
	}
}

func TestABlockNamingALabelThatIsNotThereIsRefused(t *testing.T) {
	src := bytes.Replace(embedded,
		[]byte("- block: label\n  ref: contents\n"),
		[]byte("- block: label\n  ref: no_such_label\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the front_matter label reference was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a front_matter block naming a missing label was accepted")
	}
	if !strings.Contains(err.Error(), "no_such_label") {
		t.Errorf("error %q does not name no_such_label", err)
	}
}

func TestAnUnknownBlockKindIsRefusedNamingIt(t *testing.T) {
	src := bytes.Replace(embedded, []byte("- block: legend\n"), []byte("- block: sidebar\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the legend block was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("an unknown front_matter block kind was accepted")
	}
	if !strings.Contains(err.Error(), "sidebar") {
		t.Errorf("error %q does not name sidebar", err)
	}
}

// TestAPageBreakCarryingAnotherKeyIsRefusedNamingIt. A page break is the whole
// block: it has no ref, no count and no size of its own. A key written beside
// it is a value the writers would drop in silence, and a break that is a
// paragraph tall in one route and something else in the other is the two
// routes drifting apart, so it is refused at the door.
func TestAPageBreakCarryingAnotherKeyIsRefusedNamingIt(t *testing.T) {
	src := bytes.Replace(embedded,
		[]byte("- block: page_break\n- block: label\n  ref: contents\n"),
		[]byte("- block: page_break\n  ref: contents\n- block: label\n  ref: contents\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the page_break before the contents label was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a page_break block carrying a ref was accepted")
	}
	if !strings.Contains(err.Error(), "page_break") || !strings.Contains(err.Error(), "ref") {
		t.Errorf("error %q does not name the page_break block and the ref it carries", err)
	}
}

// TestACountedPageBreakIsRefused is the same rule over the key somebody would
// reach for first: three breaks written as one block are three pages nobody
// asked for.
func TestACountedPageBreakIsRefused(t *testing.T) {
	src := bytes.Replace(embedded,
		[]byte("- block: cover\n- block: page_break\n"),
		[]byte("- block: cover\n- block: page_break\n  count: 3\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the page_break after the cover was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a page_break block carrying a count was accepted")
	}
	if !strings.Contains(err.Error(), "page_break") || !strings.Contains(err.Error(), "count") {
		t.Errorf("error %q does not name the page_break block and the count it carries", err)
	}
}

// TestTheCoverEndsWithNoTrailingBlanks. The eight blanks under the date were
// how the master pushed the version control label onto page two. The break
// does the pushing now, and eight empty paragraphs left behind are eight lines
// a person deletes by hand the first time they edit the cover.
func TestTheCoverEndsWithNoTrailingBlanks(t *testing.T) {
	cfg := load(t)
	if got := len(cfg.Cover.TrailingBlanks); got != 0 {
		t.Errorf("the cover carries %d trailing blanks, want 0", got)
	}
	if got := len(cfg.Cover.LeadingBlanks); got != 5 {
		t.Errorf("the cover opens with %d blanks, want 5", got)
	}
}

func TestAMissingStyleIsRefusedNamingIt(t *testing.T) {
	src := bytes.Replace(embedded, []byte("  heading_6:\n"), []byte("  heading_seven:\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("heading_6 was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a config missing heading_6 was accepted")
	}
	if !strings.Contains(err.Error(), "heading_6") {
		t.Errorf("error %q does not name heading_6", err)
	}
}

// TestAConfigMissingFootnoteTextIsRefusedNamingIt. The tenth style is
// required like the other nine: render reads it by name to set the notes
// under the page, so a config without it is a document it would build with
// the notes in the prose size.
func TestAConfigMissingFootnoteTextIsRefusedNamingIt(t *testing.T) {
	src := bytes.Replace(embedded, []byte("  footnote_text:\n"), []byte("  footnote_prose:\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("footnote_text was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a config missing footnote_text was accepted")
	}
	if !strings.Contains(err.Error(), "footnote_text") {
		t.Errorf("error %q does not name footnote_text", err)
	}
}

// TestAHighlightThatIsAColourIsRefused. w:highlight takes a name from OOXML's
// fixed list, never a hex value: a '#FFFF00' reached the part as
// w:val="#FFFF00", which Word repairs the document over. Every other colour in
// the file is hex, so writing one here is the natural mistake.
//
// Each arm of the walk is driven at its own anchor rather than at the first
// highlight in the file. The value that was actually written as a colour was a
// revision-history cell, and a case that replaces the first match only ever
// reaches the cover, so the table loop and the header loop could both be
// deleted and this test would stay green.
func TestAHighlightThatIsAColourIsRefused(t *testing.T) {
	cases := []struct {
		name   string
		find   string
		put    string
		where  string
		absent string
	}{
		{
			// cover.lines[N].runs[N], the cover's own run.
			name:  "a cover line's run",
			find:  "    placeholder: title\n    space_before_pt: 0\n    space_after_pt: 0\n    runs:\n    - text: (Name of)\n      highlight: yellow\n",
			put:   "    placeholder: title\n    space_before_pt: 0\n    space_after_pt: 0\n    runs:\n    - text: (Name of)\n      highlight: '#FFFF00'\n",
			where: "cover.lines[",
		},
		{
			// cover.lines[N], the line's own highlight rather than a run's.
			name:   "a cover line itself",
			find:   "  - text: May 2025\n    size_pt: 20\n    highlight: yellow\n",
			put:    "  - text: May 2025\n    size_pt: 20\n    highlight: '#FFFF00'\n",
			where:  "cover.lines[",
			absent: ".runs[",
		},
		{
			// header.default.paragraphs[N].runs[N]. The file states no
			// highlight in a header or a footer, so this one is added.
			name:  "a header run",
			find:  "      - text: Altery - xxx Policy\n        size_pt: 12\n",
			put:   "      - text: Altery - xxx Policy\n        size_pt: 12\n        highlight: '#FFFF00'\n",
			where: "header.default.paragraphs[0].runs[0]",
		},
		{
			// tables.revision_history, which is where the value that was
			// really written as a colour lived.
			name:  "a table cell's run",
			find:  "          - text: '[Relevant Board Level Committee]'\n            highlight: yellow\n",
			put:   "          - text: '[Relevant Board Level Committee]'\n            highlight: '#FFFF00'\n",
			where: "tables.revision_history",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if bytes.Count(embedded, []byte(c.find)) != 1 {
				t.Fatalf("the anchor %q is not in house.yaml exactly once", c.find)
			}
			src := bytes.Replace(embedded, []byte(c.find), []byte(c.put), 1)
			_, err := LoadFile(write(t, src))
			if err == nil {
				t.Fatal("a highlight written as a colour was accepted")
			}
			if !strings.Contains(err.Error(), "#FFFF00") {
				t.Errorf("error %q does not name the value it refused", err)
			}
			if !strings.Contains(err.Error(), c.where) {
				t.Errorf("error %q does not name %s, so another arm answered first", err, c.where)
			}
			if c.absent != "" && strings.Contains(err.Error(), c.absent) {
				t.Errorf("error %q names %s, so a run answered rather than the line itself", err, c.absent)
			}
		})
	}
}

func TestALogoThatIsNotAPNGIsRefused(t *testing.T) {
	bad := base64.StdEncoding.EncodeToString([]byte("this is not a png"))
	house := base64House(t)
	cut := bytes.Index(house, []byte("  base64: |-\n"))
	if cut < 0 {
		t.Fatal("the logo base64 block was not found to replace")
	}
	src := append(append([]byte{}, house[:cut]...), []byte("  base64: "+bad+"\n")...)
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a logo that is not a PNG was accepted")
	}
	if !strings.Contains(err.Error(), "logo") {
		t.Errorf("error %q does not name the logo", err)
	}
}

func TestABase64LogoObeysTheAssetBounds(t *testing.T) {
	pngBytes := func(width, height int) []byte {
		t.Helper()
		var buf bytes.Buffer
		if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
			t.Fatalf("encode %dx%d PNG: %v", width, height, err)
		}
		return buf.Bytes()
	}
	whole := pngBytes(1, 1)
	for _, c := range []struct {
		name string
		raw  []byte
		want string
	}{
		{name: "byte ceiling", raw: append(append([]byte{}, whole...), make([]byte, asset.MaxBytes)...), want: "may not exceed"},
		{name: "side ceiling", raw: pngBytes(asset.MaxSide+1, 1), want: "4097x1"},
		{name: "complete decode", raw: append([]byte{}, whole[:33]...), want: "not a PNG"},
	} {
		t.Run(c.name, func(t *testing.T) {
			encoded := base64.StdEncoding.EncodeToString(c.raw)
			src := houseWithLogo(t, logoBlock+"  base64: "+encoded+"\n")
			_, err := LoadFile(write(t, src))
			if err == nil {
				t.Fatal("an out-of-bounds base64 logo was accepted")
			}
			if !strings.Contains(err.Error(), "logo") || !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not name the logo and %q", err, c.want)
			}
		})
	}
}

// TestNothingHereReachesTheNetwork is the render path's own rule, stated as a
// test rather than left to review.
//
// Every production file in the package is read, not house.go alone. The
// package is one file today, which is exactly what would hide a second one:
// the boundary test bans net/http and os/exec across the whole tree, but an
// internal/gapi import names neither, so this is the only place that would see
// it. The banned strings are internal/body's, spelled the same way, so the two
// canaries state one rule.
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
				t.Errorf("%s imports %s, and the generator reaches no network", name, banned)
			}
		}
	}
}

// load is Load with the error already checked.
func load(t *testing.T) *Config {
	t.Helper()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

// write puts src in a temp file and gives back its path.
func write(t *testing.T, src []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "house.yaml")
	if err := os.WriteFile(path, src, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestUsableWidthIsTheSpaceBetweenTheSideMargins states the width a body table
// and a full-width image are sized to, as a literal. A4 is 595.28 points wide
// and the two side margins are 51.05 each, so what is left is 493.18.
func TestUsableWidthIsTheSpaceBetweenTheSideMargins(t *testing.T) {
	if got := load(t).Page.UsableWidthPt(); math.Abs(got-493.18) > 0.005 {
		t.Errorf("usable width = %v points, want 493.18", got)
	}
	// The arithmetic on its own, so a page whose margins differ still reads.
	p := Page{WidthPt: 100, MarginLeftPt: 10, MarginRightPt: 15}
	if got := p.UsableWidthPt(); got != 75 {
		t.Errorf("a 100pt page inside 10 and 15 leaves %v points, want 75", got)
	}
}

// houseWithLogo returns the embedded house file with its whole logo: block
// replaced by the one stated here. logo: is the last top-level key in the
// file, so the cut needs no end. Every variant below states its block in
// full, because the thing under test is which keys are there.
func houseWithLogo(t *testing.T, block string) []byte {
	t.Helper()
	cut := bytes.Index(embedded, []byte("\nlogo:\n"))
	if cut < 0 {
		t.Fatal("the logo block was not found to replace")
	}
	return append(append([]byte{}, embedded[:cut+1]...), []byte(block)...)
}

// logoBlock is a well-formed logo: block carrying every key but the two that
// name the bytes. The caller appends one of them, or both, or neither.
const logoBlock = `logo:
  mime: image/png
  width_pt: 146.25
  anchor:
    in: first_page_header
    relative_h: column
    offset_x_pt: 404.25
    relative_v: paragraph
    offset_y_pt: -5.15
    wrap: square
    wrap_dist_pt: 9
`

func TestALogoThatStatesPixelsIsRefusedNamingTheKey(t *testing.T) {
	for _, c := range []struct{ key, line string }{
		{"px", "  px: [195, 97]\n"},
		{"height_pt", "  height_pt: 72.75\n"},
	} {
		t.Run(c.key, func(t *testing.T) {
			block := strings.Replace(logoBlock, "  width_pt: 146.25\n", c.line+"  width_pt: 146.25\n", 1)
			_, err := LoadFile(write(t, houseWithLogo(t, block+"  drive: "+aDriveID+"\n")))
			if err == nil {
				t.Fatalf("a logo stating %s was accepted", c.key)
			}
			if !strings.Contains(err.Error(), c.key) {
				t.Errorf("error %q does not name %s", err, c.key)
			}
		})
	}
}

// aDriveID is a file id of the shape Drive mints, written out here so no test
// in this package names the firm's own file.
const aDriveID = "1AbCdEfGhIjKlMnOpQrStUvWxYz012345"

// pngOfHouseSize is a PNG of the size the house file states, so a refusal in
// these tests is never about the bytes.
func pngOfHouseSize(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 195, 97))); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// TestALogoNamesExactlyOneOfDriveAndBase64 holds the door the bytes come
// through: logo.drive, one file in Drive, or logo.base64, the bytes in the
// file. Both is a file with two answers and neither is a file with none, and
// each is refused naming the keys rather than failing later at the PNG.
func TestALogoNamesExactlyOneOfDriveAndBase64(t *testing.T) {
	for _, c := range []struct{ name, tail string }{
		{"both", "  drive: " + aDriveID + "\n  base64: " + pngOfHouseSize(t) + "\n"},
		{"neither", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := LoadFile(write(t, houseWithLogo(t, logoBlock+c.tail)))
			if err == nil {
				t.Fatalf("a logo naming %s of drive and base64 was accepted", c.name)
			}
			if !strings.Contains(err.Error(), "logo") {
				t.Errorf("error %q does not name the logo", err)
			}
			if !strings.Contains(err.Error(), "logo.drive") {
				t.Errorf("error %q does not name logo.drive, so it does not say which keys", err)
			}
			if !strings.Contains(err.Error(), "logo.base64") {
				t.Errorf("error %q does not name logo.base64, so it does not say which keys", err)
			}
		})
	}
}

// TestADriveIdThatIsNotOneIsRefusedByName is all this package can judge about
// a file it cannot read: the shape of the id. 25 or more of the characters
// Drive mints ids from, and nothing else.
func TestADriveIdThatIsNotOneIsRefusedByName(t *testing.T) {
	for _, bad := range []string{"", "a/b", "x", "1AbCdEfGhIjKlMnOpQrStUvW"} {
		t.Run("refused "+bad, func(t *testing.T) {
			tail := "  drive: " + strconv.Quote(bad) + "\n"
			_, err := LoadFile(write(t, houseWithLogo(t, logoBlock+tail)))
			if err == nil {
				t.Fatalf("drive: %q was accepted as a file id", bad)
			}
			if !strings.Contains(err.Error(), "logo.drive") {
				t.Errorf("error %q does not name logo.drive", err)
			}
		})
	}
	for _, good := range []string{aDriveID, "1AbCdEfGhIjKlMnOpQrStUvWx"} {
		t.Run("accepted "+good, func(t *testing.T) {
			tail := "  drive: " + good + "\n"
			if _, err := LoadFile(write(t, houseWithLogo(t, logoBlock+tail))); err != nil {
				t.Errorf("drive: %q was refused: %v", good, err)
			}
		})
	}
}

// TestADriveLogoCarriesNoBytes is the other half of the move: with the bytes
// in Drive this package checks no PNG, because it has none and reaches
// nothing, and says so when somebody asks it for the bytes. What it still
// requires is everything render reads to place the mark on the page.
func TestADriveLogoCarriesNoBytes(t *testing.T) {
	cfg, err := LoadFile(write(t, houseWithLogo(t, logoBlock+"  drive: "+aDriveID+"\n")))
	if err != nil {
		t.Fatalf("a logo in Drive was refused: %v", err)
	}
	if cfg.Logo.Drive != aDriveID {
		t.Errorf("logo.drive = %q, want %s", cfg.Logo.Drive, aDriveID)
	}
	raw, err := cfg.LogoBytes()
	if err == nil {
		t.Fatalf("LogoBytes returned %d bytes for a logo that is in Drive", len(raw))
	}
	if !strings.Contains(err.Error(), aDriveID) {
		t.Errorf("error %q does not name the file the bytes are in", err)
	}
	for _, c := range []struct{ name, drop, names string }{
		{"no width", "  width_pt: 146.25\n", "logo.width_pt"},
		{"no anchor", "    in: first_page_header\n", "logo.anchor"},
		{"no wrap", "    wrap: square\n", "logo.anchor.wrap"},
	} {
		t.Run(c.name, func(t *testing.T) {
			block := strings.Replace(logoBlock, c.drop, "", 1)
			if block == logoBlock {
				t.Fatalf("the line %q is not in the block, so nothing was dropped", c.drop)
			}
			_, err := LoadFile(write(t, houseWithLogo(t, block+"  drive: "+aDriveID+"\n")))
			if err == nil {
				t.Fatalf("a logo with %s was accepted", c.name)
			}
			if !strings.Contains(err.Error(), c.names) {
				t.Errorf("error %q does not name %s", err, c.names)
			}
		})
	}
}

// The firm's logo is one file in its Drive, and the bytes are in no file here.
// What stands in for them is testdata/placeholder-logo.png, a flat light grey
// PNG of the size the house file states, and testdata/house-base64.yaml, the
// embedded file with that placeholder's base64 where the firm's used to be.
// Every test that needs real logo bytes reads one of the two.
//
// The placeholder was written once, by hand, as a PNG of colour type 6: 195 by
// 97 pixels of #D9D9D9 at full alpha, one IHDR, one IDAT and one IEND. Go's
// png.Encode drops a fully opaque image to colour type 2, and the firm's logo
// is type 6, so the stand-in is written rather than encoded. The YAML beside
// it is house.yaml up to the key that names the logo's bytes, then
// "  base64: |-" and that PNG's standard base64 wrapped at 76 characters,
// indented four spaces. Either can be made again from those two sentences.
const (
	placeholderLogoPath = "testdata/placeholder-logo.png"
	base64HousePath     = "testdata/house-base64.yaml"
)

// TestThePlaceholderIsAPNGOfTheHouseSize holds the one thing the stand-in has
// to be: a PNG of the pixels the house file states. Both numbers are literals,
// as every house value in this file is.
func TestThePlaceholderIsAPNGOfTheHouseSize(t *testing.T) {
	img, err := png.DecodeConfig(bytes.NewReader(placeholderLogoBytes(t)))
	if err != nil {
		t.Fatalf("the placeholder is not a PNG: %v", err)
	}
	if img.Width != 509 || img.Height != 185 {
		t.Errorf("the placeholder is %dx%d px, want 509x185", img.Width, img.Height)
	}
}

// TestTheBase64HouseCarriesThePlaceholderAndNothingElse is the other half of
// the rule TestTheLogoIsNotInTheTree states, and the half that test cannot
// reach. The boundary scan hashes whole files, so base64 text sitting inside a
// larger YAML file is invisible to it, and this file is the one place in the
// tree whose whole purpose is to carry a logo's bytes. Nothing else pins what
// those bytes are: the copy is checked against house.yaml only above the key
// that names them, and every test reading it asks for a PNG of 195 by 97,
// which the firm's own logo also is. So the firm's bytes pasted here would
// pass every other test in the tree. This one decodes them and holds them to
// the placeholder, byte for byte.
func TestTheBase64HouseCarriesThePlaceholderAndNothingElse(t *testing.T) {
	raw, err := loadBase64House(t).LogoBytes()
	if err != nil {
		t.Fatalf("LogoBytes: %v", err)
	}
	if !bytes.Equal(raw, placeholderLogoBytes(t)) {
		t.Errorf("%s carries %d logo bytes that are not %s; the stand-in is the only picture this tree holds",
			base64HousePath, len(raw), placeholderLogoPath)
	}
}

// TestTheEmbeddedLogoIsADriveIdAndCarriesNoBytes is where this milestone ends:
// the shipped house names the firm's file and carries none of it. Bytes put
// back under logo.base64 fail here, which is the point: the tree holds no
// copy of the firm's mark.
func TestTheEmbeddedLogoIsADriveIdAndCarriesNoBytes(t *testing.T) {
	cfg := load(t)
	if strings.TrimSpace(cfg.Logo.Base64) != "" {
		t.Fatal("the embedded house carries logo.base64, and the firm's bytes do not live in this tree")
	}
	if cfg.LogoDrive() == "" {
		t.Fatal("the embedded logo names neither bytes nor a file in Drive")
	}
	raw, err := cfg.LogoBytes()
	if err == nil {
		t.Fatalf("the embedded house handed over %d logo bytes, and its logo is in Drive", len(raw))
	}
	if !strings.Contains(err.Error(), cfg.LogoDrive()) {
		t.Errorf("error %q does not name the file the bytes are in", err)
	}
}

// placeholderLogoBytes is the stand-in logo as bytes.
func placeholderLogoBytes(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(placeholderLogoPath)
	if err != nil {
		t.Fatalf("read the placeholder: %v", err)
	}
	return raw
}

// base64House is the copy of the embedded file whose logo carries the
// placeholder's bytes. It is checked against the embedded file first, because
// a copy that drifted from it would quietly test a style nothing ships: the
// two must be byte for byte the same up to the key that names the logo's
// bytes, which is the one line they are allowed to differ in.
func base64House(t *testing.T) []byte {
	t.Helper()
	copied, err := os.ReadFile(base64HousePath)
	if err != nil {
		t.Fatalf("read %s: %v", base64HousePath, err)
	}
	if !bytes.Equal(upToTheLogoBytes(t, copied), upToTheLogoBytes(t, embedded)) {
		t.Fatalf("%s has drifted from house.yaml above the logo's bytes, so it tests a style nothing ships", base64HousePath)
	}
	return copied
}

// loadBase64House is base64House parsed, with the error already checked.
func loadBase64House(t *testing.T) *Config {
	t.Helper()
	cfg, err := LoadFile(write(t, base64House(t)))
	if err != nil {
		t.Fatalf("LoadFile %s: %v", base64HousePath, err)
	}
	return cfg
}

// upToTheLogoBytes is one house file with everything from the key that names
// the logo's bytes onwards cut off. It takes whichever of the two keys is
// there, so it reads the same before and after the firm's bytes leave the
// tree.
func upToTheLogoBytes(t *testing.T, src []byte) []byte {
	t.Helper()
	for _, key := range []string{"\n  base64: |-\n", "\n  drive: "} {
		switch bytes.Count(src, []byte(key)) {
		case 0:
			continue
		case 1:
			return src[:bytes.Index(src, []byte(key))]
		default:
			t.Fatalf("a house file names %q more than once, so the logo's bytes have no one place", key)
		}
	}
	t.Fatal("a house file names neither logo.base64 nor logo.drive")
	return nil
}
