// A live watch, which the record never writes down as one.
//
// `gdoc comments --wait` is one process per poll: the skill runs it, it waits
// for something new on one document, it answers, and the skill runs it again.
// So a watch a person thinks of as "live on Chapter 5 since one o'clock" is
// twelve starts and twelve ends in the record, each with its own run and its
// own pid. This file stitches them back: the waits on one document whose gaps
// are under three minutes are one session.
//
// What a session says is the M24 specification's product decision, and each
// line of it is a fact off the record rather than a reading of one:
//
//   - since, which is the first wait's start, and ended, which is the last
//     wait's end once three minutes have passed with no new poll, or that end
//     at once where a person stopped the wait.
//   - replied and proposed, counted from the document's own ok ends inside
//     the session. A write that failed answered nobody and is not counted.
//   - the last write's time, which is what quiet is measured from. With no
//     write, the watch has been quiet since it began.
//   - whether the last poll failed, with the moment it did and gdoc's own
//     word for why, which is one of seven codes and never a sentence.
//   - whether a person stopped the last poll, which its end says and nothing
//     else in the record does.
//   - the wait in flight, if there is one, because that pid is the only thing
//     Stop can signal. Between two polls a session is open and has no pid.
//
// A last wait that wrote no end and whose pid the kernel does not know is the
// watch over, at that wait's own start. A process killed outright, or one that
// took a SIGHUP with the terminal window, writes nothing: without this the
// session would be open with no bound at all, the bar would say live for as
// long as the record keeps the line, and Stop would offer to signal a pid that
// is gone. It is the same rule NOW applies to a start whose process left, and
// the kernel's answer comes in as an argument for the same reason.
//
// The thread's own state is not here and never will be: gdoc prints no field
// for whether an author has answered, so the panel says nothing about it.

import Foundation

/// One live watch on one document.
public struct LiveSession: Sendable, Equatable {
  /// The `comments --wait` that is running right now, which is the one
  /// process the app may signal. Absent between two polls and after the
  /// watch stopped.
  public struct Flight: Sendable, Equatable {
    public let pid: Int32
    public let run: String
    public let call: Int64
    public let started: Date
  }

  public let doc: String
  public let title: String?
  /// The first wait's start.
  public let since: Date
  /// The last wait's end, once no poll followed it inside the gap, or that
  /// end at once where the wait says a person stopped it. Absent while the
  /// watch is still going.
  public let ended: Date?
  /// Where the last poll stands: its end, or its own start while it is
  /// still in flight.
  ///
  /// It is what DOCUMENTS orders an open watch by, because a watch polling
  /// right now acted a second ago and not when it began, and ordering it by
  /// `since` would put an hour-old publish above it and drop a watch left
  /// open past the horizon out of the list while NOW still drew it live:
  /// testAnOpenWatchIsOrderedByItsLastPollAndNotItsFirst. For a watch that
  /// is over it is `ended`, which is that same moment.
  public let lastPoll: Date
  public let replied: Int
  public let proposed: Int
  /// The last write on this document inside the session.
  public let lastWrite: Date?
  /// When the last poll failed, where it did.
  public let lastPollFailed: Date?
  /// Why it failed, in gdoc's own word for it. The record carries one of
  /// seven codes and no sentence, and this app keeps the word it wrote.
  public let lastPollStop: String?
  /// A person stopped the last poll. `comments` says so on its end, and it
  /// is the one way the panel can tell a watch somebody ended from a watch
  /// whose process went away.
  public let stoppedByYou: Bool
  public let flight: Flight?

  public var isOpen: Bool { ended == nil }

  /// What the panel's "quiet" counts from: the last write, and the watch's
  /// own start where nothing has been written yet.
  public var quietSince: Date { lastWrite ?? since }
}

public enum Live {
  /// Three minutes: the longest a watch may go between two polls and still be
  /// the same watch. The skill starts the next wait in a second or two, so a
  /// gap this long is a person who stopped.
  public static let gap: TimeInterval = 3 * 60

  /// Every live watch the events hold, oldest first.
  ///
  /// now decides only whether the last session on a document is still open,
  /// so a fixture folds the same way at any later moment, and alive decides
  /// whether a last wait that wrote no end is still one.
  public static func sessions(
    _ events: [Event], now: Date, alive: (Int32) -> Bool
  ) -> [LiveSession] {
    let ordered = Fold.inOrder(events)
    let titles = Fold.latestTitles(ordered)

    var polls: [String: [Poll]] = [:]
    for poll in waits(ordered) {
      polls[poll.doc, default: []].append(poll)
    }

    var sessions: [LiveSession] = []
    for (doc, byDoc) in polls {
      for run in stitch(byDoc.sorted { $0.started < $1.started }) {
        sessions.append(
          session(run, doc: doc, in: ordered, titles: titles, now: now, alive: alive))
      }
    }
    return sessions.sorted { $0.since < $1.since }
  }

