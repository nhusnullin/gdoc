// This file is the export on disk: where every file lands, what the block in
// each one says, and the one stamp on a note that was already there.
// project.go holds the text, pictures.go holds the bytes, and doc.go holds the
// package comment.

package export

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gdoc/internal/atomicfile"
	"gdoc/internal/frontmatter"
	"gdoc/internal/view"
)

// assetsDir is the folder a picture lands in, beside the note. It is created
// when it is missing and never looked for anywhere else.
const assetsDir = "assets"

// maxNumbered is how far the numbering rule counts before it gives up. A run
// that finds a thousand copies of one name has met something this code does
// not understand, and looping for ever is the worse answer.
const maxNumbered = 1000

// defaultExt is what a picture is called when the export carried no extension
// for it. Google gives PNG back, so PNG is the guess.
const defaultExt = ".png"

// folderMode is the mode the assets folder is made with.
const folderMode = 0o755

// Tab is one tab of the document as the writer needs it before anything is
// projected: which tab it is, and what it is called. The title is where a
// further tab's file name comes from, decision 11.
type Tab struct {
	ID    string
	Title string
}

// Request is one export as the writer is asked for it.
type Request struct {
	// Out is the path the person named. The first tab lands there, or beside
	// it when it is taken, and every other path in the run is built from it.
	Out string
	// DocumentID is the document that was read. It is what a note at Out is
	// checked against, and what every block written here names.
	DocumentID string
	// At is when the document was read, the one fact the block records.
	At time.Time
	// Tabs is every tab of the document, in document order.
	Tabs []Tab
	// Pictures is every picture of the document, in the order they are
	// written, as internal/export's Pictures paired them.
	Pictures []Picture
}

// Placed is where one tab's file landed: the path it took, whether the path it
// was asked for was taken by something else, and the note it landed beside.
type Placed struct {
	TabID    string `json:"tab_id,omitempty"`
	TabTitle string `json:"tab_title,omitempty"`
	Path     string `json:"path"`
	// Note is the note this file was written beside, as a path relative to
	// its own directory, and nothing when the file is the note.
	Note string `json:"note,omitempty"`
	// Taken says the path this tab was asked for held something that is not a
	// note for this document, so the file went to the next free number.
	Taken bool `json:"taken,omitempty"`
}

// Stamp is one note this export wrote the date into.
type Stamp struct {
	Path string `json:"path"`
	// SchemaRewritten says this note's block was schema 1 and the stamp moved
	// it to schema 2, which is the one line the reply owes the person:
	// scenario 23.
	SchemaRewritten bool `json:"schema_rewritten,omitempty"`
}

// Layout is every path one export will take, decided before anything is
// projected and before a byte reaches the disk.
//
// It exists because a picture's line in the body names the file that picture
// lands in, so the names have to exist before the text does. Plan takes them
// and checks every door; Write writes. The gap between the two is closed by
// atomicfile.Create, which refuses a path that appeared in between rather
// than replacing it.
type Layout struct {
	Files    []Placed
	Pictures []Picture

	dir        string
	documentID string
	at         time.Time
	// named says every entry carries its tab id, which is true of a document
	// with more than one tab and of nothing else.
	named bool
}

// Written is what one export put on disk.
type Written struct {
	Files    []Placed
	Pictures []Picture
	Stamped  []Stamp
}

