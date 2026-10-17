// A stretch of the record into one glance: what is happening now, and what
// happened today.
//
// This is the whole reading the app does. It is a pure function of the
// events, the moment, the calendar and one question to the kernel, so every
// state the glyph and the panel can show is reachable from a fixture and
// nothing in it waits for a clock.
//
// The record is forgeable and best-effort: a line may be there twice, a start
// may have lost its end, and an end may have arrived whose start was in a
// month the binary has since removed. None of that may make the fold throw or
// lose a row.

import Foundation

/// A command that has a start and no end yet.
public struct ActiveRun: Sendable, Equatable {
  public let run: String
  public let call: Int64
  public let pid: Int32
  public let command: String
  public let via: String?
  public let doc: String?
  public let title: String?
  public let started: Date
  public let isWait: Bool
}

/// A chat write waiting for a person, from a held with no ending yet.
public struct ActiveHold: Sendable, Equatable {
  public let id: String
  public let command: String?
  public let via: String?
  public let doc: String?
  public let rule: String?
  public let held: Date
  public let until: Date?
  /// The hold's life ran out before now and no event said so, because the
  /// process that owned it died with it. The panel shows it as having run
  /// out, which is all the record can support.
  public let ranOut: Bool
}

/// A command that ended, with the four facts its end event carried.
public struct FinishedRun: Sendable, Equatable {
  public let run: String
  public let call: Int64
  public let command: String
  public let via: String?
  public let doc: String?
  public let title: String?
  /// The envelope's own ok. An end with no ok is read as not ok: the writer
  /// always writes one, so a line without it is not a line gdoc wrote.
  public let ok: Bool
  /// Only the five commands that read their own write back carry this. Absent
  /// everywhere else, and never a judgement the app makes.
  public let verified: Bool?
  public let ms: Int64?
  public let ended: Date
}

/// One line of NOW, in the order the panel reads them.
public enum NowRow: Sendable, Equatable {
  case hold(ActiveHold)
  case running(ActiveRun)
  case watching(ActiveRun)
  case stopped(ActiveRun)
}

/// What the glyph and the panel are drawn from.
public struct Glance: Sendable, Equatable {
  /// The holds first, then the live starts, then the starts whose process is
  /// gone, each band newest first. A hold is first because it is the one row
  /// that is waiting for the person reading it.
  public let now: [NowRow]
  /// Every command that ended in the local day of the moment folded, newest
  /// first.
  public let today: [FinishedRun]
  /// The version of the last record that named one, which is the gdoc that
  /// wrote it. Absent for a checkout build, which names no release.
  public let gdoc: String?

  public var holds: [ActiveHold] {
    now.compactMap { if case .hold(let hold) = $0 { return hold } else { return nil } }
  }

  public var live: [ActiveRun] {
    now.compactMap {
      switch $0 {
      case .running(let run), .watching(let run): return run
      case .hold, .stopped: return nil
      }
    }
  }

  public var stopped: [ActiveRun] {
    now.compactMap { if case .stopped(let run) = $0 { return run } else { return nil } }
  }

  public var isIdle: Bool { now.isEmpty }
}

public enum Fold {
  /// The events into one glance.
  ///
  /// alive is the kernel's answer for one pid, injected so the fold is a pure
  /// function and a dead run is a fixture rather than a process a test has to
  /// kill.
  public static func glance(
    _ events: [Event], now: Date, calendar: Calendar, alive: (Int32) -> Bool
  ) -> Glance {
    var active: [String: ActiveRun] = [:]
    var holds: [String: ActiveHold] = [:]
    var finished: [FinishedRun] = []
    var seen: Set<String> = []
    var version: String?

    for event in events {
      guard seen.insert(key(event)).inserted else { continue }
      if let gdoc = event.gdoc, !gdoc.isEmpty { version = gdoc }

      switch event.kind {
      case .start:
        active[event.pairing] = run(event)
      case .end:
        active.removeValue(forKey: event.pairing)
        finished.append(end(event))
      case .step:
        // The parts of a publish have no meaning in the panel yet: M24a's job
        // card is what reads them. A kind this fold skips is a line the app
        // ignores, which is what every unknown kind already was.
        break
      case .held:
        holds[event.hold ?? ""] = hold(event, now: now)
      case .released, .expired, .landed, .dropped:
        holds.removeValue(forKey: event.hold ?? "")
      }
    }

    let waiting = holds.values.sorted { $0.held > $1.held }
    let running = active.values.filter { alive($0.pid) }.sorted { $0.started > $1.started }
    let gone = active.values.filter { !alive($0.pid) }.sorted { $0.started > $1.started }

    var rows = waiting.map(NowRow.hold)
    rows += running.map { $0.isWait ? NowRow.watching($0) : NowRow.running($0) }
    rows += gone.map(NowRow.stopped)

    let today =
      finished
      .filter { calendar.isDate($0.ended, inSameDayAs: now) }
      .sorted { $0.ended > $1.ended }

    return Glance(now: rows, today: today, gdoc: version)
  }

