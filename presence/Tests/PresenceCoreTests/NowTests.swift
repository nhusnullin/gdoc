// NOW: the one section that is about this moment, and the rules that decide
// what stays in it.
//
// Scene e3 of the screens file is the order under test: two holds, a stuck
// publish, a publish that is running, and two live watches, with the oldest
// of each class above the newer one. The fixture now-order.jsonl is that
// scene as records.
//
// Every rule here is a product decision from the M24 specification's table,
// and each test names the one it holds. The clock, the pids and the
// dismissals are injected, so no test reads this machine.

import Foundation
import XCTest

@testable import PresenceCore

final class NowTests: XCTestCase {
  /// 17:20 in the fold's zone, which is the moment scene e3's menu bar shows.
  private func seventeenTwenty() throws -> Date { try moment("2026-10-17T15:20:00Z") }

  /// The scene, folded with every pid of the scene alive.
  private func scene(dismissed: Set<String> = []) throws -> NowFold {
    let scan = Event.scan(try Fixture.data("now-order"))
    XCTAssertEqual(scan.skipped, 0, "the scene fixture is clean")
    return Now.fold(
      scan.events, now: try seventeenTwenty(), dismissed: dismissed, alive: { _ in true })
  }

  func testNowOrdersHoldsStuckJobsThenWatchesOldestFirst() throws {
    let now = try scene()

    XCTAssertEqual(
      now.entries.map(band), [0, 0, 1, 2, 3, 3],
      "holds, then stuck, then running jobs, then live watches")
    XCTAssertEqual(
      now.entries.count, 6,
      "six rows: a chat run whose hold is open is the hold card and not a second row")

    XCTAssertEqual(
      now.holds.map(\.id), ["h_aa11", "h_bb22"],
      "oldest first: the hold that has waited longest is the one most likely forgotten")
    XCTAssertEqual(
      now.holds.map(\.held),
      [
        try moment("2026-10-17T14:39:01Z"), try moment("2026-10-17T15:14:01Z"),
      ])
    XCTAssertEqual(now.holds.map(\.rule), ["dictated", "steering"])
    XCTAssertEqual(now.holds.map(\.ranOut), [false, false], "both are still waiting")

    XCTAssertEqual(now.stuck.map(\.command), ["publish"])
    XCTAssertEqual(now.stuck.map(\.title), ["Chapter 4. Pricing: the draft"])
    XCTAssertEqual(now.stuck.map(\.cause), [.failed])

    XCTAssertEqual(now.jobs.map(\.command), ["publish"], "the publish that is still running")
    XCTAssertEqual(now.jobs.map(\.title), [nil], "a publish start names no title yet")

    XCTAssertEqual(
      now.watches.map(\.title),
      ["Chapter 5. Commercial Model: the storyline", "Northwind and Altery: the screening quota"],
      "oldest first, each titled from its document's last read")
    XCTAssertEqual(
      now.watches.map(\.started),
      [
        try moment("2026-10-17T11:10:00Z"), try moment("2026-10-17T14:50:00Z"),
      ])
    XCTAssertTrue(now.unknown.isEmpty, "nothing in the scene lost its end")
  }

