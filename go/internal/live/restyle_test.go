package live

// M7b's acceptance, and SPEC item 5: an in-place restyle on a copy of the ideal
// test document preserves all ten features of the 2026-08-29 run, and the
// original is left exactly as it was.
//
// The ten are the person chip, the date chip with everything behind it, the
// calendar rich link, the Google Drawing, the footnotes, both lists, the table
// with its pinned shaded header, the inline image byte for byte, the anchored
// comment still enclosing its own words, and the pending suggestion still
// pending. GDOC_LIVE_IDEAL_DOC_ID names the document holding them, and there is
// no default: the copy grant is opened with exactly the id the run was given.
//
// Four things about how it runs are the point of it rather than plumbing.
//
// The copy is made through the API, with copyComments=true, under the per-run
// AllowCopy grant. Before 2026-09-09 the acceptance needed a document built by
// hand in a browser, because a copy dropped every comment and the guard refused
// one anyway.
//
// The copy holds all ten before anything is written to it, and a copy that does
// not fails the setup rather than the restyle. A run that cannot tell the two
// apart reports a Drive copy's own limits as damage the write did.
//
// The restyle runs on a second policy, with the copy handed in at LevelSuggest
// and then granted in place. The copy is at LevelFull on the first policy,
// because the guard learned it from the create it carried, and a restyle sent
// there would exercise no allowlist at all and pass with the whole milestone
// broken.
//
// The trash therefore runs on the first policy. judgeDrive carries a file PATCH
// at LevelFull alone, which the restyle policy deliberately does not have. The
// obvious fix for that refusal is to add PATCH to LevelInPlace or to restyle at
// LevelFull, and both undo this test.

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/drive"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/house"
	"gdoc/internal/restyle"
)

// idealVar names the document the ten features live in. It has no default, for
// the reason GDOC_LIVE_DOC_ID has none: it is a document somebody already owns,
// and this run copies the whole of it, comments included, into gdoc's folder.
const idealVar = "GDOC_LIVE_IDEAL_DOC_ID"

func TestLiveRestylePreservesTenFeatures(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 and %s=<document id> to copy the ideal document into the Drive test folder and restyle the copy; %s names another folder",
			liveVar, writeVar, idealVar, folderVar)
	}
	ideal := strings.TrimSpace(os.Getenv(idealVar))
	if ideal == "" {
		t.Fatalf("%s=1 with %s=1 needs %s=<document id>: the copy grant names exactly one source, and there is no default", liveVar, writeVar, idealVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}
	ctx := context.Background()

	// The first policy: the ideal document handed in to be read and copied, and
	// the one folder the copy may land in. Nothing here may write to the ideal
	// document: it is at LevelSuggest, and a restyle of it would be refused by
	// the guard before it left the machine.
	p := guard.NewPolicy()
	p.AllowFile(ideal, guard.LevelSuggest)
	p.AllowCreateIn(folder)
	p.AllowCopy(ideal)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}

	original, err := docs.Fetch(ctx, s, ideal)
	if err != nil {
		t.Fatalf("the ideal document %q could not be read: %v", ideal, err)
	}
	t.Logf("ideal document %s, revision %s, %d tabs", ideal, original.RevisionID, len(original.Tabs))
	// Registered before anything is created, so it runs on every path out of
	// this test, the failing ones included. The 2026-08-29 rule is that only
	// the copy is ever written to, and the revision id is what says so.
	t.Cleanup(func() { assertOriginalUnmoved(t, ctx, s, ideal, original.RevisionID) })

	copyID := copyDocument(t, ctx, s, ideal, folder)
	t.Logf("copy %s in folder %s", copyID, folder)
	t.Cleanup(func() {
		if err := drive.Trash(ctx, s, copyID); err != nil {
			t.Errorf("the copy %q is still in folder %s: %v", copyID, folder, err)
			return
		}
		t.Logf("copy %s trashed", copyID)
	})

	before := readEverything(t, ctx, s, copyID)
	if gaps := before.features.missing(); len(gaps) != 0 {
		t.Fatalf("the copy does not hold all ten features, so a restyle of it would measure the setup rather than the write: it has no %s",
			strings.Join(gaps, ", no "))
	}
	t.Logf("before: %s", before.features)

	requests, plan := restyleCopy(t, ctx, copyID)

	after := readEverything(t, ctx, s, copyID)
	t.Logf("after:  %s", after.features)
	for _, moved := range before.features.diff(after.features) {
		t.Errorf("the restyle did not preserve %s", moved)
	}

	// The production read-back on top of the ten, because what the binary
	// reports is what the skill reads out to Nail. It asks two more questions
	// the feature list does not: whether the style actually landed, and whether
	// a witness that answered before has stopped answering.
	rb, notes := restyle.Verify(before.report, after.input, after.raw, restyle.Sent{Confirmed: requests}, plan)
	for _, n := range notes {
		t.Logf("read-back warning: %s", n)
	}
	for _, m := range rb.Manual {
		t.Logf("manual step: %s (%s)", m.What, m.Where)
	}
	if !rb.Verified {
		t.Errorf("the read-back does not verify the restyle: preservation intact %v, threads %+v, suggestions %+v, chips %+v, landing %+v",
			rb.Preservation.Intact, rb.Preservation.Threads, rb.Preservation.Suggestions, rb.Preservation.Chips, rb.Landing)
	}
}

