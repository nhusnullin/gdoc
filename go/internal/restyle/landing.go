package restyle

// The landing half of the read-back: whether the style gdoc sent is actually in
// the document.
//
// It exists because a batch Docs accepted is not a change a reader can see. The
// fidelity measurement of 2026-09-09 found exactly that shape: a request kind
// that answers 200 and lands nothing. updateDocumentStyle is the live example
// here, because a document carrying section breaks has its margins governed by
// its sectionStyle and updateSectionStyle is not on the in-place allowlist. A
// run reporting verified: true over an invisible change would be that failure
// printed as a success.
//
// The check is made against the requests that were sent rather than against the
// house style, and that is the point of it. A check written from house.yaml
// would answer the question the builder already answers, and the two would drift
// the first time a builder stopped setting a field. This one asks the only
// question left: the document was told these values, does it carry them.
//
// It reads the styling out of the Docs answer itself, because internal/docs
// decodes structure and text and no styling at all. That was M7b's Task 3
// decision and it stands: a field with no reader drifts, and the one reader is
// here.

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// The four kinds the in-place level carries, in the order a check reports them.
// A kind the run did not send is not checked, because there is nothing to look
// for: a document with no table sends no updateTableCellStyle.
var checkedKinds = []string{
	"updateDocumentStyle", "updateParagraphStyle", "updateTextStyle", "updateTableCellStyle",
}

// tolerance is how close a number read back has to be to the number sent. The
// house style states 51.05pt and the API answers in floats it round-trips
// through its own precision, and a colour channel written as 201/255 comes back
// as a float32's worth of that. A tenth of a thousandth of a point is closer
// than anything a reader could see and wider than any of that.
const tolerance = 0.001

// Landing is the read-back's landing half: one check per request kind the run
// sent, each naming where it looked.
type Landing struct {
	Checks []Check `json:"checks"`
}

// Check is one kind read back. Held says the document carries every field that
// request set. Missing names the ones it does not, by their path inside the
// style object, and Note is why there was no answer at all: a paragraph the
// read-back could not find is not a paragraph whose style did not land.
type Check struct {
	Kind    string   `json:"kind"`
	Where   string   `json:"where"`
	Held    bool     `json:"held"`
	Missing []string `json:"missing,omitempty"`
	Note    string   `json:"note,omitempty"`
}

// Landed reads the document back and asks, for the first request of each kind
// the run sent, whether what it set is there.
//
// The first of each kind rather than all of them. A restyle sends one request
// per paragraph and one per table row, so checking every one would be hundreds
// of lookups answering the same question, and what a caller needs to know is
// whether the kind reached the document at all. Which one it read is named in
// the check, so nobody has to guess where it looked.
func Landed(raw []byte, sent []map[string]any) (Landing, []string) {
	out := Landing{Checks: []Check{}}
	d, err := parseStyled(raw)
	if err != nil {
		return out, []string{fmt.Sprintf(
			"the document was read back and could not be decoded for its styling, so nothing here says the style landed: %v", err)}
	}
	for _, kind := range checkedKinds {
		r := firstOfKind(sent, kind)
		if r == nil {
			continue
		}
		out.Checks = append(out.Checks, d.check(kind, r))
	}
	return out, nil
}

// firstOfKind is the first request of one kind, in the order they were sent.
func firstOfKind(sent []map[string]any, kind string) map[string]any {
	for _, r := range sent {
		if inner, ok := r[kind].(map[string]any); ok {
			return inner
		}
	}
	return nil
}

// check is one kind: find what the request addressed, then compare.
func (d *styledDoc) check(kind string, r map[string]any) Check {
	switch kind {
	case "updateDocumentStyle":
		return compare(Check{Kind: kind, Where: "the document"}, r["documentStyle"], d.documentStyle())
	case "updateParagraphStyle":
		start, ok := rangeStart(r)
		c := Check{Kind: kind, Where: fmt.Sprintf("the paragraph at %d", start)}
		if !ok {
			return unanswered(c, "the request named no range, so there was nowhere to look")
		}
		par := d.paragraphAt(start)
		if par == nil {
			return unanswered(c, "the read back holds no paragraph starting there")
		}
		return compare(c, r["paragraphStyle"], par.ParagraphStyle)
	case "updateTextStyle":
		start, ok := rangeStart(r)
		c := Check{Kind: kind, Where: fmt.Sprintf("the first text run of the paragraph at %d", start)}
		if !ok {
			return unanswered(c, "the request named no range, so there was nowhere to look")
		}
		par := d.paragraphAt(start)
		if par == nil {
			return unanswered(c, "the read back holds no paragraph starting there")
		}
		style, ok := par.firstTextStyle()
		if !ok {
			return unanswered(c, "that paragraph holds no text run, so there is no run style to read")
		}
		return compare(c, r["textStyle"], style)
	case "updateTableCellStyle":
		start, row, col, ok := cellAt(r)
		c := Check{Kind: kind, Where: fmt.Sprintf("row %d, column %d of the table at %d", row, col, start)}
		if !ok {
			return unanswered(c, "the request named no cell, so there was nowhere to look")
		}
		style, ok := d.cellStyle(start, row, col)
		if !ok {
			return unanswered(c, "the read back holds no such cell in a table starting there")
		}
		return compare(c, r["tableCellStyle"], style)
	}
	return unanswered(Check{Kind: kind}, "this kind is not one the read-back reads")
}

