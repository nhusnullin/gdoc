// The one command that reaches nothing at all.
//
// `gdoc build` turns a note into a house-style .docx on this machine. It opens
// no policy and no session, because there is no wire to judge: the house style
// is embedded in the binary, the pictures come from the note's own directory,
// and the file is written through internal/atomicfile. Publishing the result
// into Drive is `gdoc publish`, a different command, and it renders through the
// same renderNote below rather than through a second path.
//
// Nothing here decides anything either. The cover words are the note's, the
// sizes and colours are house.yaml's, and what the walker could not render is a
// warning naming the line rather than a judgement about the note.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gdoc/internal/atomicfile"
	"gdoc/internal/body"
	"gdoc/internal/cover"
	"gdoc/internal/emit"
	"gdoc/internal/house"
	"gdoc/internal/render"
)

// embeddedHouse is what data.house says when the run used the style inside the
// binary. A run given --house names that file instead, so a document built from
// a draft style says which one it came from.
const embeddedHouse = "embedded"

// buildMode is the mode a new .docx is written with. An existing file keeps its
// own, which is atomicfile.ModeOf's rule: a file somebody made group readable
// stays that way.
const buildMode = 0o644

// buildData is what `gdoc build` prints: the file it wrote, the words it took
// off the note, the style it read, and what the walker counted. Every field is
// a fact. Whether the document is right is read in Word, not here.
type buildData struct {
	Out         string      `json:"out"`
	Bytes       int64       `json:"bytes"`
	Title       string      `json:"title"`
	RunningHead string      `json:"running_head"`
	House       string      `json:"house"`
	Body        body.Counts `json:"body"`
}

func cmdBuild(a *args) emit.Result {
	md, err := required(a, "--md")
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	out, err := required(a, "--out")
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	// Before anything is read or rendered. Not knowing must never resolve to
	// overwrite, and the file that is already there is somebody's.
	if err := freeToWrite(out, a.has("--force")); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	// --force is consent to replace the document being written, never consent
	// to replace what the run was told to read.
	if err := notAnInput(out, input{"--md", md}, input{"--house", a.flags["--house"]}); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}

	source, err := noteSource(md)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	doc, err := renderNote(source, md, a)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	// The note's pictures are inputs too, and which files they are is only
	// known once the walk has read them. Asked before the walk, the check
	// covered the two flags and left `--out diagram.png --force` replacing the
	// picture it had just embedded. Nothing has been written yet, because the
	// render zips into memory.
	if err := notAnInput(out, pictures(doc.Sources)...); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if err := atomicfile.Replace(out, doc.Docx, atomicfile.ModeOf(out, buildMode)); err != nil {
		return emit.Result{OK: false, Error: fmt.Sprintf("the document could not be written: %v", err),
			Warnings: doc.Warnings}
	}

	return emit.Result{OK: true, Warnings: doc.Warnings, Data: buildData{
		Out:         absolute(out),
		Bytes:       int64(len(doc.Docx)),
		Title:       doc.Title,
		RunningHead: doc.RunningHead,
		House:       doc.House,
		Body:        doc.Counts,
	}}
}

// noteDocx is one note rendered: the bytes, the words the cover took off it,
// the style it was built from, what the walker counted, the pictures it read
// and what it could not render.
//
// Both commands that turn a note into a document print from this one struct.
// build writes the bytes to a file and publish uploads them, and two renders
// written twice would be two documents that slowly stopped being the same one.
type noteDocx struct {
	Docx        []byte
	Title       string
	RunningHead string
	House       string
	Counts      body.Counts
	Sources     []string
	Warnings    []string
}

// noteSource reads the note both commands start from. It is one function because
// publish reads the bytes for its own reason as well: the block it appends its
// entry to is in them, and the same bytes are what the re-read after the upload
// is compared against.
func noteSource(md string) ([]byte, error) {
	source, err := os.ReadFile(md)
	if err != nil {
		return nil, fmt.Errorf("the note could not be read: %v", err)
	}
	return source, nil
}

