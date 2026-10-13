package propose

import (
	"testing"

	"gdoc/internal/docs"
)

// TestFindSpanPlacesAQuoteInsideATableCell pins what paragraphs() says: a
// cell's paragraph is a paragraph, so the words kind is placed in a cell. Only
// the block kind refuses a table. A two-cell row, "Business" and "£20", and the
// quote is the second cell's words.
func TestFindSpanPlacesAQuoteInsideATableCell(t *testing.T) {
	raw := `{"documentId":"D","body":{"content":[
		{"startIndex":0,"endIndex":1,"sectionBreak":{"sectionStyle":{}}},
		{"startIndex":1,"endIndex":21,"table":{"rows":1,"columns":2,"tableRows":[{"tableCells":[
			{"startIndex":3,"endIndex":13,"content":[{"startIndex":4,"endIndex":13,"paragraph":{"elements":[
				{"startIndex":4,"endIndex":13,"textRun":{"content":"Business\n"}}]}}]},
			{"startIndex":13,"endIndex":19,"content":[{"startIndex":14,"endIndex":18,"paragraph":{"elements":[
				{"startIndex":14,"endIndex":18,"textRun":{"content":"£20\n"}}]}}]}]}]}}]}}`
	d, err := docs.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	r, err := FindSpan(d, "£20")
	if err != nil {
		t.Fatalf("a quote inside a table cell was refused: %v", err)
	}
	want := docs.Range{Tab: "t.0", Start: 14, End: 17}
	if r != want {
		t.Errorf("range = %+v, want %+v", r, want)
	}
}
