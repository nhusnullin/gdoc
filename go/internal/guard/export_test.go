package guard

import (
	"net/url"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/docx"
)

// `gdoc export` opens no door of its own. It reads the document twice, through
// the Docs read `read` makes and through the docx export `comments --witness`
// makes, so the policy it opens is the policy those two open: one file at
// LevelSuggest, and nothing else.
//
// The two URLs are built by the packages that build them, not written out
// here, so a read that grew a parameter is judged in this test the moment it
// grows one rather than the next time somebody remembers to copy it across.
const exportDocID = "1AbCdEfGhIjKlMnOpQrStUvWxYz012345"

func TestExportOpensOnlyThePolicyReadOpens(t *testing.T) {
	p := NewPolicy()
	p.AllowFile(exportDocID, LevelSuggest)

	for _, allowed := range []string{docs.URL(exportDocID), docx.ExportURL(exportDocID)} {
		if err := p.Judge("GET", parse(t, allowed), nil); err != nil {
			t.Errorf("export reads %s and the read policy refused it: %v", allowed, err)
		}
	}

	// The same two addresses under any other method. An export sends nothing
	// that can change a document, and the policy is what holds that rather
	// than the command remembering to.
	for _, method := range []string{"POST", "PATCH", "PUT", "DELETE"} {
		for _, target := range []string{docs.URL(exportDocID), docx.ExportURL(exportDocID)} {
			if err := p.Judge(method, parse(t, target), []byte(`{}`)); err == nil {
				t.Errorf("%s on %s must be refused: an export writes nothing", method, target)
			}
		}
	}

	// Nothing beyond that level is open here. There is no create, no copy, no
	// trashing, no other document and no other host.
	//
	// A comment and a suggestion are not in this list, and that is the point
	// of the level rather than a gap: LevelSuggest is what `read` opens too,
	// and it carries them. What holds export to reads is export itself, over
	// a session that fails the run on any write verb:
	// TestExportSendsNothingThatWrites in cmd/gdoc.
	for _, refused := range []struct{ name, method, target, body string }{
		{"a direct edit", "POST", "https://docs.googleapis.com/v1/documents/" + exportDocID + ":batchUpdate", `{"requests":[]}`},
		{"a create", "POST", "https://www.googleapis.com/drive/v3/files", `{"parents":["ANY"]}`},
		{"a copy", "POST", "https://www.googleapis.com/drive/v3/files/" + exportDocID + "/copy", `{}`},
		{"a trashing", "PATCH", "https://www.googleapis.com/drive/v3/files/" + exportDocID, `{"trashed":true}`},
		{"the releases listing", "GET", "https://api.github.com/repos/nailkhusnullin/gdoc/releases", ""},
		{"another document", "GET", "https://docs.googleapis.com/v1/documents/9ZzYyXxWwVvUuTtSsRrQqPpOoNn01234", ""},
		{"another host", "GET", "https://lh7-us.googleusercontent.com/picture.png", ""},
	} {
		var body []byte
		if refused.body != "" {
			body = []byte(refused.body)
		}
		if err := p.Judge(refused.method, parse(t, refused.target), body); err == nil {
			t.Errorf("%s must be refused on an export's policy: %s %s", refused.name, refused.method, refused.target)
		}
	}
}

// The picture bytes are the reason the export exists, and they are the reason
// this test names googleusercontent.com above. A picture's contentUri in the
// Docs answer is on that host, the guard admits it nowhere, and that is why
// the bytes come from the docx export instead.
func TestAPicturesContentURIHostIsNotOneTheExportMayReach(t *testing.T) {
	p := NewPolicy()
	p.AllowFile(exportDocID, LevelSuggest)

	err := p.Judge("GET", parse(t, "https://lh3.googleusercontent.com/d/"+exportDocID), nil)
	if err == nil {
		t.Fatal("the host a contentUri points at must stay refused")
	}
}

func parse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
