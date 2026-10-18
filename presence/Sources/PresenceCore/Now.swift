// NOW: the one section of the panel that is about this moment.
//
// It holds four classes of row, in this order: a chat write waiting for a
// person, a write that failed and has not been answered, a job that is
// running, and a live watch. Inside a class the oldest is above the newer
// one, because the thing that has waited longest is the thing most likely
// forgotten. That order is scene e3 of
// docs/design/2026-10-17-presence-screens-v3.html.
//
// Three rules decide what a row is, and all three are the M24 specification's
// product decisions rather than this file's reading:
//
//   - A write that failed is stuck until its key clears or the person
//     dismisses it. The key is the command and the document; a failed
//     publish names no document, so any later publish clears it.
//   - A failed read is not stuck, because it changed nothing. An end with
//     `verified: false` is never stuck either: the write happened and the
//     read-back did not confirm it, which is a hollow check and not a
//     failure.
//   - A start with no end whose pid is gone is stuck when it is a write
//     started inside fourteen days, and otherwise an outcome nobody wrote
//     down. That is the bound the backlog item asked for: without it a
//     run killed with Ctrl-C sits in NOW for as long as the record keeps it.
//
// The clock, the kernel's answer about a pid and the person's dismissals come
// in as arguments, so every row is reachable from a fixture and nothing here
// reads the machine.

import Foundation

/// A chat write waiting for a person, with the title the panel can show.
///
/// The held line carries no title, because the writer has no document read to
/// take one from, so the card takes the latest title the record named for
/// that document.
public struct HoldCard: Sendable, Equatable {
  public let id: String
  public let run: String
  public let command: String?
  public let via: String?
  public let doc: String?
  public let title: String?
  public let rule: String?
  public let held: Date
  public let until: Date?
  /// The hold's life ran out before now and no event said so, because the
  /// process that owned it died with it.
  public let ranOut: Bool
}

/// Why a write is stuck. Both are facts off the record: an end that said it
/// failed, or a start whose process is not there any more.
public enum StuckCause: String, Sendable, Equatable, CaseIterable {
  case failed
  case stopped
}

/// A write that failed and has not been answered.
public struct StuckWrite: Sendable, Equatable {
  public let run: String
  public let call: Int64
  public let command: String
  public let doc: String?
  /// The document the command was handed, where it was handed one. It is
  /// what a later start can name again, and it is not the same as `doc`: a
  /// publish is handed none and `doc` falls back to the id it made, which no
  /// start will ever carry. The key that clears the card is this one.
  public let handed: String?
  public let title: String?
  public let via: String?
  /// When it failed, or when it started for a process that is gone.
  public let at: Date
  public let cause: StuckCause
  /// Why it stopped, in one of gdoc's seven fixed words, and the line of
  /// the note the cause sits on where it has one. A process that went away
  /// wrote neither.
  public let stop: String?
  public let line: Int?

  /// What a dismissal names: the run and the call, as the record pairs them.
  public var key: String { "\(run)|\(call)" }
}

/// A command that is still running, with the title the panel can show.
public struct NowRun: Sendable, Equatable {
  public let run: String
  public let call: Int64
  public let pid: Int32
  public let command: String
  public let via: String?
  public let doc: String?
  public let title: String?
  public let started: Date
  public let isWait: Bool
  /// The part of the run that began last, which is the part running now:
  /// gdoc writes a step line immediately before the part it names starts.
  /// Only a publish has parts.
  public let step: String?
  /// That part's place in the list of parts, counted from one, and how many
  /// parts there are. They are places and never a measure of work done.
  public let index: Int?
  public let count: Int?

  public var key: String { "\(run)|\(call)" }

  /// How far along the list of parts the run is. It is the place and not a
  /// forecast: nothing says what a part costs, so nothing here says when
  /// the run finishes.
  public var progress: Double? {
    guard let index, let count, count > 0 else { return nil }
    return Double(index) / Double(count)
  }
}

/// A start whose process is gone and which NOW does not keep: the document
/// it was about says the outcome is one nobody wrote down.
public struct UnknownOutcome: Sendable, Equatable {
  public let run: String
  public let call: Int64
  public let command: String
  public let via: String?
  public let doc: String?
  public let title: String?
  public let started: Date
}

/// The last write that read itself back and matched. The menu bar shows a
/// check for five seconds after one of these arrives live, so the bar needs
/// to tell this end from the one it saw a moment ago.
public struct VerifiedWrite: Sendable, Equatable {
  public let run: String
  public let call: Int64
  public let command: String
  public let doc: String?
  public let title: String?
  public let ended: Date
}

/// One row of NOW.
public enum NowEntry: Sendable, Equatable {
  case hold(HoldCard)
  case stuck(StuckWrite)
  case job(NowRun)
  case watch(NowRun)
}

