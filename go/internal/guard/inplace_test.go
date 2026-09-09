package guard

// This file holds one subject: the third write level. LevelInPlace is the door
// M1 shut and M7b opens, so every test here is written as an attack first. The
// allowlist is the point: a request kind that is not on it is refused whatever
// it is called, and the field mask is bounded because the four kinds that do
// carry can reset every property they do not set.

import (
	"strings"
	"testing"
)

const inPlaceURL = "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate"

// granted returns a policy holding DOC1 the way a restyle run holds it: handed
// in at LevelSuggest, then upgraded by the one line that opens direct edit.
func granted(t *testing.T) *Policy {
	t.Helper()
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.GrantInPlace("DOC1")
	if lvl, known := p.level("DOC1"); !known || lvl != LevelInPlace {
		t.Fatalf("GrantInPlace left DOC1 at %v, known=%v", lvl, known)
	}
	return p
}

// batch wraps one request kind in a batchUpdate body. The inner JSON is written
// out by each test, because what this file judges is exactly those bytes.
func batch(kind, inner string) []byte {
	return []byte(`{"requests":[{"` + kind + `":` + inner + `}]}`)
}

// suggestBatch is the same body carrying the field that used to be the only
// bound on a handed-in document.
func suggestBatch(kind, inner string) []byte {
	return []byte(`{"requests":[{"` + kind + `":` + inner + `}],"writeControl":{"writeMode":"SUGGEST"}}`)
}

// narrow is a mask naming one property, which is what every builder in this
// milestone writes.
const narrow = `{"fields":"namedStyleType","range":{"startIndex":1,"endIndex":2}}`

// The four kinds, each measured landing on a real document on 2026-09-09, and
// each carried here at LevelInPlace and nowhere else.
func TestTheFourStylingKindsCarryAtLevelInPlace(t *testing.T) {
	p := granted(t)
	kinds := []string{
		"updateDocumentStyle",
		"updateParagraphStyle",
		"updateTextStyle",
		"updateTableCellStyle",
	}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			if err := p.Judge("POST", mustURL(t, inPlaceURL), batch(kind, narrow)); err != nil {
				t.Fatalf("%s must carry at the in-place level: %v", kind, err)
			}
		})
	}
}

// This is the property, not the list. Every request kind that can insert,
// delete or replace content is refused at LevelInPlace, and a reader who adds a
// fifth kind to the allowlist answers this test rather than the list.
//
// createParagraphBullets is in here for the reason the plan was corrected on
// 2026-09-09: the reference says the leading tabs that set a bullet's nesting
// level "are removed by this request", so it deletes text the author typed. The
// fidelity probe missed it because its content had no leading tabs.
func TestNothingAtLevelInPlaceCanChangeACharacter(t *testing.T) {
	kinds := []string{
		"insertText",
		"deleteContentRange",
		"replaceAllText",
		"replaceNamedRangeContent",
		"insertTable",
		"insertTableRow",
		"insertTableColumn",
		"insertPageBreak",
		"insertInlineImage",
		"replaceImage",
		"createFootnote",
		"deleteTableRow",
		"mergeTableCells",
		"createParagraphBullets",
	}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			p := granted(t)
			err := p.Judge("POST", mustURL(t, inPlaceURL), batch(kind, `{"text":"x"}`))
			if err == nil {
				t.Fatalf("%s changes content and must be refused at the in-place level", kind)
			}
			if !strings.Contains(err.Error(), kind) {
				t.Errorf("the refusal must name the kind it refused; got %v", err)
			}
		})
	}
}

// The allowlist gates every batchUpdate on a granted id, whatever writeMode
// says. Hung off the direct-edit branch alone it would let a granted document
// take an insertText under SUGGEST, and the property above would be false on
// exactly the id it is meant to protect.
func TestASuggestBatchOnAGrantedIDMeetsTheAllowlistToo(t *testing.T) {
	p := granted(t)
	if err := p.Judge("POST", mustURL(t, inPlaceURL), suggestBatch("insertText", `{"text":"x"}`)); err == nil {
		t.Fatal("a SUGGEST insertText on a granted id must meet the allowlist, not walk around it")
	}
}

