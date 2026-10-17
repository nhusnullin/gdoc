// Whether the pid a live row names is a watch the panel may signal.
//
// The record is a file anybody can write, and a pid in it is a number the
// kernel may have handed to somebody else since. Sending a signal to a
// number read out of such a file is the one thing in this app that could
// reach a program nobody asked it to reach, so the number is not believed:
// the kernel is asked what that pid is, and the answer has to agree with the
// record in every one of the ways below.
//
// This file decides and never acts. The facts come in already read, and the
// signal is Stopper's, in the app library, where the one `kill` with a
// signal lives. Nothing here opens a file or asks the kernel anything, so
// every refusal is reachable from a table.

import Foundation

/// What the kernel says about a pid: the program it is running, whose it is,
/// when it started, and whether it is there at all. A field the kernel would
/// not answer is absent, and an absent field is a refusal rather than a
/// guess.
public struct RunningProcess: Sendable, Equatable {
  public let path: String?
  public let uid: uid_t
  public let started: Date?
  public let isAlive: Bool

  public init(path: String?, uid: uid_t, started: Date?, isAlive: Bool) {
    self.path = path
    self.uid = uid
    self.started = started
    self.isAlive = isAlive
  }
}

/// The check that did not hold. The panel shows one sentence for all of
/// them, because none of them is something the person can act on; the reason
/// is here so a test can name the rule it is holding.
public enum StopRefusal: String, Sendable, Equatable, CaseIterable {
  /// Not a `comments --wait`. Nothing else gdoc runs is stoppable: only the
  /// wait traps the signal and still writes its end.
  case notAWait
  /// `via: chat`. Chat never waits, so a wait that says it did is a line no
  /// gdoc wrote.
  case fromChat
  /// The run already ended, so the pid is free for the next program.
  case ended
  /// No process by that number.
  case gone
  /// The kernel named no path for the pid.
  case pathUnknown
  /// The path is not the gdoc the app knows, nor the one `gdoc update`
  /// renamed beside it.
  case notGdoc
  /// The pid belongs to another user.
  case notOurUser
  /// The kernel named no start time for the pid.
  case startUnknown
  /// The pid started too long before the event, or after it: the number was
  /// reused.
  case outsideTheWindow
}

/// The answer, and nothing more: the caller sends the signal.
public enum StopVerdict: Sendable, Equatable {
  case stoppable
  case refused(StopRefusal)

  public var isStoppable: Bool {
    if case .stoppable = self { return true }
    return false
  }
}

public enum ProcessCheck {
  /// "Started within ten seconds before the event's time", from the
  /// specification's decision table. A wait writes its start as soon as it
  /// has one, so anything older is a different program at a reused number.
  public static let window: TimeInterval = 10

  /// The record writes RFC 3339 with no fraction of a second, so the moment
  /// it names is truncated by up to a second and a process that started
  /// inside that same second reads as having started after the event. One
  /// second of forward slack is that truncation and not a wider door.
  public static let truncation: TimeInterval = 1

  /// The suffix `gdoc update` leaves beside the new binary, so a wait that
  /// began before an update is still the gdoc the app knows.
  public static let previousSuffix = ".previous"

  /// The one sentence the panel shows for every refusal.
  public static let refusalLine = "This watch is not one gdoc presence can stop"

  /// The default resolver: symlinks followed, so the checkout route's link
  /// at `~/.local/bin/gdoc` and the binary it points at compare equal.
  public static func resolved(_ path: String) -> String {
    URL(fileURLWithPath: path).resolvingSymlinksInPath().path
  }

  /// Every check, in the order that reads the record before the kernel.
  ///
  /// - Parameters:
  ///   - wait: the start event the live row was folded from.
  ///   - hasEnd: whether that run and call already has its end.
  ///   - running: what the kernel says about `wait.pid`.
  ///   - gdoc: the binary the app itself runs, before resolution.
  ///   - user: the uid the app runs as.
  ///   - resolve: how a path is made comparable, injected so the decision
  ///     stays a pure function of its arguments.
  public static func verdict(
    wait: Event,
    hasEnd: Bool,
    running: RunningProcess,
    gdoc: String,
    user: uid_t,
    resolve: (String) -> String = ProcessCheck.resolved
  ) -> StopVerdict {
    guard wait.kind == .start, wait.command == "comments", wait.wait else {
      return .refused(.notAWait)
    }
    guard wait.via != "chat" else { return .refused(.fromChat) }
    guard !hasEnd else { return .refused(.ended) }
    guard running.isAlive else { return .refused(.gone) }

    guard let path = running.path else { return .refused(.pathUnknown) }
    let ours = resolve(gdoc)
    guard resolve(path) == ours || resolve(path) == ours + previousSuffix else {
      return .refused(.notGdoc)
    }
    guard running.uid == user else { return .refused(.notOurUser) }

    guard let started = running.started else { return .refused(.startUnknown) }
    let earliest = wait.at.addingTimeInterval(-window)
    let latest = wait.at.addingTimeInterval(truncation)
    guard started >= earliest, started <= latest else { return .refused(.outsideTheWindow) }

    return .stoppable
  }
}