  /// What makes one record the same record as another: a start and its run
  /// and call, or a hold's ending and its id. A line the writer wrote twice,
  /// or a file a person concatenated, is then one row and not two.
  private static func key(_ event: Event) -> String {
    switch event.kind {
    case .start, .end:
      return "\(event.kind.rawValue)|\(event.pairing)"
    case .step:
      // A run has four of these, so the part's own name is in the key: without
      // it a file somebody concatenated would lose three of the four.
      return "\(event.kind.rawValue)|\(event.pairing)|\(event.step ?? "")"
    case .held, .released, .expired, .landed, .dropped:
      return "\(event.kind.rawValue)|\(event.hold ?? "")"
    }
  }

  private static func run(_ event: Event) -> ActiveRun {
    ActiveRun(
      run: event.run, call: event.call ?? 0, pid: event.pid, command: event.command ?? "",
      via: event.via, doc: event.doc, title: event.title, started: event.at, isWait: event.wait)
  }

  private static func end(_ event: Event) -> FinishedRun {
    FinishedRun(
      run: event.run, call: event.call ?? 0, command: event.command ?? "", via: event.via,
      doc: event.doc, title: event.title, ok: event.ok ?? false, verified: event.verified,
      ms: event.ms, ended: event.at)
  }

  private static func hold(_ event: Event, now: Date) -> ActiveHold {
    ActiveHold(
      id: event.hold ?? "", command: event.command, via: event.via, doc: event.doc,
      rule: event.rule, held: event.at, until: event.until,
      ranOut: event.until.map { $0 < now } ?? false)
  }
}

// The two walks the sections beside NOW share. They are here rather than in
// Now.swift because every fold wants them: the record is forgeable and
// best-effort, so a line may be there twice and two month files may arrive in
// either order.
extension Fold {
  /// The events once each, oldest first.
  ///
  /// A line the writer wrote twice, or a file a person concatenated, is one
  /// event; the index breaks a tie, so two events at the same moment keep the
  /// order the record had them in.
  public static func inOrder(_ events: [Event]) -> [Event] {
    var seen: Set<String> = []
    let once = events.filter { seen.insert(key($0)).inserted }
    return once.enumerated()
      .sorted { left, right in
        left.element.at == right.element.at
          ? left.offset < right.offset : left.element.at < right.element.at
      }
      .map(\.element)
  }

  /// The document a line is about.
  ///
  /// It is the one the command was handed, and the one a publish created
  /// where it was handed none: a publish names no document until it has
  /// made one, so `made` is that line's document everywhere the panel
  /// groups by one. Nothing else writes `made`.
  public static func subject(_ event: Event) -> String? {
    if let doc = event.doc, !doc.isEmpty { return doc }
    guard let made = event.made, !made.isEmpty else { return nil }
    return made
  }

  /// Every document's latest title, by document id.
  ///
  /// A start names no title and a held line names none either, so a card for
  /// a document the record has read before takes the title from that read.
  /// The latest wins, because that is the name the person sees in Drive.
  public static func latestTitles(_ events: [Event]) -> [String: String] {
    var titles: [String: String] = [:]
    var when: [String: Date] = [:]
    for event in events {
      guard let doc = subject(event), let title = event.title, !title.isEmpty else { continue }
      if let had = when[doc], had > event.at { continue }
      titles[doc] = title
      when[doc] = event.at
    }
    return titles
  }
}
