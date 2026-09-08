// The one command that reaches nothing at all.
//
// `gdoc build` turns a note into a house-style .docx on this machine. It opens
// no policy and no session, because there is no wire to judge: the house style
// is embedded in the binary, the pictures come from the note's own directory,
// and the file is written through internal/atomicfile. Publishing the result
// into Drive is M6's, and it is a different command.
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

func cmdBuild(raw []string) emit.Result {
	a, err := parseArgsN(raw, flagSet{
		"--md": true, "--out": true, "--house": true, "--force": false}, 0)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
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

	source, err := os.ReadFile(md)
	if err != nil {
		return emit.Result{OK: false, Error: fmt.Sprintf("the note could not be read: %v", err)}
	}
	fields, markdown, err := cover.Read(source)
	if err != nil {
		return emit.Result{OK: false, Error: titleError(err, string(markdown), md)}
	}

	cfg, style, err := readHouse(a)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}

	// The note's own directory, because a picture in a note is written relative
	// to the note rather than to wherever the command was run from.
	walked, err := body.Render(cfg, keepLines(source, markdown), filepath.Dir(md), fields.HeadingNumbering)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	pkg, err := render.Build(cfg, fields, walked.Blocks, walked.Media)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}

	// Zipped whole before anything is written, so a package that cannot be
	// serialised leaves no half a file behind under the name somebody asked for.
	var buf bytes.Buffer
	if err := pkg.Write(&buf); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if err := atomicfile.Replace(out, buf.Bytes(), atomicfile.ModeOf(out, buildMode)); err != nil {
		return emit.Result{OK: false, Error: fmt.Sprintf("the document could not be written: %v", err),
			Warnings: walked.Warnings}
	}

	return emit.Result{OK: true, Warnings: walked.Warnings, Data: buildData{
		Out:         absolute(out),
		Bytes:       int64(buf.Len()),
		Title:       fields.CoverTitle(),
		RunningHead: fields.RunningHead(),
		House:       style,
		Body:        walked.Counts,
	}}
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
