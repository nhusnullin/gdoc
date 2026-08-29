package main

// Build a .docx carrying OOXML tracked changes, using only archive/zip and
// string surgery on document.xml. Proves the zip repack path and the minimal
// revision markup in one go.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

const author = "gdoc"
const date = "2026-08-29T00:00:00Z"

// A tracked insertion of a whole new paragraph: the paragraph MARK itself is
// marked inserted via w:pPr/w:rPr/w:ins, and the run is wrapped in w:ins.
const insertedParagraph = `<w:p><w:pPr><w:rPr><w:ins w:id="9001" w:author="` + author + `" w:date="` + date + `"/></w:rPr></w:pPr>` +
	`<w:ins w:id="9002" w:author="` + author + `" w:date="` + date + `">` +
	`<w:r><w:t xml:space="preserve">THIS SENTENCE WAS INSERTED.</w:t></w:r></w:ins></w:p>`

// A tracked deletion of a whole paragraph: paragraph mark deleted via
// w:pPr/w:rPr/w:del, run wrapped in w:del, and w:t becomes w:delText.
const deletedParagraph = `<w:p><w:pPr><w:rPr><w:del w:id="9003" w:author="` + author + `" w:date="` + date + `"/></w:rPr></w:pPr>` +
	`<w:del w:id="9004" w:author="` + author + `" w:date="` + date + `">` +
	`<w:r><w:delText xml:space="preserve">THIS SENTENCE WAS DELETED.</w:delText></w:r></w:del></w:p>`

func main() {
	src, err := zip.OpenReader("template.docx")
	if err != nil {
		panic(err)
	}
	defer src.Close()

	out, err := os.Create("tracked.docx")
	if err != nil {
		panic(err)
	}
	zw := zip.NewWriter(out)

	for _, f := range src.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()

		if f.Name == "word/document.xml" {
			s := string(data)
			i := strings.Index(s, "<w:body>")
			if i < 0 {
				panic("no w:body")
			}
			i += len("<w:body>")
			s = s[:i] + insertedParagraph + deletedParagraph + s[i:]
			data = []byte(s)
		}

		// preserve the original compression method and metadata
		hdr := f.FileHeader
		hdr.Method = zip.Deflate
		w, err := zw.CreateHeader(&hdr)
		if err != nil {
			panic(err)
		}
		if _, err := w.Write(data); err != nil {
			panic(err)
		}
	}
	if err := zw.Close(); err != nil {
		panic(err)
	}
	out.Close()

	st, _ := os.Stat("tracked.docx")
	fmt.Printf("wrote tracked.docx (%d bytes)\n", st.Size())
	// confirm it is still a readable zip with all parts
	check, err := zip.OpenReader("tracked.docx")
	if err != nil {
		panic(err)
	}
	fmt.Printf("parts in original: %d, parts in output: %d\n", len(src.File), len(check.File))
	var buf bytes.Buffer
	for _, f := range check.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			io.Copy(&buf, rc)
			rc.Close()
		}
	}
	fmt.Printf("w:ins count=%d  w:del count=%d  w:delText count=%d\n",
		bytes.Count(buf.Bytes(), []byte("<w:ins ")),
		bytes.Count(buf.Bytes(), []byte("<w:del ")),
		bytes.Count(buf.Bytes(), []byte("<w:delText")))
	check.Close()
}
