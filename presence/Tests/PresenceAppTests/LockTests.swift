// The one lock, and what a second instance does about it.
//
// gdoc asks whether the app is running by asking for the lock the app holds.
// Three things are held here: the path is the literal one both programs
// name, a second holder is refused while the first holds it, and an instance
// that does not get the lock quits without drawing anything.
//
// Nothing here starts an app or a child. The second holder is a second
// descriptor in this test's own process, which macOS refuses exactly as it
// refuses another program's, because a flock belongs to an open file and not
// to a process.

import Darwin
import Foundation
import PresenceCore
import XCTest

@testable import PresenceApp

final class LockTests: XCTestCase {
  private var root = URL(fileURLWithPath: "/tmp")
  private var held: [AppLock] = []

  override func setUpWithError() throws {
    try super.setUpWithError()
    root = URL(fileURLWithPath: NSTemporaryDirectory())
      .appendingPathComponent("gdoc-lock-\(UUID().uuidString)")
    try FileManager.default.createDirectory(at: root, withIntermediateDirectories: true)
  }

  override func tearDownWithError() throws {
    for lock in held { lock.release() }
    held = []
    try? FileManager.default.removeItem(at: root)
    try super.tearDownWithError()
  }

  // MARK: the path

  func testTheLockSitsBesideTheMonthFiles() {
    // Arrange: a machine with no GDOC_CONFIG_DIR, which is Nail's.
    let places = Places.ofThisMachine(
      environment: [:], bundle: nil, identifier: "", arguments: [], home: "/Users/someone")

    // Assert: the literal path, which go/cmd/gdoc names the same way.
    XCTAssertEqual(
      places.lock.path, "/Users/someone/.config/gdoc-agent/presence/app.lock")
  }

  func testTheNamedDirectoryCarriesTheLockToo() {
    let places = Places.ofThisMachine(
      environment: ["GDOC_CONFIG_DIR": "/tmp/elsewhere"], bundle: nil, identifier: "",
      arguments: [], home: "/Users/someone")

    XCTAssertEqual(places.lock.path, "/tmp/elsewhere/presence/app.lock")
  }

  // MARK: the lock

  func testTheFirstHolderGetsItAndTheFileIsTheModeTheDecisionNames() throws {
    // Arrange: a umask this test owns, so the mode asserted is the mode asked
    // for and not the mode of whoever ran the suite.
    let mask = umask(0o022)
    defer { umask(mask) }

    // Act
    let lock = take(at: lockFile)

    // Assert
    XCTAssertEqual(lock.standing, .mine)
    let mode = try FileManager.default.attributesOfItem(atPath: lockFile.path)[.posixPermissions]
    XCTAssertEqual(mode as? NSNumber, 0o644)
  }

  func testTheDirectoryIsMadeWhereGdocHasNeverWritten() {
    // Arrange: a config directory with no presence folder in it yet, which is
    // a machine where the setting was turned on and no command has run.
    let records = root.appendingPathComponent("never-written/presence")
    XCTAssertFalse(FileManager.default.fileExists(atPath: records.path))

    let lock = take(at: records.appendingPathComponent("app.lock"))

    XCTAssertEqual(lock.standing, .mine, "an app that cannot lock would never be seen as running")
  }

  func testASecondHolderIsRefusedWhileTheFirstHoldsIt() {
    // Arrange: the lock is held, as it is for the app's whole life.
    let first = take(at: lockFile)
    XCTAssertEqual(first.standing, .mine)

    // Act: a second open of the same path, which is what another instance
    // and what gdoc both do.
    let second = take(at: lockFile)

    // Assert
    XCTAssertEqual(second.standing, .another)
  }

  func testAReleasedLockIsTheNextInstancesToTake() {
    let first = take(at: lockFile)
    first.release()

    XCTAssertEqual(take(at: lockFile).standing, .mine, "the kernel let go with the descriptor")
  }

  func testAReleaseTwiceIsOneRelease() {
    let lock = take(at: lockFile)
    lock.release()
    lock.release()

    XCTAssertEqual(take(at: lockFile).standing, .mine)
  }

  // MARK: what an instance does about it

  func testAnInstanceThatDoesNotGetTheLockQuits() {
    XCTAssertEqual(AppLock.decide(.mine), .run, "the one app draws its status item")
    XCTAssertEqual(AppLock.decide(.another), .quit, "a second instance leaves at once")
    XCTAssertEqual(
      AppLock.decide(.unknown), .run,
      "a lock nobody can open refuses nothing: the menu bar is the point")
  }