// unanswered is a check with nothing to compare. It is never held: a lookup
// that found nothing says the style is not provably there, which is the honest
// answer and the one that keeps verified false.
func unanswered(c Check, note string) Check {
	c.Note = note
	return c
}

// compare is the whole comparison rule: the document holds every field the
// request set, and nothing is said about the fields it did not.
func compare(c Check, want, got any) Check {
	missing := missingFields("", want, got)
	c.Missing = missing
	c.Held = len(missing) == 0
	if want == nil {
		return unanswered(c, "the request set no style object, so there was nothing to look for")
	}
	return c
}

// missingFields is want against got, by path, and it walks want alone: a field
// the document carries that the request never set is the author's and is no
// business of this check.
//
// A field the read does not carry at all is the document's own default, because
// Docs leaves a property equal to its default out of the answer. So a zero the
// request asked for holds when the read is silent, and any other value does not.
func missingFields(path string, want, got any) []string {
	switch w := want.(type) {
	case map[string]any:
		g, _ := got.(map[string]any)
		names := make([]string, 0, len(w))
		for k := range w {
			names = append(names, k)
		}
		// Sorted, because a map is walked in no order and one read-back must
		// name the same fields in the same order twice.
		sort.Strings(names)
		var out []string
		for _, k := range names {
			sub, there := g[k]
			if !there {
				if isDefault(w[k]) {
					continue
				}
				out = append(out, join(path, k))
				continue
			}
			out = append(out, missingFields(join(path, k), w[k], sub)...)
		}
		return out
	case string:
		if s, ok := got.(string); ok && s == w {
			return nil
		}
		return []string{orRoot(path)}
	default:
		wn, wok := number(want)
		gn, gok := number(got)
		if wok && gok && math.Abs(wn-gn) <= tolerance {
			return nil
		}
		if !wok && fmt.Sprintf("%v", want) == fmt.Sprintf("%v", got) {
			return nil
		}
		return []string{orRoot(path)}
	}
}

// isDefault says whether a value the read did not carry is one Docs would have
// left out. A zero is, and a dimension of zero magnitude is: everything else,
// absent, is a field that did not land.
func isDefault(v any) bool {
	if n, ok := number(v); ok {
		return n == 0
	}
	if m, ok := v.(map[string]any); ok {
		mag, there := m["magnitude"]
		if !there {
			return false
		}
		n, ok := number(mag)
		return ok && n == 0
	}
	return false
}

// number is one JSON number, whichever Go type it arrived as. A request is
// built in Go and read back through encoding/json, so the same magnitude is an
// int on one side and a float64 on the other.
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// orRoot names the field that did not hold. A comparison at the very top has no
// path, which happens when a request sets a style object that is not an object
// at all, so it says so rather than naming an empty string.
func orRoot(path string) string {
	if path == "" {
		return "(the style object itself)"
	}
	return path
}

// rangeStart is the index a styling request addressed.
func rangeStart(r map[string]any) (int, bool) {
	span, ok := r["range"].(map[string]any)
	if !ok {
		return 0, false
	}
	n, ok := number(span["startIndex"])
	return int(n), ok
}

// cellAt is the table, row and column an updateTableCellStyle addressed.
func cellAt(r map[string]any) (start, row, col int, ok bool) {
	tr, ok := r["tableRange"].(map[string]any)
	if !ok {
		return 0, 0, 0, false
	}
	loc, ok := tr["tableCellLocation"].(map[string]any)
	if !ok {
		return 0, 0, 0, false
	}
	at, ok := loc["tableStartLocation"].(map[string]any)
	if !ok {
		return 0, 0, 0, false
	}
	n, ok := number(at["index"])
	if !ok {
		return 0, 0, 0, false
	}
	r0, _ := number(loc["rowIndex"])
	c0, _ := number(loc["columnIndex"])
	return int(n), int(r0), int(c0), true
}

