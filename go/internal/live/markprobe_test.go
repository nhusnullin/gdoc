// M18's second measurement: what Drive makes of a shaded run and of a run
// with a text colour.
//
// Part B of the milestone rests on one answer nobody here can reason out. The
// seven marks are a docx thing: six of them are w:shd w:fill on a run and the
// seventh is w:color. What a person opens is not a docx, it is what Google
// made of one, and what export later reads back is a Docs textStyle rather
// than a character style. So this uploads a document carrying one run per
// hex, reads back the colour Docs reports for each, restyles the document in
// place the way gdoc restyle does, and reads the same eight again.
//
// Like the other probes it asserts almost nothing. It fails on an upload, a
// read or a restyle that failed and on nothing else, because the sixteen
// answers are Google's and a test that asserted them would be deciding what
// it was sent to find out. What it reports goes into docs/v2/MEASURED.md by
// hand.

package live

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gdoc/internal/gapi"
)

// markProbeFixture is the built 04-minimal.md with eight runs added by hand at
// the end of the first body paragraph that carries prose.
const markProbeFixture = "testdata/mark-probe.docx"

// probeMark is one of the eight runs: the word it carries, what was written on
// it, and the value written.
//
// The hexes are literals here and in the fixture rather than read out of one
// by the other, and they are literals rather than house.yaml's palette,
// which Task 8 has not written yet: the question the probe asks is whether
// these exact values survive the import, and a test that read them from the
// file it uploaded could not tell a changed colour from a matching one.
type probeMark struct {
	label string
	kind  string
	value string
}

// The eight, in the order they stand in the paragraph. The first six are the
// six highlight hexes of the specification's palette, each as w:shd w:fill.
// The seventh is the pencil's text colour, as w:color. The eighth is Word's
// named yellow, the control: it is what the cover's placeholders already use,
// so what Drive makes of it is the shape the marks have to be told apart
// from.
var probeMarks = []probeMark{
	{"RED", "shading", "#FF6B6B"},
	{"ORANGE", "shading", "#FFB85C"},
	{"YELLOW", "shading", "#FFD500"},
	{"GREEN", "shading", "#3DD67A"},
	{"BLUE", "shading", "#5DB3FF"},
	{"PURPLE", "shading", "#BA92FF"},
	{"PENCIL", "text colour", "#9A9A9A"},
	{"WORDYELLOW", "named highlight", "yellow"},
}

// TestLiveMarkImportProbe is the gate for Part B: Tasks 8 to 11 are worth
// writing only if Drive keeps the colours as they were sent, and keeps them
// through a restyle.
//
// It creates one document in the test folder from the fixture and trashes it
// on the way out, so nothing of Nail's is in the reachable set.
func TestLiveMarkImportProbe(t *testing.T) {
	ctx, s, folder := liveWriteSession(t)

	fixture, err := os.ReadFile(markProbeFixture)
	if err != nil {
		t.Fatalf("the fixture %s could not be read: %v", markProbeFixture, err)
	}
	title := "gdoc M18 mark import probe " + time.Now().UTC().Format(time.RFC3339)
	id := uploadForDrift(t, ctx, s, folder, title, fixture)
	t.Logf("the probe document is %s", id)

	// The eight before anything is written to the document. This is the half
	// that says whether the import kept the colour.
	before := readMarkColours(t, ctx, s, id, "before the restyle")

	// One in-place restyle, the way the acceptance in restyle_test.go runs
	// one: a second policy holding the document at the level every handed-in
	// document gets, the per-run grant that raises it, and the production
	// batches. The run goes out at LevelInPlace rather than at the LevelFull
	// the upload left this document on, because that is the level a person's
	// restyle really runs at.
	requests, plan := restyleCopy(t, ctx, id)
	t.Logf("the restyle sent %d requests over %d paragraphs and %d runs", len(requests), plan.Paragraphs, plan.Text)

	// And the same eight again. This is the half that says whether the
	// styling allowlist, which can change no character, leaves a colour
	// somebody else put on a run alone.
	after := readMarkColours(t, ctx, s, id, "after the restyle")

	for _, m := range probeMarks {
		t.Logf("%s (%s %s): before %s, after %s", m.label, m.kind, m.value, before[m.label], after[m.label])
	}
}

