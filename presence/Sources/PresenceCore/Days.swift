// TODAY and Yesterday: the log of what gdoc did, cut by the local day.
//
// TODAY is the local day of the moment the panel is drawn at, newest first,
// eight rows with the rest behind Show more. Yesterday is the day before it,
// behind one chevron row that carries the two counts and opens in place.
// Nothing older is in the panel: history is a question for Claude Code or the
// hub, which is the caption of scene e2 of
// docs/design/2026-10-17-presence-screens-v3.html.
//
// Four rows are not events. The record writes a watch as one process per
// poll and a hold as a pair of lines, and a person thinks of both as one
// thing that started and stopped, so the fold writes those rows itself: a
// hold raised, a hold closed, a watch started, and a watch stopped with its
// totals. Each is placed at its own moment, so a watch that began yesterday
// and stopped today has its two rows in two sections.
//
// Reads are left out, as the M24 specification decided: a read, and a
// `comments` that did not wait, is gdoc looking at a document rather than
// doing something to it. What is left in is the seven writes, the watches,
// the holds, and a write whose process went away with nobody writing down
// how it went.
//
// The day is the calendar's and never the record file's. The record is one
// file per month, so on the first of a month yesterday is in the other file,
// and the fold neither knows nor cares which file a line came from.

import Foundation

/// How a hold stopped waiting. The record's own words, with no reading on
/// top, and each of them is `cmd/gdoc`'s own: released is the person
/// pressing the card, written before the write it approved goes out
/// (`mcprelease.go` takes the hold out and then sends); landed is that write
/// reaching the document another way, which is the model sending it again
/// with no card (`mcphold.go`'s `landed`); expired is the hold's own life
/// running out; and dropped is the sign-in it was held under stopping being
/// the sign-in on the computer.
///
/// No line says a person refused one. A card nobody answers expires, which
/// is why the panel never words any of these as a no.
public enum HoldEnding: String, Sendable, Equatable, CaseIterable {
  case released
  case expired
  case landed
  case dropped

  /// The ending an event kind is, and nothing for the kinds that are not one.
  public init?(_ kind: EventKind) {
    guard let ending = HoldEnding(rawValue: kind.rawValue) else { return nil }
    self = ending
  }
}

/// What one row of the log is.
public enum DayKind: String, Sendable, Equatable, CaseIterable {
  /// A command that ended, whether it worked or not.
  case action
  /// A command that started and whose process went away with no end written.
  case outcomeUnknown
  case holdRaised
  case holdClosed
  case liveStarted
  case liveStopped
}

/// What a watch did while it ran. The counts are the document's own ok ends
/// inside the session, never a judgement about the threads.
public struct LiveTotals: Sendable, Equatable {
  public let since: Date
  public let ended: Date
  public let replied: Int
  public let proposed: Int
  /// A person stopped the last poll, which the record says on the end of a
  /// wait and nowhere else. A watch whose process went away says nothing.
  public let stoppedByYou: Bool

  /// How long it ran.
  public var ran: TimeInterval { ended.timeIntervalSince(since) }
}

/// One row of the log.
public struct DayRow: Sendable, Equatable {
  public let kind: DayKind
  public let at: Date
  public let command: String?
  public let via: String?
  public let doc: String?
  public let title: String?
  /// The envelope's own ok, where a line carried one.
  public let ok: Bool?
  /// Only the commands that read their own write back carry this.
  public let verified: Bool?
  public let ms: Int64?
  /// The rule that held a chat write, on a hold raised row.
  public let rule: String?
  /// How a hold stopped waiting, on a hold closed row.
  public let ending: HoldEnding?
  /// A stopped watch's own totals.
  public let totals: LiveTotals?
  /// Why a failed run stopped, in gdoc's own word for it, and the line of
  /// the note the cause sits on where it has one.
  public let stop: String?
  public let line: Int?
}

/// What the chevron row of Yesterday says.
public struct DaySummary: Sendable, Equatable {
  public let actions: Int
  public let documents: Int

