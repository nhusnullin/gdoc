// Whether the app may open its own bundle again, and what a new instance
// does about the one that launched it.
//
// Two instances of a menu bar app are two glyphs, and a relaunch into a
// bundle somebody else swapped in is this app starting a program nobody
// asked for. Both are held by the table below rather than by a review: every
// guard is one row, each on its own, so a change that drops one fails here.

import Foundation
import XCTest

@testable import PresenceCore

final class RelaunchPolicyTests: XCTestCase {
  private let identifier = "com.altery.gdoc.presence"
  private let running = "2.14.0"
  private let onDisk = "2.15.0"

  func testRelaunchHappensOncePerOnDiskVersionInTenMinutes() throws {
    let at = try moment("2026-10-17T12:00:00Z")

    XCTAssertEqual(
      RelaunchPolicy.decide(ask(now: at)),
      .relaunch(version: onDisk, after: 2),
      "the disk holds another version and every guard holds, so the app reopens itself")

    XCTAssertEqual(
      RelaunchPolicy.decide(
        ask(now: at, last: RelaunchStamp(version: onDisk, at: at.addingTimeInterval(-540)))),
      .hold(.alreadyTried),
      "nine minutes ago, for this same version on the disk")

    XCTAssertEqual(
      RelaunchPolicy.decide(
        ask(now: at, last: RelaunchStamp(version: onDisk, at: at.addingTimeInterval(-600)))),
      .relaunch(version: onDisk, after: 2),
      "ten minutes is the whole of the guard")

    XCTAssertEqual(
      RelaunchPolicy.decide(
        ask(now: at, last: RelaunchStamp(version: "2.13.0", at: at.addingTimeInterval(-60)))),
      .relaunch(version: onDisk, after: 2),
      "the guard is per version on the disk, and this is another one")

    XCTAssertEqual(RelaunchPolicy.once, 600)
    XCTAssertEqual(RelaunchPolicy.settle, 2)
  }

  func testNoRelaunchWhileAnUpdateRunsOrAChildLives() throws {
    let at = try moment("2026-10-17T12:00:00Z")

    XCTAssertEqual(
      RelaunchPolicy.decide(ask(now: at, updateRunning: true)),
      .hold(.updateRunning),
      "a gdoc update is writing the bytes this relaunch would run")

    XCTAssertEqual(
      RelaunchPolicy.decide(ask(now: at, childAlive: true)),
      .hold(.childAlive),
      "the app's own child has not exited, so quitting would take it with us")

    // An update in the record outranks a child of this app's own: both are
    // reasons to wait, and the record's is the one a person can see.
    XCTAssertEqual(
      RelaunchPolicy.decide(ask(now: at, updateRunning: true, childAlive: true)),
      .hold(.updateRunning))
  }

  func testNoRelaunchIntoAnotherIdentifierOrAnInvalidSignature() throws {
    let at = try moment("2026-10-17T12:00:00Z")

    XCTAssertEqual(
      RelaunchPolicy.decide(ask(now: at, onDiskIdentifier: "com.example.something")),
      .hold(.anotherIdentifier),
      "the bundle on the disk is not this app, whatever its version says")

    XCTAssertEqual(
      RelaunchPolicy.decide(ask(now: at, onDiskIdentifier: nil)),
      .hold(.anotherIdentifier),
      "a bundle that names no identifier is not this app either")

    XCTAssertEqual(
      RelaunchPolicy.decide(ask(now: at, signatureValid: false)),
      .hold(.invalidSignature),
      "the signature on the disk did not check out")
  }

  func testGoneNeverRelaunches() throws {
    let at = try moment("2026-10-17T12:00:00Z")

    XCTAssertEqual(
      RelaunchPolicy.decide(ask(now: at, bundle: .gone)),
      .hold(.gone),
      "there is no bundle to open, and the row says the app was removed")

    XCTAssertEqual(
      RelaunchPolicy.decide(ask(now: at, bundle: .same)),
      .hold(.unchanged),
      "the disk holds the version that is running, so there is nothing to do")

    // .gone outranks every other reason, because there is nothing to open.
    XCTAssertEqual(
      RelaunchPolicy.decide(
        ask(now: at, bundle: .gone, onDiskIdentifier: nil, signatureValid: false)),
      .hold(.gone))
  }

