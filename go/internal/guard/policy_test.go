package guard

// This file holds one subject: what Judge decides, and what it says when it
// refuses. Hosts, path grammar and levels are all that subject. The transport
// is transport_test.go, the invariant that the two agree is wire_match_test.go,
// and milestone 1's acceptance run is acceptance_test.go.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func mustURL(t *testing.T, s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// TestARefusalNamesTheRuleItApplied holds what a refusal is for. It lands in
// the envelope's error field and it is the only thing the reader gets, so it
// has to say which rule refused and what the request would have had to be. Two
// rules that print the same sentence are one sentence too few: a reader cannot
// tell which of them to argue with.
func TestARefusalNamesTheRuleItApplied(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)

	cases := []struct{ name, method, url, want string }{
		{"not https", "GET", "http://docs.googleapis.com/v1/documents/DOC1", "https requests only"},
		{"another host", "GET", "https://evil.example.com/v1/documents/DOC1", "is not one gdoc talks to"},
		{"the wrong call on the token host", "GET", "https://oauth2.googleapis.com/token", "POST /token"},
		{"a docs path naming no document", "GET", "https://docs.googleapis.com/v1/documents", "names no document"},
		{"a verb the docs grammar does not have", "POST", "https://docs.googleapis.com/v1/documents/DOC1:copy", "batchUpdate"},
		{"outside the files collection", "GET", "https://www.googleapis.com/drive/v3/about", "outside /drive/v3/files"},
		{"a prefix that is not a segment", "GET", "https://www.googleapis.com/drive/v3/filesDOC1", "does not end that segment"},
		{"the collection itself", "GET", "https://www.googleapis.com/drive/v3/files", "files collection"},
		{"a sub-resource outside the grammar", "GET", "https://www.googleapis.com/drive/v3/files/DOC1/permissions", "at the suggest level"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := p.Judge(c.method, mustURL(t, c.url), nil)
			if err == nil {
				t.Fatal("want a refusal, got none")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal must say %q: %v", c.want, err)
			}
		})
	}
}

// The two drive-path rules are different rules, and a reader who cannot tell
// them apart cannot tell whether the path was wrong or the collection was.
func TestTheTwoDrivePathRefusalsReadDifferently(t *testing.T) {
	p := NewPolicy()
	outside := p.Judge("GET", mustURL(t, "https://www.googleapis.com/drive/v3/about"), nil)
	misread := p.Judge("GET", mustURL(t, "https://www.googleapis.com/drive/v3/filesDOC1"), nil)
	if outside == nil || misread == nil {
		t.Fatal("both must be refused")
	}
	if outside.Error() == misread.Error() {
		t.Fatalf("two rules, one sentence: %v", outside)
	}
}

func TestALevelNamesItselfInWords(t *testing.T) {
	if got := LevelSuggest.String(); got != "suggest" {
		t.Errorf("LevelSuggest prints %q; a refusal must not make the reader translate a number", got)
	}
	if got := LevelFull.String(); got != "full" {
		t.Errorf("LevelFull prints %q", got)
	}
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
	// A created file may be trashed with PATCH, and trashing is reversible.
	// DELETE on the file itself is not: it removes the document for everyone it
	// was shared with, with nothing to undo. The comment surface is the one
	// place DELETE is carried, because a comment gdoc wrote is gdoc's to
	// withdraw.
	p := NewPolicy()
	p.Learn("MADE1")
	if p.Judge("DELETE", mustURL(t, "https://www.googleapis.com/drive/v3/files/MADE1"), nil) == nil {
		t.Fatal("a hard delete must be refused even on a created file")
	}
}

