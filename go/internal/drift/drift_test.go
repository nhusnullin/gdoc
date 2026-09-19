package drift

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/body"
	"gdoc/internal/cover"
	"gdoc/internal/house"
	"gdoc/internal/render"
)

// masterPath is the fixture the gate measures against.
//
// It is a copy of the Altery group policy template, and it is a copy on
// purpose: nothing under go/ reads a template at runtime, so this fixture
// carries the master into the test rather than a path outside the module. The
// master is provenance, which is what DECISIONS.md left it as on 2026-08-29,
// and this is the one thing that still reads it.
const masterPath = "testdata/master.docx"

// notePath is the note the gate builds. It is the body package's copy rather
// than a second one here: one note, so a change to it is measured by the
// goldens and by this gate at the same time and the two cannot drift apart.
const notePath = "../body/testdata/docs/03-policy.md"

// TestFromDocxReadsTheMaster states the house style as literals, read out of
// the Word master. Every number here is written out rather than taken from
// house.yaml: a test that reads the constant it checks is a mirror, and what
// this one has to say is what the house style is.
func TestFromDocxReadsTheMaster(t *testing.T) {
	d := openMaster(t)

	if got := d.Page("width"); got != 595.3 {
		t.Errorf("the master's page is %v points wide, and A4 is 595.3", got)
	}
	if got := d.Page("height"); got != 841.9 {
		t.Errorf("the master's page is %v points tall, and A4 is 841.9", got)
	}
	for _, c := range []struct {
		name string
		want float64
	}{
		{"marginTop", 62.35}, {"marginBottom", 51}, {"marginLeft", 51.05},
		{"marginRight", 51.05}, {"marginHeader", 14.15}, {"marginFooter", 0},
	} {
		if got := d.Page(c.name); got != c.want {
			t.Errorf("the master's %s is %v points, and the house style is %v", c.name, got, c.want)
		}
	}
	if got := d.FirstPageHeaderFooter(); got != true {
		t.Errorf("the master's first page has its own header and footer, and titlePg read %v", got)
	}
	if got := d.PageNumberStart(); got != 1.0 {
		t.Errorf("the master's page numbers start at %v, and the house style starts them at 1", got)
	}
	// The house style sets both header and footer distances on the section, so
	// the master states them rather than leaving Word its defaults. Docs reports
	// the same fact as useCustomHeaderFooterMargins, which is why the item exists.
	if got := d.CustomHeaderFooterMargins(); got != true {
		t.Errorf("the master's section states no header and footer distances, and pgMar read %v", got)
	}
	for _, c := range []struct{ kind, which string }{
		{"header", "first"}, {"header", "default"}, {"footer", "first"}, {"footer", "default"},
	} {
		if got := d.HasReference(c.kind, c.which); got != true {
			t.Errorf("the master names no %s %s, and the house style has all four", c.which, c.kind)
		}
	}

	// The nine named styles, stated as the house values.
	h1 := d.Style("Heading1")
	if h1.FontSize != 16.0 {
		t.Errorf("Heading 1 is %v points in the master, and the house style is 16", h1.FontSize)
	}
	if !h1.Bold {
		t.Error("Heading 1 is bold in the house style, and the master's is not")
	}
	if h1.SpaceAbove != 12.0 {
		t.Errorf("Heading 1 carries %v points above it, and the house style is 12", h1.SpaceAbove)
	}
	if !h1.KeepWithNext {
		t.Error("Heading 1 keeps with the paragraph after it in the house style")
	}
	if got := d.Style("Heading3").Colour; got != "#549F99" {
		t.Errorf("Heading 3 is %v in the master, and the house teal is #549F99", got)
	}
	if got := d.Style("Normal"); got.FontSize != 11.0 || got.Font != "Calibri" || got.LineSpacing != 115.0 {
		t.Errorf("the master's body text is %v point %v at %v%% line spacing, and the house style is 11 point Calibri at 115%%",
			got.FontSize, got.Font, got.LineSpacing)
	}
	if got := d.Style("Title"); got.FontSize != 16.0 || got.Alignment != "CENTER" || got.Font != "Arial" {
		t.Errorf("the master's Title is %v point %v aligned %v, and the house style is 16 point Arial centred",
			got.FontSize, got.Font, got.Alignment)
	}
	if got := d.Style("Subtitle"); got.FontSize != 24.0 || !got.Italic || got.Colour != "#666666" {
		t.Errorf("the master's Subtitle is %v point italic %v in %v, and the house style is 24 point italic #666666",
			got.FontSize, got.Italic, got.Colour)
	}

	// The logo, positioned rather than inline, which is the whole reason the
	// document is born from a docx.
	logo := d.Logo()
	if !logo.Present || logo.Mode != "POSITIONED" {
		t.Fatalf("the master's logo reads present %v mode %q, and it is a positioned object", logo.Present, logo.Mode)
	}
	if logo.Width != 146.25 || logo.Height != 72.75 {
		t.Errorf("the master's logo is %v by %v points, and the house style is 146.25 by 72.75", logo.Width, logo.Height)
	}
	if left, ok := logo.Left.(float64); !ok || left < 404 || left > 405 {
		t.Errorf("the master's logo sits %v points from the left, and the house style puts it just past 404", logo.Left)
	}

	// The contents list is a live Word field, which is what survives Google's
	// import as a refreshable list.
	if got := d.TOCFields(); len(got) != 1 {
		t.Fatalf("the master carries %d contents fields, and it carries one", len(got))
	}
	if got := d.TOCInstr(); !strings.Contains(got.(string), "Heading 1,1,Heading 2,2,Heading 3,3") {
		t.Errorf("the master's contents field reads %q, and it lists heading levels one to three", got)
	}

	// The three front matter tables, cell by cell.
	tables := d.Tables()
	if len(tables) != 3 {
		t.Fatalf("the master carries %d tables, and the front matter is three", len(tables))
	}
	if tables[0].Rows != 5 || tables[0].Columns != 2 {
		t.Errorf("version control is %dx%d in the master, and it is 5x2", tables[0].Rows, tables[0].Columns)
	}
	if tables[1].Rows != 3 || tables[1].Columns != 7 {
		t.Errorf("revision history is %dx%d in the master, and it is 3x7", tables[1].Rows, tables[1].Columns)
	}
	if tables[2].Rows != 5 || tables[2].Columns != 2 {
		t.Errorf("document classification is %dx%d in the master, and it is 5x2", tables[2].Rows, tables[2].Columns)
	}
	if got := tables[0].Fills[0]; got != "#F5D1AE" {
		t.Errorf("the first cell of version control is filled %v in the master, and the house fill is #F5D1AE", got)
	}
	if got := tables[0].Texts[0]; got != "Document Owner" {
		t.Errorf("the first cell of version control reads %q in the master", got)
	}
	if got := tables[1].Borders[0]; !strings.HasPrefix(got, "0.500pt") {
		t.Errorf("revision history's cell border is %q in the master, and the house border is half a point", got)
	}
	if got := tables[0].ColWidths; len(got) != 2 || got[0] != 150 {
		t.Errorf("version control's columns are %v points in the master, and the first is 150", got)
	}

	// The headers and footers, and the running head's own colour.
	if got := d.Segment("header", "default"); !got.Exists || got.Paragraphs != 3 {
		t.Errorf("the master's running head is %d paragraphs, and the house style is 3", got.Paragraphs)
	}
	if got := d.Segment("header", "default").FirstRun.Colour; got != "#FF771C" {
		t.Errorf("the master's running head is %v, and the house orange is #FF771C", got)
	}
	if got := d.Segment("footer", "default").Text; !strings.Contains(got, "For internal use only") {
		t.Errorf("the master's footer reads %q", got)
	}
	if got := d.Segment("footer", "default").Text; !strings.Contains(got, "<PAGE>") {
		t.Errorf("the master's footer carries a page number field, and it reads %q", got)
	}
}

