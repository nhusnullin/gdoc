// Package house holds the Altery house style as a file, parsed.
//
// house.yaml is the style: page geometry, the nine named styles, the cover,
// the header and footer with the positioned logo, the three front-matter
// tables cell by cell, the legend, the live contents field, heading numbering
// and the logo itself as base64. TestTheNineStylesAreThere,
// TestFrontMatterIsThirteenBlocksInOrder, TestTheThreeTablesAreThereWithTheirColumns,
// TestTOCIsALiveFieldOverThreeHeadingLevels and TestTheLogoDecodesAsAPNGOfTheStatedSize
// are the pins over what the file has to carry.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The style is a file, and the Word master is provenance
//
// Nail's decision, 2026-08-29, DECISIONS.md. Nothing reads the master document
// at run time and nothing copies a master and edits it: the docx is written
// from this file, part by part. The failure that decision was taken against is
// a surgery file becoming the real template while the master stops describing
// the output. So this package reaches nothing: no network, no other file, and
// no external program. TestNothingHereReachesTheNetwork is the pin, and it
// bans os/exec in the same breath.
//
// # It is embedded, and LoadFile replaces it for one run
//
// go:embed house.yaml, so installing gdoc is still one file. LoadFile parses a
// named copy under the same strict rules, which is how a change to the style is
// reviewed before it is committed, and the caller reports which of the two it
// read so a document built from a draft says so. TestLoadReadsTheEmbeddedFile,
// TestLoadFileReadsACopyOfTheEmbeddedFile and TestLoadFileOnAMissingPathNamesThePath
// are the pins.
//
// # Points stay points
//
// Every measurement is stored in points. Twips, half-points and EMU are
// computed at the writer, never here, so one value in the file has one meaning
// and a unit conversion lives in one room. TestPageIsA4InPoints and
// TestUsableWidthIsTheSpaceBetweenTheSideMargins are the pins.
//
// # A highlight is a name, and Validate refuses anything else
//
// w:highlight takes one of OOXML's seventeen names, never a colour, while every
// other colour in this file is hex. A hex value there reaches the part as
// w:val="#FFFF00", which Word repairs the document over rather than showing.
// Writing hex is the natural mistake and nothing downstream would have named
// it, so the file is checked at the door.
// TestAHighlightThatIsAColourIsRefused is the pin.
//
// Validate is the same rule over the rest of the file: an unknown key, a block
// naming a table or a label that is not there, an unknown block kind, a missing
// style, and a logo that is not a PNG are each refused naming what was written.
// TestAnUnknownKeyIsRefusedNamingIt, TestABlockNamingATableThatIsNotThereIsRefused,
// TestABlockNamingALabelThatIsNotThereIsRefused, TestAnUnknownBlockKindIsRefusedNamingIt,
// TestAMissingStyleIsRefusedNamingIt and TestALogoThatIsNotAPNGIsRefused are the
// pins. A style file gdoc half understands is a document somebody publishes.
//
// # A house-style test states its value as a literal
//
// if got != 595.28, never if got != cfg.Page.WidthPt. A test that reads the
// constant it checks is a mirror: move the constant and the assertion follows
// it and still passes, so it says nothing about what the house style is. Every
// cfg. inside these tests is the printed actual in a t.Errorf, with the want
// written out beside it as a number. The invariant is in CLAUDE.md, and it
// holds for every test that reads this package.
package house

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed house.yaml
var embedded []byte

// styleNames is the nine styles a house config must define. A tenth is refused
// and a missing one is refused, both by name: render reads these by name, so a
// config that carries something else is a config it would build silently
// wrong.
var styleNames = []string{
	"normal",
	"heading_1", "heading_2", "heading_3", "heading_4", "heading_5", "heading_6",
	"title", "subtitle",
}

// blockKinds is what a front_matter entry may name.
var blockKinds = map[string]bool{
	"cover": true, "label": true, "table": true,
	"blank": true, "legend": true, "toc": true,
}

