// Package house holds the Altery house style as a file, parsed.
//
// house.yaml is the style: page geometry, the ten named styles, the cover,
// the header and footer with the positioned logo, the three front-matter
// tables cell by cell, the legend, the live contents field, heading numbering
// and the logo: its mime, drawn width, placement, and either the bytes or the
// id of the one file in Drive that holds them. TestTheTenStylesAreThere,
// TestFrontMatterIsFourteenBlocksInOrder, TestTheThreeTablesAreThereWithTheirColumns,
// TestTOCIsALiveFieldOverThreeHeadingLevels, TestTheLogoStatesItsMimeWidthAndPlacement,
// TestTheMarkEndsInLineWithTheText and TestTheLabelDropsSixPointsAndTheMarkStays
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
// A page_break block is held to the same rule from the other side: it is the
// whole block, and a ref, a count or a size written beside it is refused naming
// the key. TestAPageBreakCarryingAnotherKeyIsRefusedNamingIt and
// TestACountedPageBreakIsRefused are the pins.
//
// # The marks are the one palette, and Validate refuses the rest
//
// marks: holds the seven authoring marks of 2026-10-14, keyed as Obsidian
// names its six highlight colours plus pencil, each stating every opener an
// author may write, whether it is a highlight or a text colour, the hex it
// reaches the document as, and whether it is counted on the envelope. Nothing
// else holds a mark's emoji or a mark's hex: body reads the openers off this
// file, render writes the shading and the character style from it, and export
// matches the Docs answer against the same hexes, so one colour has one place
// it is written down. TestTheSevenMarksAreStatedAsLiterals and
// TestTheMarksAreTheSevenAndNoOther are the pins over what the file carries.
//
// The door is validateMarks: an eighth mark and a missing one, a kind that is
// neither highlight nor text, a colour that is not six hex digits after a
// hash, a mark with no opener and two marks claiming the same opener are each
// refused naming what was written. TestAMarkWithAnUnknownKindIsRefused,
// TestAMarkWhoseColourIsNotHexIsRefused, TestAMarkWithNoSyntaxIsRefused and
// TestTwoMarksCannotShareASyntax are the pins.
//
// The same door holds the two openers a mark has to claim. Nobody claiming
// '==' is every highlight reaching the page with no colour, and nobody
// claiming a comment's opener is every aside reaching it as prose, so each is
// refused naming the opener nobody claims. The comment's two spellings go to
// one mark, HTML's first, because that is the opener the projection writes and
// the closer it writes is always HTML's.
// TestAPaletteWhereNoMarkClaimsTheBareHighlightIsRefused,
// TestAPaletteWhereNoMarkClaimsACommentIsRefused,
// TestBothCommentSpellingsBelongToOneMark and
// TestTheWrittenCommentSpellingComesFirst are the pins.
// # The logo's bytes are in one of two places, and the file names one
//
// logo.drive is one file in Drive by id and logo.base64 is the bytes here.
// Exactly one of them is set: both is a file with two answers, neither is a
// file with none, and each is refused naming the two keys.
// TestALogoNamesExactlyOneOfDriveAndBase64 is the pin.
//
// The PNG check follows the bytes out of this package. With logo.base64 the
// bytes are here and Validate reads a real PNG size from their header. With
// logo.drive they are not, this package reaches no network, so internal/asset,
// the room that fetches them, checks the mime, byte ceiling and side ceiling.
// All this package can judge there is the id's shape, and it does.
// TestADriveIdThatIsNotOneIsRefusedByName and TestADriveLogoCarriesNoBytes
// are the pins, and the second one holds what stays required either way:
// the drawn width and the anchor. Render reads the picture's own ratio for the
// height. TestALogoThatStatesPixelsIsRefusedNamingTheKey pins that px and
// height_pt are no longer house keys. LogoBytes on a Drive house is an error naming
// the file rather than empty bytes, because an unbranded document is the one
// thing this must not resolve to.
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
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gdoc/internal/asset"

	"github.com/goccy/go-yaml"
)

