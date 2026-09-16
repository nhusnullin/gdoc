package frontmatter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

// key is the one top-level front-matter key this package owns.
const key = "gdoc"

// delimiters open and close YAML front matter. "..." closes a YAML document
// too, and a file that uses it is still front matter.
const (
	openDelimiter  = "---"
	closeAlternate = "..."
)

// bom is the UTF-8 byte order mark. An editor on Windows writes one in front of
// the opening delimiter, and the front matter behind it is still front matter.
const bom = "\uFEFF"

// wrapper carries the block under its key, so the strict decode sees the same
// shape the file has.
type wrapper struct {
	Gdoc *Block `yaml:"gdoc"`
}

// document is a markdown file cut into lines, with the front matter and the
// gdoc: span located. Every index is into lines, and every line carries its own
// terminator, so joining a slice of them reproduces the bytes exactly.
type document struct {
	lines       []string
	hasFront    bool
	openNoClose bool // line 1 opens the front matter and nothing closes it
	closeAt     int  // index of the closing delimiter line
	gdocStart   int  // -1 when the file has no gdoc: key
	gdocEnd     int  // one past the last line of the span
	eol         string
}

// Read returns the gdoc: block, or nil when the file has no front matter or the
// front matter has no gdoc: key. A block that is present and wrong is an error
// naming the key, never a block with useful-looking zeros in it.
func Read(src []byte) (*Block, error) {
	d, err := parse(src)
	if err != nil {
		return nil, err
	}
	return readFrom(d)
}

// readFrom is Read once the file has been cut into lines. Write shares it, so
// the file is parsed once and the two functions can never disagree about where
// the span is.
func readFrom(d *document) (*Block, error) {
	if !d.hasFront {
		return nil, nil
	}

	// Before the gdoc: span is looked for, not after. Write's only guard against
	// rewriting front matter it does not understand is this function, and a file
	// being paired for the first time has no gdoc: key at all: checking only
	// when one is already there is checking every case but the first.
	if err := checkFrontMatter(d); err != nil {
		return nil, err
	}

	if d.gdocStart < 0 {
		return nil, nil
	}

	span := normalize(strings.Join(d.lines[d.gdocStart:d.gdocEnd], ""))
	var w wrapper
	if err := yaml.UnmarshalWithOptions([]byte(span), &w, yaml.Strict()); err != nil {
		if id, ok := barePairing(span); ok {
			return nil, barePairingRefusal(id)
		}
		return nil, fmt.Errorf("gdoc front matter: %w", err)
	}
	if w.Gdoc == nil {
		return nil, fmt.Errorf("gdoc front matter: the %s: key is empty", key)
	}
	if err := w.Gdoc.Validate(); err != nil {
		return nil, err
	}
	return w.Gdoc, nil
}

// barePairing reports the id when the gdoc: key holds a plain string rather
// than the block this package reads. It is asked only once the strict read has
// already refused the span, so it costs nothing on a block that reads.
//
// A key holding any other scalar is not that shape and does not get its
// sentence: telling somebody to rewrite `gdoc: 3` as a document id sends them
// the wrong way.
func barePairing(span string) (string, bool) {
	var loose struct {
		Gdoc any `yaml:"gdoc"`
	}
	if err := yaml.Unmarshal([]byte(span), &loose); err != nil {
		return "", false
	}
	id, ok := loose.Gdoc.(string)
	return id, ok
}

// barePairingRefusal is the whole migration story a person gets, because there
// is no other one. Nail decided on 2026-09-08 that the reader keeps refusing
// that shape: publish is the only command that creates the block, and the
// three that write into one refuse a note that has none, so a shape this
// reader does not know never enters it. So the refusal names what it found,
// the id inside it, and the two ways out.
func barePairingRefusal(id string) error {
	return fmt.Errorf("gdoc front matter: the %s: key is the plain string %q, and gdoc reads a block; write it as a %s: block carrying schema: %d and document_id: %s, or take the line out and pair the note again with gdoc publish",
		key, id, key, Schema, id)
}

