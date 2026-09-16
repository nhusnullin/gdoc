// Package comments is Drive's threads joined to the Docs ranges, the --since
// cursor, and the --wait poll.
//
// Two sources, because neither one carries the whole answer. Drive's
// comments.list is documented and stable and carries the replies, the authors,
// resolved and modifiedTime, but its anchor is an opaque string. The Docs read
// carries the character range each thread sits on, keyed by the same comment
// id, and carries nothing about the replies. So this package joins them, and
// reports the id of a thread the Docs read did not place rather than failing
// the whole listing over one anchor. TestThreadsJoinsTheRangeDriveDoesNotCarry
// and TestAnUnusableRangeLeavesTheThreadUnplaced are the pins, and the rule
// that decides whether a range names a position is docs.Document.Places, which
// internal/view warns from too, so the two commands cannot drift apart.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # Facts, never verdicts
//
// Nothing here decides whether a thread is answered. SPEC.md was corrected on
// 2026-09-06 for exactly this: "a 🤖 reply newer than the comment" is a
// heuristic, and a wrong one when the reply was an acknowledgment or the
// follow-up changed the question. So a Thread carries every reply, its author,
// its time and whether it opens with 🤖, and the skill reading the JSON decides
// what still needs an answer. There is no Handled field, and there must not be.
// The fields are a marker (ai:, ai?, ai! or none), a resolved flag and a
// reply's by_gdoc, each a fact with a neutral name.
// TestThreadsCarriesEveryFactAndJudgesNone is the pin, with
// TestMarkerIsTheFirstTokenAndTheMatchIsExact and
// TestByGdocIsTrueOnlyForAReplyOpeningWithTheRobot over the two facts most
// easily read loosely.
//
// Waited is that same rule under the wait. Every field on it is a count, a
// duration, a list or a flag: Polls, Waited, Threads, Unplaced, Cursor and
// Interrupted. It says how many times the binary asked, how long it looked,
// what arrived and whether a signal ended it. Whether a window is work,
// whether a window that is only gdoc's own replies is worth a turn, and when
// to stop looking are the skill's, exactly as they are for a one-shot listing.
// A field named news, idle, stale or should_retry there is the same defect.
//
// # A poll reads the listing first and the document second
//
// Both reads are needed either way, so the order is free, and what it decides
// is which comments the Docs read can place. A comment written in the gap
// between them is in whichever read ran after it. Listed first, the next poll
// places it. Read first, it is a comment in the listing with no anchor in a
// document read a moment before it existed, so Threads reports it unplaced, the
// cursor moves past it, and the skill is told a thread it could have proposed
// into cannot be. A wait ends on exactly the poll that first sees a new
// comment, which is the poll the race is live on.
//
// The order is the same everywhere a poll is built: cmdComments' closure,
// internal/restyle's survey, and internal/live's poller.
// TestThePollReadsTheListingBeforeTheDocument pins cmdComments' own closure and
// TestRestyleListsTheCommentsBeforeItReadsTheDocument pins the survey, both in
// cmd/gdoc, so a later edit cannot swap them in either.
//
// # The cursor is opaque, and it dies with the session
//
// base64url(JSON{"v":1,"t":"<RFC3339 UTC>","i":["<comment id>"]}), holding the
// newest modifiedTime seen across the comments and their replies, and the ids
// of the threads whose own newest instant was that one. The binary emits it,
// the caller hands it back on the next poll, and nothing writes it anywhere: a
// live session holds it in memory and it dies with the session. What must
// survive a session lives in the front matter, and this does not.
//
// It is opaque on purpose. A caller that decodes the instant and does
// arithmetic on it has made the encoding a contract, and it is not one. The
// version field is there so a later shape is refused by name, and a cursor that
// cannot be decoded is an error naming the problem, never silently read as
// "from the beginning": that would report a window nobody asked for and look
// like a clean poll. TestParseCursorRefusesWhatItCannotRead is the pin, with
// TestCursorRoundTripsThroughItsString, TestCursorIsBase64URLWithoutPadding,
// TestCursorStringIsUTCWhateverZoneItWasGiven and TestPaddedBase64IsRead over
// the encoding itself.
//
// Three parts make one thread stop being news, and dropping any of them makes
// it news on every poll for ever.
//
// The instant keeps its milliseconds, because that is what Drive sends. Rounded
// down to the second, the floor sits up to 999 ms below the activity the run
// just reported, Drive returns that thread again, and the next cursor rounds
// down to the same second. TestCursorKeepsTheMillisecondsDriveSent is the pin.
//
// Cursor.narrow is the second part, and the precision alone does not do its
// job. startModifiedTime is documented as the minimum value of modifiedTime, so
// Drive's bound is inclusive: handed the instant of the newest thread the last
// run saw, it sends that thread back, and NextCursor cannot advance past an
// instant it already holds. The request stays inclusive on purpose, because
// asking for a window a millisecond later would tell Drive to withhold a
// comment modified inside the cursor's own millisecond, and losing a comment is
// the wrong direction to be wrong in. Fetch narrows the answer instead: a
// comment is kept when its own modifiedTime or any reply's createdTime is
// strictly newer than the cursor. The replies are in that rule for the reason
// NextCursor reads them, and an instant gdoc cannot parse keeps its comment.
// TestFetchDropsTheThreadItsCursorAlreadyReported,
// TestFetchKeepsAThreadWhoseReplyIsNewerThanTheCursor,
// TestFetchKeepsACommentWhoseTimeItCannotRead and
// TestFetchWithNoCursorNarrowsNothing are the pins.
//
// The ids are the third part, and without them "strictly newer" loses a
// comment. Two comments can share a millisecond with only one of them
// reported, when the poll landed between the two writes. The second one then
// sits exactly on the cursor's instant, has never been seen, and no comparison
// on instants can say so: Drive's precision cannot tell the two apart. So the
// cursor carries the ids it reported at its own instant, and a comment on that
// instant is news unless the cursor names it. NextCursor adds to those ids
// while the instant does not move, and replaces them when it does. Adding is
// what makes the poll go quiet: a thread narrow dropped is a thread reported on
// an earlier poll, and forgetting its id makes the two threads sharing that
// millisecond take turns being news for ever. What is left is a comment edited
// twice inside one millisecond, which is Drive's precision rather than a choice
// made here. TestCursorCarriesTheIdsItReportedAtItsInstant,
// TestFetchKeepsACommentSharingTheCursorsMillisecondItNeverReported and
// TestTheCursorAfterASharedMillisecondReportsNeitherCommentAgain are the pins.
//
// A cursor written before the ids existed still reads, and the version stays 1
// for that reason: the ids only ever narrow further, so their absence costs one
// repeated thread and never a lost one. TestACursorWithNoIdsStillReads is the
// pin.
//
// # A listing always prints a cursor, and the empty one is dated from the clock
//
// NextCursor answers nil when nothing it saw carried an instant, which is every
// document nobody has commented on yet. That is the honest answer to "what is
// the newest activity here" and it is useless as a starting point: a wait needs
// a cursor, so a live session could not start on the document it is most often
// started on. cmd/gdoc's startCursor dates that case from the run's clock, less
// cursorFloor. Backwards rather than forwards, because a clock a little ahead
// of Drive's would tell the next poll to withhold a comment written in the
// meantime, and losing a comment is the wrong direction to be wrong in. It is
// the same argument Cursor.narrow makes at the boundary.
// TestNextCursorWithNothingAtAllIsNothing and
// TestAnEmptyListingStillHandsBackACursorAWaitCanUse are the two halves, and
// TestNextCursorFallsBackToThePreviousOne,
// TestNextCursorPicksTheNewestAcrossCommentsAndReplies and
// TestNextCursorStepsOverATimeItCannotRead cover the rest of NextCursor.
//
// The floor is five minutes, and what it pays for is clock skew. The instant is
// read on this machine and compared at Drive, which applies it as
// startModifiedTime and withholds everything older, so what the floor has to
// cover is the offset between the two clocks. A machine five minutes fast would
// otherwise put the baseline cursor five minutes into Drive's future and lose
// every comment written in that window for good, because the cursor never moves
// backwards. Being early costs nothing against that: this is only reached when
// the listing carried no readable instant at all. The clock is read before the
// two reads rather than after them, and the caller passes the instant in for
// that reason: read afterwards, the reads spend the floor, and a comment written
// while a slow baseline was being read falls into a window no poll asks for
// again. TestTheBaselineCursorIsDatedFromBeforeTheReadsRatherThanAfterThem in
// cmd/gdoc is the pin.
//
// # The wait is one call that polls, and the loop is the skill's
//
// comments --since CURSOR --wait 9m is the same listing, made repeatedly inside
// one call, and it is still one JSON object and still exit 0 if and only if ok.
// Wait holds the loop and takes the poll as a Poll closure, so this package
// still holds no session and no URL: the command builds the two reads it
// already builds, and a test hands in a script.
//
// The waiting happens here rather than in the skill for one measured reason: a
// Claude Code session cannot wake more often than once a minute, and every idle
// tick would spend a model turn. So a quiet document costs nothing, and a
// comment is seen within one interval of being written. Nail's decision,
// 2026-09-07.
//
// A wait needs a cursor. A wait with no cursor answers with the whole document,
// which is the one-shot listing under another name and reads to a session as
// news, so cmd/gdoc refuses it naming the missing flag. The duration is a Go
// duration, and zero, negative, unreadable or over an hour is refused naming
// the value. TestAWaitWithoutACursorIsRefusedNamingTheCursor and
// TestAWaitRefusesADurationItCannotUse are the pins, and Wait itself refuses a
// missing poll, a non-positive interval and a negative deadline
// (TestWaitRefusesOptionsItCannotRun).
//
// The interval is ten seconds, a constant, and never a flag. It sits in the
// spec's five to fifteen. cmd/gdoc's waitInterval is a package variable only so
// a test can shorten it. The first poll happens at once rather than after an
// interval, because latency is the point of a live session: a wait that slept
// first would cost the interval on every call the skill makes, quiet document
// or not. The last gap is slept out rather than polled on, because a poll
// started on the instant the deadline lands on runs on a context that is
// already done, so it sends nothing and Polls would count a request that never
// left the process. TestWaitPollsAtOnceRatherThanSleepingFirst,
// TestWaitDoesNotSleepPastItsDeadline and
// TestTheWaitDoesNotPollOnTheDeadlineItHasAlreadyRunOutOf are the pins.
//
// The first non-empty window ends the wait. A window is what Fetch narrowed and
// Threads joined, so gdoc's own 🤖 reply arriving after a poll does end it: the
// binary reports it and the skill reads it as a receipt.
// TestWaitReturnsTheFirstWindowWithActivityAndAdvancesTheCursor and
// TestAWindowThatIsOnlyAGdocReplyIsStillReturned are the pins, with
// TestWaitCarriesTheIdsTheDocsReadCouldNotPlace over the unplaced list.
//
// The deadline bounds the call, not just the gaps between polls. Wait derives a
// second context with the deadline on it and polls on that. Without it the only
// bound on one poll is the client's own timeout, which is minutes and applies
// per request, so a poll that stalled at 8m50s carried a nine minute call past
// the ten minutes the nine was chosen to fit inside, and the harness then killed
// it with a signal the answer reports as interrupted. The two contexts stay
// apart on purpose: the caller's being done is the person, and the derived one
// being done is the deadline, which is an empty window and not a failure.
// TestTheDeadlineBoundsTheCallAndNotOnlyTheGapsBetweenPolls is the pin.
//
// Every request inside a poll carries that context, the token refresh included.
// gapi.Session's refresh takes the caller's context and auth.Token.Refresh
// builds its POST with it. Sent without one, the refresh was the single request
// in a poll that neither the deadline nor a Ctrl-C could reach: its only bound
// was the guard's five minute client timeout, which is exactly the overrun the
// derived context exists to stop.
//
// A --witness export after a wait is part of the call, so cmd/gdoc runs it on
// what is left of the deadline. Left unbounded it has only the client timeout,
// five minutes on top of the wait, so --wait 9m --witness could answer at
// fourteen minutes and be killed by the harness that waits ten, which prints
// nothing at all. An export cut short by the remainder is a warning naming it,
// on an envelope that still carries the threads.
// TestTheWitnessAfterAWaitIsBoundedByWhatIsLeftOfTheDeadline and
// TestAWitnessAppliesToTheWindowThatEndedTheWait are the pins.
//
// A failed poll is ok: false, carrying the error and the polls so far. Nothing
// is retried in silence: the skill sees the failure, says the window is unread
// rather than empty, keeps the cursor it had and calls again. A caller told the
// document was quiet would believe it.
// TestWaitCarriesAFailedPollOutWithThePollsSoFar and
// TestAFailedPollEndsTheWaitAndSaysHowManyItMade are the pins, and
// TestAPollThatFailedOfItsOwnAccordPastTheDeadlineIsStillAFailure is the arm
// that keeps a Drive failure at the tail of the window from reading as a quiet
// document.
//
// An interrupt is an answer. A deadline reached and an interrupt both come back
// with no threads and the cursor handed in unchanged, so the next call asks the
// same question and nothing is lost, and a poll that failed because the
// interrupt cut the request short is reported as the interrupt rather than as a
// failed read. TestAnInterruptEndsTheWaitAsAnAnswer,
// TestWaitAtItsDeadlineIsEmptyWithTheCursorItWasGiven,
// TestAPollThatFailedBecauseOfTheInterruptIsReportedAsTheInterrupt,
// TestAnInterruptDuringAStalledPollIsStillAnInterrupt,
// TestAContextAlreadyCancelledPollsNothing and
// TestTheDefaultSleeperEndsEarlyOnAnInterrupt are the pins. Which process
// traps the signal is cmd/gdoc's rule, and its doc comment holds it.
//
// Waited is absent without the flag. Reporting one poll on a call that never
// waited says the binary polls when it does not. An empty window is not
// witnessed either: the export would be one more request for no question, and
// an interrupted wait is that same case rather than a second one, because it
// comes back with no threads. TestAListingWithoutTheFlagCarriesNoWaitedObject
// and TestAnEmptyWindowIsNotWitnessed are the pins.
//
// Nothing is kept. The wait writes no file of its own and keeps no session
// state, and the cursor it prints is the only thing that carries to the next
// call. The one file a poll can replace is the saved OAuth token, which the
// session refreshes as it does for every other command: that is the credential,
// not anything the wait learned. The loop belongs to the review skill, which
// asks for nine minutes because the tool that runs the command gives up at ten,
// and which sets that tool's own timeout to ten minutes: its default is two,
// and a wait cut short at two is either an unmatched timeout or a signal the
// answer reports as interrupted.
package comments