// Config mirrors house.yaml, one field per key.
type Config struct {
	Meta             Meta              `yaml:"meta"`
	Page             Page              `yaml:"page"`
	Palette          map[string]string `yaml:"palette"`
	Fonts            Fonts             `yaml:"fonts"`
	Defaults         Defaults          `yaml:"defaults"`
	Styles           map[string]Style  `yaml:"styles"`
	Body             Body              `yaml:"body"`
	Header           Section           `yaml:"header"`
	Footer           Section           `yaml:"footer"`
	Cover            Cover             `yaml:"cover"`
	Labels           map[string]Label  `yaml:"labels"`
	Legend           Legend            `yaml:"legend"`
	FrontMatter      []Block           `yaml:"front_matter"`
	Tables           map[string]Table  `yaml:"tables"`
	TableText        TableText         `yaml:"table_text"`
	TOC              TOC               `yaml:"toc"`
	HeadingNumbering HeadingNumbering  `yaml:"heading_numbering"`
	Logo             Logo              `yaml:"logo"`
}

// Meta names the style and the version it was extracted at.
type Meta struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Note    string `yaml:"note"`
}

// Page is the geometry, in points.
type Page struct {
	Size               string  `yaml:"size"`
	WidthPt            float64 `yaml:"width_pt"`
	HeightPt           float64 `yaml:"height_pt"`
	MarginTopPt        float64 `yaml:"margin_top_pt"`
	MarginBottomPt     float64 `yaml:"margin_bottom_pt"`
	MarginLeftPt       float64 `yaml:"margin_left_pt"`
	MarginRightPt      float64 `yaml:"margin_right_pt"`
	HeaderMarginPt     float64 `yaml:"header_margin_pt"`
	FooterMarginPt     float64 `yaml:"footer_margin_pt"`
	DifferentFirstPage bool    `yaml:"different_first_page"`
	PageNumberStart    int     `yaml:"page_number_start"`
}

// UsableWidthPt is the width between the two side margins, which is what a
// body table and a full-width image are sized to.
func (p Page) UsableWidthPt() float64 {
	return p.WidthPt - p.MarginLeftPt - p.MarginRightPt
}

// Fonts names the three faces the style uses.
type Fonts struct {
	Body     string `yaml:"body"`
	Title    string `yaml:"title"`
	Subtitle string `yaml:"subtitle"`
}

// Defaults is what a paragraph gets when it says nothing.
type Defaults struct {
	Font          string  `yaml:"font"`
	SizePt        float64 `yaml:"size_pt"`
	SpaceBeforePt float64 `yaml:"space_before_pt"`
	SpaceAfterPt  float64 `yaml:"space_after_pt"`
	LineSpacing   float64 `yaml:"line_spacing"`
}

// Style is one named paragraph style. Every optional number is a pointer,
// because a value absent and a value of zero are two different instructions:
// absent leaves the attribute off, and zero writes it.
type Style struct {
	SizePt            *float64 `yaml:"size_pt"`
	Bold              bool     `yaml:"bold"`
	Italic            bool     `yaml:"italic"`
	Color             string   `yaml:"color"`
	Font              string   `yaml:"font"`
	Align             string   `yaml:"align"`
	SpaceBeforePt     *float64 `yaml:"space_before_pt"`
	SpaceAfterPt      *float64 `yaml:"space_after_pt"`
	LineSpacing       *float64 `yaml:"line_spacing"`
	IndentStartPt     *float64 `yaml:"indent_start_pt"`
	KeepWithNext      bool     `yaml:"keep_with_next"`
	KeepLinesTogether bool     `yaml:"keep_lines_together"`
}

// Body is how the note's own markdown is set.
type Body struct {
	SizePt                float64      `yaml:"size_pt"`
	Align                 string       `yaml:"align"`
	SpaceBeforePt         float64      `yaml:"space_before_pt"`
	SpaceAfterPt          float64      `yaml:"space_after_pt"`
	FirstHeadingPageBreak bool         `yaml:"first_heading_page_break"`
	Bullet                BulletList   `yaml:"bullet"`
	Numbered              NumberedList `yaml:"numbered"`
}

// BulletList is the bulleted list's indents and its glyph per level.
type BulletList struct {
	IndentStartPt float64  `yaml:"indent_start_pt"`
	HangingPt     float64  `yaml:"hanging_pt"`
	Glyphs        []string `yaml:"glyphs"`
}

// NumberedList is the numbered list's indents and its level-1 format.
type NumberedList struct {
	IndentStartPt float64 `yaml:"indent_start_pt"`
	HangingPt     float64 `yaml:"hanging_pt"`
	Format        string  `yaml:"format"`
}

// Section is a header or a footer: the first page's and every other page's.
type Section struct {
	First   Region `yaml:"first"`
	Default Region `yaml:"default"`
}

