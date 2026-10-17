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
/// top: landed is the write the person approved, released is the person
/// saying no, expired is the hold's own life running out.
public enum HoldEnding: String, Sendable, Equatable, CaseIterable {
  case released
  case expired
  case landed

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
  /// Eight rows, and the rest behind Show more.
  public static let defaultRows = 8

  /// The events into TODAY and Yesterday.
  ///
  /// - Parameters:
  ///   - now: the moment the panel is drawn at, whose local day is TODAY.
  ///   - calendar: the calendar the local day comes from, injected because
  ///     the day is the thing under test and the machine's zone is not it.
  ///   - alive: the kernel's answer for one pid.
  public static func fold(
    _ events: [Event], now: Date, calendar: Calendar, alive: (Int32) -> Bool
  ) -> DaysFold {
    let ordered = Fold.inOrder(events)
    let titles = Fold.latestTitles(ordered)
    let rows = every(ordered, now: now, titles: titles, alive: alive)

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
    _ ordered: [Event], now: Date, titles: [String: String], alive: (Int32) -> Bool
  ) -> [DayRow] {
    var rows: [DayRow] = []
    var ended: Set<String> = []

    for event in ordered {
      if event.kind == .end {
        ended.insert(event.pairing)
        guard !event.wait, Now.stuckCommands.contains(event.command ?? "") else { continue }
        rows.append(action(event, titles: titles))
      } else if event.kind == .held {
        rows.append(raised(event, titles: titles))
      } else if let ending = HoldEnding(event.kind) {
        rows.append(closed(event, ending: ending, titles: titles))
      }
    }

    for event in ordered {
      guard event.kind == .start, !event.wait, !alive(event.pid),
        Now.stuckCommands.contains(event.command ?? ""), !ended.contains(event.pairing)
      else { continue }
      rows.append(lost(event, titles: titles))
    }

    for session in Live.sessions(ordered, now: now) {
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

  private static func action(_ event: Event, titles: [String: String]) -> DayRow {
    row(
      .action, event, titles: titles, ok: event.ok ?? false, verified: event.verified,
      ms: event.ms)
  }

  private static func lost(_ event: Event, titles: [String: String]) -> DayRow {
    row(.outcomeUnknown, event, titles: titles)
  }

  private static func raised(_ event: Event, titles: [String: String]) -> DayRow {
    row(.holdRaised, event, titles: titles, rule: event.rule)
  }

  private static func closed(
    _ event: Event, ending: HoldEnding, titles: [String: String]
  ) -> DayRow {
    row(.holdClosed, event, titles: titles, rule: event.rule, ending: ending)
  }

  private static func began(_ session: LiveSession) -> DayRow {
    DayRow(
      kind: .liveStarted, at: session.since, command: "comments", via: "terminal",
      doc: session.doc, title: session.title, ok: nil, verified: nil, ms: nil, rule: nil,
      ending: nil, totals: nil)
  }

  private static func stopped(_ session: LiveSession, at: Date) -> DayRow {
    DayRow(
      kind: .liveStopped, at: at, command: "comments", via: "terminal", doc: session.doc,
      title: session.title, ok: nil, verified: nil, ms: nil, rule: nil, ending: nil,
      totals: LiveTotals(
        since: session.since, ended: at, replied: session.replied,
        proposed: session.proposed))
  }

  private static func row(
    _ kind: DayKind, _ event: Event, titles: [String: String], ok: Bool? = nil,
    verified: Bool? = nil, ms: Int64? = nil, rule: String? = nil, ending: HoldEnding? = nil
  ) -> DayRow {
    DayRow(
      kind: kind, at: event.at, command: event.command, via: event.via, doc: event.doc,
      title: title(event, titles), ok: ok, verified: verified, ms: ms, rule: rule,
      ending: ending, totals: nil)
  }

  /// The document's latest title, and the line's own only where the record
  /// never named one for that document.
  private static func title(_ event: Event, _ titles: [String: String]) -> String? {
    guard let doc = event.doc, let latest = titles[doc] else { return event.title }
    return latest
  }
}