  func testAFailedWriteIsStuckUntilItsKeyClearsOrIsDismissed() throws {
    XCTAssertEqual(
      Now.stuckCommands,
      ["reply", "propose", "withdraw", "annotate", "publish", "restyle", "export"],
      "the stuck set is the specification's list and nothing else")

    let failed = end(
      at: "2026-10-17T10:00:00Z", command: "propose", run: "aaaa", doc: docA, ok: false)
    XCTAssertEqual(try folded([failed]).stuck.map(\.command), ["propose"])

    // The key clears when the same command starts again on that document.
    let again = start(at: "2026-10-17T11:00:00Z", command: "propose", run: "bbbb", doc: docA)
    XCTAssertTrue(try folded([failed, again]).stuck.isEmpty, "the same command, the same document")

    let elsewhere = start(at: "2026-10-17T11:00:00Z", command: "propose", run: "bbbb", doc: docB)
    XCTAssertEqual(
      try folded([failed, elsewhere]).stuck.count, 1, "another document clears nothing")

    let other = start(at: "2026-10-17T11:00:00Z", command: "reply", run: "bbbb", doc: docA)
    XCTAssertEqual(try folded([failed, other]).stuck.count, 1, "another command clears nothing")

    let before = start(at: "2026-10-17T09:00:00Z", command: "propose", run: "cccc", doc: docA)
    XCTAssertEqual(
      try folded([before, failed]).stuck.count, 1, "the start that failed does not clear itself")

    // A failed publish names no document, so any later publish clears it.
    let publish = end(
      at: "2026-10-17T10:00:00Z", command: "publish", run: "dddd", doc: nil, ok: false)
    XCTAssertEqual(try folded([publish]).stuck.map(\.command), ["publish"])
    let anyPublish = start(at: "2026-10-17T11:00:00Z", command: "publish", run: "eeee", doc: docB)
    XCTAssertTrue(try folded([publish, anyPublish]).stuck.isEmpty, "any later publish clears it")

    // A publish that created a document and then failed carries that id,
    // and no publish start can ever name it: the command is handed no
    // document. Any later publish is still the answer to it.
    let made = end(
      at: "2026-10-17T10:00:00Z", command: "publish", run: "ffff", doc: nil, ok: false,
      made: docA)
    let card = try XCTUnwrap(try folded([made]).stuck.first)
    XCTAssertEqual(card.doc, docA, "the card opens what the run left behind")
    XCTAssertTrue(
      try folded([made, anyPublish]).stuck.isEmpty,
      "the id is the one it made, so any later publish clears the card")

    // Or the person dismissed it, which is the app's own state and not a line.
    XCTAssertTrue(
      try folded([failed], dismissed: ["aaaa|1"]).stuck.isEmpty,
      "a dismissal is keyed by the run and the call")
  }

  func testAFailedReadIsNotStuck() throws {
    let read = end(at: "2026-10-17T10:00:00Z", command: "read", run: "aaaa", doc: docA, ok: false)
    let fold = try folded([read])
    XCTAssertTrue(
      fold.stuck.isEmpty, "a read that failed changed nothing, so there is nothing to fix")
    XCTAssertTrue(fold.entries.isEmpty)
  }

  func testVerifiedFalseIsNeverStuck() throws {
    let hollow = end(
      at: "2026-10-17T10:00:00Z", command: "propose", run: "aaaa", doc: docA, ok: true,
      verified: false)
    XCTAssertTrue(
      try folded([hollow]).stuck.isEmpty,
      "the write happened and the read-back did not confirm it: a hollow check, never stuck")
  }

  func testADeadPidWriteWithinFourteenDaysIsStuckAndOlderIsUnknownInToday() throws {
    XCTAssertEqual(Now.horizon, 14 * 24 * 60 * 60, "fourteen days, as a duration")

    let recent = start(at: "2026-10-15T10:00:00Z", command: "propose", run: "aaaa", doc: docA)
    let recentFold = try folded([recent], alive: { _ in false })
    XCTAssertEqual(recentFold.stuck.map(\.cause), [.stopped], "a write whose process is gone")
    XCTAssertEqual(recentFold.stuck.map(\.command), ["propose"])
    XCTAssertTrue(recentFold.unknown.isEmpty, "it is a stuck card, so it is not an outcome row")

    let old = start(at: "2026-10-01T10:00:00Z", command: "propose", run: "bbbb", doc: docA)
    let oldFold = try folded([old], alive: { _ in false })
    XCTAssertTrue(oldFold.stuck.isEmpty, "older than the horizon, so NOW lets it go")
    XCTAssertEqual(oldFold.unknown.map(\.command), ["propose"], "a row whose outcome is unknown")

    let reading = start(at: "2026-10-15T10:00:00Z", command: "read", run: "cccc", doc: docA)
    let readFold = try folded([reading], alive: { _ in false })
    XCTAssertTrue(readFold.stuck.isEmpty, "a read that stopped wrote nothing")
    XCTAssertEqual(readFold.unknown.map(\.command), ["read"])

    let watch = start(
      at: "2026-10-17T09:00:00Z", command: "comments", run: "dddd", doc: docA, wait: true)
    let watchFold = try folded([watch], alive: { _ in false })
    XCTAssertTrue(
      watchFold.entries.isEmpty, "a watch whose process is gone is not live and not stuck")
    XCTAssertEqual(watchFold.unknown.map(\.command), ["comments"])
  }

