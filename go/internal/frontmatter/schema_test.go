package frontmatter

import (
	"strings"
	"testing"
	"time"
)

const testDocumentID = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"

func validBlock() *Block {
	return &Block{Schema: Schema, Documents: []Entry{{ID: testDocumentID}}}
}

// oneBlock is a schema 1 block as a note written before 2026-09-19 carries it:
// one document, and the field names Validate uses in its refusals are the old
// top-level ones.
func oneBlock(e Entry) *Block {
	return &Block{Schema: SchemaOne, Documents: []Entry{e}}
}

func TestValidateAcceptsTheMinimalBlock(t *testing.T) {
	if err := validBlock().Validate(); err != nil {
		t.Fatalf("minimal block refused: %v", err)
	}
}

func TestValidateNamesTheKeyItRefused(t *testing.T) {
	at := time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		block *Block
		names string
	}{
		{
			name:  "no schema",
			block: &Block{Documents: []Entry{{ID: testDocumentID}}},
			names: "schema",
		},
		{
			name:  "a schema this version does not know",
			block: &Block{Schema: 3, Documents: []Entry{{ID: testDocumentID}}},
			names: "schema",
		},
		{
			name:  "no document id",
			block: oneBlock(Entry{}),
			names: "document_id",
		},
		{
			name:  "a document id that is not a Drive id",
			block: oneBlock(Entry{ID: "not an id"}),
			names: "document_id",
		},
		{
			name:  "a document id too short to be a Drive id",
			block: oneBlock(Entry{ID: "1AbCdEfGhIj"}),
			names: "document_id",
		},
		{
			name:  "a folder id that is not a Drive id",
			block: oneBlock(Entry{ID: testDocumentID, FolderID: "../elsewhere"}),
			names: "folder_id",
		},
		{
			name: "a seen suggestion with a kind that is neither word",
			block: oneBlock(Entry{ID: testDocumentID, SuggestionsSeen: &SuggestionsSeen{
				At:    at,
				Items: []SuggestionSeen{{ID: "suggest.a1", Kind: "revision"}},
			}}),
			names: "suggestions_seen.items[0].kind",
		},
		{
			name: "a seen suggestion with no id",
			block: oneBlock(Entry{ID: testDocumentID, SuggestionsSeen: &SuggestionsSeen{
				At:    at,
				Items: []SuggestionSeen{{Kind: "insertion"}},
			}}),
			names: "suggestions_seen.items[0].id",
		},
		{
			name: "a publish record with no time",
			block: oneBlock(Entry{ID: testDocumentID,
				Published: &Published{Title: "Supplier register policy", House: "embedded"}}),
			names: "published.at",
		},
		{
			name: "a publish record with no title",
			block: oneBlock(Entry{ID: testDocumentID,
				Published: &Published{At: at, House: "embedded"}}),
			names: "published.title",
		},
		{
			name: "a proposal with no id",
			block: oneBlock(Entry{ID: testDocumentID,
				Proposals: []Proposal{{CommentID: "AAAA", At: at}}}),
			names: "proposals[0].id",
		},
		{
			name: "a proposal with no comment id",
			block: oneBlock(Entry{ID: testDocumentID,
				Proposals: []Proposal{{ID: "gdoc.p1", At: at}}}),
			names: "proposals[0].comment_id",
		},
		{
			name: "a proposal with no time",
			block: oneBlock(Entry{ID: testDocumentID,
				Proposals: []Proposal{{ID: "gdoc.p1", CommentID: "AAAA"}}}),
			names: "proposals[0].at",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.block.Validate()
			if err == nil {
				t.Fatalf("block accepted, wanted a refusal naming %q", c.names)
			}
			if !strings.Contains(err.Error(), c.names) {
				t.Fatalf("error %q does not name %q", err, c.names)
			}
		})
	}
}

func TestValidateRefusesANilBlock(t *testing.T) {
	var b *Block
	if err := b.Validate(); err == nil {
		t.Fatal("a nil block was accepted")
	}
}

func TestValidateAcceptsBothSuggestionKinds(t *testing.T) {
	at := time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC)
	for _, kind := range []string{KindInsertion, KindDeletion} {
		b := validBlock()
		b.Documents[0].SuggestionsSeen = &SuggestionsSeen{At: at, Items: []SuggestionSeen{{ID: "suggest.a1", Kind: kind}}}
		if err := b.Validate(); err != nil {
			t.Fatalf("kind %q refused: %v", kind, err)
		}
	}
}

