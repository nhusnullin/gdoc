// Launch at login, as the four rules the specification states.
//
// The app registers itself once, on the first run ever, and never again: a
// person who turned it off is not asked again by an update, a relaunch or a
// second launch. macOS answering `requiresApproval` is a row and not a retry,
// because a retry is another notice in somebody's menu bar for a choice they
// have already been handed.
//
// The service is injected, so no test here registers a login item on the
// machine it runs on, and the store is the in-memory one, so no test writes
// to that machine's defaults.

import Foundation
import PresenceCore
import PresenceUI
import XCTest

@testable import PresenceApp

final class LoginItemTests: XCTestCase {

  func testTheFirstEverRunRegistersOnceAndRecordsOn() {
    // Arrange: nothing recorded, and nothing registered with the system.
    let service = Service(registration: .notRegistered, after: .enabled)
    let preferences = Preferences(store: Store())

    // Act
    let item = LoginItem(service: service, preferences: preferences, relaunched: false)
    let state = item.settle()

    // Assert: on, recorded, and asked of the system exactly once.
    XCTAssertEqual(state, .on)
    XCTAssertEqual(preferences.loginItemChoice, .on)
    XCTAssertEqual(service.registrations, 1)
    XCTAssertEqual(service.unregistrations, 0)

    // And the second launch registers nothing, because the choice is there.
    XCTAssertEqual(item.settle(), .on)
    XCTAssertEqual(service.registrations, 1)
  }

  func testAPersonsOffIsNeverOverridden() {
    // Arrange: the HEALTH switch turned it off.
    let service = Service(registration: .enabled, after: .notRegistered)
    let preferences = Preferences(store: Store())
    let item = LoginItem(service: service, preferences: preferences, relaunched: false)

    XCTAssertEqual(item.set(.off), .off)
    XCTAssertEqual(preferences.loginItemChoice, .off)
    XCTAssertEqual(service.unregistrations, 1)

    // Act: a launch, a relaunch after an update, and a launch after that.
    let states = [
      item.settle(),
      LoginItem(service: service, preferences: preferences, relaunched: true).settle(),
      LoginItem(service: service, preferences: preferences, relaunched: false).settle(),
    ]

    // Assert: off every time, and the system was never asked again.
    XCTAssertEqual(states, [.off, .off, .off])
    XCTAssertEqual(preferences.loginItemChoice, .off)
    XCTAssertEqual(service.registrations, 0)
    XCTAssertEqual(service.unregistrations, 1)
  }

  func testARelaunchNeverRegisters() {
    // Arrange: a relaunch that carries no recorded choice, which is what an
    // update into a store somebody cleared looks like.
    let service = Service(registration: .notRegistered, after: .enabled)
    let preferences = Preferences(store: Store())

    // Act
    let item = LoginItem(service: service, preferences: preferences, relaunched: true)
    let state = item.settle()

    // Assert: nothing registered and nothing recorded. A relaunch is this
    // app opening itself, so it is never somebody's first run.
    XCTAssertEqual(state, .off)
    XCTAssertNil(preferences.loginItemChoice)
    XCTAssertEqual(service.registrations, 0)

    // The same arguments decide it in the app, which is the launcher the
    // relaunch names.
    XCTAssertNotNil(RelaunchPolicy.launcher(in: [RelaunchPolicy.argument, "6000"]))
    XCTAssertNil(RelaunchPolicy.launcher(in: ["/Applications/GdocPresence.app"]))
  }

  func testRequiresApprovalShowsTheRowAndIsNotRetried() {
    // Arrange: macOS takes the registration and then waits for the person.
    let service = Service(registration: .notRegistered, after: .requiresApproval)
    let preferences = Preferences(store: Store())
    let item = LoginItem(service: service, preferences: preferences, relaunched: false)

    // Act
    let first = item.settle()

    // Assert: the row, and the choice recorded so nothing asks again.
    XCTAssertEqual(first, .needsApproval)
    XCTAssertEqual(preferences.loginItemChoice, .on)
    XCTAssertEqual(service.registrations, 1)

    // Act: every later launch, and a relaunch after an update.
    let later = [
      item.settle(),
      item.state,
      LoginItem(service: service, preferences: preferences, relaunched: true).settle(),
    ]

    // Assert: the same row, and no second registration.
    XCTAssertEqual(later, [.needsApproval, .needsApproval, .needsApproval])
    XCTAssertEqual(service.registrations, 1)

    // The row's own action opens the Login Items settings, and nothing else
    // in this app can.
    XCTAssertEqual(service.settingsOpened, 0)
    item.openSettings()
    XCTAssertEqual(service.settingsOpened, 1)
  }

  func testARefusedRegistrationIsAnOffRowAndNotARetry() {
    // Arrange: a build macOS will not register at all, which is what a
    // bundle outside /Applications can be.
    let service = Service(registration: .notRegistered, after: .notFound, refuses: true)
    let preferences = Preferences(store: Store())
    let item = LoginItem(service: service, preferences: preferences, relaunched: false)

    // Act
    let state = item.settle()

    // Assert: the switch reads off, the choice is recorded, and the throw
    // was not a crash and not a loop.
    XCTAssertEqual(state, .off)
    XCTAssertEqual(preferences.loginItemChoice, .on)
    XCTAssertEqual(service.registrations, 1)
    XCTAssertEqual(item.settle(), .off)
    XCTAssertEqual(service.registrations, 1)
  }

  // MARK: a service of its own

  /// The system's register, as what was asked of it and what it answered.
  ///
  /// `after` is what the status becomes once a registration was taken, which
  /// is the whole of the difference between on and waiting for approval.
  private final class Service: LoginService, @unchecked Sendable {
    private let lock = NSLock()
    private var status: LoginRegistration
    private let after: LoginRegistration
    private let refuses: Bool
    private var registered = 0
    private var unregistered = 0
    private var opened = 0

    init(
      registration: LoginRegistration, after: LoginRegistration, refuses: Bool = false
    ) {
      self.status = registration
      self.after = after
      self.refuses = refuses
    }

    var registrations: Int {
      lock.lock()
      defer { lock.unlock() }
      return registered
    }

    var unregistrations: Int {
      lock.lock()
      defer { lock.unlock() }
      return unregistered
    }

    var settingsOpened: Int {
      lock.lock()
      defer { lock.unlock() }
      return opened
    }

    var registration: LoginRegistration {
      lock.lock()
      defer { lock.unlock() }
      return status
    }

    func register() throws {
      lock.lock()
      defer { lock.unlock() }
      registered += 1
      status = after
      if refuses { throw Refusal.macOSSaidNo }
    }

    func unregister() throws {
      lock.lock()
      defer { lock.unlock() }
      unregistered += 1
      status = after
    }

    func openSettings() {
      lock.lock()
      defer { lock.unlock() }
      opened += 1
    }

    enum Refusal: Error { case macOSSaidNo }
  }

  /// The app's own state, in memory, so no test touches this machine's
  /// defaults.
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
