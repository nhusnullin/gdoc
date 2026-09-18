// Package annotate leaves one comment on quoted words inside a Google Doc,
// anchored to those words, and changes nothing else.
//
// It is the least a writer can do inside somebody's document. The batch it
// sends holds one insertComment and nothing beside it, so no character can move
// whatever Google does with the write mode, and that is why this writer does
// not run internal/probe first: the probe answers whether SUGGEST makes a real
// suggestion today or a silent direct edit, and a batch that cannot edit has no
// stake in the answer. DECISIONS.md holds that decision and its date.
//
// Nothing here decides whether a comment is worth making, or what to say in it.
// The words arrive written and the placement arrives quoted, in a file the skill
// wrote. This package finds the words, writes what it was told, and reports what
// it saw.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # An annotation names text, never an index
//
// The caller hands over the exact words to comment on, and the span is looked
// up in a document that has just come back, through propose.FindSpan. The span
// walk has one owner: a quote that occurs twice, a quote that is not there, and
// a quote crossing a footnote mark or a smart chip are refused there, by the
// same rule and with the same words as they are for a proposal. A quote carrying
// a line break is refused here instead, before the document is read, because
// that walk reads one paragraph at a time and such a quote could only ever come
// back as "not found". TestCheckRefusesEachBadShapeByName is the pin.
//
// # The prefix is added here, and required in reply
//
// Body writes the comment as Prefix + Why, so Check refuses a reason that
// already opens with the robot. internal/reply does the opposite: a reply body
// arrives with the mark already on it and reply.Check refuses a body without it.
// The two writers own the mark differently on purpose, and internal/plaintext
// holds the argument. A caller carrying the reply convention over to an
// annotations file would sign the comment twice, and nothing downstream would
// catch it: the read-backs compare against the string that was sent, so both
// hold over a doubled mark in the one string that records who wrote the words.
// TestCheckRefusesEachBadShapeByName refuses it, TestCheckAcceptsAPlainAnnotation
// is the other direction, and TestBodyIsTheRobotAndTheWhy states the mark as a
// literal.
//
// # No markdown, because a thread renders it literally
//
// The reason is asked of plaintext.Markdown, alone rather than behind the
// prefix, for the reason propose.Check gives: the heading arm is anchored to a
// line start, and the prefix in front of it would let a reason opening with a
// hash through. TestCheckRefusesEachBadShapeByName carries that case.
//
// # Two read-backs, and nothing raises after the write
//
// Once the batch has gone out the comment is in somebody's document, so
// everything from there is reported rather than raised: a caller told the run
// failed is a caller that writes the comment a second time.
// TestAListingWithoutTheIdIsAWarningNotAnError and
// TestAReadBackThatFailedIsAWarningNamingTheRoute hold that, each on a route
// that could not be read or did not hold.
//
// The comment is read back through two routes the write did not go out on, and
// neither is enough alone. Drive's comment listing says the comment exists with
// the words that were sent, which the export cannot say because it carries no
// comment id. The docx export says those words are wrapped around the quote,
// which the listing cannot say because Drive keeps reporting the text a
// destroyed anchor used to hold, and a destroyed anchor is the failure this
// writer has. Both holding is verified, and TestVerifiedIsBothRoutesHolding is
// the pin. TestAnExportThatDoesNotAnchorTheCommentIsAWarning and
// TestAnExportAnchoredToOtherWordsIsAWarning are the two ways the second route
// answers no.
//
// Comments in the export that read the same words and disagree about the text
// they are attached to give no answer, because the export has no Drive comment
// id to tell one from another and taking the first would report this comment on
// the strength of somebody else's. It is the rule internal/docx's own witness
// follows on the same join, and TestTwoExportedCommentsThatDisagreeGiveNoAnswer
// pins it.
package annotate