// Region is one header or footer, its paragraphs and the defaults they inherit.
type Region struct {
	Align         string      `yaml:"align"`
	LineSpacing   *float64    `yaml:"line_spacing"`
	IndentStartPt *float64    `yaml:"indent_start_pt"`
	Paragraphs    []Paragraph `yaml:"paragraphs"`
}

// Paragraph is one line of a header or a footer.
type Paragraph struct {
	Align         string   `yaml:"align"`
	LineSpacing   *float64 `yaml:"line_spacing"`
	SpaceBeforePt *float64 `yaml:"space_before_pt"`
	SpaceAfterPt  *float64 `yaml:"space_after_pt"`
	SizePt        *float64 `yaml:"size_pt"`
	Color         string   `yaml:"color"`
	Runs          []Run    `yaml:"runs"`
	Logo          bool     `yaml:"logo"`
	Rule          *Rule    `yaml:"rule"`
}

// Rule is the horizontal line under the running head.
type Rule struct {
	HeightPt float64 `yaml:"height_pt"`
	Color    string  `yaml:"color"`
}

// Run is a stretch of text, or a tab, or the page number field. Placeholder
// names the cover field whose value replaces the text when the note carries
// one.
type Run struct {
	Text        string   `yaml:"text"`
	SizePt      *float64 `yaml:"size_pt"`
	Color       string   `yaml:"color"`
	Bold        bool     `yaml:"bold"`
	Italic      bool     `yaml:"italic"`
	Underline   bool     `yaml:"underline"`
	Highlight   string   `yaml:"highlight"`
	Tab         bool     `yaml:"tab"`
	Tabs        int      `yaml:"tabs"`
	PageNumber  bool     `yaml:"page_number"`
	Placeholder string   `yaml:"placeholder"`
}

// Cover is the first page: blank paragraphs, the lines, then more blanks. An
// empty paragraph still carries a run size, because the cover's vertical
// rhythm is made of their heights.
type Cover struct {
	LeadingBlanks  []Blank     `yaml:"leading_blanks"`
	Align          string      `yaml:"align"`
	Lines          []CoverLine `yaml:"lines"`
	TrailingBlanks []Blank     `yaml:"trailing_blanks"`
}

// Blank is an empty paragraph with a height.
type Blank struct {
	Align         string   `yaml:"align"`
	SizePt        *float64 `yaml:"size_pt"`
	SpaceBeforePt *float64 `yaml:"space_before_pt"`
	SpaceAfterPt  *float64 `yaml:"space_after_pt"`
	LineSpacing   *float64 `yaml:"line_spacing"`
}

// CoverLine is one line of the cover. A line naming a placeholder is replaced
// by the note's own value; without one it prints the template's words, which
// are the highlighted placeholders a person fills in by hand.
//
// With names a cover field the line depends on, and the line is left out when
// the note did not fill that field in. The master offers the title twice
// either side of an "or", so a person filling the cover in by hand picks one:
// without this, a note with one title published a cover reading the title,
// then "or", then the template's own highlighted placeholder.
type CoverLine struct {
	Text          string   `yaml:"text"`
	SizePt        *float64 `yaml:"size_pt"`
	Bold          bool     `yaml:"bold"`
	Italic        bool     `yaml:"italic"`
	Color         string   `yaml:"color"`
	Highlight     string   `yaml:"highlight"`
	Align         string   `yaml:"align"`
	Placeholder   string   `yaml:"placeholder"`
	With          string   `yaml:"with"`
	SpaceBeforePt *float64 `yaml:"space_before_pt"`
	SpaceAfterPt  *float64 `yaml:"space_after_pt"`
	LineSpacing   *float64 `yaml:"line_spacing"`
	Runs          []Run    `yaml:"runs"`
}

// Label is a heading-like line in the front matter, named by a block's ref.
type Label struct {
	Text              string   `yaml:"text"`
	Bold              bool     `yaml:"bold"`
	Italic            bool     `yaml:"italic"`
	SizePt            *float64 `yaml:"size_pt"`
	Color             string   `yaml:"color"`
	Align             string   `yaml:"align"`
	LineSpacing       *float64 `yaml:"line_spacing"`
	SpaceBeforePt     *float64 `yaml:"space_before_pt"`
	SpaceAfterPt      *float64 `yaml:"space_after_pt"`
	KeepWithNext      bool     `yaml:"keep_with_next"`
	KeepLinesTogether bool     `yaml:"keep_lines_together"`
}

