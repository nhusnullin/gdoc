// M18's first measurement: what Drive makes of a Word footnote.
//
// Part A of the milestone rests on one answer nobody here can reason out.
// gdoc can write word/footnotes.xml, and Word reads it, but what a person
// opens is not a docx: it is what Google made of one. So this uploads a
// document that holds one real footnote, reads back what Drive made, and
// prints three things: whether the body carries a footnoteReference, whether
// the document carries a footnotes map holding the note's own words, and
// whether an export of that document brings word/footnotes.xml back.
//
// Like the other probes it asserts almost nothing. It fails on a read or an
// export that failed and on nothing else, because the three answers are
// Google's and a test that asserted them would be deciding what it was sent
// to find out. What it reports goes into docs/v2/MEASURED.md by hand.

package live

import (
	"archive/zip"
	"bytes"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"gdoc/internal/docx"
)

// footnoteProbeFixture is the built 04-minimal.md with word/footnotes.xml
// added by hand: one note, the reference run at the end of the first body
// paragraph that carries prose, and the relationship and content type rows
// that make the part part of the package.
const footnoteProbeFixture = "testdata/footnote-probe.docx"

// probeFootnoteText is the note's own words, and it is a literal here and in
// the fixture rather than read out of one by the other: the question the
// probe asks is whether these words survive the import, and a test that read
// them from the file it uploaded could not tell a changed note from a
// matching one.
const probeFootnoteText = "The probe footnote: carried into Drive by hand on 14 October 2026."

// The three parts of the fixture a reader has to find before the upload is
// worth making.
const (
	footnotesPart        = "word/footnotes.xml"
	footnotesContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.footnotes+xml"
	footnotesRelType     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/footnotes"
)

// TestLiveFootnoteImportProbe is the gate for Part A: Tasks 5 to 7 are worth
// writing only if Drive keeps the footnote.
//
// It creates one document in the test folder from the fixture and trashes it
// on the way out, so nothing of Nail's is in the reachable set.
func TestLiveFootnoteImportProbe(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	fixture, err := os.ReadFile(footnoteProbeFixture)
	if err != nil {
		t.Fatalf("the fixture %s could not be read: %v", footnoteProbeFixture, err)
	}
	title := "gdoc M18 footnote import probe " + time.Now().UTC().Format(time.RFC3339)
	id := uploadForDrift(t, ctx, s, folder, title, fixture)
	t.Logf("the probe document is %s", id)

	var answer map[string]any
	if err := s.GetJSON(ctx, driftReadURL(id), &answer); err != nil {
		t.Fatalf("the probe document (%s) could not be read back: %v", id, err)
	}
	if len(answer) == 0 {
		t.Fatalf("the probe document (%s) came back as an empty answer", id)
	}

	// Answer 1: is the reference still in the body?
	references := countKey(answer["body"], "footnoteReference")
	t.Logf("answer 1: the body holds %d footnoteReference elements", references)

	// Answer 2: is there a footnotes map, and does it hold the note's words?
	footnotes, ok := answer["footnotes"].(map[string]any)
	switch {
	case !ok:
		t.Logf("answer 2: the document carries no footnotes map at all")
	default:
		ids := make([]string, 0, len(footnotes))
		for key := range footnotes {
			ids = append(ids, key)
		}
		sort.Strings(ids)
		t.Logf("answer 2: the document carries a footnotes map of %d: %s", len(footnotes), strings.Join(ids, ", "))
		words := strings.Join(stringsIn(answer["footnotes"]), " ")
		if strings.Contains(words, probeFootnoteText) {
			t.Logf("answer 2: the note's own words came back")
		} else {
			t.Logf("answer 2: the note's words are not in the map; what is there reads %q", words)
		}
	}

	// Answer 3: does an export bring the part back?
	raw, err := docx.Export(ctx, s, id)
	if err != nil {
		t.Fatalf("the docx export of the probe document (%s) failed: %v", id, err)
	}
	z, err := openZip(raw)
	if err != nil {
		t.Fatalf("the exported docx of %s could not be opened: %v", id, err)
	}
	names := partNames(z)
	held := false
	for _, name := range names {
		if name == footnotesPart {
			held = true
		}
	}
	t.Logf("answer 3: the export holds %s: %t; its %d parts are %s", footnotesPart, held, len(names), strings.Join(names, ", "))
	if held {
		text, err := zipPart(z, footnotesPart)
		if err != nil {
			t.Fatalf("the exported %s could not be read: %v", footnotesPart, err)
		}
		t.Logf("answer 3: the exported %s carries the note's words: %t",
			footnotesPart, strings.Contains(string(text), probeFootnoteText))
	}
}

