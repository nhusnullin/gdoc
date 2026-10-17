// The one room that opens an application, and it opens this app's own
// bundle.
//
// `gdoc update` swaps the bundle by rename under the running app, so what a
// person is left with is the old code and a row asking them to quit and
// reopen. This does it for them: it asks PresenceCore/RelaunchPolicy, waits
// the two seconds, opens the bundle on the disk as a new instance that is
// not brought to the front, and quits only once the open succeeded.
//
// Nothing is decided here. Every guard is the policy's, as a pure function
// over facts this file read: the identifier the bundle on the disk names, its
// static signature, whether an update is live, whether a child of this app's
// own is still up, and when the last attempt was. A source scan in
// PresenceCoreTests holds `openApplication` to this file, so the one
// application this app can start stays the one a person already installed.
//
// The new instance is handed `--relaunched-by <pid>` and nothing else. It
// waits up to five seconds for that pid to go and then quits if another
// instance of this identifier is still up, which is the whole of how two
// glyphs in the menu bar are kept from happening.

import AppKit
import Foundation
import PresenceCore
import Security

/// What the system is asked, as the two verbs a relaunch needs.
///
/// A protocol so a test can say what was asked without an app launching, and
/// so the signature and the identifier on the disk are facts handed in
/// rather than read inside a decision.
public protocol Launchpad: Sendable {
  /// Open this bundle as a new instance that is not activated, handing it
  /// these arguments. True where the system reported an instance.
  func openNewInstance(at bundle: URL, arguments: [String]) async -> Bool
  /// Whether another instance of this identifier is up, this one aside.
  func anotherInstance(of identifier: String) -> Bool
  /// Whether a pid is still there.
  func isAlive(_ pid: Int32) -> Bool
}

/// The system itself.
public struct SystemLaunchpad: Launchpad {
  public init() {}

  /// `createsNewApplicationInstance` because the bundle on the disk is a
  /// different build of an app that is already running, and `activates =
  /// false` because a relaunch is not something a person asked to look at.
  public func openNewInstance(at bundle: URL, arguments: [String]) async -> Bool {
    let configuration = NSWorkspace.OpenConfiguration()
    configuration.createsNewApplicationInstance = true
    configuration.activates = false
    configuration.arguments = arguments
    do {
      _ = try await NSWorkspace.shared.openApplication(at: bundle, configuration: configuration)
      return true
    } catch {
      return false
    }
  }

  public func anotherInstance(of identifier: String) -> Bool {
    let mine = ProcessInfo.processInfo.processIdentifier
    return NSRunningApplication.runningApplications(withBundleIdentifier: identifier)
      .contains { $0.processIdentifier != mine && !$0.isTerminated }
  }

  public func isAlive(_ pid: Int32) -> Bool { Liveness.isAlive(pid) }
}

/// Reopening this app into the bundle on the disk.
///
/// Every piece of the outside is injected: the launchpad, the two readers
/// over the bundle on the disk, the clock and the sleep. The app hands over
/// the real ones.
public struct Relauncher: Sendable {
  /// This app's own bundle on the disk, as the shell resolved it at launch.
  private let bundle: URL
  /// This app's own identifier.
  private let identifier: String
  /// This app's pid, which the new instance is told to wait for.
  private let pid: Int32
  private let pad: Launchpad
  private let identifierOnDisk: @Sendable (URL) -> String?
  private let signatureValid: @Sendable (URL) -> Bool
  private let now: @Sendable () -> Date
  private let sleep: @Sendable (TimeInterval) async -> Void

