// The one room that sends a signal.
//
// A live row's Stop asks a process somebody else started to stop waiting.
// That is the only thing this app does that reaches outside itself, and the
// pid it reaches is a number out of a file anybody can write, so nothing
// here believes the file: PresenceCore/ProcessCheck asks the kernel what
// that pid is and refuses unless every claim the record makes about it
// agrees. This file does the asking, the comparing and the sending, in that
// order, and nothing else in the package may send a signal at all: a source
// scan in PresenceCoreTests holds it here.
//
// One signal per wait, ever. `gdoc comments --wait` traps SIGINT and still
// writes its end, which is the whole reason Stop can exist; a second signal
// would be a stop nobody trusts, and a stronger one would take the end away.
// So a wait that has been signalled is remembered until its end is in the
// record, the row draws a ring instead of a second Stop, and a press on it
// sends nothing.
//
// What a refusal looks like to a person is one sentence, which is
// `ProcessCheck.refusalLine` and the panel's `Tips.cannotStop`. The reason
// stays here for a test: none of the reasons is something anybody can act
// on, and a sentence naming a pid's start time would be a sentence about the
// app rather than about the document.

import Darwin
import Foundation
import PresenceCore

/// Where one live row's Stop stands.
public enum StopStanding: Sendable, Equatable {
  /// Every check holds, so the icon is live.
  case ready
  /// One signal has gone to this wait and its end has not arrived.
  case signalled
  /// The check refused, so the icon is off with the one sentence.
  case refused(StopRefusal)
  /// The app never found a gdoc, so there is no path to compare the pid's
  /// against and nothing is sent.
  case noGdoc
}

/// Stopping a wait: the check, the one signal, and the memory of it.
///
/// Unchecked Sendable because the one thing it remembers is behind its lock.
/// Both halves of the kernel are injected, so no test signals anything.
public final class Stopper: @unchecked Sendable {
  /// The one signal. `comments --wait` traps it at
  /// `go/cmd/gdoc/read.go:589` and answers `data.waited.interrupted`, so the
  /// run still ends the way the skill reads.
  public static let signal = SIGINT

  private let gdoc: String?
  private let user: uid_t
  private let inspect: (Int32) -> RunningProcess
  private let send: (Int32, Int32) -> Bool
  private let resolve: (String) -> String

  private let lock = NSLock()
  /// The waits a signal has gone to, by their `run|call`. A wait leaves this
  /// list when the record pairs it with an end, which `pendingDocs` reads;
  /// nothing here is written to disk, because a stop that outlived the
  /// process it was aimed at is not a stop.
  private var signalled: Set<String> = []

  public init(
    gdoc: String?,
    user: uid_t = getuid(),
    inspect: @escaping (Int32) -> RunningProcess = Stopper.ask,
    send: @escaping (Int32, Int32) -> Bool = Stopper.sent,
    resolve: @escaping (String) -> String = ProcessCheck.resolved
  ) {
    self.gdoc = gdoc
    self.user = user
    self.inspect = inspect
    self.send = send
    self.resolve = resolve
  }

  // MARK: what the panel asks

  /// Where Stop stands for one document, without sending anything.
  public func standing(doc: String, in events: [Event]) -> StopStanding {
    guard let gdoc else { return .noGdoc }
    guard let (wait, hasEnd) = Self.latestWait(doc: doc, in: events) else {
      return .refused(.notAWait)
    }
    if !hasEnd, isSignalled(wait.pairing) { return .signalled }
    switch ProcessCheck.verdict(
      wait: wait, hasEnd: hasEnd, running: inspect(wait.pid), gdoc: gdoc, user: user,
      resolve: resolve)
    {
    case .stoppable: return .ready
    case .refused(let why): return .refused(why)
    }
  }

  /// The documents whose Stop is greyed because the check refused. A watch
  /// between two polls has no process at all and is greyed by the panel
  /// itself, so it is not here.
  public func unstoppable(in events: [Event]) -> Set<String> {
    var docs: Set<String> = []
    for doc in Self.openWaits(in: events).keys {
      switch standing(doc: doc, in: events) {
      case .refused, .noGdoc: docs.insert(doc)
      case .ready, .signalled: continue
      }
    }
    return docs
  }

  /// The documents whose signal has gone and whose end has not arrived. The
  /// live row draws a ring for each of them.
  ///
  /// This is also where a wait is forgotten: a pairing that is no longer an
  /// open wait in the record has ended, so the memory of its signal has
  /// nothing left to protect.
  public func pendingDocs(in events: [Event]) -> Set<String> {
    let open = Self.openWaits(in: events)
    let keys = Set(open.values.map(\.pairing))
    lock.lock()
    signalled.formIntersection(keys)
    let still = signalled
    lock.unlock()
    return Set(open.filter { still.contains($0.value.pairing) }.keys)
  }