// TestAFieldPrintsItsInstructionAndNotItsCachedValue. A footer saved while page
// one was on screen carries a "1" that nobody typed, and the same footer out of
// another writer does not. The instruction is the fact.
func TestAFieldPrintsItsInstructionAndNotItsCachedValue(t *testing.T) {
	built := buildNote(t)
	master := openMaster(t)
	if got := built.Segment("footer", "default").Text; strings.Contains(got, "<PAGE>1") {
		t.Errorf("the built footer reads %q, and the cached page number is not text the writer put there", got)
	}
	if got, want := built.Segment("footer", "default").Text, master.Segment("footer", "default").Text; got != want {
		t.Errorf("the two footers read %q and %q, and they are the same footer", got, want)
	}
}

// TestTheOfflineGate is the measurement DECISIONS.md left as a test: build a
// note from the embedded house style, open the Word master, read the same list
// out of both, and fail on any row that moved and is not already explained.
//
// It runs the pins over the explained rows too. Known says why a row differs,
// and knownOffline in pins_test.go says what the two sides read when that
// reason was written, so a row named in Known cannot move again in silence.
func TestTheOfflineGate(t *testing.T) {
	rows := Compare(FromDocx(buildNote(t)), FromDocx(openMaster(t)))
	t.Log(Summary(rows))
	unexplained := Unexplained(rows)
	if len(unexplained) > 0 {
		t.Errorf("%d rows moved and are not in Known:\n%s\n\nthe whole table:\n%s",
			len(unexplained), Table(unexplained), Table(rows))
	}
	for _, sentence := range unpinned(rows) {
		t.Error(sentence)
	}
}

