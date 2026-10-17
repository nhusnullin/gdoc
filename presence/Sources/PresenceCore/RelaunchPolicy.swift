// Whether the app may open its own bundle again, and what the instance that
// comes up does about the one that opened it.
//
// `gdoc update` swaps this app's bundle by rename while the app is running.
// The code already mapped keeps running, so what a person has is the old app
// and a row asking them to quit and reopen. The app can do that itself, and
// the whole of the risk is in doing it twice or doing it into something else:
// two instances are two glyphs in the menu bar, and a bundle somebody swapped
// in is a program nobody asked this app to start.
//
// So the deciding is here, as a pure function of facts the shell read, and
// the opening is in PresenceApp/Relauncher, which is the one other room that
// may ask the system to open anything. Every guard below is one row of a
// table in RelaunchPolicyTests.
//
// The stamp is written when the attempt is made and not when it worked: an
// open that keeps failing is then tried once in ten minutes rather than
// every time the watcher fires.

import Foundation

/// The last time the app opened its own bundle again, and the version on the
/// disk it did it for.
///
/// The ten-minute guard is counted from here. It is a value rather than the
/// store it comes out of, so the decision stays a function of its arguments:
/// PresenceApp/Preferences holds the key and hands this over.
public struct RelaunchStamp: Sendable, Equatable {
  public let version: String
  public let at: Date

  public init(version: String, at: Date) {
    self.version = version
    self.at = at
  }
}

/// Everything the shell knows when it asks whether to reopen itself.
public struct RelaunchAsk: Sendable, Equatable {
  /// Whether the bundle on the disk is still the one that is running, which
  /// is `BundleVersion.check` over the plist read off the disk.
  public let bundle: BundleState
  /// This app's own bundle identifier.
  public let identifier: String
  /// The identifier the bundle on the disk names now, or nothing where it
  /// names none.
  public let onDiskIdentifier: String?
  /// Whether the static signature of the bundle on the disk checked out.
  public let signatureValid: Bool
  /// Whether a `gdoc update` run is live in the record. It may be writing
  /// the bytes this relaunch would run.
  public let updateRunning: Bool
  /// Whether a child this app started is still up. Quitting would take it
  /// with us.
  public let childAlive: Bool
  /// The last attempt, where there is one.
  public let last: RelaunchStamp?
  public let now: Date

  public init(
    bundle: BundleState,
    identifier: String,
    onDiskIdentifier: String?,
    signatureValid: Bool,
    updateRunning: Bool,
    childAlive: Bool,
    last: RelaunchStamp?,
    now: Date
  ) {
    self.bundle = bundle
    self.identifier = identifier
    self.onDiskIdentifier = onDiskIdentifier
    self.signatureValid = signatureValid
    self.updateRunning = updateRunning
    self.childAlive = childAlive
    self.last = last
    self.now = now
  }
}

/// Why the app is not reopening itself. The panel shows none of these words:
/// what a person sees is the reopen row, which stays. A test names the rule
/// it is holding.
public enum RelaunchHold: String, Sendable, Equatable, CaseIterable {
  /// The disk holds the version that is running.
  case unchanged
  /// There is no bundle to open.
  case gone
  /// A `gdoc update` run is live.
  case updateRunning
  /// A child of this app's own has not exited.
  case childAlive
  /// The bundle on the disk is not this app.
  case anotherIdentifier
  /// Its static signature did not check out.
  case invalidSignature
  /// It was already tried for this version on the disk, inside the guard.
  case alreadyTried
}

/// The answer, and nothing more: the opening is the caller's.
public enum RelaunchDecision: Sendable, Equatable {
  /// Open the bundle on the disk, after this many seconds, for the version
  /// the disk names.
  case relaunch(version: String, after: TimeInterval)
  case hold(RelaunchHold)
}

/// What the instance that opened the new one does next.
public enum AfterOpen: Sendable, Equatable {
  /// The new instance is up, so this one quits.
  case quit
  /// Nothing opened, so the app stays up and HEALTH still says quit and
  /// reopen.
  case keepTheReopenRow
}

/// What an instance does about the one that launched it, asked again as it
/// waits.
public enum LaunchStep: Sendable, Equatable {
  /// The instance that opened this one is still up: wait for it, for this
  /// much of the patience that is left.
  case waitForLauncher(pid: Int32, upTo: TimeInterval)
  /// Another instance of this identifier is up, so this one quits.
  case quit
  /// Nobody else is up: this instance is the app.
  case carryOn
}

public enum RelaunchPolicy {
  /// How long the app waits after it decides, so the rename that changed the
  /// bundle has finished arriving.
  public static let settle: TimeInterval = 2

  /// At most one attempt per version on the disk in ten minutes.
  public static let once: TimeInterval = 600

  /// How long a new instance waits for the pid that launched it before it
  /// stops waiting.
  public static let patience: TimeInterval = 5

  /// The argument the new instance is given, so it knows whose going it is
  /// waiting for.
  public static let argument = "--relaunched-by"

  /// Whether to reopen, in the order that reads the disk before the clock.
  ///
  /// `.gone` is first because there is nothing to open, and the two reasons
  /// to wait come before the two about what the bundle is: a person whose
  /// update is still running is told nothing either way, and asking the
  /// kernel for a signature while the bytes are being written is asking
  /// about a file that is half there.
  public static func decide(_ ask: RelaunchAsk) -> RelaunchDecision {
    switch ask.bundle {
    case .same: return .hold(.unchanged)
    case .gone: return .hold(.gone)
    case .reopenNeeded(let version):
      if ask.updateRunning { return .hold(.updateRunning) }
      if ask.childAlive { return .hold(.childAlive) }
      guard let onDisk = ask.onDiskIdentifier, onDisk == ask.identifier else {
        return .hold(.anotherIdentifier)
      }
      guard ask.signatureValid else { return .hold(.invalidSignature) }
      if let last = ask.last, last.version == version,
        ask.now.timeIntervalSince(last.at) < once
      {
        return .hold(.alreadyTried)
      }
      return .relaunch(version: version, after: settle)
    }
  }

  /// What the instance that asked for the open does with the answer.
  public static func after(opened: Bool) -> AfterOpen {
    opened ? .quit : .keepTheReopenRow
  }

  /// The pid that launched this instance, where the argument names a
  /// plausible one. A zero or a negative number is no pid at all: the one
  /// place a number is used is a question about whether that process is
  /// still there.
  public static func launcher(in arguments: [String]) -> Int32? {
    guard let mark = arguments.firstIndex(of: argument) else { return nil }
    let next = arguments.index(after: mark)
    guard next < arguments.endIndex, let pid = Int32(arguments[next]), pid > 0 else {
      return nil
    }
    return pid
  }

  /// Whether this instance waits, quits or carries on.
  ///
  /// A launcher that stays up past the patience is an instance that did not
  /// quit, so this one yields rather than leaving two glyphs in the bar: the
  /// one that stayed keeps its reopen row and the person presses it again.
  public static func step(
    launcher: Int32?, launcherAlive: Bool, anotherInstance: Bool, waited: TimeInterval
  ) -> LaunchStep {
    if let launcher, launcherAlive, waited < patience {
      return .waitForLauncher(pid: launcher, upTo: patience - waited)
    }
    return anotherInstance ? .quit : .carryOn
  }
}
