package render

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"gdoc/internal/house"
	"github.com/beevik/etree"
)

// Fields is what the note's own front matter gives the shell: the words that
// replace the template's placeholders on the cover and in the running head.
//
// Task 5 of M5 gives this a home of its own in internal/cover, which reads and
// validates a note's keys. Until then this is the shape render needs, and
// nothing here reads a file.
type Fields struct {
	Title       string
	AltTitle    string
	RunningHead string
	Version     string
	Date        string
	Revisions   []Revision
}

// Revision is one row of a note's revision history. The house file spells the
// revision table out cell by cell today, so the shell carries a note's own
// rows no further than this; placing them is the cover milestone's.
type Revision struct {
	Version      string
	Date         string
	Author       string
	ApprovedBy   string
	ApprovalDate string
	Section      string
	Change       string
}

// Media is one relationship the body needs: a picture, or the destination of
// a link. RelID is the id the body's own drawing or w:hyperlink names.
//
// A picture carries Name, the file under word/media/, and Data. A link carries
// Target, the address, and nothing else. The two live in one list because the
// body hands out one run of relationship ids across both, and two lists could
// hand the same id to a picture and a link.
type Media struct {
	RelID  string
	Name   string
	Data   []byte
	Target string
}

// IsLink says whether this is a link's relationship rather than a picture's.
func (m Media) IsLink() bool { return m.Target != "" }

// Part is one entry in the package, held as bytes.
type Part struct {
	Name string
	Data []byte
}

// Package is every part, in the order the zip lists them.
type Package struct {
	Parts []Part
}

// Get returns one part's bytes.
func (p *Package) Get(name string) ([]byte, bool) {
	for _, part := range p.Parts {
		if part.Name == name {
			return part.Data, true
		}
	}
	return nil, false
}

// Names lists every part, in order.
func (p *Package) Names() []string {
	out := make([]string, 0, len(p.Parts))
	for _, part := range p.Parts {
		out = append(out, part.Name)
	}
	return out
}

// Write zips the parts in the order they are held. [Content_Types].xml comes
// first because that is where a reader looks for what the rest of the archive
// is.
//
// It is Write rather than WriteTo because vet reads that name as io.WriterTo,
// which hands back a byte count nothing here has to offer.
func (p *Package) Write(w io.Writer) error {
	z := zip.NewWriter(w)
	for _, part := range p.Parts {
		entry, err := z.CreateHeader(&zip.FileHeader{Name: part.Name, Method: zip.Deflate})
		if err != nil {
			return fmt.Errorf("write %s: %w", part.Name, err)
		}
		if _, err := entry.Write(part.Data); err != nil {
			return fmt.Errorf("write %s: %w", part.Name, err)
		}
	}
	return z.Close()
}

// The relationships the shell owns. A body image takes an id after these, and
// one that collides is refused rather than quietly overwriting the styles or a
// header.
var shellRels = []struct{ id, kind, target string }{
	{"rId1", "styles", "styles.xml"},
	{"rId2", "numbering", "numbering.xml"},
	{"rId3", "settings", "settings.xml"},
	{"rId4", "header", "header1.xml"},
	{"rId5", "header", "header2.xml"},
	{"rId6", "footer", "footer1.xml"},
	{"rId7", "footer", "footer2.xml"},
}

// FirstMediaRelID is the first relationship id a body image may take. The
// seven below it are the shell's own.
const FirstMediaRelID = 8

// Build renders one document: the house style, the note's fields, the body the
// markdown walker produced, and the images it read. Nothing here reaches the
// network, and nothing reads the Word master.
func Build(cfg *house.Config, f Fields, body []*etree.Element, media []Media) (*Package, error) {
	if cfg == nil {
		return nil, fmt.Errorf("render: no house style")
	}
	b := &builder{cfg: cfg, fields: f}
	if err := checkMedia(media); err != nil {
		return nil, err
	}

	logo, err := cfg.LogoBytes()
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	pkg := &Package{Parts: []Part{
		{"[Content_Types].xml", b.contentTypes(media)},
		{"_rels/.rels", b.rootRels()},
		{"word/_rels/document.xml.rels", b.documentRels(media)},
		{"word/_rels/header2.xml.rels", b.headerRels()},
		{"word/document.xml", b.documentPart(body)},
		{"word/styles.xml", b.stylesPart()},
		{"word/numbering.xml", b.numberingPart()},
		{"word/settings.xml", b.settingsPart()},
		{"word/header1.xml", b.headerPart(cfg.Header.Default, false, "word/header1.xml")},
		{"word/header2.xml", b.headerPart(cfg.Header.First, true, "word/header2.xml")},
		{"word/footer1.xml", b.footerPart(cfg.Footer.Default, "word/footer1.xml")},
		{"word/footer2.xml", b.footerPart(cfg.Footer.First, "word/footer2.xml")},
		{"word/media/logo.png", logo},
	}}
	for _, m := range media {
		if m.IsLink() {
			continue
		}
		pkg.Parts = append(pkg.Parts, Part{"word/media/" + m.Name, m.Data})
	}
	if b.err != nil {
		return nil, fmt.Errorf("render: %w", b.err)
	}
	return pkg, nil
}