// TestTheFilesCollectionIsOneGrammar holds the invariant filesCollection exists
// for. judgeDrive reads that path grammar to decide a POST is a create, and
// isCreate reads it to decide the parent check runs. Two copies could drift,
// and the direction that matters is a create Judge carries and the parent check
// never sees: it would reach Drive with a parent nobody checked.
func TestTheFilesCollectionIsOneGrammar(t *testing.T) {
	cases := []struct {
		path       string
		collection bool
	}{
		{"/drive/v3/files", true},
		{"/drive/v3/files/", true},
		{"/upload/drive/v3/files", true},
		{"/upload/drive/v3/files/", true},
		{"/drive/v3/files//", false},
		{"/drive/v3/filesX", false},
		{"/drive/v3/files-backup", false},
		{"/drive/v3/files/DOC1", false},
		{"/drive/v3/files/DOC1/comments", false},
		{"/upload/drive/v3/files/DOC1", false},
		{"/uploads/drive/v3/files", false},
		{"/drive/v3/about", false},
	}
	// Nothing is in the file set, so the only POST this policy carries is a
	// create on the collection.
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			if got := filesCollection(c.path); got != c.collection {
				t.Fatalf("filesCollection(%q) = %v, want %v", c.path, got, c.collection)
			}
			u := &url.URL{Scheme: "https", Host: "www.googleapis.com", Path: c.path}
			if got := isCreate(u, "POST"); got != c.collection {
				t.Errorf("isCreate reads %q as %v, so the parent check disagrees with the helper", c.path, got)
			}
			if got := p.Judge("POST", u, nil) == nil; got != c.collection {
				t.Errorf("Judge carries POST %q = %v, so the policy disagrees with the helper", c.path, got)
			}
		})
	}
}

// TestDriveSubResourcesOutsideTheGrammarAreRefused holds the plan's sentence
// that everything outside `{id}`, `{id}/export` and `{id}/comments*` is
// refused. A GET was allowed on any depth under a known id, which put
// /permissions (collaborator identities) and /revisions inside a level-1 read.
func TestDriveSubResourcesOutsideTheGrammarAreRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.Learn("MADE1")

	refused := []string{
		"https://www.googleapis.com/drive/v3/files/DOC1/permissions",
		"https://www.googleapis.com/drive/v3/files/DOC1/revisions",
		"https://www.googleapis.com/drive/v3/files/MADE1/permissions",
		"https://www.googleapis.com/drive/v3/files/DOC1/anything/else/deep",
	}
	for _, raw := range refused {
		t.Run(raw, func(t *testing.T) {
			if p.Judge("GET", mustURL(t, raw), nil) == nil {
				t.Errorf("%s is outside the grammar and must be refused", raw)
			}
		})
	}

	carried := []string{
		"https://www.googleapis.com/drive/v3/files/DOC1",
		"https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=x",
		"https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=x",
		"https://www.googleapis.com/drive/v3/files/DOC1/comments/C1/replies",
	}
	for _, raw := range carried {
		t.Run(raw, func(t *testing.T) {
			if err := p.Judge("GET", mustURL(t, raw), nil); err != nil {
				t.Errorf("%s is in the grammar and must be carried: %v", raw, err)
			}
		})
	}
}

// A prefix is not a segment. `/drive/v3/filesDOC1` is not file DOC1, and the
// guard must not judge a path it has misread.
func TestAPrefixIsNotASegment(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	for _, raw := range []string{
		"https://www.googleapis.com/drive/v3/filesDOC1",
		"https://www.googleapis.com/drive/v3/files-backup/DOC1",
	} {
		if p.Judge("GET", mustURL(t, raw), nil) == nil {
			t.Errorf("%s is not a file path and must be refused", raw)
		}
	}
}

// AllowCreateIn names a folder to create in. It does not put the folder in the
// reachable set: the set has exactly two doors, and a create target is neither.
func TestAllowCreateInDoesNotAdmitTheFolder(t *testing.T) {
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	if p.Judge("GET", mustURL(t, "https://www.googleapis.com/drive/v3/files/FOLDER1"), nil) == nil {
		t.Fatal("being allowed to create in a folder is not permission to read it")
	}
	if p.Judge("POST", mustURL(t, "https://www.googleapis.com/drive/v3/files/FOLDER1/comments"), nil) == nil {
		t.Fatal("nor to comment on it")
	}
	if err := p.Judge("POST", mustURL(t, "https://www.googleapis.com/drive/v3/files"), nil); err != nil {
		t.Fatalf("the create itself must still be allowed: %v", err)
	}
}

