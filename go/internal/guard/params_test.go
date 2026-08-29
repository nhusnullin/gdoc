package guard

// This file holds one subject: the query surface. What a request may carry
// besides its path, and what the guard does with a spelling it has never been
// told about. The header half of the same rule is in wire_match_test.go.

import (
	"testing"
)

// TestAnUnknownQueryParameterIsRefused is the property that makes the next
// spelling harmless. None of these is blocked by name anywhere in the guard:
// they are refused because they are not on the allowlist, which is what a
// blocklist could not do until somebody had read about each one.
func TestAnUnknownQueryParameterIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.Learn("MADE1")
	p.AllowCreateIn("FOLDER1")

	cases := []struct{ method, url string }{
		// The `$` spellings of the system parameters, which the plain names
		// would never have caught.
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?$fields=permissions"},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?%24fields=id"},
		// A second credential on a request gdoc authenticates itself.
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?access_token=x"},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?oauth_token=x"},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?key=x"},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?callback=f"},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?prettyPrint=true"},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?quotaUser=someone"},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?$.xgafv=2"},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?madeUpTomorrow=1"},
		// A read is not a listing, whatever the query says.
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?q=name"},
		// The same rule on every other call, not only on a read.
		{"POST", "https://www.googleapis.com/drive/v3/files/DOC1/comments?access_token=x"},
		{"PATCH", "https://www.googleapis.com/drive/v3/files/MADE1?key=x"},
		{"POST", "https://www.googleapis.com/drive/v3/files?callback=f"},
		{"GET", "https://docs.googleapis.com/v1/documents/DOC1?access_token=x"},
		{"POST", "https://oauth2.googleapis.com/token?key=x"},
		// One value each. Which copy the server takes is not decided here.
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?fields=id&fields=name"},
	}
	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			body := []byte(`{"parents":["FOLDER1"]}`)
			if err := p.Judge(c.method, mustURL(t, c.url), body); err == nil {
				t.Errorf("%s %s must be refused: the guard has decided nothing about that parameter", c.method, c.url)
			}
		})
	}
}

// TestTheCallsGdocMakesStillCarry is the other direction. An allowlist that
// refuses the real calls is not a guard, it is a broken client.
func TestTheCallsGdocMakesStillCarry(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.Learn("MADE1")
	p.AllowCreateIn("FOLDER1")

	cases := []struct {
		method, url string
		body        []byte
	}{
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1?fields=id,name,mimeType", nil},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=text/markdown&alt=media", nil},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1/comments?pageSize=100&includeDeleted=false", nil},
		{"POST", "https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=id", nil},
		{"PATCH", "https://www.googleapis.com/drive/v3/files/MADE1?fields=id", []byte(`{"trashed":true}`)},
		{"GET", "https://docs.googleapis.com/v1/documents/DOC1?suggestionsViewMode=SUGGESTIONS_INLINE", nil},
		{"POST", "https://www.googleapis.com/drive/v3/files?fields=id", nil},
		{"POST", "https://oauth2.googleapis.com/token", nil},
	}
	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			if err := p.Judge(c.method, mustURL(t, c.url), c.body); err != nil {
				t.Errorf("%s %s must be carried: %v", c.method, c.url, err)
			}
		})
	}
}

// TestUploadProtocolIsTheSameChoiceAsUploadType pins the sibling spelling.
// `upload_protocol=raw` is `uploadType=media` under Drive's newer name: the
// whole body becomes the file's content, so `{"parents":[...]}` is bytes to
// Drive and metadata to the parent check, and the guard would learn the id of a
// file it never placed.
func TestUploadProtocolIsTheSameChoiceAsUploadType(t *testing.T) {
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	refused := []string{
		"https://www.googleapis.com/upload/drive/v3/files?upload_protocol=raw",
		"https://www.googleapis.com/upload/drive/v3/files?upload_protocol=RAW",
		"https://www.googleapis.com/upload/drive/v3/files?upload_protocol=",
		"https://www.googleapis.com/upload/drive/v3/files?upload_protocol=multipart&upload_protocol=raw",
		// Both names at once: the guard would read one and Drive may obey the other.
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&upload_protocol=raw",
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&upload_protocol=multipart",
	}
	for _, raw := range refused {
		if err := p.Judge("POST", mustURL(t, raw), nil); err == nil {
			t.Errorf("%s must be refused: the guard cannot read parents out of that body", raw)
		}
	}
	for _, raw := range []string{
		"https://www.googleapis.com/upload/drive/v3/files?upload_protocol=multipart",
		"https://www.googleapis.com/upload/drive/v3/files?upload_protocol=resumable",
	} {
		if err := p.Judge("POST", mustURL(t, raw), nil); err != nil {
			t.Errorf("%s must be carried: %v", raw, err)
		}
	}
}

// TestAnEmptyFieldMaskIsRefused: Google's system-parameter documentation reads
// a mask with nothing in it as every field. So `fields=` asks for exactly what
// `fields=*` asks for, including the permission surface, and it names no
// blocked token while doing it.
func TestAnEmptyFieldMaskIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	for _, raw := range []string{
		"https://www.googleapis.com/drive/v3/files/DOC1?fields=",
		"https://www.googleapis.com/drive/v3/files/DOC1?fields=%20",
		"https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=",
		"https://docs.googleapis.com/v1/documents/DOC1?fields=",
	} {
		if err := p.Judge("GET", mustURL(t, raw), nil); err == nil {
			t.Errorf("%s must be refused: an empty field mask means every field", raw)
		}
	}
}

// A query the guard cannot parse is a query it cannot judge, and Google may
// still read the half that url.Values drops.
func TestAnUnreadableQueryIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	u := mustURL(t, "https://www.googleapis.com/drive/v3/files/DOC1")
	u.RawQuery = "fields=id&%zz=1"
	if err := p.Judge("GET", u, nil); err == nil {
		t.Error("a query that cannot be parsed must be refused, not judged on the part that parsed")
	}
}
