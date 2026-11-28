package guard

import (
	"bytes"
	"encoding/json"
	"unicode/utf16"
)

// This file holds the edit level: the one place a handed-in document's words
// may change without a suggestion. doc.go, "A fourth write level, and the two
// grammars it carries", holds why each rule is what it is.

// EditYellow is the one colour an edit paints its own insertion, and the one
// colour a clear takes off. It is outside the house palette on purpose: the
// house yellow is a mark the author makes, and a clear that matched it would
// erase their mark. TestTheEditYellowIsNoHouseColour is the pin.
const EditYellow = "#FFF59D"

// The yellow as three channels over 255, which is how a request states it and
// how Docs reads it back. Measured 2026-11-28: the floats come back as sent.
const (
	editRed   = 0xFF / 255.0
	editGreen = 0xF5 / 255.0
	editBlue  = 0x9D / 255.0
)

// editCeiling is the most one edit may delete and the most it may insert, in
// UTF-16 code units. It is a ceiling under the span grant and not the bound:
// the bound is the span the command granted out of its own read.
const editCeiling = 300

// Span is one half-open range of body indexes, [Start, End), in the units the
// Docs API counts.
type Span struct{ Start, End int }

// EditKind is which of the two grammars a span grant is for. A grant for a
// clear carries no edit: `gdoc seen` promises that no word changes, and the
// guard holds that promise rather than the caller.
type EditKind int

const (
	EditWords EditKind = 1 // delete the span, insert words, paint them
	EditClear EditKind = 2 // set the spans' background to no colour
)

// editGrant is what the next batch at the edit level may touch: the spans its
// command read, at the revision it read them in, for one grammar. It is spent
// by the batch it carries.
type editGrant struct {
	kind     EditKind
	revision string
	spans    map[Span]bool
}