// TestValidateAcceptsAPublishRecord is the other direction: the three facts M6
// records are enough, and house is optional in Validate because a block written
// before it existed carries none.
func TestValidateAcceptsAPublishRecord(t *testing.T) {
	b := validBlock()
	b.Documents[0].Published = &Published{
		At:    time.Date(2026, 9, 8, 14, 30, 0, 0, time.UTC),
		Title: "Supplier register policy",
		House: "embedded",
	}
	if err := b.Validate(); err != nil {
		t.Fatalf("a publish record was refused: %v", err)
	}
}

// TestValidateNamesTheEntryItRefused is the same rules over a list: the refusal
// names the key and which entry it is in, because a block with four documents
// in it is otherwise four places to look.
func TestValidateNamesTheEntryItRefused(t *testing.T) {
	at := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		block *Block
		names []string
	}{
		{
			name: "two entries naming one document",
			block: &Block{Schema: Schema, Documents: []Entry{
				{ID: testDocumentID},
				{ID: testDocumentID},
			}},
			names: []string{"documents[1].id", testDocumentID},
		},
		{
			name: "an exported record with neither field",
			block: &Block{Schema: Schema, Documents: []Entry{
				{ID: testDocumentID},
				{ID: "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz", Exported: &Exported{}},
			}},
			names: []string{"documents[1].exported"},
		},
		{
			name: "a tab id that is a path",
			block: &Block{Schema: Schema, Documents: []Entry{
				{ID: testDocumentID, TabID: "../t.0"},
			}},
			names: []string{"documents[0].tab_id"},
		},
		{
			name:  "a schema 2 block naming no document at all",
			block: &Block{Schema: Schema},
			names: []string{"documents"},
		},
		{
			name: "a schema 1 block naming two documents",
			block: &Block{Schema: SchemaOne, Documents: []Entry{
				{ID: testDocumentID},
				{ID: "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz"},
			}},
			names: []string{"schema"},
		},
		{
			name: "a proposal under the second entry",
			block: &Block{Schema: Schema, Documents: []Entry{
				{ID: testDocumentID},
				{ID: "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz",
					Proposals: []Proposal{{ID: "gdoc.p1", CommentID: "AAAA"}}},
			}},
			names: []string{"documents[1].proposals[0].at"},
		},
		{
			name: "a seen suggestion under the second entry",
			block: &Block{Schema: Schema, Documents: []Entry{
				{ID: testDocumentID},
				{ID: "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz",
					SuggestionsSeen: &SuggestionsSeen{At: at, Items: []SuggestionSeen{{ID: "suggest.a1", Kind: "revision"}}}},
			}},
			names: []string{"documents[1].suggestions_seen.items[0].kind"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.block.Validate()
			if err == nil {
				t.Fatalf("block accepted, wanted a refusal naming %v", c.names)
			}
			for _, want := range c.names {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not name %q", err, want)
				}
			}
		})
	}
}

// An unknown key inside an entry is the strict read's refusal, not Validate's,
// and it names the key the same way an unknown key beside the block does.
func TestAnUnknownKeyInsideAnEntryIsNamed(t *testing.T) {
	b, err := Read(fixture(t, "entry-unknown-key.md"))
	if err == nil {
		t.Fatalf("an unknown key inside an entry was accepted: %+v", b)
	}
	if !strings.Contains(err.Error(), "reviewed_by") {
		t.Errorf("error %q does not name reviewed_by", err)
	}
}

// A tab id is a Docs tab id, "t.0" and the like, so the rule is narrower than
// the Drive id rule and wider than nothing: what it exists to refuse is a path
// or a URL fragment landing in a field a later run reads.
func TestValidateAcceptsADocsTabID(t *testing.T) {
	for _, id := range []string{"t.0", "t.s7ld4qy0abcd", "kix.p1"} {
		b := validBlock()
		b.Documents[0].TabID = id
		if err := b.Validate(); err != nil {
			t.Errorf("tab_id %q refused: %v", id, err)
		}
	}
	for _, id := range []string{"../t.0", "t.0/t.1", "t 0", ""} {
		b := validBlock()
		b.Documents[0].TabID = id
		err := b.Validate()
		if id == "" {
			if err != nil {
				t.Errorf("an absent tab_id was refused: %v", err)
			}
			continue
		}
		if err == nil {
			t.Errorf("tab_id %q was accepted", id)
		}
	}
}

// An exported record with one field is enough: the stamp on the note it came
// from carries the time alone, and the copy beside it carries both.
func TestValidateAcceptsAnExportedRecord(t *testing.T) {
	at := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	for _, e := range []*Exported{{At: at}, {Note: "notes/a.md"}, {At: at, Note: "notes/a.md"}} {
		b := validBlock()
		b.Documents[0].Exported = e
		if err := b.Validate(); err != nil {
			t.Errorf("exported %+v refused: %v", e, err)
		}
	}
}
