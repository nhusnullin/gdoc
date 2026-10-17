// TODAY and Yesterday: the log of what gdoc did, cut by the local day.
//
// The rules under test are the M24 specification's product decision: TODAY is
// the local day newest first, eight rows by default, with a synthetic row for
// a hold raised, a hold closed, a live watch started and a live watch
// stopped with its totals; reads are left out; Yesterday is one summary that
// opens inline; nothing older appears.
//
// The day is the calendar's, never the record file's: the record is one file
// per month, so yesterday is in another file on the first of a month, and
// that is what testAMonthBoundaryIsNotADayBoundary holds.

import Foundation
import XCTest

@testable import PresenceCore

final class DaysTests: XCTestCase {
  /// 17:20 in the fold's zone on 2026-10-17.
  private func seventeenTwenty() throws -> Date { try moment("2026-10-17T15:20:00Z") }

  private func fixture(now: Date? = nil) throws -> DaysFold {
    let scan = Event.scan(try Fixture.data("fourteen-days"))
    XCTAssertEqual(scan.skipped, 0, "the fixture is clean")
    let at = try now ?? seventeenTwenty()
    return Days.fold(
      scan.events, now: at, calendar: try amsterdam(), alive: { _ in false })
  }

  func testTodayIsTheLocalDayNewestFirstDefaultEight() throws {
    XCTAssertEqual(Days.defaultRows, 8, "eight rows, as the screens draw them")

    let fold = try fixture()
    XCTAssertEqual(
      fold.today.map(\.kind),
      [
        .outcomeUnknown, .liveStopped, .action, .liveStarted, .action, .holdClosed, .holdRaised,
        .action, .action, .action,
      ],
      "newest first, whatever the row is")
    XCTAssertEqual(
      fold.today.map(\.at),
      [
        try moment("2026-10-17T12:00:00Z"), try moment("2026-10-17T11:02:30Z"),
        try moment("2026-10-17T11:02:00Z"), try moment("2026-10-17T11:00:00Z"),
        try moment("2026-10-17T10:20:30Z"), try moment("2026-10-17T10:20:00Z"),
        try moment("2026-10-17T10:00:00Z"), try moment("2026-10-17T09:40:20Z"),
        try moment("2026-10-17T09:10:30Z"), try moment("2026-10-17T08:00:00Z"),
      ])

    XCTAssertEqual(fold.visible(expanded: false).count, 8)
    XCTAssertEqual(fold.more, 2, "Show more names what is left")
    XCTAssertEqual(fold.visible(expanded: true), fold.today)

    // The day is the one the calendar gives, so the same event is today in
    // one zone and yesterday in another.
    let late = pair(at: "2026-10-16T23:30:00Z", command: "reply", run: "aaaa", doc: docA)
    var utc = Calendar(identifier: .gregorian)
    utc.timeZone = try XCTUnwrap(TimeZone(identifier: "UTC"))

    let amsterdamFold = Days.fold(
      late, now: try seventeenTwenty(), calendar: try amsterdam(), alive: { _ in false })
    XCTAssertEqual(amsterdamFold.today.count, 1, "01:30 on 2026-10-17 in Amsterdam")
    XCTAssertTrue(amsterdamFold.yesterday.isEmpty)

    let utcFold = Days.fold(
      late, now: try seventeenTwenty(), calendar: utc, alive: { _ in false })
    XCTAssertTrue(utcFold.today.isEmpty, "23:30 on 2026-10-16 in UTC")
    XCTAssertEqual(utcFold.yesterday.count, 1)
  }

