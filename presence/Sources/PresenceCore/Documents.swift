// DOCUMENTS: the record grouped by the document it touched.
//
// One row per document, and the list is two lists: the documents a person
// pinned, newest pin first, and then the last five by their latest action,
// newest first. Nothing else is in it. A document already drawn as a pin is
// not drawn again under the five, and it does not use one of them up.
//
// That is the third look, docs/v2/DECISIONS.md, 2026-10-18: Nail asked for
// the last five plus the pinned ones, no duplicates and no scroll. The
// fortnight ordered by how much each document was used, which the second
// morning added, is withdrawn with it, and so is the day: a count is a
// judgement and the day was a second order inside one list. What is left is
// the one thing a hand goes to, which is what gdoc touched last.
//
// A document still leaves the list fourteen days after its last action. With
// five rows the horizon rarely decides anything on its own; it decides what
// a pin survives, which is scene e2.
//
// Two rules decide what a row is, each a product decision of the M24
// specification rather than this file's reading:
//
//   - A read alone makes no row. What makes a row is a write, or a live
//     watch; a read, and a `comments` that did not wait, are gdoc looking at
//     a document rather than doing something to it.
//   - The state is the open action where there is one, in the order hold,
//     running, live, and the latest finished action otherwise. The order the
//     rows are in never changes with it, because a document that jumps when
//     a job ends is a document a person loses.
//
// A pin and a hide are the person's own answers about one document, and they
// come in as arguments like everything else here: they are the app's
// preferences and never a line in the record. A pin holds the document in
// the list past the horizon and past the five, and holds its title and its
// id, so a document whose lines the tail can no longer see still has a row
// and an open action. A hide takes the document out until gdoc finishes
// something on it again, which is the one way back. A document that is both
// is kept: the pin is the louder answer.
//
// A row also remembers the Claude Code session its document was last worked
// on from, where the record named one: the record carries a session on the
// start of a `via: code` run, and Open chat is the one press that would use
// it. Measured 2026-10-18, MEASURED.md: `claude://code/continue` activates
// Claude Desktop from outside and switches nothing, so nothing opens that
// link today and the row holds the id for the day a form does.
//
// The clock and the kernel's answer about a pid come in as arguments, so
// every row is reachable from a fixture and nothing here reads the machine.

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
  /// The record holds no action on it at all: the row is here because it is
  /// pinned, and its id is the pin's own. Its name is whatever the record
  /// last called it, and the pin's remembered title where the record has
  /// none. Only a pinned document is ever in this state.
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
  /// Whether the person pinned this document, which is why it is first and
  /// why it is still here.
  public let isPinned: Bool
  /// The Claude Code session this document was last worked on from, where the
  /// record named one. It is nothing for a document only a terminal or a chat
  /// touched, which is every document until gdoc v2.17.0 wrote the field.
  public let session: String?
}

/// DOCUMENTS: the rows, and nothing beside them.
///
/// There is no count here and none on the label. "14 days · 12" and
/// "3 today" were both the panel saying how much of something there is,
/// which is a judgement over a list a person can see the whole of:
/// the third look took them off with the sections they stood over.
public struct DocumentsFold: Sendable, Equatable {
  /// The list, in its one order: the pinned documents first, newest pin
  /// first, then the last five by their latest action, newest first.
  ///
  /// A pin is in the list however old it is, so a row here may be a document
  /// whose last action is past the fortnight, or one the record says nothing
  /// about at all, which is `.pinnedOnly`.
  public let rows: [DocumentRow]
}

public enum Documents {
  /// Fourteen days after its last action, a document leaves the list. The
  /// same horizon NOW holds a stopped write inside.
  public static let horizon: TimeInterval = Now.horizon

  /// Five documents under the pins, and nothing behind anything: the
  /// panel draws every row the fold hands it.
  public static let recentRows = 5