//go:embed house.yaml
var embedded []byte

// styleNames is the ten styles a house config must define. An eleventh is
// refused and a missing one is refused, both by name: render reads these by
// name, so a config that carries something else is a config it would build
// silently wrong.
//
// footnote_text is the tenth, added with the footnotes of 2026-10-14. It is
// the only one of the ten that is not a Docs named style: a restyle cannot
// set it, and nothing in internal/drift reads it, which is why that package's
// styleIDs stays at nine.
var styleNames = []string{
	"normal",
	"heading_1", "heading_2", "heading_3", "heading_4", "heading_5", "heading_6",
	"title", "subtitle", "footnote_text",
}

// markNames is the seven authoring marks, Obsidian's six highlight colours
// and the one grey. body parses them, render writes a character style for
// each, export reads each back by its colour and the three debts are counted
// on the envelope, all of it by name, so an eighth nobody reads and a seventh
// that is gone are each a document built silently wrong.
var markNames = []string{
	"yellow", "green", "purple", "blue", "red", "orange", "pencil",
}

// MarkNames is the seven marks in the order they are written down: the order
// render writes their character styles in and help prints them in. A copy,
// because a caller holding the slice itself could reorder the palette for
// every other caller.
func MarkNames() []string { return append([]string(nil), markNames...) }

// markKindHighlight is one of the two kinds a mark may be, and the one that
// reaches the docx as shading. The other, text, is the run's own colour.
const markKindHighlight = "highlight"

// markKinds is how a mark reaches the docx: a highlight is run shading with
// the exact hex, and text is the run's own colour.
var markKinds = map[string]bool{markKindHighlight: true, "text": true}

// Mark is one authoring mark. Syntax is every opener an author may write for
// it: one colour emoji for five of the highlights, both '==' and 🟡 for the
// yellow a plain highlight is, and two for the pencil, which is the comment
// it is written as from milestone 20 on, HTML's '<!--' and Obsidian's '%%'.
// The first is the one export writes, which is why HTML's comes first.
// Debt says the mark is counted on the build and publish envelopes, which is
// a fact the skills judge rather than a refusal.
type Mark struct {
	Syntax []string `yaml:"syntax"`
	Kind   string   `yaml:"kind"`
	Color  string   `yaml:"color"`
	Debt   bool     `yaml:"debt"`
}

// IsHighlight says which way the mark's colour is written: a highlight is run
// shading with the exact hex, and the one other kind is the run's own text
// colour. Asked here rather than in render and body, so neither of those two
// writes the name of a kind out and the two cannot read one differently.
func (m Mark) IsHighlight() bool { return m.Kind == markKindHighlight }

// blockKinds is what a front_matter entry may name.
var blockKinds = map[string]bool{
	"cover": true, "label": true, "table": true,
	"blank": true, "legend": true, "toc": true, "page_break": true,
}

