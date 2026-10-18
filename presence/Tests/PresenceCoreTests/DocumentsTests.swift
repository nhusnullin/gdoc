// DOCUMENTS: the same record as TODAY, grouped by the document it touched.
//
// The rules under test are the M24 specification's product decision as the
// second morning changed it: every event grouped by document inside fourteen
// days of its last action, the pins first and then the day and then the
// fortnight by use, no sorting by state, a read alone making no row, five
// rows by default with Show more, and a label that counts the horizon and the
// day. A pin holds a document in the list past the horizon; a hide takes one
// out until gdoc finishes something on it again.
//
// The fixture fourteen-days.jsonl is one record that spans the horizon and a
// month boundary: two documents outside the horizon or read-only, and six
// that make a row. Every moment is a literal and the clock is injected, so
// nothing here reads this machine.

import Foundation
import XCTest

@testable import PresenceCore

final class DocumentsTests: XCTestCase {
  /// 17:20 in the fold's zone on 2026-10-17, which is the moment every
  /// fixture assertion below is made at.
  private func seventeenTwenty() throws -> Date { try moment("2026-10-17T15:20:00Z") }

  private func fixture() throws -> DocumentsFold {
    let scan = Event.scan(try Fixture.data("fourteen-days"))
    XCTAssertEqual(scan.skipped, 0, "the fixture is clean")
    XCTAssertFalse(scan.newerSchema)
    return Documents.fold(
      scan.events, now: try seventeenTwenty(), calendar: try amsterdam(),
      alive: { _ in false })
  }

  func testDocumentsGroupByIdInsideFourteenDaysNewestFirst() throws {
    let fold = try fixture()

    XCTAssertEqual(
      fold.rows.map(\.doc), [falcon, chapter5, policy, northwind, chapter4, concept],
      "newest first by last action, with no pinning and no sorting by state")
    XCTAssertEqual(
      fold.rows.map(\.at),
      [
        try moment("2026-10-17T12:00:00Z"), try moment("2026-10-17T11:02:30Z"),
        try moment("2026-10-17T10:20:30Z"), try moment("2026-10-17T09:10:30Z"),
        try moment("2026-10-17T08:00:00Z"), try moment("2026-10-05T10:03:00Z"),
      ])
    // The policy row is done and not held: its landed line carries the
    // hold's id and no document, which is the only shape gdoc writes, and a
    // hold that was answered waits for nobody.
    XCTAssertEqual(
      fold.rows.map(\.state), [.unknown, .liveStopped, .done, .done, .failed, .done])
    XCTAssertEqual(
      fold.rows.map(\.command),
      ["restyle", "comments", "reply", "annotate", "publish", "publish"])
    XCTAssertEqual(
      fold.rows.map(\.title),
      [
        "Project Falcon: the regions proposal",
        "Chapter 5. Commercial Model: the storyline",
        "Group Channel Policy: the frozen vocabulary",
        "Northwind and Altery: the screening quota",
        "Chapter 4. Pricing: the draft",
        "B2B2C payment card concept: the frozen vocabulary",
      ])

    XCTAssertEqual(
      fold.rows.filter { $0.doc == chapter5 }.count, 1,
      "one row for a document the record touched eleven times")
    XCTAssertFalse(
      fold.rows.contains { $0.doc == old },
      "the publish of 2026-09-30 is seventeen days back, so the document is gone")
    XCTAssertEqual(Documents.horizon, 14 * 24 * 60 * 60, "fourteen days, as a duration")
  }

