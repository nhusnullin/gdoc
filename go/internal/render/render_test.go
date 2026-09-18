package render

import (
	"archive/zip"
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/cover"
	"gdoc/internal/house"
	"github.com/beevik/etree"
)

// Every house value in this file is a literal on purpose. A test that reads
// the constant it checks is a mirror: change the config and the assertion
// follows it and still passes. See CLAUDE.md, "A house-style test must never
// read the constant it tests".

var update = flag.Bool("update", false, "rewrite the golden parts under testdata")

// fields is a note the shell can be built from. The running head is not stated
// here because internal/cover computes it from the title, which is the point of
// it being a method there.
func fields() cover.Fields {
	return cover.Fields{
		Title:   "Supplier Register Policy",
		Version: "1.0",
		Date:    "8 September 2026",
	}
}

func config(t *testing.T) *house.Config {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("house.Load: %v", err)
	}
	return cfg
}

func build(t *testing.T) *Package {
	t.Helper()
	pkg, err := Build(config(t), fields(), nil, nil, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return pkg
}

// part returns one part's bytes, failing when it is not there.
func part(t *testing.T, pkg *Package, name string) []byte {
	t.Helper()
	data, ok := pkg.Get(name)
	if !ok {
		t.Fatalf("the package holds no %s", name)
	}
	return data
}

// parse reads one part with etree, which is what Word and Google do.
func parse(t *testing.T, data []byte) *etree.Document {
	t.Helper()
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		t.Fatalf("the part does not parse: %v", err)
	}
	return doc
}

func TestBuildWritesTheTwelvePartsAndTheLogo(t *testing.T) {
	pkg := build(t)

	want := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"word/_rels/document.xml.rels",
		"word/_rels/header2.xml.rels",
		"word/document.xml",
		"word/styles.xml",
		"word/numbering.xml",
		"word/settings.xml",
		"word/header1.xml",
		"word/header2.xml",
		"word/footer1.xml",
		"word/footer2.xml",
		"word/media/logo.png",
	}
	got := pkg.Names()
	if len(got) != len(want) {
		t.Fatalf("the package holds %d parts, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("part %d is %q, want %q", i, got[i], want[i])
		}
	}
}

func TestEveryXMLPartParses(t *testing.T) {
	pkg := build(t)
	for _, p := range pkg.Parts {
		if !strings.HasSuffix(p.Name, ".xml") && !strings.HasSuffix(p.Name, ".rels") {
			continue
		}
		doc := etree.NewDocument()
		if err := doc.ReadFromBytes(p.Data); err != nil {
			t.Errorf("%s does not parse: %v", p.Name, err)
			continue
		}
		if !bytes.HasPrefix(p.Data, []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)) {
			t.Errorf("%s does not open with the XML declaration", p.Name)
		}
	}
}

func TestTheLogoPartIsThePNGTheConfigCarries(t *testing.T) {
	pkg := build(t)
	logo := part(t, pkg, "word/media/logo.png")
	if !bytes.HasPrefix(logo, []byte("\x89PNG\r\n\x1a\n")) {
		t.Error("word/media/logo.png does not open with the PNG signature")
	}
}

func TestTheZipListsContentTypesFirst(t *testing.T) {
	pkg := build(t)
	var buf bytes.Buffer
	if err := pkg.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("the written file is not a zip: %v", err)
	}
	if len(r.File) != 13 {
		t.Fatalf("the zip holds %d entries, want 13", len(r.File))
	}
	if r.File[0].Name != "[Content_Types].xml" {
		t.Errorf("the first entry is %q, want [Content_Types].xml", r.File[0].Name)
	}
}