// probeRun is one run as Docs sent it back: its text, and the two colours.
type probeRun struct {
	text       string
	background string
	foreground string
}

// runsWithColour reads one document bare and hands back every run that
// carries text, with the two colours Docs reports for it, printing each as it
// goes.
//
// The read is driftReadURL's, documents.get with no query: the legacy body is
// the first tab, which is the whole of a document converted from one docx, and
// it is the shape the colour is documented on.
//
// The acceptance in marks_test.go reads a published document through this same
// helper, so the probe and the acceptance ask Google the same question in the
// same words and a difference between them is a difference in the document.
func runsWithColour(t *testing.T, ctx context.Context, s *gapi.Session, id, when string) []probeRun {
	t.Helper()
	var answer map[string]any
	if err := s.GetJSON(ctx, driftReadURL(id), &answer); err != nil {
		t.Fatalf("the document (%s) could not be read %s: %v", id, when, err)
	}
	if len(answer) == 0 {
		t.Fatalf("the document (%s) came back as an empty answer %s", id, when)
	}

	var runs []probeRun
	for _, p := range everyParagraph(answer) {
		els, _ := p["elements"].([]any)
		for _, e := range els {
			run, ok := dig(e, "textRun").(map[string]any)
			if !ok {
				continue
			}
			// The field is content rather than text: a textRun carries its
			// characters under "content", and a probe reading "text" finds
			// every run empty and reports that the import dropped them all.
			text, _ := run["content"].(string)
			if strings.TrimSpace(text) == "" {
				continue
			}
			style := run["textStyle"]
			runs = append(runs, probeRun{
				text:       text,
				background: probeHex(dig(style, "backgroundColor")),
				foreground: probeHex(dig(style, "foregroundColor")),
			})
		}
	}
	t.Logf("%s: %d runs carrying text", when, len(runs))
	for _, r := range runs {
		t.Logf("%s: %q background %s, foreground %s", when, r.text, r.background, r.foreground)
	}
	return runs
}

// readMarkColours answers the two colours against each of the eight words the
// probe's fixture carries.
//
// A word is looked for as a run of its own first and as part of a longer run
// second, because a run Docs joined to its neighbour still carries one colour
// and is still an answer. The order matters and is not a style: YELLOW is part
// of WORDYELLOW, so a single pass on "somewhere in the run" could answer about
// the control where the yellow mark was asked for. Where the answer came from
// a longer run it says so.
func readMarkColours(t *testing.T, ctx context.Context, s *gapi.Session, id, when string) map[string]string {
	t.Helper()
	runs := runsWithColour(t, ctx, s, id, when)

	colours := func(r probeRun) string {
		return fmt.Sprintf("background %s, foreground %s", r.background, r.foreground)
	}
	out := map[string]string{}
	for _, m := range probeMarks {
		out[m.label] = "no run carries this word any more"
		for _, r := range runs {
			if strings.TrimSpace(r.text) == m.label {
				out[m.label] = colours(r)
				break
			}
		}
		if out[m.label] != "no run carries this word any more" {
			continue
		}
		for _, r := range runs {
			if !strings.Contains(r.text, m.label) {
				continue
			}
			out[m.label] = colours(r) + fmt.Sprintf(" (in a run reading %q)", r.text)
			break
		}
	}
	return out
}

// probeHex is a Docs colour as the hex a docx would have written, and the
// words "none stated" where the API left the field out. Each channel is a
// fraction of one, and a channel that is not there is zero, which is the
// API's own way of writing black.
//
// It is drift's hexOf spelled again rather than imported, because that one is
// unexported and a measurement is not a reason to widen a production
// package's surface.
func probeHex(v any) string {
	colour, ok := dig(v, "color", "rgbColor").(map[string]any)
	if !ok {
		return "none stated"
	}
	channel := func(name string) int {
		f, _ := colour[name].(float64)
		return int(f*255 + 0.5)
	}
	return fmt.Sprintf("#%02X%02X%02X", channel("red"), channel("green"), channel("blue"))
}