// Config mirrors house.yaml, one field per key.
type Config struct {
	Meta             Meta              `yaml:"meta"`
	Page             Page              `yaml:"page"`
	Palette          map[string]string `yaml:"palette"`
	Marks            map[string]Mark   `yaml:"marks"`
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
	SizePt        float64 `yaml:"size_pt"`
	Align         string  `yaml:"align"`
	SpaceBeforePt float64 `yaml:"space_before_pt"`
	SpaceAfterPt  float64 `yaml:"space_after_pt"`
	// ListItemSpaceAfterPt is the gap between two items of a list. The last
	// item takes SpaceAfterPt instead, so the list ends as a paragraph does.
	// build and restyle both read it here. DECISIONS.md, 2026-10-14.
	ListItemSpaceAfterPt  float64      `yaml:"list_item_space_after_pt"`
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
// where it is. Without is the other half: the blank row a person would fill in
// by hand is left out once the note declares the rows itself.
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
// the row blank. So the rule is: clear every description cell, then shade the
// chosen one.
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

// Logo is the mark in the first page's header. Its drawn width, placement and
// mime are always here; its height follows the picture's own ratio. The bytes
// are in one of two places, and Drive and Base64 are those two. Exactly one of
// them is set.
type Logo struct {
	Mime    string  `yaml:"mime"`
	WidthPt float64 `yaml:"width_pt"`
	Anchor  Anchor  `yaml:"anchor"`
	Drive   string  `yaml:"drive"`
	Base64  string  `yaml:"base64"`
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

// LogoDrive is the id of the file in Drive this house's logo is, or empty
// when the bytes are in logo.base64. It is the one place the key is trimmed,
// so the question "is this logo in Drive" has one answer wherever it is
// asked: here, in Validate, and in the command that fetches it.
func (c *Config) LogoDrive() string {
	return strings.TrimSpace(c.Logo.Drive)
}

// LogoBytes decodes the logo out of the config. A house whose logo is one
// file in Drive carries no bytes here, and the error names the file so
// whoever reads it knows where to look: internal/asset fetches those, and
// this package reaches nothing.
func (c *Config) LogoBytes() ([]byte, error) {
	if drive := c.LogoDrive(); drive != "" {
		return nil, fmt.Errorf("logo: this house's logo is file %s in Drive, and logo.base64 is not set", drive)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(c.Logo.Base64))
	if err != nil {
		return nil, fmt.Errorf("logo: base64: %w", err)
	}
	return raw, nil
}

// Validate says what YAML cannot: the ten styles are there and nothing else
// is, every front_matter block names a kind that exists and a ref that is
// there, every highlight is one of OOXML's names rather than a colour, the
// seven marks are there with a kind, a hex colour and an opener of their own,
// and the logo really has a PNG header with positive dimensions. Each is something
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
	if err := c.validateMarks(); err != nil {
		return err
	}
	return c.validateLogo()
}

// validateMarks requires exactly the seven marks, each with a kind the writer
// knows, a colour it can write and at least one opener an author can type, no
// two of them claiming the same opener, and one highlight claiming the bare
// "==" an author writes when they mean nothing by the colour.
//
// The bare delimiter is required because internal/body resolves it to a mark by
// name: a palette where no mark claims it resolves to no name at all, and every
// ==highlight== in the note then reached the document with no colour on it and
// nothing said so. A house file is a file somebody edits, which is why this is
// a refusal and not a comment.
//
// The walk is ordered for the reason validateHighlights is ordered: a draft
// with two bad marks in it is refused naming the same one on every run, so
// somebody fixing them one at a time is not sent somewhere else each time.
func (c *Config) validateMarks() error {
	want := make(map[string]bool, len(markNames))
	for _, name := range markNames {
		want[name] = true
		if _, ok := c.Marks[name]; !ok {
			return fmt.Errorf("marks: %s is missing", name)
		}
	}
	var extra []string
	for name := range c.Marks {
		if !want[name] {
			extra = append(extra, name)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return fmt.Errorf("marks: %s is not a house mark", strings.Join(extra, ", "))
	}
	owner := make(map[string]string, len(markNames))
	for _, name := range markNames {
		mark := c.Marks[name]
		if !markKinds[mark.Kind] {
			return fmt.Errorf("marks.%s: %q is not a mark kind", name, mark.Kind)
		}
		if !isHexColor(mark.Color) {
			return fmt.Errorf("marks.%s: %q is not a hex colour", name, mark.Color)
		}
		if len(mark.Syntax) == 0 {
			return fmt.Errorf("marks.%s: no syntax, so nothing an author writes is this mark", name)
		}
		for _, syntax := range mark.Syntax {
			if syntax == "" {
				return fmt.Errorf("marks.%s: an empty syntax is not something an author writes", name)
			}
			if first, taken := owner[syntax]; taken {
				return fmt.Errorf("marks.%s: %q is already mark %s", name, syntax, first)
			}
			owner[syntax] = name
		}
	}
	if name := owner[plainHighlight]; name == "" {
		return fmt.Errorf("marks: no mark claims %q, which is the highlight an author writes when they mean nothing by the colour, so every one of them would reach the document with no colour", plainHighlight)
	} else if !c.Marks[name].IsHighlight() {
		return fmt.Errorf("marks.%s: %q is a highlight an author writes, and this mark is %q rather than a highlight", name, plainHighlight, c.Marks[name].Kind)
	}
	return c.validateComment(owner)
}

// validateComment holds the palette to the rule a comment is read and written
// by: one mark claims both spellings an author may write a comment with, the
// HTML one first, and that mark is a text colour.
//
// Each clause is a document published wrong where it is missing. A spelling no
// mark claims resolves to no mark at all in internal/body, so the aside the
// author meant nobody to read reaches the page as black prose and the build
// says ok, which is the same failure the bare highlight is checked for one
// clause above. Two marks sharing the work would write one spelling's comment
// with the other's opener, because internal/export writes the mark's first
// syntax and always the same closer. A highlight there would shade the aside
// instead of greying it, and bring it home as a highlight rather than as the
// comment it was.
// TestAPaletteWhereNoMarkClaimsACommentIsRefused,
// TestBothCommentSpellingsBelongToOneMark,
// TestTheWrittenCommentSpellingComesFirst and
// TestTheCommentBelongsToATextColour are the pins.
func (c *Config) validateComment(owner map[string]string) error {
	first := ""
	for _, spelling := range commentSpellings {
		name := owner[spelling]
		if name == "" {
			return fmt.Errorf("marks: no mark claims %q, which is how an author writes a comment, so every aside written with it would reach the document as prose rather than grey", spelling)
		}
		if first == "" {
			first = name
		}
		if name != first {
			return fmt.Errorf("marks.%s: %q is a comment an author writes and mark %s claims the other spelling: one mark carries both, because the projection writes that mark's first opener and always the same closer", name, spelling, first)
		}
	}
	mark := c.Marks[first]
	if mark.Syntax[0] != commentSpellings[0] {
		return fmt.Errorf("marks.%s: %q is the spelling a document comes home in, so it is this mark's first opener and not %q: the projection writes the first opener and shuts it with the HTML closer", first, commentSpellings[0], mark.Syntax[0])
	}
	if mark.IsHighlight() {
		return fmt.Errorf("marks.%s: %q is a comment an author writes, and this mark is %q rather than a text colour", first, commentSpellings[0], mark.Kind)
	}
	return nil
}

// commentSpellings is the two openers a comment may be written with, HTML's
// first because that is the one a document comes home in. They are stated here
// as well as in internal/body for the reason plainHighlight is: this package is
// the door a palette passes, and a palette claiming neither of them publishes
// an author's own aside as prose.
// TestTheSevenMarksAreStatedAsLiterals pins the two the file gives the pencil.
var commentSpellings = []string{"<!--", "%%"}

// plainHighlight is the syntax a bare ==highlight== is written with. It is
// stated in internal/body and in internal/export as well, each for its own
// reason. TestAPaletteWhereNoMarkClaimsTheBareHighlightIsRefused and
// TestTheBareHighlightBelongsToAHighlight are the pins over the two refusals it
// is here for.
const plainHighlight = "=="

// isHexColor is '#' and six hex digits, which is what w:shd w:fill and w:color
// take and what the Docs answer is matched against.
func isHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
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

// validateStyles requires exactly the ten names, no more and no fewer.
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
		case "page_break":
			if key := pageBreakExtra(b); key != "" {
				return fmt.Errorf("front_matter[%d]: a page_break block carries no other key, and this one names %s", i, key)
			}
		}
	}
	return nil
}

