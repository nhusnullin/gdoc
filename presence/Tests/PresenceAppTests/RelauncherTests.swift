// Reopening the app into the bundle on the disk, without an app launching.
//
// The launchpad is a fake that writes down what it was asked, and the two
// readers over the bundle, the clock and the sleep are closures this file
// holds. So nothing here opens an app, reads a signature or waits, and every
// guard the policy states is a fact handed in. The fake sleep is there for
// the new instance's own wait for the pid that launched it, which is the one
// wait left in the file.
//
// The rule under test here is the one the policy cannot hold on its own:
// that a reopen the policy held is asked again, and that the holding cost
// nothing. `gdoc update` swaps the bundle before it writes the run's end,
// so the first ask lands while the update is still live in the record
// almost every time.
//
// The second rule is that asking is free. The session asks the policy on
// every tick while its count runs, so `decides` has to write no stamp and
// open nothing, and only `reopen` may.

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

  func testTheOpenHappensOnTheAskWithNoSleepBeforeIt() async throws {
    // Arrange: every guard holds, and the sleep writes down what it was
    // asked to wait for.
    let pad = FakePad()
    let waits = Waits()
    let preferences = Preferences(store: Store())
    let relauncher = made(pad: pad, sleep: { waits.note($0) })

    // Act: the ask that opens.
    let answer = await relauncher.reopen(
      bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: false,
      remembering: preferences)

    // Assert: the new instance is up and this one quits, and the relauncher
    // waited for nothing on the way. The five seconds are the session's
    // count on the notice row, which has already been read down to zero by
    // the time this is asked.
    XCTAssertEqual(answer, .quit)
    XCTAssertEqual(pad.opened.count, 1)
    XCTAssertEqual(pad.opened.first?.arguments, [RelaunchPolicy.argument, "4242"])
    XCTAssertEqual(waits.all, [], "the open is asked for at zero and not two seconds later")
  }

  /// Asking the policy is free, so the session may ask it on every tick of
  /// its count: nothing is opened and nothing is written down.
  func testDecidingWritesNoStampAndOpensNothing() async throws {
    // Arrange: a relauncher over a bundle that is a newer build of this app.
    let pad = FakePad()
    let store = Store()
    let preferences = Preferences(store: store)
    let relauncher = made(pad: pad)

    // Act and assert: the clear ask, and the two holds that clear by
    // themselves, which are the gate the count reads.
    XCTAssertEqual(
      relauncher.decides(
        bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: false,
        remembering: preferences),
      .relaunch(version: "2.16.0"))
    XCTAssertEqual(
      relauncher.decides(
        bundle: .reopenNeeded("2.16.0"), updateRunning: true, childAlive: false,
        remembering: preferences),
      .hold(.updateRunning))
    XCTAssertEqual(
      relauncher.decides(
        bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: true,
        remembering: preferences),
      .hold(.childAlive))

    // And a hold that never clears, which is the gate that refuses for good.
    let unsigned = made(pad: pad, signatureValid: { _ in false })
    XCTAssertEqual(
      unsigned.decides(
        bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: false,
        remembering: preferences),
      .hold(.invalidSignature))

    // Assert: ten asks later the system has been asked nothing and the ten
    // minutes have not begun, so the one open the count asks for is still
    // the policy's to admit.
    for _ in 0..<10 {
      _ = relauncher.decides(
        bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: false,
        remembering: preferences)
    }
    XCTAssertEqual(pad.opened.count, 0, "deciding opens nothing")
    XCTAssertNil(preferences.lastRelaunch(), "and writes no stamp")
  }

  /// A hold that clears by itself reads no byte of the bundle.
  ///
  /// The session asks on every tick while a reopen is pending, and the
  /// pending window is the whole hold: a `gdoc update` stays live in the
  /// record after it swapped the bundle, and a sign-in child is up for
  /// minutes. The ask runs on the main actor, and a static signature over a
  /// universal bundle is tens of milliseconds, so a read per tick there is a
  /// dropped frame per second for as long as the hold lasts. The two guards
  /// the policy answers before it looks at the disk are answered without
  /// looking at the disk.
  func testAHeldDecisionReadsNoByteOfTheBundle() async throws {
    // Arrange: a relauncher whose two readers over the disk count their
    // calls.
    let reads = Reads()
    let preferences = Preferences(store: Store())
    let relauncher = made(
      pad: FakePad(),
      identifierOnDisk: { _ in
        reads.readTheIdentifier(); return Self.identifier
      },
      signatureValid: { _ in
        reads.readTheSignature(); return true
      })

    // Act: ten asks under each of the two holds that clear by themselves,
    // and the two bundle states that have nothing to open.
    for _ in 0..<10 {
      XCTAssertEqual(
        relauncher.decides(
          bundle: .reopenNeeded("2.16.0"), updateRunning: true, childAlive: false,
          remembering: preferences),
        .hold(.updateRunning))
      XCTAssertEqual(
        relauncher.decides(
          bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: true,
          remembering: preferences),
        .hold(.childAlive))
      XCTAssertEqual(
        relauncher.decides(
          bundle: .same, updateRunning: false, childAlive: false, remembering: preferences),
        .hold(.unchanged))
      XCTAssertEqual(
        relauncher.decides(
          bundle: .gone, updateRunning: false, childAlive: false, remembering: preferences),
        .hold(.gone))
    }

    // Assert: forty asks, and the disk was never touched.
    XCTAssertEqual(reads.identifiers, 0, "a held ask reads no plist")
    XCTAssertEqual(reads.signatures, 0, "and asks the kernel for no signature")

    // And the ask the count's zero is gated on still reads both, because
    // the policy cannot admit an open without them.
    XCTAssertEqual(
      relauncher.decides(
        bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: false,
        remembering: preferences),
      .relaunch(version: "2.16.0"))
    XCTAssertEqual(reads.identifiers, 1)
    XCTAssertEqual(reads.signatures, 1)
  }

  /// An open the system refused leaves the app up with the row asking for a
  /// hand, and the attempt is written down: an open that keeps failing is
  /// tried once in ten minutes and not on every tick.
  func testAFailedOpenKeepsTheRowAndWritesTheStamp() async throws {
    let pad = FakePad()
    pad.opens = false
    let preferences = Preferences(store: Store())
    let relauncher = made(pad: pad)

    let after = await relauncher.reopen(
      bundle: .reopenNeeded("2.16.0"), updateRunning: false, childAlive: false,
      remembering: preferences)

    XCTAssertEqual(after, .keepTheReopenRow)
    XCTAssertEqual(pad.opened.count, 1, "the system was asked once")
    XCTAssertEqual(
      preferences.lastRelaunch()?.version, "2.16.0",
      "and the ten minutes count from the attempt")
  }

  // MARK: the outside this file hands over

  private func made(
    pad: FakePad, sleep: @escaping @Sendable (TimeInterval) async -> Void = { _ in },
    identifierOnDisk: @escaping @Sendable (URL) -> String? = { _ in RelauncherTests.identifier },
    signatureValid: @escaping @Sendable (URL) -> Bool = { _ in true }
  ) -> Relauncher {
    Relauncher(
      bundle: URL(fileURLWithPath: "/Applications/GdocPresence.app"),
      identifier: Self.identifier,
      pid: 4242,
      pad: pad,
      identifierOnDisk: identifierOnDisk,
      signatureValid: signatureValid,
      now: { Date(timeIntervalSince1970: 1_792_000_000) },
      sleep: sleep)
  }

  /// How often each of the two readers over the bundle on the disk was
  /// called.
  private final class Reads: @unchecked Sendable {
    private let lock = NSLock()
    private var plists = 0
    private var seals = 0

    var identifiers: Int { lock.withLock { plists } }
    var signatures: Int { lock.withLock { seals } }

    func readTheIdentifier() { lock.withLock { plists += 1 } }
    func readTheSignature() { lock.withLock { seals += 1 } }
  }

  /// Every interval the relauncher asked to wait for.
  private final class Waits: @unchecked Sendable {
    private let lock = NSLock()
    private var asked: [TimeInterval] = []

    var all: [TimeInterval] { lock.withLock { asked } }

    func note(_ seconds: TimeInterval) { lock.withLock { asked.append(seconds) } }
  }

  /// What was asked of the system, and nothing done to it.
  private final class FakePad: Launchpad, @unchecked Sendable {
    struct Open: Equatable {
      let bundle: URL
      let arguments: [String]
    }

    private let lock = NSLock()
    private var asks: [Open] = []
    private var answer = true

    var opened: [Open] { lock.withLock { asks } }

    /// What the system answers the open with. True is an instance that came
    /// up; false is the open the row's last sentence is for.
    var opens: Bool {
      get { lock.withLock { answer } }
      set { lock.withLock { answer = newValue } }
    }

    func openNewInstance(at bundle: URL, arguments: [String]) async -> Bool {
      lock.withLock {
        asks.append(Open(bundle: bundle, arguments: arguments))
        return answer
      }
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