  func testAReadAloneMakesNoRow() throws {
    let fold = try fixture()
    XCTAssertFalse(
      fold.rows.contains { $0.doc == readOnly }, "the record only ever read that document")

    let read = pair(at: "2026-10-17T10:00:00Z", command: "read", run: "aaaa", doc: docA)
    XCTAssertTrue(try folded(read).rows.isEmpty, "a read alone makes no row")

    let threads = pair(at: "2026-10-17T10:10:00Z", command: "comments", run: "bbbb", doc: docA)
    XCTAssertTrue(
      try folded(read + threads).rows.isEmpty,
      "a comments that did not wait is a read of the threads")

    // And a read after a write never moves the order, because the row is the
    // last action and a read is not one.
    let write = pair(at: "2026-10-17T11:00:00Z", command: "reply", run: "cccc", doc: docA)
    let later = pair(at: "2026-10-17T12:00:00Z", command: "read", run: "dddd", doc: docA)
    let rows = try folded(write + later).rows
    XCTAssertEqual(rows.count, 1)
    XCTAssertEqual(
      rows[0].at, try moment("2026-10-17T11:00:20Z"),
      "the moment the write ended, and not the read that came after it")
    XCTAssertEqual(rows[0].command, "reply")
  }

  func testTheLabelCountsTheHorizonAndToday() throws {
    let fold = try fixture()
    XCTAssertEqual(fold.count, 6, "the rows inside the horizon")
    XCTAssertEqual(fold.today, 5, "and how many of them were acted on in the local day")
    XCTAssertEqual(
      fold.rows.map(\.isToday), [true, true, true, true, true, false],
      "the concept was published on 2026-10-05, inside the horizon and not today")
  }

  func testTheDefaultIsFiveWithShowMore() throws {
    XCTAssertEqual(Documents.defaultRows, 5, "five rows, as the screens draw them")

    let fold = try fixture()
    XCTAssertEqual(fold.visible(expanded: false).count, 5)
    XCTAssertEqual(
      fold.visible(expanded: false).map(\.doc),
      [falcon, chapter5, policy, northwind, chapter4], "the newest five")
    XCTAssertEqual(fold.more, 1, "Show more names what is left")
    XCTAssertEqual(fold.visible(expanded: true), fold.rows, "Show less goes back to five")

    let short = pair(at: "2026-10-17T11:00:00Z", command: "reply", run: "aaaa", doc: docA)
    let one = try folded(short)
    XCTAssertEqual(one.more, 0, "no Show more under the default")
    XCTAssertEqual(one.visible(expanded: false), one.rows)
  }

  func testTheStateIsTheOpenActionThenTheLatestFinishedOne() throws {
    let done = pair(at: "2026-10-17T09:00:00Z", command: "reply", run: "aaaa", doc: docA)

    let running = [start(at: "2026-10-17T10:00:00Z", command: "publish", run: "bbbb", doc: docA)]
    XCTAssertEqual(
      try folded(done + running, alive: { _ in true }).rows.map(\.state), [.running],
      "a write that is still going wins over the reply that finished")
    XCTAssertEqual(
      try folded(done + running, alive: { _ in false }).rows.map(\.state), [.unknown],
      "and its process being gone is an outcome nobody wrote down")

    let watch = pair(
      at: "2026-10-17T10:30:00Z", command: "comments", run: "cccc", doc: docA, wait: true)
    XCTAssertEqual(
      try folded(done + watch).rows.map(\.state), [.liveStopped],
      "the last poll ended more than three minutes ago")
    XCTAssertEqual(
      try folded(done + watch).rows.map(\.command), ["comments"])

    let open = [
      start(
        at: "2026-10-17T15:19:00Z", command: "comments", run: "dddd", doc: docA, wait: true)
    ]
    XCTAssertEqual(
      try folded(done + open, alive: { _ in true }).rows.map(\.state), [.live],
      "a watch whose poll is in flight is live")

    let held = [
      start(at: "2026-10-17T11:00:00Z", command: "reply", run: "eeee", doc: docA),
      hold(at: "2026-10-17T11:00:10Z", run: "eeee", doc: docA, id: "h_1"),
    ]
    XCTAssertEqual(
      try folded(done + open + held, alive: { _ in true }).rows.map(\.state), [.hold],
      "a hold waits for a person, so it wins over everything else on that document")
  }

