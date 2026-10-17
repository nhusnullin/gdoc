// Launch at login: the one room that names SMAppService.
//
// The app starts by itself, which is a product decision reversed: an app a
// person has to remember to open is an app that is not watching the thing it
// was installed to watch. So the first run ever registers this bundle as a
// login item and writes down that it did, and that is the whole of the
// automatic part.
//
// Everything after the first run is the person's. The HEALTH switch records
// their answer, and an "off" is never overridden: not by a later launch, not
// by a relaunch after `gdoc update` swapped the bundle, and not by this file
// noticing that macOS says the item is gone. A registration nobody asked for
// twice is a notice in somebody's menu bar twice.
//
// `requiresApproval` is macOS saying it took the registration and is waiting
// for the person to allow it in System Settings. It is a row with a quiet
// sentence and a button that opens that settings pane, and it is never
// retried: the choice is already recorded, so no later launch registers
// again. The pane is opened through SMAppService's own call rather than a
// settings URL, so the app's two schemes stay two.
//
// A source scan in PresenceCoreTests holds SMAppService to this file, so what
// the app can do to this machine's login items is a page one person can read.

import Foundation
import PresenceUI
import ServiceManagement

/// What macOS says about this app's own login item.
public enum LoginRegistration: Sendable, Equatable {
  case enabled
  case notRegistered
  /// It took the registration and is waiting for the person to allow it.
  case requiresApproval
  /// There is no login item to speak of, which a bundle macOS will not
  /// register answers.
  case notFound
}

/// The system's login-item register, as the four things the app needs of it.
///
/// A protocol so no test registers a login item on the machine it runs on.
public protocol LoginService: Sendable {
  /// Where the registration stands now.
  var registration: LoginRegistration { get }
  /// Ask for this bundle to be started at login.
  func register() throws
  /// Take that back.
  func unregister() throws
  /// Open the Login Items pane of System Settings.
  func openSettings()
}

/// The system itself, as `SMAppService.mainApp`.
public struct SystemLoginService: LoginService {
  public init() {}

  public var registration: LoginRegistration {
    switch SMAppService.mainApp.status {
    case .enabled: .enabled
    case .requiresApproval: .requiresApproval
    case .notRegistered: .notRegistered
    case .notFound: .notFound
    @unknown default: .notFound
    }
  }

  public func register() throws { try SMAppService.mainApp.register() }

  public func unregister() throws { try SMAppService.mainApp.unregister() }

  public func openSettings() { SMAppService.openSystemSettingsLoginItems() }
}

/// Whether this app starts itself at login, as the panel's own switch and the
/// one registration the first run makes.
public struct LoginItem: Sendable {
  private let service: LoginService
  private let preferences: Preferences
  /// Whether this instance was opened by another instance of itself. A
  /// relaunch is never somebody's first run.
  private let relaunched: Bool

  public init(
    service: LoginService = SystemLoginService(),
    preferences: Preferences,
    relaunched: Bool
  ) {
    self.service = service
    self.preferences = preferences
    self.relaunched = relaunched
  }

  /// What the app does about launching at login when it comes up, and the
  /// row that comes out of it.
  ///
  /// One registration, on the first run ever, where this instance is not a
  /// relaunch. Every other launch reads and asks the system for nothing.
  @discardableResult public func settle() -> LoginItemState {
    guard !relaunched, preferences.loginItemChoice == nil else { return state }
    // Recorded before the call and whatever the call answers: the record is
    // what stops a second registration, and a registration that was refused
    // or that is waiting for the person is still a registration this app has
    // already made.
    preferences.loginItemChoice = .on
    try? service.register()
    return state
  }

  /// The HEALTH switch. The answer is the person's, so it is written down
  /// before the system is asked, and an "off" stops every later launch from
  /// registering again.
  @discardableResult public func set(_ choice: LoginItemChoice) -> LoginItemState {
    preferences.loginItemChoice = choice
    switch choice {
    case .on: try? service.register()
    case .off: try? service.unregister()
    }
    return state
  }

  /// The row that `requiresApproval` draws, pressed.
  public func openSettings() {
    service.openSettings()
  }

  /// Where the switch stands, as the panel reads it.
  ///
  /// The person's "off" is the answer where they gave one. Otherwise it is
  /// what macOS says, so an item somebody removed in System Settings reads
  /// off here rather than claiming an "on" that is not true.
  public var state: LoginItemState {
    guard preferences.loginItemChoice != .off else { return .off }
    switch service.registration {
    case .requiresApproval: return .needsApproval
    case .enabled: return .on
    case .notRegistered, .notFound: return .off
    }
  }
}
