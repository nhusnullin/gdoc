package guard

import (
	"net/http"
	"net/url"
	"testing"
)

func mustURL(t *testing.T, s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestJudge(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest) // handed in on the command line
	p.AllowCreateIn("FOLDER1")
	p.Learn("MADE1") // came back from a create gdoc carried

	suggestBody := []byte(`{"requests":[],"writeControl":{"writeMode":"SUGGEST"}}`)
	editBody := []byte(`{"requests":[]}`)

	cases := []struct {
		name, method, url string
		body              []byte
		ok                bool
	}{
		{"read known doc", "GET", "https://docs.googleapis.com/v1/documents/DOC1", nil, true},
		{"read unknown doc", "GET", "https://docs.googleapis.com/v1/documents/EVIL", nil, false},
		{"suggest on handed-in", "POST", "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate", suggestBody, true},
		{"direct edit on handed-in refused", "POST", "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate", editBody, false},
		{"direct edit on created ok", "POST", "https://docs.googleapis.com/v1/documents/MADE1:batchUpdate", editBody, true},
		{"files.list refused", "GET", "https://www.googleapis.com/drive/v3/files?q=x", nil, false},
		{"create allowed with folder", "POST", "https://www.googleapis.com/drive/v3/files", nil, true},
		{"export known", "GET", "https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=x", nil, true},
		{"comments on handed-in", "POST", "https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=x", nil, true},
		{"trash created", "PATCH", "https://www.googleapis.com/drive/v3/files/MADE1", []byte(`{"trashed":true}`), true},
		{"patch handed-in refused", "PATCH", "https://www.googleapis.com/drive/v3/files/DOC1", []byte(`{"trashed":true}`), false},
		{"token endpoint", "POST", "https://oauth2.googleapis.com/token", nil, true},
		{"wrong host", "GET", "https://evil.example.com/v1/documents/DOC1", nil, false},
		{"http not https", "GET", "http://docs.googleapis.com/v1/documents/DOC1", nil, false},
		{"unknown path shape", "GET", "https://www.googleapis.com/drive/v3/about", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := p.Judge(c.method, mustURL(t, c.url), c.body)
			if (err == nil) != c.ok {
				t.Errorf("got err=%v, want ok=%v", err, c.ok)
			}
		})
	}
}

func TestGrantInPlaceUpgrades(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	edit := []byte(`{"requests":[]}`)
	u := mustURL(t, "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if p.Judge("POST", u, edit) == nil {
		t.Fatal("edit must be refused before the grant")
	}
	p.GrantInPlace("DOC1")
	if err := p.Judge("POST", u, edit); err != nil {
		t.Fatalf("edit must be allowed after the grant: %v", err)
	}
}

func TestEmptyPolicyRefusesEverything(t *testing.T) {
	p := NewPolicy()
	u := mustURL(t, "https://docs.googleapis.com/v1/documents/ANY")
	if p.Judge("GET", u, nil) == nil {
		t.Fatal("an empty set must refuse everything")
	}
}

func TestGrantInPlaceNeverAdmitsAnUnknownID(t *testing.T) {
	// GrantInPlace upgrades a level; it is not a third door into the set.
	p := NewPolicy()
	p.GrantInPlace("EVIL")
	u := mustURL(t, "https://docs.googleapis.com/v1/documents/EVIL")
	if p.Judge("GET", u, nil) == nil {
		t.Fatal("GrantInPlace must not admit an id that was never given")
	}
}

func TestUnreadableBatchUpdateBodyIsNotASuggestion(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	u := mustURL(t, "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if p.Judge("POST", u, []byte(`{not json`)) == nil {
		t.Fatal("a body the guard cannot read must not pass as SUGGEST")
	}
	if p.Judge("POST", u, nil) == nil {
		t.Fatal("an absent body must not pass as SUGGEST")
	}
}

// TestAKnownIDMayNotWalkToAnEndpointTheGuardRefuses is the traversal case. Both
// paths below are judged as a read of DOC1, which is in the set, and both arrive
// at the server as drive.about.get, which the table above refuses when it is
// asked plainly. The URL the guard reads and the URL the transport sends have to
// be the same one.
func TestAKnownIDMayNotWalkToAnEndpointTheGuardRefuses(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)

	cases := []struct{ name, raw string }{
		{"dot segments", "https://www.googleapis.com/drive/v3/files/DOC1/../../../about"},
		{"encoded separators", "https://www.googleapis.com/drive/v3/files/DOC1%2F..%2F..%2Fabout"},
		{"dot segments on the docs host", "https://docs.googleapis.com/v1/documents/DOC1/../EVIL"},
		{"a single dot", "https://www.googleapis.com/drive/v3/files/DOC1/./export"},
	}
	for _, c := range cases {
		req, err := http.NewRequest("GET", c.raw, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if err := p.Judge("GET", req.URL, nil); err == nil {
			t.Errorf("%s: %s was carried; wire path %q", c.name, c.raw, req.URL.RequestURI())
		}
	}
}

// TestAPlainPathIsStillCarried is the other direction: the traversal check must
// not refuse the ordinary URLs gdoc actually builds.
func TestAPlainPathIsStillCarried(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	for _, raw := range []string{
		"https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=text/markdown",
		"https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=comments(id)",
		"https://docs.googleapis.com/v1/documents/DOC1",
	} {
		req, err := http.NewRequest("GET", raw, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Judge("GET", req.URL, nil); err != nil {
			t.Errorf("%s must be carried: %v", raw, err)
		}
	}
}

func TestCreateRefusedWithoutANamedFolder(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	if p.Judge("POST", mustURL(t, "https://www.googleapis.com/drive/v3/files"), nil) == nil {
		t.Fatal("a create must be refused when no folder was named")
	}
}

func TestUploadPathFollowsTheSameRules(t *testing.T) {
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	up := mustURL(t, "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart")
	if err := p.Judge("POST", up, nil); err != nil {
		t.Fatalf("upload create must be allowed once a folder is named: %v", err)
	}
	if p.Judge("GET", mustURL(t, "https://www.googleapis.com/upload/drive/v3/files"), nil) == nil {
		t.Fatal("a listing on the upload path must be refused too")
	}
}

func TestTokenHostCarriesNothingElse(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	if p.Judge("GET", mustURL(t, "https://oauth2.googleapis.com/token"), nil) == nil {
		t.Fatal("only POST reaches the token endpoint")
	}
	if p.Judge("POST", mustURL(t, "https://oauth2.googleapis.com/revoke"), nil) == nil {
		t.Fatal("only /token exists on the token host")
	}
}

func TestDeleteOnAFileItselfIsRefused(t *testing.T) {
	// A created file may be trashed with PATCH. A hard delete is not part of
	// any level: nothing here destroys data.
	p := NewPolicy()
	p.Learn("MADE1")
	if p.Judge("DELETE", mustURL(t, "https://www.googleapis.com/drive/v3/files/MADE1"), nil) == nil {
		t.Fatal("a hard delete must be refused even on a created file")
	}
}