  func testAFailedOpenLeavesTheReopenRow() throws {
    let at = try moment("2026-10-17T12:00:00Z")

    XCTAssertEqual(
      RelaunchPolicy.after(opened: true),
      .quit,
      "the new instance is up, so this one goes")

    XCTAssertEqual(
      RelaunchPolicy.after(opened: false),
      .keepTheReopenRow,
      "nothing opened, so the app stays up and HEALTH still says quit and reopen")

    // The stamp is written when the attempt is made and not when it worked,
    // so an open that keeps failing is tried once in ten minutes and never
    // in a loop.
    XCTAssertEqual(
      RelaunchPolicy.decide(
        ask(now: at, last: RelaunchStamp(version: onDisk, at: at.addingTimeInterval(-5)))),
      .hold(.alreadyTried))
  }

  func testANewInstanceWaitsForItsLauncherThenASecondInstanceQuits() {
    XCTAssertEqual(RelaunchPolicy.argument, "--relaunched-by")
    XCTAssertEqual(RelaunchPolicy.patience, 5)

    XCTAssertEqual(
      RelaunchPolicy.launcher(in: ["GdocPresence", "--relaunched-by", "4242"]), 4242)
    XCTAssertNil(RelaunchPolicy.launcher(in: ["GdocPresence"]))
    XCTAssertNil(RelaunchPolicy.launcher(in: ["GdocPresence", "--relaunched-by"]))
    XCTAssertNil(RelaunchPolicy.launcher(in: ["GdocPresence", "--relaunched-by", "nobody"]))
    XCTAssertNil(RelaunchPolicy.launcher(in: ["GdocPresence", "--relaunched-by", "0"]))
    XCTAssertNil(RelaunchPolicy.launcher(in: ["GdocPresence", "--relaunched-by", "-1"]))

    XCTAssertEqual(
      RelaunchPolicy.step(
        launcher: 4242, launcherAlive: true, anotherInstance: true, waited: 0),
      .waitForLauncher(pid: 4242, upTo: 5),
      "the instance that opened this one is still quitting")

    XCTAssertEqual(
      RelaunchPolicy.step(
        launcher: 4242, launcherAlive: true, anotherInstance: true, waited: 4),
      .waitForLauncher(pid: 4242, upTo: 1),
      "four of the five seconds are spent")

    XCTAssertEqual(
      RelaunchPolicy.step(
        launcher: 4242, launcherAlive: false, anotherInstance: false, waited: 0.2),
      .carryOn,
      "the launcher is gone and nobody else is up, so this instance is the app")

    XCTAssertEqual(
      RelaunchPolicy.step(
        launcher: 4242, launcherAlive: false, anotherInstance: true, waited: 0.2),
      .quit,
      "somebody else of this identifier is up, so this one yields")

    XCTAssertEqual(
      RelaunchPolicy.step(
        launcher: 4242, launcherAlive: true, anotherInstance: true, waited: 5),
      .quit,
      "the five seconds are spent and the launcher stayed, so it keeps its reopen row")

    XCTAssertEqual(
      RelaunchPolicy.step(
        launcher: nil, launcherAlive: false, anotherInstance: true, waited: 0),
      .quit,
      "a second instance quits however it was started")

    XCTAssertEqual(
      RelaunchPolicy.step(
        launcher: nil, launcherAlive: false, anotherInstance: false, waited: 0),
      .carryOn,
      "an ordinary first launch waits for nothing")
  }

  /// One ask with every guard holding, and the one field a row changes.
  private func ask(
    now: Date,
    bundle: BundleState? = nil,
    onDiskIdentifier: String? = "com.altery.gdoc.presence",
    signatureValid: Bool = true,
    updateRunning: Bool = false,
    childAlive: Bool = false,
    last: RelaunchStamp? = nil
  ) -> RelaunchAsk {
    RelaunchAsk(
      bundle: bundle ?? .reopenNeeded(onDisk),
      identifier: identifier,
      onDiskIdentifier: onDiskIdentifier,
      signatureValid: signatureValid,
      updateRunning: updateRunning,
      childAlive: childAlive,
      last: last,
      now: now)
  }
}
