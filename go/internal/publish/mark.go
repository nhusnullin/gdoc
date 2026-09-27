package publish

// The marker: a named range over the cover publish wrote, so a restyle knows
// where gdoc's own words end and the note's begin. The rule and its reason are
// in the package comment, under "publish marks its own cover".

import (
	"context"
	"fmt"

	"gdoc/internal/docs"
	"gdoc/internal/prelude"
)

// batchURL is the Docs write path, spelled here rather than imported: the
// marker is the one Docs write this package makes.
func batchURL(id string) string {
	return "https://docs.googleapis.com/v1/documents/" + id + ":batchUpdate"
}

// mark writes the marker over [1, end of the contents list) and reads it back.
// It raises nothing, for the reason verify raises nothing: the document exists
// whatever happens here, and an unmarked one is still correct.
func mark(ctx context.Context, s Session, id string, tab docs.Tab) (bool, []string) {
	end, ok := coverEnd(tab)
	if !ok {
		return false, []string{unmarked(fmt.Sprintf(
			"publish's cover ends at its contents list, and the body holds %d of them where it should hold one, so there is no single boundary to mark", countTOCs(tab)))}
	}
	body := map[string]any{"requests": []map[string]any{prelude.PublishedRequest(1, end)}}
	if err := s.PostJSON(ctx, batchURL(id), body, nil); err != nil && !sentAnyway(err) {
		return false, []string{unmarked(fmt.Sprintf("the marker batch failed: %v", err))}
	}
	// A batch whose answer was lost may have landed, so the read-back decides
	// that case too rather than the error.
	d, err := docs.Fetch(ctx, s, id)
	if err != nil {
		return false, []string{unmarked(fmt.Sprintf("the marker was sent and the document could not be read back to see it: %v", err))}
	}
	found, err := prelude.Published(d)
	if err != nil {
		return false, []string{unmarked(fmt.Sprintf("the marker read back in a shape gdoc does not make: %v", err))}
	}
	if len(found) != 1 || found[0].Start != 1 || found[0].End != end {
		return false, []string{unmarked(fmt.Sprintf(
			"the read-back carries %d ranges called %s, and it should carry one over [1,%d)", len(found), prelude.PublishedName, end))}
	}
	return true, nil
}

// coverEnd is where publish's cover ends: the end of the one contents list at
// the top level of the body. None, or more than one, is no boundary.
func coverEnd(tab docs.Tab) (int, bool) {
	var toc *docs.TOC
	for _, b := range tab.Body {
		if b.TOC == nil {
			continue
		}
		if toc != nil {
			return 0, false
		}
		toc = b.TOC
	}
	if toc == nil || toc.EndIndex <= 1 {
		return 0, false
	}
	return toc.EndIndex, true
}

func countTOCs(tab docs.Tab) int {
	n := 0
	for _, b := range tab.Body {
		if b.TOC != nil {
			n++
		}
	}
	return n
}

// unmarked is one reason the cover carries no marker, with what that costs.
func unmarked(why string) string {
	return fmt.Sprintf("the cover carries no %s marker, because %s. The document is there and correct, but a restyle will read its cover as body text and restyle it: do not run one on this document",
		prelude.PublishedName, why)
}