// renderNote turns the note into docx bytes. It reaches nothing and writes
// nothing: the house style is embedded or named by --house, the pictures come
// from the note's own directory, and the package is zipped into memory.
//
// Zipping whole before the caller does anything with the bytes is deliberate. A
// package that cannot be serialised leaves no half a file behind under the name
// somebody asked for, and no half a document in somebody's Drive.
func renderNote(source []byte, md string, a *args) (*noteDocx, error) {
	fields, markdown, err := cover.Read(source)
	if err != nil {
		return nil, errors.New(titleError(err, string(markdown), md))
	}
	cfg, style, err := readHouse(a)
	if err != nil {
		return nil, err
	}
	// The note's own directory, because a picture in a note is written relative
	// to the note rather than to wherever the command was run from.
	walked, err := body.Render(cfg, keepLines(source, markdown), filepath.Dir(md), fields.HeadingNumbering)
	if err != nil {
		return nil, err
	}
	pkg, err := render.Build(cfg, fields, walked.Blocks, walked.Media, walked.NumberedLists)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := pkg.Write(&buf); err != nil {
		return nil, err
	}
	return &noteDocx{
		Docx:        buf.Bytes(),
		Title:       fields.CoverTitle(),
		RunningHead: fields.RunningHead(),
		House:       style,
		Counts:      walked.Counts,
		Sources:     walked.Sources,
		Warnings:    walked.Warnings,
	}, nil
}

// freeToWrite refuses a file that is already there unless --force says
// otherwise. A directory is refused whatever the flag says: --force is somebody
// agreeing to replace a document, not to replace a folder.
func freeToWrite(out string, force bool) error {
	info, err := os.Stat(out)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s could not be looked at: %w", out, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, and --out names the file to write", out)
	}
	if !force {
		return fmt.Errorf("%s is already there. Add --force to replace it", out)
	}
	return nil
}

// input is one file this run reads, and what to call it in a refusal.
type input struct {
	name string
	path string
}

// pictures names the note's own pictures as inputs.
func pictures(paths []string) []input {
	inputs := make([]input, 0, len(paths))
	for _, path := range paths {
		inputs = append(inputs, input{"a picture the note names", path})
	}
	return inputs
}

// notAnInput refuses an --out that names a file this run reads.
//
// `--out note.md --force` read the note, rendered it, and then replaced it with
// the .docx, so the source the document was built from was gone the moment it
// was built. `--out diagram.png --force` is the same thing one step along: the
// picture is embedded and then written over, so the note points at a .docx from
// then on. --force says a document may be replaced; nothing says the note, the
// style file or a picture may be. Two paths spelled differently are compared by
// os.SameFile rather than by their text, because "note.md" and "./note.md" are
// one file and a symlink into another directory is too.
func notAnInput(out string, inputs ...input) error {
	written, err := os.Stat(out)
	if err != nil {
		// Nothing is there to alias. A path that cannot be looked at was
		// already refused by freeToWrite.
		return nil
	}
	for _, in := range inputs {
		if in.path == "" {
			continue
		}
		read, err := os.Stat(in.path)
		if err != nil {
			continue
		}
		if os.SameFile(written, read) {
			return fmt.Errorf("--out names the same file as %s (%s), and a build would replace what it reads", in.name, in.path)
		}
	}
	return nil
}

// readHouse returns the style this run builds from and what to call it. The
// file is parsed under the same strict rules as the embedded one, so a draft
// that has drifted is refused naming its own path rather than half read.
func readHouse(a *args) (*house.Config, string, error) {
	path := a.flags["--house"]
	if !a.has("--house") {
		cfg, err := house.Load()
		return cfg, embeddedHouse, err
	}
	cfg, err := house.LoadFile(path)
	if err != nil {
		return nil, "", err
	}
	return cfg, path, nil
}

// titleError names a candidate drawn from the file name as well as from the
// body. cover.Read is handed the note's bytes and not its path, so it can only
// offer the first heading; the command knows the name the note is saved under,
// which is the better proposal for a note that carries no heading either.
//
// The words are cover.MissingTitle's own, so the two readings of this failure
// cannot drift into two sentences.
func titleError(err error, markdown, path string) string {
	var missing *cover.MissingTitle
	if !errors.As(err, &missing) {
		return err.Error()
	}
	candidate, source := cover.TitleCandidate(markdown, path)
	return (&cover.MissingTitle{Candidate: candidate, Source: source}).Error()
}

// keepLines hands the walker the body with the front matter blanked out rather
// than cut away.
//
// body.Render counts lines from the markdown it is given, and the author counts
// them from the top of the file. Cut away, a warning about the note's first code
// block names a line several above it, and the deeper the front matter the
// further off it points. Blank lines in front of the first block change nothing
// goldmark does with the rest.
func keepLines(source, markdown []byte) []byte {
	head := len(source) - len(markdown)
	if head <= 0 {
		return markdown
	}
	return append(bytes.Repeat([]byte("\n"), bytes.Count(source[:head], []byte("\n"))), markdown...)
}

// absolute is the path a caller can act on. A relative --out is resolved
// against the working directory the run had, which the JSON object outlives.
func absolute(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