  func testTodayCarriesTheSyntheticRows() throws {
    let fold = try fixture()

    let raised = try XCTUnwrap(fold.today.first { $0.kind == .holdRaised })
    XCTAssertEqual(raised.at, try moment("2026-10-17T10:00:00Z"))
    XCTAssertEqual(raised.command, "reply")
    XCTAssertEqual(raised.doc, policy)
    XCTAssertEqual(raised.title, "Group Channel Policy: the frozen vocabulary")
    XCTAssertEqual(raised.rule, "dictated")
    XCTAssertEqual(raised.via, "chat")
    // How the hold ended, on the row that raised it: the row is read in the
    // present tense until it has one, and this hold's card went at 10:20
    // because the write it held reached the document another way.
    XCTAssertEqual(raised.ending, .landed)

    // The ending line carries the hold's id and nothing else, so everything
    // a person reads on this row comes off the held line it closes. A row
    // built from the ending alone would say only "Hold closed" and would
    // leave the document out of the day's count.
    let closed = try XCTUnwrap(fold.today.first { $0.kind == .holdClosed })
    XCTAssertEqual(closed.at, try moment("2026-10-17T10:20:00Z"), "the ending's own moment")
    XCTAssertEqual(closed.ending, .landed, "the record's own word for it")
    XCTAssertEqual(closed.doc, policy)
    XCTAssertEqual(closed.command, "reply")
    XCTAssertEqual(closed.via, "chat")
    XCTAssertEqual(closed.rule, "dictated")
    XCTAssertEqual(closed.title, "Group Channel Policy: the frozen vocabulary")

    let started = try XCTUnwrap(fold.today.first { $0.kind == .liveStarted })
    XCTAssertEqual(started.at, try moment("2026-10-17T11:00:00Z"))
    XCTAssertEqual(started.doc, chapter5)
    XCTAssertEqual(started.title, "Chapter 5. Commercial Model: the storyline")

    let stopped = try XCTUnwrap(fold.today.first { $0.kind == .liveStopped })
    XCTAssertEqual(stopped.at, try moment("2026-10-17T11:02:30Z"))
    let totals = try XCTUnwrap(stopped.totals, "a stopped watch carries its own totals")
    XCTAssertEqual(totals.since, try moment("2026-10-17T11:00:00Z"))
    XCTAssertEqual(totals.ended, try moment("2026-10-17T11:02:30Z"))
    XCTAssertEqual(totals.ran, 150, "two polls and the second over in two and a half minutes")
    XCTAssertEqual(totals.replied, 0)
    XCTAssertEqual(totals.proposed, 1, "the one proposal written inside the session")

    // A watch that is still going has a started row and no stopped row.
    let open = [
      start(at: "2026-10-17T15:19:00Z", command: "comments", run: "zzzz", doc: docA, wait: true)
    ]
    let live = Days.fold(
      open, now: try seventeenTwenty(), calendar: try amsterdam(), alive: { _ in true })
    XCTAssertEqual(live.today.map(\.kind), [.liveStarted], "nothing says it stopped")

    // An end is an action row with the facts its line carried.
    let action = try XCTUnwrap(
      fold.today.first { $0.kind == .action && $0.command == "publish" })
    XCTAssertEqual(action.at, try moment("2026-10-17T08:00:00Z"))
    XCTAssertEqual(action.ok, false)
    XCTAssertNil(action.verified, "a publish that stopped read nothing back")
    XCTAssertEqual(action.ms, 610_000)
    XCTAssertEqual(action.doc, chapter4)

    // A write whose process is gone is the row the stuck card's dismissal
    // leaves behind: an outcome nobody wrote down.
    let unknown = try XCTUnwrap(fold.today.first { $0.kind == .outcomeUnknown })
    XCTAssertEqual(unknown.at, try moment("2026-10-17T12:00:00Z"))
    XCTAssertEqual(unknown.command, "restyle")
    XCTAssertEqual(unknown.doc, falcon)
    XCTAssertNil(unknown.ok, "nothing said it worked and nothing said it failed")
  }

  /// Scene r1's row: a publish that stopped is named in TODAY out of the one
  /// line of its own that carried a name, which is a step. It is handed no
  /// document, so the document's own title names nothing here.
  ///
  /// `presence.Begin` writes no title on a start and `End` lifts one only
  /// out of a read-back, so the lines that can carry the cover's name are
  /// the steps after the render and the end of a publish that worked: a
  /// publish that stopped inside the render has no name anywhere, and the
  /// row says "Publish" alone.
  func testAStoppedPublishIsNamedInItsRow() throws {
    let fold = Days.fold(
      [
        start(at: "2026-10-17T09:03:10Z", command: "publish", run: "aaaa", doc: nil),
        step(at: "2026-10-17T09:03:50Z", name: "upload", index: 2, run: "aaaa", title: "Chapter 4"),
        end(
          at: "2026-10-17T09:05:00Z", command: "publish", run: "aaaa", doc: nil, ok: false,
          stop: "network"),
      ], now: try seventeenTwenty(), calendar: try amsterdam(), alive: { _ in false })

    XCTAssertEqual(fold.today.map(\.kind), [.action])
    XCTAssertEqual(fold.today.first?.title, "Chapter 4", "the name its step carried")
    XCTAssertEqual(fold.today.first?.stop, "network")

    // Scene r2's run: the render refused a picture, so the only lines are a
    // start and an end and neither of them names the note.
    let render = Days.fold(
      [
        start(at: "2026-10-17T09:03:10Z", command: "publish", run: "bbbb", doc: nil),
        end(
          at: "2026-10-17T09:05:00Z", command: "publish", run: "bbbb", doc: nil, ok: false,
          stop: "svg_picture", line: 41),
      ], now: try seventeenTwenty(), calendar: try amsterdam(), alive: { _ in false })

    XCTAssertNil(render.today.first?.title, "no line of that run carried a name")
    XCTAssertEqual(render.today.first?.line, 41)
  }