// Write returns src with the gdoc: block replaced, added inside the existing
// front matter, or added with new delimiters. Every other byte, line endings
// and the trailing newline included, is carried through unchanged. A file whose
// own block cannot be read is refused rather than overwritten: not knowing what
// is there must never resolve to replacing it.
func Write(src []byte, b *Block) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	d, err := parse(src)
	if err != nil {
		return nil, err
	}
	if _, err := readFrom(d); err != nil {
		return nil, err
	}

	block, err := marshal(b, d.eol)
	if err != nil {
		return nil, err
	}

	if d.openNoClose {
		return nil, fmt.Errorf("front matter: line 1 opens with %s and the front matter never closes; add the closing %s, because gdoc will not write a second block in front of it", openDelimiter, openDelimiter)
	}

	var out bytes.Buffer
	switch {
	case !d.hasFront:
		// A byte order mark stays in front of everything, the new block
		// included: moving the block before it would leave the mark sitting in
		// the body, where it is a character rather than an encoding.
		rest := src
		if bytes.HasPrefix(rest, []byte(bom)) {
			out.WriteString(bom)
			rest = rest[len(bom):]
		}
		out.WriteString(openDelimiter + d.eol)
		out.Write(block)
		out.WriteString(openDelimiter + d.eol)
		out.Write(rest)
	case d.gdocStart >= 0:
		out.WriteString(strings.Join(d.lines[:d.gdocStart], ""))
		out.Write(block)
		out.WriteString(strings.Join(d.lines[d.gdocEnd:], ""))
	default:
		out.WriteString(strings.Join(d.lines[:d.closeAt], ""))
		out.Write(block)
		out.WriteString(strings.Join(d.lines[d.closeAt:], ""))
	}
	return out.Bytes(), nil
}

// marshal renders the block under its key, with the file's line endings.
func marshal(b *Block, eol string) ([]byte, error) {
	out, err := render(b)
	if err != nil {
		return nil, err
	}
	// The block is read back before any caller writes it. A block that does not
	// survive its own round trip is a note gdoc would corrupt and then refuse to
	// touch, because Write reads the block it finds before replacing it. So it
	// fails here, where the file on disk is still untouched.
	if err := verify(out, b); err != nil {
		return nil, err
	}
	if !bytes.HasSuffix(out, []byte("\n")) {
		out = append(out, '\n')
	}
	if eol != "\n" {
		out = bytes.ReplaceAll(out, []byte("\n"), []byte(eol))
	}
	return out, nil
}

// render is the block as YAML, before the line endings are put back.
//
// A string carrying a control character is written double quoted, because the
// emitter writes it as a plain scalar it cannot read back: a tab inside one is
// dropped on the way in, and a bare carriage return produces a block that fails
// to parse at all. The snapshot carries a suggestion's own words, and Google
// Docs puts a tab in a text run wherever the author typed one, so what goes in
// has to come back out.
func render(b *Block) ([]byte, error) {
	out, err := yaml.MarshalWithOptions(wrapper{Gdoc: b},
		yaml.Indent(2), yaml.IndentSequence(true), yaml.CustomMarshaler[string](quoteControls))
	if err != nil {
		return nil, fmt.Errorf("gdoc front matter: %w", err)
	}
	return out, nil
}

// quoteControls is the one string rule render adds. A string with no control
// character in it is left to the emitter's own choice, so the block reads the
// way it always has.
func quoteControls(s string) ([]byte, error) {
	if strings.IndexFunc(s, isControl) < 0 {
		return yaml.Marshal(s)
	}
	return []byte(strconv.Quote(s)), nil
}

func isControl(r rune) bool { return r < 0x20 || r == 0x7f }

// verify refuses a block that does not survive its own round trip. It reads the
// rendered YAML back and renders it again: anything the emitter wrote and the
// parser then read differently shows up as different bytes.
func verify(out []byte, b *Block) error {
	var back wrapper
	if err := yaml.UnmarshalWithOptions(out, &back, yaml.Strict()); err != nil {
		return fmt.Errorf("gdoc front matter: the block gdoc rendered does not read back, so the file is left as it was: %w", err)
	}
	if back.Gdoc == nil {
		return errors.New("gdoc front matter: the block gdoc rendered read back empty, so the file is left as it was")
	}
	again, err := render(back.Gdoc)
	if err != nil {
		return err
	}
	if !bytes.Equal(again, out) {
		return errors.New("gdoc front matter: the block gdoc rendered does not read back as what it was given, so the file is left as it was")
	}
	return nil
}