func TestSectionPropertiesAreA4InTwipsWithTheFourReferences(t *testing.T) {
	pkg := build(t)
	doc := parse(t, part(t, pkg, "word/document.xml"))
	sect := doc.FindElement("//w:body/w:sectPr")
	if sect == nil {
		t.Fatal("word/document.xml carries no sectPr")
	}
	pgSz := sect.SelectElement("w:pgSz")
	if pgSz == nil {
		t.Fatal("the sectPr carries no pgSz")
	}
	if got := pgSz.SelectAttrValue("w:w", ""); got != "11906" {
		t.Errorf("pgSz w:w = %q, want 11906", got)
	}
	if got := pgSz.SelectAttrValue("w:h", ""); got != "16838" {
		t.Errorf("pgSz w:h = %q, want 16838", got)
	}
	mar := sect.SelectElement("w:pgMar")
	if mar == nil {
		t.Fatal("the sectPr carries no pgMar")
	}
	for _, c := range []struct{ attr, want string }{
		{"w:top", "1247"},
		{"w:bottom", "1020"},
		{"w:left", "1021"},
		{"w:right", "1021"},
		{"w:header", "283"},
		{"w:footer", "0"},
	} {
		if got := mar.SelectAttrValue(c.attr, ""); got != c.want {
			t.Errorf("pgMar %s = %q, want %s", c.attr, got, c.want)
		}
	}
	if sect.SelectElement("w:titlePg") == nil {
		t.Error("the sectPr carries no titlePg, and the house has a different first page")
	}
	refs := map[string]string{}
	for _, e := range sect.ChildElements() {
		switch e.Tag {
		case "headerReference", "footerReference":
			refs[e.FullTag()+":"+e.SelectAttrValue("w:type", "")] = e.SelectAttrValue("r:id", "")
		}
	}
	want := map[string]string{
		"w:headerReference:default": "rId4",
		"w:headerReference:first":   "rId5",
		"w:footerReference:default": "rId6",
		"w:footerReference:first":   "rId7",
	}
	for k, v := range want {
		if refs[k] != v {
			t.Errorf("%s = %q, want %q", k, refs[k], v)
		}
	}
}