/// NOW, and the two things the rest of the panel needs from the same walk.
public struct NowFold: Sendable, Equatable {
  /// The holds, then the stuck writes, then the running jobs, then the live
  /// watches, each class oldest first.
  public let entries: [NowEntry]
  /// The starts whose process is gone that NOW does not keep.
  public let unknown: [UnknownOutcome]
  public let lastVerified: VerifiedWrite?
  /// Every hold the record has open, the dismissed ones included. `holds` is
  /// what the panel draws and this is what the record says is waiting, which
  /// is what a sign-out would drop: a card nobody can see is a hold all the
  /// same. testTheSignOutCountsTheHoldsTheRecordHasOpen.
  public let everyHold: [HoldCard]

  public var holds: [HoldCard] {
    entries.compactMap { if case .hold(let card) = $0 { return card } else { return nil } }
  }

  public var stuck: [StuckWrite] {
    entries.compactMap { if case .stuck(let write) = $0 { return write } else { return nil } }
  }

  public var jobs: [NowRun] {
    entries.compactMap { if case .job(let run) = $0 { return run } else { return nil } }
  }

  public var watches: [NowRun] {
    entries.compactMap { if case .watch(let run) = $0 { return run } else { return nil } }
  }

  public var isEmpty: Bool { entries.isEmpty }
}

public enum Now {
  /// The seven commands that change a document. A failure of one of these is
  /// something a person has to answer; a failed read is not. The list is the
  /// specification's, written out here rather than derived from anything.
  public static let stuckCommands: Set<String> = [
    "reply", "propose", "withdraw", "annotate", "publish", "restyle", "export",
  ]

  /// Fourteen days: how far back a start with no end may be and still be a
  /// card rather than a row of its document alone.
  public static let horizon: TimeInterval = 14 * 24 * 60 * 60

  /// The events into NOW.
  ///
  /// - Parameters:
  ///   - now: the moment the panel is drawn at.
  ///   - dismissed: the keys the person hid, a run and its call for a write
  ///     and a hold's own id for a hold. The app's state, never the record's.
  ///   - alive: the kernel's answer for one pid, injected so a dead run is a
  ///     fixture rather than a process a test has to kill.
  public static func fold(
    _ events: [Event], now: Date, dismissed: Set<String> = [], alive: (Int32) -> Bool
  ) -> NowFold {
    let ordered = Fold.inOrder(events)
    let titles = Fold.latestTitles(ordered)
    let calls = Fold.latestCallTitles(ordered)

    var starts: [String: Event] = [:]
    var steps: [String: Event] = [:]
    var ended: Set<String> = []
    var held: [String: Event] = [:]
    var failures: [StuckWrite] = []
    var verified: VerifiedWrite?

    for event in ordered {
      switch event.kind {
      case .start:
        starts[event.pairing] = event
      case .end:
        ended.insert(event.pairing)
        // `ok` as well as `verified`: a publish whose pairing failed and
        // whose rollback failed answers `ok: false` with the read-back it
        // did manage still in it, and a green check over a run that NOW
        // holds a stuck card for says two things at once.
        if event.ok == true, event.verified == true {
          verified = check(event, titles: titles, calls: calls)
        }
        if event.ok == false, stuckCommands.contains(event.command ?? "") {
          failures.append(stuck(event, at: event.at, cause: .failed, titles: titles, calls: calls))
        }
      case .held:
        held[event.hold ?? ""] = event
      case .released, .expired, .landed, .dropped:
        held.removeValue(forKey: event.hold ?? "")
      case .step:
        // The parts arrive in order, so the last one read is the one that
        // began last, which is the one running.
        steps[event.pairing] = event
      }
    }

    let everyStart = ordered.filter { $0.kind == .start }

    var jobs: [NowRun] = []
    var watches: [NowRun] = []
    var unknown: [UnknownOutcome] = []

    // A held write is never one of these: `mcpJudge` answers the hold before
    // the command runs, so nothing wrote a start for it. The pairing is the
    // whole of what keeps a run out of NOW, and a run is not one call: a
    // chat session with a hold open goes on taking other calls, and dropping
    // every start of that run would hide the one still running and the one
    // whose process died: testAHoldHidesNoOtherCallOfTheSameRun.
    for (pairing, start) in starts where !ended.contains(pairing) {
      if alive(start.pid) {
        let row = run(start, step: steps[pairing], titles: titles, calls: calls)
        if row.isWait { watches.append(row) } else { jobs.append(row) }
        continue
      }
      let write = stuckCommands.contains(start.command ?? "")
      if write, start.at >= now.addingTimeInterval(-horizon) {
        failures.append(stuck(start, at: start.at, cause: .stopped, titles: titles, calls: calls))
      } else {
        unknown.append(outcome(start, titles: titles, calls: calls))
      }
    }

    let answered = failures.filter { cleared($0, by: everyStart) || dismissed.contains($0.key) }
    let open = failures.filter { !cleared($0, by: everyStart) && !dismissed.contains($0.key) }

    // A stuck card that cleared or was dismissed leaves no card, and
    // nothing in the record moves: a dismissal hides a card and rewrites
    // nothing. Only the ones whose end never came are an outcome nobody
    // wrote down, because the others have an end that said how it went.
    for write in answered where write.cause == .stopped {
      if let start = starts["\(write.run)|\(write.call)"] {
        unknown.append(outcome(start, titles: titles, calls: calls))
      }
    }

    let everyHold =
      held.values
      .map { hold($0, now: now, titles: titles, calls: calls) }
      .sorted { $0.held < $1.held }
    let cards = everyHold.filter { !dismissed.contains($0.id) }

    var entries = cards.map(NowEntry.hold)
    entries += open.sorted { $0.at < $1.at }.map(NowEntry.stuck)
    entries += jobs.sorted { $0.started < $1.started }.map(NowEntry.job)
    entries += watches.sorted { $0.started < $1.started }.map(NowEntry.watch)

    return NowFold(
      entries: entries, unknown: unknown.sorted { $0.started < $1.started },
      lastVerified: verified, everyHold: everyHold)
  }