// Legend is the four-line key under the revision history. Each line is a bold
// word and the sentence behind it.
type Legend struct {
	SizePt      *float64   `yaml:"size_pt"`
	Color       string     `yaml:"color"`
	LineSpacing *float64   `yaml:"line_spacing"`
	Lines       [][]string `yaml:"lines"`
}

// Block is one entry in the front matter's order. Ref names the label or the
// table, and the rest is the blank block's own shape.
type Block struct {
	Block         string   `yaml:"block"`
	Ref           string   `yaml:"ref"`
	Count         int      `yaml:"count"`
	Align         string   `yaml:"align"`
	SizePt        *float64 `yaml:"size_pt"`
	LineSpacing   *float64 `yaml:"line_spacing"`
	SpaceBeforePt *float64 `yaml:"space_before_pt"`
	SpaceAfterPt  *float64 `yaml:"space_after_pt"`
}

// Table is one front-matter table, cell by cell.
type Table struct {
	ColumnsPt []float64 `yaml:"columns_pt"`
	Border    Border    `yaml:"border"`
	Rows      []Row     `yaml:"rows"`
}

// Border is the single line every cell edge carries.
type Border struct {
	WidthPt float64 `yaml:"width_pt"`
	Color   string  `yaml:"color"`
}

// Row is one table row and the height it is at least. Repeat names the note's
// own list this row is a prototype for: "revisions" renders the row once per
// revision the note declares, and a note with none leaves the template's row
// where it is, which is v1's rule. Without is the other half: the blank row a
// person would fill in by hand is left out once the note declares the rows
// itself.
type Row struct {
	MinHeightPt float64 `yaml:"min_height_pt"`
	Repeat      string  `yaml:"repeat"`
	Without     string  `yaml:"without"`
	Cells       []Cell  `yaml:"cells"`
}

// Cell is one table cell. PadTLBRPt, when it is there, wins over PadPt.
//
// Classification names the class this cell describes, and a cell that names
// one is filled only when the note declares that class. The master was
// captured with Internal marked, so writing its fills verbatim marked every
// document Internal whatever the note said.
type Cell struct {
	Fill           string          `yaml:"fill"`
	Classification string          `yaml:"classification"`
	PadPt          *float64        `yaml:"pad_pt"`
	PadTLBRPt      []float64       `yaml:"pad_tlbr_pt"`
	Valign         string          `yaml:"valign"`
	Paragraphs     []CellParagraph `yaml:"paragraphs"`
}

// FillFor is the shading this cell carries in a document declaring the named
// classification.
//
// A cell that names a classification is the description beside one class, and
// it is shaded only when the document declares that class. The master was
// captured with Internal marked, so writing the file's fills verbatim marked
// every document Internal whatever the note said, which is worse than leaving
// the row blank. It is v1's mark_classification: clear every description cell,
// then shade the chosen one.
//
// The rule is here rather than in a writer because the house style is written
// twice, once as a docx by internal/render and once as Docs requests by
// internal/prelude, and a document that is marked Internal in one and unmarked
// in the other is the failure the mechanism exists to stop.
func (c Cell) FillFor(classification string) string {
	if c.Classification == "" {
		return c.Fill
	}
	if c.Classification != classification {
		return ""
	}
	return c.Fill
}

// CellParagraph is one paragraph inside a table cell.
type CellParagraph struct {
	Runs              []Run    `yaml:"runs"`
	Align             string   `yaml:"align"`
	LineSpacing       *float64 `yaml:"line_spacing"`
	SpaceBeforePt     *float64 `yaml:"space_before_pt"`
	SpaceAfterPt      *float64 `yaml:"space_after_pt"`
	IndentStartPt     *float64 `yaml:"indent_start_pt"`
	IndentFirstLinePt *float64 `yaml:"indent_first_line_pt"`
	KeepWithNext      bool     `yaml:"keep_with_next"`
	KeepLinesTogether bool     `yaml:"keep_lines_together"`
}

// TableText is the face and size a table cell falls back to.
type TableText struct {
	Font          string  `yaml:"font"`
	DefaultSizePt float64 `yaml:"default_size_pt"`
}

// TOC is the contents list. It is a Word field, so Google refreshes it live
// rather than gdoc measuring the pages.
type TOC struct {
	Live   bool    `yaml:"live"`
	Instr  string  `yaml:"instr"`
	SizePt float64 `yaml:"size_pt"`
}

// HeadingNumbering is how a level-1 heading gets its number. The mechanism is
// literal on purpose: list numbering would renumber the ordinary paragraphs
// between the headings.
type HeadingNumbering struct {
	Mechanism    string `yaml:"mechanism"`
	Level1Format string `yaml:"level1_format"`
	Note         string `yaml:"note"`
}