func TestATitleCarryingXMLCharactersIsEscaped(t *testing.T) {
	f := fields()
	f.Title = "R&D <policy>"
	pkg, err := Build(config(t), f, nil, nil, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	raw := part(t, pkg, "word/document.xml")
	if bytes.Contains(raw, []byte("R&D <policy>")) {
		t.Error("the title reached the XML unescaped")
	}
	if !bytes.Contains(raw, []byte("R&amp;D &lt;policy&gt;")) {
		t.Errorf("the escaped title is not in the part")
	}
	doc := parse(t, raw)
	found := false
	for _, e := range doc.FindElements("//w:t") {
		if e.Text() == "R&D <policy>" {
			found = true
		}
	}
	if !found {
		t.Error("the title does not read back as the words it was given")
	}
}

func TestTheBodyElementsSitBetweenTheFrontMatterAndTheSectionProperties(t *testing.T) {
	p := etree.NewElement("w:p")
	p.CreateElement("w:r").CreateElement("w:t").SetText("the body")
	pkg, err := Build(config(t), fields(), []*etree.Element{p}, nil, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	doc := parse(t, part(t, pkg, "word/document.xml"))
	kids := doc.FindElement("//w:body").ChildElements()
	last := kids[len(kids)-1]
	if last.FullTag() != "w:sectPr" {
		t.Fatalf("the body ends with %s, want w:sectPr", last.FullTag())
	}
	before := kids[len(kids)-2]
	if got := before.FindElement("w:r/w:t"); got == nil || got.Text() != "the body" {
		t.Error("the body element is not the last block before the sectPr")
	}
}

func TestAnImageIsWrittenWithItsRelationshipAndItsContentType(t *testing.T) {
	media := []Media{{RelID: "rId8", Name: "image1.jpeg", Data: []byte("not really a jpeg")}}
	pkg, err := Build(config(t), fields(), nil, media, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, ok := pkg.Get("word/media/image1.jpeg"); !ok {
		t.Error("the image is not in the package")
	}
	rels := parse(t, part(t, pkg, "word/_rels/document.xml.rels"))
	found := false
	for _, e := range rels.FindElements("//Relationship") {
		if e.SelectAttrValue("Id", "") == "rId8" {
			found = true
			if got := e.SelectAttrValue("Target", ""); got != "media/image1.jpeg" {
				t.Errorf("the image relationship targets %q, want media/image1.jpeg", got)
			}
		}
	}
	if !found {
		t.Error("word/_rels/document.xml.rels names no rId8")
	}
	types := parse(t, part(t, pkg, "[Content_Types].xml"))
	seen := false
	for _, e := range types.FindElements("//Default") {
		if e.SelectAttrValue("Extension", "") == "jpeg" {
			seen = true
		}
	}
	if !seen {
		t.Error("[Content_Types].xml declares no jpeg extension")
	}
}

// A link's relationship is external and carries no part, so nothing lands
// under word/media/ and [Content_Types].xml gains no extension.
func TestALinkIsAnExternalRelationshipWithNoPart(t *testing.T) {
	media := []Media{{RelID: "rId8", Target: "https://example.com/p?a=1&b=2"}}
	pkg, err := Build(config(t), fields(), nil, media, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, name := range pkg.Names() {
		if strings.HasPrefix(name, "word/media/") && name != "word/media/logo.png" {
			t.Errorf("a link put %s in the package", name)
		}
	}
	rels := parse(t, part(t, pkg, "word/_rels/document.xml.rels"))
	found := false
	for _, e := range rels.FindElements("//Relationship") {
		if e.SelectAttrValue("Id", "") != "rId8" {
			continue
		}
		found = true
		if got := e.SelectAttrValue("Target", ""); got != "https://example.com/p?a=1&b=2" {
			t.Errorf("the link relationship targets %q", got)
		}
		if got := e.SelectAttrValue("TargetMode", ""); got != "External" {
			t.Errorf("the link relationship is %q, want External", got)
		}
		if got := e.SelectAttrValue("Type", ""); !strings.HasSuffix(got, "/hyperlink") {
			t.Errorf("the link relationship is typed %q", got)
		}
	}
	if !found {
		t.Error("word/_rels/document.xml.rels names no rId8")
	}
}

// A relationship that is both a link and a file is refused: one of the two
// would be written and the other silently lost.
func TestARelationshipThatIsBothALinkAndAFileIsRefused(t *testing.T) {
	media := []Media{{RelID: "rId8", Name: "image1.png", Data: []byte("x"),
		Target: "https://example.com/"}}
	_, err := Build(config(t), fields(), nil, media, 0)
	if err == nil {
		t.Fatal("Build accepted a relationship that is a link and a file at once")
	}
	if !strings.Contains(err.Error(), "rId8") {
		t.Errorf("the error does not name rId8: %v", err)
	}
}

func TestAnImageIdThatCollidesWithTheShellIsRefused(t *testing.T) {
	media := []Media{{RelID: "rId4", Name: "image1.png", Data: []byte("x")}}
	_, err := Build(config(t), fields(), nil, media, 0)
	if err == nil {
		t.Fatal("Build accepted an image relationship id the shell already uses")
	}
	if !strings.Contains(err.Error(), "rId4") {
		t.Errorf("the error does not name rId4: %v", err)
	}
}

func TestTwoImagesSharingOneIdAreRefused(t *testing.T) {
	media := []Media{
		{RelID: "rId8", Name: "image1.png", Data: []byte("x")},
		{RelID: "rId8", Name: "image2.png", Data: []byte("y")},
	}
	_, err := Build(config(t), fields(), nil, media, 0)
	if err == nil {
		t.Fatal("Build accepted two images with one relationship id")
	}
	if !strings.Contains(err.Error(), "rId8") {
		t.Errorf("the error does not name rId8: %v", err)
	}
}

// golden compares one part against the file under testdata, which is the
// specification of what the house style renders to. -update rewrites them.
func golden(t *testing.T, name string, data []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Equal(want, data) {
		t.Errorf("%s differs from the golden. Read the difference, then rerun with -update if it is wanted.\n got: %s\nwant: %s",
			name, first(data), first(want))
	}
}

func first(b []byte) string {
	if len(b) > 400 {
		return string(b[:400]) + "..."
	}
	return string(b)
}

func TestTheGoldenPartsAreWhatTheHouseStyleRendersTo(t *testing.T) {
	pkg := build(t)
	for _, name := range []string{
		"[Content_Types].xml", "_rels/.rels",
		"word/_rels/document.xml.rels", "word/_rels/header2.xml.rels",
		"word/styles.xml", "word/numbering.xml", "word/settings.xml",
		"word/header1.xml", "word/header2.xml",
		"word/footer1.xml", "word/footer2.xml",
	} {
		file := strings.ReplaceAll(strings.TrimPrefix(name, "word/"), "/", "_")
		golden(t, file, part(t, pkg, name))
	}
}
