// Package docx reads the docx export of a Google Doc, and it exists for one
// question: is this comment still attached to text?
//
// Drive answers that question and its answer cannot be trusted. A comment's
// anchor and its quotedFileContent both survive the text they pointed at being
// deleted, so a thread whose sentence is gone still comes back with the
// sentence in it. The docx export does not: word/comments.xml carries the
// comment, and word/document.xml carries a commentRangeStart only while the
// comment is anchored to something. That is why this is the honest witness, and
// why it is a second read rather than a field on the first one.
//
// Read-only, on encoding/xml. SPEC.md's reason for etree is that encoding/xml
// corrupts OOXML on the way back out, and nothing here writes OOXML.
package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// wNS is the WordprocessingML namespace. Elements are matched by this URI and
// never by the `w:` prefix: a prefix is the document's choice, and an export
// that spelled it differently would read as an empty document.
const wNS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

// docxMime is the export format. It is not a PDF, which is the one format the
// guard refuses outright, and it is the only export that carries the comment
// ranges at all.
const docxMime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

// MaxExportBytes is the ceiling on one export, and on one part read out of it.
// A body without a bound is a memory limit somebody else sets, and a zip that
// decompresses to more than the whole export is not a document gdoc made.
const MaxExportBytes = 32 << 20

// File is one export, reduced to the comments in it.
type File struct {
	Comments []Comment
}

// Comment is one comment of word/comments.xml. ID is the docx's own number and
// has nothing to do with the Drive comment id, which is why Match joins on the
// text rather than on this. Anchored is the fact the whole package is for.
type Comment struct {
	ID       string
	Author   string
	Date     string
	Text     string
	Anchored bool
	Span     string
}

// Reader is the one thing this package needs of a session: a byte GET. A
// *gapi.Session satisfies it, and so does a fake in a test. Naming the
// interface here rather than the struct keeps net/http out of this room and out
// of its tests, which is what the boundary test asks of every package but the
// four that build requests.
type Reader interface {
	GetBytes(ctx context.Context, rawURL string, limit int64) ([]byte, error)
}

// ExportURL is the docx export of one file.
//
// supportsAllDrives is absent, for the same reason it is absent from the comment
// listing. files.export defines two parameters and neither is that one: measured
// against the live Drive v3 discovery document on 2026-09-06, files.export takes
// fileId and mimeType, while files.get is where supportsAllDrives lives. Sending
// a parameter the method does not define is a parameter the server may reject,
// and it would take every --witness run with it.
func ExportURL(id string) string {
	q := url.Values{
		"mimeType": {docxMime},
	}
	return "https://www.googleapis.com/drive/v3/files/" + id + "/export?" + q.Encode()
}

// Export reads the docx through the session's guarded client. A document the
// command was not given is refused inside the process, before anything reaches
// a wire, and that refusal comes back as this function's error.
func Export(ctx context.Context, s Reader, id string) ([]byte, error) {
	return s.GetBytes(ctx, ExportURL(id), MaxExportBytes)
}

// Parse reads the export. It takes bytes, so the fixtures run the same decoder
// the wire feeds.
//
// A document nobody commented on carries no word/comments.xml, and that is no
// comments rather than an error. Bytes with no word/document.xml in them are
// refused by name: an export that failed is usually an HTML sign-in page, and
// reading that as a document with no comments would report every thread as
// detached.
func Parse(b []byte) (*File, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, fmt.Errorf("the export is not a docx: %w", err)
	}
	body, found, err := part(z, "word/document.xml", MaxExportBytes)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("the export is not a docx: it carries no word/document.xml")
	}
	anchored, spans, err := ranges(body)
	if err != nil {
		return nil, err
	}

	f := &File{Comments: []Comment{}}
	raw, found, err := part(z, "word/comments.xml", MaxExportBytes)
	if err != nil {
		return nil, err
	}
	if !found {
		return f, nil
	}
	list, err := parseComments(raw)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		c.Anchored = anchored[c.ID]
		c.Span = spans[c.ID]
		f.Comments = append(f.Comments, c)
	}
	return f, nil
}

