package live

// M13's measurement, and nothing else. Export has to pair a picture in the
// docx with the object the Docs read names, and neither answer says which
// picture the other one means. Three questions decide how that pairing is
// written, and all three are Google's to answer:
//
//  1. Does the docx carry the pictures in the same order the Docs read walks
//     them, counting a floating picture and a Google Drawing?
//  2. Are the bytes in word/media/ the bytes that were uploaded?
//  3. What does the docx export of a two-tab document hold for the second tab?
//
// So this test reads one document Nail made by hand, through both routes, and
// prints the two lists side by side. It asserts nothing about what it finds:
// a measurement that fails the build when Google answers differently has
// already decided the answer. It fails when a read or an export fails, and
// that is all.
//
// The answers become MEASURED.md rows, and the recordings this test saves
// under GDOC_LIVE_RECORD become the export package's fixtures.

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

// The three documents this measurement reads. Each is made by hand, each is
// named by its own variable, and none has a default: the guard is opened with
// exactly the document the run named, the way every other read here is.
const (
	exportDocVar    = "GDOC_LIVE_EXPORT_DOC_ID"
	publishedDocVar = "GDOC_LIVE_PUBLISHED_DOC_ID"
	restyledDocVar  = "GDOC_LIVE_RESTYLED_DOC_ID"
)

// Where a recording lands. These are the export package's testdata folders,
// and that package does not exist yet: Task 7 reads what this test writes.
const (
	measuredDir = "../export/testdata/fixture-measured"
	publishDir  = "../export/testdata/publish-prelude"
	restyleDir  = "../export/testdata/restyle-prelude"
)

// picturesNoteDir holds the pictures the fixture document was uploaded from,
// which is what makes question 2 answerable at all: the bytes on this disk are
// the bytes that went up, so a hash that matches says Drive gave back what it
// was given.
const picturesNoteDir = "../body/testdata/docs"

// TestLiveExportMeasurements reads one document both ways and prints what each
// route says about its pictures.
//
// The document is made by hand in the test folder and holds, in this order:
// one inline PNG picture uploaded from the pictures note under
// ../body/testdata/docs, one Google Drawing, one floating picture, and a
// second tab titled Appendix with one picture in it. It creates nothing and it
// writes nothing to Drive.
func TestLiveExportMeasurements(t *testing.T) {
	if os.Getenv(liveVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=<document id> to measure one real document through both routes", liveVar, exportDocVar)
	}
	id := strings.TrimSpace(os.Getenv(exportDocVar))
	if id == "" {
		t.Skipf("skipped: %s names the hand-made fixture document, and there is no default", exportDocVar)
	}

	p := guard.NewPolicy()
	p.AllowFile(id, guard.LevelSuggest)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	ctx := context.Background()

	// The raw answer, because the question here is what Google sends. The docs
	// package carries the object ids now, in Tab.Positioned and
	// Paragraph.Positioned, and export.Objects reads them into the same list
	// measuredTabs builds by hand: walking the bytes as they came is what keeps
	// this measurement independent of the walk it measures against.
	var raw json.RawMessage
	if err := s.GetJSON(ctx, docs.URL(id), &raw); err != nil {
		t.Fatalf("the Docs read failed: %v", err)
	}
	d, err := docs.Parse(raw)
	if err != nil {
		t.Fatalf("the Docs read did not parse: %v", err)
	}
	t.Logf("document %q, %d tabs, revision %s", d.Title, len(d.Tabs), d.RevisionID)

	tabs, err := measuredTabs(raw)
	if err != nil {
		t.Fatalf("the Docs read could not be walked for its objects: %v", err)
	}

	export, err := docx.Export(ctx, s, id)
	if err != nil {
		t.Fatalf("the docx export failed: %v", err)
	}
	if _, err := docx.Parse(export); err != nil {
		t.Fatalf("the docx export did not parse: %v", err)
	}
	media, drawings, err := measuredMedia(export)
	if err != nil {
		t.Fatalf("the docx export could not be walked for its pictures: %v", err)
	}

	// Measurement 1: the two orders, side by side. A line per position, the
	// Docs object on the left and the media part on the right, so a route that
	// skips a picture shows up as the line where the two stop agreeing.
	var objects []measuredObjectRef
	for _, tab := range tabs {
		for _, o := range tab.Objects {
			objects = append(objects, o)
		}
	}
	t.Logf("measurement 1: %d objects in the Docs read, %d media parts in the docx, %d w:drawing elements",
		len(objects), len(media), drawings)
	for i := 0; i < len(objects) || i < len(media); i++ {
		left := "none"
		if i < len(objects) {
			o := objects[i]
			left = fmt.Sprintf("%s %s %s (%s)", o.Tab, o.Placement, o.ID, o.Shape)
		}
		right := "none"
		if i < len(media) {
			right = fmt.Sprintf("%s %s %s", media[i].Rel, media[i].Part, media[i].SHA256[:12])
		}
		t.Logf("  %2d  docs: %-48s docx: %s", i+1, left, right)
	}

	// Measurement 2: the bytes. Every picture beside the pictures note is
	// hashed, and every media part is looked up in that set, so the answer
	// names the file rather than saying yes or no.
	uploaded, err := uploadedHashes()
	if err != nil {
		t.Fatalf("the pictures beside the note could not be read: %v", err)
	}
	t.Logf("measurement 2: %d pictures on disk under %s", len(uploaded), picturesNoteDir)
	for i, m := range media {
		if name, ok := uploaded[m.SHA256]; ok {
			t.Logf("  %2d  %s is byte for byte %s", i+1, m.Part, name)
			continue
		}
		t.Logf("  %2d  %s matches no picture on disk, sha256 %s", i+1, m.Part, m.SHA256)
	}
	if len(media) > 0 {
		name, ok := uploaded[media[0].SHA256]
		t.Logf("measurement 2, the first media part: equal to an uploaded picture: %t (%s)", ok, name)
	}

	// Measurement 3: the second tab. The docx is one word/document.xml for the
	// whole export, so what is counted here is the total against each tab's
	// own count: a second tab that never reached the export shows as a total
	// that holds the first tab alone.
	t.Logf("measurement 3: %d w:drawing in the docx, against the tabs:", drawings)
	for i, tab := range tabs {
		t.Logf("  tab %d %s %q: %d objects (%d inline, %d floating)",
			i+1, tab.ID, tab.Title, len(tab.Objects), tab.Inline, tab.Floating)
	}

	if os.Getenv(recordVar) != "1" {
		t.Logf("set %s=1 to save this read and this export as the export package's fixtures", recordVar)
		return
	}
	recordAt(t, measuredDir, "docs-read.json", indented(t, raw))
	recordAt(t, measuredDir, "export.docx", export)
	recordPrelude(t, publishedDocVar, publishDir)
	recordPrelude(t, restyledDocVar, restyleDir)
}

