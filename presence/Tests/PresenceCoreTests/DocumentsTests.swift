// DOCUMENTS: the same record as TODAY, grouped by the document it touched.
//
// The rules under test are the M24 specification's product decision: every
// event grouped by document inside fourteen days of its last action, newest
// first, no pinning and no sorting by state, a read alone making no row, five
// rows by default with Show more, and a label that counts the horizon and the
// day.
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

  private func folded(
    _ events: [Event], alive: @escaping (Int32) -> Bool = { _ in false }
  ) throws -> DocumentsFold {
    Documents.fold(
      events, now: try seventeenTwenty(), calendar: try amsterdam(), alive: alive)
  }
}