// restyleCopy is the write half: a fresh policy holding the copy at the level
// every handed-in document gets, the grant that raises it, and the batches.
//
// The refusal before the grant is asserted rather than assumed. Without it a
// policy that had quietly kept the copy at LevelFull, or a grant that had
// stopped being needed, would run the whole acceptance through a door this
// milestone is about closing, and every assertion below it would still pass.
func restyleCopy(t *testing.T, ctx context.Context, copyID string) ([]map[string]any, restyle.Plan) {
	t.Helper()
	p := guard.NewPolicy()
	p.AllowFile(copyID, guard.LevelSuggest)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the restyle session could not be opened: %v", err)
	}
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the house style could not be loaded: %v", err)
	}
	d, err := docs.Fetch(ctx, s, copyID)
	if err != nil {
		t.Fatalf("the copy %q could not be read before the restyle: %v", copyID, err)
	}
	if d.MultiTab() {
		t.Fatalf("the copy has %d tabs, and a restyle styles a document with one", len(d.Tabs))
	}
	plan := restyle.TabRequests(d.Tabs[0], cfg)
	requests := append([]map[string]any{restyle.PageRequest(cfg)}, plan.Requests...)

	_, err = restyle.Apply(ctx, s, copyID, requests[:1], d.RevisionID)
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("a direct edit on a handed-in document must be refused until this run grants it, and it was not: %v", err)
	}

	// The line the milestone is about. One id, this run, and it dies with the
	// process.
	p.GrantInPlace(copyID)

	applied, err := restyle.Apply(ctx, s, copyID, requests, d.RevisionID)
	for _, w := range applied.Warnings {
		t.Logf("apply warning: %s", w)
	}
	if err != nil {
		t.Fatalf("the restyle of %q stopped after %d batches: %v", copyID, applied.Batches, err)
	}
	t.Logf("restyled: %d requests in %d batches, revision %s, %d paragraphs, %d runs, %d cells, %d bulleted, %d tables",
		applied.Requests, applied.Batches, applied.RevisionID, plan.Paragraphs, plan.Text, plan.Cells, plan.Bulleted, plan.Tables)
	return requests, plan
}

// copyDocument duplicates the ideal document into the granted folder, with its
// comments and its pending suggestions. copyComments=true is what carries them,
// measured on 2026-09-09; without it Drive carries neither, which is what was
// measured in August and why this run needed a browser until then.
//
// The body names exactly one parent, the folder the run was given. A copy that
// names none lands beside the original, in a folder gdoc was never given, and
// the guard's parent check refuses it.
func copyDocument(t *testing.T, ctx context.Context, s *gapi.Session, src, folder string) string {
	t.Helper()
	var made struct {
		ID string `json:"id"`
	}
	url := "https://www.googleapis.com/drive/v3/files/" + src + "/copy?copyComments=true&fields=id&supportsAllDrives=true"
	body := map[string]any{
		"name":    "gdoc M7b acceptance copy " + time.Now().UTC().Format(time.RFC3339),
		"parents": []string{folder},
	}
	if err := s.PostJSON(ctx, url, body, &made); err != nil {
		t.Fatalf("the ideal document %q could not be copied into folder %q: %v", src, folder, err)
	}
	if made.ID == "" {
		t.Fatalf("Drive accepted the copy of %q and its answer carried no id, so there is nothing to restyle and nothing to trash", src)
	}
	return made.ID
}