// recordPrelude saves one more document's Docs read, for the strip Task 7
// writes. The two documents are a publish's and a restyle's, both made by hand
// in the test folder, so the strip rests on the preludes gdoc really writes
// rather than on a prelude a test wrote for itself.
func recordPrelude(t *testing.T, variable, dir string) {
	t.Helper()
	id := strings.TrimSpace(os.Getenv(variable))
	if id == "" {
		t.Logf("%s names no document, so %s keeps whatever it holds", variable, dir)
		return
	}
	p := guard.NewPolicy()
	p.AllowFile(id, guard.LevelSuggest)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session for %s could not be opened: %v", variable, err)
	}
	var raw json.RawMessage
	if err := s.GetJSON(context.Background(), docs.URL(id), &raw); err != nil {
		t.Fatalf("the Docs read of %s failed: %v", variable, err)
	}
	recordAt(t, dir, "docs-read.json", indented(t, raw))
}

// recordAt saves one answer under dir. What it writes is a real document's
// real content: it is read and redacted by a person before it is committed,
// and the log line says so rather than leaving it to be discovered in a diff.
func recordAt(t *testing.T, dir, name string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("recorded %s: this is a real document's content, so redact it before committing it", path)
}

// uploadedHashes is every picture beside the pictures note, by sha256. The
// note's own pictures are what the fixture document was built from, so this is
// the one set a media part can be compared against.
func uploadedHashes() (map[string]string, error) {
	entries, err := os.ReadDir(picturesNoteDir)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".png", ".jpg", ".jpeg":
		default:
			continue
		}
		b, err := os.ReadFile(filepath.Join(picturesNoteDir, e.Name()))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		out[hex.EncodeToString(sum[:])] = e.Name()
	}
	return out, nil
}

// measuredObjectRef is one picture as the Docs read names it: which tab it is
// in, whether it sits in the text or floats beside it, its object id, and what
// the embedded object turned out to be.
type measuredObjectRef struct {
	Tab       string
	Placement string
	ID        string
	Shape     string
}

// measuredTab is one tab's objects in body order, with the two counts the
// third measurement prints.
type measuredTab struct {
	ID       string
	Title    string
	Objects  []measuredObjectRef
	Inline   int
	Floating int
}

// measuredMediaFile is one picture as the docx names it: the relationship the
// body referred to, the part it resolved to, and the hash of its bytes.
type measuredMediaFile struct {
	Rel    string
	Part   string
	SHA256 string
}

