package guard

// This file holds one subject: the prelude marker, and the proof that the
// prelude itself needed no new permission.
//
// M7c proposes the house template as a suggestion and marks what it proposed
// with a named range, so a second run replaces its own prelude rather than
// adding a second cover. createNamedRange cannot be a suggestion: Docs answered
// "Request does not support application as suggestion" on 2026-09-10. So the
// marker is the one thing written directly, and it gets a door of its own in
// AllowReject's shape rather than a place on the styling allowlist.
//
// Every test here is written as an attack first, the way inplace_test.go is.

import (
	"strconv"
	"strings"
	"testing"
)

const (
	markerName  = "gdoc:house-prelude"
	markerStart = 1
	markerEnd   = 23
)

// marked returns a restyle run's phase 2 policy: DOC1 handed in, granted for
// direct edit, and granted the one marker it is about to create.
func marked(t *testing.T) *Policy {
	t.Helper()
	p := granted(t)
	p.AllowMarker(markerName, markerStart, markerEnd)
	return p
}

// markerRequest is one createNamedRange body, written out because what this
// file judges is exactly those bytes.
func markerRequest(name string, start, end int) string {
	return `{"name":"` + name + `","range":{"startIndex":` + strconv.Itoa(start) + `,"endIndex":` + strconv.Itoa(end) + `}}`
}

// The attack first. A createNamedRange is a direct write, so it is refused on a
// handed-in document by the level rule, and on a granted document by the
// styling allowlist. Only the grant opens it.
func TestTheMarkerIsRefusedWithoutItsOwnGrant(t *testing.T) {
	body := []byte(`{"requests":[{"createNamedRange":` + markerRequest(markerName, markerStart, markerEnd) + `}]}`)

	t.Run("handed in, no grant of any kind", func(t *testing.T) {
		p := NewPolicy()
		p.AllowFile("DOC1", LevelSuggest)
		if p.Judge("POST", mustURL(t, inPlaceURL), body) == nil {
			t.Fatal("a direct createNamedRange on a handed-in document must be refused")
		}
	})

	t.Run("handed in, marker granted but not the document", func(t *testing.T) {
		p := NewPolicy()
		p.AllowFile("DOC1", LevelSuggest)
		p.AllowMarker(markerName, markerStart, markerEnd)
		if p.Judge("POST", mustURL(t, inPlaceURL), body) == nil {
			t.Fatal("the marker grant is not a second door into direct edit")
		}
	})

	t.Run("granted for styling, no marker grant", func(t *testing.T) {
		p := granted(t)
		if p.Judge("POST", mustURL(t, inPlaceURL), body) == nil {
			t.Fatal("the styling grant does not carry a createNamedRange on its own")
		}
	})
}

// The grant carries exactly one range, spelled exactly, and nothing beside it.
func TestAGrantedMarkerCarriesAndNothingElseDoes(t *testing.T) {
	p := marked(t)
	ok := []byte(`{"requests":[{"createNamedRange":` + markerRequest(markerName, markerStart, markerEnd) + `}]}`)
	if err := p.Judge("POST", mustURL(t, inPlaceURL), ok); err != nil {
		t.Fatalf("the marker this run granted must carry: %v", err)
	}

	cases := []struct{ name, inner, want string }{
		{
			"a second one naming a different range",
			markerRequest(markerName, 1, 400),
			"range",
		},
		{
			"the same range under another name",
			markerRequest("somebody-elses-range", markerStart, markerEnd),
			"name",
		},
		{
			"a field beside the two",
			`{"name":"` + markerName + `","range":{"startIndex":1,"endIndex":23},"tabId":"t.0"}`,
			"two fields",
		},
		{
			"a field beside the two inside the range",
			`{"name":"` + markerName + `","range":{"startIndex":1,"endIndex":23,"segmentId":"h.1"}}`,
			"startIndex",
		},
		{
			"no range at all",
			`{"name":"` + markerName + `"}`,
			"two fields",
		},
		{
			"the name under another spelling",
			`{"Name":"` + markerName + `","range":{"startIndex":1,"endIndex":23}}`,
			"name",
		},
		{
			"an index that is not a number",
			`{"name":"` + markerName + `","range":{"startIndex":"1","endIndex":23}}`,
			"startIndex",
		},
		{
			"a range that is not an object",
			`{"name":"` + markerName + `","range":[1,23]}`,
			"range",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := marked(t)
			body := []byte(`{"requests":[{"createNamedRange":` + c.inner + `}]}`)
			err := p.Judge("POST", mustURL(t, inPlaceURL), body)
			if err == nil {
				t.Fatalf("%s must be refused: the grant is one range, and nobody here has read what a second field does", c.name)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal must name %q; got %v", c.want, err)
			}
		})
	}
}