  public init(actions: Int, documents: Int) {
    self.actions = actions
    self.documents = documents
  }
}

/// TODAY and Yesterday, each newest first.
public struct DaysFold: Sendable, Equatable {
  public let today: [DayRow]
  public let yesterday: [DayRow]

  /// What the one chevron row carries before a person opens it.
  public var yesterdaySummary: DaySummary {
    DaySummary(
      actions: yesterday.count, documents: Set(yesterday.compactMap(\.doc)).count)
  }

  /// What is behind Show more in TODAY.
  public var more: Int { max(0, today.count - Days.defaultRows) }

  /// The rows TODAY draws, which is the newest eight until a person opens it.
  public func visible(expanded: Bool) -> [DayRow] {
    expanded ? today : Array(today.prefix(Days.defaultRows))
  }
}

public enum Days {
  /// How long after the released line the write that approval sent may
  /// begin. The press and the write are one tool call of `gdoc mcp`, and one
  /// tool call is bounded by that server's own 200-second deadline
  /// (`internal/mcp`'s callDeadline), so a start later than this is a write
  /// of some other call and never the one the person approved.
  static let approvalWindow: TimeInterval = 200

  /// Eight rows, and the rest behind Show more.
  public static let defaultRows = 8

  /// The events into TODAY and Yesterday.
  ///
  /// - Parameters:
  ///   - now: the moment the panel is drawn at, whose local day is TODAY.
  ///   - calendar: the calendar the local day comes from, injected because
  ///     the day is the thing under test and the machine's zone is not it.
  ///   - alive: the kernel's answer for one pid.
  /// - Parameter carded: the `run|call` of every start NOW is holding a
  ///   stuck card for. Those rows are not written twice: a write whose
  ///   process went away is one card while the card is on the screen, and
  ///   the row here is what the card leaves behind when it clears or is
  ///   dismissed. The caller folds NOW first and hands the keys over,
  ///   because the horizon and the dismissals are that fold's.
  public static func fold(
    _ events: [Event], now: Date, calendar: Calendar, carded: Set<String> = [],
    alive: (Int32) -> Bool
  ) -> DaysFold {
    let ordered = Fold.inOrder(events)
    let titles = Fold.latestTitles(ordered)
    let calls = Fold.latestCallTitles(ordered)
    let rows = every(
      ordered, now: now, titles: titles, calls: calls, carded: carded, alive: alive)

    let yesterday = calendar.date(byAdding: .day, value: -1, to: now)

    return DaysFold(
      today: rows.filter { calendar.isDate($0.at, inSameDayAs: now) },
      yesterday: rows.filter { day in
        guard let yesterday else { return false }
        return calendar.isDate(day.at, inSameDayAs: yesterday)
      })
  }

  /// Every row the record holds, newest first. The index breaks a tie, so two
  /// rows at the same moment keep the order the record had them in.
  private static func every(
    _ ordered: [Event], now: Date, titles: [String: String], calls: [String: String],
    carded: Set<String>, alive: (Int32) -> Bool
  ) -> [DayRow] {
    var rows: [DayRow] = []
    var ended: Set<String> = []
    var held: [String: Event] = [:]
    let approved = approvals(ordered)
    let closings = endings(ordered)

    for event in ordered {
      if event.kind == .end {
        ended.insert(event.pairing)
        guard !event.wait, Now.stuckCommands.contains(event.command ?? "") else { continue }
        rows.append(
          action(event, titles: titles, calls: calls, ending: approved.byWrite[event.pairing]))
      } else if event.kind == .held {
        held[event.hold ?? ""] = event
        rows.append(
          raised(
            event, titles: titles, calls: calls, ending: closings[event.hold ?? ""]))
      } else if let ending = HoldEnding(event.kind) {
        // The write the approval sent carries it, so the closing line is
        // not a second row: scene r7 draws one Reply and no Hold closed.
        guard !approved.absorbed.contains(event.hold ?? "") else { continue }
        rows.append(
          closed(
            event, held: held[event.hold ?? ""], ending: ending, titles: titles, calls: calls))
      }
    }

    for event in ordered {
      guard event.kind == .start, !event.wait,
        Now.stuckCommands.contains(event.command ?? ""), !ended.contains(event.pairing),
        !carded.contains(event.pairing), !alive(event.pid)
      else { continue }
      rows.append(lost(event, titles: titles, calls: calls))
    }

    for session in Live.sessions(ordered, now: now, alive: alive) {
      rows.append(began(session))
      if let ended = session.ended { rows.append(stopped(session, at: ended)) }
    }

    return rows.enumerated()
      .sorted { left, right in
        left.element.at == right.element.at
          ? left.offset > right.offset : left.element.at > right.element.at
      }
      .map(\.element)
  }

