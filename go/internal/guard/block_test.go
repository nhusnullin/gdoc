package guard

// This file holds one subject: the block proposal, and the proof that it needed
// no new permission.
//
// A block proposal is new paragraphs, headings and lists, placed after a quoted
// paragraph or in place of a run of whole paragraphs. It is one SUGGEST
// batchUpdate on a handed-in document, which is what propose has sent every day
// since M1. So this milestone added no grant, and this test is what says so: if
// it ever needs one to pass, the block has stopped being a suggestion and the
// wall in front of somebody's document has moved.
//
// It is the block's half of TestThePreludeNeedsNoGrantAtAll, which says the
// same about the house prelude. Both send deleteParagraphBullets, built by
// docsreq.DeleteBullets, to take off a list marker the insert inherited: a
// block for every new paragraph that is not a list item, and the prelude only
// when it lands in front of one.

import (
	"strings"
	"testing"
)

// blockKinds is the batch the block builder writes, in its order: the insert,
// the named style per new paragraph, the updateTextStyle that clears the marks
// the insert inherited, the marks of the block's own runs, the bullets of each
// list run, the bullet removal over every new paragraph that is not a list
// item, the deletion of the paragraphs a replace stands in for, and the comment
// carrying the reason.
// The indexes are the ones propose's own builder computes for this content
// inserted at 41, so a reader comparing the two files sees one write:
// propose/testdata/block-replace-batch.json is the same batch under the
// builder's own test.
var blockKinds = []string{
	`{"insertText":{"location":{"index":41},"text":"3.6 Limits\nBody text with bold.\none\ntwo\n"}}`,
	`{"updateParagraphStyle":{"paragraphStyle":{"namedStyleType":"HEADING_2"},"fields":"namedStyleType","range":{"startIndex":41,"endIndex":52}}}`,
	`{"updateParagraphStyle":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},"fields":"namedStyleType","range":{"startIndex":52,"endIndex":73}}}`,
	`{"updateParagraphStyle":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},"fields":"namedStyleType","range":{"startIndex":73,"endIndex":77}}}`,
	`{"updateParagraphStyle":{"paragraphStyle":{"namedStyleType":"NORMAL_TEXT"},"fields":"namedStyleType","range":{"startIndex":77,"endIndex":81}}}`,
	`{"updateTextStyle":{"textStyle":{},"fields":"bold,italic,underline,strikethrough,link","range":{"startIndex":41,"endIndex":81}}}`,
	`{"updateTextStyle":{"textStyle":{"bold":true},"fields":"bold","range":{"startIndex":67,"endIndex":71}}}`,
	`{"createParagraphBullets":{"range":{"startIndex":73,"endIndex":81},"bulletPreset":"BULLET_DISC_CIRCLE_SQUARE"}}`,
	`{"deleteParagraphBullets":{"range":{"startIndex":41,"endIndex":73}}}`,
	`{"deleteContentRange":{"range":{"startIndex":81,"endIndex":119}}}`,
	`{"insertComment":{"range":{"startIndex":41,"endIndex":51},"content":"[gdoc] why this section is here"}}`,
}

// The whole block batch, on a document handed in and granted nothing.
func TestABlockProposalNeedsNoGrant(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)

	body := []byte(`{"requests":[` + strings.Join(blockKinds, ",") + `],"writeControl":{"writeMode":"SUGGEST"}}`)
	if err := p.Judge("POST", mustURL(t, inPlaceURL), body); err != nil {
		t.Fatalf("a block proposal is a SUGGEST batch on a handed-in document, and that has always carried: %v", err)
	}

	t.Run("and the same batch direct is still refused", func(t *testing.T) {
		p := NewPolicy()
		p.AllowFile("DOC1", LevelSuggest)
		direct := []byte(`{"requests":[` + strings.Join(blockKinds, ",") + `]}`)
		if p.Judge("POST", mustURL(t, inPlaceURL), direct) == nil {
			t.Fatal("without SUGGEST the same requests are a direct edit of somebody's document")
		}
	})

	// Each kind alone, so a failure names the request that stopped carrying
	// rather than the batch it sat in.
	t.Run("each kind on its own", func(t *testing.T) {
		for _, req := range blockKinds {
			kind := req[2:strings.Index(req, `":`)]
			t.Run(kind, func(t *testing.T) {
				p := NewPolicy()
				p.AllowFile("DOC1", LevelSuggest)
				one := []byte(`{"requests":[` + req + `],"writeControl":{"writeMode":"SUGGEST"}}`)
				if err := p.Judge("POST", mustURL(t, inPlaceURL), one); err != nil {
					t.Fatalf("%s carries in a SUGGEST batch: %v", kind, err)
				}
				if p.Judge("POST", mustURL(t, inPlaceURL), []byte(`{"requests":[`+req+`]}`)) == nil {
					t.Fatalf("%s without SUGGEST is a direct edit and must be refused", kind)
				}
			})
		}
	})
}
