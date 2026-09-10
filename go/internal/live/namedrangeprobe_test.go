package live

// What does a named range do when the text under it is only a suggestion?
//
// Nail's question, 2026-09-10, and M7c waits on the answer. The house prelude
// is proposed rather than written: the cover, the front-matter tables and the
// legend all go into a document as a pending insertion, so that Nail accepts
// them in the browser and gdoc never writes a character on its own authority.
// A second run then has to find what the first one proposed, and the marker for
// that is a named range over gdoc's own prelude. createNamedRange is the one
// request kind the 2026-09-10 suggested-insert probe found Docs refuses as a
// suggestion ("Request does not support application as suggestion"), so the
// marker is created directly, which is safe on its own terms because it adds
// and removes no text.
//
// What nobody has measured is what that range does over text that exists only
// as a pending insertion:
//
//   - can it be created there at all;
//   - what does it cover while the insertion is still pending;
//   - what is left of it when the suggestion is rejected and the text vanishes;
//   - what is left of it when the suggestion is accepted.
//
// The first three are asked here, one fresh document per case, for the reason
// the suggested-insert probe learned the hard way: a probe that measures its
// own leftovers answers about itself.
//
// The fourth cannot be asked from inside gdoc, and that is a finding rather
// than a gap. acceptSuggestion is refused by the guard at every level, with no
// door: SPEC's Never list is "never accept, reject or delete anyone else's
// suggestion", and the guard holds it on the request kind's own name rather
// than on the level, so a document the probe created a second ago is refused
// like any other. rejectSuggestion has one door, AllowReject, because withdraw
// needs it; accept has none, and opening one for a probe would be widening the
// guard to measure it, which is the tail wagging the dog. So the accept case
// leaves its document behind, prints its URL, and asks Nail to accept the
// suggestion in the browser the way he will accept a real prelude.
// TestLiveNamedRangeAfterAcceptedByHand then reads that document back and
// finishes the table.
//
// It asserts almost nothing, for TestLiveStyleFidelity's reason: a measurement
// that fails the build when Google answers differently has already decided the
// answer. It fails only when it cannot create or cannot trash.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/drive"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

// acceptedVar names the document a person accepted the probe's suggestion in.
// It has no default: the follow-up reads a document somebody has just worked in
// by hand, and guessing which one is not something a test may do.
const acceptedVar = "GDOC_LIVE_ACCEPTED_DOC_ID"

const (
	// namedRangeBody is the author's own text, written directly, so the
	// proposed line meets a document that already holds words somebody else
	// wrote. That is the shape a real prelude insert meets.
	namedRangeBody = "The supplier register is reviewed annually by the operations team.\n"

	// preludeLine stands for the whole prelude. One line at index 1, which is
	// where a cover goes.
	preludeLine = "SUGGESTED COVER TITLE\n"

	// markerName is the name M7c's marker would wear. It is a name here and
	// nothing more: the range is read back by its id, because two named ranges
	// may share one name and Docs keys the map by name.
	markerName = "gdoc:house-prelude"
)

// preludeProbe is one document set up the way M7c's first run leaves one: the
// author's text, a proposed line in front of it, and a named range created
// directly over that pending insertion.
type preludeProbe struct {
	id      string // the document, created by this probe
	sugID   string // the suggestion id on the proposed line
	markID  string // the id createNamedRange answered with
	created bool   // whether createNamedRange was carried at all
	refused string // and why, when it was not
	spans   []markerSpan
	start   int
	end     int
}

// markerSpan is one named range as a read-back shows it: its id, its name, the
// span it covers and the words in that span.
type markerSpan struct {
	id    string
	name  string
	start int
	end   int
	text  string
}

func (m markerSpan) String() string {
	return fmt.Sprintf("%s [%d,%d) %q", m.name, m.start, m.end, oneLine(m.text))
}

