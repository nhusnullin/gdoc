// Package restyle is the survey, what a document holds before anything is done
// to it, and the house look applied to that document where it stands.
//
// Two commands live on it. The survey is `restyle --dry-run`, which reads and
// writes nothing at all. The apply is `restyle --from survey.json`, which gives
// a handed-in document the house style: the page geometry, each paragraph's
// spacing and indent, each run's face, size and colour, and each table cell's
// padding and borders. They are two runs on purpose, and the section below on
// the two-run rule says why.
//
// Apply is the only room here that sends anything. PageRequest, TabRequests and
// TabRequestsExcept are pure functions of the house style and the document, and
// the survey takes the three decoded reads as a value, so every case in this
// package is testable on something a test wrote out.
//
// Nothing here judges anything. The survey reports counts, ids and a witness
// per thread, and the read-back reports what moved. Whether a document is worth
// restyling, and whether a count that moved matters, are the skill's and past
// that Nail's.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The survey writes nothing
//
// It reports the threads with a witness for each, what is pending, the chips,
// the tabs, the named ranges and the revision the reads were made against. It
// writes to no document and to no file. The one file any run of it can touch is
// the saved OAuth token, which every command replaces on refresh.
//
// Three reads, not four: the comment listing, the Docs read, and the docx
// export the witness is read from. The named ranges come out of the Docs read,
// which carries them already. This comment said four once, and the code below
// it said three: a count in a doc comment that the code contradicts is read
// before the code is.
//
// The listing goes out before the Docs read, which is the order every poll in
// this binary holds and which internal/comments states once for all of them.
// Read first, a comment written in the gap is a comment in the listing with no
// anchor in a document read a moment before it existed, so the survey would
// report a thread placed nowhere.
// TestRestyleListsTheCommentsBeforeItReadsTheDocument in cmd/gdoc is what stops
// a later edit from swapping them.
//
// A failed Docs read is a failed survey, and a failed export is a warning. The
// chips, the pending suggestions, the named ranges, the tab count and the
// revision id are all in that one read, so it has no honest partial answer. The
// export has one: every thread unmatched with a warning, on an envelope that
// still carries the threads, exactly as `comments --witness` behaves.
// TestRestyleFailsWhenADocumentReadFailed and
// TestAnExportThatCouldNotBeReadLeavesEveryThreadUnmatched are the pins, and
// exportFile in cmd/gdoc is the one export path both callers use, because two
// would be two chances for one of them to ask Drive for a different document or
// to read the answer to a different ceiling.
//
// The pending count is suggestions.All's, never List's. List drops a suggestion
// whose text is only whitespace, and a whitespace-only suggestion is still
// something a replacement would destroy, so counting from List reports nothing
// to protect on a document that has something to protect.
// TestThePendingCountComesFromAllAndNotFromList is the pin.
//
// The witness is carried twice, as counts and as one line per thread. The
// read-back compares before with after per thread, which totals cannot answer,
// and the survey is the only place the before-witness is read: a thread
// detached before the work reads as damage the work did when only the totals
// are kept. The witness keeps the two limits it has everywhere else, and they
// are worth stating rather than discovering: it names a destroyed anchor and
// not a moved one, and two exported comments that share words and disagree give
// no answer for either.
//
// The revision id is why the survey is machine-readable. The apply hands it
// back and refuses a document that moved, which is the reachable-set principle
// at the one moment gdoc has the power to overwrite.
//
// Schema is 1, and the apply refuses a survey that states anything else. The
// apply reads this file as the record of what the document held, and a field
// added later is a field an older survey simply does not carry. Ids is the
// field that made the point: it arrived with the apply, and a survey printed
// before it read as a survey where nothing was pending, so a suggestion the run
// destroyed would have been reported as one that had never been there. A strict
// decoder refuses a key it does not know and says nothing at all about a key
// that is absent, so the version is what closes that half.
// TestRestyleReadsTheSurveyStrictly and
// TestASurveyThisBinaryPrintedIsOneItCanApply are the pins. Bumping Schema is a
// decision, not a refactor.
//
// # NothingToProtect is five zeros, never a recommendation
//
// No threads, no pending suggestions, no chips, no paragraph element the
// decoder could not name, and no named range.
// TestNothingToProtectIsTrueOnlyWhenAllFiveCountsAreZero is the pin.
//
// Pending is two counts rather than one. Pending is suggestions.All's, and
// OnElements is the ids on the runs the pending walk skips, which is every run
// that is not text. Counting only the first answered "nothing to protect" over
// a suggested page break. The rule is what the listing can report and not
// whether the run holds text, so a footnote reference carries its number and is
// counted here all the same, because the pending walk skips it too. The two are
// in different units, which the field names cannot say: pending counts the
// insert-and-delete entries a replacement makes two of, and on_elements counts
// ids, so adding them is comparing two things.
// TestASuggestionTheListingCannotReportIsCounted and
// TestAnIDThePendingWalkSawIsNotCountedOnElementsAsWell are the pins.
//
// The last two counts are the ones worth writing down. A document holding an
// element gdoc has never seen holds something no count here speaks for, so the
// survey warns naming the member and refuses to say there is nothing to
// protect: TestAnElementTheReadCannotNameIsWarnedAbout. A named range is the
// other, and it is the reason the survey lists them at all: it is a label Docs
// keeps in step with its own edits, so a replacement of the words it covers
// takes it with them. Nothing compares that field before and after yet, and it
// does not need to: none of the four request kinds the in-place level carries
// can change a character, so no named range can move. A milestone that writes
// text compares it.
//
// A survey with no document read is not nothing to protect either. It reports
// the threads it has and warns that the chips, the pending suggestions and the
// named ranges are unknown. Answering true there would be the one false fact in
// the field a later run reads before it writes.
// TestASurveyWithNoDocumentSaysSoRatherThanReportingNothingToProtect is the pin.
//
// What a caller does over a true answer is the skill's, and it is not a flag
// here. The route for a document with nothing to protect is `read` it, put the
// text in a note with front matter, and `publish` that note. The second restyle
// mode that would have done it in one command is not built, and that is a
// decision rather than a deferral: DECISIONS.md 2026-09-11.
//
// # The apply loop, and what a failed restyle leaves behind
//
// Every batch carries writeControl.requiredRevisionId, and an empty one is
// refused at the call site. The survey rechecks the document before the run
// opens the grant, and a recheck followed by a write leaves a window that is
// the whole run across a dozen batches. requiredRevisionId closes it inside
// Docs: a document that moved refuses the batch itself, and unlike writeMode
// the field is in the public discovery document, so it is a rule the server
// holds rather than a statement of intent. A read that carried no revision id
// would ship an empty string, Docs would take the batch, and the only
// protection this run has would be gone with nothing said.
// TestEachBatchCarriesTheRevisionFromTheLastAnswer and
// TestAnEmptyRevisionIsRefusedBeforeAnythingIsSent are the pins.
//
// The loop reads nothing between batches. Each batch is sent against the
// revision the batch before it answered with, and an answer that names none
// ends the run: reading the document for a revision would read it as it is
// now, foreign edit included, so that edit would be adopted as this run's own
// and every batch behind it accepted against it. The run stops instead, the
// batches that landed stay, and the warnings say the document is half styled.
// The last batch is the one case with nothing behind it, so a quiet answer
// there is a warning and not a stop.
// TestAnAnswerCarryingNoRevisionMidRunStopsTheRun and
// TestTheLastAnswerCarryingNoRevisionIsAWarningAndNotARead are the pins.
//
// A refusal is never retried, and a stale revision is reported as itself. Docs
// refusing a batch on a moved revision means somebody edited the document after
// the survey, and retrying against a fresh revision would be gdoc styling a
// document being edited, which is the exact case the field exists to refuse.
// TestAStaleRevisionIsReportedAndNeverRetried is the pin.
//
// A batch Docs accepted whose answer could not be read is the third case, as it
// is for every other writer here. It may be in the document, so it is never
// sent again, and the run stops because the revision the next batch needs was
// in the answer it could not read. That case is MaybeApplied, a field rather
// than a batch counted in Batches: Batches is what Docs confirmed, folding the
// two together would lose which it was, and leaving the case out of the report
// was worse than either, because the run then said "no batch was applied, so
// the document is as it was" one warning after saying the batch may be in the
// document, and the caller skipped the read-back on a document that may have
// just been directly edited. MaybeRequests beside it is how many requests that
// batch held, which is what reached needs to say which of them left the
// machine. TestAnAnswerThatCouldNotBeReadStopsTheRun and
// TestAFirstBatchThatMayHaveLandedNeverSaysTheDocumentIsAsItWas are the pins.
//
// Batches are sized in bytes, under the guard's own peek. The guard refuses a
// batchUpdate it cannot read whole, so maxBatchBytes is half of what it peeks
// and batchEnvelope is a generous over-estimate of everything in a body that is
// not a request. Bytes rather than a count of requests, because two
// updateParagraphStyle requests are nothing like the same size. One request too
// large to send at all is refused naming its kind, because splitting one is not
// something this loop can do. TestBatchesAreSizedUnderTheGuardsPeek and
// TestOneRequestOverTheCeilingIsRefusedByName are the pins.
//
// A failed run has no rollback, and what it leaves behind is written down
// rather than discovered. A run that stops at batch twelve leaves a half-styled
// document, and the recovery is the document's own version history, by hand. No
// text was touched, so nothing the author wrote is lost, but their own run
// formatting inside the paragraphs that were restyled is. That sentence is in
// leftBehind, so it reaches the envelope's warnings on every path that stops
// early and the skill reads it out. noRollback holds it once, because every
// path that stops early says it and two copies would be two sentences that
// drift. On a run sent in suggested mode the recovery is a rejection in the
// browser rather than the version history, which is the one word that changes:
// TestASuggestedRunThatStoppedSaysToRejectRatherThanToUndo is the pin.
//
// "The document is as it was" is a claim, so it is kept for the two paths that
// can make it. A batch that could not be built never left the machine, and one
// Docs refused on a moved revision it refused whole. A batch that failed on the
// request itself is one of three things internal/gapi cannot tell apart, and
// its own doc comment names them: a guard refusal, where nothing left the
// machine; a 4xx, where Docs rejected the batch whole; and a 5xx or a dropped
// connection, where the request was written and may have been applied. So
// leftBehind takes that path as maybeReached and says the document is either as
// it was or part styled. TestAFailedRequestNeverSaysTheDocumentIsAsItWas and
// TestAStaleFirstBatchStillSaysTheDocumentIsAsItWas are the pins. The read-back
// gate is not widened to match, and cmdRestyle says why: three reads on every
// refusal would be paid on the common case to answer the rare one, and the
// sentence sends the caller to the document instead.
//
// # The survey and the apply are two runs
//
// A run naming both flags is refused, and so is a run naming neither.
// TestRestyleRefusesTheTwoHalvesTogether and TestRestyleRefusesNeitherFlag are
// the pins. The survey is what makes the write safe, so it has to be a thing a
// person read before the write was asked for, rather than something the same
// run produced a moment earlier and never showed anybody. readSurvey in
// cmd/gdoc reads that file the way internal/frontmatter reads a note, with
// DisallowUnknownFields and a refusal for a second JSON object behind the
// first: it is the only record of what the document held before the run, and
// the only thing standing between a direct-edit grant and a document nobody
// looked at. A survey that did not succeed is refused by name, because its zero
// fields are not facts.
//
// Four refusals happen before the in-place grant is opened, and the order in
// applyRestyle is deliberate: a survey of another document, a survey reporting
// more than one tab, a revisionId that has moved, and a document that has more
// than one tab when read fresh. TestRestyleRefusesASurveyOfAnotherDocument,
// TestRestyleRefusesADocumentThatMoved and TestRestyleRefusesASecondTab are the
// pins. The tab rule is asked twice because the survey's count and the read's
// count are two different moments, and a style request names a range, which
// means nothing without saying which tab it is in. The document is handed to
// the policy at the suggest level like every other handed-in id, and the grant
// is a separate line further down: TestRestyleGrantsTheOneDocumentInPlaceAndNothingElse.
//
// # What a restyle overwrites, and what it leaves alone
//
// The face, the size and the colour of every run go to the house value, so an
// author's own emphasis by size or colour is gone. Bold, italic, keep_with_next
// and keep_lines_together are never written, because absent and false are one
// word in house.yaml and writing them would clear an author's emphasis on the
// strength of a value the file may never have stated. A table cell's own fill
// is left alone for the mirror reason: which row of somebody's table is a
// header is not something gdoc can read.
// TestNothingWritesAFlagOrABorderTheHouseStyleDoesNotState is the pin. Not a
// character of the author's text moves, and the threads, the pending
// suggestions and the chips are what the read-back counts. Named ranges are
// surveyed and not compared, for the reason the survey section above gives.
// That the four request kinds preserve all of it was measured rather than
// reasoned about: MEASURED.md "In-place styling preserves anchors and pending
// suggestions", and which kinds land at all is MEASURED.md "Which styling
// requests land in place", which is where the scope of this package came from.
//
// A restyle is a moment, not a setting, and that sentence belongs in the
// report. updateNamedStyle does not exist, so the look is applied paragraph by
// paragraph. The document looks right afterwards and the next heading the
// author types is Google's Heading 1 again. Nothing in the binary can fix that,
// and the skill says it out loud rather than letting somebody discover it a
// week later.
//
// A table's cells are one request naming the table, not one per row.
// updateTableCellStyle takes either a tableRange or a tableStartLocation, and
// the reference documents the second as applying "to all the cells in the
// table". The row form was built from the row's own cell count, and that count
// is not the row's width: the reference says "It is possible for a table to be
// non-rectangular, so some rows may have a different number of cells", so a row
// whose first two columns are merged carries one cell object for the pair, and
// the span covered the merged cell alone and left the row's last column with
// the look it had. internal/docs decodes no columnSpan and no column count, so
// the real width is not something the builder could compute, and a request that
// names the table needs neither. cellAt in landing.go reads that one shape, and
// a tableRange is no answer rather than a second reading nothing sends.
// TestATablesCellsTakeThePaddingAndTheBorders and
// TestTheCellCheckReadsTheTableTheRequestNamed are the pins.
//
// # The read-back is two halves, and it runs on a failed run too
//
// Preserve compares the survey with a fresh survey: thread counts and
// per-thread witness, pending suggestion ids, chips. Landed reads the styling
// back out of the Docs answer and asks whether the requests that were sent are
// really there. Neither half can see the other's failure, which is why there
// are two, and TestVerifiedIsBothHalvesTogether is the pin.
//
// A run that wrote nothing reads nothing back, because the document is as it
// was. A run that stopped at batch twelve does, because that is the run the
// preservation facts are most needed for. A read the run could not make is a
// warning and no read-back at all, since a preservation half built from a
// listing that never arrived names every thread in the survey as gone.
// TestRestyleReadsNothingBackWhenNoBatchLanded and
// TestRestyleSaysWhenItCouldNotReadTheDocumentBack are the pins.
//
// Seven rules inside those halves are decisions rather than details.
//
// A witness that reads unmatched now is unwitnessed, never a lost anchor.
// Unmatched is the export giving no answer, and calling absence of evidence
// damage is the cry-wolf warning this tool avoids everywhere else. It keeps
// verified false all the same, because nothing then says the anchor survived.
// TestAWitnessThatStoppedAnsweringIsNotAnchorDamage is the pin.
//
// A witness that reads detached now, on a thread the survey could not witness,
// is now_detached, and it is neither of the two lists either side of it.
// Detached is the export answering, and its answer is that the text a comment
// was attached to has gone, so it is not the absence of evidence unwitnessed
// holds. What is missing is any answer about whether it was already detached
// before the run, so it is not lost_anchor either, whose warning says in its
// own words that the thread was anchored before. Verified is false over it. The
// case is not a corner: a survey whose own export failed reports every thread
// unmatched, which is exactly the before-picture this arrives from, and with no
// list of its own it left verified true on a run that destroyed an anchor.
// TestAThreadDetachedWithNoWitnessBeforeIsNeverCalledIntactlyWitnessed and
// TestVerifiedIsFalseWhenAThreadReadsDetachedWithNothingBefore are the pins.
//
// The survey carries suggestion ids and not only counts. Two counts that did
// not move cannot tell one suggestion destroyed and another created from
// nothing having happened, so SuggestionCounts carries the ids the read could
// see, from the listing's walk and from the elements both.
// TestAPendingSuggestionIsComparedByID is the pin.
//
// The landing check is made against the requests that were sent, never against
// house.yaml. A check written from the house style asks the question the
// builder already answers, and the two drift the first time a builder stops
// setting a field. It reads the first request of each kind and names where it
// looked, because a restyle sends one request per paragraph and hundreds of
// lookups answer one question: TestTheCheckReadsTheFirstRequestOfItsKind. A
// field the read does not carry at all is the document's own default, so a zero
// holds where the read is silent: Docs leaves a property equal to its default
// out of the answer, and reading that silence as a failure would report every
// zero the house style states as not landed.
// TestAZeroTheReadDoesNotCarryIsTheZeroBeingThere and
// TestANonZeroTheReadDoesNotCarryDidNotLand are the two directions of it.
//
// Sent means sent, so a run that stopped is given the prefix and not the plan.
// reached hands the read-back the requests that left the machine, which on a
// run that stopped at batch three is not the table request sitting in batch
// nine. Given the whole plan the check looked for a request that never left the
// machine and named the style as one that did not land, on a run whose warnings
// somebody is already reading to work out what state the document is in.
// TestAKindThatWasNeverSentIsNotChecked is the pin.
//
// Left the machine is two lists, and a batch Docs accepted is in the second
// one. Sent is Confirmed and Unconfirmed: the requests inside the batches Docs
// answered for, and the requests of the one batch Docs accepted whose answer
// could not be read, which Applied.MaybeRequests counts. Both are read back,
// because on that path reading the document is the only way anybody finds out
// whether the styling is there, and the run has already paid for the read. What
// an unconfirmed batch may never do is make the run verified, whatever its
// checks say: Verify forces verified false over a non-empty Unconfirmed, since
// the landing half reads the first request of each kind and a batch of hundreds
// can hold one that landed and hundreds that did not.
// TestAnUnconfirmedBatchIsCheckedAndNeverVerified is the pin.
//
// The page check reads the sections too, and it is the one case of a request
// accepted and invisible that this half was written for. updateDocumentStyle
// writes documentStyle, so on a document whose section break carries its own
// margins the request reads back exactly as it was sent while the page a reader
// sees never moved: comparing the two alone answered held on the only document
// the check exists for. sectionOverrides names the fields a section states for
// itself, and a field on that list makes the check no answer with the reason in
// it. flipPageOrientation is on that list and is matched by no name, because no
// request this level carries can set it: a section stating it differently from
// the document's own shows the page size that was sent transposed, which is the
// same accepted-and-invisible shape one field along. Two sections say nothing
// either way. A section that sets none of the fields the request set overrides
// nothing, because an unset section margin is the document's own, so the
// ordinary first section break of every document is not a warning; and one
// restating the value that was sent overrides nothing a reader could see, so
// the check answers. The second is the rule a warning on the working case is
// one people learn to ignore: a document gdoc built and published already
// carries the house geometry, and that is what the live acceptance restyles a
// copy of. The comparison is missingFields, the rule the checks themselves are
// made of, so one tolerance decides both.
// TestAPageMarginASectionOverridesIsNotReportedAsLanded,
// TestASectionThatFlipsTheOrientationIsNotReportedAsLanded,
// TestASectionBreakSettingNoneOfTheFieldsSentStillAnswers and
// TestASectionRestatingTheMarginThatWasSentStillAnswers are the pins.
//
// # Manual is what gdoc could not do, each with its menu path
//
// Three the API cannot do at all, the first-page header carrying the logo, the
// contents list and the footer page numbers, and two this package chose not to,
// the lists and the table column widths. A named style the house has no look
// for is the sixth entry, conditional like the other two.
// TestTheManualStepsNameTheMenuPathForEach and
// TestTheManualStepsAlsoNameWhatTheRunLeftAlone are the pins. gdoc reports the
// list and the skill reads it out, rather than writing it into the document as
// a finishing checklist: DECISIONS.md 2026-09-09.
//
// # Verified false is not a failure
//
// It is not one for propose or publish either. Verified is every landing check
// holding, the preservation half intact, and no thread whose witness stopped
// answering. What was applied is in the document either way, and a caller told
// the run failed is a caller that runs it again.
// TestRestyleSaysWhenTheStyleDidNotLand is the pin on the envelope, and the
// TestVerifiedIsFalseWhen... family in readback_test.go on each way it goes
// false.
package restyle
