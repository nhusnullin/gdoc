// Reopening the app into the bundle on the disk, without an app launching.
//
// The launchpad is a fake that writes down what it was asked, and the two
// readers over the bundle, the clock and the sleep are closures this file
// holds. So nothing here opens an app, reads a signature or waits two
// seconds, and every guard the policy states is a fact handed in.
//
// The rule under test here is the one the policy cannot hold on its own:
// that a reopen the policy held is asked again, and that the holding cost
// nothing. `gdoc update` swaps the bundle before it writes the run's end,
// so the first ask lands while the update is still live in the record
// almost every time.

import Foundation
import PresenceCore
import XCTest

@testable import PresenceApp

final class RelauncherTests: XCTestCase {
  private static let identifier = "com.altery.gdoc.presence"

  func testAHeldReopenCostsNothingAndTheNextAskOpensTheNewInstance() async throws {
    // Arrange: the bundle on the disk is a newer build of this app, and a
    // `gdoc update` is still live in the record.
    let pad = FakePad()
    let store = Store()
    let preferences = Preferences(store: store)
    let relauncher = made(pad: pad)

    // Act: the ask the bundle's own change makes, while the update runs.
    let held = await relauncher.reopen(
      bundle: .reopenNeeded("2.16.0"), updateRunning: true, childAlive: false,
      remembering: preferences)

    // Assert: nothing opened, and nothing was written down either, so the
    // ten-minute guard does not close the door on the ask that follows.
    XCTAssertEqual(held, .keepTheReopenRow)
    XCTAssertEqual(pad.opened.count, 0, "an update still running opens nothing")
    XCTAssertNil(preferences.lastRelaunch(), "a held ask writes no stamp")

    // Act: the same ask once the update and the child are gone.
    let again = await relauncher.reopen(
      bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: false,
      remembering: preferences)

    // Assert: the new instance is up, this one quits, and it is written down.
    XCTAssertEqual(again, .quit)
    XCTAssertEqual(pad.opened.count, 1)
    XCTAssertEqual(
      pad.opened.first?.arguments, [RelaunchPolicy.argument, "4242"],
      "the new instance is told whose going to wait for")
    XCTAssertEqual(preferences.lastRelaunch()?.version, "2.16.0")

    // And a third ask inside the ten minutes opens nothing: asking again is
    // bounded by the stamp and not by nobody asking.
    let third = await relauncher.reopen(
      bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: false,
      remembering: preferences)
    XCTAssertEqual(third, .keepTheReopenRow)
    XCTAssertEqual(pad.opened.count, 1, "one attempt per version in ten minutes")
  }

  func testAChildOfThisAppsOwnHoldsTheReopenAndThenLetsItGo() async throws {
    let pad = FakePad()
    let preferences = Preferences(store: Store())
    let relauncher = made(pad: pad)

    let held = await relauncher.reopen(
      bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: true,
      remembering: preferences)
    XCTAssertEqual(held, .keepTheReopenRow)
    XCTAssertEqual(pad.opened.count, 0, "quitting would take the child with us")
    XCTAssertNil(preferences.lastRelaunch())

    let again = await relauncher.reopen(
      bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: false,
      remembering: preferences)
    XCTAssertEqual(again, .quit)
    XCTAssertEqual(pad.opened.count, 1)
  }

  // MARK: the outside this file hands over

  private func made(pad: FakePad) -> Relauncher {
    Relauncher(
      bundle: URL(fileURLWithPath: "/Applications/GdocPresence.app"),
      identifier: Self.identifier,
      pid: 4242,
      pad: pad,
      identifierOnDisk: { _ in Self.identifier },
      signatureValid: { _ in true },
      now: { Date(timeIntervalSince1970: 1_792_000_000) },
      sleep: { _ in })
  }

  /// What was asked of the system, and nothing done to it.
  private final class FakePad: Launchpad, @unchecked Sendable {
    struct Open: Equatable {
      let bundle: URL
      let arguments: [String]
    }

    private let lock = NSLock()
    private var asks: [Open] = []

    var opened: [Open] { lock.withLock { asks } }

    func openNewInstance(at bundle: URL, arguments: [String]) async -> Bool {
      lock.withLock { asks.append(Open(bundle: bundle, arguments: arguments)) }
      return true
    }

    func anotherInstance(of identifier: String) -> Bool { false }

    func isAlive(_ pid: Int32) -> Bool { false }
  }

  /// The defaults in memory, so no test writes to this machine's.
  private final class Store: DefaultsStore {
    private var values: [String: Any] = [:]

    func object(forKey key: String) -> Any? { values[key] }

    func set(_ value: Any?, forKey key: String) {
      guard let value else {
        values.removeValue(forKey: key)
        return
      }
      values[key] = value
    }

    func removeObject(forKey key: String) { values.removeValue(forKey: key) }
  }
}
