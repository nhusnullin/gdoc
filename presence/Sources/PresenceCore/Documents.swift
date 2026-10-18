// DOCUMENTS: the same record as TODAY, grouped by the document it touched.
//
// One row per document, and the document leaves the list fourteen days after
// its last action. The order is three lists: the documents a person pinned,
// newest pin first; then every document acted on in the local day, newest
// first; and then the rest of the fortnight by how many finished actions
// each one carries in it, most first, ties newest first.
// What gdoc just made is what a person reaches for, and under the day the
// fortnight's own documents are the ones it was mostly about; a document
// touched once a week ago sinks instead of sitting above one worked on every
// day. There is still no sorting by state, because a document that jumps when
// a job ends is a document a person loses, and the count is never drawn: a
// count is a judgement, so the row says the last action and its time as it
// always did. The decision is docs/v2/DECISIONS.md, 2026-10-18, "The second
// morning"; the earlier rule is the caption of scene e1 of
// docs/design/2026-10-17-presence-screens-v3.html.
//
// Three rules decide what a row is, each a product decision of the M24
// specification rather than this file's reading:
//
//   - A read alone makes no row. What makes a row is a write, or a live
//     watch; a read, and a `comments` that did not wait, are gdoc looking at
//     a document rather than doing something to it.
//   - The state is the open action where there is one, in the order hold,
//     running, live, and the latest finished action otherwise. The order the
//     rows are in never changes with it.
//   - Five rows by default, with the rest behind Show more, and the label
//     counts the horizon and the local day.
//
// A pin and a hide are the person's own answers about one document, and they
// come in as arguments like everything else here: they are the app's
// preferences and never a line in the record. A pin holds the document in
// the list past the horizon, and holds its title and its id, so a document
// whose lines the tail can no longer see still has a row and an open action.
// A hide takes the document out until gdoc finishes something on it again,
// which is the one way back; TODAY still carries its runs, because the log
// is the log. A document that is both is kept: the pin is the louder answer.
//
// The clock, the calendar and the kernel's answer about a pid come in as
// arguments, so every row is reachable from a fixture and nothing here reads
// the machine.

import Foundation

/// What a document's row says is going on with it. Each is a fact off the
/// record: an end that said so, a process that is there or gone, a watch
/// whose polls are still arriving.
public enum DocumentState: String, Sendable, Equatable, CaseIterable {
  /// A chat write on it is waiting for a person.
  case hold
  /// A command on it is running.
  case running
  /// A live watch on it is still polling.
  case live
  /// Its live watch stopped, and that is the last thing that happened.
  case liveStopped
  /// Its last action ended and said it worked.
  case done
  /// Its last action ended and said it failed.
  case failed
  /// Its last action started and nobody wrote down how it went.
  case unknown
  /// The record says nothing about it at all: the row is here because it is
  /// pinned, and all the panel has is the pin's own title and id. Only a
  /// pinned document is ever in this state.
  case pinnedOnly
}

/// What a person decided about one document: its id, the moment they decided
/// it, and the title the list was drawing then.
///
/// The title is kept for one reason: a pinned document whose lines are past
/// the horizon, or past the two month files the tail reads, has no title
/// anywhere else. It is the app's own preference and never a line in the
/// record.
public struct DocumentChoice: Sendable, Equatable {
  public let doc: String
  public let at: Date
  public let title: String?

  public init(doc: String, at: Date, title: String?) {
    self.doc = doc
    self.at = at
    self.title = title
  }
}

/// One document's row.
public struct DocumentRow: Sendable, Equatable {
  public let doc: String
  public let title: String?
  /// The command of the action the row shows: the open one where there is
  /// one, and the latest finished one otherwise.
  public let command: String
  /// The document's last action, which is the one thing the order is by.
  public let at: Date
  public let state: DocumentState
  /// Whether that action is in the local day the fold was made in.
  public let isToday: Bool
  /// Whether the person pinned this document, which is why it is first and
  /// why it is still here.
  public let isPinned: Bool
}

/// DOCUMENTS, and the two numbers its label carries.
public struct DocumentsFold: Sendable, Equatable {
  /// Every document inside the horizon: the day's, newest first, then the
  /// fortnight's by use.
  public let rows: [DocumentRow]