// pageBreakExtra names the first key a page_break block carries beside its
// kind, or the empty string when it carries none.
//
// A page break is the whole block: a ref, a count or a size written beside it
// is a value both writers would drop in silence. The docx route makes the
// break an empty paragraph and the Docs route sends insertPageBreak, so a
// height stated here could only be honoured by one of them, and one layout
// that is a paragraph taller in the docx than in the Doc is the two routes
// drifting apart.
func pageBreakExtra(b Block) string {
	switch {
	case b.Ref != "":
		return "ref"
	case b.Count != 0:
		return "count"
	case b.Align != "":
		return "align"
	case b.SizePt != nil:
		return "size_pt"
	case b.LineSpacing != nil:
		return "line_spacing"
	case b.SpaceBeforePt != nil:
		return "space_before_pt"
	case b.SpaceAfterPt != nil:
		return "space_after_pt"
	}
	return ""
}

// driveFileID is the shape Drive mints a file id in: 25 or more of these
// characters and nothing else. The length is a floor and not a size, because
// Drive has minted ids of several lengths. It is all this package can judge
// about a file it cannot read.
var driveFileID = regexp.MustCompile(`^[A-Za-z0-9_-]{25,}$`)

// validateLogo checks the logo against what the file says of it.
//
// The bytes are in one of two places and the file names exactly one of them:
// logo.drive, one file in Drive by id, or logo.base64, the bytes here. Both
// is a file with two answers and neither is a file with none, and each is
// refused naming the two keys.
//
// The PNG check follows the bytes. With logo.base64 the bytes are here and
// the check is here. With logo.drive they are not, this package reaches no
// network and never will, so internal/asset checks the PNG against the mime,
// byte ceiling and side ceiling. What stays here either way is everything
// render reads to place the mark on the page: the width and the anchor.
func (c *Config) validateLogo() error {
	if c.Logo.Mime != "image/png" {
		return fmt.Errorf("logo: mime is %q, and only image/png is carried", c.Logo.Mime)
	}
	if err := c.validateLogoPlacement(); err != nil {
		return err
	}
	drive, b64 := c.LogoDrive(), strings.TrimSpace(c.Logo.Base64)
	switch {
	case drive != "" && b64 != "":
		return fmt.Errorf("logo: exactly one of logo.drive and logo.base64 is wanted, and both are set")
	case drive == "" && b64 == "":
		return fmt.Errorf("logo: exactly one of logo.drive and logo.base64 is wanted, and neither is set")
	case drive != "":
		if !driveFileID.MatchString(drive) {
			return fmt.Errorf("logo.drive: %q is not a Drive file id", drive)
		}
		return nil
	}
	return c.validateLogoPNG()
}

