// This file is the pictures: which objects the document holds, what the docx
// export carried for each, and which of them the note at --out already has.
// project.go holds the text, and doc.go holds the package comment.

package export

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"gdoc/internal/docs"
	"gdoc/internal/docx"
)

// MaxPictureBytes is the ceiling on one picture read off the disk. A note may
// name any file at all, and a read with no bound is a memory limit somebody
// else sets.
const MaxPictureBytes = 32 << 20

// Picture is one picture of the document, with what the export carried for it.
//
// Bytes empty and Matched empty together are a placeholder: the projection
// prints the comment internal/view writes and no file is written. Matched set
// is the note's own picture, kept rather than copied, and it carries no bytes
// for the same reason.
type Picture struct {
	// ObjectID is the object's own id, inline or floating. It is what
	// internal/view names a picture's file by.
	ObjectID string `json:"-"`
	// Kind is docs.KindImage or docs.KindDrawing.
	Kind string `json:"-"`
	// Ext is the extension of the part the bytes came from, ".png" and the
	// rest, so the writer names the file what it is without reading it.
	Ext string `json:"-"`
	// Bytes is what the docx export carried, and nothing when there is
	// nothing to write.
	Bytes []byte `json:"-"`
	// File is the path the writer wrote, filled in by it.
	File string `json:"file,omitempty"`
	// Matched is the note's own picture, as the note spells it, when the bytes
	// are the same. Empty is every other case, matched and unmatched alike:
	// the match is off until it is measured.
	Matched string `json:"matched,omitempty"`
}

// Options is what the caller decides about the pictures. One field, and a
// field rather than a package variable: a package variable is shared state two
// tests running under -race would fight over.
type Options struct {
	// MatchByHash keeps the note's own picture when the exported bytes are the
	// same, instead of writing a copy. It is off until measurement 2 of
	// docs/v2/MEASURED.md says a PNG publish uploaded comes back from the
	// export byte for byte.
	MatchByHash bool
}

// NotePicture is one picture the note at --out already holds: how the note
// spells it, where that resolved to, and its bytes.
type NotePicture struct {
	Target string
	Path   string
	Bytes  []byte
}

// Objects is every picture of one tab, in body order: an inline picture where
// its run stands, and a floating one after the paragraph it is anchored to,
// which is where internal/view prints its placeholder.
//
// Only a picture and a drawing are counted. An equation or an object this read
// cannot name is a placeholder in the text and never a file, so counting it
// here would put the pairing one place out.
func Objects(t docs.Tab) []docs.Object {
	var out []docs.Object
	walk(t.Body, func(b docs.Block) {
		if b.Paragraph == nil {
			return
		}
		for _, r := range b.Paragraph.Runs {
			if isPicture(r.Kind) && r.Detail != nil && r.Detail.ID != "" {
				out = append(out, docs.Object{ID: r.Detail.ID, Kind: r.Kind})
			}
		}
		for _, id := range b.Paragraph.Positioned {
			if o, ok := t.Positioned[id]; ok && isPicture(o.Kind) {
				out = append(out, o)
			}
		}
	})
	return out
}

// Pictures pairs the objects the read holds with the bytes the export carried,
// by order, and says which of them the note already has.
//
// Order is the whole pairing. Nothing crosses the two routes: the read has
// object ids the export never mentions and the export has part names the read
// never mentions, so the k-th picture in one is the k-th in the other. When
// the two counts disagree nothing says which bytes belong to which object, so
// every picture is a placeholder and no file is written at all.
// TestPicturesPairByOrder, TestTwoIdenticalPicturesKeepTheirOrder and
// TestACountMismatchWritesNoPictureAndWarns are the pins.
//
// It decides nothing else. What a picture means, and whether a note should
// take it, are the session's.
func Pictures(objects []docs.Object, media []docx.Medium, note []NotePicture, opts Options) ([]Picture, []string) {
	pics := make([]Picture, 0, len(objects))
	for _, o := range objects {
		pics = append(pics, Picture{ObjectID: o.ID, Kind: o.Kind})
	}
	if len(objects) == 0 {
		return pics, nil
	}
	if len(objects) != len(media) {
		return pics, []string{fmt.Sprintf(
			"the document holds %d picture(s) and its export carried %d, and nothing pairs the two "+
				"routes but their order, so every picture is a placeholder and no file was written",
			len(objects), len(media))}
	}

	for i := range pics {
		pics[i].Ext = strings.ToLower(filepath.Ext(media[i].Name))
		pics[i].Bytes = media[i].Bytes
	}
	if !opts.MatchByHash {
		if len(note) == 0 {
			// Nothing to match against, so the match being off cost nothing
			// and there is nothing to say about it.
			return pics, nil
		}
		return pics, []string{
			"the match against the note's own pictures is off until an exported picture is measured " +
				"byte for byte against the one publish uploaded, so every picture here was written as " +
				"a new file even where the note already holds those bytes",
		}
	}

	held := map[[32]byte]string{}
	for _, n := range note {
		sum := sha256.Sum256(n.Bytes)
		if _, seen := held[sum]; !seen {
			held[sum] = n.Target
		}
	}
	for i := range pics {
		target, ok := held[sha256.Sum256(pics[i].Bytes)]
		if !ok {
			continue
		}
		// The note already holds these bytes, so its own link stands and
		// nothing is written. Dropping the bytes here is what makes that
		// structural rather than a rule the writer has to remember.
		pics[i].Matched, pics[i].Bytes = target, nil
	}
	return pics, nil
}