  func testAnOpenWatchIsOrderedByItsLastPollAndNotItsFirst() throws {
    // The screens file's r6 draws the live watch above a publish that
    // finished after the watch began, because a watch polling right now
    // acted a second ago: its first poll is where it started and not what
    // it last did. One session, because the second poll began inside the
    // three-minute gap, and it is in flight.
    let watch = [
      start(at: "2026-10-17T11:10:00Z", command: "comments", run: "aaaa", doc: docA, wait: true),
      end(at: "2026-10-17T11:11:00Z", command: "comments", run: "aaaa", doc: docA, wait: true),
      start(
        at: "2026-10-17T11:13:00Z", command: "comments", run: "aaaa", doc: docA, call: 2,
        wait: true),
    ]
    let published = pair(
      at: "2026-10-17T11:11:40Z", command: "publish", run: "bbbb", doc: docB)

    let fold = try folded(watch + published, alive: { _ in true })

    XCTAssertEqual(fold.rows.map(\.state), [.live, .done])
    XCTAssertEqual(
      fold.rows.map(\.doc), [docA, docB],
      "the watch polling now is above the publish that finished a minute ago")
    XCTAssertEqual(
      fold.rows.map(\.at),
      [try moment("2026-10-17T11:13:00Z"), try moment("2026-10-17T11:12:00Z")],
      "the watch's moment is its last poll, not its first")
  }

  /// The order: what gdoc did today, newest first, and then the fortnight
  /// by how much of it each document was.
  ///
  /// A table of seven documents: three acted on in the local day, and four
  /// older ones carrying one, two, two and three finished actions. The old
  /// rule put the four older ones in `docD, docF, docE, docG`, which is
  /// their last action newest first; the rule under test puts them in
  /// `docE, docF, docG, docD`, which is their count most first with the tie
  /// broken by the last action.
  func testTheOrderIsTodayNewestFirstThenTheFortnightByUse() throws {
    // doc, the moments its writes finished at, newest last.
    let table: [(doc: String, moments: [String])] = [
      (docA, ["2026-10-17T09:00:00Z"]),
      (docB, ["2026-10-17T11:00:00Z"]),
      (docC, ["2026-10-17T13:00:00Z"]),
      (docD, ["2026-10-16T10:00:00Z"]),
      (docE, ["2026-10-08T10:00:00Z", "2026-10-11T10:00:00Z", "2026-10-14T10:00:00Z"]),
      (docF, ["2026-10-09T10:00:00Z", "2026-10-15T10:00:00Z"]),
      (docG, ["2026-10-06T10:00:00Z", "2026-10-10T10:00:00Z"]),
    ]
    var events: [Event] = []
    for (index, entry) in table.enumerated() {
      for (call, at) in entry.moments.enumerated() {
        events += pair(
          at: at, command: "reply", run: "run\(index)\(call)", doc: entry.doc)
      }
    }

    let fold = try folded(events)

    XCTAssertEqual(
      fold.rows.map(\.doc), [docC, docB, docA, docE, docF, docG, docD],
      "today newest first, then the fortnight by its count of finished actions")
    XCTAssertEqual(
      fold.rows.map(\.isToday), [true, true, true, false, false, false, false])
    XCTAssertEqual(
      fold.rows.map(\.at),
      [
        try moment("2026-10-17T13:00:20Z"), try moment("2026-10-17T11:00:20Z"),
        try moment("2026-10-17T09:00:20Z"), try moment("2026-10-14T10:00:20Z"),
        try moment("2026-10-15T10:00:20Z"), try moment("2026-10-10T10:00:20Z"),
        try moment("2026-10-16T10:00:20Z"),
      ],
      "the row's moment is still its last action, so the moments are not in order")

    XCTAssertEqual(
      fold.visible(expanded: false).map(\.doc), [docC, docB, docA, docE, docF],
      "the five the section draws are the day and the two most used")
    XCTAssertEqual(fold.more, 2, "Show more names what is left, as it did")
    XCTAssertEqual(Documents.horizon, 14 * 24 * 60 * 60, "and the horizon is unchanged")
  }