// The allowlist is scoped to LevelInPlace alone. Applied globally it breaks the
// probe's direct insertText at LevelFull and every propose batch at
// LevelSuggest, so it is pinned at all three levels here.
func TestTheAllowlistIsScopedToTheInPlaceLevel(t *testing.T) {
	t.Run("LevelFull carries a direct insertText", func(t *testing.T) {
		p := NewPolicy()
		p.Learn("MADE1")
		u := mustURL(t, "https://docs.googleapis.com/v1/documents/MADE1:batchUpdate")
		if err := p.Judge("POST", u, batch("insertText", `{"text":"x"}`)); err != nil {
			t.Fatalf("the probe writes directly into the document it created: %v", err)
		}
	})
	t.Run("LevelSuggest carries a SUGGEST insertText", func(t *testing.T) {
		p := NewPolicy()
		p.AllowFile("DOC1", LevelSuggest)
		if err := p.Judge("POST", mustURL(t, inPlaceURL), suggestBatch("insertText", `{"text":"x"}`)); err != nil {
			t.Fatalf("propose suggests text into a handed-in document: %v", err)
		}
	})
	t.Run("LevelInPlace refuses both", func(t *testing.T) {
		p := granted(t)
		if err := p.Judge("POST", mustURL(t, inPlaceURL), batch("insertText", `{"text":"x"}`)); err == nil {
			t.Fatal("a direct insertText must be refused at the in-place level")
		}
	})
}

// The allowlist bounds the kind. It does not bound the mask, and that is where
// this milestone's real danger lives: the reference says a field named in the
// mask and left unset is reset to its default, so an updateTextStyle carrying
// "*" resets bold, italic, links, colours and highlights over a range, with
// every character intact.
func TestAnInPlaceFieldMaskIsBounded(t *testing.T) {
	cases := []struct {
		name, inner, want string
	}{
		{"a star", `{"fields":"*"}`, "*"},
		{"a star in a path", `{"fields":"textStyle.*"}`, "*"},
		{"an empty mask", `{"fields":""}`, "empty"},
		{"a mask of commas", `{"fields":" , "}`, "empty"},
		{"no mask at all", `{"textStyle":{}}`, "fields"},
		{"the mask under another spelling", `{"Fields":"bold"}`, "fields"},
		// hasDuplicateKeys folds case and reaches every nested object, so two
		// spellings of the mask in one request are refused a layer out, by the
		// rule that stops the guard reading one copy while the server reads the
		// other. checkInPlaceMask holds the same rule for itself, so it is
		// honest read on its own.
		{"two spellings at once", `{"fields":"bold","Fields":"*"}`, "twice inside one object"},
		{"the mask is not a string", `{"fields":["bold"]}`, "fields"},
		{"the first-page header toggle", `{"fields":"useFirstPageHeaderFooter"}`, "useFirstPageHeaderFooter"},
		{"the even-page header toggle", `{"fields":"marginTop,useEvenPageHeaderFooter"}`, "useEvenPageHeaderFooter"},
		{"the toggle inside a path", `{"fields":"documentStyle.useFirstPageHeaderFooter"}`, "useFirstPageHeaderFooter"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := granted(t)
			err := p.Judge("POST", mustURL(t, inPlaceURL), batch("updateTextStyle", c.inner))
			if err == nil {
				t.Fatalf("%s must be refused", c.name)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal must name %q; got %v", c.want, err)
			}
		})
	}
}

// A mask naming several properties is ordinary work, and spaces around the
// commas are the field mask's own JSON encoding.
func TestAnOrdinaryMaskCarries(t *testing.T) {
	p := granted(t)
	body := batch("updateDocumentStyle", `{"fields":"marginTop, marginBottom,marginLeft"}`)
	if err := p.Judge("POST", mustURL(t, inPlaceURL), body); err != nil {
		t.Fatalf("a mask naming what it sets is the whole point: %v", err)
	}
}

// The one-way doors DECISIONS.md records, the two deletes nobody has a caller
// for, and a kind nobody has heard of. The last is the inversion of
// judgeRequests' unknown-kinds rule and the whole point of an allowlist here:
// what bounded an unknown kind all along was writeMode, and this level removes
// that bound.
func TestTheOneWayDoorsAndTheUnknownKindAreRefusedInPlace(t *testing.T) {
	kinds := []string{
		"deleteHeader",
		"deleteFooter",
		"deletePositionedObject",
		"deleteNamedRange",
		"createHeader",
		"updateSectionStyle",
		"createNamedRange",
		"aKindNobodyHasHeardOf",
	}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			p := granted(t)
			if err := p.Judge("POST", mustURL(t, inPlaceURL), batch(kind, `{}`)); err == nil {
				t.Fatalf("%s is not on the allowlist and must be refused", kind)
			}
		})
	}
}