// validateLogoPlacement requires every value render reads to put the mark on
// the page. A logo whose bytes are in Drive is still placed from this file, so
// this holds whichever of the two keys names the bytes.
//
// The walk is ordered for the reason validateMarks is ordered: a file with two
// of these missing is refused naming the same one on every run.
func (c *Config) validateLogoPlacement() error {
	for _, f := range []struct {
		key   string
		value float64
	}{
		{"logo.width_pt", c.Logo.WidthPt},
	} {
		if f.value <= 0 {
			return fmt.Errorf("%s is %v, and render scales the mark by it", f.key, f.value)
		}
	}
	for _, f := range []struct {
		key   string
		value string
	}{
		{"logo.anchor.in", c.Logo.Anchor.In},
		{"logo.anchor.relative_h", c.Logo.Anchor.RelativeH},
		{"logo.anchor.relative_v", c.Logo.Anchor.RelativeV},
		{"logo.anchor.wrap", c.Logo.Anchor.Wrap},
	} {
		if strings.TrimSpace(f.value) == "" {
			return fmt.Errorf("%s is empty, and render reads it by name", f.key)
		}
	}
	return nil
}

// validateLogoPNG holds bytes in logo.base64 to the same mime, byte, side and
// complete-decode checks as a logo read from Drive or cache.
func (c *Config) validateLogoPNG() error {
	raw, err := c.LogoBytes()
	if err != nil {
		return err
	}
	if err := asset.Validate(raw, asset.Want{Mime: c.Logo.Mime}); err != nil {
		return fmt.Errorf("logo: %w", err)
	}
	return nil
}