  /// How many documents the horizon holds.
  public var count: Int { rows.count }

  /// How many of them were acted on in the local day.
  public var today: Int { rows.filter(\.isToday).count }

  /// What is behind Show more.
  public var more: Int { max(0, rows.count - Documents.defaultRows) }

  /// The rows the section draws, which is the newest five until a person
  /// opens it.
  public func visible(expanded: Bool) -> [DocumentRow] {
    expanded ? rows : Array(rows.prefix(Documents.defaultRows))
  }
}

public enum Documents {
  /// Fourteen days after its last action, a document leaves the list. The
  /// same horizon NOW holds a stopped write inside.
  public static let horizon: TimeInterval = Now.horizon

  /// Five rows, and the rest behind Show more.
  public static let defaultRows = 5

  /// The events into DOCUMENTS.
  ///
  /// - Parameters:
  ///   - now: the moment the panel is drawn at.
  ///   - calendar: the calendar the local day comes from, injected because
  ///     the day is the thing under test and the machine's zone is not it.
  ///   - alive: the kernel's answer for one pid.
  public static func fold(
    _ events: [Event], now: Date, calendar: Calendar, pinned: [DocumentChoice] = [],
    hidden: [DocumentChoice] = [], alive: (Int32) -> Bool
  ) -> DocumentsFold {
    let ordered = Fold.inOrder(events)
    let titles = Fold.latestTitles(ordered)

    var actions: [String: [Action]] = [:]
    for action in every(ordered, now: now, alive: alive) {
      actions[action.doc, default: []].append(action)
    }

    let pins = Dictionary(pinned.map { ($0.doc, $0.at) }, uniquingKeysWith: { $1 })
    for doc in away(hidden, pins: pins, actions: actions) {
      actions.removeValue(forKey: doc)
    }

    let since = now.addingTimeInterval(-horizon)
    let finished = actions.mapValues { found in
      found.filter { $0.isFinished && $0.at >= since }.count
    }
    var rows =
      actions
      .compactMap { doc, found in
        row(doc, found, titles: titles, now: now, calendar: calendar, pinned: pins[doc] != nil)
      }
      .filter { $0.at >= since || pins[$0.doc] != nil }
    for pin in pinned where actions[pin.doc] == nil {
      rows.append(
        DocumentRow(
          doc: pin.doc, title: pin.title, command: "", at: pin.at, state: .pinnedOnly,
          isToday: false, isPinned: true))
    }

    return DocumentsFold(rows: rows.sorted { before($0, $1, finished: finished, pins: pins) })
  }

  /// The documents a hide takes out of the list.
  ///
  /// A hide holds until gdoc finishes something on that document again,
  /// which is the one way back: nothing in the panel unhides one, because
  /// the row a person took away is a row they took away. A pinned document
  /// is never away, whatever a hide of it says.
  private static func away(
    _ hidden: [DocumentChoice], pins: [String: Date], actions: [String: [Action]]
  ) -> Set<String> {
    var found: Set<String> = []
    for hide in hidden where pins[hide.doc] == nil {
      let since = actions[hide.doc] ?? []
      if !since.contains(where: { $0.isFinished && $0.at > hide.at }) { found.insert(hide.doc) }
    }
    return found
  }

  /// One thing that happened to one document, which is a candidate for being
  /// its row.
  private struct Action {
    let doc: String
    let at: Date
    let command: String
    let state: DocumentState
    /// Whether the thing is still going, which decides it over any finished
    /// action however late.
    var isOpen: Bool { state == .hold || state == .running || state == .live }
    /// Whether the thing is over and said how it went, which is what the
    /// order under the day counts. A start whose process is gone said
    /// nothing, so it counts for nothing.
    var isFinished: Bool { state == .done || state == .failed || state == .liveStopped }
  }

