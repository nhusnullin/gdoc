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

    let glance = Fold.glance(
      [start(pid: mine)],
      now: try moment("2026-10-17T12:00:00Z"),
      calendar: try amsterdam(),
      alive: Liveness.isAlive
    )
    guard case .running(let run) = glance.now.first else {
      return XCTFail("a live start is running, and this fold said \(glance.now)")
    }
    XCTAssertEqual(run.pid, mine)
    XCTAssertEqual(run.command, "propose")
    XCTAssertTrue(glance.stopped.isEmpty)
  }

  func testAStartWithNoEndFromADeadPidIsOutcomeUnknown() throws {
    let glance = Fold.glance(
      [start(pid: 4299)],
      now: try moment("2026-10-17T12:00:00Z"),
      calendar: try amsterdam(),
      alive: { _ in false }
    )
    guard case .stopped(let run) = glance.now.first else {
      return XCTFail("a dead start stopped, and this fold said \(glance.now)")
    }
    XCTAssertEqual(run.pid, 4299)
    XCTAssertEqual(glance.stopped.map(\.pid), [4299])
    XCTAssertTrue(glance.today.isEmpty, "a run with no end is nothing that finished")
  }

  private func start(pid: Int32) -> Event {
    Event(
      schema: 1, at: Date(timeIntervalSince1970: 1_792_000_000), gdoc: "v2.13.0", pid: pid,
      run: "c3c3c3c3c3c3c3c3", call: 1, kind: .start, command: "propose", via: "chat",
      doc: "1ZyXwVuTsRqPoNmLkJiHgFeDcBa9876543210", wait: false, title: nil, ok: nil,
      verified: nil, ms: nil, hold: nil, rule: nil, until: nil)
  }
}
