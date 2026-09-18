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
package annotate