  func testOnlyAnInstanceAnotherOneLaunchedWaitsForTheLock() {
    XCTAssertEqual(AppLock.patience(ofAnInstanceWith: []), 0)
    XCTAssertEqual(AppLock.patience(ofAnInstanceWith: ["GdocPresence"]), 0)
    XCTAssertEqual(
      AppLock.patience(ofAnInstanceWith: ["GdocPresence", RelaunchPolicy.argument, "4242"]),
      RelaunchPolicy.patience,
      "the instance an update opened waits for the one that opened it to go")
  }

  func testAnInstanceThatWaitsAsksAgainUntilItsPatienceIsSpent() {
    // Arrange: the lock is held and nobody releases it.
    let first = take(at: lockFile)
    XCTAssertEqual(first.standing, .mine)

    // Act: three waits' worth of patience, and a sleep this test does not
    // spend.
    var asked = 0
    let second = AppLock.hold(
      at: lockFile, waitingUpTo: AppLock.asksEvery * 3, sleep: { _ in asked += 1 })
    held.append(second)

    // Assert: four asks, three waits between them, and then it leaves.
    XCTAssertEqual(second.standing, .another)
    XCTAssertEqual(asked, 3)
    XCTAssertEqual(AppLock.asks(within: AppLock.asksEvery * 3), 4)
  }

  func testAnInstanceWithNoPatienceAsksOnceAndWaitsNotAtAll() {
    let first = take(at: lockFile)
    XCTAssertEqual(first.standing, .mine)

    var asked = 0
    let second = AppLock.hold(at: lockFile, waitingUpTo: 0, sleep: { _ in asked += 1 })
    held.append(second)

    XCTAssertEqual(second.standing, .another)
    XCTAssertEqual(asked, 0, "a second instance somebody started twice leaves at once")
    XCTAssertEqual(AppLock.asks(within: 0), 1)
  }

  func testAWaitEndsAsSoonAsTheLockIsFree() {
    let first = take(at: lockFile)
    var asked = 0

    // The holder lets go between two asks, which is the launcher quitting.
    let second = AppLock.hold(
      at: lockFile, waitingUpTo: 5,
      sleep: { _ in
        asked += 1
        if asked == 2 { first.release() }
      })
    held.append(second)

    XCTAssertEqual(second.standing, .mine)
    XCTAssertEqual(asked, 2, "it waited no longer than it had to")
  }

  // MARK: the shell

  /// The lock is taken before anything is drawn, and let go only when the
  /// app terminates.
  ///
  /// The shell is an executable target, which no test can import, so this is
  /// a source scan: the hold comes before the status item and the session,
  /// the quit comes before them too, and the one release is in
  /// `applicationWillTerminate`.
  func testTheShellTakesTheLockFirstAndLetsGoLast() throws {
    let text = try String(contentsOf: shellMain, encoding: .utf8)

    let hold = try XCTUnwrap(text.range(of: "AppLock.hold("), "the shell takes no lock")
    let item = try XCTUnwrap(
      text.range(of: "StatusItem {"), "the shell draws no status item")
    let session = try XCTUnwrap(text.range(of: "Session("), "the shell starts no session")
    XCTAssertLessThan(hold.lowerBound, item.lowerBound, "the lock comes before the glyph")
    XCTAssertLessThan(hold.lowerBound, session.lowerBound, "and before anything is read")

    let quit = try XCTUnwrap(
      text.range(of: "AppLock.decide("), "the shell never asks what the lock said")
    XCTAssertLessThan(quit.lowerBound, item.lowerBound, "a second instance draws nothing")

    let terminate = try XCTUnwrap(
      text.range(of: "func applicationWillTerminate"), "the shell has no terminate")
    let release = try XCTUnwrap(text.range(of: ".release()"), "the shell never lets go")
    XCTAssertGreaterThan(
      release.lowerBound, terminate.lowerBound, "the descriptor is held until the app ends")
    XCTAssertEqual(
      text.components(separatedBy: ".release()").count - 1, 1, "one release, and no second")
  }

  // MARK: this test's own machine

  private var lockFile: URL { root.appendingPathComponent("app.lock") }

  /// Take the lock and keep it until the test ends, so no descriptor outlives
  /// the case that opened it.
  private func take(at url: URL) -> AppLock {
    let lock = AppLock.hold(at: url)
    held.append(lock)
    return lock
  }

  /// This file is Tests/PresenceAppTests/LockTests.swift, so the package is
  /// three directories up from it.
  private var shellMain: URL {
    URL(fileURLWithPath: #filePath)
      .deletingLastPathComponent()
      .deletingLastPathComponent()
      .deletingLastPathComponent()
      .appendingPathComponent("Sources/Presence/main.swift")
  }
}
