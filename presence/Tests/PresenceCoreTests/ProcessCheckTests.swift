// Whether the panel may send a signal to the pid a live row names.
//
// The record is forgeable, so every fact in it is read as a claim. The check
// is the table below: one row per claim that can be wrong, each on its own,
// so a change that drops a check fails here rather than in somebody's
// terminal.

import Foundation
import XCTest

@testable import PresenceCore

final class ProcessCheckTests: XCTestCase {
  func testAWaitIsStoppableOnlyWhenEveryCheckHolds() throws {
    let at = try moment("2026-10-17T12:00:00Z")
    let gdoc = "/Users/nail/.local/bin/gdoc"

    XCTAssertEqual(
      ProcessCheck.verdict(
        wait: wait(at: at), hasEnd: false, running: alive(at: at), gdoc: gdoc, user: 501,
        resolve: { $0 }),
      .stoppable,
      "every check holds, so this watch is one the panel may stop")

    // The record writes whole seconds, so a start inside the second the
    // event names reads as later than it by up to a second.
    XCTAssertEqual(
      ProcessCheck.verdict(
        wait: wait(at: at), hasEnd: false,
        running: alive(at: at, started: at.addingTimeInterval(1)),
        gdoc: gdoc, user: 501, resolve: { $0 }),
      .stoppable,
      "a second of truncation is not a different process")

    XCTAssertEqual(
      ProcessCheck.verdict(
        wait: wait(at: at), hasEnd: false,
        running: alive(at: at, started: at.addingTimeInterval(-10)), gdoc: gdoc, user: 501,
        resolve: { $0 }),
      .stoppable,
      "ten seconds before the event is inside the window")

    let table: [(String, StopVerdict, Event, Bool, RunningProcess)] = [
      (
        "the path is another program", .refused(.notGdoc), wait(at: at), false,
        alive(at: at, path: "/usr/bin/vim")
      ),
      (
        "the kernel named no path", .refused(.pathUnknown), wait(at: at), false,
        alive(at: at, path: nil)
      ),
      (
        "the pid belongs to another user", .refused(.notOurUser), wait(at: at), false,
        alive(at: at, uid: 502)
      ),
      (
        "it started eleven seconds before the event", .refused(.outsideTheWindow), wait(at: at),
        false, alive(at: at, started: at.addingTimeInterval(-11))
      ),
      (
        "it started two seconds after the event", .refused(.outsideTheWindow), wait(at: at), false,
        alive(at: at, started: at.addingTimeInterval(2))
      ),
      (
        "the kernel named no start time", .refused(.startUnknown), wait(at: at), false,
        RunningProcess(path: gdoc, uid: 501, started: nil, isAlive: true)
      ),
      (
        "the pid is dead", .refused(.gone), wait(at: at), false,
        RunningProcess(path: gdoc, uid: 501, started: at, isAlive: false)
      ),
      ("an end is already written", .refused(.ended), wait(at: at), true, alive(at: at)),
      ("the run says chat", .refused(.fromChat), wait(at: at, via: "chat"), false, alive(at: at)),
      (
        "the run is not a wait", .refused(.notAWait), wait(at: at, isWait: false), false,
        alive(at: at)
      ),
      (
        "the run is another command", .refused(.notAWait), wait(at: at, command: "propose"), false,
        alive(at: at)
      ),
    ]

    for (reason, expected, start, hasEnd, running) in table {
      XCTAssertEqual(
        ProcessCheck.verdict(
          wait: start, hasEnd: hasEnd, running: running, gdoc: gdoc, user: 501, resolve: { $0 }),
        expected,
        "\(reason), so the panel may not stop it")
    }

    XCTAssertEqual(
      ProcessCheck.refusalLine, "This watch is not one gdoc presence can stop",
      "one sentence for every refusal, because the person cannot act on the reason")
  }

  func testPreviousCountsAsGdocAndPathsAreResolvedBeforeComparing() throws {
    let at = try moment("2026-10-17T12:00:00Z")

    // gdoc update renames the running binary beside the new one.
    XCTAssertEqual(
      ProcessCheck.verdict(
        wait: wait(at: at), hasEnd: false,
        running: alive(at: at, path: "/Users/nail/.local/bin/gdoc.previous"),
        gdoc: "/Users/nail/.local/bin/gdoc", user: 501, resolve: { $0 }),
      .stoppable,
      "the binary this wait runs is the one the app knows, renamed")

    XCTAssertEqual(
      ProcessCheck.verdict(
        wait: wait(at: at), hasEnd: false,
        running: alive(at: at, path: "/Users/nail/.local/bin/gdoc.previously"),
        gdoc: "/Users/nail/.local/bin/gdoc", user: 501, resolve: { $0 }),
      .refused(.notGdoc),
      "only that one name, not anything beginning with it")

    // The checkout route links ~/.local/bin/gdoc at the repository's binary,
    // so the two sides are the same program under two names.
    let room = URL(fileURLWithPath: NSTemporaryDirectory())
      .appendingPathComponent("process-check-\(UUID().uuidString)")
    try FileManager.default.createDirectory(at: room, withIntermediateDirectories: true)
    defer { try? FileManager.default.removeItem(at: room) }
    let real = room.appendingPathComponent("built")
    let link = room.appendingPathComponent("gdoc")
    try Data("#!/bin/sh\n".utf8).write(to: real)
    try FileManager.default.createSymbolicLink(at: link, withDestinationURL: real)

    XCTAssertEqual(
      ProcessCheck.verdict(
        wait: wait(at: at), hasEnd: false, running: alive(at: at, path: real.path),
        gdoc: link.path, user: 501),
      .stoppable,
      "the link and its target are one program, and the default resolver says so")

    XCTAssertEqual(
      ProcessCheck.resolved(link.path), ProcessCheck.resolved(real.path),
      "the resolver follows the link rather than comparing the two names")
  }

  private func wait(
    at: Date, command: String = "comments", via: String? = "terminal", isWait: Bool = true
  ) -> Event {
    Event(
      schema: 1, at: at, gdoc: "v2.14.0", pid: 4311, run: "a1b2c3d4e5f60718", call: 3,
      kind: .start, command: command, via: via,
      doc: "1ZyXwVuTsRqPoNmLkJiHgFeDcBa9876543210", wait: isWait, title: nil, ok: nil,
      verified: nil, ms: nil, hold: nil, rule: nil, until: nil)
  }

  private func alive(
    at: Date, path: String? = "/Users/nail/.local/bin/gdoc", uid: uid_t = 501,
    started: Date? = nil
  ) -> RunningProcess {
    RunningProcess(
      path: path, uid: uid, started: started ?? at.addingTimeInterval(-2), isAlive: true)
  }
}