// TestTheMarkProbeFixtureIsADocx runs in `make test`, with neither variable
// set.
//
// The probe above uploads a file on this disk, so a fixture that lost a run,
// or carries one whose run properties are out of schema order, would otherwise
// be found by Nail in the middle of a live run, measuring nothing and saying
// the import dropped the colour.
func TestTheMarkProbeFixtureIsADocx(t *testing.T) {
	raw, err := os.ReadFile(markProbeFixture)
	if err != nil {
		t.Fatalf("the fixture %s could not be read: %v", markProbeFixture, err)
	}
	z, err := openZip(raw)
	if err != nil {
		t.Fatalf("the fixture is not a zip: %v", err)
	}
	names := partNames(z)
	for _, want := range []string{"[Content_Types].xml", "word/document.xml", "word/styles.xml"} {
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

	data, err := zipPart(z, "word/document.xml")
	if err != nil {
		t.Fatalf("word/document.xml could not be read: %v", err)
	}
	doc := string(data)

	// Each run, the property it carries and the word it carries, stated as the
	// XML the fixture really holds. The run properties are in the schema's own
	// order, w:color then w:sz and w:szCs then w:highlight then w:shd, which
	// is what Word opens and what Task 4 writes.
	for _, c := range []struct{ want, why string }{
		{`<w:shd w:val="clear" w:color="auto" w:fill="FF6B6B"/></w:rPr><w:t xml:space="preserve"> RED</w:t>`, "the red run is not shaded red"},
		{`<w:shd w:val="clear" w:color="auto" w:fill="FFB85C"/></w:rPr><w:t xml:space="preserve"> ORANGE</w:t>`, "the orange run is not shaded orange"},
		{`<w:shd w:val="clear" w:color="auto" w:fill="FFD500"/></w:rPr><w:t xml:space="preserve"> YELLOW</w:t>`, "the yellow run is not shaded yellow"},
		{`<w:shd w:val="clear" w:color="auto" w:fill="3DD67A"/></w:rPr><w:t xml:space="preserve"> GREEN</w:t>`, "the green run is not shaded green"},
		{`<w:shd w:val="clear" w:color="auto" w:fill="5DB3FF"/></w:rPr><w:t xml:space="preserve"> BLUE</w:t>`, "the blue run is not shaded blue"},
		{`<w:shd w:val="clear" w:color="auto" w:fill="BA92FF"/></w:rPr><w:t xml:space="preserve"> PURPLE</w:t>`, "the purple run is not shaded purple"},
		{`<w:color w:val="9A9A9A"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr><w:t xml:space="preserve"> PENCIL</w:t>`, "the pencil run does not carry the grey text colour, or carries it out of schema order"},
		{`<w:highlight w:val="yellow"/></w:rPr><w:t xml:space="preserve"> WORDYELLOW</w:t>`, "the control run does not carry Word's named yellow"},
	} {
		if !strings.Contains(doc, c.want) {
			t.Errorf("word/document.xml does not carry %q, so %s", c.want, c.why)
		}
	}

	// The eight are one paragraph's, and it is the first body paragraph that
	// carries prose. A fixture whose runs drifted into the cover or into a
	// control table would measure the template rather than the marks.
	anchor := "look right anyway.</w:t></w:r>"
	at := strings.Index(doc, anchor)
	if at < 0 {
		t.Fatalf("the fixture's first body paragraph of prose is not there, so the eight runs are somewhere else")
	}
	tail := doc[at+len(anchor):]
	end := strings.Index(tail, "</w:p>")
	if end < 0 {
		t.Fatalf("the paragraph the eight runs were added to does not close")
	}
	for _, m := range probeMarks {
		if !strings.Contains(tail[:end], "> "+m.label+"</w:t>") {
			t.Errorf("the run carrying %s is not in the first body paragraph of prose", m.label)
		}
	}
}