  /// A document the fortnight listed by use draws exactly what it would
  /// draw alone: a count is a judgement, and the panel prints none.
  func testADocumentListedByUseShowsNothingNew() throws {
    let moments = ["2026-10-08T10:00:00Z", "2026-10-11T10:00:00Z", "2026-10-14T10:00:00Z"]
    var used: [Event] = []
    for (call, at) in moments.enumerated() {
      used += pair(at: at, command: "reply", run: "used\(call)", doc: docE, title: "The note")
    }
    let today = pair(at: "2026-10-17T13:00:00Z", command: "publish", run: "day0", doc: docC)

    let alone = try XCTUnwrap(try folded(used).rows.first)
    let listed = try XCTUnwrap(try folded(used + today).rows.first { $0.doc == docE })

    XCTAssertEqual(listed, alone, "the row by use is the row it always was")
    XCTAssertEqual(listed.command, "reply", "the last action, which is what the sub-line says")
    XCTAssertEqual(listed.at, try moment("2026-10-14T10:00:20Z"), "and its time")
    XCTAssertEqual(listed.state, .done)
  }

  /// A pin comes before every other row and never falls out of the list.
  ///
  /// Two pins: one on a document whose last action is sixty days back, which
  /// the horizon would have dropped, and one on a document the record has no
  /// line for at all, which draws the title and the moment the pin itself
  /// kept. The order between two pins is the order they were pinned in,
  /// newest pin first, and never what the record says about them.
  func testAPinnedDocumentIsFirstAndNeverFallsOutOfTheList() throws {
    let today = pair(at: "2026-10-17T13:00:00Z", command: "reply", run: "aaaa", doc: docA)
    let sixtyDays = pair(
      at: "2026-08-18T10:00:00Z", command: "publish", run: "bbbb", doc: docB,
      title: "Chapter 4. Pricing: the draft")
    let older = DocumentChoice(
      doc: docB, at: try moment("2026-08-18T10:30:00Z"),
      title: "Chapter 4. Pricing: the draft")
    let newer = DocumentChoice(
      doc: docC, at: try moment("2026-10-16T09:00:00Z"), title: "The regions proposal")

    XCTAssertEqual(
      try folded(today + sixtyDays).rows.map(\.doc), [docA],
      "without the pins the sixty-day-old document is past the horizon")

    let fold = try folded(today + sixtyDays, pinned: [older, newer])

    XCTAssertEqual(fold.rows.map(\.doc), [docC, docB, docA], "the pins first, newest pin first")
    XCTAssertEqual(fold.rows.map(\.isPinned), [true, true, false])
    XCTAssertEqual(fold.count, 3, "the label counts a pinned row like any other")
    XCTAssertEqual(fold.today, 1, "and a pin is not an action, so it is not today's")

    // The one the record still carries draws what the record says.
    XCTAssertEqual(fold.rows[1].state, .done)
    XCTAssertEqual(fold.rows[1].command, "publish")
    XCTAssertEqual(fold.rows[1].at, try moment("2026-08-18T10:00:20Z"))
    XCTAssertEqual(fold.rows[1].title, "Chapter 4. Pricing: the draft")

    // The one it has no line for draws the pin's own title and moment, and
    // says nothing about an action, because there is none to say.
    XCTAssertEqual(fold.rows[0].state, .pinnedOnly)
    XCTAssertEqual(fold.rows[0].command, "")
    XCTAssertEqual(fold.rows[0].at, try moment("2026-10-16T09:00:00Z"))
    XCTAssertEqual(fold.rows[0].title, "The regions proposal")
    XCTAssertFalse(fold.rows[0].isToday)
  }