// TestTheFootnoteProbeFixtureIsADocx runs in `make test`, with neither
// variable set.
//
// The probe above uploads a file on this disk, so a fixture that lost its
// footnotes part, its relationship or its content type row would otherwise be
// found by Nail in the middle of a live run, measuring nothing and saying the
// import dropped the note.
func TestTheFootnoteProbeFixtureIsADocx(t *testing.T) {
	raw, err := os.ReadFile(footnoteProbeFixture)
	if err != nil {
		t.Fatalf("the fixture %s could not be read: %v", footnoteProbeFixture, err)
	}
	z, err := openZip(raw)
	if err != nil {
		t.Fatalf("the fixture is not a zip: %v", err)
	}
	names := partNames(z)
	for _, want := range []string{"[Content_Types].xml", "word/document.xml", footnotesPart} {
		found := false
		for _, name := range names {
			if name == want {
				found = true
			}
		}
		if !found {
			t.Errorf("the fixture holds no %s; its parts are %s", want, strings.Join(names, ", "))
		}
	}

	for _, c := range []struct{ part, want, why string }{
		{"[Content_Types].xml", footnotesContentType, "nothing says what the footnotes part is"},
		{"word/_rels/document.xml.rels", footnotesRelType, "the document points at no footnotes part"},
		{"word/document.xml", `<w:footnoteReference w:id="1"/>`, "the body references no footnote"},
		{footnotesPart, `<w:footnote w:id="1">`, "the footnotes part holds no note numbered 1"},
		{footnotesPart, `w:type="separator"`, "the footnotes part has no separator"},
		{footnotesPart, `w:type="continuationSeparator"`, "the footnotes part has no continuation separator"},
		{footnotesPart, probeFootnoteText, "the note does not carry the words the probe looks for"},
	} {
		data, err := zipPart(z, c.part)
		if err != nil {
			t.Errorf("%s could not be read: %v", c.part, err)
			continue
		}
		if !strings.Contains(string(data), c.want) {
			t.Errorf("%s does not carry %q, so %s", c.part, c.want, c.why)
		}
	}
}

// countKey counts how many times one key appears anywhere under a decoded
// JSON value. The Docs answer nests a paragraph element many levels down and
// the probe asks whether it is there at all, not where.
func countKey(v any, key string) int {
	switch t := v.(type) {
	case map[string]any:
		n := 0
		for k, child := range t {
			if k == key {
				n++
			}
			n += countKey(child, key)
		}
		return n
	case []any:
		n := 0
		for _, child := range t {
			n += countKey(child, key)
		}
		return n
	}
	return 0
}

// stringsIn is every string under a decoded JSON value, in no order anyone
// should rely on. It is how the probe asks whether the note's words are
// anywhere in the footnotes map without knowing the shape Google chose.
func stringsIn(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case map[string]any:
		var out []string
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, stringsIn(t[k])...)
		}
		return out
	case []any:
		var out []string
		for _, child := range t {
			out = append(out, stringsIn(child)...)
		}
		return out
	}
	return nil
}

// openZip opens one archive held in memory. zipPart beside it, in
// export_measure_test.go, reads a member out of it.
func openZip(raw []byte) (*zip.Reader, error) {
	return zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
}

// partNames lists one archive's parts in the order they are stored.
func partNames(z *zip.Reader) []string {
	names := make([]string, 0, len(z.File))
	for _, f := range z.File {
		names = append(names, f.Name)
	}
	return names
}
