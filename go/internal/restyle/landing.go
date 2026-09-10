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
// That case is why the page check reads more than the value it sent back.
// updateDocumentStyle writes documentStyle, so on a document whose section
// overrides a margin the request reads back exactly as it was sent while the
// page a reader sees never moved: comparing the two alone would answer held on
// the one document this check was written for. sectionOverrides is where that
// lives, and an overridden field is reported as no answer.
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
// per paragraph and one per table, so checking every one would be hundreds
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
		c := Check{Kind: kind, Where: "the document"}
		// The page geometry is the one check whose own answer can be beside the
		// point. A section break carrying its own margins governs what a reader
		// sees on its pages, and updateSectionStyle is not a kind this level
		// carries, so the request lands in documentStyle, reads back exactly as
		// it was sent, and changes nothing. That is the shape the fidelity
		// measurement of 2026-09-09 found, and reading documentStyle alone is
		// reporting a page as restyled on the one document where it is not.
		//
		// So an overridden field is no answer rather than a held check. A
		// section that sets none of the fields the request set overrides
		// nothing, because an unset section margin is the document's own, and
		// neither does one restating the value that was sent, which is why the
		// note fires on the difference and not on the section break.
		if over := d.sectionOverrides(r["documentStyle"]); len(over) > 0 {
			return unanswered(c, fmt.Sprintf(
				"a section break in this document sets its own %s, and a section's own value is what a reader sees, "+
					"so nothing here says the page geometry landed: updateSectionStyle is not a request kind this level carries",
				strings.Join(over, ", ")))
		}
		return compare(c, r["documentStyle"], d.documentStyle())
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
		start, ok := cellAt(r)
		c := Check{Kind: kind, Where: fmt.Sprintf("the first cell of the table at %d, which the request styled whole", start)}
		if !ok {
			return unanswered(c, "the request named no table, so there was nowhere to look")
		}
		style, ok := d.firstCellStyle(start)
		if !ok {
			return unanswered(c, "the read back holds no cell in a table starting there")
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
	c.Held = false
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

// cellAt is the table an updateTableCellStyle addressed.
//
// It reads the one shape a restyle sends, tableStartLocation, which names every
// cell in the table, so there is no row and no column to return: the request
// addressed all of them. The cell read back is then the first one, which is
// inside the request like every other cell, and reading one of them is the same
// rule as reading the first request of each kind. A request naming a tableRange
// instead is a subset this builder never asks for, so it is no answer rather
// than a second reading nothing sends: the milestone that builds one reads it
// here, beside its caller.
func cellAt(r map[string]any) (start int, ok bool) {
	at, whole := r["tableStartLocation"].(map[string]any)
	if !whole {
		return 0, false
	}
	n, ok := number(at["index"])
	return int(n), ok
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
	// SectionBreak is read for its style alone, and only because a section's
	// own margins override the page geometry the run sent. Nothing else here
	// reads it, and nothing may: this file reads styling, not structure.
	SectionBreak *struct {
		SectionStyle map[string]any `json:"sectionStyle"`
	} `json:"sectionBreak"`
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

// sectionOverrides names the fields of the page request that a section break
// states differently for itself, once each and in one order.
//
// It walks the top level of the body alone, because that is where a section
// break sits: one inside a table cell is not a shape Docs has. The fields are
// matched by name at the top of the style object, which is the granularity
// SectionStyle shares with DocumentStyle: the four margins by those names, and
// nothing else the page request sets.
//
// flipPageOrientation is the one field matched by no name, and it is here
// because the page request can never carry it. The reference: it "indicates
// whether to flip the dimensions of DocumentStyle's page_size for this section",
// and unset it inherits the document's own. So a section that states it
// differently shows the pageSize that was sent transposed, which is the accepted
// -and-invisible shape again, one field along: the request reads back exactly as
// it was sent while the pages a reader turns are the other way up. Differently
// rather than true, for the reason the margins are matched differently: a
// section agreeing with the document overrides nothing anybody could see, and a
// document already flipped throughout was flipped before this run.
//
// Differently, because a section restating the value the request sent overrides
// nothing a reader could see. A section margin governs its pages only when it
// is set, so an explicitly equal one leaves the visible geometry the one that
// was asked for, and the check can answer. Naming it all the same would be a
// warning that fires on the working case, which is the defect MissingScopes was
// fixed for. The comparison is missingFields, the same rule the checks
// themselves are made of, so one tolerance decides both.
func (d *styledDoc) sectionOverrides(want any) []string {
	fields, ok := want.(map[string]any)
	if !ok || len(fields) == 0 {
		return nil
	}
	_, sizeAsked := fields["pageSize"]
	docFlip, _ := d.documentStyle()[flipField].(bool)

	seen := map[string]bool{}
	for _, el := range d.content() {
		if el.SectionBreak == nil {
			continue
		}
		for name, got := range el.SectionBreak.SectionStyle {
			asked, there := fields[name]
			if !there {
				continue
			}
			if len(missingFields("", asked, got)) == 0 {
				continue
			}
			seen[name] = true
		}
		if !sizeAsked {
			continue
		}
		if flip, ok := el.SectionBreak.SectionStyle[flipField].(bool); ok && flip != docFlip {
			seen[flipField] = true
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	// Sorted, because a map is walked in no order and one read-back must name
	// the same fields in the same order twice.
	sort.Strings(out)
	return out
}

// flipField is the one SectionStyle field read for itself rather than matched
// against the request, because no request this level carries can set it.
const flipField = "flipPageOrientation"

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

// firstCellStyle is the first cell of the table starting at an index. A table
// inside a cell carries its own start index, so the walk recurses rather than
// answering with the outer table's.
//
// The first cell rather than a named one, because the request this reads back
// names the table and styles every cell in it. A row whose first two columns
// are merged carries one cell object for the pair, so a row and a column index
// would be positions the request never spoke in.
func (d *styledDoc) firstCellStyle(start int) (map[string]any, bool) {
	t := tableIn(d.content(), start)
	if t == nil || len(t.Rows) == 0 || len(t.Rows[0].Cells) == 0 {
		return nil, false
	}
	return t.Rows[0].Cells[0].Style, true
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
