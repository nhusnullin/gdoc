package guard

import "testing"

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
