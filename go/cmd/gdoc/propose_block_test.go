package main

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// This file is the block kind at the command's own door: the proposals file
// read, the marker check over the four fields a block adds, a file holding both
// kinds, and what the note is left carrying. write_test.go holds the words
// kind's half, and internal/propose holds the write itself.

// blockContent is the everyday block the tests propose, and it is the content
// block-after.json records as one suggested insertion. The two are one write, so
// a change to either has to move the other.
const blockContent = "## 3.6 Limits\\n\\nBody text with **bold**.\\n\\n- one\\n- two\\n"

const blockWhy = "the policy says nothing about limits"

// oneBlock is a proposals file holding one block, placed after the anchor
// block-before.json carries.
const oneBlock = `[{"kind":"block","after":"reviewed annually","content":"` + blockContent + `",` +
	`"why":"` + blockWhy + `"}]`

// blockAnswers is the probe followed by one block proposal: the command's own
// read, the read ApplyBlock places from, the batch, and the three read-backs.
//
// The two reads before the write are the same URL and the same fixture, and the
// third is the document with the block pending in it. One fixture for all three
// would verify the document as it stood before the write.
func blockAnswers(t *testing.T) []*answer {
	t.Helper()
	out := probeAnswers(t, "probe-enrolled.json")
	return append(out,
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "block-before.json"), once: true},
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "block-before.json"), once: true},
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "block-after.json"), once: true},
		&answer{method: "POST", match: proposeDocID + ":batchUpdate", json: readFixture(t, "propose-batch.json"), once: true},
		&answer{method: "GET", match: "PREVIEW_WITHOUT_SUGGESTIONS", json: readFixture(t, "block-before.json"), once: true},
		&answer{method: "GET", match: "/export?", bytes: exportWithBlockComment(t, "\U0001F916 "+blockWhy, "3.6 Limits"), once: true},
	)
}

// exportWithBlockComment is the docx a verified block is confirmed against: the
// robot comment gdoc wrote, attached to the block's first line.
func exportWithBlockComment(t *testing.T, body, on string) []byte {
	t.Helper()
	comments := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:comment w:id="0" w:author="Nail Khusnullin" w:date="2026-09-28T09:00:00Z">
    <w:p><w:r><w:t>` + body + `</w:t></w:r></w:p>
  </w:comment>
</w:comments>`
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:commentRangeStart w:id="0"/>
      <w:r><w:t>` + on + `</w:t></w:r>
      <w:commentRangeEnd w:id="0"/>
      <w:r><w:commentReference w:id="0"/></w:r>
    </w:p>
  </w:body>
</w:document>`
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, part := range map[string]string{"word/document.xml": document, "word/comments.xml": comments} {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestProposeReadsABlockEntry is the door itself. A proposals file naming the
// block kind runs the block's own write, and the envelope says where the block
// went in the words the file asked for rather than in an index.
func TestProposeReadsABlockEntry(t *testing.T) {
	stubWire(t, &fakeWire{answers: blockAnswers(t)})
	from := tempFile(t, "proposals.json", oneBlock)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("a block proposal must run: %v (exit %d)", got, code)
	}
	list, _ := dataOf(t, got)["proposals"].([]any)
	if len(list) != 1 {
		t.Fatalf("proposals = %v, want one entry", list)
	}
	one, _ := list[0].(map[string]any)
	if one["after"] != "reviewed annually" {
		t.Errorf("after = %v, and a block reports the words it was placed by", one["after"])
	}
	if one["quoted"] != nil || one["replacement"] != nil {
		t.Errorf("the words kind's fields must stay off a block: %v", one)
	}
	if one["sent"] != true || one["verified"] != true {
		t.Errorf("the block came back %v, want sent and verified", one)
	}
	if one["comment_id"] != "AAAC" {
		t.Errorf("comment_id = %v", one["comment_id"])
	}
}

// TestProposeRefusesAnUnknownFieldInABlockEntry is the strict read. A block
// entry is decoded with the same rule as a words entry, so a misspelled field
// is refused rather than dropped: a dropped `content` is a block with nothing
// in it, and a dropped `after` is a block placed somewhere else.
func TestProposeRefusesAnUnknownFieldInABlockEntry(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: blockAnswers(t)})
	from := tempFile(t, "proposals.json",
		`[{"kind":"block","after":"reviewed annually","contents":"## 3.6 Limits\n","why":"c"}]`)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an unknown field must stop the run: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "contents") {
		t.Errorf("the error must name the field it did not know: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach Google, the probe document included: %v", f.calls)
	}
}