// assertOriginalUnmoved re-reads the ideal document and says whether it is on
// the revision it was on before the run. It is the whole of the 2026-08-29
// rule: only the copy is written to.
func assertOriginalUnmoved(t *testing.T, ctx context.Context, s *gapi.Session, id, was string) {
	t.Helper()
	now, err := docs.Fetch(ctx, s, id)
	if err != nil {
		t.Errorf("the ideal document %q could not be read again, so nothing here says it was left alone: %v", id, err)
		return
	}
	if now.RevisionID != was {
		t.Errorf("the ideal document %q was on revision %s before the run and is on %s now: something wrote to the original", id, was, now.RevisionID)
		return
	}
	t.Logf("the ideal document %s is still on revision %s", id, was)
}

// liveRead is one document read every way this test needs it: the raw bytes the
// landing half reads the styling out of, the decoded tree, the same bytes as a
// map for the shapes internal/docs deliberately does not decode, the survey,
// and the ten features made from all of them.
type liveRead struct {
	raw      json.RawMessage
	input    restyle.Input
	report   restyle.Report
	features tenFeatures
}

// readEverything makes the survey's three reads and builds the ten features
// from them. The listing goes out before the Docs read, which is the order
// every poll in this binary holds.
//
// The Docs read is docs.URL's, and not the fidelity probe's readRaw: this one
// needs SUGGESTIONS_INLINE for the pending suggestion and the comment view mode
// for the anchor, and one read feeds all three readers so they cannot be
// looking at different documents.
func readEverything(t *testing.T, ctx context.Context, s *gapi.Session, id string) liveRead {
	t.Helper()
	raws, err := comments.Fetch(ctx, s, id, nil)
	if err != nil {
		t.Fatalf("the comment listing of %q failed: %v", id, err)
	}
	var raw json.RawMessage
	if err := s.GetJSON(ctx, docs.URL(id), &raw); err != nil {
		t.Fatalf("the Docs read of %q failed: %v", id, err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("the Docs read of %q did not parse: %v", id, err)
	}
	export, err := docx.Export(ctx, s, id)
	if err != nil {
		t.Fatalf("the docx export of %q failed, and it is the only witness of an anchor: %v", id, err)
	}
	f, err := docx.Parse(export)
	if err != nil {
		t.Fatalf("the docx export of %q did not parse: %v", id, err)
	}
	in := restyle.Input{Document: d, Comments: raws, Export: f}
	report, notes := restyle.Survey(in)
	for _, n := range notes {
		t.Logf("survey warning: %s", n)
	}
	flat, err := flattenFirstTab(raw)
	if err != nil {
		t.Fatalf("the Docs read of %q could not be walked as JSON: %v", id, err)
	}
	return liveRead{raw: raw, input: in, report: report, features: readFeatures(t, report, d, flat, export)}
}

// tenFeatures is SPEC item 5's list, read out of one document. Every field is a
// fact taken from the document, and none of them is a verdict: missing says
// which of the ten are not there, and diff says which of them moved.
type tenFeatures struct {
	// person, date and link are the three smart chips, each as the JSON Docs
	// sends for it with the look taken off. The look is taken off because the
	// restyle changes it on purpose: what has to survive is the chip's identity
	// and the data behind it, which for the date chip is its timestamp, its
	// locale and its format.
	person []string
	date   []string
	link   []string
	// drawings and images are the inline objects by kind.
	drawings int
	images   int
	// footnotes is each footnote's id and its text.
	footnotes []string
	// lists is each list a paragraph belongs to, with its glyphs and how many
	// paragraphs are in it. numbered and bulleted are whether both kinds are
	// there at all, which is the feature SPEC names.
	lists    []string
	numbered bool
	bulleted bool
	// The first table's header row: whether it is pinned, and the fill of each
	// of its cells. A restyle writes no backgroundColor, so the shading is the
	// author's and has to come through untouched.
	tableFound  bool
	tablePinned bool
	tableFills  []string
	// media is every picture in the docx export, by size and by hash. It is how
	// "byte-identical" is asked: the bytes themselves rather than a content URI
	// that is regenerated on every read.
	media []string
	// threads is each comment with its witness and the words its anchor
	// encloses. anchored is whether any of them is still attached to text,
	// which is the feature, and the witness is the docx export because
	// comments.list reports a destroyed anchor as healthy.
	threads  []string
	anchored bool
	// pending is every pending suggestion id the read could see.
	pending []string
}

func readFeatures(t *testing.T, rep restyle.Report, d *docs.Document, flat map[string]any, export []byte) tenFeatures {
	t.Helper()
	var f tenFeatures
	f.person, f.date, f.link = chips(flat)
	f.images, f.drawings = inlineObjects(flat)
	f.footnotes = footnotes(d)
	f.lists, f.numbered, f.bulleted = lists(flat)
	f.tableFound, f.tablePinned, f.tableFills = headerRow(flat)
	media, err := mediaParts(export)
	if err != nil {
		t.Errorf("the pictures could not be read out of the export, so nothing here says the image survived: %v", err)
	}
	f.media = media
	f.threads, f.anchored = threadWords(rep, d)
	f.pending = append([]string{}, rep.Suggestions.IDs...)
	return f
}

// missing names the features that are not in the document, in the words SPEC
// item 5 uses. An empty answer is a document worth restyling as an acceptance.
func (f tenFeatures) missing() []string {
	var out []string
	if len(f.person) == 0 {
		out = append(out, "person chip")
	}
	if len(f.date) == 0 {
		out = append(out, "date chip")
	}
	if len(f.link) == 0 {
		out = append(out, "rich link chip")
	}
	if f.drawings == 0 {
		out = append(out, "Google Drawing")
	}
	if len(f.footnotes) == 0 {
		out = append(out, "footnote")
	}
	if !f.numbered || !f.bulleted {
		out = append(out, "numbered and bulleted list")
	}
	if !f.tableFound || !f.tablePinned || !anyFill(f.tableFills) {
		out = append(out, "table with a pinned shaded header row")
	}
	if f.images == 0 || len(f.media) == 0 {
		out = append(out, "inline image")
	}
	if !f.anchored {
		out = append(out, "comment anchored to its words")
	}
	if len(f.pending) == 0 {
		out = append(out, "pending suggestion")
	}
	return out
}

// diff is the before against the after, one line per feature that moved. It
// names what changed and shows both sides, because a person reads this run and
// records anything that did not survive as a decision.
func (f tenFeatures) diff(g tenFeatures) []string {
	var out []string
	list := func(name string, a, b []string) {
		if strings.Join(a, "\n") == strings.Join(b, "\n") {
			return
		}
		out = append(out, fmt.Sprintf("%s: before %v, after %v", name, a, b))
	}
	count := func(name string, a, b int) {
		if a == b {
			return
		}
		out = append(out, fmt.Sprintf("%s: %d before, %d after", name, a, b))
	}
	list("the person chips", f.person, g.person)
	list("the date chips", f.date, g.date)
	list("the rich link chips", f.link, g.link)
	count("the Google Drawings", f.drawings, g.drawings)
	count("the inline images", f.images, g.images)
	list("the footnotes", f.footnotes, g.footnotes)
	list("the lists", f.lists, g.lists)
	if f.tablePinned != g.tablePinned {
		out = append(out, fmt.Sprintf("the table's pinned header: %v before, %v after", f.tablePinned, g.tablePinned))
	}
	list("the table header's shading", f.tableFills, g.tableFills)
	list("the pictures in the export", f.media, g.media)
	list("the comments and the words they enclose", f.threads, g.threads)
	list("the pending suggestions", f.pending, g.pending)
	return out
}

// String is the one line the run logs before and after. It is a summary for a
// person reading the output, and the assertions read the fields.
func (f tenFeatures) String() string {
	return fmt.Sprintf("%d person, %d date, %d link, %d drawing, %d image, %d footnote, %d list (numbered %v, bulleted %v), header pinned %v with %d fills, %d picture(s) exported, %d thread(s) (anchored %v), %d pending",
		len(f.person), len(f.date), len(f.link), f.drawings, f.images, len(f.footnotes),
		len(f.lists), f.numbered, f.bulleted, f.tablePinned, len(f.tableFills),
		len(f.media), len(f.threads), f.anchored, len(f.pending))
}

// chips reads the three smart chips as Docs sends them, with the look removed.
func chips(flat map[string]any) (person, date, link []string) {
	for _, p := range everyParagraph(flat) {
		els, _ := p["elements"].([]any)
		for _, e := range els {
			m, ok := e.(map[string]any)
			if !ok {
				continue
			}
			if v, ok := m["person"]; ok {
				person = append(person, canonical(withoutLook(v)))
			}
			if v, ok := m["dateElement"]; ok {
				date = append(date, canonical(withoutLook(v)))
			}
			if v, ok := m["richLink"]; ok {
				link = append(link, canonical(withoutLook(v)))
			}
		}
	}
	sort.Strings(person)
	sort.Strings(date)
	sort.Strings(link)
	return
}

// inlineObjects counts the pictures by kind. A Drawing is the one the docx
// routes flattened and the one this milestone exists to keep.
func inlineObjects(flat map[string]any) (images, drawings int) {
	objs, _ := flat["inlineObjects"].(map[string]any)
	for _, o := range objs {
		emb := dig(o, "inlineObjectProperties", "embeddedObject")
		switch {
		case dig(emb, "embeddedDrawingProperties") != nil:
			drawings++
		case dig(emb, "imageProperties") != nil:
			images++
		}
	}
	return
}

// footnotes is each footnote's id and its text, from the decoded document. The
// text is here because nothing on the allowlist can change a character, so a
// footnote whose words moved is a finding rather than a detail.
func footnotes(d *docs.Document) []string {
	var out []string
	for id, text := range d.Footnotes {
		out = append(out, fmt.Sprintf("%s %q", id, text))
	}
	sort.Strings(out)
	return out
}

// lists reads the lists the paragraphs actually belong to, and whether both
// kinds are there. Docs states a glyphType on an ordered level and a
// glyphSymbol on an unordered one, which is how the two are told apart.
func lists(flat map[string]any) (out []string, numbered, bulleted bool) {
	defined, _ := flat["lists"].(map[string]any)
	counts := map[string]int{}
	for _, p := range everyParagraph(flat) {
		if id, ok := dig(p, "bullet", "listId").(string); ok {
			counts[id]++
		}
	}
	for id, n := range counts {
		levels, _ := dig(defined[id], "listProperties", "nestingLevels").([]any)
		var glyphs []string
		for _, l := range levels {
			switch {
			case dig(l, "glyphType") != nil:
				glyphs = append(glyphs, fmt.Sprint(dig(l, "glyphType")))
				numbered = true
			case dig(l, "glyphSymbol") != nil:
				glyphs = append(glyphs, fmt.Sprint(dig(l, "glyphSymbol")))
				bulleted = true
			default:
				glyphs = append(glyphs, "?")
			}
		}
		out = append(out, fmt.Sprintf("%s paragraphs=%d glyphs=%v", id, n, glyphs))
	}
	sort.Strings(out)
	return
}

// headerRow reads the first table's first row: whether Docs pins it as a header
// and what each of its cells is filled with.
func headerRow(flat map[string]any) (found, pinned bool, fills []string) {
	for _, e := range content(flat) {
		tbl, ok := dig(e, "table").(map[string]any)
		if !ok {
			continue
		}
		rows, _ := tbl["tableRows"].([]any)
		if len(rows) == 0 {
			continue
		}
		found = true
		if v, ok := dig(rows[0], "tableRowStyle", "tableHeader").(bool); ok {
			pinned = v
		}
		cells, _ := dig(rows[0], "tableCells").([]any)
		for _, c := range cells {
			fills = append(fills, canonical(dig(c, "tableCellStyle", "backgroundColor")))
		}
		return
	}
	return
}

// anyFill reports whether a cell in the header row carries a fill at all. Docs
// leaves the field out of a cell nobody shaded, so a row of nulls is a header
// row with no shading in it.
func anyFill(fills []string) bool {
	for _, f := range fills {
		if f != "null" && f != "{}" {
			return true
		}
	}
	return false
}

// mediaParts is every picture in the docx export, by length and by hash. That
// is what makes "byte-identical" a question this test can ask: an image's
// contentUri in a Docs read is regenerated on every read and says nothing.
func mediaParts(export []byte) ([]string, error) {
	z, err := zip.NewReader(bytes.NewReader(export), int64(len(export)))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range z.File {
		if !strings.HasPrefix(f.Name, "word/media/") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, docx.MaxExportBytes))
		rc.Close()
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		out = append(out, fmt.Sprintf("%d bytes sha256:%s", len(b), hex.EncodeToString(sum[:])))
	}
	sort.Strings(out)
	return out, nil
}