// checkFrontMatter reads the whole front matter loosely, so YAML the author's
// own keys break, a duplicate key among them included, is refused before Write
// rewrites the file around it. The strict decode further up sees the gdoc: span
// alone and cannot see any of that.
//
// It is one YAML document by construction: the front matter closes at the first
// --- or ... line, which is where a second document would have begun.
func checkFrontMatter(d *document) error {
	body := normalize(strings.Join(d.lines[1:d.closeAt], ""))
	var loose any
	if err := yaml.NewDecoder(strings.NewReader(body)).Decode(&loose); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("front matter: %w", err)
	}
	return nil
}

// parse locates the front matter and the gdoc: span. A file with no front
// matter is not an error: it is a file that has not been paired yet.
func parse(src []byte) (*document, error) {
	d := &document{lines: splitLines(string(src)), gdocStart: -1, eol: "\n"}
	// The line endings of the gdoc: block follow the first line of the file's
	// own front matter, or of its body when it has none. Reading them off the
	// whole file instead put CRLF inside an LF block whenever one CRLF sat
	// anywhere in the prose, which is gdoc's own block churning the endings of
	// a file it promises to leave byte for byte.
	if len(d.lines) > 0 && strings.HasSuffix(d.lines[0], "\r\n") {
		d.eol = "\r\n"
	}
	if len(d.lines) == 0 || !isDelimiter(strings.TrimPrefix(d.lines[0], bom), openDelimiter) {
		return d, nil
	}

	d.closeAt = -1
	for i := 1; i < len(d.lines); i++ {
		if isDelimiter(d.lines[i], openDelimiter) || isDelimiter(d.lines[i], closeAlternate) {
			d.closeAt = i
			break
		}
	}
	if d.closeAt < 0 {
		// An opening delimiter with no closing one is not front matter, so
		// there is no block to read. It is not an unpaired note either, and
		// Write refuses it rather than putting a second block in front of the
		// author's keys.
		d.openNoClose = true
		return d, nil
	}
	d.hasFront = true

	for i := 1; i < d.closeAt; i++ {
		if !isKeyLine(d.lines[i]) {
			continue
		}
		if d.gdocStart >= 0 {
			return nil, fmt.Errorf("front matter: the %s: key is given twice, at lines %d and %d", key, d.gdocStart+1, i+1)
		}
		d.gdocStart = i
		d.gdocEnd = spanEnd(d.lines, i, d.closeAt)
	}
	return d, nil
}

// isKeyLine reports whether the line is the gdoc: key at column 0. A key is
// only this key when the colon ends it: gdoc:foo is a different key.
func isKeyLine(line string) bool {
	rest, ok := strings.CutPrefix(trim(line), key+":")
	return ok && (rest == "" || strings.HasPrefix(rest, " "))
}

// spanEnd returns one past the last line belonging to the key that starts at
// start. The span runs to the next line at column 0 that is not blank, or to
// the closing delimiter, and gives back any blank lines at its end: those
// separate the author's keys and must not move.
func spanEnd(lines []string, start, closeAt int) int {
	end := closeAt
	for i := start + 1; i < closeAt; i++ {
		t := trim(lines[i])
		if t == "" || strings.HasPrefix(t, " ") || strings.HasPrefix(t, "\t") {
			continue
		}
		end = i
		break
	}
	for end > start+1 && trim(lines[end-1]) == "" {
		end--
	}
	return end
}

// splitLines cuts text into lines that keep their terminators, so joining them
// gives the input back byte for byte.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.SplitAfter(s, "\n")
	if last := len(lines) - 1; lines[last] == "" {
		lines = lines[:last]
	}
	return lines
}

// trim drops a line's terminator, CRLF included.
func trim(line string) string {
	return strings.TrimRight(line, "\r\n")
}

// isDelimiter reports whether the line is that front-matter delimiter. Trailing
// spaces and tabs do not stop it being one: Jekyll, python-frontmatter and
// goldmark-meta all accept them, so a note written that way has front matter
// everywhere else and gdoc must not read it as unpaired. Reading it as unpaired
// is not a quiet no-op. Write then takes its "never paired" branch and puts a
// second block in front of the author's own keys, demoting them to prose, which
// is the corruption the unclosed-delimiter refusal exists to prevent.
func isDelimiter(line, delimiter string) bool {
	return strings.TrimRight(trim(line), " \t") == delimiter
}

// normalize hands the YAML decoder LF line endings, whatever the file uses.
func normalize(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