// Logo is the mark in the first page's header, carried in the file itself so
// the binary is the whole dependency.
type Logo struct {
	Mime     string  `yaml:"mime"`
	Px       []int   `yaml:"px"`
	WidthPt  float64 `yaml:"width_pt"`
	HeightPt float64 `yaml:"height_pt"`
	Anchor   Anchor  `yaml:"anchor"`
	Base64   string  `yaml:"base64"`
}

// Anchor is where the logo sits and what the text does around it.
type Anchor struct {
	In         string  `yaml:"in"`
	RelativeH  string  `yaml:"relative_h"`
	OffsetXPt  float64 `yaml:"offset_x_pt"`
	RelativeV  string  `yaml:"relative_v"`
	OffsetYPt  float64 `yaml:"offset_y_pt"`
	Wrap       string  `yaml:"wrap"`
	WrapDistPt float64 `yaml:"wrap_dist_pt"`
}

// Load returns the embedded house style.
func Load() (*Config, error) {
	return parse(embedded, "embedded")
}

// LoadFile returns the house style in one named file, read under the same
// strict rules as the embedded one. The error names the path, because a run
// given --house has two configs in play and has to say which one it refused.
func LoadFile(path string) (*Config, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("house style %s: %w", path, err)
	}
	return parse(src, path)
}

// parse decodes one house style strictly and validates it. name is what an
// error calls the source.
func parse(src []byte, name string) (*Config, error) {
	var cfg Config
	if err := yaml.UnmarshalWithOptions(src, &cfg, yaml.Strict()); err != nil {
		return nil, fmt.Errorf("house style %s: %w", name, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("house style %s: %w", name, err)
	}
	return &cfg, nil
}

// Style returns one named style.
func (c *Config) Style(name string) (Style, bool) {
	s, ok := c.Styles[name]
	return s, ok
}

// LogoBytes decodes the logo out of the config.
func (c *Config) LogoBytes() ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(c.Logo.Base64))
	if err != nil {
		return nil, fmt.Errorf("logo: base64: %w", err)
	}
	return raw, nil
}

// Validate says what YAML cannot: the nine styles are there and nothing else
// is, every front_matter block names a kind that exists and a ref that is
// there, every highlight is one of OOXML's names rather than a colour, and the
// logo really is a PNG of the size the file states. Each of those is something
// render reads by name, so a config that fails one here is a document it would
// build silently wrong.
//
// The list is meant to stay complete: a check added below without a clause
// here leaves --house refusing a draft for a reason this contract never
// mentions.
func (c *Config) Validate() error {
	if err := c.validateStyles(); err != nil {
		return err
	}
	if err := c.validateFrontMatter(); err != nil {
		return err
	}
	if err := c.validateHighlights(); err != nil {
		return err
	}
	return c.validateLogo()
}

// highlightNames is OOXML's ST_HighlightColor, which is the whole of what
// w:highlight takes.
var highlightNames = map[string]bool{
	"black": true, "blue": true, "cyan": true, "darkBlue": true,
	"darkCyan": true, "darkGray": true, "darkGreen": true, "darkMagenta": true,
	"darkRed": true, "darkYellow": true, "green": true, "lightGray": true,
	"magenta": true, "none": true, "red": true, "white": true, "yellow": true,
}

// validateHighlights refuses a highlight the writer cannot spell.
//
// w:highlight takes a name from a fixed list, never a colour: a '#FFFF00' in
// the file reached the part as w:val="#FFFF00", which is not conforming OOXML,
// and Word repairs the document rather than showing the mark. Every other
// colour in the file is a hex value, so writing one here is the natural
// mistake and nothing downstream would have named it.
func (c *Config) validateHighlights() error {
	if err := checkCoverHighlights(c.Cover.Lines); err != nil {
		return err
	}
	// The walk is ordered, and the tables are sorted, because the first bad
	// highlight is the one reported: over a map, a draft misspelling two of
	// them is refused naming a different one on each run, and somebody fixing
	// them one at a time is sent somewhere else every time. validateStyles
	// sorts for the same reason.
	sections := []struct {
		name    string
		section Section
	}{{"header", c.Header}, {"footer", c.Footer}}
	for _, entry := range sections {
		if err := checkSectionHighlights(entry.name, entry.section); err != nil {
			return err
		}
	}
	names := make([]string, 0, len(c.Tables))
	for name := range c.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := checkTableHighlights(name, c.Tables[name]); err != nil {
			return err
		}
	}
	return nil
}