// A folder that was handed in at LevelFull and then named as the create target
// must not be quietly downgraded.
func TestAllowCreateInDoesNotDowngradeAKnownID(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("FOLDER1", LevelFull)
	p.AllowCreateIn("FOLDER1")
	if lvl, known := p.level("FOLDER1"); !known || lvl != LevelFull {
		t.Fatalf("naming a create target must not change a level: known=%v lvl=%d", known, lvl)
	}
}

func TestDocsPathsOutsideTheGrammarAreRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	for _, raw := range []string{
		"https://docs.googleapis.com/v1/documents",
		"https://docs.googleapis.com/v1/documents/",
		"https://docs.googleapis.com/v1/anything/DOC1",
		"https://docs.googleapis.com/",
	} {
		if p.Judge("GET", mustURL(t, raw), nil) == nil {
			t.Errorf("%s is not a document read and must be refused", raw)
		}
	}
	// A verb the grammar does not name, on an id that is in the set.
	if p.Judge("POST", mustURL(t, "https://docs.googleapis.com/v1/documents/DOC1:copy"), nil) == nil {
		t.Error("only batchUpdate is a POST verb here")
	}
}

// The host is matched exactly. A port, a different case and userinfo are all
// somebody else's host as far as the guard is concerned.
func TestHostEdgeCasesAreRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	for _, raw := range []string{
		"https://docs.googleapis.com:8443/v1/documents/DOC1",
		"https://DOCS.googleapis.com/v1/documents/DOC1",
		"https://docs.googleapis.com.evil.example/v1/documents/DOC1",
		"https://user:pass@docs.googleapis.com/v1/documents/DOC1",
	} {
		if err := p.Judge("GET", mustURL(t, raw), nil); err == nil {
			t.Errorf("%s must be refused", raw)
		}
	}
}

// TestACreateTheGuardCannotCheckIsRefused pins the upload shapes. uploadType
// decides what the body is: with `media` the body IS the file's content, so
// `{"parents":["FOLDER1"]}` would be read as bytes by Drive and as metadata by
// the parent check, and the file would land unparented while the guard learned
// its id at full level.
func TestACreateTheGuardCannotCheckIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	for _, raw := range []string{
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=media",
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=MEDIA",
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=",
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&uploadType=media",
	} {
		if err := p.Judge("POST", mustURL(t, raw), nil); err == nil {
			t.Errorf("%s must be refused: the guard cannot read parents out of that body", raw)
		}
	}
	for _, raw := range []string{
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart",
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=resumable",
		"https://www.googleapis.com/drive/v3/files",
	} {
		if err := p.Judge("POST", mustURL(t, raw), nil); err != nil {
			t.Errorf("%s must be carried: %v", raw, err)
		}
	}
}

// TestAReadMayNotAskForThePermissionSurface closes the query door on the rule
// that refuses /permissions. The path names one file and the level says read,
// but `fields` alone decides how much of that file comes back.
func TestAReadMayNotAskForThePermissionSurface(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.Learn("MADE1")
	for _, raw := range []string{
		"https://www.googleapis.com/drive/v3/files/DOC1?fields=*",
		"https://www.googleapis.com/drive/v3/files/DOC1?fields=permissions(emailAddress)",
		"https://www.googleapis.com/drive/v3/files/DOC1?fields=id,permissions/role",
		"https://www.googleapis.com/drive/v3/files/DOC1?fields=permissionIds",
		"https://www.googleapis.com/drive/v3/files/MADE1?fields=*",
		"https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=comments(author,permissions)",
		"https://www.googleapis.com/drive/v3/files/DOC1?q=name",
		"https://www.googleapis.com/drive/v3/files/DOC1?corpora=allDrives",
	} {
		if err := p.Judge("GET", mustURL(t, raw), nil); err == nil {
			t.Errorf("%s must be refused", raw)
		}
	}
	for _, raw := range []string{
		"https://www.googleapis.com/drive/v3/files/DOC1",
		"https://www.googleapis.com/drive/v3/files/DOC1?fields=id,name,mimeType",
		"https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=comments(id,content)&pageSize=100",
		"https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=text/markdown&alt=media",
	} {
		if err := p.Judge("GET", mustURL(t, raw), nil); err != nil {
			t.Errorf("%s must be carried: %v", raw, err)
		}
	}
}
