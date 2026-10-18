// Launch at login: the one room that names SMAppService.
//
// It is a setting and nothing else. No run of this app registers it: not the
// first run ever, not a relaunch after `gdoc update` swapped the bundle, and
// not a launch that notices macOS says the item is gone. M24 registered on
// the first run, and the 2026-10-18 decision withdrew that: a gdoc command
// now opens the app when it is not running, which is what that registration
// was for, and product decision 32 stands again.
//
// So the HEALTH switch is the one route to a registration. It records the
// person's answer, because their answer is theirs, and it asks the system
// once each way.
//
// What the switch shows is what macOS says. An item somebody removed in
// System Settings reads off here rather than claiming an "on" out of this
// app's own memory, and nothing re-registers it: the record is the answer
// they gave, not a promise about this machine.
//
// `requiresApproval` is macOS saying it took the registration and is waiting
// for the person to allow it in System Settings. It is a row with a quiet
// sentence and a button that opens that settings pane, and it is never
// retried: the choice is already recorded, so a second registration would be
// a second notice in somebody's menu bar for an answer nobody changed. The
// pane is opened through SMAppService's own call rather than a settings URL,
// so the app's two schemes stay two.
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

/// Whether this app starts itself at login, as the panel's own switch.
public struct LoginItem: Sendable {
  private let service: LoginService
  private let preferences: Preferences

  public init(
    service: LoginService = SystemLoginService(),
    preferences: Preferences
  ) {
    self.service = service
    self.preferences = preferences
  }

  /// The HEALTH switch, and the one thing in this app that registers
  /// anything. The answer is the person's, so it is written down before the
  /// system is asked: a registration macOS refused or is waiting on is still
  /// the answer they gave.
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

  /// Where the switch stands, as the panel reads it: what macOS says, asked
  /// every time the facts are taken.
  ///
  /// Nothing of this app's own memory is in it. An item somebody removed in
  /// System Settings reads off, and one they added by hand reads on, because
  /// the register is the thing that decides whether this app starts at
  /// login: testTheSwitchShowsWhatMacOSSays.
  public var state: LoginItemState {
    switch service.registration {
    case .requiresApproval: return .needsApproval
    case .enabled: return .on
    case .notRegistered, .notFound: return .off
    }
  }
}
