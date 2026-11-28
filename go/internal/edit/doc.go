// Package edit is a change written directly into a handed-in document, with
// its new words painted yellow, and the yellow taken off again.
//
// Nail's decision of 2026-11-28, DECISIONS.md: suggestions stay the default,
// and a request that says edit mode in words changes the text directly. The
// yellow is how he finds what changed, and `gdoc seen` is how he says he has
// read it. This comment holds the rules, the reason for each, and the test
// that pins it.
//
// # One shape: words for words, inside one paragraph
//
// An edit is the words kind of a proposals file and nothing else: a quote
// found exactly once, replaced by words that are not empty. A block is refused
// by name and stays a suggestion. A plain deletion has no shape here, because
// there would be nothing to paint and so nothing to find afterwards.
// TestAnEditIsWordsForWords is the pin.
//
// # The ground an edit stands on is plain
//
// The quoted words must carry no background, no link and one look, and the
// character in front of them must have that look too. Docs gives inserted text
// the style of its neighbour, so words written at a bold edge or behind a link
// would take a style nobody chose. And a quote that already carries a colour
// may be the author's own mark, which the clear must never be able to erase.
// TestAnEditSpanCarryingABackgroundOrStyleEdgeIsRefused is the pin.
//
// # The guard holds the span, and the revision
//
// Apply reads the document, finds its span, and hands exactly that span and
// that read's revision to the caller's grant before it writes. The guard then
// refuses a batch that deletes anything else or names another revision, and
// Docs refuses a batch whose revision the document has moved past. Nothing is
// retried: a moved document is a new read and a new judgement, which is the
// caller's. TestApplyGrantsExactlyTheSpanItRead and
// TestAnEditAnswerNamingNoRevisionStopsTheRun are the pins.
//
// # The read-back
//
// The document is read again after the write. The new words must stand where
// the quote stood, painted the edit yellow, carrying no suggestion id and the
// look the old words had. Verified false is reported and not raised, because
// the write happened. The read is a GET and the write a batchUpdate, so the
// read-back is a route the write did not go out on.
// TestAnEditIsReadBackPaintedAndNotPending is the pin.
//
// # The clear
//
// Painted is every body run whose background is exactly the edit yellow and
// which is not pending. Clear grants those spans, sets their background to no
// colour in one batch, and reads back that none is left and that the text is
// the same, character for character. A run of any other colour, the house
// yellow mark included, is not in the list and so is not in the batch.
// TestSeenClearsOnlyRunsOfExactlyTheEditYellow and
// TestAClearThatChangedAWordIsNotVerified are the pins.
package edit
