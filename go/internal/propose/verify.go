package propose

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gdoc/internal/docs"
	"gdoc/internal/docx"
)

// PreviewURL is the document as it reads with every pending suggestion hidden.
//
// It is the same read the three read commands make, on the other view. A
// suggestion changes nothing until somebody accepts it, so the original words
// must still be here: this is the only route that can tell a suggestion from an
// edit, and telling them apart is what the whole milestone rests on.
func PreviewURL(id string) string {
	return "https://docs.googleapis.com/v1/documents/" + id +
		"?includeTabsContent=true" +
		"&suggestionsViewMode=PREVIEW_WITHOUT_SUGGESTIONS"
}

// Verify reads the document back through three routes and says which of them
// held, along with the suggestion ids the first one found.
//
// It never fails. Each route answers for itself, a route that could not be read
// is a warning naming it, and the caller reports the checks as they came. A
// verification that raised would hide a change that is already in the document.
func Verify(ctx context.Context, s Session, docID string, r docs.Range, p Proposal, commentBody string) (Checks, []string, []string) {
	var checks Checks
	var warns []string

	inline, err := docs.Fetch(ctx, s, docID)
	if err != nil {
		warns = append(warns, fmt.Sprintf("the document could not be read back with suggestions inline, so the change could not be confirmed as pending: %v", err))
	}
	var ids []string
	if inline != nil {
		var why string
		checks.SuggestionsInline, ids, why = inlineHolds(inline, r, p)
		if why != "" {
			warns = append(warns, why)
		}
	}

	preview, err := fetchPreview(ctx, s, docID)
	switch {
	case err != nil:
		warns = append(warns, fmt.Sprintf("the document could not be read back in the preview view, so the change could not be told from a direct edit: %v", err))
	case StartsAt(preview, r, p.Quoted):
		checks.PreviewWithoutSuggestions = true
	default:
		warns = append(warns, fmt.Sprintf(
			"the preview view does not carry the quoted text %q where the change was made, which is what a direct edit looks like rather than a suggestion", p.Quoted))
	}

	anchored, why := docxHolds(ctx, s, docID, commentBody)
	checks.DocxAnchored = anchored
	if why != "" {
		warns = append(warns, why)
	}
	return checks, ids, warns
}

// fetchPreview reads the document with pending suggestions hidden. It is
// docs.Fetch on the other URL, decoded by the same parser, so the two views
// cannot drift apart in how they are read.
func fetchPreview(ctx context.Context, s Session, docID string) (*docs.Document, error) {
	var raw json.RawMessage
	if err := s.GetJSON(ctx, PreviewURL(docID), &raw); err != nil {
		return nil, err
	}
	return docs.Parse(raw)
}

// inlineHolds is the first check: at the start of the span the write named,
// the replacement is there carrying a suggestion id, and the quoted words are
// there carrying a deletion id.
//
// Both halves matter. An insertion alone would be the new words added beside
// the old ones, and a deletion alone would be the old words struck out with
// nothing put in their place.
func inlineHolds(d *docs.Document, r docs.Range, p Proposal) (bool, []string, string) {
	runs := textRuns(d, r.Tab)
	i := 0
	for ; i < len(runs); i++ {
		if runs[i].StartIndex == r.Start && len(runs[i].InsertionIDs) > 0 {
			break
		}
	}
	if i == len(runs) {
		return false, nil, fmt.Sprintf("the read-back carries no suggested insertion at index %d, where the replacement was written", r.Start)
	}

	// Docs cuts one insert into as many runs as it likes, so the inserted text
	// is every run from here on that carries an insertion id.
	var inserted strings.Builder
	var ids []string
	seen := map[string]bool{}
	for ; i < len(runs) && len(runs[i].InsertionIDs) > 0; i++ {
		inserted.WriteString(runs[i].Text)
		for _, id := range runs[i].InsertionIDs {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	if inserted.String() != p.Replacement {
		return false, ids, fmt.Sprintf("the read-back carries %q as the suggested insertion where %q was written", inserted.String(), p.Replacement)
	}

	var deleted strings.Builder
	for ; i < len(runs) && len(runs[i].DeletionIDs) > 0; i++ {
		deleted.WriteString(runs[i].Text)
	}
	if deleted.String() != p.Quoted {
		return false, ids, fmt.Sprintf("the read-back carries %q as the suggested deletion where %q was quoted", deleted.String(), p.Quoted)
	}
	return true, ids, ""
}

// docxHolds is the third check: the export carries the comment gdoc wrote, and
// it is attached to text rather than floating.
//
// The export is the only truthful answer to whether a comment is anchored,
// which is why internal/docx exists. An export that did not arrive says nothing
// about it, so it is a warning and a false check, never a true one.
func docxHolds(ctx context.Context, s Session, docID, body string) (bool, string) {
	raw, err := docx.Export(ctx, s, docID)
	if err != nil {
		return false, fmt.Sprintf("the docx export could not be read, so the comment could not be confirmed as anchored: %v", err)
	}
	f, err := docx.Parse(raw)
	if err != nil {
		return false, fmt.Sprintf("the docx export did not parse, so the comment could not be confirmed as anchored: %v", err)
	}
	for _, c := range f.Comments {
		if sameWords(c.Text, body) {
			if c.Anchored {
				return true, ""
			}
			return false, fmt.Sprintf("the export carries the comment %q and it is not attached to any text", body)
		}
	}
	return false, fmt.Sprintf("the export carries no comment reading %q, so the explanation may not have been saved with the change", body)
}

// sameWords compares two comment bodies the way internal/docx's own match does:
// the export re-wraps a comment across runs, so the whitespace is the export's
// rather than the author's.
func sameWords(a, b string) bool {
	return strings.Join(strings.Fields(a), " ") == strings.Join(strings.Fields(b), " ")
}

// textRuns is one tab's text runs in reading order, tables walked into. The
// order is what the checks above read the document as: one insert followed by
// the words it replaces.
func textRuns(d *docs.Document, tabID string) []docs.Run {
	for _, t := range d.Tabs {
		if t.ID == tabID {
			return blockRuns(t.Body)
		}
	}
	return nil
}

func blockRuns(bs []docs.Block) []docs.Run {
	var out []docs.Run
	for _, b := range bs {
		if b.Paragraph != nil {
			for _, r := range b.Paragraph.Runs {
				if r.Kind == docs.KindText {
					out = append(out, r)
				}
			}
			continue
		}
		for _, row := range b.Table {
			for _, c := range row {
				out = append(out, blockRuns(c.Blocks)...)
			}
		}
	}
	return out
}