  func testAHoldCardTakesTheDocumentsLatestTitle() throws {
    let now = try scene()
    let card = try XCTUnwrap(now.holds.first { $0.id == "h_aa11" })
    XCTAssertEqual(
      card.title, "Group Channel Policy: the frozen vocabulary",
      "the held line carries no title, so the card takes the document's")

    let scan = Event.scan(try Fixture.data("now-order"))
    let renamed = end(
      at: "2026-10-17T15:00:00Z", command: "read", run: "ffff",
      doc: "1PolAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp", ok: true, title: "Group Channel Policy v2")
    let after = Now.fold(
      scan.events + [renamed], now: try seventeenTwenty(), alive: { _ in true })
    XCTAssertEqual(
      after.holds.first { $0.id == "h_aa11" }?.title, "Group Channel Policy v2",
      "the latest title wins, because that is what the person sees in Drive")
  }

  func testDismissalsAreAnInputAndTheRowStays() throws {
    let stopped = start(at: "2026-10-15T10:00:00Z", command: "propose", run: "aaaa", doc: docA)
    let dismissed = try folded([stopped], dismissed: ["aaaa|1"], alive: { _ in false })
    XCTAssertTrue(dismissed.stuck.isEmpty, "the card goes")
    XCTAssertEqual(
      dismissed.unknown.map(\.run), ["aaaa"],
      "and the row TODAY draws stays: a dismissal hides a card, it does not rewrite the record")

    let whole = try scene()
    let withoutTheHold = try scene(dismissed: ["h_aa11"])
    XCTAssertEqual(
      withoutTheHold.holds.map(\.id), ["h_bb22"], "a hold is dismissed by its own id")
    XCTAssertEqual(
      withoutTheHold.entries.count, whole.entries.count - 1, "one card fewer and nothing else")
    XCTAssertEqual(withoutTheHold.stuck, whole.stuck)
    XCTAssertEqual(withoutTheHold.jobs, whole.jobs)
    XCTAssertEqual(withoutTheHold.watches, whole.watches)
    XCTAssertEqual(withoutTheHold.unknown, whole.unknown)
  }

  func testTheLastVerifiedWriteIsReportedForTheCheck() throws {
    let first = end(
      at: "2026-10-17T10:00:00Z", command: "annotate", run: "aaaa", doc: docA, ok: true,
      verified: true, title: "The register")
    let second = end(
      at: "2026-10-17T11:00:00Z", command: "propose", run: "bbbb", doc: docB, ok: true,
      verified: true, title: "The draft")
    let hollow = end(
      at: "2026-10-17T12:00:00Z", command: "propose", run: "cccc", doc: docB, ok: true,
      verified: false)
    let silent = end(at: "2026-10-17T13:00:00Z", command: "read", run: "dddd", doc: docA, ok: true)

    let fold = try folded([first, second, hollow, silent])
    let check = try XCTUnwrap(fold.lastVerified, "the bar's green check needs one")
    XCTAssertEqual(check.run, "bbbb", "the latest read-back that confirmed")
    XCTAssertEqual(check.call, 1)
    XCTAssertEqual(check.command, "propose")
    XCTAssertEqual(check.doc, docB)
    XCTAssertEqual(check.title, "The draft")
    XCTAssertEqual(check.ended, try moment("2026-10-17T11:00:00Z"))

    XCTAssertNil(
      try folded([hollow, silent]).lastVerified,
      "a read-back that did not confirm, and a command that reads nothing back, are not a check")

    // A publish whose pairing failed and whose rollback failed answers
    // `ok: false` with the read-back it did manage still in the object. NOW
    // holds a stuck card for that run, so the bar may not show a check over
    // it: the two would say the opposite thing at the same moment.
    let brokenPairing = end(
      at: "2026-10-17T14:00:00Z", command: "publish", run: "eeee", doc: nil, ok: false,
      verified: true, title: "The draft", made: docA)
    let broken = try folded([first, second, brokenPairing])
    XCTAssertEqual(broken.stuck.map { $0.run }, ["eeee"], "the failed publish is the card")
    XCTAssertEqual(
      broken.lastVerified?.run, "bbbb",
      "a write that failed is no check, whatever its read-back managed")
  }