// checkMedia refuses an image whose relationship the shell already uses, and
// two images sharing one. Either would give a caller a document whose picture
// is somebody else's part.
func checkMedia(media []Media) error {
	seen := map[string]bool{}
	for _, m := range media {
		if m.RelID == "" {
			return fmt.Errorf("render: a relationship needs an id")
		}
		if m.IsLink() {
			if m.Name != "" || len(m.Data) > 0 {
				return fmt.Errorf("render: relationship %s is a link and carries a file as well", m.RelID)
			}
		} else {
			if m.Name == "" {
				return fmt.Errorf("render: an image needs a relationship id and a file name")
			}
			if strings.ContainsAny(m.Name, "/\\") {
				return fmt.Errorf("render: image name %q is a path, and images are one file under word/media", m.Name)
			}
		}
		for _, rel := range shellRels {
			if rel.id == m.RelID {
				return fmt.Errorf("render: image relationship %s is the shell's own %s, and an image starts at rId%d",
					m.RelID, rel.target, FirstMediaRelID)
			}
		}
		if seen[m.RelID] {
			return fmt.Errorf("render: two images share the relationship id %s", m.RelID)
		}
		seen[m.RelID] = true
	}
	return nil
}

// documentPart writes word/document.xml: the front matter the config orders,
// the body the walker produced, then the section properties.
func (b *builder) documentPart(body []*etree.Element) []byte {
	doc, root := newPart("w:document")
	wBody := sub(root, "w:body")
	for _, block := range b.frontMatter() {
		wBody.AddChild(block)
	}
	for _, block := range body {
		wBody.AddChild(block)
	}
	b.sectionProperties(wBody)
	return b.serialise(doc, "word/document.xml")
}

// sectionProperties is the page: the four header and footer references, the
// size and margins in twips, where the page numbers start, and a different
// first page when the house says so.
func (b *builder) sectionProperties(body *etree.Element) {
	p := b.cfg.Page
	sect := sub(body, "w:sectPr")
	sub(sect, "w:headerReference", "r:id", "rId4", "w:type", "default")
	sub(sect, "w:headerReference", "r:id", "rId5", "w:type", "first")
	sub(sect, "w:footerReference", "r:id", "rId6", "w:type", "default")
	sub(sect, "w:footerReference", "r:id", "rId7", "w:type", "first")
	sub(sect, "w:pgSz",
		"w:w", twips(p.WidthPt), "w:h", twips(p.HeightPt), "w:orient", "portrait")
	sub(sect, "w:pgMar",
		"w:top", twips(p.MarginTopPt),
		"w:bottom", twips(p.MarginBottomPt),
		"w:left", twips(p.MarginLeftPt),
		"w:right", twips(p.MarginRightPt),
		"w:header", twips(p.HeaderMarginPt),
		"w:footer", twips(p.FooterMarginPt))
	sub(sect, "w:pgNumType", "w:start", strconv.Itoa(p.PageNumberStart))
	if p.DifferentFirstPage {
		sub(sect, "w:titlePg", "w:val", "1")
	}
}