// checkHighlight refuses one value that is not a highlight name. An empty one
// is a run asking for no highlight at all.
func checkHighlight(where, value string) error {
	if value == "" || highlightNames[value] {
		return nil
	}
	return fmt.Errorf("%s: %q is not a highlight name", where, value)
}

// checkRunHighlights is every run of one paragraph, one cell or one cover line.
func checkRunHighlights(where string, list []Run) error {
	for i, run := range list {
		if err := checkHighlight(fmt.Sprintf("%s.runs[%d]", where, i), run.Highlight); err != nil {
			return err
		}
	}
	return nil
}

// checkCoverHighlights is the cover's lines: each line's own highlight, which a
// whole-line placeholder carries, and then its runs.
func checkCoverHighlights(lines []CoverLine) error {
	for i, line := range lines {
		where := fmt.Sprintf("cover.lines[%d]", i)
		if err := checkHighlight(where, line.Highlight); err != nil {
			return err
		}
		if err := checkRunHighlights(where, line.Runs); err != nil {
			return err
		}
	}
	return nil
}

// checkSectionHighlights is a header or a footer, both of its regions.
func checkSectionHighlights(name string, section Section) error {
	regions := []struct {
		name   string
		region Region
	}{{"first", section.First}, {"default", section.Default}}
	for _, entry := range regions {
		for i, paragraph := range entry.region.Paragraphs {
			where := fmt.Sprintf("%s.%s.paragraphs[%d]", name, entry.name, i)
			if err := checkRunHighlights(where, paragraph.Runs); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkTableHighlights is one front-matter table, cell paragraph by cell
// paragraph. The revision row is where a hex value was really written.
func checkTableHighlights(name string, table Table) error {
	for ri, row := range table.Rows {
		for ci, cell := range row.Cells {
			for pi, paragraph := range cell.Paragraphs {
				where := fmt.Sprintf("tables.%s.rows[%d].cells[%d].paragraphs[%d]", name, ri, ci, pi)
				if err := checkRunHighlights(where, paragraph.Runs); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validateStyles requires exactly the nine names, no more and no fewer.
func (c *Config) validateStyles() error {
	want := make(map[string]bool, len(styleNames))
	for _, name := range styleNames {
		want[name] = true
		if _, ok := c.Styles[name]; !ok {
			return fmt.Errorf("styles: %s is missing", name)
		}
	}
	var extra []string
	for name := range c.Styles {
		if !want[name] {
			extra = append(extra, name)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return fmt.Errorf("styles: %s is not a house style", strings.Join(extra, ", "))
	}
	return nil
}

// validateFrontMatter checks every block's kind and its reference.
func (c *Config) validateFrontMatter() error {
	if len(c.FrontMatter) == 0 {
		return fmt.Errorf("front_matter: no blocks")
	}
	for i, b := range c.FrontMatter {
		if !blockKinds[b.Block] {
			return fmt.Errorf("front_matter[%d]: %q is not a block kind", i, b.Block)
		}
		switch b.Block {
		case "label":
			if _, ok := c.Labels[b.Ref]; !ok {
				return fmt.Errorf("front_matter[%d]: no label named %q", i, b.Ref)
			}
		case "table":
			if _, ok := c.Tables[b.Ref]; !ok {
				return fmt.Errorf("front_matter[%d]: no table named %q", i, b.Ref)
			}
		}
	}
	return nil
}

// validateLogo decodes the logo and checks it against what the file says it
// is. A stated size that is not the real one would put the mark on the page at
// the wrong scale, and nothing downstream reads the PNG to notice.
func (c *Config) validateLogo() error {
	if c.Logo.Mime != "image/png" {
		return fmt.Errorf("logo: mime is %q, and only image/png is carried", c.Logo.Mime)
	}
	if len(c.Logo.Px) != 2 {
		return fmt.Errorf("logo: px names %d numbers, want width and height", len(c.Logo.Px))
	}
	raw, err := c.LogoBytes()
	if err != nil {
		return err
	}
	img, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("logo: not a PNG: %w", err)
	}
	if img.Width != c.Logo.Px[0] || img.Height != c.Logo.Px[1] {
		return fmt.Errorf("logo: the file says %dx%d px and the image is %dx%d",
			c.Logo.Px[0], c.Logo.Px[1], img.Width, img.Height)
	}
	return nil
}