// styledDoc is the Docs answer read for its styling alone: the document style,
// and the paragraph, run and cell styles inside the first tab.
//
// The first tab, because the command refuses a document with more than one
// before it opens the grant, and a style request names a range which means
// nothing without saying which tab it is in. A document written before tabs
// existed answers with a top-level body instead, and that is read too, exactly
// as internal/docs reads it.
type styledDoc struct {
	Style map[string]any `json:"documentStyle"`
	Body  *styledBody    `json:"body"`
	Tabs  []struct {
		DocumentTab *struct {
			Style map[string]any `json:"documentStyle"`
			Body  *styledBody    `json:"body"`
		} `json:"documentTab"`
	} `json:"tabs"`
}

type styledBody struct {
	Content []styledElement `json:"content"`
}

type styledElement struct {
	StartIndex int              `json:"startIndex"`
	Paragraph  *styledParagraph `json:"paragraph"`
	Table      *styledTable     `json:"table"`
}

type styledParagraph struct {
	ParagraphStyle map[string]any `json:"paragraphStyle"`
	Elements       []struct {
		TextRun *struct {
			TextStyle map[string]any `json:"textStyle"`
		} `json:"textRun"`
	} `json:"elements"`
}

type styledTable struct {
	Rows []struct {
		Cells []struct {
			Style   map[string]any  `json:"tableCellStyle"`
			Content []styledElement `json:"content"`
		} `json:"tableCells"`
	} `json:"tableRows"`
}

func parseStyled(raw []byte) (*styledDoc, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, fmt.Errorf("the read back carried no bytes")
	}
	var d styledDoc
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// documentStyle is the page geometry as the read back carries it.
func (d *styledDoc) documentStyle() map[string]any {
	if len(d.Tabs) > 0 && d.Tabs[0].DocumentTab != nil {
		return d.Tabs[0].DocumentTab.Style
	}
	return d.Style
}

// content is the first tab's body, or the pre-tabs body beside it.
func (d *styledDoc) content() []styledElement {
	if len(d.Tabs) > 0 && d.Tabs[0].DocumentTab != nil && d.Tabs[0].DocumentTab.Body != nil {
		return d.Tabs[0].DocumentTab.Body.Content
	}
	if d.Body != nil {
		return d.Body.Content
	}
	return nil
}

// paragraphAt is the paragraph starting at one index, wherever it sits. The
// walk goes into table cells, because a restyle styles the paragraphs in them.
func (d *styledDoc) paragraphAt(start int) *styledParagraph {
	return paragraphIn(d.content(), start)
}

func paragraphIn(content []styledElement, start int) *styledParagraph {
	for _, el := range content {
		if el.Paragraph != nil && el.StartIndex == start {
			return el.Paragraph
		}
		if el.Table == nil {
			continue
		}
		for _, row := range el.Table.Rows {
			for _, cell := range row.Cells {
				if p := paragraphIn(cell.Content, start); p != nil {
					return p
				}
			}
		}
	}
	return nil
}

// firstTextStyle is the style of the first text run in a paragraph. A paragraph
// holding only a chip, a break or a picture has none, and the caller says so
// rather than reporting a run style that is not there.
func (p *styledParagraph) firstTextStyle() (map[string]any, bool) {
	for _, e := range p.Elements {
		if e.TextRun != nil {
			return e.TextRun.TextStyle, true
		}
	}
	return nil, false
}

// cellStyle is one cell of the table starting at an index. A table inside a
// cell carries its own start index, so the walk recurses rather than answering
// with the outer table's.
func (d *styledDoc) cellStyle(start, row, col int) (map[string]any, bool) {
	t := tableIn(d.content(), start)
	if t == nil || row >= len(t.Rows) || col >= len(t.Rows[row].Cells) {
		return nil, false
	}
	return t.Rows[row].Cells[col].Style, true
}

func tableIn(content []styledElement, start int) *styledTable {
	for _, el := range content {
		if el.Table == nil {
			continue
		}
		if el.StartIndex == start {
			return el.Table
		}
		for _, row := range el.Table.Rows {
			for _, cell := range row.Cells {
				if t := tableIn(cell.Content, start); t != nil {
					return t
				}
			}
		}
	}
	return nil
}