  /// Which of two rows is the higher one.
  ///
  /// A pin is above everything, and two pins are in the order they were
  /// made, newest first: a person who pins a second document is working on
  /// it. The local day is above the fortnight, and inside the day the last
  /// action is the only order, because the thing that just happened is the
  /// thing a hand goes to. Under the day the count of finished actions in the
  /// fortnight is the order, most first, and two documents used the same
  /// number of times are newest first. A document's id breaks a tie nothing
  /// else broke, so one record folds one way:
  /// testTheOrderIsTodayNewestFirstThenTheFortnightByUse.
  private static func before(
    _ left: DocumentRow, _ right: DocumentRow, finished: [String: Int], pins: [String: Date]
  ) -> Bool {
    if left.isPinned != right.isPinned { return left.isPinned }
    if left.isPinned, let mine = pins[left.doc], let theirs = pins[right.doc], mine != theirs {
      return mine > theirs
    }
    if left.isToday != right.isToday { return left.isToday }
    if !left.isToday {
      let mine = finished[left.doc] ?? 0
      let theirs = finished[right.doc] ?? 0
      if mine != theirs { return mine > theirs }
    }
    if left.at != right.at { return left.at > right.at }
    return left.doc < right.doc
  }

  /// A document's row: the open action where there is one, and the latest
  /// finished one otherwise, at the moment of the latest action either way.
  private static func row(
    _ doc: String, _ actions: [Action], titles: [String: String], now: Date, calendar: Calendar,
    pinned: Bool
  ) -> DocumentRow? {
    guard let last = actions.max(by: { $0.at < $1.at }) else { return nil }
    let open = actions.filter(\.isOpen).min { rank($0.state) < rank($1.state) }
    let shown = open ?? last

    return DocumentRow(
      doc: doc, title: titles[doc], command: shown.command, at: last.at, state: shown.state,
      isToday: calendar.isDate(last.at, inSameDayAs: now), isPinned: pinned)
  }

  /// Which open action wins: a hold waits for a person, so it is first; a
  /// running job is next, because it is about to change the document; a live
  /// watch is the quiet one.
  private static func rank(_ state: DocumentState) -> Int {
    switch state {
    case .hold: return 0
    case .running: return 1
    case .live: return 2
    default: return 3
    }
  }

  /// Every action the record holds, which is every write and every live
  /// watch. A read is not one.
  private static func every(
    _ ordered: [Event], now: Date, alive: (Int32) -> Bool
  ) -> [Action] {
    var found: [Action] = []
    var held: [String: Event] = [:]
    var ended: Set<String> = []

    for event in ordered {
      if event.kind == .end { ended.insert(event.pairing) }
      // The hold's two halves are read before the document is, because a
      // released, expired, landed or dropped line carries the hold's id and
      // nothing else: `go/internal/presence` writes no document on one. A
      // document guard above this would never let an ending through, and
      // the row would wait for a person for ever:
      // testDocumentsGroupByIdInsideFourteenDaysNewestFirst.
      if event.kind == .held {
        held[event.hold ?? ""] = event
      } else if HoldEnding(event.kind) != nil {
        held.removeValue(forKey: event.hold ?? "")
      }
      // A publish is handed no document and names the one it created, so
      // the row it makes is under `made`: Fold.subject is that rule.
      guard let doc = Fold.subject(event) else { continue }
      guard event.kind == .end, !event.wait, Now.stuckCommands.contains(event.command ?? "") else {
        continue
      }
      found.append(
        Action(
          doc: doc, at: event.at, command: event.command ?? "",
          state: event.ok == true ? .done : .failed))
    }

    for event in ordered {
      guard event.kind == .start, !event.wait, let doc = event.doc, !doc.isEmpty,
        Now.stuckCommands.contains(event.command ?? ""), !ended.contains(event.pairing)
      else { continue }
      found.append(
        Action(
          doc: doc, at: event.at, command: event.command ?? "",
          state: alive(event.pid) ? .running : .unknown))
    }

    for event in held.values {
      guard let doc = event.doc, !doc.isEmpty else { continue }
      found.append(
        Action(doc: doc, at: event.at, command: event.command ?? "", state: .hold))
    }

    for session in Live.sessions(ordered: ordered, now: now, alive: alive) {
      // The watch's moment is its last poll and never its first: a watch
      // polling right now acted a second ago, and `since` would sort it
      // under every write that finished since it began and would age it out
      // of the horizon while NOW still drew it live. For a watch that is
      // over, `lastPoll` is `ended`.
      found.append(
        Action(
          doc: session.doc, at: session.lastPoll, command: "comments",
          state: session.isOpen ? .live : .liveStopped))
    }

    return found
  }
}