  /// Scene r7: the hold the person approved and the write that approval sent
  /// are one thing that happened, so TODAY draws one row for them, and that
  /// row says the person approved it rather than where it came from.
  ///
  /// `released` is the line that says they pressed the card. `mcprelease.go`
  /// writes it and then sends, so the write's own start and end follow it,
  /// and the record never holds an approval after the write it approved.
  func testAnApprovedHoldIsTheRowOfTheWriteItApproved() throws {
    let raised = hold(at: "2026-10-17T14:20:00Z", run: "chat", doc: policy, id: "h_1")
    let approved = holdEnd(
      at: "2026-10-17T14:40:55Z", kind: .released, run: "chat", doc: nil, id: "h_1")
    let wrote = [
      start(
        at: "2026-10-17T14:40:55Z", command: "reply", run: "chat", doc: policy, call: 2,
        via: "chat"),
      end(
        at: "2026-10-17T14:41:00Z", command: "reply", run: "chat", doc: policy, call: 2,
        via: "chat", ok: true, verified: true, title: "Group Channel Policy"),
    ]

    let fold = Days.fold(
      [raised, approved] + wrote, now: try seventeenTwenty(), calendar: try amsterdam(),
      alive: { _ in false })
    XCTAssertEqual(
      fold.today.map(\.kind), [.action, .holdRaised],
      "one row for the write, and the hold it waited under")

    let row = try XCTUnwrap(fold.today.first)
    XCTAssertEqual(row.ending, .released, "the person pressed the card in the chat")
    XCTAssertEqual(row.verified, true, "and the write read itself back")
    XCTAssertEqual(row.at, try moment("2026-10-17T14:41:00Z"))

    // An approval whose write is in no part of the record this app read
    // joins nothing, and its closing line stays a row of its own.
    let alone = Days.fold(
      [raised, approved], now: try seventeenTwenty(), calendar: try amsterdam(),
      alive: { _ in false })
    XCTAssertEqual(alone.today.map(\.kind), [.holdClosed, .holdRaised])
    XCTAssertEqual(alone.today.first?.ending, .released)
    XCTAssertEqual(alone.today.last?.ending, .released, "the raised row knows it is over")

    // A hold still waiting carries no ending, which is the only state the
    // row says "waiting in Desktop chat" in.
    let waiting = Days.fold(
      [raised], now: try seventeenTwenty(), calendar: try amsterdam(), alive: { _ in false })
    XCTAssertEqual(waiting.today.map(\.kind), [.holdRaised])
    XCTAssertNil(waiting.today.first?.ending)
  }

  /// A release whose send wrote nothing claims no later write.
  ///
  /// `mcprelease.go` writes the released line before it sends, and the send
  /// can come back before any line of its own: the pin reads the document
  /// and fails on the wire, or the title drifted. A reply the model sent
  /// half an hour later with no card is not the write that approval sent, so
  /// the approval's own closing line stays a row and says the card was
  /// pressed.
  func testAnApprovalThatSentNothingClaimsNoLaterWrite() throws {
    let raised = hold(at: "2026-10-17T14:20:00Z", run: "chat", doc: policy, id: "h_1")
    let approved = holdEnd(
      at: "2026-10-17T14:40:55Z", kind: .released, run: "chat", doc: nil, id: "h_1")
    let later = [
      start(
        at: "2026-10-17T15:10:00Z", command: "reply", run: "chat", doc: policy, call: 9,
        via: "chat"),
      end(
        at: "2026-10-17T15:10:04Z", command: "reply", run: "chat", doc: policy, call: 9,
        via: "chat", ok: true, verified: true, title: "Group Channel Policy"),
    ]

    let fold = Days.fold(
      [raised, approved] + later, now: try seventeenTwenty(), calendar: try amsterdam(),
      alive: { _ in false })
    XCTAssertEqual(
      fold.today.map(\.kind), [.action, .holdClosed, .holdRaised],
      "the reply, the card that was pressed, and the hold it was raised for")
    XCTAssertNil(
      fold.today.first?.ending, "nobody approved this reply, so its row says nothing about one")
    XCTAssertEqual(
      fold.today.first(where: { $0.kind == .holdClosed })?.ending, .released,
      "and the approval is still a row of its own")

    // Two approvals where the first sent nothing: the write belongs to the
    // second, and the first is not absorbed into it.
    let again = hold(at: "2026-10-17T15:00:00Z", run: "chat", doc: policy, id: "h_2")
    let approvedAgain = holdEnd(
      at: "2026-10-17T15:09:59Z", kind: .released, run: "chat", doc: nil, id: "h_2")
    let both = Days.fold(
      [raised, approved, again, approvedAgain] + later, now: try seventeenTwenty(),
      calendar: try amsterdam(), alive: { _ in false })
    XCTAssertEqual(
      both.today.map(\.kind), [.action, .holdRaised, .holdClosed, .holdRaised],
      "one approval went with the write and the other kept its own row")
    XCTAssertEqual(
      both.today.first?.ending, .released, "the write the second card sent says so")
  }

