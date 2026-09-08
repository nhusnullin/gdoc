package frontmatter

import (
	"strings"
	"testing"
	"time"
)

const testDocumentID = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"

func validBlock() *Block {
	return &Block{Schema: 1, DocumentID: testDocumentID}
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
			block: &Block{DocumentID: testDocumentID},
			names: "schema",
		},
		{
			name:  "a schema this version does not know",
			block: &Block{Schema: 2, DocumentID: testDocumentID},
			names: "schema",
		},
		{
			name:  "no document id",
			block: &Block{Schema: 1},
			names: "document_id",
		},
		{
			name:  "a document id that is not a Drive id",
			block: &Block{Schema: 1, DocumentID: "not an id"},
			names: "document_id",
		},
		{
			name:  "a document id too short to be a Drive id",
			block: &Block{Schema: 1, DocumentID: "1AbCdEfGhIj"},
			names: "document_id",
		},
		{
			name:  "a folder id that is not a Drive id",
			block: &Block{Schema: 1, DocumentID: testDocumentID, FolderID: "../elsewhere"},
			names: "folder_id",
		},
		{
			name: "a seen suggestion with a kind that is neither word",
			block: &Block{Schema: 1, DocumentID: testDocumentID, SuggestionsSeen: &SuggestionsSeen{
				At:    at,
				Items: []SuggestionSeen{{ID: "suggest.a1", Kind: "revision"}},
			}},
			names: "suggestions_seen.items[0].kind",
		},
		{
			name: "a seen suggestion with no id",
			block: &Block{Schema: 1, DocumentID: testDocumentID, SuggestionsSeen: &SuggestionsSeen{
				At:    at,
				Items: []SuggestionSeen{{Kind: "insertion"}},
			}},
			names: "suggestions_seen.items[0].id",
		},
		{
			name: "a publish record with no time",
			block: &Block{Schema: 1, DocumentID: testDocumentID,
				Published: &Published{Title: "Supplier register policy", House: "embedded"}},
			names: "published.at",
		},
		{
			name: "a publish record with no title",
			block: &Block{Schema: 1, DocumentID: testDocumentID,
				Published: &Published{At: at, House: "embedded"}},
			names: "published.title",
		},
		{
			name: "a proposal with no id",
			block: &Block{Schema: 1, DocumentID: testDocumentID,
				Proposals: []Proposal{{CommentID: "AAAA", At: at}}},
			names: "proposals[0].id",
		},
		{
			name: "a proposal with no comment id",
			block: &Block{Schema: 1, DocumentID: testDocumentID,
				Proposals: []Proposal{{ID: "gdoc.p1", At: at}}},
			names: "proposals[0].comment_id",
		},
		{
			name: "a proposal with no time",
			block: &Block{Schema: 1, DocumentID: testDocumentID,
				Proposals: []Proposal{{ID: "gdoc.p1", CommentID: "AAAA"}}},
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
		b.SuggestionsSeen = &SuggestionsSeen{At: at, Items: []SuggestionSeen{{ID: "suggest.a1", Kind: kind}}}
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
	b.Published = &Published{
		At:    time.Date(2026, 9, 8, 14, 30, 0, 0, time.UTC),
		Title: "Supplier register policy",
		House: "embedded",
	}
	if err := b.Validate(); err != nil {
		t.Fatalf("a publish record was refused: %v", err)
	}
}