// The suggestion family stays refused at this level, and it is refused by its
// own rule rather than by the allowlist, so the reader is told which one
// stopped them.
func TestTheSuggestionFamilyStaysRefusedInPlace(t *testing.T) {
	for _, kind := range []string{"acceptSuggestion", "deleteSuggestion", "rejectSuggestion"} {
		t.Run(kind, func(t *testing.T) {
			p := granted(t)
			body := batch(kind, `{"suggestionId":"s.1"}`)
			if err := p.Judge("POST", mustURL(t, inPlaceURL), body); err == nil {
				t.Fatalf("%s must be refused on a document nobody granted it for", kind)
			}
		})
	}
}

// The one door through the allowlist, and it is the door AllowReject already
// opened: one suggestion id the note records as gdoc's own. No command opens
// both grants, so this is written down rather than left to be discovered.
func TestAGrantedRejectStillCarriesInPlace(t *testing.T) {
	p := granted(t)
	p.AllowReject("s.mine")
	body := batch("rejectSuggestion", `{"suggestionId":"s.mine"}`)
	if err := p.Judge("POST", mustURL(t, inPlaceURL), body); err != nil {
		t.Fatalf("a rejectSuggestion the run granted must carry at every level: %v", err)
	}
	other := batch("rejectSuggestion", `{"suggestionId":"s.somebody-elses"}`)
	if p.Judge("POST", mustURL(t, inPlaceURL), other) == nil {
		t.Fatal("the grant is one id, and the in-place level does not widen it")
	}
}

// GrantInPlace upgrades a level. It is not a third door into the set, and the
// test that said so came back with the grant.
func TestGrantInPlaceNeverAdmitsAnUnknownID(t *testing.T) {
	p := NewPolicy()
	p.GrantInPlace("EVIL")
	if p.Judge("GET", mustURL(t, "https://docs.googleapis.com/v1/documents/EVIL"), nil) == nil {
		t.Fatal("GrantInPlace must not admit an id that was never given")
	}
}

// AllowFile is the side door around the grant's own invariant: it took any
// level, so a caller could have written the in-place level straight into the
// set and skipped the one line this milestone put the decision in. It admits
// nothing at that level, and it says so on the envelope rather than quietly.
func TestAllowFileCannotOpenTheInPlaceDoor(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelInPlace)
	if _, known := p.level("DOC1"); known {
		t.Fatal("AllowFile must not admit an id at the in-place level")
	}
	if p.Judge("GET", mustURL(t, "https://docs.googleapis.com/v1/documents/DOC1"), nil) == nil {
		t.Fatal("an id AllowFile refused must not be reachable at all")
	}
	warnings := p.Warnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "GrantInPlace") {
		t.Fatalf("the refusal must be on the envelope and must name the one door that opens it; got %v", warnings)
	}
}

// The levels are names, not a ladder. The grant reaches the document's own
// styling and nothing else, so the Drive file stays where LevelSuggest left it:
// a restyle cannot trash or rename the document it is styling.
func TestTheInPlaceGrantDoesNotReachTheDriveFile(t *testing.T) {
	p := granted(t)
	u := mustURL(t, "https://www.googleapis.com/drive/v3/files/DOC1")
	if p.Judge("PATCH", u, []byte(`{"trashed":true}`)) == nil {
		t.Fatal("a PATCH on the Drive file must stay refused at the in-place level")
	}
}

// A granted document is still read, exported and commented on, because the
// grant raised one thing and left the rest alone.
func TestAGrantedDocumentIsStillReadAndCommentedOn(t *testing.T) {
	p := granted(t)
	if err := p.Judge("GET", mustURL(t, "https://docs.googleapis.com/v1/documents/DOC1"), nil); err != nil {
		t.Fatalf("the read must still carry: %v", err)
	}
	u := mustURL(t, "https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=x")
	if err := p.Judge("POST", u, []byte(`{"content":"x"}`)); err != nil {
		t.Fatalf("the comment surface must still carry: %v", err)
	}
}
