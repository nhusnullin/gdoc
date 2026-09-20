// Package docx reads the docx export of a Google Doc. It answers two questions
// the Docs read cannot: is this comment still attached to text, and what are
// the bytes of the pictures?
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
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The witness is a second read, not a field
//
// Asking whether a comment is still attached costs an export, which is why
// --witness is a flag and not something every listing pays for. The three
// answers are anchored, detached and unmatched, and every one of them is a fact
// about the export rather than a judgement about the thread: what a detached
// comment means, and whether it still needs an answer, is the skill's.
// TestMatchGivesAnchoredDetachedAndUnmatched is the pin.
//
// An export that could not be read at all is every thread unmatched, on a
// listing that still carries the threads, rather than a failed listing. The
// caller warns and says so. TestMatchWithoutAnExportLeavesEveryThreadUnmatched
// is the pin.
//
// # The witness has two limits, and both are worth stating
//
// It names a destroyed anchor and not a moved one. detached means
// word/document.xml carries no commentRangeStart for the comment, so the text
// it was written about has gone; text that was cut and pasted elsewhere is
// still anchored and reads as untouched.
//
// And two exported comments that match one thread and disagree about being
// anchored give no answer. The thread comes back unmatched rather than taking
// the first. The join is on the comment's own words and its author's name,
// because the docx carries no Drive comment id, and nothing orders the two
// sides against each other: Drive's comments.list defines no ordering and
// word/comments.xml is numbered by the export, so first-fit would hand one
// thread id the other's witness. --since reaches it with one thread in view,
// because the listing is narrowed to the cursor window and the export is not.
// TestOneThreadWithTwoDisagreeingCandidatesIsUnmatched and
// TestAnAmbiguousWitnessIsUnmatchedRatherThanGuessed are the pins, and
// TestTwoThreadsWithTheSameWordsTakeTwoExportedComments is the case that still
// answers: two comments that agree, or two threads with one exported comment
// each, are joined rather than refused.
//
// Refusing to guess is the same answer this tool gives everywhere else it
// cannot stand behind a fact. A witness pinned on the wrong id points the skill
// at the wrong sentence.
//
// # The pictures are the body's, in the body's order
//
// Media is the second read this package is for. The Docs answer says a picture
// is there and never what it holds: its contentUri is a googleusercontent.com
// host the guard does not admit, so the bytes come from the export instead.
//
// Nothing crosses the two routes. The read has object ids the export never
// mentions and the export has part names the read never mentions, so the k-th
// picture in one is the k-th in the other, and order is the whole pairing.
// That is why Media refuses a picture it cannot resolve rather than skipping
// it: one skipped picture moves every picture after it one place along, and a
// caller would then write the wrong bytes under the right name.
// TestMediaComesInBodyOrder and TestMediaRefusesWhatItCannotResolve are the
// pins, and internal/export holds the other half of the pairing.
//
// Only word/document.xml is read, so the header's logo is not a picture of the
// body, and only a:blip is read, so a floating picture Word wrote twice, as
// DrawingML and as a VML fallback, is one picture rather than two.
// TestOnePartReferencedTwiceIsTwoPictures is the pin on the other direction:
// one part used twice is two pictures, because what the list says is where
// each picture stands.
package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path"
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

// aNS is the DrawingML namespace, where a picture's reference to its bytes
// lives, and rNS is the namespace of the attribute that reference is written
// in. pkgRelNS is the relationship part's own namespace, which is a third one
// again: a docx names the same idea in three vocabularies.
const (
	aNS      = "http://schemas.openxmlformats.org/drawingml/2006/main"
	rNS      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	pkgRelNS = "http://schemas.openxmlformats.org/package/2006/relationships"
)

// Medium is one picture of the body: the part it came from and its bytes.
//
// Name is the part's own name, "word/media/image1.png", so the extension says
// what the bytes are without reading them. Two pictures of one part are two
// Media with one Name, because what this list says is where each picture
// stands rather than which files the package holds.
type Medium struct {
	Name  string `json:"name"`
	Bytes []byte `json:"-"`
}