// contentTypes declares what each part is. A body image adds its own
// extension, so a JPEG the note carries is a picture rather than a part Word
// cannot read.
func (b *builder) contentTypes(media []Media) []byte {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", xmlProcInst)
	root := doc.CreateElement("Types")
	root.CreateAttr("xmlns", "http://schemas.openxmlformats.org/package/2006/content-types")

	defaults := map[string]string{
		"rels": "application/vnd.openxmlformats-package.relationships+xml",
		"xml":  "application/xml",
		"png":  "image/png",
	}
	for _, m := range media {
		if m.IsLink() {
			continue
		}
		ext := strings.ToLower(strings.TrimPrefix(path.Ext(m.Name), "."))
		if ext == "" {
			b.fail("image %s has no extension, so nothing can say what it is", m.Name)
			continue
		}
		if _, ok := defaults[ext]; !ok {
			defaults[ext] = "image/" + ext
		}
	}
	exts := make([]string, 0, len(defaults))
	for ext := range defaults {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	for _, ext := range exts {
		sub(root, "Default", "Extension", ext, "ContentType", defaults[ext])
	}

	const wml = "application/vnd.openxmlformats-officedocument.wordprocessingml."
	for _, o := range [][2]string{
		{"/word/document.xml", wml + "document.main+xml"},
		{"/word/styles.xml", wml + "styles+xml"},
		{"/word/numbering.xml", wml + "numbering+xml"},
		{"/word/settings.xml", wml + "settings+xml"},
		{"/word/header1.xml", wml + "header+xml"},
		{"/word/header2.xml", wml + "header+xml"},
		{"/word/footer1.xml", wml + "footer+xml"},
		{"/word/footer2.xml", wml + "footer+xml"},
	} {
		sub(root, "Override", "PartName", o[0], "ContentType", o[1])
	}
	return b.serialise(doc, "[Content_Types].xml")
}

// relsRoot starts a relationships part.
func relsRoot() (*etree.Document, *etree.Element) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", xmlProcInst)
	root := doc.CreateElement("Relationships")
	root.CreateAttr("xmlns", "http://schemas.openxmlformats.org/package/2006/relationships")
	return doc, root
}

const relType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/"

// rootRels points the package at its one document.
func (b *builder) rootRels() []byte {
	doc, root := relsRoot()
	sub(root, "Relationship",
		"Id", "rId1",
		"Type", relType+"officeDocument",
		"Target", "word/document.xml")
	return b.serialise(doc, "_rels/.rels")
}

// documentRels is the shell's seven parts, then one relationship per image.
func (b *builder) documentRels(media []Media) []byte {
	doc, root := relsRoot()
	for _, rel := range shellRels {
		sub(root, "Relationship", "Id", rel.id, "Type", relType+rel.kind, "Target", rel.target)
	}
	for _, m := range media {
		if m.IsLink() {
			// A link's target is somebody else's address, so the relationship
			// says so: Word refuses to follow an external target held as an
			// internal one.
			sub(root, "Relationship", "Id", m.RelID, "Type", relType+"hyperlink",
				"Target", m.Target, "TargetMode", "External")
			continue
		}
		sub(root, "Relationship", "Id", m.RelID, "Type", relType+"image", "Target", "media/"+m.Name)
	}
	return b.serialise(doc, "word/_rels/document.xml.rels")
}

// headerRels is the first page header's one relationship, the logo.
func (b *builder) headerRels() []byte {
	doc, root := relsRoot()
	sub(root, "Relationship", "Id", logoRelID, "Type", relType+"image", "Target", "media/logo.png")
	return b.serialise(doc, "word/_rels/header2.xml.rels")
}

// placeholder returns the note's own value for a named cover field, and
// whether the note filled it in. A name the config uses and this package does
// not know is refused: a placeholder silently left as the template's words
// would publish a document reading "(Name of) Framework/Policy".
func (b *builder) placeholder(name string) (string, bool) {
	if name == "" {
		return "", false
	}
	var value string
	switch name {
	case "title":
		value = b.fields.Title
	case "alt_title":
		value = b.fields.AltTitle
	case "running_head":
		value = b.fields.RunningHead
	case "version":
		value = b.fields.Version
	case "date":
		value = b.fields.Date
	default:
		b.fail("placeholder %q is not a cover field", name)
		return "", false
	}
	return value, value != ""
}

// text is one run's words: the note's own where it filled a placeholder in,
// else the config's.
func (b *builder) text(configured, placeholder string) string {
	if value, filled := b.placeholder(placeholder); filled {
		return value
	}
	return configured
}

// trimFloat writes a number the way the config states it, with no trailing
// zeroes, because it lands in a VML style string rather than in an attribute
// OOXML counts.
func trimFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
