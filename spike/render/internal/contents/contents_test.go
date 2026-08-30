package contents

import (
	"strings"
	"testing"
)

const fieldDocument = `<w:document><w:body>` +
	`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>` +
	`<w:r><w:fldChar w:fldCharType="begin"/><w:instrText>TOC \o "1-3"</w:instrText>` +
	`<w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>1. Stale entry</w:t></w:r></w:p>` +
	`<w:p><w:r><w:t>2. Another stale entry</w:t></w:r>` +
	`<w:r><w:fldChar w:fldCharType="end"/></w:r></w:p>` +
	`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>1-Purpose</w:t></w:r></w:p>` +
	`<w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr><w:r><w:t>1.1-Scope</w:t></w:r></w:p>` +
	`</w:body></w:document>`

func TestHeadingsAreFoundInDocumentOrder(t *testing.T) {
	found := Headings(fieldDocument)

	if len(found) != 3 {
		t.Fatalf("found %d headings, want 3: %#v", len(found), found)
	}
	if found[1].Text != "1-Purpose" || found[1].Level != 1 {
		t.Errorf("second heading = %#v", found[1])
	}
	if found[2].Text != "1.1-Scope" || found[2].Level != 2 {
		t.Errorf("third heading = %#v", found[2])
	}
}

func TestHeadingTextIsHeldUnescaped(t *testing.T) {
	// A <w:t> body is already escaped, and the write site escapes again.
	// Escaping twice puts "&amp;" on the contents page under a heading
	// reading "&", and breaks the page-number lookup with it.
	document := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>` +
		`<w:r><w:t>Risk &amp; Control</w:t></w:r></w:p>`

	found := Headings(document)

	if len(found) != 1 || found[0].Text != "Risk & Control" {
		t.Fatalf("found = %#v, want a single real ampersand", found)
	}
	if Escape(found[0].Text) != "Risk &amp; Control" {
		t.Errorf("re-escaped = %q", Escape(found[0].Text))
	}
}

func TestUnescapeDoesNotUndoAnEscapeTwice(t *testing.T) {
	if got := Unescape("&amp;lt;"); got != "&lt;" {
		t.Errorf("Unescape(\"&amp;lt;\") = %q, want %q", got, "&lt;")
	}
}

func TestTheFieldIsFoundByItsInstructionNotByBeingFirst(t *testing.T) {
	// A document re-saved from Word can carry a PAGE field earlier on. Splicing
	// over that leaves the real contents list where it is and reports success.
	withPageField := `<w:document><w:body>` +
		`<w:p><w:r><w:fldChar w:fldCharType="begin"/><w:instrText>PAGE</w:instrText>` +
		`<w:fldChar w:fldCharType="end"/></w:r></w:p>` + fieldDocument[len(`<w:document><w:body>`):]

	located, err := locateField(withPageField)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(withPageField[located.start:located.end], "1. Stale entry") {
		t.Error("the located field must span the stale contents list, not the PAGE field")
	}
	if strings.Contains(withPageField[located.start:located.end], ">PAGE<") {
		t.Error("the PAGE field must be left alone")
	}
}

func TestTheFieldSpansEveryParagraphOfItsCachedResult(t *testing.T) {
	// Taking the first end marker stops the field at the first entry, and the
	// rest of the stale list survives the splice, so the reader gets two lists.
	located, err := locateField(fieldDocument)
	if err != nil {
		t.Fatal(err)
	}

	if located.paragraphs != 2 {
		t.Errorf("field spans %d paragraphs, want both stale entries", located.paragraphs)
	}
	span := fieldDocument[located.start:located.end]
	if !strings.Contains(span, "2. Another stale entry") {
		t.Error("the second stale entry is outside the field, so it would survive")
	}
}

func TestAnUnclosedFieldIsRefusedRatherThanGuessedAt(t *testing.T) {
	unclosed := `<w:p><w:r><w:fldChar w:fldCharType="begin"/>` +
		`<w:instrText>TOC \o "1-3"</w:instrText></w:r></w:p>`

	if _, err := locateField(unclosed); err == nil {
		t.Fatal("a field with no end marker must be refused")
	}
}

func TestOurOwnBookmarksAreReplacedRatherThanAddedTo(t *testing.T) {
	// Bookmark names are unique document-wide. A second write without this
	// leaves two starts carrying the same name, which is invalid OOXML.
	document := `<w:p><w:bookmarkStart w:id="7" w:name="_gdoc_toc1"/>` +
		`<w:r><w:t>x</w:t></w:r><w:bookmarkEnd w:id="7"/></w:p>` +
		`<w:p><w:bookmarkStart w:id="8" w:name="theirs"/><w:bookmarkEnd w:id="8"/></w:p>`

	stripped := stripOurBookmarks(document)

	if strings.Contains(stripped, "_gdoc_toc1") {
		t.Error("our own bookmark survived the strip")
	}
	if !strings.Contains(stripped, `w:name="theirs"`) {
		t.Error("the template's own bookmark was removed")
	}
	if strings.Contains(stripped, `<w:bookmarkEnd w:id="7"/>`) {
		t.Error("our start went but its end stayed, leaving a dangling end tag")
	}
	if !strings.Contains(stripped, `<w:bookmarkEnd w:id="8"/>`) {
		t.Error("the template's end tag was removed with ours")
	}
}

func TestANewBookmarkIdClearsEveryIdAlreadyInUse(t *testing.T) {
	// Ends are read as well as starts: an id used only by a stray end tag still
	// counts. A duplicate id is invalid OOXML, and the file still parses.
	document := `<w:bookmarkStart w:id="3" w:name="a"/><w:bookmarkEnd w:id="41"/>`

	if got := nextBookmarkID(document); got != 42 {
		t.Errorf("next id = %d, want 42", got)
	}
}

func TestAnEntryDeclaresNoFontAndNoWeight(t *testing.T) {
	// Anything declared here overrides the template, and anything left
	// undeclared inherits the document default. Google supplies Arial and bold
	// only when nothing else does.
	line := entry(1, "1-Purpose", "4", "_gdoc_toc1", "", "", true)

	for _, forbidden := range []string{"<w:rFonts", "<w:b ", "<w:sz "} {
		if strings.Contains(line, forbidden) {
			t.Errorf("entry declares %s, which would override the template:\n%s",
				forbidden, line)
		}
	}
	if !strings.Contains(line, `w:val="IndexLink"`) {
		t.Error("the entry must carry IndexLink, or Word's Hyperlink style makes it blue")
	}
	if !strings.Contains(line, `w:anchor="_gdoc_toc1"`) {
		t.Error("the entry must link to the heading's bookmark")
	}
}

func TestTheRightTabSitsAtTheTextEdge(t *testing.T) {
	// The page width less both margins. A page size change has to move the tab
	// with it or the page numbers stop sitting at the margin.
	line := entry(1, "x", "1", "a", "", "", false)

	if !strings.Contains(line, `w:pos="9864" w:leader="dot"`) &&
		!strings.Contains(line, `w:pos="9863" w:leader="dot"`) {
		t.Errorf("no dotted right tab at the text edge:\n%s", line)
	}
}

func TestADeepHeadingIsCappedAtTOC3(t *testing.T) {
	line := entry(6, "deep", "9", "a", "", "", false)

	if !strings.Contains(line, `w:pStyle w:val="TOC3"`) {
		t.Errorf("level 6 must fall back to TOC3:\n%s", line)
	}
}

func TestAHeadingWithNoPageNumberGetsABlankRatherThanAGuess(t *testing.T) {
	line := entry(1, "Purpose", "", "a", "", "", false)

	if !strings.Contains(line, "<w:tab/><w:t></w:t>") {
		t.Errorf("want an empty page number, got:\n%s", line)
	}
}