  /// One `comments --wait` call: a start, and the end that paired with it.
  private struct Poll {
    let doc: String
    let pid: Int32
    let run: String
    let call: Int64
    let started: Date
    var ended: Date?
    var ok: Bool?
    var stop: String?
    var interrupted = false

    /// When the next poll's gap is measured from: this one's end, or its
    /// start while it is still in flight.
    var until: Date { ended ?? started }
  }

  /// The waits, each with its end where the record holds one.
  ///
  /// A wait is a start whose line says `wait`, which only `comments` writes.
  /// A wait on no document is left out: a watch is a watch of something, and
  /// nothing in the panel could name that row.
  ///
  /// The `wait` test is the start's alone. `go/internal/presence` writes the
  /// field on the start and never on the end, so an end held to it would
  /// pair with nothing and every poll would stay open for ever: the end is
  /// found by the start's own `run|call` instead, which is what pairs the
  /// two halves of every other command.
  /// testAnEndWithoutTheWaitFieldStillClosesItsStart.
  private static func waits(_ ordered: [Event]) -> [Poll] {
    var polls: [String: Int] = [:]
    var found: [Poll] = []

    for event in ordered {
      switch event.kind {
      case .start:
        guard event.wait, let doc = event.doc, !doc.isEmpty else { continue }
        polls[event.pairing] = found.count
        found.append(
          Poll(
            doc: doc, pid: event.pid, run: event.run, call: event.call ?? 0, started: event.at,
            ended: nil, ok: nil))
      case .end:
        guard let at = polls[event.pairing] else { continue }
        found[at].ended = event.at
        found[at].ok = event.ok ?? false
        found[at].stop = event.stop
        found[at].interrupted = event.interrupted
      case .step, .held, .released, .expired, .landed, .dropped:
        continue
      }
    }
    return found
  }

  /// One document's polls, oldest first, cut into sessions wherever three
  /// minutes passed between one poll's end and the next poll's start.
  private static func stitch(_ polls: [Poll]) -> [[Poll]] {
    var sessions: [[Poll]] = []
    var current: [Poll] = []
    for poll in polls {
      if let last = current.last, poll.started.timeIntervalSince(last.until) >= gap {
        sessions.append(current)
        current = []
      }
      current.append(poll)
    }
    if !current.isEmpty { sessions.append(current) }
    return sessions
  }

  private static func session(
    _ polls: [Poll], doc: String, in ordered: [Event], titles: [String: String], now: Date,
    alive: (Int32) -> Bool
  ) -> LiveSession {
    let since = polls[0].started
    let last = polls[polls.count - 1]

    var flight: LiveSession.Flight?
    var ended: Date?
    if let close = last.ended {
      // A wait a person stopped ends the watch where it ended, and does not
      // wait out the gap: no poll ever follows one, because the skill reads
      // `waited.interrupted`, prints its totals and stops. Every other end
      // waits, because there the next poll may be a second away:
      // testAnInterruptedWaitEndsTheWatchAtOnce.
      if last.interrupted || now.timeIntervalSince(close) >= gap {
        ended = close
      } else {
        ended = nil
      }
    } else if alive(last.pid) {
      flight = LiveSession.Flight(
        pid: last.pid, run: last.run, call: last.call, started: last.started)
    } else {
      // The last wait wrote no end and its process is not there, so the
      // watch ended where that wait began: nothing later than its start is
      // a fact the record holds about it.
      // testAWaitWhoseProcessWentAwayEndsTheWatch.
      ended = last.started
    }

    // The writes inside the session's own span. An open session runs to the
    // moment it is folded at, so a reply written a second ago is counted.
    let until = ended ?? now
    let mine = ordered.filter {
      $0.kind == .end && $0.doc == doc && $0.ok == true && $0.at >= since && $0.at <= until
    }

    return LiveSession(
      doc: doc,
      title: titles[doc],
      since: since,
      ended: ended,
      lastPoll: last.until,
      replied: mine.filter { $0.command == "reply" }.count,
      proposed: mine.filter { $0.command == "propose" }.count,
      lastWrite: mine.filter { Now.stuckCommands.contains($0.command ?? "") }.map(\.at).max(),
      lastPollFailed: last.ok == false ? last.ended : nil,
      lastPollStop: last.ok == false ? last.stop : nil,
      stoppedByYou: last.interrupted,
      flight: flight)
  }
}