// TestEveryKnownDifferenceStillDiffers is the other direction on Known, the
// one TestEveryKnownDifferenceNamesARealItem does not cover.
//
// That test says every entry names a row the list produces. This one says
// every entry still names a row that really disagrees. An entry left behind
// after its row went IDENTICAL exempts that row for ever, so the next real
// regression on it passes the gate in silence. Eight of the entries are of the
// "the two documents hold different words" kind, which is exactly the kind
// that can converge by accident. It is the rule the boundary test's
// TestAllowedModulesAreReallyRequired holds over the other allowlist here: an
// allowlist naming something that is not there stops describing the tree.
func TestEveryKnownDifferenceStillDiffers(t *testing.T) {
	rows := Compare(FromDocx(buildNote(t)), FromDocx(openMaster(t)))
	verdicts := map[string]Verdict{}
	for _, r := range rows {
		verdicts[r.Item] = r.Verdict
	}
	for name := range Known {
		v, ok := verdicts[name]
		if !ok {
			// TestEveryKnownDifferenceNamesARealItem says this properly.
			continue
		}
		if v != Different && v != Missing {
			t.Errorf("known difference %q now reads %s, so the entry protects a row that no longer moves and has to come out of Known", name, v)
		}
	}
}

// unstated is every row neither document says anything about, written out.
//
// A row that is nil on both sides still compares: it goes MISSING the moment
// one side starts stating a value. But a row that is nil on both sides measures
// nothing today, so the set of them is pinned here as literals rather than
// left to grow. A value quietly dropped out of house.yaml would otherwise turn
// its row into one of these and the gate would keep passing.
//
// Each one is a value both documents leave to the reader's default: a style
// that states no justification is left aligned, and a style that states no
// indent starts at the margin.
var unstated = []string{
	"NORMAL_TEXT alignment", "NORMAL_TEXT indentStart",
	"HEADING_1 alignment", "HEADING_2 alignment", "HEADING_3 alignment",
	"HEADING_4 alignment", "HEADING_5 alignment", "HEADING_6 alignment",
	"TITLE indentStart", "SUBTITLE alignment", "SUBTITLE indentStart",
}

// TestTheGateReadsTheWholeList is the other direction. A gate that compared
// nothing would pass, so this states how much of the list really carries a
// measurement, and that no two rows share a name.
func TestTheGateReadsTheWholeList(t *testing.T) {
	built, master := FromDocx(buildNote(t)), FromDocx(openMaster(t))
	if len(Items) != len(built) || len(built) != len(master) {
		t.Fatalf("%d items read as %d and %d values", len(Items), len(built), len(master))
	}

	want := map[string]bool{}
	for _, name := range unstated {
		want[name] = true
	}
	for i, v := range built {
		empty := v.V == nil && master[i].V == nil
		if empty && !want[v.Name] {
			t.Errorf("item %q reads nothing in either document, and it is not one of the values both are known to leave unstated", v.Name)
		}
		if !empty && want[v.Name] {
			t.Errorf("item %q now reads a value, so it is no longer one of the unstated ones", v.Name)
		}
		delete(want, v.Name)
	}

	seen := map[string]bool{}
	for _, it := range Items {
		if seen[it.Name] {
			t.Errorf("two items are both called %q, so one of them can never be found in the report", it.Name)
		}
		seen[it.Name] = true
	}
}

// openMaster opens the Word master.
func openMaster(t *testing.T) *Docx {
	t.Helper()
	d, err := OpenDocxFile(masterPath)
	if err != nil {
		t.Fatalf("the master fixture could not be opened: %v", err)
	}
	return d
}

// buildNote runs the whole generator over the note and opens what it wrote.
// Nothing is written to disk: the gate measures the package the builder
// produced, which is the same bytes `gdoc build` would put in a file.
func buildNote(t *testing.T) *Docx {
	t.Helper()
	source, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("the note could not be read: %v", err)
	}
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style did not load: %v", err)
	}
	fields, markdown, err := cover.Read(source)
	if err != nil {
		t.Fatalf("the note's front matter did not read: %v", err)
	}
	walked, err := body.Render(cfg, markdown, filepath.Dir(notePath), fields.HeadingNumbering)
	if err != nil {
		t.Fatalf("the note's markdown did not render: %v", err)
	}
	pkg, err := render.Build(cfg, fields, walked.Blocks, walked.Media, walked.NumberedLists)
	if err != nil {
		t.Fatalf("the document did not build: %v", err)
	}
	var buf bytes.Buffer
	if err := pkg.Write(&buf); err != nil {
		t.Fatalf("the document did not zip: %v", err)
	}
	d, err := OpenDocx(buf.Bytes())
	if err != nil {
		t.Fatalf("what the builder wrote did not open as a docx: %v", err)
	}
	return d
}