// The raw shapes this measurement walks. They name the fields it counts and
// nothing else, and they are here rather than in the docs package because the
// tree that package builds does not carry an object id yet.
type rawMeasuredRead struct {
	Body *rawMeasuredBody    `json:"body"`
	Tabs []rawMeasuredTabWra `json:"tabs"`
}

type rawMeasuredTabWra struct {
	TabProperties struct {
		TabID string `json:"tabId"`
		Title string `json:"title"`
	} `json:"tabProperties"`
	DocumentTab *struct {
		Body              *rawMeasuredBody           `json:"body"`
		InlineObjects     map[string]json.RawMessage `json:"inlineObjects"`
		PositionedObjects map[string]json.RawMessage `json:"positionedObjects"`
	} `json:"documentTab"`
	ChildTabs []rawMeasuredTabWra `json:"childTabs"`
}

type rawMeasuredBody struct {
	Content []rawMeasuredElement `json:"content"`
}

type rawMeasuredElement struct {
	Paragraph *struct {
		Elements []struct {
			InlineObjectElement *struct {
				InlineObjectID string `json:"inlineObjectId"`
			} `json:"inlineObjectElement"`
		} `json:"elements"`
		PositionedObjectIDs []string `json:"positionedObjectIds"`
	} `json:"paragraph"`
	Table *struct {
		TableRows []struct {
			TableCells []struct {
				Content []rawMeasuredElement `json:"content"`
			} `json:"tableCells"`
		} `json:"tableRows"`
	} `json:"table"`
}

// rawMeasuredObject is enough of an inline or positioned object to say what it
// holds. A picture and a Google Drawing are two different embedded objects,
// and the difference is the second half of question 1.
type rawMeasuredObject struct {
	InlineObjectProperties     *rawMeasuredProps `json:"inlineObjectProperties"`
	PositionedObjectProperties *rawMeasuredProps `json:"positionedObjectProperties"`
}

type rawMeasuredProps struct {
	EmbeddedObject struct {
		ImageProperties           json.RawMessage `json:"imageProperties"`
		EmbeddedDrawingProperties json.RawMessage `json:"embeddedDrawingProperties"`
	} `json:"embeddedObject"`
}

// measuredTabs walks the read into one entry per tab, each holding its objects
// in body order: an inline object where it sits in the text, and a paragraph's
// floating objects after that paragraph, which is the order export.Objects
// gives the pairing.
func measuredTabs(raw json.RawMessage) ([]measuredTab, error) {
	var r rawMeasuredRead
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	if len(r.Tabs) == 0 {
		// A document written before tabs existed answers with a body and no
		// tabs array. Nothing here knows its objects' shapes, so they are
		// reported as unknown rather than guessed at.
		tab := measuredTab{ID: "t.0"}
		fill(&tab, r.Body, nil, nil)
		return []measuredTab{tab}, nil
	}
	var out []measuredTab
	var walk func(tabs []rawMeasuredTabWra)
	walk = func(tabs []rawMeasuredTabWra) {
		for _, t := range tabs {
			tab := measuredTab{ID: t.TabProperties.TabID, Title: t.TabProperties.Title}
			if t.DocumentTab != nil {
				fill(&tab, t.DocumentTab.Body, t.DocumentTab.InlineObjects, t.DocumentTab.PositionedObjects)
			}
			out = append(out, tab)
			walk(t.ChildTabs)
		}
	}
	walk(r.Tabs)
	return out, nil
}

// fill puts one body's objects on the tab, in reading order.
func fill(tab *measuredTab, body *rawMeasuredBody, inline, positioned map[string]json.RawMessage) {
	if body == nil {
		return
	}
	var content func(elements []rawMeasuredElement)
	content = func(elements []rawMeasuredElement) {
		for _, el := range elements {
			switch {
			case el.Paragraph != nil:
				for _, e := range el.Paragraph.Elements {
					if e.InlineObjectElement == nil {
						continue
					}
					id := e.InlineObjectElement.InlineObjectID
					tab.Objects = append(tab.Objects, measuredObjectRef{
						Tab: tab.ID, Placement: "inline", ID: id, Shape: shape(inline[id]),
					})
					tab.Inline++
				}
				for _, id := range el.Paragraph.PositionedObjectIDs {
					tab.Objects = append(tab.Objects, measuredObjectRef{
						Tab: tab.ID, Placement: "floating", ID: id, Shape: shape(positioned[id]),
					})
					tab.Floating++
				}
			case el.Table != nil:
				for _, row := range el.Table.TableRows {
					for _, cell := range row.TableCells {
						content(cell.Content)
					}
				}
			}
		}
	}
	content(body.Content)
}