  /// A hold the person approved, joined to the write that approval sent:
  /// that write's own pairing, and the hold ids whose closing line is then
  /// not a row of its own.
  private struct Approvals {
    var byWrite: [String: HoldEnding] = [:]
    var absorbed: Set<String> = []
  }

  /// Which write each approved hold sent.
  ///
  /// The record writes four lines for one thing that happened: the chat held
  /// the write, the person pressed the card, and the write went out with its
  /// own start and end. `mcprelease.go` takes the hold out before it sends,
  /// so the released line comes first and the write's end follows it.
  ///
  /// The ending line carries the hold's id and nothing else, so what joins
  /// it to the write is the one run the approval sent in: the chat that
  /// raised the hold is the one that sends it, and one gdoc process is one
  /// run.
  ///
  /// `landed` is not an approval. It is the write reaching the document
  /// another way, which is the model sending it again with no card, so its
  /// closing line stays a row of its own and says so.
  ///
  /// An approval whose write is in no part of the record this app read joins
  /// nothing, and its closing line stays a row of its own.
  private static func approvals(_ ordered: [Event]) -> Approvals {
    var held: [String: Event] = [:]
    var found = Approvals()

    for event in ordered {
      let id = event.hold ?? ""
      if event.kind == .held {
        held[id] = event
        continue
      }
      guard HoldEnding(event.kind) != nil else { continue }
      let opened = held.removeValue(forKey: id)
      guard event.kind == .released, let opened,
        let write = wrote(opened, after: event.at, in: ordered)
      else { continue }
      found.byWrite[write] = .released
      found.absorbed.insert(id)
    }
    return found
  }

  /// The write an approval sent, which is the next thing that run did: the
  /// first start inside the window after the released line, from the process
  /// that raised the hold, and it has to be the held line's own command on
  /// the held line's own document.
  ///
  /// The window is there because `mcprelease.go` writes the released line
  /// before it sends and the send may write no line at all: the document
  /// check reads the document, and a read that failed on the wire or a title
  /// that drifted ends the call before the command runs. Without a bound,
  /// the approval would reach forward to whatever that chat wrote on that
  /// document next, which may be an hour later and nobody's approval:
  /// testAnApprovalThatSentNothingClaimsNoLaterWrite.
  ///
  /// A second approval inside the window is the same case: the first one
  /// sent nothing the record kept, so it joins nothing rather than the write
  /// the second one sent.
  private static func wrote(_ held: Event, after released: Date, in ordered: [Event]) -> String? {
    let window = released...released.addingTimeInterval(approvalWindow)
    for event in ordered where event.run == held.run && window.contains(event.at) {
      if event.kind == .released, event.hold != held.hold { return nil }
      guard event.kind == .start, !event.wait else { continue }
      guard event.command == held.command, Fold.subject(event) == Fold.subject(held)
      else { return nil }
      // The end is what carries the row, so a write whose end never arrived
      // leaves the approval's own closing line to say what happened.
      return ordered.contains { $0.kind == .end && $0.pairing == event.pairing }
        ? event.pairing : nil
    }
    return nil
  }

  private static func action(
    _ event: Event, titles: [String: String], calls: [String: String], ending: HoldEnding?
  ) -> DayRow {
    row(
      .action, event, titles: titles, calls: calls, ok: event.ok ?? false,
      verified: event.verified, ms: event.ms, ending: ending, stop: event.stop, line: event.line)
  }