  /// Whether a later start answers a stuck write. The same command on the
  /// same document, or, where the write named no document, the same command
  /// anywhere: a failed publish names none, and the next publish is the
  /// answer to it.
  ///
  /// The document it asks about is the one the run was handed and never the
  /// one it made: a publish that created a document and then failed is a
  /// card with an id on it, and no publish start can carry that id, because
  /// the command is handed no document. Matching on it would leave the card
  /// on the screen until somebody dismissed it.
  private static func cleared(_ write: StuckWrite, by starts: [Event]) -> Bool {
    starts.contains { start in
      guard start.command == write.command, start.at > write.at else { return false }
      guard let doc = write.handed else { return true }
      return start.doc == doc
    }
  }

  private static func stuck(
    _ event: Event, at: Date, cause: StuckCause, titles: [String: String],
    calls: [String: String]
  ) -> StuckWrite {
    StuckWrite(
      run: event.run, call: event.call ?? 0, command: event.command ?? "",
      doc: Fold.subject(event), handed: handed(event), title: title(event, titles, calls),
      via: event.via, at: at, cause: cause, stop: event.stop, line: event.line)
  }

  /// One running command, with the part it is on where the record names one.
  /// A publish is handed no document, so its name is whichever of its own
  /// lines carried one: the start, a step, or nothing yet.
  private static func run(
    _ event: Event, step: Event?, titles: [String: String], calls: [String: String]
  ) -> NowRun {
    NowRun(
      run: event.run, call: event.call ?? 0, pid: event.pid, command: event.command ?? "",
      via: event.via, doc: Fold.subject(event),
      title: title(event, titles, calls), started: event.at,
      isWait: event.wait, step: step?.step, index: step?.index, count: step?.count)
  }

  private static func outcome(
    _ event: Event, titles: [String: String], calls: [String: String]
  ) -> UnknownOutcome {
    UnknownOutcome(
      run: event.run, call: event.call ?? 0, command: event.command ?? "", via: event.via,
      doc: Fold.subject(event), title: title(event, titles, calls), started: event.at)
  }

  private static func hold(
    _ event: Event, now: Date, titles: [String: String], calls: [String: String]
  ) -> HoldCard {
    HoldCard(
      id: event.hold ?? "", run: event.run, command: event.command, via: event.via,
      doc: Fold.subject(event), title: title(event, titles, calls), rule: event.rule,
      held: event.at,
      until: event.until, ranOut: event.until.map { $0 < now } ?? false)
  }

  private static func check(
    _ event: Event, titles: [String: String], calls: [String: String]
  ) -> VerifiedWrite {
    VerifiedWrite(
      run: event.run, call: event.call ?? 0, command: event.command ?? "",
      doc: Fold.subject(event), title: title(event, titles, calls), ended: event.at)
  }

  /// The document the line was handed, which is nothing where the field is
  /// absent or empty: `cmd/gdoc` writes the empty string for every command
  /// whose first word is not a document.
  private static func handed(_ event: Event) -> String? {
    guard let doc = event.doc, !doc.isEmpty else { return nil }
    return doc
  }

  /// What a card is named by, which is `Fold.title`: the document's latest
  /// title, the call's own where the record named no document, and the
  /// line's own where it named neither.
  ///
  /// It is the fold's own function and not a copy of it, because the card
  /// here and the row in DOCUMENTS name one write, and a publish is the write
  /// that makes the rule: handed no document, it has only the name its own
  /// lines carried, which is a step after the render or the end of a publish
  /// that worked.
  private static func title(
    _ event: Event, _ titles: [String: String], _ calls: [String: String]
  ) -> String? {
    Fold.title(event, titles: titles, calls: calls)
  }
}
