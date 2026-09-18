// This file is the two read-backs: Drive's comment listing and the docx export,
// each asked after the batch has gone out, each answering for itself. annotate.go
// holds the write they answer about.

package annotate

import (
	"context"
	"fmt"
	"strings"

	"gdoc/internal/comments"
	"gdoc/internal/docx"
)

// Verify reads the comment back through the two routes the write did not go out
// on, and says which of them held.
//
// It never fails. The comment is in the document by the time this runs, so
// every answer here is a fact about something that already happened: a route
// that could not be read is a warning naming it, a route that read something
// else is a warning saying what, and the caller reports the checks as they came.
// A verification that raised would hide a comment that is already there, and the
// caller that heard it failed would write the comment a second time.
//
// The two routes answer different halves of one question, and neither is
// enough. Drive's listing says the comment exists with the words that were sent,
// which the export cannot say because it carries no comment id. The export says
// those words are wrapped around the quote, which the listing cannot say
// because Drive keeps reporting the text a destroyed anchor used to hold.
func Verify(ctx context.Context, s Session, docID, commentID, quoted, body string) (Checks, []string) {
	var checks Checks
	var warns []string

	listed, why := listingHolds(ctx, s, docID, commentID, quoted, body)
	checks.DriveListing = listed
	if why != "" {
		warns = append(warns, why)
	}

	anchored, why := docxHolds(ctx, s, docID, quoted, body)
	checks.DocxAnchored = anchored
	if why != "" {
		warns = append(warns, why)
	}
	return checks, warns
}

// listingHolds is the first route: Drive lists the comment under the id the
// write answered with, reading the words that were sent, anchored to the words
// that were quoted.
//
// It is a different product answering about the same comment. The write went to
// the Docs API and this comes from Drive, so a comment Docs reported as saved
// and Drive does not list is the disagreement worth hearing about.
func listingHolds(ctx context.Context, s Session, docID, commentID, quoted, body string) (bool, string) {
	if commentID == "" {
		// Nothing to look for. Searching the listing for an empty id would find
		// nothing and report the comment as missing, which blames the document
		// for an answer that could not be read.
		return false, "the write answered no comment id, so Drive's listing could not be asked whether the comment is there"
	}
	listed, err := comments.Fetch(ctx, s, docID, nil)
	if err != nil {
		return false, fmt.Sprintf("Drive's comment listing could not be read, so the comment could not be confirmed as saved: %v", err)
	}
	for _, c := range listed {
		if c.ID != commentID {
			continue
		}
		if !sameWords(c.Content, body) {
			return false, fmt.Sprintf("Drive's listing carries comment %s reading %q rather than the %q that was sent", commentID, c.Content, body)
		}
		anchor := ""
		if c.QuotedFileContent != nil {
			anchor = c.QuotedFileContent.Value
		}
		if anchor == "" {
			return false, fmt.Sprintf("Drive's listing carries comment %s with no quoted text, so it is attached to nothing Drive can name", commentID)
		}
		if !holdsWords(anchor, quoted) {
			return false, fmt.Sprintf("Drive's listing anchors comment %s to %q rather than to the quoted %q", commentID, anchor, quoted)
		}
		return true, ""
	}
	return false, fmt.Sprintf("Drive's listing carries no comment %s, so the write may not have been saved", commentID)
}

// docxHolds is the second route: the export carries the comment gdoc wrote, it
// is attached to text rather than floating, and the text it is attached to is
// the quote.
//
// The export is the only truthful answer to whether a comment is anchored,
// which is why internal/docx exists. An export that did not arrive says nothing
// about it, so it is a warning and a false check, never a true one. internal/
// propose reads the same route for its own comment, in the same shape and with
// the same rules, and adds no question about the span: there the anchor is the
// replacement, and the words on the page are about to be different ones.
func docxHolds(ctx context.Context, s Session, docID, quoted, body string) (bool, string) {
	raw, err := docx.Export(ctx, s, docID)
	if err != nil {
		return false, fmt.Sprintf("the docx export could not be read, so the comment could not be confirmed as anchored: %v", err)
	}
	f, err := docx.Parse(raw)
	if err != nil {
		return false, fmt.Sprintf("the docx export did not parse, so the comment could not be confirmed as anchored: %v", err)
	}
	var hits []docx.Comment
	for _, c := range f.Comments {
		if sameWords(c.Text, body) {
			hits = append(hits, c)
		}
	}
	if len(hits) == 0 {
		return false, fmt.Sprintf("the export carries no comment reading %q, so the comment may not have been saved", body)
	}
	// Two annotations in one run may carry the same reason, and the export has
	// no Drive comment id to tell one from the other. Comments reading the same
	// words that disagree about where they are attached give no answer: taking
	// the first would report this comment on the strength of another one. It is
	// internal/docx's question on the same join and one more: docx compares
	// whether the candidates are anchored, and this compares the words they are
	// attached to as well, because the span is what is being verified here.
	for _, c := range hits[1:] {
		if c.Anchored != hits[0].Anchored || c.Span != hits[0].Span {
			return false, fmt.Sprintf(
				"the export carries %d comments reading %q and they do not agree about the text they are attached to, so this one could not be told from the others", len(hits), body)
		}
	}
	if !hits[0].Anchored {
		return false, fmt.Sprintf("the export carries the comment %q and it is not attached to any text", body)
	}
	if !holdsWords(hits[0].Span, quoted) {
		return false, fmt.Sprintf("the export attaches the comment %q to %q rather than to the quoted %q", body, hits[0].Span, quoted)
	}
	return true, ""
}

// sameWords compares two comment bodies the way internal/docx's own match does:
// the export re-wraps a comment across runs, so the whitespace is the export's
// rather than the author's.
func sameWords(a, b string) bool {
	return words(a) == words(b)
}

// holdsWords says whether an anchor holds the quoted words, on the same reading
// of whitespace.
//
// It is containment rather than equality because an anchor is the document's
// answer about where a comment sits, and both routes have been seen to widen
// one: Drive returns the quoted text it stored, and the export's span is every
// run between the range marks, which the editor may have split differently from
// where the quote begins and ends. Widening is the honest shape to allow, and a
// comment placed on other words fails it either way.
func holdsWords(anchor, quoted string) bool {
	return strings.Contains(words(anchor), words(quoted))
}

// words is one string with its whitespace made the same as every other's.
func words(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