  private static func lost(
    _ event: Event, titles: [String: String], calls: [String: String]
  ) -> DayRow {
    row(.outcomeUnknown, event, titles: titles, calls: calls)
  }

  /// How each hold stopped waiting, by its id, so the row that raised one
  /// knows whether it is still waiting. The row is written in the present
  /// tense, and a hold the person approved an hour ago is not waiting for
  /// anybody: testAHoldThatClosedStopsSayingItIsWaiting.
  private static func endings(_ ordered: [Event]) -> [String: HoldEnding] {
    var found: [String: HoldEnding] = [:]
    for event in ordered {
      guard let ending = HoldEnding(event.kind) else { continue }
      found[event.hold ?? ""] = ending
    }
    return found
  }

  private static func raised(
    _ event: Event, titles: [String: String], calls: [String: String], ending: HoldEnding?
  ) -> DayRow {
    row(.holdRaised, event, titles: titles, calls: calls, rule: event.rule, ending: ending)
  }

  /// A hold that stopped waiting, with the held line's own words on it.
  ///
  /// The ending line carries the hold's id and nothing else:
  /// `go/internal/presence`'s released, expired, landed and dropped write no
  /// command, no document and no rule, so a row built from the ending alone
  /// would say only "Hold closed" and would not count towards the day's
  /// documents. The moment and the kind are the ending's; everything a
  /// person reads is the held line's: testTodayCarriesTheSyntheticRows.
  ///
  /// A held line this app never saw, because it is in a month the tail does
  /// not read, leaves the row as bare as the ending that closed it.
  private static func closed(
    _ event: Event, held: Event?, ending: HoldEnding, titles: [String: String],
    calls: [String: String]
  ) -> DayRow {
    guard let held else {
      return row(
        .holdClosed, event, titles: titles, calls: calls, rule: event.rule, ending: ending)
    }
    return DayRow(
      kind: .holdClosed, at: event.at, command: held.command, via: held.via,
      doc: Fold.subject(held), title: title(held, titles, calls), ok: nil, verified: nil,
      ms: nil, rule: held.rule, ending: ending, totals: nil, stop: nil, line: nil)
  }

  private static func began(_ session: LiveSession) -> DayRow {
    DayRow(
      kind: .liveStarted, at: session.since, command: "comments", via: "terminal",
      doc: session.doc, title: session.title, ok: nil, verified: nil, ms: nil, rule: nil,
      ending: nil, totals: nil, stop: nil, line: nil)
  }

  private static func stopped(_ session: LiveSession, at: Date) -> DayRow {
    DayRow(
      kind: .liveStopped, at: at, command: "comments", via: "terminal", doc: session.doc,
      title: session.title, ok: nil, verified: nil, ms: nil, rule: nil, ending: nil,
      totals: LiveTotals(
        since: session.since, ended: at, replied: session.replied,
        proposed: session.proposed, stoppedByYou: session.stoppedByYou),
      stop: nil, line: nil)
  }

  private static func row(
    _ kind: DayKind, _ event: Event, titles: [String: String], calls: [String: String],
    ok: Bool? = nil, verified: Bool? = nil, ms: Int64? = nil, rule: String? = nil,
    ending: HoldEnding? = nil, stop: String? = nil, line: Int? = nil
  ) -> DayRow {
    DayRow(
      kind: kind, at: event.at, command: event.command, via: event.via,
      doc: Fold.subject(event), title: title(event, titles, calls), ok: ok,
      verified: verified, ms: ms, rule: rule, ending: ending, totals: nil, stop: stop,
      line: line)
  }

  /// What a row is named by, which is `Fold.title`: the document's latest
  /// title, the call's own where the record named no document, and the
  /// line's own where it named neither. NOW's card calls the same function,
  /// so the card and the row cannot name one write two ways.
  private static func title(
    _ event: Event, _ titles: [String: String], _ calls: [String: String]
  ) -> String? {
    Fold.title(event, titles: titles, calls: calls)
  }
}
