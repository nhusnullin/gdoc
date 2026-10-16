// The panel as rows, before anything is drawn.
//
// The view draws this and decides nothing. So every line a person can read is
// a value a test can state, and the one thing a render test adds over these
// tests is whether the pixels came out at all.
//
// The words here are the app's reading of the record, which is the one place
// in this repository where a reading is allowed to use a word like "stopped":
// the binary printed the facts and this is the surface that judges them.

import Foundation
import PresenceCore

/// Which section a row came from, and how the view draws it.
public enum RowKind: String, Sendable, Equatable {
  case hold
  case running
  case watching
  case stopped
  case done
  case version
  case notice
}

/// The one symbol a finished row carries. `verified` is the five commands
/// that read their own write back; everywhere else there is nothing to say,
/// and a hollow check is never a failure.
public enum RowMark: String, Sendable, Equatable {
  case none
  case ok
  case failed
  case verified
  case unverified
}

/// One line of the panel: what it is, the sentence under it, the value on the
/// right, and the one symbol.
public struct Row: Sendable, Equatable {
  public let kind: RowKind
  public let leading: String
  public let detail: String?
  public let value: String?
  public let mark: RowMark

  public init(
    kind: RowKind, leading: String, detail: String? = nil, value: String? = nil,
    mark: RowMark = .none
  ) {
    self.kind = kind
    self.leading = leading
    self.detail = detail
    self.value = value
    self.mark = mark
  }
}

/// The whole panel: NOW, TODAY, the two versions, and the one notice about
/// the bundle under the running app.
public struct PanelModel: Sendable, Equatable {
  public let now: [Row]
  public let today: [Row]
  public let versions: [Row]
  public let notice: Row?

  public init(now: [Row], today: [Row], versions: [Row], notice: Row?) {
    self.now = now
    self.today = today
    self.versions = versions
    self.notice = notice
  }

  /// NOW is hidden entirely when nothing is happening, rather than shown
  /// empty: an empty section spends the panel's space on nothing.
  public var showsNow: Bool { !now.isEmpty }

  /// One glance into the rows.
  ///
  /// The calendar carries the zone and the locale every time in the panel is
  /// written in, so the fold's moments and the panel's clock agree and
  /// neither reads the machine's own settings in a test.
  public static func build(
    glance: Glance, appVersion: String, bundle: BundleState, calendar: Calendar
  ) -> PanelModel {
    let clock = Clock(calendar: calendar)
    return PanelModel(
      now: glance.now.map { row($0, clock: clock) },
      today: glance.today.map { row($0, clock: clock) },
      versions: [
        Row(kind: .version, leading: "gdoc presence", value: appVersion),
        Row(kind: .version, leading: "gdoc", value: glance.gdoc ?? "checkout build"),
      ],
      notice: notice(bundle))
  }

  private static func row(_ now: NowRow, clock: Clock) -> Row {
    switch now {
    case .hold(let hold):
      return Row(
        kind: .hold, leading: "waiting for you",
        detail: sentence([hold.command, hold.rule]),
        value: hold.ranOut ? "ran out" : hold.until.map(clock.time))
    case .running(let run):
      return Row(
        kind: .running, leading: run.title ?? run.command,
        detail: under(run), value: clock.time(run.started))
    case .watching(let run):
      return Row(
        kind: .watching, leading: run.title ?? run.command,
        detail: under(run), value: clock.time(run.started))
    case .stopped(let run):
      return Row(
        kind: .stopped, leading: run.title ?? run.command,
        detail: "stopped, outcome unknown", value: clock.time(run.started))
    }
  }

  private static func row(_ done: FinishedRun, clock: Clock) -> Row {
    Row(
      kind: .done, leading: done.title ?? done.command,
      detail: sentence([
        done.title == nil ? nil : done.command, done.via, clock.took(done.ms),
      ]),
      value: clock.time(done.ended), mark: mark(done))
  }

  /// What the end event said, and nothing else. A run that is not ok is
  /// failed; one of the five read-back writers carries its own verified, and
  /// a false there is a hollow check rather than a failure.
  private static func mark(_ done: FinishedRun) -> RowMark {
    guard done.ok else { return .failed }
    switch done.verified {
    case .some(true): return .verified
    case .some(false): return .unverified
    case .none: return .ok
    }
  }

  private static func notice(_ bundle: BundleState) -> Row? {
    switch bundle {
    case .same:
      return nil
    case .reopenNeeded(let version):
      return Row(kind: .notice, leading: "Updated to \(version). Quit and reopen")
    case .gone:
      return Row(kind: .notice, leading: "gdoc presence was removed")
    }
  }

  /// The second line of a live or stopped run: the command when the title
  /// took the first line, and where it was run from.
  private static func under(_ run: ActiveRun) -> String? {
    sentence([run.title == nil ? nil : run.command, run.via])
  }

  private static func sentence(_ parts: [String?]) -> String? {
    let kept = parts.compactMap { $0 }.filter { !$0.isEmpty }
    return kept.isEmpty ? nil : kept.joined(separator: " · ")
  }
}

/// The two numbers the panel writes: a time of day and an elapsed time.
///
/// Both are built per model rather than held in a global, because the zone
/// and the locale are injected and a formatter promises nothing about two
/// threads.
private struct Clock {
  private let formatter: DateFormatter

  init(calendar: Calendar) {
    let formatter = DateFormatter()
    formatter.calendar = calendar
    formatter.timeZone = calendar.timeZone
    formatter.locale = calendar.locale ?? Locale(identifier: "en_US_POSIX")
    formatter.dateFormat = "HH:mm"
    self.formatter = formatter
  }

  func time(_ date: Date) -> String { formatter.string(from: date) }

  /// How long a command took, as the record measured it. Under a second it
  /// is milliseconds, because a run that took 820 ms did not take 0.8 s to
  /// the person who waited for it.
  func took(_ ms: Int64?) -> String? {
    guard let ms, ms >= 0 else { return nil }
    if ms < 1_000 { return "\(ms)ms" }
    return String(format: "%.1fs", Double(ms) / 1_000)
  }
}
