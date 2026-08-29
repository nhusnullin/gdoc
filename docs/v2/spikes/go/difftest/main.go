package main

// The real product mechanism: word-level diff between two versions of a
// paragraph -> a single paragraph carrying interleaved w:ins / w:del runs.
// Uploaded to Drive this should become native accept/reject suggestions.

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

const oldText = "Altery reviews each supplier annually and records the outcome in the register."
const newText = "Altery reviews each critical supplier every six months and records the outcome in the supplier register."

const author = "gdoc"
const date = "2026-08-29T00:00:00Z"

var nextID = 5000

func id() string { nextID++; return strconv.Itoa(nextID) }

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func runXML(t string) string {
	return `<w:r><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:sz w:val="24"/></w:rPr>` +
		`<w:t xml:space="preserve">` + esc(t) + `</w:t></w:r>`
}

func delRunXML(t string) string {
	return `<w:r><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:sz w:val="24"/></w:rPr>` +
		`<w:delText xml:space="preserve">` + esc(t) + `</w:delText></w:r>`
}

// wordDiff produces the run-level XML for one paragraph as a redline.
func wordDiff(a, b string) (string, int, int) {
	dmp := diffmatchpatch.New()
	// diff on words, not characters: chars produce unreadable shredded runs
	wa, wb, arr := dmp.DiffLinesToChars(strings.ReplaceAll(a, " ", "\n"), strings.ReplaceAll(b, " ", "\n"))
	diffs := dmp.DiffCharsToLines(dmp.DiffMain(wa, wb, false), arr)
	diffs = dmp.DiffCleanupSemantic(diffs)

	var sb strings.Builder
	ins, del := 0, 0
	for _, d := range diffs {
		txt := strings.ReplaceAll(d.Text, "\n", " ")
		if txt == "" {
			continue
		}
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			sb.WriteString(runXML(txt))
		case diffmatchpatch.DiffInsert:
			ins++
			sb.WriteString(`<w:ins w:id="` + id() + `" w:author="` + author + `" w:date="` + date + `">` + runXML(txt) + `</w:ins>`)
		case diffmatchpatch.DiffDelete:
			del++
			sb.WriteString(`<w:del w:id="` + id() + `" w:author="` + author + `" w:date="` + date + `">` + delRunXML(txt) + `</w:del>`)
		}
	}
	return sb.String(), ins, del
}

func main() {
	runs, ins, del := wordDiff(oldText, newText)
	fmt.Printf("word diff produced %d insertions and %d deletions\n\n", ins, del)

	para := `<w:p><w:pPr><w:spacing w:after="240" w:line="276" w:lineRule="auto"/><w:jc w:val="both"/></w:pPr>` + runs + `</w:p>`

	zr, _ := zip.OpenReader("template.docx")
	defer zr.Close()
	out, _ := os.Create("redline.docx")
	zw := zip.NewWriter(out)
	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()
		if f.Name == "word/document.xml" {
			s := string(data)
			i := strings.Index(s, "<w:body>") + len("<w:body>")
			data = []byte(s[:i] + para + s[i:])
		}
		hdr := f.FileHeader
		hdr.Method = zip.Deflate
		w, _ := zw.CreateHeader(&hdr)
		w.Write(data)
	}
	zw.Close()
	out.Close()
	fmt.Println("wrote redline.docx")
}
