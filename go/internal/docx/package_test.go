package docx

import (
	"archive/zip"
	"bytes"
	"testing"
)

func fixture(t *testing.T) *Package {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, part := range []struct{ name, data string }{
		{"[Content_Types].xml", "<Types/>"},
		{"word/document.xml", "<w:document/>"},
		{"word/media/image1.png", "PNGBYTES"},
	} {
		entry, err := writer.Create(part.name)
		if err != nil {
			t.Fatal(err)
		}
		entry.Write([]byte(part.data))
	}
	writer.Close()

	path := t.TempDir() + "/fixture.docx"
	if err := writeFile(path, buffer.Bytes()); err != nil {
		t.Fatal(err)
	}
	pkg, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestEveryPartSurvivesARoundTrip(t *testing.T) {
	// Only the parts we edit may change. That is what keeps the cover, the logo
	// and the coloured tables pixel-identical to the master.
	pkg := fixture(t)
	path := t.TempDir() + "/out.docx"

	if err := pkg.Save(path); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(reopened.Parts) != len(pkg.Parts) {
		t.Fatalf("parts = %d, want %d", len(reopened.Parts), len(pkg.Parts))
	}
	for i := range pkg.Parts {
		if reopened.Parts[i].Name != pkg.Parts[i].Name {
			t.Errorf("part %d = %q, want %q", i, reopened.Parts[i].Name, pkg.Parts[i].Name)
		}
		if !bytes.Equal(reopened.Parts[i].Data, pkg.Parts[i].Data) {
			t.Errorf("part %q changed on the round trip", pkg.Parts[i].Name)
		}
	}
}

func TestSetReplacesInPlaceRatherThanAppending(t *testing.T) {
	pkg := fixture(t)
	before := len(pkg.Parts)

	pkg.Set("word/document.xml", []byte("<w:document>edited</w:document>"))

	if len(pkg.Parts) != before {
		t.Errorf("parts = %d, want %d: Set on an existing part must replace it",
			len(pkg.Parts), before)
	}
	data, _ := pkg.Get("word/document.xml")
	if string(data) != "<w:document>edited</w:document>" {
		t.Errorf("data = %q", data)
	}
}

func TestRemoveIsSilentAboutAPartThatWasNeverThere(t *testing.T) {
	// The callers removing comment parts should not have to know whether the
	// template carried them.
	pkg := fixture(t)
	before := len(pkg.Parts)

	pkg.Remove("word/comments.xml")

	if len(pkg.Parts) != before {
		t.Errorf("parts = %d, want %d unchanged", len(pkg.Parts), before)
	}
}

func TestMustGetNamesThePartItCouldNotFind(t *testing.T) {
	pkg := fixture(t)

	_, err := pkg.MustGet("word/styles.xml")

	if err == nil {
		t.Fatal("a missing part must be reported, not returned as empty bytes")
	}
}

func TestHasPrefixListsTheMediaAlreadyInThePackage(t *testing.T) {
	// The master's own logo lives in word/media, so a new picture must be
	// numbered past it rather than over it.
	pkg := fixture(t)

	if got := pkg.HasPrefix("word/media/"); len(got) != 1 {
		t.Errorf("media = %v, want the template's one picture", got)
	}
}
