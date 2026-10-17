// Every word the panel writes about a write, a moment or a stretch of time.
//
// The panel is a menu bar's worth of room, so a word in it is a decision and
// not a label: "not read back" and "failed" are two different things to the
// person reading them. Each word here is a literal, taken from
// docs/design/2026-10-17-presence-screens-v3.html, where the `LOG` rows carry
// the headings, the `DOC` rows the past tense, and the live rows the clock
// and the stretches.
//
// Two rules hold over the whole file:
//
//   - It states facts and never a forecast, which is principle 5. Nothing
//     here says when a job finishes, how much of it is still to come, or how
//     long a hold has before it runs out: a hold's own deadline is a moment,
//     written as a moment.
//   - A command with no word here has no word at all. `via: code` is M24b's
//     field and `read` is not a write, so both answer nil rather than a
//     guess, and the row that asked drops the piece.
//
// WordsTests holds each of them.

import Foundation
import PresenceCore

/// The words, as functions of the one fact each needs.
public enum Words {
  /// What one write is called: the heading a log row opens with, and the past
  /// tense a document row carries.
  public struct Verb: Equatable, Sendable {
    public let heading: String
    public let past: String

    public init(heading: String, past: String) {
      self.heading = heading
      self.past = past
    }
  }

  /// The seven writes, keyed by the command the record names. The keys are
  /// `Now.stuckCommands`, because the set of writes a person has to answer
  /// for is the set of writes the panel has words for.
  ///
  /// Six pairs are read off the screens file. Withdraw is in no scene, so it
  /// follows Proposal's shape: the thing it makes, and what gdoc did.
  public static let verbs: [String: Verb] = [
    "reply": Verb(heading: "Reply", past: "replied"),
    "propose": Verb(heading: "Proposal", past: "proposed"),
    "withdraw": Verb(heading: "Withdrawal", past: "withdrew"),
    "annotate": Verb(heading: "Comment", past: "commented"),
    "publish": Verb(heading: "Publish", past: "published"),
    "restyle": Verb(heading: "Restyle", past: "restyled"),
    "export": Verb(heading: "Export", past: "exported"),
  ]

  /// The two vias the record writes today, in the panel's words.
  public static let vias: [String: String] = [
    "terminal": "from the terminal",
    "chat": "from Desktop chat",
  ]

  /// How many days back a weekday still names one day. A weekday seven days
  /// back names two, so that is where the date starts.
  public static let weekdayHorizon = 6

  /// The words for one command, or none where the command is not a write.
  public static func verb(_ command: String?) -> Verb? {
    guard let command, let found = verbs[command] else { return nil }
    return found
  }

  /// Where a command came from, or none where the record did not say, or
  /// said something this version has no word for.
  public static func via(_ value: String?) -> String? {
    guard let value, let found = vias[value] else { return nil }
    return found
  }

  /// What a read-back came to. A false is a hollow check and not a failure:
  /// the write happened and the second route did not confirm it, so the word
  /// says that and nothing more.
  public static func verified(_ state: Bool?) -> String? {
    guard let state else { return nil }
    return state ? "verified" : "not read back"
  }

  /// One moment, as near or as far as it is: the clock inside the local day,
  /// Yesterday the day before, the weekday inside the week, and the date past
  /// it.
  ///
  /// The day is the local day the calendar names, never the last 24 hours, so
  /// a line written at one in the morning reads as a clock and not as
  /// Yesterday.
  public static func time(_ moment: Date, now: Date, calendar: Calendar) -> String {
    let days =
      calendar.dateComponents(
        [.day], from: calendar.startOfDay(for: moment), to: calendar.startOfDay(for: now)
      ).day ?? 0

    switch days {
    case ..<1: return written("HH:mm", moment, calendar)
    case 1: return "Yesterday"
    case 2...weekdayHorizon: return written("EEE", moment, calendar)
    default: return written("d MMM", moment, calendar)
    }
  }

  /// One stretch of time: minutes under an hour, then hours, with the minutes
  /// beside them where there are any.
  ///
  /// The minute a stretch is inside is the measurement, so the seconds go and
  /// nothing is rounded up. Under a minute, and a clock that went backwards,
  /// are both no minutes at all.
  public static func duration(_ seconds: TimeInterval) -> String {
    let minutes = seconds > 0 ? Int(seconds / 60) : 0
    let hours = minutes / 60
    let rest = minutes % 60

    if hours == 0 { return "\(minutes) min" }
    if rest == 0 { return "\(hours) h" }
    return "\(hours) h \(rest) min"
  }

  /// A moment in the panel's own English, in the calendar's zone.
  ///
  /// The zone is the person's and the words are not: every other word in the
  /// panel is an English literal, so a weekday is read with en_US_POSIX
  /// rather than with whatever the machine's language is.
  private static func written(_ template: String, _ moment: Date, _ calendar: Calendar) -> String {
    let formatter = DateFormatter()
    formatter.calendar = calendar
    formatter.timeZone = calendar.timeZone
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.dateFormat = template
    return formatter.string(from: moment)
  }
}