  public init(
    bundle: URL,
    identifier: String,
    pid: Int32 = ProcessInfo.processInfo.processIdentifier,
    pad: Launchpad = SystemLaunchpad(),
    identifierOnDisk: @escaping @Sendable (URL) -> String? = Relauncher.identifierRead,
    signatureValid: @escaping @Sendable (URL) -> Bool = Relauncher.signatureChecked,
    now: @escaping @Sendable () -> Date = Date.init,
    sleep: @escaping @Sendable (TimeInterval) async -> Void = Relauncher.slept
  ) {
    self.bundle = bundle
    self.identifier = identifier
    self.pid = pid
    self.pad = pad
    self.identifierOnDisk = identifierOnDisk
    self.signatureValid = signatureValid
    self.now = now
    self.sleep = sleep
  }

  // MARK: reopening

  /// Reopen the app where every guard holds, and say what the caller does
  /// next. The stamp is written before the open, so an open that keeps
  /// failing is tried once in ten minutes.
  public func reopen(
    bundle state: BundleState, updateRunning: Bool, childAlive: Bool,
    remembering preferences: Preferences
  ) async -> AfterOpen {
    let moment = now()
    let ask = RelaunchAsk(
      bundle: state,
      identifier: identifier,
      onDiskIdentifier: identifierOnDisk(bundle),
      signatureValid: signatureValid(bundle),
      updateRunning: updateRunning,
      childAlive: childAlive,
      last: preferences.lastRelaunch(now: moment),
      now: moment)
    guard case .relaunch(let version, let after) = RelaunchPolicy.decide(ask) else {
      return .keepTheReopenRow
    }

    preferences.noteRelaunch(version: version, at: moment)
    await sleep(after)
    let opened = await pad.openNewInstance(
      at: bundle, arguments: [RelaunchPolicy.argument, String(pid)])
    return RelaunchPolicy.after(opened: opened)
  }

  // MARK: coming up

  /// What this instance does about the one that launched it: wait for it to
  /// go, then quit if somebody else of this identifier is still up.
  ///
  /// An ordinary first launch names no launcher and asks once.
  public func settle(arguments: [String] = ProcessInfo.processInfo.arguments) async -> LaunchStep {
    let launcher = RelaunchPolicy.launcher(in: arguments)
    let began = now()
    while true {
      let step = RelaunchPolicy.step(
        launcher: launcher,
        launcherAlive: launcher.map { pad.isAlive($0) } ?? false,
        anotherInstance: pad.anotherInstance(of: identifier),
        waited: now().timeIntervalSince(began))
      guard case .waitForLauncher = step else { return step }
      await sleep(Self.asksEvery)
    }
  }

  // MARK: the outside

  /// How often the waiting instance asks again whether its launcher has
  /// gone. Short enough that a relaunch feels like one app and long enough
  /// that it is not a spin.
  static let asksEvery: TimeInterval = 0.25

  /// The identifier the bundle on the disk names now, read off its plist
  /// with no help from Bundle.main, which cached what it read at launch.
  public static let identifierRead: @Sendable (URL) -> String? = { bundle in
    let plist = bundle.appendingPathComponent("Contents/Info.plist")
    guard let data = try? Data(contentsOf: plist),
      let values = try? PropertyListSerialization.propertyList(
        from: data, options: [], format: nil) as? [String: Any]
    else { return nil }
    return values["CFBundleIdentifier"] as? String
  }

  /// Whether the bundle on the disk is internally consistent with its own
  /// signature. Ad-hoc is what `make app` and the release both produce, so
  /// this is the seal over the bytes and never a named authority.
  public static let signatureChecked: @Sendable (URL) -> Bool = { bundle in
    var code: SecStaticCode?
    guard SecStaticCodeCreateWithPath(bundle as CFURL, [], &code) == errSecSuccess,
      let code
    else { return false }
    let flags = SecCSFlags(rawValue: kSecCSCheckAllArchitectures)
    return SecStaticCodeCheckValidity(code, flags, nil) == errSecSuccess
  }

  /// The wait, as the one sleep this file does.
  public static let slept: @Sendable (TimeInterval) async -> Void = { seconds in
    try? await Task.sleep(nanoseconds: UInt64(max(0, seconds) * 1_000_000_000))
  }
}
