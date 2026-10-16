// The fold: a day of records into what is happening now and what happened
// today.
//
// The fold is the whole reading the app does. It is a pure function of the
// records, the moment, the calendar and one question to the kernel, so every
// state the glyph and the panel can show is reachable from a fixture and
// nothing here waits for a clock.

import Foundation
import XCTest

@testable import PresenceCore

final class FoldTests: XCTestCase {
  /// 2026-10-17 at midday UTC, which is the afternoon in the fold's zone and
  /// well inside the fixture's day.
  private func midday() throws -> Date { try moment("2026-10-17T12:00:00Z") }

  /// The fixture's day, folded with the pids of the fixture's own processes
  /// alive except the one that stopped.
  private func day(now: Date? = nil, dead: Set<Int32> = [4299]) throws -> Glance {
    let scan = Event.scan(try Fixture.data("day"))
    XCTAssertEqual(scan.skipped, 0, "the day fixture is clean")
    return Fold.glance(
      scan.events,
      now: try now ?? midday(),
      calendar: try amsterdam(),
      alive: { !dead.contains($0) }
    )
  }

  func testADuplicateRecordCountsOnce() throws {
    let data = try Fixture.data("day")
    let twice = data + data

    let once = Event.scan(data)
    let doubled = Event.scan(twice)
    XCTAssertEqual(doubled.events.count, once.events.count * 2, "the file really has them twice")

    let want = try day()
    let got = Fold.glance(
      doubled.events,
      now: try midday(),
      calendar: try amsterdam(),
      alive: { $0 != 4299 }
    )
    XCTAssertEqual(got, want, "a line written twice is one run plus one call, and so one row")
  }

  func testAWaitThatIsRunningIsAWatch() throws {
    let glance = try day()
    let watches = glance.now.compactMap { row -> ActiveRun? in
      if case .watching(let run) = row { return run }
      return nil
    }
    XCTAssertEqual(watches.count, 1)
    XCTAssertEqual(watches[0].command, "comments")
    XCTAssertEqual(watches[0].pid, 4242)
    XCTAssertTrue(watches[0].isWait)
    XCTAssertEqual(watches[0].started, try moment("2026-10-17T09:00:00Z"))

    let running = glance.now.compactMap { row -> ActiveRun? in
      if case .running(let run) = row { return run }
      return nil
    }
    XCTAssertEqual(
      running.map(\.command), ["reply"], "a start with no wait is a run and not a watch")
  }

  func testAHeldEventIsAHoldUntilReleasedExpiredOrLanded() throws {
    let glance = try day()
    XCTAssertEqual(
      glance.holds.map(\.id), ["h_cd34", "h_ab12"], "newest first, and both still held")

    let scan = Event.scan(try Fixture.data("day"))
    for ending in [EventKind.released, .expired, .landed] {
      let closed = scan.events + [hold(id: "h_cd34", kind: ending)]
      let after = Fold.glance(
        closed, now: try midday(), calendar: try amsterdam(), alive: { $0 != 4299 })
      XCTAssertEqual(
        after.holds.map(\.id), ["h_ab12"],
        "a \(ending.rawValue) takes the hold out of now")
    }
  }

  func testAHoldPastItsUntilIsShownAsRanOutEvenWithNoEvent() throws {
    let glance = try day()
    let byID = Dictionary(uniqueKeysWithValues: glance.holds.map { ($0.id, $0) })

    let old = try XCTUnwrap(byID["h_ab12"])
    XCTAssertEqual(old.until, try moment("2026-10-17T10:35:01Z"))
    XCTAssertTrue(old.ranOut, "its life ran out before now and no event said so")
    XCTAssertEqual(old.rule, "echo")

    let waiting = try XCTUnwrap(byID["h_cd34"])
    XCTAssertFalse(waiting.ranOut, "this one is still waiting for a person")
    XCTAssertEqual(waiting.command, "propose")
  }

  func testTodayIsTheLocalDayNewestFirst() throws {
    let glance = try day()
    XCTAssertEqual(
      glance.today.map(\.command), ["read", "annotate", "publish"],
      "three ends today, newest first")
    XCTAssertEqual(
      glance.today.map(\.title), ["Treasury policy", "Supplier register", "Q3 board pack"])
    XCTAssertEqual(glance.today.map(\.ok), [true, true, false])
    XCTAssertEqual(glance.today[1].verified, true)
    XCTAssertEqual(glance.today[0].ms, 820)
    XCTAssertEqual(glance.gdoc, "v2.13.0", "the version of the last record that named one")

    // A UTC month boundary is not a local day boundary. Both of these are
    // 2026-11-01 in Amsterdam, and one of them is in October's file.
    let lastOfOctober = ended(at: "2026-10-31T23:30:00Z", command: "read", run: "1111111111111111")
    let firstOfNovember = ended(
      at: "2026-11-01T00:30:00Z", command: "reply", run: "2222222222222222")
    let dayBefore = ended(at: "2026-10-31T22:30:00Z", command: "publish", run: "3333333333333333")
    let across = Fold.glance(
      [dayBefore, lastOfOctober, firstOfNovember],
      now: try moment("2026-11-01T02:00:00Z"),
      calendar: try amsterdam(),
      alive: { _ in true }
    )
    XCTAssertEqual(
      across.today.map(\.command), ["reply", "read"],
      "the 22:30Z end is the local day before, and the 23:30Z end is not")
  }

  func testNowIsHoldsThenLiveStartsThenStopped() throws {
    let glance = try day()
    XCTAssertEqual(glance.now.count, 5, "two holds, a watch, a run and a stopped run")
    XCTAssertEqual(
      glance.now.map(order), [0, 0, 1, 1, 2],
      "the holds first, then the live starts, then the one whose pid is gone")
    XCTAssertEqual(
      glance.live.map(\.command), ["reply", "comments"], "newest first inside the band")
  }

  /// A hold's ending event, built here rather than kept in a fixture, so the
  /// three endings are one loop and the fixture stays the day it describes.
  private func hold(id: String, kind: EventKind) -> Event {
    Event(
      schema: 1, at: Date(timeIntervalSince1970: 1_792_000_000), gdoc: "v2.13.0", pid: 4243,
      run: "b2b2b2b2b2b2b2b2", call: nil, kind: kind, command: nil, via: nil, doc: nil,
      wait: false, title: nil, ok: nil, verified: nil, ms: nil, hold: id, rule: nil, until: nil)
  }

  /// One end event, for the day-boundary case the fixture cannot hold.
  private func ended(at text: String, command: String, run: String) -> Event {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return Event(
      schema: 1, at: formatter.date(from: text) ?? Date(), gdoc: "v2.13.0", pid: 4242, run: run,
      call: 1, kind: .end, command: command, via: "terminal", doc: nil, wait: false, title: nil,
      ok: true, verified: nil, ms: 10, hold: nil, rule: nil, until: nil)
  }

  /// The three bands of now, as numbers, so one assertion states the order.
  private func order(_ row: NowRow) -> Int {
    switch row {
    case .hold: return 0
    case .running, .watching: return 1
    case .stopped: return 2
    }
  }
}