  // MARK: the signal

  /// Stop the wait on one document: one SIGINT, after the check and only
  /// after it. The standing afterwards is what the panel draws.
  public func stop(doc: String, in events: [Event]) -> StopStanding {
    let standing = standing(doc: doc, in: events)
    guard case .ready = standing else { return standing }
    guard let (wait, _) = Self.latestWait(doc: doc, in: events) else {
      return .refused(.notAWait)
    }

    remember(wait.pairing)
    guard send(wait.pid, Self.signal) else {
      // The pid went between the check and the signal, which is the one
      // moment this app cannot close. Nothing was signalled, so nothing is
      // remembered.
      forget(wait.pairing)
      return .refused(.gone)
    }
    return .signalled
  }

  // MARK: the kernel

  /// What the kernel says about a pid.
  ///
  /// A path the kernel will not name is a refusal rather than a guess, and
  /// it is also what a pid that has already died answers: measured
  /// 2026-10-17, `proc_pidpath` returns ESRCH for a process that is gone
  /// while `kill(pid, 0)` can still say yes for one nobody has reaped.
  public static func ask(_ pid: Int32) -> RunningProcess {
    RunningProcess(
      path: path(of: pid), uid: info(of: pid)?.uid ?? UInt32.max,
      started: info(of: pid)?.started, isAlive: Liveness.isAlive(pid))
  }

  /// One signal to one pid. The only call in the package that carries one.
  public static func sent(_ pid: Int32, _ signal: Int32) -> Bool {
    guard pid > 0 else { return false }
    return kill(pid, signal) == 0
  }

  /// The program a pid is running, as the kernel names it now. A binary that
  /// was renamed under the running process answers with its new name, which
  /// is why `ProcessCheck` takes `gdoc.previous` for gdoc: measured
  /// 2026-10-17 and written into `docs/v2/MEASURED.md`.
  private static func path(of pid: Int32) -> String? {
    var buffer = [CChar](repeating: 0, count: Int(4 * MAXPATHLEN))
    let length = proc_pidpath(pid, &buffer, UInt32(buffer.count))
    guard length > 0 else { return nil }
    return String(cString: buffer)
  }

  /// Whose the process is and when it began.
  private static func info(of pid: Int32) -> (uid: uid_t, started: Date)? {
    var info = proc_bsdinfo()
    let size = Int32(MemoryLayout<proc_bsdinfo>.size)
    let read = withUnsafeMutablePointer(to: &info) {
      proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, $0, size)
    }
    guard read == size else { return nil }
    let seconds =
      Double(info.pbi_start_tvsec) + Double(info.pbi_start_tvusec) / 1_000_000
    return (info.pbi_uid, Date(timeIntervalSince1970: seconds))
  }

  // MARK: the memory

  private func isSignalled(_ pairing: String) -> Bool {
    lock.lock()
    defer { lock.unlock() }
    return signalled.contains(pairing)
  }

  private func remember(_ pairing: String) {
    lock.lock()
    signalled.insert(pairing)
    lock.unlock()
  }

  private func forget(_ pairing: String) {
    lock.lock()
    signalled.remove(pairing)
    lock.unlock()
  }

  // MARK: the record

  /// The newest wait on one document, and whether its end is written.
  ///
  /// A wait is a start whose line says `wait`. Whether it is the right
  /// command, from the right place and inside its window is the check's
  /// question and not this one's, so a line that says the wrong thing is
  /// found here and refused there by name.
  static func latestWait(doc: String, in events: [Event]) -> (Event, Bool)? {
    var ends: Set<String> = []
    var newest: Event?
    for event in events {
      guard event.doc == doc, event.wait else { continue }
      switch event.kind {
      case .end: ends.insert(event.pairing)
      case .start:
        if newest == nil || event.at >= (newest?.at ?? event.at) { newest = event }
      default: continue
      }
    }
    guard let newest else { return nil }
    return (newest, ends.contains(newest.pairing))
  }

  /// Every document with a wait that has no end, by document.
  static func openWaits(in events: [Event]) -> [String: Event] {
    var found: [String: Event] = [:]
    for doc in Set(events.compactMap { $0.wait ? $0.doc : nil }) {
      guard let (wait, hasEnd) = latestWait(doc: doc, in: events), !hasEnd else { continue }
      found[doc] = wait
    }
    return found
  }
}