func TestLiveNamedRangeOverSuggestionProbe(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create documents in the Drive test folder and probe them; %s names another folder", liveVar, writeVar, folderVar)
	}
	folder := strings.TrimSpace(os.Getenv(folderVar))
	if folder == "" {
		folder = testFolder
	}

	p := guard.NewPolicy()
	p.AllowCreateIn(folder)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	ctx := context.Background()

	var table []string
	say := func(question, format string, a ...any) {
		table = append(table, fmt.Sprintf("%-46s %s", question, fmt.Sprintf(format, a...)))
	}

	// Case 1. Can the marker be created over a pending insertion at all, and
	// what does it cover while the insertion is still pending?
	pending := newPreludeProbe(t, ctx, s, folder, "gdoc NAMED RANGE PROBE - pending")
	defer trashProbe(t, ctx, s, pending.id, "pending")
	if !pending.created {
		say("created over a pending insertion?", "NO: %s", pending.refused)
	} else {
		say("created over a pending insertion?", "yes, id %s", pending.markID)
		say("what it covers while pending", "%s", spanList(pending.spans))
	}

	// Case 2. The suggestion is rejected, so the text under the marker goes.
	// Does the marker collapse, vanish, or survive covering something else?
	rejected := newPreludeProbe(t, ctx, s, folder, "gdoc NAMED RANGE PROBE - rejected")
	defer trashProbe(t, ctx, s, rejected.id, "rejected")
	switch {
	case !rejected.created:
		say("after the suggestion is rejected", "not asked: the marker was never created")
	case rejected.sugID == "":
		say("after the suggestion is rejected", "not asked: the proposed line carried no suggestion id")
	default:
		// The one door the guard has in the suggestion family, and it is the
		// door withdraw already uses: one id, named for this run only. The id
		// is the probe's own proposal, in the probe's own document.
		p.AllowReject(rejected.sugID)
		if err := batch(ctx, s, rejected.id, req("rejectSuggestion", map[string]any{"suggestionId": rejected.sugID})); err != nil {
			say("after the suggestion is rejected", "the reject itself failed: %s", oneLine(err.Error()))
			break
		}
		after, err := readInline(ctx, s, rejected.id)
		if err != nil {
			say("after the suggestion is rejected", "the read after the reject failed: %s", oneLine(err.Error()))
			break
		}
		spans := namedRangeSpans(after)
		if len(spans) == 0 {
			say("after the suggestion is rejected", "the marker is GONE")
		} else {
			say("after the suggestion is rejected", "%s", spanList(spans))
		}
	}

	// Case 3. The accept, which gdoc cannot make. The document is left behind
	// on purpose, so the last line of the table is filled in by a person.
	accepted := newPreludeProbe(t, ctx, s, folder, "gdoc NAMED RANGE PROBE - accept me by hand")
	if err := batch(ctx, s, accepted.id, req("acceptSuggestion", map[string]any{"suggestionId": accepted.sugID})); err != nil {
		say("can gdoc accept its own suggestion?", "NO: %s", oneLine(err.Error()))
	} else {
		say("can gdoc accept its own suggestion?", "yes, and that is a finding: the guard should have refused it")
	}

	t.Log("")
	t.Log("  a named range over suggested text")
	t.Log("  ------------------------------------------------------------------------")
	for _, line := range table {
		t.Logf("  %s", line)
	}
	t.Log("")
	t.Logf("  The accept is Nail's, in the browser. The document is left in the folder:")
	t.Logf("    https://docs.google.com/document/d/%s/edit", accepted.id)
	t.Logf("    marker id %s, created %v, over [%d,%d)", accepted.markID, accepted.created, accepted.start, accepted.end)
	t.Log("  Accept the suggestion there, then finish the table with:")
	t.Logf("    GDOC_LIVE_TEST=1 %s=%s go test ./internal/live -run TestLiveNamedRangeAfterAcceptedByHand -v", acceptedVar, accepted.id)
	t.Log("  and trash that document by hand once the answer is written down.")
}

