package propose

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/guard"
)

// testWhy is the reason the golden batches carry.
const testWhy = "the limits section was missing"

// checkGolden compares one batch against the request list recorded beside this
// file, request by request. A golden is what this test is for: the order of the
// requests is the reason the batch works, and a field-by-field test would say
// nothing about the order.
func checkGolden(t *testing.T, name string, raw []byte) {
	t.Helper()
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		t.Fatalf("the batch is not JSON: %v", err)
	}
	pretty.WriteString("\n")
	want, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading the golden batch: %v", err)
	}
	if !bytes.Equal(pretty.Bytes(), want) {
		t.Errorf("the batch is not testdata/%s.\n--- built ---\n%s\n--- recorded ---\n%s", name, pretty.Bytes(), want)
	}
}

// blockContent is the content of the block the guard's own
// TestABlockProposalNeedsNoGrant sends, read into paragraphs.
func blockContent(t *testing.T) []Para {
	t.Helper()
	paras, err := ParseContent(testContent)
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}
	return paras
}

// TestBlockBatchForAnAfterBlock is the everyday shape: the insert at the start
// of the paragraph behind the anchor, a named style per new paragraph, the
// clearing text style over the whole insert, the block's own marks, the bullets
// of its list, the bullet removal over everything that is not a list item, and
// the comment on the first new paragraph.
func TestBlockBatchForAnAfterBlock(t *testing.T) {
	content := blockContent(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	checkGolden(t, "block-after-batch.json", BlockBatch(place, content, testWhy, ""))
}

// TestBlockBatchForAReplace is the same batch with two requests more: the
// deletion of the old paragraphs at the indexes the insert left them at, and
// the assignee on the comment.
func TestBlockBatchForAReplace(t *testing.T) {
	content := blockContent(t)
	place, err := PlaceReplace(document(t, "block-body.json"), "reviewed annually", "risk matrix")
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	checkGolden(t, "block-replace-batch.json", BlockBatch(place, content, testWhy, "nail@altery.com"))
}

// TestBlockBatchAfterTheLastParagraph is the shape MEASURED.md row 7 pins. The
// text opens with a newline and carries no trailing one, so the anchor gets a
// new mark and the block's last paragraph owns the body's old final one.
func TestBlockBatchAfterTheLastParagraph(t *testing.T) {
	content := plainBlock(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "operations lead", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	if !place.AtEnd {
		t.Fatal("the anchor is the document's last paragraph, and this test is about that case")
	}
	checkGolden(t, "block-at-end-batch.json", BlockBatch(place, content, testWhy, ""))
}

// TestBlockBatchAfterTheLastParagraphRestatesNothingOnTheFinalMark is the rest
// of MEASURED.md row 7. That row came back with one suggestion id because
// nothing restated the paragraph owning the body's old final mark, and a
// restated one there was not measured. The block's last paragraph owns that
// mark, so no request of this batch may reach it: the placement has already
// refused an anchor that is not plain body text, which is what makes the
// inherited style and the missing bullet the ones the block wanted.
func TestBlockBatchAfterTheLastParagraphRestatesNothingOnTheFinalMark(t *testing.T) {
	content := plainBlock(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "operations lead", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	// The final mark is the last unit of the block's last paragraph.
	l := layOutBlock(place, content)
	mark := l.Paras[len(l.Paras)-1].End - 1
	for _, r := range requestsOf(t, BlockBatch(place, content, testWhy, "")) {
		for _, kind := range []string{"updateParagraphStyle", "deleteParagraphBullets", "createParagraphBullets"} {
			if _, ok := r[kind]; !ok {
				continue
			}
			at := rangeIn(t, r, kind)
			if at[0] <= mark && mark < at[1] {
				t.Errorf("the %s covering %v reaches the body's old final mark at %d, which MEASURED.md row 7 never restated", kind, at, mark)
			}
		}
	}
}

// TestBlockBatchCountsInUTF16CodeUnits is the hazard every index in this
// package has. A run behind an emoji is two units further on than its bytes
// say, and a mark written at the wrong offset lands on the wrong words.
func TestBlockBatchCountsInUTF16CodeUnits(t *testing.T) {
	content, err := ParseContent("Body 🤖 **bold** tail\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	reqs := requestsOf(t, BlockBatch(place, content, testWhy, ""))

	// "Body " is five units and the robot is two, so the bold run opens at
	// 93 + 8. Counted in bytes the robot is four and the mark would land at
	// 93 + 10, two characters into the word behind it.
	mark := rangeIn(t, reqs[3], "updateTextStyle")
	if mark[0] != 101 || mark[1] != 105 {
		t.Errorf("the bold run is %v, want 101..105", mark)
	}
	// The paragraph is seventeen units and its mark is one more.
	style := rangeIn(t, reqs[1], "updateParagraphStyle")
	if style[0] != 93 || style[1] != 111 {
		t.Errorf("the paragraph is %v, want 93..111", style)
	}
}

// TestBlockBatchClearsTheMarksTheInsertInherited is the second half of
// MEASURED.md's start-of-paragraph rule. Text inserted at a paragraph's start
// takes that paragraph's first run style, so a block in front of a bold linked
// sentence would arrive bold and linked. The clearing request covers the whole
// insert and comes before the block's own marks, so a run that really is bold
// is written bold again afterwards.
func TestBlockBatchClearsTheMarksTheInsertInherited(t *testing.T) {
	content := blockContent(t)
	place, err := PlaceAfter(document(t, "block-bold-next.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	reqs := requestsOf(t, BlockBatch(place, content, testWhy, ""))

	var clearing, marks int
	for i, r := range reqs {
		body, ok := r["updateTextStyle"].(map[string]any)
		if !ok {
			continue
		}
		if len(body["textStyle"].(map[string]any)) == 0 {
			clearing = i
			if got := body["fields"]; got != "bold,italic,underline,strikethrough,link" {
				t.Errorf("the clearing mask is %v, and it names every mark the insert can inherit", got)
			}
			at := rangeIn(t, r, "updateTextStyle")
			// The insert is forty units long and goes in at 93.
			if at[0] != 93 || at[1] != 133 {
				t.Errorf("the clearing request covers %v, want the whole insert 93..133", at)
			}
			continue
		}
		marks = i
	}
	if clearing == 0 || marks == 0 {
		t.Fatalf("the batch carries %d requests and not both a clearing style and a mark", len(reqs))
	}
	if clearing > marks {
		t.Errorf("the clearing request is at %d and the block's own mark at %d; clearing after the marks would take them off again", clearing, marks)
	}
}

// TestBlockBatchRemovesBulletsFromItsOwnParagraphsOnly is the other thing the
// insert inherits. New text takes the list membership of the paragraph it
// lands in front of, and stating a named style does not clear it. The removal
// covers every new paragraph that is not a list item, in one request per
// stretch of them, and never reaches a paragraph that was already there.
func TestBlockBatchRemovesBulletsFromItsOwnParagraphsOnly(t *testing.T) {
	// A heading, a list, and a paragraph behind it, so the new paragraphs that
	// are not list items are two stretches with the list between them.
	content, err := ParseContent("## Limits\n\n- one\n- two\n\nAnd a closing line.\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}
	place, err := PlaceAfter(document(t, "block-body.json"), "top ten suppliers", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	reqs := requestsOf(t, BlockBatch(place, content, testWhy, ""))

	// The anchor is the bullet at 142..185 and the block goes in at 185.
	// "Limits" is the paragraph 185..192, the two items run to 200, and the
	// closing line runs to 220.
	var got [][2]int
	for _, r := range reqs {
		if _, ok := r["deleteParagraphBullets"]; ok {
			got = append(got, rangeIn(t, r, "deleteParagraphBullets"))
		}
	}
	want := [][2]int{{185, 192}, {200, 220}}
	if len(got) != len(want) {
		t.Fatalf("the bullet removal is %v, want one request per stretch: %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("bullet removal %d covers %v, want %v", i, got[i], want[i])
		}
	}
}

// TestBlockBatchNumbersANumberedList is the other bullet preset. The two list
// kinds are two presets, which is why the content carries the kind.
func TestBlockBatchNumbersANumberedList(t *testing.T) {
	content, err := ParseContent("1. first\n2. second\n")
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	reqs := requestsOf(t, BlockBatch(place, content, testWhy, ""))
	for _, r := range reqs {
		body, ok := r["createParagraphBullets"].(map[string]any)
		if !ok {
			continue
		}
		if body["bulletPreset"] != "NUMBERED_DECIMAL_ALPHA_ROMAN" {
			t.Errorf("the preset is %v, want the numbered one", body["bulletPreset"])
		}
		return
	}
	t.Fatal("the batch carries no createParagraphBullets for a numbered list")
}

// TestBlockBatchWritesTheReasonAsPlainTextUnderTheRobot is the rule everything
// gdoc writes into a thread keeps: the robot prefix, and no markdown. The
// comment is anchored on the words of the first new paragraph, without its
// paragraph mark, because a range that takes a mark anchors across into the
// paragraph behind it.
func TestBlockBatchWritesTheReasonAsPlainTextUnderTheRobot(t *testing.T) {
	content := blockContent(t)
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", content)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	reqs := requestsOf(t, BlockBatch(place, content, testWhy, ""))
	body, ok := reqs[len(reqs)-1]["insertComment"].(map[string]any)
	if !ok {
		t.Fatalf("the last request is %v, and the comment is written last", reqs[len(reqs)-1])
	}
	if want := Prefix + testWhy; body["content"] != want {
		t.Errorf("the comment reads %v, want %q", body["content"], want)
	}
	// "3.6 Limits" is ten units at 93, and its mark is the eleventh.
	if at := rangeIn(t, reqs[len(reqs)-1], "insertComment"); at[0] != 93 || at[1] != 103 {
		t.Errorf("the comment is anchored on %v, want the first new paragraph's words 93..103", at)
	}
}

// TestALargeBlockStaysUnderThePeek is what MaxContent is for. The guard reads a
// batchUpdate body to judge the requests in it, and a body past the first
// 1048576 bytes arrives there truncated and is refused rather than carried
// unread. That refusal names the guard and not the block, so it must never be
// what a person sees: Check refuses a content longer than MaxContent first.
//
// The ceiling is stated as a literal here rather than read out of the guard,
// which is the house rule for a value a test is pinning: a test that reads the
// constant follows it wherever somebody moves it.
func TestALargeBlockStaysUnderThePeek(t *testing.T) {
	const maxPeek = 1 << 20

	t.Run("sixty paragraphs of an ordinary rewritten section", func(t *testing.T) {
		var b strings.Builder
		for i := 0; i < 20; i++ {
			b.WriteString("## A section heading\n\nA paragraph of body text with **bold** and a [link](https://example.com/p) in it, about as long as a real one.\n\n- one list item\n- and another\n\n")
		}
		raw := buildLargest(t, b.String())
		if len(raw) >= maxPeek {
			t.Errorf("the batch is %d bytes, and the guard reads only the first %d", len(raw), maxPeek)
		}
	})

	// The worst content MaxContent allows is the one that makes the most
	// requests per byte. Each case below is one way of doing that, at exactly
	// the limit, and the worst of the four is what the constant was chosen
	// from: paragraphs alternating with list items, where every paragraph costs
	// a named style and a bullet request of its own.
	for name, content := range map[string]string{
		"paragraphs alternating with list items": strings.Repeat("a\n\n- b\n\n", MaxContent/8),
		"headings alternating with list items":   strings.Repeat("#a\n\n- b\n\n", MaxContent/9),
		"the shortest paragraphs there are":      strings.Repeat("a\n\n", MaxContent/3),
		"every run of one paragraph marked":      strings.Repeat("**a** ", MaxContent/6),
	} {
		t.Run(name+", at the content limit", func(t *testing.T) {
			raw := buildLargest(t, content)
			if len(raw) >= maxPeek {
				t.Errorf("a block at the content limit builds %d bytes, and the guard reads only the first %d", len(raw), maxPeek)
			}
		})
	}
}

// buildLargest is one content built into a batch against a real placement, and
// judged by the real guard, so the size this test measures is the size the
// guard would read.
func buildLargest(t *testing.T, content string) []byte {
	t.Helper()
	if len(content) > MaxContent {
		t.Fatalf("the test content is %d bytes, past the limit of %d it is meant to sit at", len(content), MaxContent)
	}
	paras, err := ParseContent(content)
	if err != nil {
		t.Fatalf("parsing the content: %v", err)
	}
	place, err := PlaceAfter(document(t, "block-body.json"), "reviewed annually", paras)
	if err != nil {
		t.Fatalf("placing the block: %v", err)
	}
	raw := BlockBatch(place, paras, testWhy, "")

	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	u, err := url.Parse(BatchURL(testDocID))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Judge("POST", u, raw); err != nil {
		t.Errorf("the guard refused a batch of %d bytes: %v", len(raw), err)
	}
	return raw
}

// requestsOf is the request list of one batch, decoded.
func requestsOf(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	body := decodeBatch(t, raw)
	if wc, ok := body["writeControl"].(map[string]any); !ok || wc["writeMode"] != "SUGGEST" {
		t.Fatalf("writeControl = %v, want SUGGEST", body["writeControl"])
	}
	list, ok := body["requests"].([]any)
	if !ok {
		t.Fatalf("requests = %v, want a list", body["requests"])
	}
	out := make([]map[string]any, 0, len(list))
	for _, r := range list {
		out = append(out, r.(map[string]any))
	}
	return out
}

// rangeIn is the range one request names.
func rangeIn(t *testing.T, req map[string]any, kind string) [2]int {
	t.Helper()
	body, ok := req[kind].(map[string]any)
	if !ok {
		t.Fatalf("the request is not a %s: %v", kind, req)
	}
	at, ok := body["range"].(map[string]any)
	if !ok {
		t.Fatalf("the %s names no range: %v", kind, body)
	}
	return [2]int{int(at["startIndex"].(float64)), int(at["endIndex"].(float64))}
}