  /// `landed` is not an approval. `mcphold.go` writes it for a held write
  /// the model sent again with no card at all, so the write is a row of its
  /// own and the closing line is a second row saying the card went with it.
  /// Nothing on either row says a person answered anything.
  func testAHoldThatLandedIsTwoRowsAndNamesNoApproval() throws {
    let raised = hold(at: "2026-10-17T14:20:00Z", run: "chat", doc: policy, id: "h_1")
    let wrote = [
      start(
        at: "2026-10-17T14:40:55Z", command: "reply", run: "chat", doc: policy, call: 2,
        via: "chat"),
      end(
        at: "2026-10-17T14:41:00Z", command: "reply", run: "chat", doc: policy, call: 2,
        via: "chat", ok: true, verified: true, title: "Group Channel Policy"),
    ]
    let landed = holdEnd(
      at: "2026-10-17T14:41:01Z", kind: .landed, run: "chat", doc: nil, id: "h_1")

    let fold = Days.fold(
      [raised] + wrote + [landed], now: try seventeenTwenty(), calendar: try amsterdam(),
      alive: { _ in false })

    XCTAssertEqual(
      fold.today.map(\.kind), [.holdClosed, .action, .holdRaised],
      "the write is its own row and the card's going is another")
    XCTAssertEqual(fold.today.first?.ending, .landed)
    XCTAssertNil(
      fold.today.first(where: { $0.kind == .action })?.ending,
      "nobody approved the write, so its row carries no ending")
  }

  /// The card in NOW and the row in TODAY name one write with one rule,
  /// which is `Fold.title` and not a copy of it in each file: a publish
  /// named only by a step line is named by it in both places, and a run no
  /// line named is nameless in both.
  func testTheCardAndTheRowNameAWriteTheSameWay() throws {
    let events = [
      start(at: "2026-10-17T12:00:00Z", command: "publish", run: "aaaa", doc: nil),
      step(at: "2026-10-17T12:00:30Z", name: "upload", index: 2, run: "aaaa", title: "Chapter 4"),
    ]
    let at = try seventeenTwenty()

    let card = Now.fold(events, now: at, alive: { _ in false })
    let row = Days.fold(events, now: at, calendar: try amsterdam(), alive: { _ in false })

    XCTAssertEqual(card.stuck.map(\.title), ["Chapter 4"], "the card is named from the step")
    XCTAssertEqual(
      row.today.map(\.title), ["Chapter 4"], "and the row the card leaves behind the same way")
  }

  /// The row and the card are one run, so they are not both on the screen.
  /// NOW decides which it is: the horizon and the dismissals are that
  /// fold's, so it is folded first and hands over the keys of the cards it
  /// is keeping. What is left here is the row a card leaves behind when it
  /// clears or is dismissed.
  func testAWriteNowIsCardingIsNotAlsoARowInToday() throws {
    let lost = start(at: "2026-10-17T12:00:00Z", command: "restyle", run: "aaaa", doc: docA)
    let at = try seventeenTwenty()

    let both = Days.fold(
      [lost], now: at, calendar: try amsterdam(), alive: { _ in false })
    XCTAssertEqual(
      both.today.map(\.kind), [.outcomeUnknown],
      "with no card for it the row is the only thing that says the run happened")

    let carded = Days.fold(
      [lost], now: at, calendar: try amsterdam(), carded: ["aaaa|1"], alive: { _ in false })
    XCTAssertTrue(carded.today.isEmpty, "NOW is holding the card, so TODAY writes nothing")

    // The key is the record's own pairing, so another run's card hides no
    // row of this one.
    let other = Days.fold(
      [lost], now: at, calendar: try amsterdam(), carded: ["bbbb|1"], alive: { _ in false })
    XCTAssertEqual(other.today.map(\.kind), [.outcomeUnknown])
  }

