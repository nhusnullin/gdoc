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
		// comments.list requires `fields`: Drive answers a request without one
		// with an error, so a case that omits it passes the guard and still
		// fails the real API, and it does not pin the shape M2 sends.
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=comments(id,content,author,quotedFileContent,replies),nextPageToken&pageSize=100&includeDeleted=false", nil},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1/comments/C1/replies?fields=replies(id,content,author,createdTime)&pageSize=100", nil},
		{"POST", "https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=id", nil},
		{"PATCH", "https://www.googleapis.com/drive/v3/files/MADE1?fields=id", []byte(`{"trashed":true}`)},
		{"GET", "https://docs.googleapis.com/v1/documents/DOC1?suggestionsViewMode=SUGGESTIONS_INLINE", nil},
		{"POST", "https://www.googleapis.com/drive/v3/files?fields=id", nil},
		{"POST", "https://oauth2.googleapis.com/token", nil},
		// The two M2 reads. Comment threads with real character ranges are a
		// documents.get with commentsViewMode, which needs includeTabsContent,
		// and the --since cursor is comments.list with startModifiedTime. The
		// cursor call carries `fields` for the same reason the listing above
		// does: it is the same method, and Drive requires it.
		{"GET", "https://docs.googleapis.com/v1/documents/DOC1?commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED&includeTabsContent=true", nil},
		{"GET", "https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=comments(id,content,author,quotedFileContent,replies),nextPageToken&startModifiedTime=2026-08-29T09:00:00Z&pageToken=t", nil},
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

// SPEC.md's Never list: "Never export a PDF. Nail downloads it from the
// browser." mimeType was on the allowlist with no opinion about its value, so
// the one export format the spec names was the one nothing stopped.
func TestAPDFExportIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	const base = "https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType="
	for _, raw := range []string{
		base + "application/pdf",
		base + "APPLICATION/PDF",             // the scheme is case-insensitive, so the rule is too
		base + "application/pdf;+charset=x",  // a parameter after the type is still the type
		base + "+application/pdf+&alt=media", // and so is one with spaces around it
	} {
		if err := p.Judge("GET", mustURL(t, raw), nil); err == nil {
			t.Errorf("%s must be refused: gdoc never exports a PDF", raw)
		}
	}
	docx := base + "application/vnd.openxmlformats-officedocument.wordprocessingml.document&alt=media"
	if err := p.Judge("GET", mustURL(t, docx), nil); err != nil {
		t.Errorf("the docx export is the honest witness and must still be carried: %v", err)
	}
}

// SPEC.md: "Always read with includeTabsContent=true. Reading without it
// silently sees one tab." The guard decides the value, not the presence: see
// checkParamValue for where that line is drawn and why.
func TestIncludeTabsContentMustSayTrue(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	const base = "https://docs.googleapis.com/v1/documents/DOC1?includeTabsContent="
	for _, raw := range []string{base + "false", base, base + "1", base + "TRUE"} {
		if err := p.Judge("GET", mustURL(t, raw), nil); err == nil {
			t.Errorf("%s must be refused: a read that asks for one tab reads one tab and says nothing about the rest", raw)
		}
	}
	if err := p.Judge("GET", mustURL(t, base+"true"), nil); err != nil {
		t.Errorf("the read gdoc makes must be carried: %v", err)
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

// alt=media on a bare files.get is not a metadata read: it hands back the
// file's bytes. gdoc reads bytes through /export, where checkExportMime decides
// the format, so the parameter has no business on the metadata call.
func TestAltMediaOnAPlainFileGetIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.Learn("MADE1")
	for _, raw := range []string{
		"https://www.googleapis.com/drive/v3/files/DOC1?alt=media",
		"https://www.googleapis.com/drive/v3/files/DOC1?alt=json",
		"https://www.googleapis.com/drive/v3/files/MADE1?alt=media",
		"https://www.googleapis.com/drive/v3/files/DOC1?fields=id&alt=media",
	} {
		if p.Judge("GET", mustURL(t, raw), nil) == nil {
			t.Errorf("%s must be refused: a metadata read carries no alt", raw)
		}
	}
	// Where the bytes are asked for on purpose, alt still carries.
	ok := "https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=text/markdown&alt=media"
	if err := p.Judge("GET", mustURL(t, ok), nil); err != nil {
		t.Errorf("an export asks for bytes on purpose: %v", err)
	}
}

// Each read shape carries only the parameters its own method defines. One
// shared list let a paging parameter onto a metadata read and an export format
// onto a comment listing, and neither is a call Drive has.
func TestEachDriveReadCarriesOnlyItsOwnParameters(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	for _, raw := range []string{
		// files.get takes no paging, no export format, no comment filter.
		"https://www.googleapis.com/drive/v3/files/DOC1?pageSize=100",
		"https://www.googleapis.com/drive/v3/files/DOC1?pageToken=t",
		"https://www.googleapis.com/drive/v3/files/DOC1?mimeType=text/markdown",
		"https://www.googleapis.com/drive/v3/files/DOC1?includeDeleted=false",
		"https://www.googleapis.com/drive/v3/files/DOC1?startModifiedTime=2026-08-29T09:00:00Z",
		// files.export takes no paging and no comment filter.
		"https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=text/plain&pageSize=100",
		"https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=text/plain&includeDeleted=false",
		// The comment reads take no export format and no bytes.
		"https://www.googleapis.com/drive/v3/files/DOC1/comments?mimeType=text/markdown",
		"https://www.googleapis.com/drive/v3/files/DOC1/comments?alt=media",
		// startModifiedTime is the comments.list cursor and nothing else.
		"https://www.googleapis.com/drive/v3/files/DOC1/comments/C1/replies?startModifiedTime=2026-08-29T09:00:00Z",
		"https://www.googleapis.com/drive/v3/files/DOC1/comments/C1?startModifiedTime=2026-08-29T09:00:00Z",
		// A single comment or reply is not a page of them.
		"https://www.googleapis.com/drive/v3/files/DOC1/comments/C1?pageToken=t",
		"https://www.googleapis.com/drive/v3/files/DOC1/comments/C1/replies/R1?pageSize=100",
	} {
		if p.Judge("GET", mustURL(t, raw), nil) == nil {
			t.Errorf("%s must be refused: that parameter is not on this call", raw)
		}
	}
}

// TestResumableUploadIsRefused: a resumable create is a two-leg call. The start
// leg posts the metadata, and the content goes up on a PUT to a location URL
// carrying `upload_id`, both of which the guard refuses. So the shape could
// never complete, and half-permitting it reads as a working route to whoever
// tries it next. The guard says no at the leg it can see.
func TestResumableUploadIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	for _, raw := range []string{
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=resumable",
		"https://www.googleapis.com/upload/drive/v3/files?upload_protocol=resumable",
	} {
		if err := p.Judge("POST", mustURL(t, raw), nil); err == nil {
			t.Errorf("%s must be refused: the guard carries no leg of a resumable upload", raw)
		}
	}
}