// Plan decides where every file of this export lands and refuses what it must
// refuse, writing nothing.
//
// Two refusals, the ones the spec names: a note at a path this run would land
// on that names other documents, and front matter that does not read. Both are
// answered here, before any file exists, so a run that refuses leaves the
// folder exactly as it found it. Anything else at a path is simply a taken
// path, and the numbering rule answers it.
func Plan(req Request) (*Layout, error) {
	if req.Out == "" {
		return nil, fmt.Errorf("export: no path to write to")
	}
	if req.DocumentID == "" {
		return nil, fmt.Errorf("export: no document id, and the block in every file names one")
	}
	if req.At.IsZero() {
		return nil, fmt.Errorf("export: no time, and the block in every file records when the document was read")
	}
	if len(req.Tabs) == 0 {
		return nil, fmt.Errorf("export: no tabs, and a document has at least one")
	}

	dir := filepath.Dir(req.Out)
	base := filepath.Base(req.Out)
	ext := filepath.Ext(base)
	if ext == "" {
		ext = ".md"
	}
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	l := &Layout{dir: dir, documentID: req.DocumentID, at: req.At, named: len(req.Tabs) > 1}
	taken := map[string]bool{}
	for i, tab := range req.Tabs {
		name := stem
		if i > 0 {
			name = stem + "-" + tabSlug(tab, i)
		}
		placed, err := place(dir, name, ext, req.DocumentID, taken)
		if err != nil {
			return nil, err
		}
		placed.TabID, placed.TabTitle = tab.ID, tab.Title
		l.Files = append(l.Files, placed)
	}

	pics, err := placePictures(dir, stem, req.Pictures)
	if err != nil {
		return nil, err
	}
	l.Pictures = pics
	return l, nil
}

// PictureNames is what each picture is written as in the body: the file this
// run will write, the note's own link when the bytes matched, and nothing at
// all for a picture with neither, which leaves internal/view's placeholder and
// its warning where they were.
func (l *Layout) PictureNames() PictureNames {
	names := make(map[string]string, len(l.Pictures))
	for _, p := range l.Pictures {
		switch {
		case p.Matched != "":
			names[p.ObjectID] = imageLink(p.Matched)
		case p.File != "":
			names[p.ObjectID] = imageLink(p.File)
		}
	}
	return func(id string) string { return names[id] }
}

// Write puts the bodies at the paths the layout took, writes every picture
// that carries bytes, and stamps every note this export landed beside.
//
// Every new file goes through atomicfile.Create, which ends in a link and so
// refuses a path that exists. The one call to atomicfile.Replace in this
// package is the stamp, on a note that was already there and whose block is
// read fresh a moment before it is written. TestNothingCanReplaceAFile is the
// pin over the source.
//
// The pictures go first, so a file that exists always has the pictures its
// text names, and the stamp goes last, because it is the only thing here that
// touches a file somebody else wrote.
func Write(l *Layout, files []TabFile) (*Written, error) {
	if l == nil {
		return nil, fmt.Errorf("export: nothing was planned, so there is nothing to write")
	}
	if len(files) != len(l.Files) {
		return nil, fmt.Errorf("export: %d tab(s) were planned and %d projected", len(l.Files), len(files))
	}

	if err := makeAssets(l); err != nil {
		return nil, err
	}
	for _, p := range l.Pictures {
		if len(p.Bytes) == 0 {
			continue
		}
		if err := atomicfile.Create(filepath.Join(l.dir, filepath.FromSlash(p.File)), p.Bytes); err != nil {
			return nil, fmt.Errorf("export: the picture %s was not written: %w", p.File, err)
		}
	}

	for i, placed := range l.Files {
		if files[i].TabID != placed.TabID {
			return nil, fmt.Errorf("export: the file at position %d is tab %q and the plan holds tab %q",
				i, files[i].TabID, placed.TabID)
		}
		b, err := File(files[i].Body, l.block(placed))
		if err != nil {
			return nil, err
		}
		if err := atomicfile.Create(placed.Path, b); err != nil {
			return nil, fmt.Errorf("export: %s was not written: %w", placed.Path, err)
		}
	}

	w := &Written{Files: l.Files, Pictures: l.Pictures}
	for _, placed := range l.Files {
		if placed.Note == "" {
			continue
		}
		s, err := stamp(filepath.Join(l.dir, placed.Note), l.documentID, l.at)
		if err != nil {
			return nil, err
		}
		w.Stamped = append(w.Stamped, s)
	}
	return w, nil
}

// block is the front matter of one file this export writes: one entry, the
// document it came from, the day it was read, and the note it belongs to when
// it is a copy beside one.
//
// The tab id is written only when the document has more than one tab. A
// document with one tab is one file, and a tab id there would be a fact about
// nothing.
func (l *Layout) block(p Placed) *frontmatter.Block {
	e := frontmatter.Entry{ID: l.documentID, Exported: &frontmatter.Exported{At: l.at, Note: p.Note}}
	if l.named {
		e.TabID = p.TabID
	}
	return &frontmatter.Block{Schema: frontmatter.Schema, Documents: []frontmatter.Entry{e}}
}