// NotePictures is every picture the note at --out holds, found through its own
// image links, resolved against its directory and read.
//
// Only this note is read. A second note in the same folder naming the same
// document is somebody else's pairing, and reading it would offer a picture
// this file never mentioned. TestTheNotesPicturesAreFoundThroughItsLinks is
// the pin.
//
// A data: destination came with the markdown and is not a file, and an http
// one is a download, which nothing here does: both are stepped over. A file
// that is neither PNG nor JPEG, and one that could not be read at all, are
// named rather than dropped in silence: the export is the one read that could
// have told the session that link is broken.
func NotePictures(src []byte, dir string) ([]NotePicture, []string) {
	var found []NotePicture
	var warnings []string
	for _, target := range imageTargets(src) {
		lower := strings.ToLower(target)
		if strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "http://") ||
			strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "//") {
			continue
		}
		path := target
		if !filepath.IsAbs(path) {
			if dir == "" {
				continue
			}
			path = filepath.Join(dir, filepath.FromSlash(target))
		}
		b, err := readPicture(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"the note links the picture %s and it was not read, so nothing here can say whether "+
					"the document still holds it: %s", target, err))
			continue
		}
		found = append(found, NotePicture{Target: target, Path: path, Bytes: b})
	}
	return found, warnings
}

// readPicture is one file of the note's own, bounded and checked. Only PNG and
// JPEG are read, which is what publish embeds and what the export gives back,
// so anything else could not match a picture of the document anyway.
func readPicture(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("it could not be opened")
	}
	// A regular file, because the size of anything else says nothing about how
	// much reading it gives back: a device node stats at nothing and reads for
	// ever, and a pipe blocks the run. A directory is one of these and keeps
	// its own sentence, because it is the one a person actually writes by
	// mistake.
	if info.IsDir() {
		return nil, fmt.Errorf("it is a directory")
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("it is not a regular file")
	}
	if info.Size() > MaxPictureBytes {
		return nil, fmt.Errorf("it is larger than the %d bytes this read allows", MaxPictureBytes)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("it could not be read")
	}
	defer f.Close()
	// The ceiling again, over the read itself. The stat above is a fact about
	// the file a moment ago, and the file can grow between the two.
	b, err := io.ReadAll(io.LimitReader(f, MaxPictureBytes+1))
	if err != nil {
		return nil, fmt.Errorf("it could not be read")
	}
	if int64(len(b)) > MaxPictureBytes {
		return nil, fmt.Errorf("it is larger than the %d bytes this read allows", MaxPictureBytes)
	}
	if !isPNG(b) && !isJPEG(b) {
		return nil, fmt.Errorf("it is neither a PNG nor a JPEG, and those are the two this read carries")
	}
	return b, nil
}

func isPNG(b []byte) bool  { return len(b) > 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n" }
func isJPEG(b []byte) bool { return len(b) > 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF }

func isPicture(kind string) bool {
	return kind == docs.KindImage || kind == docs.KindDrawing
}

// imageTargets is every image destination of the note, in the order it writes
// them. goldmark is the same reader publish walks the note with, so a picture
// this finds is a picture that would be embedded.
func imageTargets(src []byte) []string {
	root := goldmark.New().Parser().Parse(text.NewReader(src))
	var out []string
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if image, ok := n.(*ast.Image); ok {
			out = append(out, string(image.Destination))
		}
		return ast.WalkContinue, nil
	})
	return out
}
