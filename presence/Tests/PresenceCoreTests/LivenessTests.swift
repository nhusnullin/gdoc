// Whether the process that wrote a start is still there.
//
// A start with no end means one of two things, and only the kernel knows
// which: the command is still running, or it died without writing its end.
// The app asks kill(pid, 0) and says "stopped, outcome unknown" for a dead
// one. That is the only thing it may say about such a run: the record cannot
// tell whether the write landed.

import Foundation
import XCTest

@testable import PresenceCore

final class LivenessTests: XCTestCase {
  func testAStartWithNoEndFromALivePidIsRunning() throws {
    let mine = ProcessInfo.processInfo.processIdentifier
    XCTAssertTrue(Liveness.isAlive(mine), "this very process is alive")

    let fold = Now.fold(
      [start(pid: mine)], now: try moment("2026-10-17T12:00:00Z"), alive: Liveness.isAlive)
    let run = try XCTUnwrap(fold.jobs.first, "a live start is a job, and NOW held \(fold.entries)")
    XCTAssertEqual(run.pid, mine)
    XCTAssertEqual(run.command, "propose")
    XCTAssertTrue(fold.stuck.isEmpty, "nothing is stuck while the process is there")
  }

  func testAStartWithNoEndFromADeadPidIsOutcomeUnknown() throws {
    let fold = Now.fold(
      [start(pid: 4299)], now: try moment("2026-10-17T12:00:00Z"), alive: { _ in false })
    let write = try XCTUnwrap(
      fold.stuck.first, "a dead write is stuck, and NOW held \(fold.entries)")
    XCTAssertEqual(write.cause, .stopped, "the process went away, and nothing wrote an end")
    XCTAssertEqual(write.command, "propose")
    XCTAssertTrue(fold.jobs.isEmpty, "a run whose process is gone is running nothing")
  }

  private func start(pid: Int32) -> Event {
    Event(
      schema: 1, at: Date(timeIntervalSince1970: 1_792_000_000), gdoc: "v2.13.0", pid: pid,
      run: "c3c3c3c3c3c3c3c3", call: 1, kind: .start, command: "propose", via: "chat",
      doc: "1ZyXwVuTsRqPoNmLkJiHgFeDcBa9876543210", wait: false, title: nil, ok: nil,
      verified: nil, ms: nil, hold: nil, rule: nil, until: nil)
  }
}
