package house

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if cfg.Page.WidthPt != 595.28 {
		t.Errorf("page.width_pt = %v, want 595.28", cfg.Page.WidthPt)
	}
	if cfg.Page.HeightPt != 841.89 {
		t.Errorf("page.height_pt = %v, want 841.89", cfg.Page.HeightPt)
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
	if !s.KeepWithNext {
		t.Error("heading_1.keep_with_next = false, want true")
	}
}

func TestTheNineStylesAreThere(t *testing.T) {
	cfg := load(t)
	want := []string{
		"normal", "heading_1", "heading_2", "heading_3", "heading_4",
		"heading_5", "heading_6", "title", "subtitle",
	}
	if len(cfg.Styles) != 9 {
		t.Errorf("styles has %d entries, want 9", len(cfg.Styles))
	}
	for _, name := range want {
		if _, ok := cfg.Style(name); !ok {
			t.Errorf("styles.%s is missing", name)
		}
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

func TestFrontMatterIsThirteenBlocksInOrder(t *testing.T) {
	cfg := load(t)
	type want struct {
		kind string
		ref  string
	}
	order := []want{
		{"cover", ""},
		{"label", "version_control"},
		{"table", "version_control"},
		{"blank", ""},
		{"table", "revision_history"},
		{"blank", ""},
		{"legend", ""},
		{"blank", ""},
		{"label", "document_classification"},
		{"table", "document_classification"},
		{"blank", ""},
		{"label", "contents"},
		{"toc", ""},
	}
	if len(cfg.FrontMatter) != 13 {
		t.Fatalf("front_matter has %d blocks, want 13", len(cfg.FrontMatter))
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

func TestTheLogoDecodesAsAPNGOfTheStatedSize(t *testing.T) {
	cfg := load(t)
	if cfg.Logo.Mime != "image/png" {
		t.Errorf("logo.mime = %q, want image/png", cfg.Logo.Mime)
	}
	if len(cfg.Logo.Px) != 2 || cfg.Logo.Px[0] != 195 || cfg.Logo.Px[1] != 97 {
		t.Errorf("logo.px = %v, want [195 97]", cfg.Logo.Px)
	}
	if cfg.Logo.WidthPt != 146.25 || cfg.Logo.HeightPt != 72.75 {
		t.Errorf("logo size = %v by %v pt, want 146.25 by 72.75", cfg.Logo.WidthPt, cfg.Logo.HeightPt)
	}
	raw, err := cfg.LogoBytes()
	if err != nil {
		t.Fatalf("LogoBytes: %v", err)
	}
	img, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("the logo is not a PNG: %v", err)
	}
	if img.Width != 195 || img.Height != 97 {
		t.Errorf("logo is %dx%d px, want 195x97", img.Width, img.Height)
	}
	if cfg.Logo.Anchor.OffsetXPt != 404.25 || cfg.Logo.Anchor.OffsetYPt != -5.15 {
		t.Errorf("logo anchor offsets = %v, %v, want 404.25, -5.15",
			cfg.Logo.Anchor.OffsetXPt, cfg.Logo.Anchor.OffsetYPt)
	}
}

func TestLoadFileReadsACopyOfTheEmbeddedFile(t *testing.T) {
	path := write(t, embedded)
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if cfg.Page.WidthPt != 595.28 {
		t.Errorf("page.width_pt = %v, want 595.28", cfg.Page.WidthPt)
	}
	if len(cfg.FrontMatter) != 13 {
		t.Errorf("front_matter has %d blocks, want 13", len(cfg.FrontMatter))
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

func TestALogoThatIsNotAPNGIsRefused(t *testing.T) {
	bad := base64.StdEncoding.EncodeToString([]byte("this is not a png"))
	cut := bytes.Index(embedded, []byte("  base64: |-\n"))
	if cut < 0 {
		t.Fatal("the logo base64 block was not found to replace")
	}
	src := append(append([]byte{}, embedded[:cut]...), []byte("  base64: "+bad+"\n")...)
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a logo that is not a PNG was accepted")
	}
	if !strings.Contains(err.Error(), "logo") {
		t.Errorf("error %q does not name the logo", err)
	}
}

func TestNothingHereReachesTheNetwork(t *testing.T) {
	// The render path is offline by construction. This states it where a
	// person adding an import would see it fail.
	src, err := os.ReadFile("house.go")
	if err != nil {
		t.Fatalf("read house.go: %v", err)
	}
	for _, banned := range []string{"net/http", "gdoc/internal/gapi", "os/exec"} {
		if bytes.Contains(src, []byte(banned)) {
			t.Errorf("house.go imports %s, and the generator reaches no network", banned)
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
