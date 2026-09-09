package guard

// This file holds one subject: files.copy, and the per-run grant that opens it.
//
// A copy is the widest reach a handed-in id has ever produced. It takes a full
// duplicate of somebody's document, comments and pending suggestions included,
// into gdoc's own folder, where the guard learns it at LevelFull. So every test
// here is written as an attack first: the grant names one source, the copy
// lands in the one folder the run named, and a copy that names neither is
// refused before the wire.

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

const copyURL = "https://www.googleapis.com/drive/v3/files/DOC1/copy"

// copyGranted returns a policy holding a copy run's two grants: the source
// handed in, the copy grant naming it, and the folder the copy may land in.
func copyGranted() *Policy {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.AllowCopy("DOC1")
	p.AllowCreateIn("FOLDER1")
	return p
}

func TestAGrantedCopyCarries(t *testing.T) {
	p := copyGranted()
	body := []byte(`{"name":"acceptance copy","parents":["FOLDER1"]}`)
	if err := p.Judge("POST", mustURL(t, copyURL+"?copyComments=true"), body); err != nil {
		t.Fatalf("a copy of the granted source into the granted folder must carry: %v", err)
	}
}

// The copy is refused without the grant, and the policy below is not a shape
// somebody had to invent. It is cmdPropose's: one document handed in at the
// suggest level, one folder the probe may create in. A copy keyed on holding a
// create folder would have let every propose run duplicate the document it was
// reviewing and learn the duplicate at LevelFull, which is direct edit with no
// allowlist and a Drive PATCH. So the grant is what opens it, and nothing else.
func TestACopyIsRefusedWithoutTheGrant(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.AllowCreateIn("FOLDER1")
	body := []byte(`{"parents":["FOLDER1"]}`)
	err := p.Judge("POST", mustURL(t, copyURL+"?copyComments=true"), body)
	if err == nil {
		t.Fatal("a copy with no grant must be refused")
	}
	if !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("want a guard refusal, got %v", err)
	}
}

// The grant is one source. A run that may copy DOC1 may not copy DOC2, even
// when DOC2 was handed in at the same level.
func TestACopyOfAnotherDocumentIsRefused(t *testing.T) {
	p := copyGranted()
	p.AllowFile("DOC2", LevelSuggest)
	body := []byte(`{"parents":["FOLDER1"]}`)
	err := p.Judge("POST", mustURL(t, "https://www.googleapis.com/drive/v3/files/DOC2/copy"), body)
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("the grant names one source, and DOC2 is not it: %v", err)
	}
}

// The grant is not a third door. AllowCopy names a source; it does not admit
// one. An id nobody handed in is refused by the file check, before the grant is
// ever read.
func TestTheCopyGrantAdmitsNoID(t *testing.T) {
	p := NewPolicy()
	p.AllowCopy("DOC9")
	p.AllowCreateIn("FOLDER1")
	body := []byte(`{"parents":["FOLDER1"]}`)
	err := p.Judge("POST", mustURL(t, "https://www.googleapis.com/drive/v3/files/DOC9/copy"), body)
	if err == nil || !strings.Contains(err.Error(), "was not given to this command") {
		t.Fatalf("the grant must not admit an id to the reachable set: %v", err)
	}
}

// A copy is a create, so it needs the one folder a create may target. Without
// AllowCreateIn there is nowhere the guard has agreed the duplicate may land.
func TestACopyNeedsAFolderToLandIn(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.AllowCopy("DOC1")
	body := []byte(`{"parents":["FOLDER1"]}`)
	err := p.Judge("POST", mustURL(t, copyURL), body)
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("a copy with no create folder must be refused: %v", err)
	}
}

// Three parameters, and nothing else. copyComments is the one the measurement
// of 2026-09-09 turned on, supportsAllDrives is what the test folder needs, and
// fields narrows the answer the new id is read out of.
func TestTheCopyCarriesThreeParametersAndNoOthers(t *testing.T) {
	body := []byte(`{"parents":["FOLDER1"]}`)
	carried := []string{
		"copyComments=true",
		"supportsAllDrives=true",
		"fields=id",
		"copyComments=true&supportsAllDrives=true&fields=id",
	}
	for _, q := range carried {
		t.Run("carried/"+q, func(t *testing.T) {
			p := copyGranted()
			if err := p.Judge("POST", mustURL(t, copyURL+"?"+q), body); err != nil {
				t.Fatalf("%s must carry on a copy: %v", q, err)
			}
		})
	}
	// Each of these is a Drive files.copy parameter the guard has not decided
	// about, or a spelling that changes what comes back. ocr runs the source
	// through text recognition, keepRevisionForever pins a revision in the
	// owner's storage quota, includePermissionsForView reaches the permission
	// surface, and alt=media asks for bytes rather than the metadata the new id
	// is read out of.
	refused := []string{
		"ocr=true",
		"ocrLanguage=en",
		"keepRevisionForever=true",
		"ignoreDefaultVisibility=true",
		"includePermissionsForView=published",
		"enforceSingleParent=true",
		"alt=media",
		"uploadType=multipart",
		"fields=*",
		"fields=",
		"fields=permissions",
	}
	for _, q := range refused {
		t.Run("refused/"+q, func(t *testing.T) {
			p := copyGranted()
			err := p.Judge("POST", mustURL(t, copyURL+"?"+q), body)
			if err == nil || !strings.Contains(err.Error(), "guard refused") {
				t.Fatalf("%s must be refused on a copy: %v", q, err)
			}
		})
	}
}