  /// The events into DOCUMENTS.
  ///
  /// - Parameters:
  ///   - now: the moment the panel is drawn at.
  ///   - alive: the kernel's answer for one pid.
  public static func fold(
    _ events: [Event], now: Date, pinned: [DocumentChoice] = [],
    hidden: [DocumentChoice] = [], alive: (Int32) -> Bool
  ) -> DocumentsFold {
    let ordered = Fold.inOrder(events)
    let titles = Fold.latestTitles(ordered)
    let sessions = Fold.latestSessions(ordered)

    var actions: [String: [Action]] = [:]
    for action in every(ordered, now: now, alive: alive) {
      actions[action.doc, default: []].append(action)
    }

    let pins = Dictionary(pinned.map { ($0.doc, $0.at) }, uniquingKeysWith: { $1 })
    // The title a pin kept, which fills a gap the record has and never
    // speaks over it: testAPinsTitleFillsAGapAndNeverSpeaksOverTheRecord.
    let pinTitles = Dictionary(
      pinned.compactMap { pin in pin.title.map { (pin.doc, $0) } }, uniquingKeysWith: { $1 })
    for doc in away(hidden, pins: pins, actions: actions) {
      actions.removeValue(forKey: doc)
    }

    let since = now.addingTimeInterval(-horizon)
    var rows =
      actions
      .compactMap { doc, found in
        row(
          doc, found, titles: titles, pinTitle: pinTitles[doc], session: sessions[doc],
          pinned: pins[doc] != nil)
      }
      .filter { $0.at >= since || pins[$0.doc] != nil }
    for pin in pinned where actions[pin.doc] == nil {
      rows.append(
        DocumentRow(
          doc: pin.doc, title: titles[pin.doc] ?? pin.title, command: "", at: pin.at,
          state: .pinnedOnly, isPinned: true, session: sessions[pin.doc]))
    }

    // The pins, then the five. A pinned document is taken out of the second
    // list rather than counted in it: it is already on the screen, and
    // counting it would make a pin cost a person one of their five:
    // testTheListIsThePinnedThenTheLastFiveWithNoDuplicates.
    let held = rows.filter(\.isPinned).sorted { byPin($0, $1, pins: pins) }
    let rest = rows.filter { !$0.isPinned }.sorted(by: byAction).prefix(recentRows)
    return DocumentsFold(rows: held + rest)
  }

  /// Which of two pinned rows is the higher one: the order they were pinned
  /// in, newest pin first, because a person who pins a second document is
  /// working on it. The last action breaks a tie, and the id breaks what
  /// that leaves, so one record folds one way.
  private static func byPin(
    _ left: DocumentRow, _ right: DocumentRow, pins: [String: Date]
  ) -> Bool {
    if let mine = pins[left.doc], let theirs = pins[right.doc], mine != theirs {
      return mine > theirs
    }
    return byAction(left, right)
  }

  /// Which of two rows gdoc touched last, with the id breaking a tie
  /// nothing else broke.
  private static func byAction(_ left: DocumentRow, _ right: DocumentRow) -> Bool {
    left.at == right.at ? left.doc < right.doc : left.at > right.at
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
    /// Whether the thing is over and said how it went, which is what brings
    /// a hidden document back. A start whose process is gone said nothing,
    /// so it is not one.
    var isFinished: Bool { state == .done || state == .failed || state == .liveStopped }
  }

  /// A document's row: the open action where there is one, and the latest
  /// finished one otherwise, at the moment of the latest action either way.
  /// The pin's own title is the fallback and never the answer: a start
  /// carries no title, so a pinned document that is only running has one
  /// nowhere else, and a row that lost the name of the thing a person pinned
  /// draws a Drive id at them.
  private static func row(
    _ doc: String, _ actions: [Action], titles: [String: String], pinTitle: String?,
    session: String?, pinned: Bool
  ) -> DocumentRow? {
    guard let last = actions.max(by: { $0.at < $1.at }) else { return nil }
    let open = actions.filter(\.isOpen).min { rank($0.state) < rank($1.state) }
    let shown = open ?? last

    return DocumentRow(
      doc: doc, title: titles[doc] ?? pinTitle, command: shown.command, at: last.at,
      state: shown.state, isPinned: pinned, session: session)
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
