// The one lock, which is how anybody knows this app is running.
//
// gdoc opens the app when the app is not open, and the question it asks is
// this file's answer: the app holds `flock(LOCK_EX)` on `presence/app.lock`
// in the config directory for its whole life, and the kernel lets go when
// the process dies, however it dies. gdoc asks for the same lock without
// blocking, lets go at once and reads no byte of the file; a file that is
// not there is an app that never ran. A pid written down would be a number
// the kernel reuses; a lock is not reused, which is the whole reason it is
// this and not a pid file. Decided 2026-10-18, DECISIONS.md.
//
// The lock is also what keeps two glyphs out of the menu bar: an instance
// that does not get it quits before it draws anything. The one instance that
// waits is the one another instance opened after an update, which carries
// `--relaunched-by <pid>` and whose launcher quits as soon as it is up: it
// asks again for the launcher's own patience, which is PresenceCore's, and
// yields if the launcher never goes.
//
// This file writes the only file this app ever writes into gdoc's directory,
// and it writes no byte into it: the file exists so there is something to
// lock.

import Darwin
import Foundation
import PresenceCore

/// What this instance found when it reached for the lock.
public enum LockStanding: Sendable, Equatable {
  /// This process holds it, and keeps it until it dies.
  case mine
  /// Another process holds it, so there is an app in the menu bar already.
  case another
  /// The file could not be opened at all, so nothing is known about who is
  /// running. Nothing is refused on a guess.
  case unknown
}

/// What a just-launched instance does with what the lock said.
public enum AfterTheLock: Sendable, Equatable {
  /// Draw the status item and start reading.
  case run
  /// Leave at once, having drawn nothing.
  case quit
}

/// The lock this app holds for its life.
///
/// Unchecked Sendable because the one thing that changes is the released
/// flag, which is behind its lock. The descriptor is never closed by anything
/// but `release`, which the shell calls when the app terminates.
public final class AppLock: @unchecked Sendable {
  /// What the reach for the lock answered.
  public let standing: LockStanding

  /// How often a waiting instance asks again. Short enough that a relaunch
  /// feels like one app, long enough that it is not a spin.
  public static let asksEvery: TimeInterval = 0.25

  /// The mode the file is made with where it is made at all. It holds no
  /// bytes, and the lock is the kernel's rather than the file's, so it is an
  /// ordinary file nobody but its owner writes.
  public static let mode: mode_t = 0o644

  private let descriptor: Int32?
  private let gate = NSLock()
  private var releasedAlready = false

  private init(standing: LockStanding, descriptor: Int32?) {
    self.standing = standing
    self.descriptor = descriptor
  }

  /// Reach for the lock, waiting only as long as the caller says.
  ///
  /// The sleep is injected so a test proves the waiting without spending it.
  /// Nothing is read from the file and nothing is written to it; the
  /// directory is made where it is missing, because an app that could not
  /// lock would be an app gdoc opens again on every command.
  public static func hold(
    at url: URL,
    waitingUpTo patience: TimeInterval = 0,
    sleep: (TimeInterval) -> Void = { Thread.sleep(forTimeInterval: $0) }
  ) -> AppLock {
    try? FileManager.default.createDirectory(
      at: url.deletingLastPathComponent(), withIntermediateDirectories: true)

    var left = asks(within: patience)
    while true {
      switch reach(for: url) {
      case .success(let descriptor):
        return AppLock(standing: .mine, descriptor: descriptor)
      case .failure(.unknown):
        return AppLock(standing: .unknown, descriptor: nil)
      case .failure(.another):
        left -= 1
        guard left > 0 else {
          return AppLock(standing: .another, descriptor: nil)
        }
        sleep(asksEvery)
      }
    }
  }

  /// How many times an instance asks for the lock inside its patience. One
  /// ask where it waits none, and one more for every wait that fits, so the
  /// count is a number a test can state rather than a clock it has to spend.
  public static func asks(within patience: TimeInterval) -> Int {
    guard patience > 0 else { return 1 }
    return Int((patience / asksEvery).rounded(.down)) + 1
  }

  /// Let the lock go. The shell calls this when the app terminates, and
  /// nothing else calls it: a descriptor closed earlier is an app gdoc would
  /// open a second time while this one is still in the menu bar.
  public func release() {
    gate.lock()
    defer { gate.unlock() }
    guard !releasedAlready, let descriptor else { return }
    releasedAlready = true
    flock(descriptor, LOCK_UN)
    close(descriptor)
  }

  /// What the instance does, given what the lock said.
  public static func decide(_ standing: LockStanding) -> AfterTheLock {
    standing == .another ? .quit : .run
  }

  /// How long this instance waits for the lock. An instance another instance
  /// opened waits for the launcher's patience, because that launcher quits
  /// as soon as this one is up; a second instance anybody else started waits
  /// none and leaves.
  public static func patience(ofAnInstanceWith arguments: [String]) -> TimeInterval {
    RelaunchPolicy.launcher(in: arguments) == nil ? 0 : RelaunchPolicy.patience
  }

  // MARK: the kernel

  /// Why a reach for the lock came back empty.
  private enum NotMine: Error {
    case another
    case unknown
  }

  /// One open and one non-blocking `flock`. A flock belongs to the open file
  /// and not to the process, so a second open of the same path is refused
  /// while the first holds it, whoever opened it.
  private static func reach(for url: URL) -> Result<Int32, NotMine> {
    let descriptor = open(url.path, O_CREAT | O_RDWR | O_CLOEXEC, mode)
    guard descriptor >= 0 else { return .failure(.unknown) }
    guard flock(descriptor, LOCK_EX | LOCK_NB) == 0 else {
      close(descriptor)
      return .failure(.another)
    }
    return .success(descriptor)
  }
}
