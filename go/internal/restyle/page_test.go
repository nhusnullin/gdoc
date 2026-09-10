package restyle

// This file holds one subject: the page. PageRequest is the first thing a
// restyle sends and the only request kind that names no range, so it is also
// the smallest place to state the rule every builder in this milestone
// follows: the mask names exactly what the request sets, and nothing else.
//
// Every expected number here is written out as a literal. A test reading
// cfg.Page.WidthPt would be a mirror, following the constant wherever somebody
// moved it and passing over a house style that had quietly changed. The house
// values are printed beside the want in each failure instead.

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/guard"
	"gdoc/internal/house"
)

// embeddedHouse is the style the binary ships with, which is what a restyle
// with no --house applies.
func embeddedHouse(t *testing.T) *house.Config {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("the embedded house style must parse: %v", err)
	}
	return cfg
}

// style is the documentStyle object of the one request, decoded.
func style(t *testing.T, req map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("the request must marshal: %v", err)
	}
	var got struct {
		Update struct {
			DocumentStyle map[string]any `json:"documentStyle"`
			Fields        string         `json:"fields"`
		} `json:"updateDocumentStyle"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("the request must decode as an updateDocumentStyle: %v", err)
	}
	if got.Update.DocumentStyle == nil {
		t.Fatalf("the request names no documentStyle: %s", raw)
	}
	return got.Update.DocumentStyle
}

// dimension reads one {magnitude, unit} out of the documentStyle.
func dimension(t *testing.T, in map[string]any, key string) (float64, string) {
	t.Helper()
	v, ok := in[key].(map[string]any)
	if !ok {
		t.Fatalf("the documentStyle carries no %s", key)
	}
	mag, ok := v["magnitude"].(float64)
	if !ok {
		t.Fatalf("%s carries no magnitude: %v", key, v["magnitude"])
	}
	unit, _ := v["unit"].(string)
	return mag, unit
}

// The A4 page the house style states, in points, written out.
func TestThePageSizeIsTheHouseGeometry(t *testing.T) {
	cfg := embeddedHouse(t)
	ds := style(t, PageRequest(cfg))
	size, ok := ds["pageSize"].(map[string]any)
	if !ok {
		t.Fatalf("the documentStyle carries no pageSize: %v", ds)
	}
	w, unit := dimension(t, size, "width")
	if w != 595.2755905511811 {
		t.Errorf("page width is %v, want A4's 595.2755905511811 (house says %v)", w, cfg.Page.WidthPt)
	}
	if unit != "PT" {
		t.Errorf("page width is in %q, want PT", unit)
	}
	h, unit := dimension(t, size, "height")
	if h != 841.8897637795275 {
		t.Errorf("page height is %v, want A4's 841.8897637795275 (house says %v)", h, cfg.Page.HeightPt)
	}
	if unit != "PT" {
		t.Errorf("page height is in %q, want PT", unit)
	}
}

// The four margins, each a literal.
func TestTheMarginsAreTheHouseGeometry(t *testing.T) {
	cfg := embeddedHouse(t)
	ds := style(t, PageRequest(cfg))
	for _, c := range []struct {
		key   string
		want  float64
		house float64
	}{
		{"marginTop", 62.35, cfg.Page.MarginTopPt},
		{"marginBottom", 51, cfg.Page.MarginBottomPt},
		{"marginLeft", 51.05, cfg.Page.MarginLeftPt},
		{"marginRight", 51.05, cfg.Page.MarginRightPt},
	} {
		got, unit := dimension(t, ds, c.key)
		if got != c.want {
			t.Errorf("%s is %v, want %v (house says %v)", c.key, got, c.want, c.house)
		}
		if unit != "PT" {
			t.Errorf("%s is in %q, want PT", c.key, unit)
		}
	}
}

// The mask names exactly what the request sets, both directions. A path in the
// mask that the request leaves unset is the reset the reference documents, and
// a field set outside the mask is a value the server ignores.
func TestThePageMaskNamesExactlyWhatItSets(t *testing.T) {
	cfg := embeddedHouse(t)
	req := PageRequest(cfg)
	ds := style(t, req)
	raw, _ := json.Marshal(req)
	var got struct {
		Update struct {
			Fields string `json:"fields"`
		} `json:"updateDocumentStyle"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("the request must decode: %v", err)
	}
	if got.Update.Fields != "pageSize,marginTop,marginBottom,marginLeft,marginRight" {
		t.Fatalf("the mask is %q, want %q", got.Update.Fields,
			"pageSize,marginTop,marginBottom,marginLeft,marginRight")
	}
	var set []string
	for k := range ds {
		set = append(set, k)
	}
	named := strings.Split(got.Update.Fields, ",")
	sort.Strings(set)
	sort.Strings(named)
	if strings.Join(set, ",") != strings.Join(named, ",") {
		t.Errorf("the request sets %v and the mask names %v, and they must be the same set", set, named)
	}
}

// The two refusals the guard makes, asked of the builder rather than left to
// be refused. A star resets every property the request does not set, and either
// header toggle hides the first-page header that carries the logo.
func TestThePageMaskCarriesNoStarAndNoHeaderToggle(t *testing.T) {
	cfg := embeddedHouse(t)
	raw, _ := json.Marshal(PageRequest(cfg))
	for _, bad := range []string{"*", "useFirstPageHeaderFooter", "useEvenPageHeaderFooter"} {
		if strings.Contains(string(raw), bad) {
			t.Errorf("the page request carries %q, and the guard refuses it: %s", bad, raw)
		}
	}
}

// The builder is judged by the guard it writes for. A refusal here means the
// two drifted, which is the failure a mask written by hand is most likely to
// have.
func TestTheGuardCarriesThePageRequest(t *testing.T) {
	cfg := embeddedHouse(t)
	body, err := json.Marshal(map[string]any{"requests": []any{PageRequest(cfg)}})
	if err != nil {
		t.Fatalf("the batch must marshal: %v", err)
	}
	p := guard.NewPolicy()
	p.AllowFile("DOC1", guard.LevelSuggest)
	p.GrantInPlace("DOC1")
	u, err := url.Parse("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if err != nil {
		t.Fatalf("the URL must parse: %v", err)
	}
	if err := p.Judge("POST", u, body); err != nil {
		t.Fatalf("the guard refused the page request this milestone builds: %v", err)
	}
}