// POST is the method files.copy defines, and it is the only one the guard
// carries on that path. A GET there is not a read gdoc makes, and a PATCH or a
// DELETE on it is a method Drive does not define at all.
func TestOnlyPOSTIsCarriedOnTheCopyPath(t *testing.T) {
	for _, method := range []string{"GET", "PATCH", "DELETE", "PUT"} {
		t.Run(method, func(t *testing.T) {
			p := copyGranted()
			err := p.Judge(method, mustURL(t, copyURL), []byte(`{"parents":["FOLDER1"]}`))
			if err == nil || !strings.Contains(err.Error(), "guard refused") {
				t.Fatalf("%s on the copy path must be refused: %v", method, err)
			}
		})
	}
}

// A path that looks like a copy and is not one stays refused. The segment count
// is part of the name, exactly as it is for every other Drive shape.
func TestOnlyTheExactCopyPathIsACopy(t *testing.T) {
	paths := []string{
		"https://www.googleapis.com/drive/v3/files/DOC1/copy/again",
		"https://www.googleapis.com/drive/v3/files/DOC1/copyright",
		"https://www.googleapis.com/drive/v3/files/DOC1/Copy",
	}
	for _, u := range paths {
		t.Run(u, func(t *testing.T) {
			p := copyGranted()
			err := p.Judge("POST", mustURL(t, u), []byte(`{"parents":["FOLDER1"]}`))
			if err == nil || !strings.Contains(err.Error(), "guard refused") {
				t.Fatalf("%s is not files.copy and must be refused: %v", u, err)
			}
		})
	}
}

// The transport half, and the reason this task has one. isCreate keys the
// parent check and the id learning on filesCollection, and {id}/copy is not the
// files collection. Without the copy in that grammar a copy carries with no
// parent check and no id learned, so the duplicate is unreachable and the run
// cannot even trash it.
func TestACopyIsParentCheckedAndLearned(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"COPY1","name":"acceptance copy"}`}
	p := copyGranted()
	c := NewClient(p, f)
	resp, err := c.Post(copyURL+"?copyComments=true&fields=id",
		"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body) // the body must survive the guard's peek
	if !strings.Contains(string(got), "COPY1") {
		t.Fatalf("the response body was consumed: %q", got)
	}
	if p.files["COPY1"] != LevelFull {
		t.Fatalf("the copy's id was not learned at LevelFull, got %v", p.files["COPY1"])
	}
}

// A copy that omits parents lands in the source's own parent, which is a folder
// gdoc was never given. So the one-parent rule has to be reached rather than
// skipped, and this is the case that says so.
func TestACopyWithNoParentsIsRefused(t *testing.T) {
	cases := []struct{ name, body string }{
		{"no parents", `{"name":"acceptance copy"}`},
		{"another folder", `{"parents":["OTHER"]}`},
		{"two parents", `{"parents":["FOLDER1","OTHER"]}`},
		{"empty body", ``},
		{"not json", `not json at all`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fake{status: 200, body: `{"id":"COPY1"}`}
			p := copyGranted()
			c := NewClient(p, f)
			_, err := c.Post(copyURL, "application/json", bytes.NewReader([]byte(tc.body)))
			if err == nil || !strings.Contains(err.Error(), "guard refused") {
				t.Fatalf("want a guard refusal, got %v", err)
			}
			if len(f.seen) != 0 {
				t.Fatal("it reached the wire")
			}
			if _, known := p.level("COPY1"); known {
				t.Fatal("a refused copy taught the policy an id")
			}
		})
	}
}

// The policy and the transport read one grammar, and this is the spelling that
// used to make them disagree. judgeDrive strips /upload before it splits the
// path, so a copy written there is judged as a copy; filesCopy strips it too,
// so the parent check runs on the same request. A reader that stripped it in
// one place only would carry an unparented duplicate.
func TestACopyOnTheUploadPathIsJudgedTheSameWay(t *testing.T) {
	const uploadCopy = "https://www.googleapis.com/upload/drive/v3/files/DOC1/copy"
	f := &fake{status: 200, body: `{"id":"COPY1"}`}
	p := copyGranted()
	c := NewClient(p, f)
	_, err := c.Post(uploadCopy, "application/json", bytes.NewReader([]byte(`{"parents":["OTHER"]}`)))
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("the upload spelling must reach the same parent check: %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
}

// The grant is per run and one source, in the shape AllowReject and
// AllowCreateIn already have. Nothing writes it down and no flag turns it on
// for every document, so a fresh policy carries no copy at all.
func TestTheCopyGrantDiesWithThePolicy(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.AllowCreateIn("FOLDER1")
	if p.mayCopy("DOC1") {
		t.Fatal("a fresh policy must grant no copy")
	}
	p.AllowCopy("DOC1")
	if !p.mayCopy("DOC1") {
		t.Fatal("the grant did not name its source")
	}
	if p.mayCopy("DOC2") {
		t.Fatal("the grant names one source, and it named two")
	}
	if p.mayCopy("") {
		t.Fatal("an empty id must never be granted")
	}
}

// A second AllowCopy replaces the first rather than collecting both. The grant
// is one source, and a caller that names two has made a mistake the guard must
// not turn into two reachable copies.
func TestTheCopyGrantIsOneSource(t *testing.T) {
	p := NewPolicy()
	p.AllowCopy("DOC1")
	p.AllowCopy("DOC2")
	if p.mayCopy("DOC1") {
		t.Fatal("the first grant survived a second one")
	}
	if !p.mayCopy("DOC2") {
		t.Fatal("the second grant did not take")
	}
}