  // MARK: - The pieces a test states

  private let docA = "1AaaAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let docB = "1BbbAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"

  /// The four classes of NOW, as numbers, so one assertion states the order.
  /// A publish is handed no document, so nothing the record says about a
  /// document names it: the only name it has is the one its own lines
  /// carried. Scene r2's card reads "Publish stopped · Chapter 4" off a run
  /// whose end names nothing at all.
  func testACallIsNamedFromItsOwnLinesTheLatestWinning() throws {
    let named = try folded(
      [
        start(
          at: "2026-10-17T09:03:10Z", command: "publish", run: "aaaa", doc: nil,
          title: "Chapter 4"),
        end(at: "2026-10-17T09:05:00Z", command: "publish", run: "aaaa", doc: nil, ok: false),
      ], alive: { _ in false })
    XCTAssertEqual(
      named.stuck.map(\.title), ["Chapter 4"],
      "the name gdoc read off the note before the run began")

    // The latest line wins, because that is the name on the cover.
    let renamed = try folded(
      [
        start(
          at: "2026-10-17T09:03:10Z", command: "publish", run: "bbbb", doc: nil,
          title: "Chapter 4"),
        end(
          at: "2026-10-17T09:05:00Z", command: "publish", run: "bbbb", doc: nil, ok: false,
          title: "Chapter 4, draft v2"),
      ], alive: { _ in false })
    XCTAssertEqual(renamed.stuck.map(\.title), ["Chapter 4, draft v2"])

    // The record stamps one run on every line a process writes, so a chat
    // session is one run and a dozen calls. The name of one of its writes
    // is not the name of another.
    let session = try folded(
      [
        start(
          at: "2026-10-17T14:20:00Z", command: "reply", run: "cccc", doc: docA, call: 1),
        end(
          at: "2026-10-17T14:20:10Z", command: "reply", run: "cccc", doc: docA, ok: true,
          call: 1, verified: true, title: "Group Channel Policy"),
        start(at: "2026-10-17T14:30:00Z", command: "publish", run: "cccc", doc: nil, call: 2),
        end(
          at: "2026-10-17T14:31:00Z", command: "publish", run: "cccc", doc: nil, ok: false,
          call: 2),
      ], alive: { _ in false })
    XCTAssertEqual(session.stuck.map(\.command), ["publish"], "the read-back reply is not stuck")
    XCTAssertNil(session.stuck.first?.title, "no line of that call named anything")
  }

  private func band(_ entry: NowEntry) -> Int {
    switch entry {
    case .hold: return 0
    case .stuck: return 1
    case .job: return 2
    case .watch: return 3
    }
  }

  private func folded(
    _ events: [Event], dismissed: Set<String> = [], alive: @escaping (Int32) -> Bool = { _ in true }
  ) throws -> NowFold {
    Now.fold(events, now: try seventeenTwenty(), dismissed: dismissed, alive: alive)
  }

  private func start(
    at text: String, command: String, run: String, doc: String?, call: Int64 = 1,
    wait: Bool = false, title: String? = nil
  ) -> Event {
    event(
      at: text, kind: .start, command: command, run: run, doc: doc, call: call, wait: wait,
      title: title)
  }

  private func end(
    at text: String, command: String, run: String, doc: String?, ok: Bool, call: Int64 = 1,
    verified: Bool? = nil, title: String? = nil, made: String? = nil
  ) -> Event {
    event(
      at: text, kind: .end, command: command, run: run, doc: doc, call: call, ok: ok,
      verified: verified, title: title, made: made)
  }

  private func event(
    at text: String, kind: EventKind, command: String, run: String, doc: String?,
    call: Int64 = 1, wait: Bool = false, ok: Bool? = nil, verified: Bool? = nil,
    title: String? = nil, made: String? = nil
  ) -> Event {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return Event(
      schema: 1, at: formatter.date(from: text) ?? Date(), gdoc: "v2.14.0", pid: 6000, run: run,
      call: call, kind: kind, command: command, via: "terminal", doc: doc, wait: wait,
      title: title, made: made, ok: ok, verified: verified, ms: ok == nil ? nil : 1200,
      hold: nil, rule: nil, until: nil)
  }
}