// The grant is one range, so a second call replaces the first and the first
// stops carrying. A caller naming two has made a mistake the guard must not
// turn into two markers on somebody's document.
func TestASecondAllowMarkerReplacesTheFirst(t *testing.T) {
	p := granted(t)
	p.AllowMarker(markerName, markerStart, markerEnd)
	p.AllowMarker(markerName, 1, 99)

	first := []byte(`{"requests":[{"createNamedRange":` + markerRequest(markerName, markerStart, markerEnd) + `}]}`)
	if p.Judge("POST", mustURL(t, inPlaceURL), first) == nil {
		t.Fatal("the grant a second call replaced must stop carrying")
	}
	second := []byte(`{"requests":[{"createNamedRange":` + markerRequest(markerName, 1, 99) + `}]}`)
	if err := p.Judge("POST", mustURL(t, inPlaceURL), second); err != nil {
		t.Fatalf("the grant that replaced it must carry: %v", err)
	}
}

// A grant that names nothing usable grants nothing, and it says so on the
// envelope rather than quietly. AllowFile holds the same rule for the same
// reason: not knowing must never resolve to a wider reach.
func TestAnUnusableMarkerGrantOpensNothing(t *testing.T) {
	cases := []struct {
		name             string
		mark             string
		start, end       int
		wantWarningNames string
	}{
		{"no name", "", 1, 23, "name"},
		{"an empty range", markerName, 5, 5, "range"},
		{"an inverted range", markerName, 9, 5, "range"},
		{"a negative start", markerName, -1, 23, "range"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := granted(t)
			p.AllowMarker(c.mark, c.start, c.end)
			body := []byte(`{"requests":[{"createNamedRange":` + markerRequest(c.mark, c.start, c.end) + `}]}`)
			if p.Judge("POST", mustURL(t, inPlaceURL), body) == nil {
				t.Fatal("a grant the guard could not read must open nothing")
			}
			warnings := p.Warnings()
			if len(warnings) != 1 || !strings.Contains(warnings[0], c.wantWarningNames) {
				t.Fatalf("the refused grant must be on the envelope naming %q; got %v", c.wantWarningNames, warnings)
			}
		})
	}
}

// This is the more important test in this milestone, and it asserts an absence.
//
// The prelude phase sends the cover, the tables and the legend as a SUGGEST
// batchUpdate on a handed-in document with no grant of any kind, which is
// exactly what propose sends every day. So M7c added no permission for it, and
// this test is what says so: if it ever needs a grant to pass, the two phases
// have been collapsed into one and the milestone's whole shape has gone.
func TestThePreludeNeedsNoGrantAtAll(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)

	kinds := []string{
		`{"insertText":{"location":{"index":1},"text":"ALTERY GROUP\n"}}`,
		`{"insertTable":{"location":{"index":1},"rows":2,"columns":2}}`,
		`{"insertPageBreak":{"location":{"index":1}}}`,
		`{"createParagraphBullets":{"range":{"startIndex":1,"endIndex":9},"bulletPreset":"BULLET_DISC_CIRCLE_SQUARE"}}`,
		`{"deleteContentRange":{"range":{"startIndex":1,"endIndex":9}}}`,
		`{"updateParagraphStyle":{"paragraphStyle":{"alignment":"CENTER"},"fields":"alignment","range":{"startIndex":1,"endIndex":9}}}`,
		`{"updateTextStyle":{"textStyle":{"bold":true},"fields":"bold","range":{"startIndex":1,"endIndex":9}}}`,
	}
	body := []byte(`{"requests":[` + strings.Join(kinds, ",") + `],"writeControl":{"writeMode":"SUGGEST"}}`)
	if err := p.Judge("POST", mustURL(t, inPlaceURL), body); err != nil {
		t.Fatalf("the whole prelude is a SUGGEST batch on a handed-in document, and that has always carried: %v", err)
	}

	t.Run("and the same batch direct is still refused", func(t *testing.T) {
		p := NewPolicy()
		p.AllowFile("DOC1", LevelSuggest)
		direct := []byte(`{"requests":[` + strings.Join(kinds, ",") + `]}`)
		if p.Judge("POST", mustURL(t, inPlaceURL), direct) == nil {
			t.Fatal("without SUGGEST the same requests are a direct edit of somebody's document")
		}
	})
}

// The marker door is scoped to the level that opens it, so it does not quietly
// widen the styling allowlist for anything else.
func TestTheMarkerGrantWidensNothingElse(t *testing.T) {
	p := marked(t)
	for _, kind := range []string{"insertText", "deleteContentRange", "deleteNamedRange", "createHeader"} {
		t.Run(kind, func(t *testing.T) {
			if p.Judge("POST", mustURL(t, inPlaceURL), batch(kind, `{"text":"x"}`)) == nil {
				t.Fatalf("%s must stay refused on a document granted only a marker", kind)
			}
		})
	}
}