// TestProposeRefusesABlockFieldWithNoKind is the other half of the strict read,
// and the decoder cannot hold it: `after` and `content` are fields of the one
// struct both kinds are read into, so an entry naming no kind carries them
// quietly and is then refused for quoting no text. That names nothing the author
// did wrong. The refusal names the field and the kind it belongs to.
func TestProposeRefusesABlockFieldWithNoKind(t *testing.T) {
	for _, tc := range []struct{ name, file, says string }{
		{"after with no kind",
			`[{"after":"reviewed annually","content":"## 3.6 Limits\n","why":"c"}]`, "after"},
		{"a replace with no kind",
			`[{"replace_from":"reviewed annually","replace_to":"risk matrix","content":"x\n","why":"c"}]`, "replace_from"},
		{"replace_to with no kind",
			`[{"quoted":"a","replacement":"b","replace_to":"risk matrix","why":"c"}]`, "replace_to"},
		{"content with no kind",
			`[{"quoted":"a","replacement":"b","content":"## 3.6 Limits\n","why":"c"}]`, "content"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := stubWire(t, &fakeWire{answers: blockAnswers(t)})
			from := tempFile(t, "proposals.json", tc.file)

			got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
			if code == 0 || got["ok"] != false {
				t.Fatalf("a block field on an entry with no kind must stop the run: %v (exit %d)", got, code)
			}
			msg, _ := got["error"].(string)
			if !strings.Contains(msg, tc.says) {
				t.Errorf("the error must name the field %q: %q", tc.says, msg)
			}
			if !strings.Contains(msg, `"block"`) {
				t.Errorf("the error must name the kind the field belongs to: %q", msg)
			}
			if len(f.calls) != 0 {
				t.Errorf("nothing may reach Google: %v", f.calls)
			}
		})
	}
}

// TestProposeRefusesAMarkerInABlockField is decision 3's door widened to the
// block. A block's placement quotes and its content are document text, and
// document text never passes through internal/plaintext, so the marker rule is
// asked here over every field that goes into the document.
//
// A skill proposing a section out of a note it has just exported, with the
// markers unresolved, is what this refuses.
func TestProposeRefusesAMarkerInABlockField(t *testing.T) {
	for _, tc := range []struct{ name, file, says string }{
		{"the content carries an insertion marker",
			`[{"kind":"block","after":"reviewed annually","content":"## 3.6 {+Limits+}[s:AAA]\n","why":"c"}]`,
			"content"},
		{"the anchor carries a deletion marker",
			`[{"kind":"block","after":"reviewed {-annually-}[s:AAA]","content":"## 3.6 Limits\n","why":"c"}]`,
			"after"},
		{"replace_from carries a comment anchor",
			`[{"kind":"block","replace_from":"[[c:AAA]]reviewed annually[[/c]]","replace_to":"risk matrix",` +
				`"content":"## 3.6 Limits\n","why":"c"}]`,
			"replace_from"},
		{"replace_to carries a comment anchor",
			`[{"kind":"block","replace_from":"reviewed annually","replace_to":"[[c:AAA]]risk matrix[[/c]]",` +
				`"content":"## 3.6 Limits\n","why":"c"}]`,
			"replace_to"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := stubWire(t, &fakeWire{answers: blockAnswers(t)})
			from := tempFile(t, "proposals.json", tc.file)

			got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
			if code == 0 || got["ok"] != false {
				t.Fatalf("a marker in a block field must stop the run: %v (exit %d)", got, code)
			}
			msg, _ := got["error"].(string)
			if !strings.Contains(msg, tc.says) {
				t.Errorf("the error must say %q: %q", tc.says, msg)
			}
			if !strings.Contains(msg, "marker") {
				t.Errorf("the error must call it a marker: %q", msg)
			}
			if len(f.calls) != 0 {
				t.Errorf("nothing may reach Google, the probe document included: %v", f.calls)
			}
		})
	}
}

// TestProposeRunsABothKindsFileInFileOrder is the mixed file. One list holds
// both kinds, the loop asks nothing about which an entry is, and each entry gets
// its own read: the block moves every index behind it, so the words proposal
// after it is found in a document that has just come back.
func TestProposeRunsABothKindsFileInFileOrder(t *testing.T) {
	answers := append(blockAnswers(t),
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-before.json"), once: true},
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "propose-after.json"), once: true},
		&answer{method: "POST", match: proposeDocID + ":batchUpdate", json: readFixture(t, "propose-batch.json"), once: true},
		&answer{method: "GET", match: "PREVIEW_WITHOUT_SUGGESTIONS", json: readFixture(t, "propose-preview.json"), once: true},
		&answer{method: "GET", match: "/export?", bytes: exportWithComment(t, true), once: true},
	)
	f := stubWire(t, &fakeWire{answers: answers})
	from := tempFile(t, "proposals.json",
		`[{"kind":"block","after":"reviewed annually","content":"`+blockContent+`","why":"`+blockWhy+`"},`+
			`{"quoted":"reviewed annually","replacement":"reviewed every six months",`+
			`"why":"the policy says twice a year"}]`)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("a file of both kinds must run: %v (exit %d)", got, code)
	}
	list, _ := dataOf(t, got)["proposals"].([]any)
	if len(list) != 2 {
		t.Fatalf("proposals = %v, want one entry per proposal in the file", list)
	}
	first, _ := list[0].(map[string]any)
	second, _ := list[1].(map[string]any)
	if first["after"] != "reviewed annually" {
		t.Errorf("the first entry is the block the file names first: %v", first)
	}
	if second["quoted"] != "reviewed annually" {
		t.Errorf("the second entry is the words proposal the file names second: %v", second)
	}
	if first["verified"] != true || second["verified"] != true {
		t.Errorf("both must come back verified: %v, %v", first, second)
	}
	// One batch per proposal, in the file's order: the block's own insertText
	// first, then the words kind's deleteContentRange. The probe's writes go to
	// its own document and are left out.
	var batches [][]byte
	for _, c := range f.writes() {
		if strings.Contains(c.URL, proposeDocID+":batchUpdate") {
			batches = append(batches, c.Body)
		}
	}
	if len(batches) != 2 {
		t.Fatalf("batches into the document = %d, want one per proposal", len(batches))
	}
	if !bytes.Contains(batches[0], []byte("3.6 Limits")) {
		t.Errorf("the first batch is not the block's: %s", batches[0])
	}
	if !bytes.Contains(batches[1], []byte("deleteContentRange")) {
		t.Errorf("the second batch is not the words kind's: %s", batches[1])
	}
}