// makeAssets creates the folder beside the note, and only when there is a
// picture to put in it. An export that writes no picture leaves no empty
// folder behind.
func makeAssets(l *Layout) error {
	for _, p := range l.Pictures {
		if len(p.Bytes) > 0 {
			return os.MkdirAll(filepath.Join(l.dir, assetsDir), folderMode)
		}
	}
	return nil
}

// stamp is the one dated fact export writes into a note: when this document
// was read into the hub.
//
// The file is read again here rather than at the door, because a live review
// session may have written the same block in between, decision 14. The write
// replaces the whole file in one rename through internal/atomicfile, so that
// session sees the old bytes or the new ones and never a half file.
func stamp(path, documentID string, at time.Time) (Stamp, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return Stamp{}, fmt.Errorf("export: the note at %s was not read back to be stamped: %w", path, err)
	}
	b, err := frontmatter.Read(src)
	if err != nil {
		return Stamp{}, fmt.Errorf("export: the note at %s: %w", path, err)
	}
	if b == nil {
		return Stamp{}, fmt.Errorf("export: the note at %s lost its gdoc: block while this export ran, so it was not stamped", path)
	}
	e, err := b.Entry(documentID)
	if err != nil {
		return Stamp{}, fmt.Errorf("export: the note at %s: %w", path, err)
	}

	// The note: line is the copy's, not this note's, and nothing in the binary
	// takes it out: a person who adopted a copy as their note removes it by
	// hand. So a stamp carries through whatever is there.
	kept := ""
	if e.Exported != nil {
		kept = e.Exported.Note
	}
	e.Exported = &frontmatter.Exported{At: at, Note: kept}

	out, err := frontmatter.Write(src, b)
	if err != nil {
		return Stamp{}, fmt.Errorf("export: the note at %s was left as it was: %w", path, err)
	}
	if err := atomicfile.Replace(path, out, atomicfile.ModeOf(path, atomicfile.NewMode)); err != nil {
		return Stamp{}, fmt.Errorf("export: the note at %s was not stamped: %w", path, err)
	}
	return Stamp{Path: path, SchemaRewritten: !bytes.Equal(src, out) && schemaOf(src) == frontmatter.SchemaOne}, nil
}

// schemaOf is what the block in these bytes said before the write. A file
// whose block does not read is not this function's problem: it was refused at
// the door and read again in stamp.
func schemaOf(src []byte) int {
	b, err := frontmatter.Read(src)
	if err != nil || b == nil {
		return 0
	}
	return b.Schema
}

// place is one tab's path, and what was at the path it asked for.
//
// Three answers, the spec's own: the path is free and the file lands there; it
// holds a note whose list names this document, which is stamped while the file
// lands beside it; it holds anything else, and the file lands beside it with
// the reply saying the path was taken. A note naming other documents and front
// matter that does not read are the two refusals.
func place(dir, name, ext, documentID string, taken map[string]bool) (Placed, error) {
	base := filepath.Join(dir, name+ext)
	state, err := inspect(base, documentID)
	if err != nil {
		return Placed{}, err
	}
	if state == free && !taken[base] {
		taken[base] = true
		return Placed{Path: base}, nil
	}
	path, err := nextFree(dir, name, ext, taken)
	if err != nil {
		return Placed{}, err
	}
	taken[path] = true
	p := Placed{Path: path}
	switch state {
	case isNote:
		p.Note = filepath.Base(base)
	case other:
		p.Taken = true
	}
	return p, nil
}

// state is what a path this export would land on holds.
type state int

const (
	// free is nothing at all.
	free state = iota
	// isNote is a note whose block names the document being exported.
	isNote
	// other is any other file, which is a taken path and nothing more.
	other
)