// Media is every picture of word/document.xml, in the order the body holds
// them, with the bytes each points at.
//
// The order is the whole answer. Nothing crosses the Docs read and the docx
// export: the read has object ids the export never mentions, and the export
// has part names the read never mentions, so the k-th picture in one is the
// k-th in the other and there is no second way to pair them. internal/export
// does that pairing and refuses it when the counts disagree.
//
// Only word/document.xml is walked, so the header's logo and a footer's
// picture are not in the list: they are the package's furniture rather than
// the body's content, and counting them would move every picture after them
// one place along.
//
// Only a:blip is read, and never VML's v:imagedata. Word writes a floating
// picture twice, as DrawingML inside mc:Choice and as VML inside mc:Fallback,
// so reading both would count one picture as two. A picture that reaches the
// body as VML alone is therefore missing from this list, and the count
// mismatch in internal/export is what catches it: a missing picture is a
// placeholder, never a wrong file.
//
// TestMediaComesInBodyOrder and TestOnePartReferencedTwiceIsTwoPictures are
// the pins.
func Media(b []byte) ([]Medium, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, fmt.Errorf("the export is not a docx: %w", err)
	}
	doc, found, err := part(z, "word/document.xml", MaxExportBytes)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("the export is not a docx: it carries no word/document.xml")
	}
	ids, err := blips(doc)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	rels, _, err := part(z, "word/_rels/document.xml.rels", MaxExportBytes)
	if err != nil {
		return nil, err
	}
	targets, err := embedded(rels)
	if err != nil {
		return nil, err
	}

	out := make([]Medium, 0, len(ids))
	held := map[string][]byte{}
	for _, id := range ids {
		target, ok := targets[id]
		if !ok {
			// A picture whose bytes this package does not hold, or does not
			// name at all. Skipping it would move every picture after it one
			// place along, and the pairing is by position, so it is refused.
			return nil, fmt.Errorf("word/document.xml holds a picture at the relationship %s, "+
				"and word/_rels/document.xml.rels names no part of this file for it", id)
		}
		name := mediaPath(target)
		data, read := held[name]
		if !read {
			data, found, err = part(z, name, MaxExportBytes)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, fmt.Errorf("the export names the picture part %s and does not carry it", name)
			}
			held[name] = data
		}
		out = append(out, Medium{Name: name, Bytes: data})
	}
	return out, nil
}

// blips is the relationship id of every picture of the body, in document
// order. A picture that names no embedded part is refused rather than skipped,
// for the reason Media's caller pairs by position.
func blips(b []byte) ([]string, error) {
	d := xml.NewDecoder(bytes.NewReader(b))
	var out []string
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("word/document.xml did not parse: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Space != aNS || se.Name.Local != "blip" {
			continue
		}
		id := rAttr(se, "embed")
		if id == "" {
			return nil, fmt.Errorf("word/document.xml holds a picture with no embedded part behind it, " +
				"so nothing in this export says what it shows")
		}
		out = append(out, id)
	}
}

// embedded is the relationship id of every part of this package, by id. A
// relationship whose target is somewhere else is left out: its bytes are not
// in the export, so a picture pointing at it has nothing to carry.
func embedded(b []byte) (map[string]string, error) {
	if len(b) == 0 {
		return map[string]string{}, nil
	}
	d := xml.NewDecoder(bytes.NewReader(b))
	out := map[string]string{}
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("word/_rels/document.xml.rels did not parse: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Space != pkgRelNS || se.Name.Local != "Relationship" {
			continue
		}
		if plain(se, "TargetMode") == "External" {
			continue
		}
		if id, target := plain(se, "Id"), plain(se, "Target"); id != "" && target != "" {
			out[id] = target
		}
	}
}

// mediaPath is one relationship target as a part name. A target is relative to
// the part that names it, word/document.xml, so it is resolved against word/;
// a target opening with a slash is already from the package root.
func mediaPath(target string) string {
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(target, "/")
	}
	return path.Join("word", target)
}

// rAttr reads one attribute of the relationship namespace. The namespace is
// allowed to be absent as well, for the same reason attr allows it: an
// unprefixed embed is how a hand-written fixture reads.
func rAttr(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local && (a.Name.Space == rNS || a.Name.Space == "") {
			return a.Value
		}
	}
	return ""
}

// plain reads one unnamespaced attribute, which is how the relationship part
// writes every one of its own.
func plain(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local && a.Name.Space == "" {
			return a.Value
		}
	}
	return ""
}
