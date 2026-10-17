// DOCUMENTS: the same record as TODAY, grouped by the document it touched.
//
// One row per document, newest first by its last action, and the document
// leaves the list fourteen days after that action. There is no pinning and no
// sorting by state, because a document that jumps when a job ends is a
// document a person loses: the order is the last action and nothing else.
// That is the caption of scene e1 of
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
}

/// DOCUMENTS, and the two numbers its label carries.
public struct DocumentsFold: Sendable, Equatable {
  /// Every document inside the horizon, newest first by its last action.
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
    _ events: [Event], now: Date, calendar: Calendar, alive: (Int32) -> Bool
  ) -> DocumentsFold {
    let ordered = Fold.inOrder(events)
    let titles = Fold.latestTitles(ordered)

    var actions: [String: [Action]] = [:]
    for action in every(ordered, now: now, alive: alive) {
      actions[action.doc, default: []].append(action)
    }

    let rows =
      actions
      .compactMap { doc, found in row(doc, found, titles: titles, now: now, calendar: calendar) }
      .filter { $0.at >= now.addingTimeInterval(-horizon) }
      .sorted { $0.at == $1.at ? $0.doc < $1.doc : $0.at > $1.at }

    return DocumentsFold(rows: rows)
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
  }

  /// A document's row: the open action where there is one, and the latest
  /// finished one otherwise, at the moment of the latest action either way.
  private static func row(
    _ doc: String, _ actions: [Action], titles: [String: String], now: Date, calendar: Calendar
  ) -> DocumentRow? {
    guard let last = actions.max(by: { $0.at < $1.at }) else { return nil }
    let open = actions.filter(\.isOpen).min { rank($0.state) < rank($1.state) }
    let shown = open ?? last

    return DocumentRow(
      doc: doc, title: titles[doc], command: shown.command, at: last.at, state: shown.state,
      isToday: calendar.isDate(last.at, inSameDayAs: now))
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