// inspect reads what is at a path, and refuses the two things the spec says to
// refuse. It never writes and never repairs.
func inspect(path, documentID string) (state, error) {
	if _, err := os.Lstat(path); err != nil {
		return free, nil
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return free, fmt.Errorf("export: %s is in the way and could not be read, so nothing was written: %w", path, err)
	}
	b, err := frontmatter.Read(src)
	if err != nil {
		return free, fmt.Errorf("export: %s holds front matter gdoc cannot read, so nothing was written: %w", path, err)
	}
	if b == nil {
		return other, nil
	}
	if _, err := b.Entry(documentID); err != nil {
		return free, fmt.Errorf("export: the note at %s is paired with other documents, so nothing was written: %w", path, err)
	}
	return isNote, nil
}

// nextFree is the numbering rule of decision 2: <name>.2<ext>, then
// <name>.3<ext>, and so on past every name this run has already taken.
func nextFree(dir, name, ext string, taken map[string]bool) (string, error) {
	for n := 2; n <= maxNumbered; n++ {
		path := filepath.Join(dir, name+"."+strconv.Itoa(n)+ext)
		if taken[path] {
			continue
		}
		if _, err := os.Lstat(path); err == nil {
			continue
		}
		return path, nil
	}
	return "", fmt.Errorf("export: %s and its first %d numbered names are all taken, so this run has nowhere to write",
		filepath.Join(dir, name+ext), maxNumbered)
}

// placePictures names every picture that has bytes, from the next free number
// in the assets folder. A picture the note already holds and a picture with no
// bytes at all are left as they are: neither becomes a file.
func placePictures(dir, stem string, pics []Picture) ([]Picture, error) {
	out := make([]Picture, len(pics))
	copy(out, pics)

	wanted := false
	for _, p := range out {
		if len(p.Bytes) > 0 {
			wanted = true
		}
	}
	if !wanted {
		return out, nil
	}

	folder := filepath.Join(dir, assetsDir)
	if info, err := os.Lstat(folder); err == nil && !info.IsDir() {
		return nil, fmt.Errorf("export: %s is a file and the pictures go in a folder of that name, so nothing was written", folder)
	}
	held, err := heldNumbers(folder, stem)
	if err != nil {
		return nil, err
	}

	n := 1
	for i := range out {
		if len(out[i].Bytes) == 0 {
			continue
		}
		for held[n] {
			n++
		}
		if n > maxNumbered {
			return nil, fmt.Errorf("export: the first %d picture names under %s are taken", maxNumbered, folder)
		}
		held[n] = true
		out[i].File = assetsDir + "/" + stem + "-" + strconv.Itoa(n) + extOf(out[i])
	}
	return out, nil
}

// heldNumbers is every number already used under this stem in the assets
// folder, whatever extension it was written with. A PNG at <stem>-1.png means
// the number is taken, so a JPEG cannot land beside it as <stem>-1.jpg and
// leave a reader guessing which is which.
func heldNumbers(folder, stem string) (map[int]bool, error) {
	held := map[int]bool{}
	entries, err := os.ReadDir(folder)
	if err != nil {
		if os.IsNotExist(err) {
			return held, nil
		}
		return nil, fmt.Errorf("export: %s could not be read, so nothing was written: %w", folder, err)
	}
	for _, e := range entries {
		rest, ok := strings.CutPrefix(e.Name(), stem+"-")
		if !ok {
			continue
		}
		digits := strings.TrimSuffix(rest, filepath.Ext(rest))
		if n, err := strconv.Atoi(digits); err == nil {
			held[n] = true
		}
	}
	return held, nil
}

// extOf is what a picture's file is called, from the part its bytes came out
// of. A part with no extension at all is a PNG, which is what the export
// gives back.
func extOf(p Picture) string {
	if strings.HasPrefix(p.Ext, ".") {
		return p.Ext
	}
	return defaultExt
}

// tabSlug is a tab's half of its file name: the title slugged by the one rule
// internal/view holds, and tab-<n> by position when the title slugs to
// nothing, so an untitled tab still lands somewhere a person can find.
func tabSlug(t Tab, i int) string {
	if s := view.Slug(t.Title); s != "" {
		return s
	}
	return "tab-" + strconv.Itoa(i+1)
}

// imageLink is one picture's markdown. The address goes through the same rule
// the text link route writes its own targets by, so a picture and a link in the
// same file cannot disagree about what a parenthesis does.
func imageLink(target string) string {
	return "![](" + view.Destination(target) + ")"
}