// GrantEdit raises an id already in the set at the suggest level to LevelEdit,
// for one run. It is not a door into the set: an id nobody handed in is not
// admitted by asking to edit it.
//
// Nothing writes the grant down, nothing reads it from a file and no setting
// turns it on: it dies with the process. Nail's decision of 2026-11-28.
//
// It never narrows and it never stacks. An id at the full level is left there,
// as GrantInPlace leaves it, and an id at the in-place level is left there
// too, so no run holds both a restyle's reach and an edit's.
// TestGrantEditNeverAdmitsAnUnknownID, TestGrantEditLeavesACreatedDocumentAtFull
// and TestGrantEditRefusesAnInPlaceID are the pins.
func (p *Policy) GrantEdit(id string) {
	lvl, known := p.level(id)
	switch {
	case !known:
		return
	case lvl == LevelFull:
		p.note("GrantEdit on %q changed nothing: the id is at the full level, which a create the guard carried gave it", id)
		return
	case lvl == LevelInPlace:
		p.note("GrantEdit on %q changed nothing: the id is at the in-place level, and one run holds a restyle's reach or an edit's, never both", id)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.files[id] == LevelSuggest {
		p.files[id] = LevelEdit
	}
}

// AllowEditSpans names the spans the next batch at the edit level may delete
// or clear, and the revision it must be sent against. The command reads the
// document, finds its spans in that read, and grants exactly those: the guard
// cannot see a colour or a paragraph mark, so what it holds instead is that
// nothing outside the command's own read is touched.
//
// The grant names its grammar, and it is for one batch. A grant for a clear
// carries no edit over the same spans, and the batch a grant carries spends
// it, so the same body judged a second time is refused: a replay after a write
// that never left would otherwise be carried, and Docs would take it too,
// because the revision had not moved. TestASpanGrantIsForOneGrammar and
// TestASpanGrantIsSpentByTheBatchItCarries are the pins.
//
// A second call replaces the first. A grant the guard cannot read, with no
// kind, no revision, no span, or a span that does not end after it starts,
// opens nothing and takes the standing grant back, which is AllowMarker's
// rule. TestASecondSpanGrantReplacesTheFirst and
// TestAnUnreadableSpanGrantTakesTheStandingOneBack are the pins.
func (p *Policy) AllowEditSpans(kind EditKind, revision string, spans []Span) {
	if kind != EditWords && kind != EditClear {
		p.note("an edit grant names the grammar it is for, an edit or a clear, and this one named neither; nothing may be edited")
		p.revokeEdit()
		return
	}
	if kind == EditWords && len(spans) != 1 {
		p.note("an edit grant for words names exactly one span, and this one named %d; nothing may be edited", len(spans))
		p.revokeEdit()
		return
	}
	if revision == "" {
		p.note("an edit grant carries the revision the spans were read in, and this one named none; nothing may be edited")
		p.revokeEdit()
		return
	}
	if len(spans) == 0 {
		p.note("an edit grant names at least one span, and this one named none; nothing may be edited")
		p.revokeEdit()
		return
	}
	set := make(map[Span]bool, len(spans))
	for _, s := range spans {
		if s.Start < 1 || s.End <= s.Start {
			p.note("the edit grant names the span [%d,%d), which is not a range of body text; nothing may be edited", s.Start, s.End)
			p.revokeEdit()
			return
		}
		set[s] = true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.edit = &editGrant{kind: kind, revision: revision, spans: set}
}

func (p *Policy) revokeEdit() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.edit = nil
}

func (p *Policy) grantedEdit() *editGrant {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.edit
}

// judgeEditBatch is the whole of what a batchUpdate at the edit level may be.
// It runs before any other rule about request kinds, so a grant another door
// opened, a rejectSuggestion or a named range, carries nothing here.
//
// The body is exactly requests and writeControl, writeControl is exactly the
// granted revision under requiredRevisionId, and the requests are one of two
// grammars: an edit, or a clear. Whatever writeMode would have said is not
// asked, because the key itself is refused.
func (p *Policy) judgeEditBatch(top map[string]json.RawMessage) error {
	granted := p.grantedEdit()
	if granted == nil {
		return refuse("this document is at the edit level and the run granted no span, so there is nothing a direct edit may touch")
	}
	if _, ok := exactObject(top, "requests", "writeControl"); !ok {
		return refuse("a batchUpdate at the edit level carries exactly requests and writeControl, spelled exactly")
	}
	control, ok := exactObject(asObject(top["writeControl"]), "requiredRevisionId")
	if !ok {
		return refuse("a batchUpdate at the edit level carries writeControl with requiredRevisionId and nothing else: a write mode is not asked for there, and a target revision merges where this must refuse")
	}
	var revision string
	if json.Unmarshal(control["requiredRevisionId"], &revision) != nil || revision != granted.revision {
		return refuse("a batchUpdate at the edit level is sent against the revision its spans were read in, and this one names another")
	}
	var reqs []map[string]json.RawMessage
	if err := json.Unmarshal(top["requests"], &reqs); err != nil || len(reqs) == 0 {
		return refuse("a batchUpdate at the edit level names its requests as a list, and this one names none that can be read")
	}
	for _, req := range reqs {
		if len(req) != 1 {
			return refuse("a request in this batchUpdate names %d kinds, and a request names one", len(req))
		}
	}
	_, isClear := reqs[0]["updateTextStyle"]
	var err error
	switch {
	case isClear && granted.kind != EditClear:
		err = refuse("this batch is a clear, and the spans this run granted are for an edit")
	case isClear:
		err = judgeClear(reqs, granted)
	case granted.kind != EditWords:
		err = refuse("this batch changes words, and the spans this run granted are for a clear, which changes none")
	default:
		err = judgeEdit(reqs, granted)
	}
	if err != nil {
		return err
	}
	// The grant is spent by the batch it carries. It is taken back only if it
	// is still the one that was judged, so a grant made meanwhile stands.
	p.mu.Lock()
	if p.edit == granted {
		p.edit = nil
	}
	p.mu.Unlock()
	return nil
}

// judgeEdit holds the edit grammar: insert words at the end of one granted
// span, delete that span, paint exactly those words the edit yellow where the
// span started. Three requests, in that order, and nothing beside them.
//
// The insert comes first and at the span's end because Docs gives inserted
// text the style of the character in front of it: there, that is the last of
// the words being replaced. The delete is read first all the same, since it
// is the request that names the span the grant is asked about.
func judgeEdit(reqs []map[string]json.RawMessage, granted *editGrant) error {
	if len(reqs) != 3 {
		return refuse("an edit is exactly three requests, insertText, deleteContentRange and updateTextStyle, and this batch carries %d", len(reqs))
	}
	del, ok := exactObject(asObject(reqs[1]["deleteContentRange"]), "range")
	if !ok {
		return refuse("the second request of an edit is a deleteContentRange carrying range and nothing else")
	}
	span, ok := exactSpan(del["range"])
	if !ok {
		return refuse("the range of an edit's deleteContentRange carries exactly startIndex and endIndex, as whole numbers; a segment or a tab is not named there")
	}
	if !granted.spans[span] {
		return refuse("the edit deletes [%d,%d), which is not a span this run read and granted", span.Start, span.End)
	}
	if span.End-span.Start > editCeiling {
		return refuse("the edit deletes %d units, and one edit replaces at most %d", span.End-span.Start, editCeiling)
	}

	ins, ok := exactObject(asObject(reqs[0]["insertText"]), "location", "text")
	if !ok {
		return refuse("the first request of an edit is an insertText carrying location and text and nothing else")
	}
	loc, ok := exactObject(asObject(ins["location"]), "index")
	if !ok {
		return refuse("the location of an edit's insertText carries exactly index; a segment or a tab is not named there")
	}
	if at, ok := exactIndex(loc["index"]); !ok || at != span.End {
		return refuse("an edit inserts its words at the end of the span it deletes, at %d, and this one inserts elsewhere", span.End)
	}
	at := span.Start
	var text string
	if json.Unmarshal(ins["text"], &text) != nil || text == "" {
		return refuse("an edit inserts words, and this one inserts none")
	}
	for _, r := range text {
		if r < 0x20 || r == 0x7F || r == 0x2028 || r == 0x2029 {
			return refuse("an edit inserts words inside one paragraph, and this text carries U+%04X", r)
		}
		// Docs strips a private-use character out of an inserted text, and
		// a text that cannot be read as Unicode decodes to U+FFFD here while
		// the server reads its own bytes. Either way the units counted below
		// would not be the units that land, and the paint would reach a
		// character this batch did not insert.
		if (r >= 0xE000 && r <= 0xF8FF) || r == 0xFFFD || r == 0xFFFE || r == 0xFFFF || r == 0xFEFF || r == 0x85 {
			return refuse("an edit's text carries U+%04X, which Docs may not store as sent, so the paint could not be held to the insertion", r)
		}
	}
	units := len(utf16.Encode([]rune(text)))
	if units > editCeiling {
		return refuse("the edit inserts %d units, and one edit inserts at most %d", units, editCeiling)
	}

	style, ok := exactObject(asObject(reqs[2]["updateTextStyle"]), "range", "textStyle", "fields")
	if !ok {
		return refuse("the third request of an edit is an updateTextStyle carrying range, textStyle and fields and nothing else")
	}
	painted, ok := exactSpan(style["range"])
	if !ok || painted.Start != at || painted.End != at+units {
		return refuse("an edit paints exactly the words it inserted, which stand at [%d,%d) once the span is deleted, and this one paints another range", at, at+units)
	}
	if !isBackgroundMask(style["fields"]) {
		return refuse("an edit's updateTextStyle names backgroundColor in fields and nothing else")
	}
	ts, ok := exactObject(asObject(style["textStyle"]), "backgroundColor")
	if !ok || !isEditYellow(ts["backgroundColor"]) {
		return refuse("an edit paints its words %s and sets nothing else", EditYellow)
	}
	return nil
}

// judgeClear holds the clear grammar: one updateTextStyle per granted span,
// each setting the background to no colour. It changes no character.
func judgeClear(reqs []map[string]json.RawMessage, granted *editGrant) error {
	for _, req := range reqs {
		style, ok := exactObject(asObject(req["updateTextStyle"]), "range", "textStyle", "fields")
		if !ok {
			return refuse("a clear is updateTextStyle requests and nothing else, each carrying range, textStyle and fields")
		}
		span, ok := exactSpan(style["range"])
		if !ok {
			return refuse("the range of a clear carries exactly startIndex and endIndex, as whole numbers")
		}
		if !granted.spans[span] {
			return refuse("the clear names [%d,%d), which is not a span this run read and granted", span.Start, span.End)
		}
		if !isBackgroundMask(style["fields"]) {
			return refuse("a clear names backgroundColor in fields and nothing else")
		}
		ts, ok := exactObject(asObject(style["textStyle"]), "backgroundColor")
		if !ok {
			return refuse("a clear sets backgroundColor to no colour and sets nothing else")
		}
		if empty, ok := exactObject(asObject(ts["backgroundColor"])); !ok || len(empty) != 0 {
			return refuse("a clear sets backgroundColor to no colour, written as an empty object")
		}
	}
	return nil
}

// asObject reads raw as a JSON object, nil when it is anything else.
func asObject(raw json.RawMessage) map[string]json.RawMessage {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return nil
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

// exactObject says whether m carries exactly the named keys, spelled exactly,
// and nothing beside them. A nil map is no object at all.
func exactObject(m map[string]json.RawMessage, keys ...string) (map[string]json.RawMessage, bool) {
	if m == nil || len(m) != len(keys) {
		return nil, false
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return nil, false
		}
	}
	return m, true
}

// exactSpan reads a range carrying exactly startIndex and endIndex.
func exactSpan(raw json.RawMessage) (Span, bool) {
	m, ok := exactObject(asObject(raw), "startIndex", "endIndex")
	if !ok {
		return Span{}, false
	}
	start, ok := exactIndex(m["startIndex"])
	if !ok {
		return Span{}, false
	}
	end, ok := exactIndex(m["endIndex"])
	if !ok || end <= start {
		return Span{}, false
	}
	return Span{Start: start, End: end}, true
}

func isBackgroundMask(raw json.RawMessage) bool {
	var mask string
	return json.Unmarshal(raw, &mask) == nil && mask == "backgroundColor"
}

// isEditYellow reads one OptionalColor and says whether it is the edit yellow,
// stated as an rgbColor with all three channels and nothing beside them.
func isEditYellow(raw json.RawMessage) bool {
	outer, ok := exactObject(asObject(raw), "color")
	if !ok {
		return false
	}
	colour, ok := exactObject(asObject(outer["color"]), "rgbColor")
	if !ok {
		return false
	}
	rgb, ok := exactObject(asObject(colour["rgbColor"]), "red", "green", "blue")
	if !ok {
		return false
	}
	for name, want := range map[string]float64{"red": editRed, "green": editGreen, "blue": editBlue} {
		var got float64
		if json.Unmarshal(rgb[name], &got) != nil || got != want {
			return false
		}
	}
	return true
}
