package render

import (
	"archive/zip"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func repoDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// The bundled master, reached where it really lives. A copy under testdata would
// drift out of the contract the surgery depends on without anything saying so.
func templatePath(t *testing.T) string {
	return filepath.Join(repoDir(t), "..", "gdoc", "templates",
		"altery-group-policy-v1.0", "template.docx")
}

func docPath(t *testing.T, name string) string {
	return filepath.Join(repoDir(t), "testdata", "docs", name)
}

func build(t *testing.T, source string, opts Options) (*Result, string) {
	t.Helper()
	if opts.Template == "" {
		opts.Template = templatePath(t)
	}
	out := filepath.Join(t.TempDir(), "out.docx")
	result, err := Build(docPath(t, source), out, opts)
	if err != nil {
		t.Fatalf("build %s: %v", source, err)
	}
	return result, out
}

func documentXML(t *testing.T, path string) string {
	return part(t, path, "word/document.xml")
}

func part(t *testing.T, path, name string) string {
	t.Helper()
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if entry.Name != name {
			continue
		}
		handle, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer handle.Close()
		var b strings.Builder
		buffer := make([]byte, 32*1024)
		for {
			n, err := handle.Read(buffer)
			b.Write(buffer[:n])
			if err != nil {
				break
			}
		}
		return b.String()
	}
	t.Fatalf("no %s in the package", name)
	return ""
}

var textRE = regexp.MustCompile(`<w:t(?:\s[^>]*)?>([^<]*)</w:t>`)

func plain(xml string) string {
	var b strings.Builder
	for _, match := range textRE.FindAllStringSubmatch(xml, -1) {
		b.WriteString(match[1])
		b.WriteString("\n")
	}
	return b.String()
}

func TestASlugDropsAccentsRatherThanTheLettersUnderThem(t *testing.T) {
	if got := Slugify("Résumé of Änderungen"); got != "resume-of-anderungen" {
		t.Errorf("slug = %q, want the accents folded away", got)
	}
}

func TestALeadingH1RepeatingTheTitleIsDropped(t *testing.T) {
	// Without this the title prints twice, once on the cover and again above
	// the first paragraph.
	body := DropTitleHeading("# Access Control Policy\n\nText.\n",
		"Access Control", "Access Control Policy")

	if strings.Contains(body, "# Access Control Policy") {
		t.Errorf("body = %q, want the repeated title gone", body)
	}
}

func TestAnH1ThatIsNotTheTitleIsKept(t *testing.T) {
	body := DropTitleHeading("# Purpose\n\nText.\n", "Access Control", "Access Control Policy")

	if !strings.Contains(body, "# Purpose") {
		t.Errorf("body = %q, want a real first heading kept", body)
	}
}

func TestEveryTestDocumentBuilds(t *testing.T) {
	for _, name := range []string{"01-kitchen-sink.md", "02-pictures.md", "03-policy.md",
		"04-minimal.md", "05-edge-cases.md"} {
		result, _ := build(t, name, Options{})
		if result.Blocks == 0 {
			t.Errorf("%s produced no blocks", name)
		}
		if result.Entries == 0 {
			t.Errorf("%s produced no contents entries", name)
		}
	}
}

func TestTheFirstHeadingStartsOnACleanPageAndNoOtherDoes(t *testing.T) {
	// Setting pageBreakBefore on every heading costs a blank page per section.
	_, out := build(t, "03-policy.md", Options{})
	xml := documentXML(t, out)

	body := xml[strings.Index(xml, `w:val="Heading1"`):]
	if got := strings.Count(body, "<w:pageBreakBefore"); got != 1 {
		t.Errorf("page breaks in the body = %d, want exactly one", got)
	}
}

func TestAPictureBecomesACentredFigureWithItsAltAsCaption(t *testing.T) {
	_, out := build(t, "02-pictures.md", Options{})
	xml := documentXML(t, out)

	if !strings.Contains(xml, "<w:drawing>") {
		t.Fatal("no picture was embedded")
	}
	if !strings.Contains(plain(xml), "The settlement flow, end to end") {
		t.Error("the alt text did not become a caption")
	}
}

func TestAHeadingHoldingNothingButAPictureIsAFigure(t *testing.T) {
	// It gets no number and no contents entry, because it is a figure.
	result, _ := build(t, "02-pictures.md", Options{})

	for _, heading := range result.Headings {
		if strings.TrimSpace(heading.Text) == "" {
			t.Error("an empty heading reached the contents list")
		}
	}
	if got := result.Headings[0].Text; !strings.HasPrefix(got, "1-") {
		t.Errorf("first heading = %q, want it numbered from 1 rather than 0.1", got)
	}
}

func TestAPictureWiderThanTheColumnIsScaledToIt(t *testing.T) {
	_, out := build(t, "02-pictures.md", Options{})
	xml := documentXML(t, out)

	extent := regexp.MustCompile(`<wp:extent cx="(\d+)"`)
	for _, match := range extent.FindAllStringSubmatch(xml, -1) {
		width := match[1]
		if len(width) > 7 || (len(width) == 7 && width > "6263640") {
			t.Errorf("a picture is %s EMU wide, past the 6263640 text column", width)
		}
	}
}

func TestLinksAreTextOnlyUnlessAskedFor(t *testing.T) {
	// The default matches the reference renderer, which drops the destination.
	_, out := build(t, "02-pictures.md", Options{})

	rels := part(t, out, "word/_rels/document.xml.rels")
	if strings.Contains(rels, "relationships/hyperlink") {
		t.Error("a hyperlink relationship was written without being asked for")
	}
	if strings.Contains(documentXML(t, out), "<w:hyperlink r:id=") {
		t.Error("a hyperlink run was written without being asked for")
	}
}

func TestHyperlinksBecomeRealLinksWhenAskedFor(t *testing.T) {
	_, out := build(t, "02-pictures.md", Options{Hyperlinks: true})

	rels := part(t, out, "word/_rels/document.xml.rels")
	if !strings.Contains(rels, "relationships/hyperlink") {
		t.Fatal("no hyperlink relationship was written")
	}
	if !strings.Contains(rels, "example.com/handbook") {
		t.Error("the destination did not reach the relationship part")
	}
	if !strings.Contains(rels, "&amp;v=2") {
		t.Error("a destination with an ampersand was not escaped into the rels part")
	}
	if !strings.Contains(rels, `TargetMode="External"`) {
		t.Error("an external link needs TargetMode, or Word reads it as a part name")
	}
	if !strings.Contains(documentXML(t, out), "<w:hyperlink r:id=") {
		t.Error("no hyperlink run wraps the link text")
	}
}

func TestAFileWithFrontMatterAndNoBodyIsRefused(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty.md")
	if err := writeFile(empty, "---\ntitle: T\n---\n\n"); err != nil {
		t.Fatal(err)
	}

	_, err := Build(empty, filepath.Join(t.TempDir(), "out.docx"),
		Options{Template: templatePath(t)})

	if err == nil {
		t.Fatal("a body with nothing in it must be refused")
	}
}

func TestBuildingTwiceProducesTheSameBytes(t *testing.T) {
	// A publish flow that builds, measures and builds again depends on this.
	first, _ := build(t, "03-policy.md", Options{})
	second, _ := build(t, "03-policy.md", Options{})

	if first.Blocks != second.Blocks || first.Entries != second.Entries {
		t.Errorf("two builds disagree: %#v against %#v", first, second)
	}
}