// threadWords is each thread's witness and the words its anchor encloses. The
// witness comes from the export, never from Drive, and the words come from the
// Docs read's own anchor range: a comment reported anchored to different words
// is an anchor that moved, which the witness alone cannot see.
func threadWords(rep restyle.Report, d *docs.Document) (out []string, anchored bool) {
	for _, w := range rep.Threads.Witnessed {
		words := ""
		if r, ok := d.CommentRanges[w.ID]; ok {
			words = anchoredWords(d, r)
		}
		if w.Witness == docx.WitnessAnchored && words != "" {
			anchored = true
		}
		out = append(out, fmt.Sprintf("%s witness=%s words=%q", w.ID, w.Witness, words))
	}
	sort.Strings(out)
	return
}

// anchoredWords is the text a comment anchor encloses, sliced exactly. The
// indexes are UTF-16 code units, which is what the Docs API counts, and the
// slice is per run rather than per paragraph because a restyle merges runs: a
// whole paragraph read back where a phrase was quoted would be this test crying
// wolf over styling that did exactly what it was asked.
func anchoredWords(d *docs.Document, r docs.Range) string {
	for _, t := range d.Tabs {
		if t.ID != r.Tab {
			continue
		}
		var b strings.Builder
		spanOf(t.Body, r, &b)
		return b.String()
	}
	return ""
}