// part reads one file out of the zip, bounded by limit. The second return says
// whether the part was there at all, so an absent comments part and an
// unreadable one are two different answers.
//
// A member over the ceiling is an error naming it, never the short bytes. This
// is internal/gapi's rule about a response body, one layer in: handing the XML
// parser a document cut mid-element makes it report that word/document.xml did
// not parse, which blames Google's export for a limit gdoc chose.
func part(z *zip.Reader, name string, limit int64) ([]byte, bool, error) {
	for _, f := range z.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, true, fmt.Errorf("%s could not be opened: %w", name, err)
		}
		defer rc.Close()
		// One byte past the ceiling, so a part that fills it can be told from a
		// part that ended.
		b, err := io.ReadAll(io.LimitReader(rc, limit+1))
		if err != nil {
			return nil, true, fmt.Errorf("%s could not be read: %w", name, err)
		}
		if int64(len(b)) > limit {
			return nil, true, fmt.Errorf("%s is larger than the %d bytes this read allows", name, limit)
		}
		return b, true, nil
	}
	return nil, false, nil
}

// parseComments walks word/comments.xml. Every w:comment is a comment, whatever
// it is nested in, and its text is the w:t runs inside it joined in order: one
// comment is often several runs because the editor split it where the author
// stopped typing.
func parseComments(b []byte) ([]Comment, error) {
	d := xml.NewDecoder(bytes.NewReader(b))
	var out []Comment
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("word/comments.xml did not parse: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || !isW(se.Name, "comment") {
			continue
		}
		text, err := textUntil(d)
		if err != nil {
			return nil, fmt.Errorf("word/comments.xml did not parse: %w", err)
		}
		out = append(out, Comment{
			ID:     attr(se, "id"),
			Author: attr(se, "author"),
			Date:   attr(se, "date"),
			Text:   text,
		})
	}
}

// ranges walks word/document.xml for the comment ranges. A comment is anchored
// when a w:commentRangeStart names it; the span is the text between that and
// its w:commentRangeEnd. Ranges nest and overlap, so every open range collects
// the same text: a run inside two ranges belongs to both.
func ranges(b []byte) (map[string]bool, map[string]string, error) {
	d := xml.NewDecoder(bytes.NewReader(b))
	anchored := map[string]bool{}
	spans := map[string]string{}
	open := map[string]*strings.Builder{}
	inText := 0
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return anchored, spans, nil
		}
		if err != nil {
			return nil, nil, fmt.Errorf("word/document.xml did not parse: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case isW(t.Name, "commentRangeStart"):
				id := attr(t, "id")
				anchored[id] = true
				open[id] = &strings.Builder{}
			case isW(t.Name, "commentRangeEnd"):
				id := attr(t, "id")
				if b, ok := open[id]; ok {
					spans[id] += b.String()
					delete(open, id)
				}
			case isW(t.Name, "t"):
				inText++
			}
		case xml.EndElement:
			if isW(t.Name, "t") && inText > 0 {
				inText--
			}
		case xml.CharData:
			if inText > 0 {
				for _, b := range open {
					b.Write(t)
				}
			}
		}
	}
}

// textUntil reads to the end of the element the decoder is inside and returns
// the w:t text in it, one \n per paragraph that ended. The trailing newline of
// the last paragraph is dropped: it is the element ending, not a blank line the
// author typed.
func textUntil(d *xml.Decoder) (string, error) {
	var b strings.Builder
	depth := 1
	inText := 0
	for {
		tok, err := d.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if isW(t.Name, "t") {
				inText++
			}
		case xml.EndElement:
			depth--
			if isW(t.Name, "t") && inText > 0 {
				inText--
			}
			if isW(t.Name, "p") {
				b.WriteString("\n")
			}
			if depth == 0 {
				return strings.TrimRight(b.String(), "\n"), nil
			}
		case xml.CharData:
			if inText > 0 {
				b.Write(t)
			}
		}
	}
}

// isW is an element of WordprocessingML, matched by namespace URI.
func isW(n xml.Name, local string) bool {
	return n.Space == wNS && n.Local == local
}

// attr reads one attribute by its local name. The namespace is allowed to be
// the WordprocessingML one or none at all: w:id is how Word writes it, and an
// unprefixed id is how a hand-written fixture reads.
func attr(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local && (a.Name.Space == wNS || a.Name.Space == "") {
			return a.Value
		}
	}
	return ""
}
