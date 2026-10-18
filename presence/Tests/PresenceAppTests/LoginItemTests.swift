// Launch at login, as the four rules the specification states.
//
// No run registers anything. The app does not put itself in somebody's login
// items on its first launch, on a relaunch after an update or on any launch
// after that: the 2026-10-18 decision withdrew that registration, because a
// gdoc command now opens the app when it is not running, which is what the
// registration was for.
//
// So the switch in HEALTH is the only route to a registration. It records the
// person's answer, and macOS answering `requiresApproval` is a row and not a
// retry, because a retry is another notice in somebody's menu bar for a
// choice they have already been handed. What the switch shows is what macOS
// says, so an item removed in System Settings reads off here.
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

  func testTheFirstEverRunRegistersNothing() {
    // Arrange: nothing recorded, and nothing registered with the system,
    // which is a machine where the app has never run.
    let service = Service(registration: .notRegistered, after: .enabled)
    let preferences = Preferences(store: Store())

    // Act: the app comes up and the panel reads the row, twice, and a second
    // launch does the same.
    let item = LoginItem(service: service, preferences: preferences)
    let states = [
      item.state,
      item.state,
      LoginItem(service: service, preferences: preferences).state,
    ]

    // Assert: macOS was left as it was found, and nothing was written down.
    XCTAssertEqual(states, [.off, .off, .off])
    XCTAssertNil(preferences.loginItemChoice, "no run answers for the person")
    XCTAssertEqual(service.registrations, 0)
    XCTAssertEqual(service.unregistrations, 0)
  }

  func testTheSwitchRegistersUnregistersAndRecordsTheChoice() {
    // Arrange
    let service = Service(registration: .notRegistered, after: .enabled)
    let preferences = Preferences(store: Store())
    let item = LoginItem(service: service, preferences: preferences)

    // Act and assert: on.
    XCTAssertEqual(item.set(.on), .on)
    XCTAssertEqual(preferences.loginItemChoice, .on)
    XCTAssertEqual(service.registrations, 1)

    // Act and assert: off.
    XCTAssertEqual(item.set(.off), .off)
    XCTAssertEqual(preferences.loginItemChoice, .off)
    XCTAssertEqual(service.unregistrations, 1)
    XCTAssertEqual(service.registrations, 1, "and the system was asked once each way")
  }

  func testAPersonsOffIsNeverOverridden() {
    // Arrange: the HEALTH switch turned it off.
    let service = Service(registration: .enabled, after: .notRegistered)
    let preferences = Preferences(store: Store())
    let item = LoginItem(service: service, preferences: preferences)

    XCTAssertEqual(item.set(.off), .off)
    XCTAssertEqual(preferences.loginItemChoice, .off)
    XCTAssertEqual(service.unregistrations, 1)

    // Act: a launch, a relaunch after an update, and a launch after that.
    let states = [
      item.state,
      LoginItem(service: service, preferences: preferences).state,
      LoginItem(service: service, preferences: preferences).state,
    ]

    // Assert: off every time, and the system was never asked again.
    XCTAssertEqual(states, [.off, .off, .off])
    XCTAssertEqual(preferences.loginItemChoice, .off)
    XCTAssertEqual(service.registrations, 0)
    XCTAssertEqual(service.unregistrations, 1)
  }

  func testAnOnIsNeverMadeForThePerson() {
    // Arrange: a machine with no answer recorded, which is every machine
    // until somebody presses the switch.
    let service = Service(registration: .notRegistered, after: .enabled)
    let preferences = Preferences(store: Store())

    // Act: ten launches, each reading the row the panel draws.
    for _ in 0..<10 {
      _ = LoginItem(service: service, preferences: preferences).state
    }

    // Assert: no registration and no record. The one route to an "on" is the
    // switch, and it is the person's hand.
    XCTAssertEqual(service.registrations, 0)
    XCTAssertNil(preferences.loginItemChoice)
    XCTAssertEqual(LoginItem(service: service, preferences: preferences).set(.on), .on)
    XCTAssertEqual(service.registrations, 1)
  }

  func testTheSwitchShowsWhatMacOSSays() {
    // Arrange: every answer macOS has, and what the row makes of it.
    let answers: [(LoginRegistration, LoginItemState)] = [
      (.enabled, .on),
      (.notRegistered, .off),
      (.requiresApproval, .needsApproval),
      (.notFound, .off),
    ]

    for (registration, expected) in answers {
      let service = Service(registration: registration, after: registration)
      let item = LoginItem(service: service, preferences: Preferences(store: Store()))

      // Act and assert
      XCTAssertEqual(item.state, expected, "macOS said \(registration)")
      XCTAssertEqual(service.registrations, 0, "and reading it registers nothing")
    }

    // And a registration the person removed in System Settings reads off,
    // whatever this app wrote down when they switched it on.
    let service = Service(registration: .notRegistered, after: .enabled)
    let preferences = Preferences(store: Store())
    let item = LoginItem(service: service, preferences: preferences)

    XCTAssertEqual(item.set(.on), .on)
    service.becomes(.notRegistered)

    XCTAssertEqual(item.state, .off, "System Settings is the one that knows")
    XCTAssertEqual(preferences.loginItemChoice, .on, "their answer is still their answer")
    XCTAssertEqual(service.registrations, 1, "and nothing registered it again")
  }

  func testRequiresApprovalShowsTheRowAndIsNotRetried() {
    // Arrange: macOS takes the registration and then waits for the person.
    let service = Service(registration: .notRegistered, after: .requiresApproval)
    let preferences = Preferences(store: Store())
    let item = LoginItem(service: service, preferences: preferences)

    // Act: the switch pressed on.
    let first = item.set(.on)

    // Assert: the row, and the choice recorded.
    XCTAssertEqual(first, .needsApproval)
    XCTAssertEqual(preferences.loginItemChoice, .on)
    XCTAssertEqual(service.registrations, 1)

    // Act: every later launch, and the panel opened again.
    let later = [
      item.state,
      LoginItem(service: service, preferences: preferences).state,
    ]

    // Assert: the same row, and no second registration.
    XCTAssertEqual(later, [.needsApproval, .needsApproval])
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
    let item = LoginItem(service: service, preferences: preferences)

    // Act: the switch pressed on, and the panel read again after it.
    let state = item.set(.on)

    // Assert: the switch reads off, the choice is recorded, and the throw
    // was not a crash and not a loop.
    XCTAssertEqual(state, .off)
    XCTAssertEqual(preferences.loginItemChoice, .on)
    XCTAssertEqual(service.registrations, 1)
    XCTAssertEqual(item.state, .off)
    XCTAssertEqual(service.registrations, 1)
  }

  // MARK: a service of its own

  /// The system's register, as what was asked of it and what it answered.
  ///
  /// `after` is what the status becomes once a registration was taken, which
  /// is the whole of the difference between on and waiting for approval. An
  /// unregistration leaves nothing registered, as macOS does. `becomes` is
  /// System Settings: the one place a registration changes without this app
  /// asking for anything.
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

    /// Somebody changed it in System Settings.
    func becomes(_ registration: LoginRegistration) {
      lock.lock()
      defer { lock.unlock() }
      status = registration
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
      status = .notRegistered
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