func spanOf(blocks []docs.Block, r docs.Range, b *strings.Builder) {
	for _, blk := range blocks {
		if p := blk.Paragraph; p != nil {
			for _, run := range p.Runs {
				if run.Kind != docs.KindText || run.EndIndex <= r.Start || run.StartIndex >= r.End {
					continue
				}
				units := utf16.Encode([]rune(run.Text))
				from, to := 0, len(units)
				if r.Start > run.StartIndex {
					from = r.Start - run.StartIndex
				}
				if r.End < run.EndIndex {
					to = len(units) - (run.EndIndex - r.End)
				}
				if from < 0 {
					from = 0
				}
				if to > len(units) {
					to = len(units)
				}
				if from < to {
					b.WriteString(string(utf16.Decode(units[from:to])))
				}
			}
		}
		if tbl := blk.Table; tbl != nil {
			for _, row := range tbl.Rows {
				for _, cell := range row {
					spanOf(cell.Blocks, r, b)
				}
			}
		}
	}
}

// everyParagraph is every paragraph in the tab, a table cell's included. The
// fidelity probe's own helper reads the top level alone, and a chip inside a
// cell would be a feature this run reported as absent.
func everyParagraph(flat map[string]any) []map[string]any {
	var out []map[string]any
	var walk func(cs []any)
	walk = func(cs []any) {
		for _, e := range cs {
			if p, ok := dig(e, "paragraph").(map[string]any); ok {
				out = append(out, p)
			}
			tbl, ok := dig(e, "table").(map[string]any)
			if !ok {
				continue
			}
			rows, _ := tbl["tableRows"].([]any)
			for _, r := range rows {
				cells, _ := dig(r, "tableCells").([]any)
				for _, c := range cells {
					inner, _ := dig(c, "content").([]any)
					walk(inner)
				}
			}
		}
	}
	walk(content(flat))
	return out
}

// withoutLook drops the styling from a value, so that comparing a chip before
// and after asks about the chip rather than about the restyle. Both keys are
// the look: textStyle is what updateTextStyle writes, and
// suggestedTextStyleChanges is the same thing pending.
func withoutLook(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range t {
			if k == "textStyle" || k == "suggestedTextStyleChanges" {
				continue
			}
			out[k] = withoutLook(val)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, val := range t {
			out = append(out, withoutLook(val))
		}
		return out
	}
	return v
}

// canonical is a value as one comparable string. encoding/json sorts a map's
// keys, so the same value reads the same way twice.
func canonical(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// flattenFirstTab moves the first tab's content up to the top level, which is
// where the shapes read here are documented. Every read gdoc makes carries
// includeTabsContent=true, so the legacy body is empty and the content is in
// tabs[0].documentTab.
func flattenFirstTab(raw []byte) (map[string]any, error) {
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, err
	}
	tabs, _ := d["tabs"].([]any)
	if len(tabs) == 0 {
		return d, nil
	}
	dt, ok := dig(tabs[0], "documentTab").(map[string]any)
	if !ok {
		return nil, fmt.Errorf("the first tab carries no documentTab")
	}
	for k, v := range dt {
		d[k] = v
	}
	return d, nil
}