// TestLiveNamedRangeAfterAcceptedByHand is the last row of the probe's table.
// It reads the document the probe left behind, after a person has accepted the
// suggestion in it, and reports what the marker covers now. It reads and writes
// nothing, so it asks for no write variable: the write was Nail's, in the
// browser.
func TestLiveNamedRangeAfterAcceptedByHand(t *testing.T) {
	id := strings.TrimSpace(os.Getenv(acceptedVar))
	if os.Getenv(liveVar) != "1" || id == "" {
		t.Skipf("skipped: set %s=1 and %s=<the document the probe left behind, with its suggestion accepted> to finish the named range table", liveVar, acceptedVar)
	}

	p := guard.NewPolicy()
	p.AllowFile(id, guard.LevelSuggest)
	s, err := gapi.Open(p, nil)
	if err != nil {
		t.Fatalf("the session could not be opened: %v", err)
	}
	d, err := readInline(context.Background(), s, id)
	if err != nil {
		t.Fatalf("the accepted document %q could not be read: %v", id, err)
	}

	spans := namedRangeSpans(d)
	t.Log("")
	t.Log("  a named range after the suggestion under it was accepted")
	t.Log("  ------------------------------------------------------------------------")
	if len(spans) == 0 {
		t.Log("  the marker is GONE: an accepted prelude carries no marker, so a second")
		t.Log("  run cannot find it by one, and M7c falls back to refusing that run")
	}
	for _, sp := range spans {
		t.Logf("  %s", sp)
	}
	if ids := suggestionIDsIn(d); len(ids) != 0 {
		var ks []string
		for k := range ids {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		t.Logf("  note: this document still holds pending suggestions (%s), so the", strings.Join(ks, ", "))
		t.Log("  answer above is not the accepted one yet")
	}
}

// newPreludeProbe creates a document and leaves it in the state M7c's first run
// leaves one in. It fails the test only when the document cannot be created,
// which is the probe not running rather than an answer; everything after that
// is recorded and reported.
func newPreludeProbe(t *testing.T, ctx context.Context, s *gapi.Session, folder, name string) *preludeProbe {
	t.Helper()
	id, err := createDoc(ctx, s, folder, name)
	if err != nil {
		t.Fatalf("the probe document %q could not be created: %v", name, err)
	}
	pr := &preludeProbe{id: id}

	// The author's own text first, directly, because this document is gdoc's
	// own and the point is what the prelude meets rather than what wrote it.
	if err := batch(ctx, s, id, req("insertText", map[string]any{
		"location": map[string]any{"index": 1},
		"text":     namedRangeBody,
	})); err != nil {
		pr.refused = "the author's text could not be written: " + oneLine(err.Error())
		return pr
	}
	// Then the prelude, proposed. This is the request propose sends every day.
	if err := suggestBatch(ctx, s, id, req("insertText", map[string]any{
		"location": map[string]any{"index": 1},
		"text":     preludeLine,
	})); err != nil {
		pr.refused = "the prelude could not be proposed: " + oneLine(err.Error())
		return pr
	}

	d, err := readInline(ctx, s, id)
	if err != nil {
		pr.refused = "the read after the proposal failed: " + oneLine(err.Error())
		return pr
	}
	start, end, sugID, ok := suggestedRun(d)
	if !ok {
		pr.refused = "the read back shows no suggested run, so there is nothing to mark"
		return pr
	}
	pr.start, pr.end, pr.sugID = start, end, sugID

	// The marker itself, direct, because Docs refuses createNamedRange as a
	// suggestion. This document is at LevelFull, so the guard carries it.
	var reply struct {
		Replies []struct {
			CreateNamedRange struct {
				NamedRangeID string `json:"namedRangeId"`
			} `json:"createNamedRange"`
		} `json:"replies"`
	}
	body := map[string]any{"requests": []map[string]any{req("createNamedRange", map[string]any{
		"name":  markerName,
		"range": map[string]any{"startIndex": start, "endIndex": end},
	})}}
	if err := s.PostJSON(ctx, "https://docs.googleapis.com/v1/documents/"+id+":batchUpdate", body, &reply); err != nil {
		pr.refused = oneLine(err.Error())
		return pr
	}
	pr.created = true
	if len(reply.Replies) > 0 {
		pr.markID = reply.Replies[0].CreateNamedRange.NamedRangeID
	}

	after, err := readInline(ctx, s, id)
	if err != nil {
		pr.refused = "the read after the marker failed: " + oneLine(err.Error())
		return pr
	}
	pr.spans = namedRangeSpans(after)
	return pr
}

func trashProbe(t *testing.T, ctx context.Context, s *gapi.Session, id, which string) {
	t.Helper()
	if id == "" {
		return
	}
	if err := drive.Trash(ctx, s, id); err != nil {
		t.Errorf("the %s probe document %q is still in the folder: %v", which, id, err)
	}
}

func spanList(spans []markerSpan) string {
	if len(spans) == 0 {
		return "no named range at all"
	}
	var out []string
	for _, sp := range spans {
		out = append(out, sp.String())
	}
	return strings.Join(out, "; ")
}

// suggestedRun finds the first run the read back marks as a pending insertion,
// and answers its span and the id on it. The span is the element's own, because
// a named range is created over indexes and those are the indexes the read
// gives.
func suggestedRun(d map[string]any) (start, end int, id string, ok bool) {
	for _, e := range content(d) {
		els, _ := dig(e, "paragraph", "elements").([]any)
		for _, el := range els {
			m, isMap := el.(map[string]any)
			if !isMap {
				continue
			}
			ids, _ := dig(m, "textRun", "suggestedInsertionIds").([]any)
			if len(ids) == 0 {
				continue
			}
			first, _ := ids[0].(string)
			return intAt(m, "startIndex"), intAt(m, "endIndex"), first, true
		}
	}
	return 0, 0, "", false
}

// namedRangeSpans flattens a read-back document's named ranges, with the words
// each one covers. Docs keys the map by name and repeats the name inside, and
// two ranges may share one, which is why the id is carried beside it.
func namedRangeSpans(d map[string]any) []markerSpan {
	groups, ok := d["namedRanges"].(map[string]any)
	if !ok {
		return nil
	}
	var out []markerSpan
	for name, group := range groups {
		list, _ := dig(group, "namedRanges").([]any)
		for _, nr := range list {
			m, isMap := nr.(map[string]any)
			if !isMap {
				continue
			}
			rid, _ := m["namedRangeId"].(string)
			ranges, _ := m["ranges"].([]any)
			if len(ranges) == 0 {
				out = append(out, markerSpan{id: rid, name: name})
				continue
			}
			for _, r := range ranges {
				rm, isMap := r.(map[string]any)
				if !isMap {
					continue
				}
				start, end := intAt(rm, "startIndex"), intAt(rm, "endIndex")
				out = append(out, markerSpan{
					id:    rid,
					name:  name,
					start: start,
					end:   end,
					text:  textBetween(d, start, end),
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].start != out[j].start {
			return out[i].start < out[j].start
		}
		return out[i].id < out[j].id
	})
	return out
}

// textBetween reads the words a span covers out of the same answer the span
// came from. The probe writes ASCII, so counting runes is counting what Docs
// counts here; a measurement that had to be right about UTF-16 would read the
// production walk instead.
func textBetween(d map[string]any, start, end int) string {
	var b strings.Builder
	for _, e := range content(d) {
		els, _ := dig(e, "paragraph", "elements").([]any)
		for _, el := range els {
			m, isMap := el.(map[string]any)
			if !isMap {
				continue
			}
			run, isMap := dig(m, "textRun").(map[string]any)
			if !isMap {
				continue
			}
			text, _ := run["content"].(string)
			from, to := intAt(m, "startIndex"), intAt(m, "endIndex")
			if to <= start || from >= end {
				continue
			}
			rs := []rune(text)
			lo, hi := max(start-from, 0), min(end-from, len(rs))
			if lo < hi {
				b.WriteString(string(rs[lo:hi]))
			}
		}
	}
	return b.String()
}

func intAt(m map[string]any, key string) int {
	f, _ := m[key].(float64)
	return int(f)
}
