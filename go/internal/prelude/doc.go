// Package prelude is the house template as Docs API requests: the cover, the
// three front-matter tables and the legend, proposed into a document that
// already exists.
//
// It is the second half of `restyle <url> --from survey.json --fields
// fields.json`. The first half is internal/restyle, which gives the body the
// house look where it stands. This half adds the template in front of it, and
// every character of it is a suggestion: Nail accepts it in the browser the way
// he accepts any suggestion, or rejects it and the document is as it was.
//
// Nothing here reaches the network, reads a file or decides anything. Propose
// and FrontMatter are pure functions of the house style and the cover's values,
// Decide and Markers read a document that was handed to them, and Verify reads
// one back. What the requests did is sent and read by cmd/gdoc.
//
// Nothing here is a verdict either. Verify reports counts and three flags, and
// whether the cover reads right is read in the document by the person being
// asked to accept it.
//
// This comment holds why the package is shaped the way it is. What it reports
// is in the code beside it.
//
// # Two phases, two permissions, and neither can do the other's job
//
// Phase 1 proposes the prelude: insertText, insertTable, insertPageBreak,
// deleteContentRange, updateParagraphStyle, updateTextStyle and
// updateTableCellStyle, in writeMode SUGGEST, on a document that was handed in
// and granted nothing. Two of the four that touch text, insertText and
// deleteContentRange, are exactly the shape internal/propose has sent every day
// since it existed, and insertTable, insertPageBreak and the three styling
// kinds ride the same SUGGEST batch, so the phase needed no new guard
// permission for any of it, and TestThePreludeNeedsNoGrantAtAll in
// internal/guard says that rather than a comment.
//
// Phase 2 styles the body: the four styling kinds internal/restyle builds,
// direct, on the same document granted LevelInPlace.
//
// They are two policies and two sessions rather than one policy that does both,
// because the guard's own rule is right and stays. At LevelInPlace the
// request-kind allowlist gates every batchUpdate whatever writeMode says, so a
// granted document cannot take an insertText and an ungranted one cannot take a
// direct edit. proposeThenStyle in cmd/gdoc builds the second policy and the
// second session itself, since a session is built from a policy and the first
// request in its history is already judged against it.
// TestThePreludePhaseIsSentOnAPolicyThatGrantsNothing hands each recorded batch
// body to the policy it really went out on and asks the two refusals as well as
// the two carries.
//
// # The prelude phase is unprobed, and the read-back is the only bar
//
// internal/propose runs internal/probe before every proposal, because writeMode
// is a field gdoc supplies, it is absent from the public discovery document, and
// one morning the same call came back 200 having made a direct edit:
// BLOCKED-BY-API.md holds both measurements. Phase 1 makes that same claim, on a
// whole cover rather than one word, and asks the question of no throwaway
// document first. internal/guard's paragraph about there being nothing for a
// probe to test is true of LevelInPlace and is not true of this phase.
//
// The reason is that a probe needs a folder to create its document in, through
// AllowCreateIn, and restyle takes no folder. Giving it one is a flag, a second
// create door on a command whose whole shape is that it writes to the one
// document it was handed, and that is Nail's decision rather than a refactor.
//
// What stands in its place is Verify's first field. Every piece of the prelude
// is read back and asked whether it carries a suggestion id, and a piece that
// does not is counted as written, which the read-back section below calls the
// failure rather than a difference. So an unenrolled morning is caught after the
// fact and never before it, and the recovery is the document's own version
// history.
//
// # AllowMarker is this milestone's one new door
//
// It has AllowReject's shape: per-run, one object, dying with the process,
// nothing writing it down and no flag turning it on. It opens createNamedRange
// at LevelInPlace alone, for exactly the range that was granted, spelled
// exactly, with nothing beside name and range. A second call replaces the first,
// because a caller naming two ranges has made a mistake the guard must not turn
// into two markers, and a grant the guard cannot read opens nothing, says so on
// the envelope, and takes back the grant standing before it rather than leaving
// that one live. TestTheMarkerIsRefusedWithoutItsOwnGrant,
// TestAGrantedMarkerCarriesAndNothingElseDoes, TestASecondAllowMarkerReplacesTheFirst,
// TestAnUnusableMarkerGrantOpensNothing and TestARefusedSecondMarkerGrantTakesTheFirstBack
// in internal/guard are the pins, and TestNothingAtLevelInPlaceCanChangeACharacter
// stays green and untouched: a named range adds and removes no character.
// MarkerRequest is the one request this package builds for it, and
// TestTheMarkerRequestIsWhatTheGuardGrants states that it is the shape the grant
// carries.
//
// The marker is written rather than suggested because createNamedRange is the
// one request Docs refuses to apply as a suggestion, in its own words. The probe
// sent one request kind per case and had nine of ten recorded as suggestions,
// and its ten-row table is in DECISIONS.md, "The house template reaches a
// document as a suggestion, not as a direct edit". MEASURED.md "A named range
// over a suggested insertion" is what says a named range over a pending
// insertion covers exactly the proposed line, survives the accept with its id
// and range intact, and vanishes with the reject.
//
// createParagraphBullets is the difference between the two levels, and it is one
// rule read twice. At LevelInPlace it is refused, because the reference says the
// leading tabs that set a bullet's nesting level are removed by the request, so
// on somebody's own paragraph it deletes text they typed. Here it may be sent,
// because it would land on text gdoc itself proposed a moment earlier, where
// there are no author tabs to remove, and the probe measured it accepted as a
// suggestion. Nothing sends it today: the house legend carries no bullets, its
// lines are a bold word, a tab and a sentence. The reasoning is written down
// beside the statement that nothing sends it, so the next person to need one has
// the answer rather than the question.
//
// MarkerName is gdoc's whole memory of having been here. A restyle writes no
// file beside the document and has no note to pair, so the document is the
// record. It is read by id and never by name, because docs.NamedRange is keyed
// by id and two ranges may wear one name: a run acting by name would act on
// both, so two ranges wearing this name are refused rather than guessed between.
// TestTwoRangesWearingTheMarkerNameAreRefusedByID is the pin.
//
// # Decide answers three shapes, and the third is a refusal
//
// No marker is a first run, which proposes at index 1. It is also a document
// whose prelude was rejected, and that is the same shape rather than a second
// one: the rejection took the marker with it.
// TestAFirstRunProposesAtTheTopAndDeletesNothing and
// TestAFirstRunDeletesNothingAtAll are the pins.
//
// A marker over settled text is replaced: a deleteContentRange in SUGGEST mode
// over the marked span, and then the fresh prelude at that span's own start.
// That is the order and the shape internal/propose sends a replacement in. A
// suggested delete marks text rather than removing it, so nothing behind it
// moves and every index the front matter computed is still the index it named.
// TestASecondRunReplacesTheMarkedPreludeRatherThanAddingASecond and
// TestASecondRunProposesTheDeletionOfTheOldPreludeFirst are the pins.
//
// A marker over text that is still pending is refused, naming the suggestion ids
// and saying to accept or reject in the browser first. Replacing it would
// propose deleting text that has never been written.
// TestASecondRunRefusesWhileTheFirstPreludeIsStillPending,
// TestAPreludePendingDeletionIsRefusedTheSameWay and
// TestAPendingSuggestionInsideAPreludeTableIsFound are the pins.
//
// The pending question is asked before the count, and that is the third shape's
// own rule rather than a detail of it. Two markers is exactly what a replace run
// leaves while its suggestions are unsettled: the new one over the prelude that
// was proposed, the old one over the prelude proposed for deletion, because
// nothing deletes a named range. Asked the other way round, a third run over
// that ordinary state fell into the ambiguity refusal and told somebody to
// remove a named range by hand, which the Docs UI gives no way to do and
// deleteNamedRange is on no allowlist for. So a marker carrying a pending
// suggestion is the pending refusal however many markers there are, and the
// ambiguity refusal is kept for two markers over settled text, which is a
// document gdoc cannot guess between.
// TestAReplaceRunsTwoPendingMarkersAskForTheBrowserRatherThanAHandEdit is the
// pin, and readback.go's marked says the same thing from the other side, that
// more than one marker is not itself a fault.
//
// Nothing deletes a named range and nothing needs to. The marker tracks its
// text, so the old one goes when the deletion under it is accepted and comes
// back when that deletion is rejected, and either way one marker is left. That
// last step is an inference from the measurement rather than a fifth measured
// row, and Decide's own comment says so in those words.
//
// Pending is read over the marked span, in both suggestion lists and inside the
// table cells, because the front matter is three tables and a walk reading
// paragraphs alone would call a wholly proposed prelude settled. A marker this
// package cannot read as one contiguous span of one tab's body is refused rather
// than reported: a span in a header, a footer or a footnote, and two spans with
// the author's own words between them. Two spans that touch are one span,
// because Docs may cut a range at a boundary of its own and the text is still
// gdoc's. TestAMarkerBrokenAcrossTheAuthorsTextIsRefused,
// TestAMarkerInASegmentIsRefused and TestAMarkerCutIntoTwoTouchingSpansIsOnePrelude
// are the pins.
//
// # The marker goes out in a batch of its own, ahead of the styling
//
// A styling batch that does not land still leaves a prelude Nail can accept, and
// a marked prelude is one the next run can find. Folded into the styling it
// would be lost with it. An unmarked prelude is one the next run proposes a
// second cover in front of, so a marker that does not land stops the run and the
// warning says to accept or reject before running again.
// TestAnUnmarkedPreludeSaysSoOnEveryPathThatLeavesOne in cmd/gdoc is the pin.
//
// A marker batch Docs accepted whose answer could not be read is
// marker_maybe_created, not a marker that did not land. That is the rule every
// other writer here holds, said at the one field that had missed it: reported
// from the batch count alone the run said the marker did not land one line under
// a warning saying that batch may be in the document.
// TestAMarkerBatchThatMayHaveLandedSaysSo is the pin.
//
// Being its own batch is also what made it the last batch, and the revision has
// to be read for. restyle.Apply warns and stops chaining when the final batch
// answers with no revision id, leaving its RevisionID at the revision that batch
// was sent against, which was safe while nothing followed the last batch. Here
// the styling phase follows it, and sent that revision the first styling batch
// is refused as stale and reported as somebody having edited the document after
// the survey: a third party named for a revision gdoc itself moved. So
// restyle.Applied.RevisionUnconfirmed says which of the two RevisionID is, and
// cmd/gdoc reads it and makes the narrowed restyle.RevisionOf read on that one
// rare path, dropping restyle.RevisionUnconfirmedWarning by value when the read
// answers. Left on the envelope that sentence stands beside a revision Docs
// named, and two sentences contradicting each other in one warnings list is
// worse than either of them.
// TestThePreludeBatchNamingNoRevisionIsAnsweredByTheFreshRead and
// TestTheStylingIsSentAgainstARevisionTheMarkerBatchDidNotName are the pins.
//
// The marker batch is sent against phase 1's own answer, and that is what keeps
// the revision chained across the phase boundary. Every batch requires the
// revision the one before it ended on, and the fresh read between the phases is
// the one place that chain could be broken: sent the revision that read named,
// an edit somebody made in the window between phase 1's last answer and it is
// carried rather than refused, so the marker lands on their revision, the
// styling chains off the marker, and phase 2 rewrites their paragraph by direct
// edit at LevelInPlace. Docs makes the refusal rather than gdoc comparing two
// strings: that a batch answer's revision is one a later write accepts is
// measured in every multi-batch run, while whether it is spelled the way
// documents.get spells it is not, and a run refused on a comparison gdoc made
// itself could cry wolf on every document.
// TestTheMarkerIsSentAgainstTheRevisionThePreludePhaseEndedOn is the pin.
//
// A prelude phase that stopped says to accept or reject before running again,
// the same sentence the marker failure says, because it is the same hazard by
// another route: what landed is unmarked. It is read from the batches Docs
// confirmed and from the batch Docs may have taken, since the live prelude goes
// out in one batch, so the path where Docs accepted it and the answer could not
// be read is the path where a whole unmarked prelude may be in the document
// while the run counts no batch at all.
// TestAPreludePhaseThatStoppedSaysToRejectBeforeRunningAgain is the pin.
//
// # Phase 2 walks past the span phase 1 wrote
//
// Without that the milestone defeats itself. Every paragraph proposed here is a
// NORMAL_TEXT stating the cover's own sizes and colours in full, because
// inserted text takes the look of the text it lands beside. A styling phase
// reading the document after phase 1 finds those paragraphs and gives each of
// them the house body look, so the 26pt cover title Nail is being asked to
// accept is 11pt prose by the time he reads it. restyle.TabRequestsExcept takes
// the span and reports what it left alone as planned.skipped; TabRequests is
// that function with no span, so a styling-only run is unchanged. The overlap is
// read rather than containment: a block half gdoc's words and half the author's
// is one no request can name without writing over one of them.
//
// On a replace run that span is two preludes, not one, and the marker's is still
// one. Result.Occupies is the arithmetic and its own comment is the reason. The
// deletion goes out in SUGGEST mode, so the prelude the run before this one left
// is still real text: the insert at Start pushes it along by the length of the
// new prelude and it comes to rest immediately behind the words that replace it.
// Given the marker's span alone, phase 2 walked past the new prelude and then
// gave the old one the house body look by direct edit at LevelInPlace, which
// flattens a cover Nail may yet reject the deletion of and makes the run's own
// recovery sentence false. Nothing downstream could have named it: internal/restyle
// reads no suggestion id, and Verify's body check compares characters. So the
// caller carries two spans on purpose, and they must not be folded: the replaced
// prelude carries a deletion id rather than an insertion one, so asked about the
// wider span the read-back would report gdoc's own replaced words as text
// somebody wrote. TestAFirstRunOccupiesTheSpanItProposed and
// TestAReplaceRunOccupiesTheOldPreludeToo here, with
// TestAReplaceRunStylesNeitherPrelude in cmd/gdoc, are the pins.
//
// # gdoc's own words state their look in full
//
// That is the opposite of internal/restyle's rule, and for the opposite reason.
// A restyle writes onto the author's text, where a flag house.yaml never stated
// would clear emphasis somebody meant. These are gdoc's own lines and nobody
// else's emphasis can be in them, so every paragraph states its named style, its
// alignment, its spacing and its indents, and every run states its face, its
// size, its weight, its slope, its underline and both colours.
// TestEveryBareParagraphMarkGdocProposesStatesItsOwnLook holds that no
// paragraph mark goes out without a text style over it, and
// TestTheCoverLinesAreCentredAtTheHouseSizes reads the stated fields themselves.
//
// The one look an inserted paragraph inherits and this package cannot state away
// is a list marker: taking one off needs deleteParagraphBullets, which nothing
// here sends, so a prelude proposed at the top of a document whose first
// paragraph is a list item arrives bulleted. That is
// docs/backlog/prelude-inherits-a-list-marker.md.
//
// One layout, two writers. internal/render writes this template into a docx and
// this package writes it as requests, both from house.Config, and neither holds
// a layout of its own. A value added to house.yaml has to reach both. What they
// share lives where its value lives: cover.Fields.Placeholder for what a
// placeholder name means, house.Cell.FillFor for the classification shading, and
// internal/docsreq for a measurement, a colour, an alignment, a length in the
// units the API counts, and a style object built with the mask that names it.
// internal/restyle was moved onto docsreq in the same commit, so the line-spacing
// rounding and the colour conversion have one copy between the two request
// builders.
//
// # The index arithmetic is computed here and never read back
//
// So it is pinned as literals, and it is measured rather than reasoned about.
//
// A page break is one request and two index units. insertPageBreak inserts a
// page break followed by a newline, in the reference's own words, so the break
// takes one unit and the newline it brings takes another. An insertText writing
// that newline as well is what used to stand beside it, and it put three units
// in the document where the builder counted two: the third was a stray empty
// paragraph that every later insert pushed along until it sat one past the
// prelude's end, outside the marker, outside the span phase 2 walks past, left
// behind by a second run's deleteContentRange and joined by another on the
// third. Nothing in the suite could see it, because every check here is the
// builder's arithmetic asked about itself and the live acceptance reads only
// inside the span. It is read out of the reference rather than measured, and the
// live acceptance is what confirms it.
//
// There are two of them, and house.yaml states where both go: a page_break
// block after the cover, and another after the classification table, so the
// version control heading and the contents heading each start a page.
// TestTheCoverEndsWithAPageBreak and TestTheFrontMatterEndsWithAPageBreak are
// the pins, and both read the breaks out of FrontMatter, because no writer here
// holds a page turn of its own: TestTheCoverBlockWritesNoPageBreakOfItsOwn says
// the cover writer does not.
//
// Both breaks are inside the span the marker covers, so the read-back counts
// them. A break carries no words, so it reaches the read as a run of its own
// kind beside the newline behind it, which is two of the runs Proposed counts
// per break. TestThePreludeReadsBackAsSuggestions states those counts as
// literals.
//
// A table is inserted and then filled, which is more requests than the docx
// writer needs and is the shape the Docs API has: insertTable makes a grid of
// empty cells, so every word in it is an insertText afterwards and every fill,
// padding and border an updateTableCellStyle. An empty table is one unit for the
// newline insertTable writes in front of it, one for the table, one per row, one
// per cell plus one for that cell's own paragraph mark, and one for the table's
// own end, and the paragraph behind a table begins at the table's own endIndex.
// MEASURED.md "A table takes one index of its own at the end" is where those
// numbers are, with the live refusal that found the missing unit and the sweep
// that read every accepted index back rather than believing the status code.
// TestATablesIndexesFollowTheDocsAccounting and
// TestTheIndexBehindATableIsTheTablesOwnEnd state it here.
//
// One request per cell rather than one per table, which is the opposite of
// internal/restyle's rule and for the opposite reason again. A restyle gives
// every cell of somebody's table one look and cannot read that table's real
// width, so it names the table. Here the fills differ cell by cell, because a
// classification row is shaded only when the fields declare that class, and this
// writer built the grid itself so it knows exactly how wide it is.
// TestTheClassificationTableShadesOnlyTheDeclaredClass and
// TestACellStatesItsPaddingAndItsBordersFromTheHouseFile are the pins.
//
// # The fields file is where the thirteen values come from
//
// A restyle has no note, so cover.Fields arrives as JSON from a file the skill
// proposes and Nail confirms, read by internal/cover with DisallowUnknownFields
// and a refusal for a second object behind the first, the way cmd/gdoc reads a
// survey and a proposals list. It is read before a session is opened, because a
// file gdoc half understands must never reach a document. The keys are the
// note's own keys, and the two normalisations and the two defaults are shared
// rather than copied, so a file stating no version publishes as 1.0 and one
// stating no date as this month, exactly as a note does.
// TestTheFieldsFileNamesTheSameThirteenValuesTheNoteDoes in internal/cover
// states that one note and one fields file naming the same values read to the
// same Fields.
//
// A value carrying a character Docs strips out of an insert is refused, naming
// the key, and the reason it is refused at the door is this package's. Every
// index here is computed from the length of the string it is about to send, so a
// unit Docs drops puts every later insert one place out. The loud outcome is
// Docs refusing the whole batch for an index inside no paragraph, which is
// exactly what the missing table unit did. The quiet one is worse: where the
// wrong index is still valid the prelude ends one short, the marker is written
// one character into the author's own text, and the next run proposes deleting a
// character they wrote. TestAValueCarryingACharacterDocsStripsIsRefusedByKey and
// TestEveryValueInTheFileIsAskedForStrippedCharacters are the pins, with
// TestATabInAValueIsCarried for the one control character that is deliberately
// outside the set, which is why the house legend can write one.
//
// Nothing infers a title. A missing one is cover.MissingTitle, the same error
// type a note's reader raises, so the skill reads one shape. It carries an empty
// Candidate from a fields file and that is honest rather than a gap: a candidate
// is drawn from a note's first heading or its file name, and a fields file has
// neither. TestAFieldsFileWithNoTitleIsRefusedAsAMissingTitle and
// TestANoteWithNoTitleStillSaysFrontMatter are the pins.
//
// # Verify asks three questions, and none can see the other two's failure
//
// Whether every piece of the prelude carries a suggestion id, whether the marker
// is over the span that was proposed, and whether the author's own text is
// character for character what it was. A prelude wholly written rather than
// proposed would pass the second; one wholly proposed and unmarked would pass
// the first; a run that took a paragraph of somebody's prose with it would pass
// both. So they are three fields, and verified is all of them together with the
// read having found the prelude at all. TestThePreludeReadsBackAsSuggestions,
// TestAPreludeWithNoMarkerOverItIsNotVerified and
// TestTheAuthorsOwnTextIsCheckedBeforeAgainstAfter are the pins, one each.
//
// Four rules inside those questions are decisions rather than details.
//
// The classification unit is the run, and the counts are paragraphs, tables and
// cells. Pieces counts what Result counts, so what was sent and what came back
// sit beside each other with nothing to convert. A paragraph, a table or a cell
// is proposed when every run of it inside the span carries an insertion id, and
// written when one of them does not. Written is the failure rather than a
// difference: those are characters in somebody's document on gdoc's own
// authority. TestARunWithNoSuggestionIDIsWrittenRatherThanProposed is the pin.
//
// Proposed is a count of suggestions, never a verdict. It is not compared with
// what was sent either, because Docs numbers an empty paragraph differently from
// the request that made it and a mismatch there would name the wrong problem. A
// field named looks_right, complete or ready here is the defect the facts rule
// names, and TestNothingInThePreludeReadBackIsAVerdict is the pin.
//
// Both sides of the body check are read by one rule. AuthorText drops every text
// run carrying a suggested insertion id and keeps every run carrying a suggested
// deletion id: an insertion is nobody's text yet, whoever proposed it, and a
// suggested deletion has removed nothing. The before side is taken off the
// document phase 1 was computed from, which is the last moment it can be taken,
// because every read after that one carries the prelude.
// TestAuthorTextDropsInsertionsAndKeepsDeletions is the pin.
//
// A run that proposed a prelude reads back whatever phase 2 did. The gate a
// styling-only run uses is the styling batches alone, and on a two-phase run
// that answered "nothing was written, so the document is as it was" about a
// document phase 1 had just put a cover into. TestAStylingRunReadsNoPreludeBack
// and TestTheReportCarriesBothPhases in cmd/gdoc are the pins either side of it.
//
// verified false is not a failure, as it is not for propose, publish or a
// styling run. What was proposed is in the document either way, and a caller
// told the run failed is a caller that runs it again.
//
// # A failed phase 1 does not run phase 2
//
// The report says which phase stopped. What the document carries is whatever of
// the prelude Docs took, and every character of that is a suggestion, so
// rejecting it puts the document back. A failed phase 2 after a successful phase
// 1 leaves a proposed prelude and an unstyled body, which is a document Nail can
// still act on. restyle.Suggest is restyle.Apply with one field,
// writeControl.writeMode, and what that field changes is the recovery a run that
// stopped early prints: a suggested batch is rejected in the browser, not undone
// through the version history. requiredRevisionId and writeMode together are
// unmeasured, MEASURED.md holds no section for the pair, and the live acceptance
// is what confirms them; Docs refusing the pair fails the prelude phase whole,
// which is the direction to be wrong in. TestAFailedPreludePhaseDoesNotStyle is
// the pin.
//
// # Manual, and what is out of this milestone
//
// Manual is what the house file states and no request here sends, each with a
// menu path: every column width and every row height of the front-matter tables.
// updateTableColumnProperties and updateTableRowStyle were not among the kinds
// DECISIONS.md, "The house template reaches a document as a suggestion, not as
// a direct edit", records the probe accepting as suggestions, and a request
// Docs refuses takes the whole batch with it. The contents list is there too, and it is blocked rather than
// deferred: no Docs request makes one, which is BLOCKED-BY-API.md's.
// TestTheContentsListIsAManualStep is the pin. The list is a fact and not a
// verdict: it says what is left to do, never whether the document is finished.
//
// Heading numbering is out, and not because of the guard. It is insertText,
// which the probe measured as suggestible, so it can be proposed and its own
// milestone starts from suggested mode. What keeps it out is idempotence and
// placement. Nothing marks a number gdoc wrote, and numberedHeadingRE in
// internal/body does not recognise the house's own format, because the separator
// is a hyphen and the pattern wants a dot, a bracket or a space, so a second run
// makes "1-1-Introduction". Placement is the second: the prelude is one
// insertion at index 1, while a number names a position inside the author's
// prose, which is what internal/propose's "names text, never an index" rule
// exists to refuse.
//
// # The live acceptance
//
// TestLivePreludeIsProposedNotWritten in internal/live is the one that runs all
// of this against Google. It copies a document it was named, proposes the
// prelude into the copy on a policy that granted nothing, sends the marker once
// before the grant to assert the guard refuses it, then grants and sends it
// again, and reads the whole thing back through Verify. It leaves the copy
// behind with its URL in the log, because whether the cover reads right is
// Nail's, in the document.
package prelude