// TestProposeRecordsABlocksPlacementInTheNote is the provenance. A block is
// remembered by the same two ids as a words proposal, and the quote it is shown
// by later is the words it was placed by, because a block replaced no words.
func TestProposeRecordsABlocksPlacementInTheNote(t *testing.T) {
	stubWire(t, &fakeWire{answers: blockAnswers(t)})
	note := copyFixture(t, "propose-note.md")
	from := tempFile(t, "proposals.json", oneBlock)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID, "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("propose --md: %v (exit %d)", got, code)
	}
	grown := entryOf(t, note, proposeDocID).Proposals
	if len(grown) != 1 {
		t.Fatalf("the note records %d proposals, want one", len(grown))
	}
	if grown[0].ID != "suggest.block" || grown[0].CommentID != "AAAC" {
		t.Errorf("the entry records %+v, want both ids the write left behind", grown[0])
	}
	if grown[0].Quoted != "reviewed annually" {
		t.Errorf("quoted = %q, and a block is remembered by the words it was placed by", grown[0].Quoted)
	}
}

// TestProposeRecordsAReplacesFirstQuoteInTheNote is the other placement form. A
// replace names two quotes and the note has one field for them, so it keeps the
// first: that is where the block went in, and it is what a later run shows.
func TestProposeRecordsAReplacesFirstQuoteInTheNote(t *testing.T) {
	answers := probeAnswers(t, "probe-enrolled.json")
	answers = append(answers,
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "block-before.json"), once: true},
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "block-before.json"), once: true},
		&answer{method: "GET", match: proposeDocID + "?includeTabsContent", json: readFixture(t, "block-replace-after.json"), once: true},
		&answer{method: "POST", match: proposeDocID + ":batchUpdate", json: readFixture(t, "propose-batch.json"), once: true},
		&answer{method: "GET", match: "PREVIEW_WITHOUT_SUGGESTIONS", json: readFixture(t, "block-before.json"), once: true},
		&answer{method: "GET", match: "/export?", bytes: exportWithBlockComment(t, "\U0001F916 "+blockWhy, "3.6 Limits"), once: true},
	)
	stubWire(t, &fakeWire{answers: answers})
	note := copyFixture(t, "propose-note.md")
	from := tempFile(t, "proposals.json",
		`[{"kind":"block","replace_from":"reviewed annually","replace_to":"risk matrix",`+
			`"content":"`+blockContent+`","why":"`+blockWhy+`"}]`)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID, "--md", note)
	if code != 0 || got["ok"] != true {
		t.Fatalf("a replace must run: %v (exit %d)", got, code)
	}
	one, _ := dataOf(t, got)["proposals"].([]any)[0].(map[string]any)
	if one["replace_from"] != "reviewed annually" || one["replace_to"] != "risk matrix" {
		t.Errorf("the envelope must carry both quotes of the replace: %v", one)
	}
	grown := entryOf(t, note, proposeDocID).Proposals
	if len(grown) != 1 || grown[0].Quoted != "reviewed annually" {
		t.Errorf("the note records %+v, want the replace's first quote", grown)
	}
}

// TestHelpProposeDescribesTheBlockEntry is the one place a skill or a person
// learns the file's shape without reading the source. Both kinds are named, so
// nobody has to guess that a block exists.
func TestHelpProposeDescribesTheBlockEntry(t *testing.T) {
	got, code := runJSON(t, "help", "propose")
	if code != 0 || got["ok"] != true {
		t.Fatalf("help propose: %v (exit %d)", got, code)
	}
	list, _ := dataOf(t, got)["commands"].([]any)
	if len(list) != 1 {
		t.Fatalf("commands = %v, want propose alone", list)
	}
	one, _ := list[0].(map[string]any)
	flags, _ := one["flags"].([]any)
	var said string
	for _, raw := range flags {
		f, _ := raw.(map[string]any)
		if f["name"] == "--from" {
			said, _ = f["summary"].(string)
		}
	}
	if said == "" {
		t.Fatalf("help propose names no --from: %v", one)
	}
	for _, word := range []string{"block", "after", "replace_from"} {
		if !strings.Contains(said, word) {
			t.Errorf("the --from summary must name %q: %q", word, said)
		}
	}
}
