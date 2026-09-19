package docx

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gdoc/internal/body"
	"gdoc/internal/cover"
	"gdoc/internal/house"
	"gdoc/internal/render"
)

// noteDir is where the pictures the fixtures name live. It is the body
// package's own testdata rather than a second copy: one badge.png, so a change
// to it moves the generator's goldens and this decoder at the same time.
const noteDir = "../body/testdata/docs"

// TestMediaComesInBodyOrder reads the pictures of a document gdoc rendered:
// two of them, in the order word/document.xml holds them, with the bytes the
// note's own files carry.
//
// The header's logo is a picture of the package and not of the body, so it is
// not in the list: the pairing in internal/export counts what the body holds.
func TestMediaComesInBodyOrder(t *testing.T) {
	doc := rendered(t, "![first](badge.png)\n\n![second](diagram.png)\n")

	media, err := Media(doc)
	if err != nil {
		t.Fatalf("the rendered document's media did not read: %v", err)
	}
	if len(media) != 2 {
		t.Fatalf("the body holds two pictures and Media returned %d: %v", len(media), names(media))
	}
	for i, want := range []string{"badge.png", "diagram.png"} {
		file := filepath.Join(noteDir, want)
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("%s could not be read: %v", file, err)
		}
		if !bytes.Equal(media[i].Bytes, b) {
			t.Errorf("picture %d carries %d bytes and %s is %d", i, len(media[i].Bytes), want, len(b))
		}
		if filepath.Ext(media[i].Name) != ".png" {
			t.Errorf("picture %d is named %q, and a PNG part ends in .png", i, media[i].Name)
		}
	}
	for _, m := range media {
		if m.Name == "word/media/logo.png" {
			t.Errorf("the header's logo is in the body's picture list: %v", names(media))
		}
	}
}

// TestOnePartReferencedTwiceIsTwoPictures pins the count rather than the file:
// the pairing in internal/export is by position, so a picture used twice is
// two positions and two sets of bytes.
func TestOnePartReferencedTwiceIsTwoPictures(t *testing.T) {
	doc := buildDocx(t, map[string]string{
		"word/document.xml":            blipBody("rId9", "rId9"),
		"word/_rels/document.xml.rels": relsPart(map[string]string{"rId9": "media/one.png"}),
		"word/media/one.png":           "the bytes",
	})

	media, err := Media(doc)
	if err != nil {
		t.Fatalf("the document's media did not read: %v", err)
	}
	if len(media) != 2 {
		t.Fatalf("the body names one part twice, which is two pictures, and Media returned %d", len(media))
	}
	if string(media[0].Bytes) != "the bytes" || string(media[1].Bytes) != "the bytes" {
		t.Errorf("both pictures are the same part and read %q and %q", media[0].Bytes, media[1].Bytes)
	}
}

// TestMediaRefusesWhatItCannotResolve names the two shapes that would move
// every picture after them one place along: a relationship the part list does
// not hold, and a picture whose bytes are somewhere else.
func TestMediaRefusesWhatItCannotResolve(t *testing.T) {
	for _, c := range []struct {
		name  string
		parts map[string]string
		want  string
	}{
		{
			"no document part",
			map[string]string{"hello.txt": "hi"},
			"word/document.xml",
		},
		{
			"a relationship that is not there",
			map[string]string{
				"word/document.xml":            blipBody("rId9"),
				"word/_rels/document.xml.rels": relsPart(nil),
			},
			"rId9",
		},
		{
			"a picture that is external",
			map[string]string{
				"word/document.xml":            blipBody("rId9"),
				"word/_rels/document.xml.rels": externalRels("rId9", "https://example.com/one.png"),
			},
			"rId9",
		},
		{
			"a part the zip does not hold",
			map[string]string{
				"word/document.xml":            blipBody("rId9"),
				"word/_rels/document.xml.rels": relsPart(map[string]string{"rId9": "media/gone.png"}),
			},
			"word/media/gone.png",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Media(buildDocx(t, c.parts))
			if err == nil {
				t.Fatalf("%s was read rather than refused", c.name)
			}
			if !bytes.Contains([]byte(err.Error()), []byte(c.want)) {
				t.Errorf("the refusal reads %q and it names %q", err, c.want)
			}
		})
	}
}

// TestADocumentWithNoPicturesHasNoMedia is the ordinary document: no pictures,
// no relationship part needed, and no error.
func TestADocumentWithNoPicturesHasNoMedia(t *testing.T) {
	media, err := Media(buildDocx(t, map[string]string{
		"word/document.xml": `<w:document xmlns:w="` + wNS + `"><w:body><w:p/></w:body></w:document>`,
	}))
	if err != nil {
		t.Fatalf("a document with no pictures did not read: %v", err)
	}
	if len(media) != 0 {
		t.Errorf("a document with no pictures carries %d pictures", len(media))
	}
}

// blipBody is a document.xml whose body holds one drawing per relationship id,
// in the order they are given.
func blipBody(ids ...string) string {
	var b bytes.Buffer
	b.WriteString(`<w:document xmlns:w="` + wNS + `" xmlns:a="` + aNS + `" xmlns:r="` + rNS + `"><w:body>`)
	for _, id := range ids {
		b.WriteString(`<w:p><w:r><w:drawing><a:blip r:embed="` + id + `"/></w:drawing></w:r></w:p>`)
	}
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

// relsPart is word/_rels/document.xml.rels with one image relationship per
// entry, written in no particular order: the order is the body's.
func relsPart(targets map[string]string) string {
	var b bytes.Buffer
	b.WriteString(`<Relationships xmlns="` + pkgRelNS + `">`)
	for id, target := range targets {
		b.WriteString(`<Relationship Id="` + id + `" Type="` + imageRelType + `" Target="` + target + `"/>`)
	}
	b.WriteString(`</Relationships>`)
	return b.String()
}

// externalRels is a relationship whose bytes are not in the package.
func externalRels(id, target string) string {
	return `<Relationships xmlns="` + pkgRelNS + `"><Relationship Id="` + id +
		`" Type="` + imageRelType + `" Target="` + target + `" TargetMode="External"/></Relationships>`
}

func names(media []Medium) []string {
	out := make([]string, 0, len(media))
	for _, m := range media {
		out = append(out, m.Name)
	}
	return out
}

// rendered is the note's markdown through the whole generator, zipped, which is
// the same bytes `gdoc build` writes. The pictures resolve against the body
// package's own testdata.
func rendered(t *testing.T, markdown string) []byte {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style did not load: %v", err)
	}
	fields, md, err := cover.Read([]byte("---\ntitle: Two Pictures\n---\n\n" + markdown))
	if err != nil {
		t.Fatalf("the note's front matter did not read: %v", err)
	}
	walked, err := body.Render(cfg, md, noteDir, fields.HeadingNumbering)
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
	return buf.Bytes()
}

// imageRelType is what a picture's relationship is called. Nothing in the
// decoder reads it: a:blip already says the reference is a picture, and an
// export that spelled the type differently would otherwise lose every one.
const imageRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