// shape says what an object holds: a picture, a Google Drawing, or something
// this walk cannot name.
func shape(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "unknown"
	}
	var o rawMeasuredObject
	if err := json.Unmarshal(raw, &o); err != nil {
		return "unknown"
	}
	props := o.InlineObjectProperties
	if props == nil {
		props = o.PositionedObjectProperties
	}
	if props == nil {
		return "unknown"
	}
	switch {
	case len(props.EmbeddedObject.ImageProperties) > 0:
		return "image"
	case len(props.EmbeddedObject.EmbeddedDrawingProperties) > 0:
		return "drawing"
	}
	return "other"
}

// measuredMedia walks the docx: every a:blip in word/document.xml order, each
// resolved through word/_rels/document.xml.rels to a part under word/media/
// and hashed. The second return is the count of w:drawing elements, which is
// the docx's own answer to how many pictures it thinks it carries.
func measuredMedia(export []byte) ([]measuredMediaFile, int, error) {
	z, err := zip.NewReader(bytes.NewReader(export), int64(len(export)))
	if err != nil {
		return nil, 0, fmt.Errorf("the export is not a zip: %w", err)
	}
	body, err := zipPart(z, "word/document.xml")
	if err != nil {
		return nil, 0, err
	}
	rels, err := zipPart(z, "word/_rels/document.xml.rels")
	if err != nil {
		return nil, 0, err
	}
	targets, err := relTargets(rels)
	if err != nil {
		return nil, 0, err
	}
	ids, drawings, err := embedded(body)
	if err != nil {
		return nil, 0, err
	}
	var out []measuredMediaFile
	for _, id := range ids {
		part := targets[id]
		if part == "" {
			out = append(out, measuredMediaFile{Rel: id, Part: "unresolved", SHA256: ""})
			continue
		}
		name := "word/" + strings.TrimPrefix(part, "/")
		b, err := zipPart(z, name)
		if err != nil {
			out = append(out, measuredMediaFile{Rel: id, Part: name, SHA256: "unreadable"})
			continue
		}
		sum := sha256.Sum256(b)
		out = append(out, measuredMediaFile{Rel: id, Part: name, SHA256: hex.EncodeToString(sum[:])})
	}
	return out, drawings, nil
}

// zipPart reads one member out of the export, bounded by the same ceiling the
// docx package reads a part under.
func zipPart(z *zip.Reader, name string) ([]byte, error) {
	for _, f := range z.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%s could not be opened: %w", name, err)
		}
		defer rc.Close()
		b, err := io.ReadAll(io.LimitReader(rc, docx.MaxExportBytes))
		if err != nil {
			return nil, fmt.Errorf("%s could not be read: %w", name, err)
		}
		return b, nil
	}
	return nil, fmt.Errorf("the export carries no %s", name)
}

// relTargets reads the relationship part into id to target. Every relationship
// is read, not only the picture ones, because an id that resolves to something
// other than word/media/ is itself an answer worth printing.
func relTargets(b []byte) (map[string]string, error) {
	type relationship struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	}
	var doc struct {
		Relationships []relationship `xml:"Relationship"`
	}
	if err := xml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("word/_rels/document.xml.rels did not parse: %w", err)
	}
	out := map[string]string{}
	for _, r := range doc.Relationships {
		out[r.ID] = r.Target
	}
	return out, nil
}

// embedded walks word/document.xml in order for the relationship id of every
// a:blip, and counts every w:drawing on the way. Order is the whole point: it
// is what Task 8's pairing rests on.
func embedded(b []byte) ([]string, int, error) {
	dec := xml.NewDecoder(bytes.NewReader(b))
	var ids []string
	drawings := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return ids, drawings, nil
		}
		if err != nil {
			return nil, 0, fmt.Errorf("word/document.xml did not parse: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == "drawing" {
			drawings++
		}
		if start.Name.Local != "blip" {
			continue
		}
		for _, a := range start.Attr {
			if a.Name.Local == "embed" {
				ids = append(ids, a.Value)
			}
		}
	}
}

// TestTheMeasurementReadsTheFixtureItPairsAgainst is the one part of this file
// that runs in `make test`: the pictures the second measurement compares
// against are on this disk, and a note that moved would otherwise be found by
// Nail in the middle of a live run.
func TestTheMeasurementReadsTheFixtureItPairsAgainst(t *testing.T) {
	uploaded, err := uploadedHashes()
	if err != nil {
		t.Fatalf("the pictures beside the pictures note could not be read: %v", err)
	}
	if len(uploaded) == 0 {
		t.Fatalf("%s holds no picture, and the fixture document is uploaded from it", picturesNoteDir)
	}
	var names []string
	for _, name := range uploaded {
		names = append(names, name)
	}
	sort.Strings(names)
	if !contains(names, "diagram.png") {
		t.Errorf("the pictures are %v, and diagram.png is the inline picture the fixture document holds first", names)
	}
}

func contains(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
