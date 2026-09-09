package live

// The fidelity probe, M7b's first question and the reason its plan waited.
//
// The 2026-08-29 run measured survival: what an in-place batchUpdate does NOT
// destroy. It never measured fidelity: how much of the house style an in-place
// write can actually apply. Those are different questions, and SPEC's sentence
// "the house style is approximate" has never been a list.
//
// This is the list. It creates its own document, writes known content into it,
// sends each candidate request kind in its own batch so one refusal cannot hide
// the rest, reads the document back, and reports three things per kind: whether
// Docs accepted the request, whether the read-back shows the value, and what it
// shows instead when it does not.
//
// It asserts almost nothing on purpose. A measurement that fails the build when
// Google answers differently than expected is a measurement that has already
// decided the answer. It fails only when it cannot create or cannot trash, and
// the table it logs is the output. Nail reads it and scopes M7b from it.
//
// Every document it touches is one it made, so the policy has one door open,
// AllowCreateIn on the folder, and the created id reaches LevelFull through the
// create the guard itself carried. No new guard level is needed to ask this
// question, which is why it runs before M7b rather than inside it.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"gdoc/internal/drive"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
)

// fidelityDoc is the content the probe styles. One heading, one body paragraph,
// two list items and a two-by-two table, which between them carry every shape
// house.yaml has an opinion about.
const fidelityBody = "Supplier review\n" +
	"Altery reviews each supplier annually and records the outcome in the register.\n" +
	"First item\n" +
	"Second item\n"

// styleCase is one request kind, the body it is sent with, and what the
// read-back has to find for the request to count as landed.
type styleCase struct {
	name string
	// req is the batchUpdate request, built once the document's indexes are
	// known.
	req func(end int, tableStart int) map[string]any
	// want describes what to look for in the read-back, in words. The check
	// itself is `read`, because what counts as landed differs per kind.
	want string
	read func(doc map[string]any) (got string, ok bool)
}