  /// The pin's title fills a gap in the record and never speaks over it.
  ///
  /// The title a pin keeps is there for one reason, which this file's own
  /// comment states: a document the record cannot name still has its name on
  /// its row. So the record wins wherever it has a title, and the pin's title
  /// is the fallback, on every pinned row and not only on the stateless one.
  ///
  /// Two gaps, one each way. A pinned document whose only lines since are
  /// reads has no action, so its row is `.pinnedOnly`, and the read's own
  /// title is newer than the one the pin kept. A pinned document with a start
  /// and no end has an action and no title at all, because `presence.Begin`
  /// writes none, and without the fallback that row draws a Drive id at a
  /// person.
  func testAPinsTitleFillsAGapAndNeverSpeaksOverTheRecord() throws {
    let read = pair(
      at: "2026-10-17T12:00:00Z", command: "read", run: "dddd", doc: docD,
      title: "The regions proposal, final")
    let renamed = DocumentChoice(
      doc: docD, at: try moment("2026-10-16T09:00:00Z"), title: "The regions proposal")

    let newer = try folded(read, pinned: [renamed])
    XCTAssertEqual(newer.rows.map(\.doc), [docD], "a read is no action, so the pin is the row")
    XCTAssertEqual(newer.rows[0].state, .pinnedOnly)
    XCTAssertEqual(
      newer.rows[0].title, "The regions proposal, final",
      "the record named it after the pin was made, and the record is what it is called")

    let running = [start(at: "2026-10-17T15:00:00Z", command: "publish", run: "eeee", doc: docE)]
    let named = DocumentChoice(
      doc: docE, at: try moment("2026-10-17T09:00:00Z"), title: "Chapter 4. Pricing")

    let open = try folded(running, pinned: [named], alive: { _ in true })
    XCTAssertEqual(open.rows.map(\.doc), [docE])
    XCTAssertEqual(open.rows[0].state, .running, "a start whose process is there is running")
    XCTAssertEqual(
      open.rows[0].title, "Chapter 4. Pricing",
      "a start carries no title, so the pin's own title is the row's")
  }

  /// A hidden document leaves the list, and the next finished action on it
  /// is the one way back.
  func testAHiddenDocumentLeavesTheListUntilTheNextActionOnIt() throws {
    let kept = pair(at: "2026-10-17T09:00:00Z", command: "reply", run: "aaaa", doc: docA)
    var used: [Event] = []
    for (call, at) in ["2026-10-08T10:00:00Z", "2026-10-11T10:00:00Z"].enumerated() {
      used += pair(at: at, command: "reply", run: "used\(call)", doc: docB)
    }
    let hide = DocumentChoice(
      doc: docB, at: try moment("2026-10-17T11:00:00Z"), title: "Chapter 4")

    XCTAssertEqual(try folded(kept + used).rows.map(\.doc), [docA, docB], "both, unhidden")

    let after = try folded(kept + used, hidden: [hide])
    XCTAssertEqual(after.rows.map(\.doc), [docA], "the hidden document leaves the list")
    XCTAssertEqual(after.count, 1, "and the label counts what is left")

    let again = pair(at: "2026-10-17T12:00:00Z", command: "publish", run: "cccc", doc: docB)
    let back = try folded(kept + used + again, hidden: [hide])
    XCTAssertEqual(
      back.rows.map(\.doc), [docB, docA],
      "the next finished action gdoc did on it brings it back")
    XCTAssertEqual(back.count, 2)

    // A pin is the louder answer: a document that is both is in the list.
    let pin = DocumentChoice(doc: docB, at: try moment("2026-10-17T11:30:00Z"), title: nil)
    XCTAssertEqual(
      try folded(kept + used, pinned: [pin], hidden: [hide]).rows.map(\.doc), [docB, docA],
      "a pin over a hide is the document kept")
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
  private let docB = "1BbbAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let docC = "1CccAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let docD = "1DddAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let docE = "1EeeAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let docF = "1FffAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let docG = "1GggAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"

  private func folded(
    _ events: [Event], pinned: [DocumentChoice] = [], hidden: [DocumentChoice] = [],
    alive: @escaping (Int32) -> Bool = { _ in false }
  ) throws -> DocumentsFold {
    Documents.fold(
      events, now: try seventeenTwenty(), calendar: try amsterdam(), pinned: pinned,
      hidden: hidden, alive: alive)
  }
}