  func testReadsAreLeftOutOfToday() throws {
    let fold = try fixture()
    XCTAssertFalse(
      fold.today.contains { $0.command == "read" }, "a read is not something gdoc did to a document"
    )
    XCTAssertFalse(
      fold.today.contains { $0.doc == readOnly }, "and the document it read makes no row either")
    XCTAssertFalse(
      fold.today.contains { $0.kind == .action && $0.command == "comments" },
      "a poll of the threads is a read; the session's own two rows stand for the watch")

    let threads = pair(at: "2026-10-17T10:00:00Z", command: "comments", run: "aaaa", doc: docA)
    let folded = Days.fold(
      threads, now: try seventeenTwenty(), calendar: try amsterdam(), alive: { _ in false })
    XCTAssertTrue(folded.today.isEmpty, "a comments that did not wait is a read")

    let lost = [start(at: "2026-10-17T10:00:00Z", command: "read", run: "bbbb", doc: docA)]
    let gone = Days.fold(
      lost, now: try seventeenTwenty(), calendar: try amsterdam(), alive: { _ in false })
    XCTAssertTrue(gone.today.isEmpty, "a read whose process is gone wrote nothing to report")
  }

  func testYesterdayIsOneSummaryThatOpensInline() throws {
    let fold = try fixture()

    XCTAssertEqual(
      fold.yesterday.map(\.kind), [.liveStopped, .liveStarted, .action, .action],
      "the same rows as TODAY, newest first, behind one chevron")
    XCTAssertEqual(
      fold.yesterday.map(\.at),
      [
        try moment("2026-10-16T16:05:00Z"), try moment("2026-10-16T16:00:00Z"),
        try moment("2026-10-16T15:00:20Z"), try moment("2026-10-16T14:00:30Z"),
      ])
    XCTAssertEqual(fold.yesterdaySummary.actions, 4, "the count the chevron row carries")
    XCTAssertEqual(
      fold.yesterdaySummary.documents, 3, "and how many documents those rows touched")
    XCTAssertEqual(
      Set(fold.yesterday.compactMap(\.doc)), [chapter5, policy, northwind],
      "the rows are there to open, so the chevron opens them in place")

    let quiet = try fixture(now: try moment("2026-10-05T15:20:00Z"))
    XCTAssertEqual(
      quiet.yesterdaySummary, DaySummary(actions: 0, documents: 0), "a day with nothing in it")
  }

  func testNothingOlderAppears() throws {
    let fold = try fixture()
    let rows = fold.today + fold.yesterday

    XCTAssertFalse(
      rows.contains { $0.doc == old }, "the publish of 2026-09-30 is in no section of the panel")
    XCTAssertFalse(
      rows.contains { $0.doc == concept },
      "nor the publish of 2026-10-05, which DOCUMENTS still carries")
    let midnight = try moment("2026-10-16T00:00:00Z")
    XCTAssertTrue(
      rows.allSatisfy { $0.at >= midnight },
      "two days, and history is a question for Claude Code or the hub")
    XCTAssertEqual(rows.count, 14, "ten rows today and four yesterday, and nothing else")
  }

  func testAMonthBoundaryIsNotADayBoundary() throws {
    // The record is one file per month, so on the first of a month yesterday
    // is the other file. The day comes from the calendar either way.
    let fold = try fixture(now: try moment("2026-10-01T10:00:00Z"))

    XCTAssertTrue(fold.today.isEmpty, "nothing happened on 2026-10-01")
    XCTAssertEqual(fold.yesterday.map(\.kind), [.action], "the publish of 2026-09-30")
    XCTAssertEqual(fold.yesterday.map(\.doc), [old])
    XCTAssertEqual(fold.yesterday.map(\.at), [try moment("2026-09-30T08:02:00Z")])
    XCTAssertEqual(fold.yesterdaySummary, DaySummary(actions: 1, documents: 1))

    // And a day inside one month is still one day: the publishes of
    // 2026-10-05 are two hours apart on the same date.
    let inside = try fixture(now: try moment("2026-10-06T10:00:00Z"))
    XCTAssertEqual(inside.yesterday.count, 2, "the export and the publish of 2026-10-05")
    XCTAssertEqual(inside.yesterdaySummary, DaySummary(actions: 2, documents: 2))
  }

  // MARK: - The pieces a test states

  private let chapter5 = "1Ch5AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let policy = "1PolAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let chapter4 = "1Ch4AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let northwind = "1NorAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let falcon = "1FalAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let concept = "1B2bAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let old = "1OldAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let readOnly = "1RedAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let docA = "1AaaAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
}