func TestLiveStyleFidelity(t *testing.T) {
	if os.Getenv(liveVar) != "1" || os.Getenv(writeVar) != "1" {
		t.Skipf("skipped: set %s=1 and %s=1 to create a document in the Drive test folder and style it; %s names another folder", liveVar, writeVar, folderVar)
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

	id, err := createDoc(ctx, s, folder, "gdoc FIDELITY PROBE - what in-place styling reaches")
	if err != nil {
		t.Fatalf("the probe document could not be created: %v", err)
	}
	t.Logf("probe document %s in folder %s", id, folder)
	defer func() {
		if err := drive.Trash(ctx, s, id); err != nil {
			t.Errorf("the probe document %q is still in the folder: %v", id, err)
		}
	}()

	if err := writeBody(ctx, s, id); err != nil {
		t.Fatalf("the content could not be written: %v", err)
	}

	// One batch per case. A rejected request answers 400 and takes only its own
	// batch with it, so the kinds after it are still measured.
	type result struct {
		name     string
		accepted bool
		reason   string
		want     string
		got      string
		landed   bool
	}
	var results []result

	// The table's index is read before the cases run, because one of them
	// addresses the table by it.
	start := 0
	if pre, err := readRaw(ctx, s, id); err == nil {
		if ts, ok := tableStart(pre); ok {
			start = ts
			t.Logf("table starts at index %d", start)
		} else {
			t.Log("no table in the document, so updateTableCellStyle cannot be measured")
		}
	}

	for _, c := range fidelityCases() {
		r := result{name: c.name, want: c.want}
		err := batch(ctx, s, id, c.req(0, start))
		if err != nil {
			r.reason = oneLine(err.Error())
		} else {
			r.accepted = true
		}
		results = append(results, r)
	}

	doc, err := readRaw(ctx, s, id)
	if err != nil {
		t.Fatalf("the read-back failed, so nothing above can be confirmed: %v", err)
	}
	for i, c := range fidelityCases() {
		if !results[i].accepted {
			continue
		}
		got, ok := c.read(doc)
		results[i].got, results[i].landed = got, ok
	}

	t.Log("")
	t.Log("FIDELITY: what an in-place styling request reaches")
	t.Log("kind                          accepted  landed  want / got")
	t.Log(strings.Repeat("-", 78))
	for _, r := range results {
		acc, land := "no", "no"
		if r.accepted {
			acc = "yes"
		}
		if r.landed {
			land = "yes"
		}
		detail := r.want
		if r.accepted && !r.landed {
			detail = fmt.Sprintf("%s / got %q", r.want, r.got)
		}
		if !r.accepted {
			detail = fmt.Sprintf("refused: %s", r.reason)
		}
		t.Logf("%-28s  %-8s  %-6s  %s", r.name, acc, land, detail)
	}
	t.Log(strings.Repeat("-", 78))
	t.Log("A kind accepted but not landed is the interesting row: Docs took the")
	t.Log("request and the document does not show it. That is the shape SPEC's")
	t.Log("word \"approximate\" has been standing in for.")
}

// fidelityCases is the list. Each one is a request kind house.yaml needs if the
// in-place restyle is to reach the value it states.
func fidelityCases() []styleCase {
	return []styleCase{
		{
			name: "updateDocumentStyle margins",
			req: func(int, int) map[string]any {
				return req("updateDocumentStyle", map[string]any{
					"documentStyle": map[string]any{
						"marginTop":    dim(72),
						"marginBottom": dim(72),
						"marginLeft":   dim(56.7),
						"marginRight":  dim(56.7),
					},
					"fields": "marginTop,marginBottom,marginLeft,marginRight",
				})
			},
			want: "marginLeft 56.7pt",
			read: func(d map[string]any) (string, bool) {
				v := dig(d, "documentStyle", "marginLeft", "magnitude")
				return fmt.Sprint(v), near(v, 56.7)
			},
		},
		{
			name: "updateParagraphStyle named",
			req: func(int, int) map[string]any {
				return req("updateParagraphStyle", map[string]any{
					"range":          rng(1, 16),
					"paragraphStyle": map[string]any{"namedStyleType": "HEADING_1"},
					"fields":         "namedStyleType",
				})
			},
			want: "first paragraph is HEADING_1",
			read: func(d map[string]any) (string, bool) {
				v := firstParagraphStyle(d, "namedStyleType")
				return fmt.Sprint(v), v == "HEADING_1"
			},
		},
		{
			name: "updateParagraphStyle spacing",
			req: func(int, int) map[string]any {
				return req("updateParagraphStyle", map[string]any{
					"range": rng(1, 16),
					"paragraphStyle": map[string]any{
						"spaceAbove":      dim(18),
						"spaceBelow":      dim(6),
						"indentFirstLine": dim(0),
					},
					"fields": "spaceAbove,spaceBelow,indentFirstLine",
				})
			},
			want: "spaceAbove 18pt",
			read: func(d map[string]any) (string, bool) {
				v := dig(firstParagraph(d), "paragraphStyle", "spaceAbove", "magnitude")
				return fmt.Sprint(v), near(v, 18)
			},
		},
		{
			name: "updateTextStyle font and size",
			req: func(int, int) map[string]any {
				return req("updateTextStyle", map[string]any{
					"range": rng(1, 16),
					"textStyle": map[string]any{
						"weightedFontFamily": map[string]any{"fontFamily": "Georgia"},
						"fontSize":           dim(16),
					},
					"fields": "weightedFontFamily,fontSize",
				})
			},
			want: "first run is Georgia 16pt",
			read: func(d map[string]any) (string, bool) {
				f := dig(firstRun(d), "textStyle", "weightedFontFamily", "fontFamily")
				return fmt.Sprint(f), f == "Georgia"
			},
		},
		{
			name: "updateTextStyle colour",
			req: func(int, int) map[string]any {
				return req("updateTextStyle", map[string]any{
					"range": rng(1, 16),
					"textStyle": map[string]any{
						"foregroundColor": rgb(0.11, 0.21, 0.36),
					},
					"fields": "foregroundColor",
				})
			},
			want: "first run carries a foreground colour",
			read: func(d map[string]any) (string, bool) {
				v := dig(firstRun(d), "textStyle", "foregroundColor", "color", "rgbColor", "blue")
				return fmt.Sprint(v), v != nil
			},
		},
		{
			name: "updateTextStyle backgroundColor",
			req: func(int, int) map[string]any {
				return req("updateTextStyle", map[string]any{
					"range":     rng(18, 30),
					"textStyle": map[string]any{"backgroundColor": rgb(1, 1, 0)},
					"fields":    "backgroundColor",
				})
			},
			want: "a highlight, which house.yaml states as an OOXML name",
			read: func(d map[string]any) (string, bool) {
				runs := allRuns(d)
				for _, r := range runs {
					if dig(r, "textStyle", "backgroundColor", "color", "rgbColor") != nil {
						return "present", true
					}
				}
				return "absent", false
			},
		},
		{
			name: "createParagraphBullets",
			req: func(int, int) map[string]any {
				return req("createParagraphBullets", map[string]any{
					"range":        rng(95, 118),
					"bulletPreset": "BULLET_DISC_CIRCLE_SQUARE",
				})
			},
			want: "a paragraph carries a bullet",
			read: func(d map[string]any) (string, bool) {
				for _, p := range allParagraphs(d) {
					if p["bullet"] != nil {
						return "present", true
					}
				}
				return "absent", false
			},
		},
		{
			name: "createNamedRange",
			req: func(int, int) map[string]any {
				return req("createNamedRange", map[string]any{
					"name":  "gdoc-checklist",
					"range": rng(1, 16),
				})
			},
			want: "the document reports a named range",
			read: func(d map[string]any) (string, bool) {
				nr, ok := d["namedRanges"].(map[string]any)
				return fmt.Sprint(len(nr)), ok && len(nr) > 0
			},
		},
		{
			name: "updateParagraphStyle borders",
			req: func(int, int) map[string]any {
				return req("updateParagraphStyle", map[string]any{
					"range": rng(1, 16),
					"paragraphStyle": map[string]any{
						"borderBottom": map[string]any{
							"color":     rgb(0.8, 0.1, 0.1),
							"width":     dim(1),
							"padding":   dim(2),
							"dashStyle": "SOLID",
						},
					},
					"fields": "borderBottom",
				})
			},
			want: "the heading rule house.yaml draws under a heading",
			read: func(d map[string]any) (string, bool) {
				v := dig(firstParagraph(d), "paragraphStyle", "borderBottom", "width", "magnitude")
				return fmt.Sprint(v), v != nil
			},
		},
		{
			name: "updateTableCellStyle",
			req: func(_ int, tableStart int) map[string]any {
				return req("updateTableCellStyle", map[string]any{
					"tableRange": map[string]any{
						"tableCellLocation": map[string]any{
							"tableStartLocation": map[string]any{"index": tableStart},
							"rowIndex":           0,
							"columnIndex":        0,
						},
						"rowSpan":    1,
						"columnSpan": 2,
					},
					"tableCellStyle": map[string]any{
						"backgroundColor": rgb(0.85, 0.89, 0.94),
						"paddingTop":      dim(3),
						"paddingBottom":   dim(3),
					},
					"fields": "backgroundColor,paddingTop,paddingBottom",
				})
			},
			want: "the header row house.yaml shades and pads",
			read: func(d map[string]any) (string, bool) {
				for _, e := range content(d) {
					m, ok := e.(map[string]any)
					if !ok || m["table"] == nil {
						continue
					}
					rows, _ := dig(m, "table", "tableRows").([]any)
					if len(rows) == 0 {
						continue
					}
					cells, _ := dig(rows[0], "tableCells").([]any)
					if len(cells) == 0 {
						continue
					}
					v := dig(cells[0], "tableCellStyle", "backgroundColor", "color", "rgbColor")
					return fmt.Sprint(v != nil), v != nil
				}
				return "no table found", false
			},
		},
		{
			name: "updateParagraphStyle tab stops",
			req: func(int, int) map[string]any {
				// house.yaml lays the header and footer out with tab stops. The
				// reference marks ParagraphStyle.tabStops read-only, so this is
				// expected to be accepted and not land, which is the row this
				// whole probe exists to find.
				return req("updateParagraphStyle", map[string]any{
					"range": rng(1, 16),
					"paragraphStyle": map[string]any{
						"tabStops": []any{
							map[string]any{"offset": dim(240), "alignment": "CENTER"},
							map[string]any{"offset": dim(480), "alignment": "END"},
						},
					},
					"fields": "tabStops",
				})
			},
			want: "the tab stops house.yaml lays the running head out with",
			read: func(d map[string]any) (string, bool) {
				v, _ := dig(firstParagraph(d), "paragraphStyle", "tabStops").([]any)
				return fmt.Sprintf("%d stops", len(v)), len(v) > 0
			},
		},
		{
			name: "createHeader FIRST_PAGE",
			req: func(int, int) map[string]any {
				// DECISIONS.md records that HeaderFooterType is exactly
				// UNSPECIFIED and DEFAULT, so a first-page header cannot be
				// created. It is measured rather than trusted, because the
				// whole finishing checklist exists because of it.
				return req("createHeader", map[string]any{"type": "FIRST_PAGE_HEADER"})
			},
			want: "a first-page header, where the logo lives",
			read: func(d map[string]any) (string, bool) {
				v := dig(d, "documentStyle", "firstPageHeaderId")
				return fmt.Sprint(v), v != nil
			},
		},
		{
			name: "updateNamedStyle (expected absent)",
			req: func(int, int) map[string]any {
				// There is no such request kind. Sending it is the measurement:
				// the refusal is the evidence that the nine named styles cannot
				// be redefined, which is the single biggest limit on what an
				// in-place restyle can mean.
				return req("updateNamedStyle", map[string]any{
					"namedStyleType": "HEADING_1",
					"textStyle":      map[string]any{"fontSize": dim(20)},
					"fields":         "fontSize",
				})
			},
			want: "no such request kind; the refusal is the point",
			read: func(map[string]any) (string, bool) { return "", false },
		},
	}
}

// --- the small helpers the cases are written with ---

func req(kind string, body map[string]any) map[string]any {
	return map[string]any{kind: body}
}

func dim(pt float64) map[string]any {
	return map[string]any{"magnitude": pt, "unit": "PT"}
}

func rgb(r, g, b float64) map[string]any {
	return map[string]any{"color": map[string]any{"rgbColor": map[string]any{"red": r, "green": g, "blue": b}}}
}

func rng(start, end int) map[string]any {
	return map[string]any{"startIndex": start, "endIndex": end}
}

func near(v any, want float64) bool {
	f, ok := v.(float64)
	return ok && f > want-0.5 && f < want+0.5
}

// dig walks a decoded JSON object by key, answering nil at the first key that
// is not there. A measurement reads a shape nobody has pinned yet, so every
// step has to survive the value being absent.
func dig(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

func content(d map[string]any) []any {
	c, _ := dig(d, "body", "content").([]any)
	return c
}

func allParagraphs(d map[string]any) []map[string]any {
	var out []map[string]any
	for _, e := range content(d) {
		if p, ok := dig(e, "paragraph").(map[string]any); ok {
			out = append(out, p)
		}
	}
	return out
}

func firstParagraph(d map[string]any) map[string]any {
	ps := allParagraphs(d)
	if len(ps) == 0 {
		return nil
	}
	return ps[0]
}

func firstParagraphStyle(d map[string]any, key string) any {
	return dig(firstParagraph(d), "paragraphStyle", key)
}

func allRuns(d map[string]any) []map[string]any {
	var out []map[string]any
	for _, p := range allParagraphs(d) {
		els, _ := p["elements"].([]any)
		for _, e := range els {
			if r, ok := dig(e, "textRun").(map[string]any); ok {
				out = append(out, r)
			}
		}
	}
	return out
}

func firstRun(d map[string]any) map[string]any {
	rs := allRuns(d)
	if len(rs) == 0 {
		return nil
	}
	return rs[0]
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 90 {
		s = s[:90] + "..."
	}
	return s
}

// --- the three calls the probe makes ---

func createDoc(ctx context.Context, s *gapi.Session, folder, name string) (string, error) {
	var created struct {
		ID string `json:"id"`
	}
	body := map[string]any{
		"name":     name,
		"mimeType": "application/vnd.google-apps.document",
		"parents":  []string{folder},
	}
	url := "https://www.googleapis.com/drive/v3/files?fields=id&supportsAllDrives=true"
	if err := s.PostJSON(ctx, url, body, &created); err != nil {
		return "", err
	}
	if created.ID == "" {
		return "", fmt.Errorf("Drive accepted the create and its answer carried no id")
	}
	return created.ID, nil
}

// writeBody lays the content down in two batches. The text first, then a table
// at the end of it, because a table has to exist before updateTableCellStyle
// has anything to act on. The first run of this probe had no table at all,
// which is why three request kinds went unmeasured and had to be assumed.
func writeBody(ctx context.Context, s *gapi.Session, id string) error {
	if err := batch(ctx, s, id, req("insertText", map[string]any{
		"location": map[string]any{"index": 1},
		"text":     fidelityBody,
	})); err != nil {
		return err
	}
	// The table goes at the end of the body. Docs numbers a table's own
	// content, so everything after this index moves; nothing below addresses
	// text after it, which is why the table is written last.
	return batch(ctx, s, id, req("insertTable", map[string]any{
		"location": map[string]any{"index": len([]rune(fidelityBody))},
		"rows":     2,
		"columns":  2,
	}))
}

// tableStart finds the table's own start index in a read-back document. A table
// cell style request names the table by that index, and the index is whatever
// the insert above happened to produce, so it is read rather than computed.
func tableStart(d map[string]any) (int, bool) {
	for _, e := range content(d) {
		m, ok := e.(map[string]any)
		if !ok || m["table"] == nil {
			continue
		}
		if f, ok := m["startIndex"].(float64); ok {
			return int(f), true
		}
	}
	return 0, false
}

func batch(ctx context.Context, s *gapi.Session, id string, requests ...map[string]any) error {
	body := map[string]any{"requests": requests}
	url := "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
	return s.PostJSON(ctx, url, body, nil)
}

func readRaw(ctx context.Context, s *gapi.Session, id string) (map[string]any, error) {
	var raw json.RawMessage
	url := "https://docs.googleapis.com/v1/documents/" + id + "?includeTabsContent=true"
	if err := s.GetJSON(ctx, url, &raw); err != nil {
		return nil, err
	}
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, err
	}
	// With includeTabsContent the body lives under the first tab. The probe
	// creates a one-tab document, so this is the whole of it.
	if tabs, ok := d["tabs"].([]any); ok && len(tabs) > 0 {
		if dt, ok := dig(tabs[0], "documentTab").(map[string]any); ok {
			for k, v := range dt {
				d[k] = v
			}
		}
	}
	return d, nil
}
