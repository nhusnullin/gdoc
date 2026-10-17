// The words the panel writes, every one of them a literal.
//
// A sentence a person reads in the menu bar is a decision, so each word here
// is stated as a literal rather than read out of the thing it checks. The
// source of truth is docs/design/2026-10-17-presence-screens-v3.html: its
// `LOG` rows carry the headings, its `DOC` rows the past-tense verbs, and its
// live rows the clock and the durations.

import Foundation
import PresenceCore
import XCTest

@testable import PresenceUI

final class WordsTests: XCTestCase {
  /// The seven writes, each with the heading a log row opens with and the
  /// verb a document row carries. Six of the fourteen are read off the
  /// screens file: Publish, Restyle, Export, Comment, Reply, Proposal, and
  /// published, restyled, exported, commented, replied, proposed. Withdraw
  /// appears in no scene, so its two words follow Proposal's shape.
  func testTheVerbsAreTheSevenWords() {
    XCTAssertEqual(Words.verbs.count, 7)
    XCTAssertEqual(Set(Words.verbs.keys), Now.stuckCommands)

    XCTAssertEqual(Words.verb("reply"), Words.Verb(heading: "Reply", past: "replied"))
    XCTAssertEqual(Words.verb("propose"), Words.Verb(heading: "Proposal", past: "proposed"))
    XCTAssertEqual(Words.verb("withdraw"), Words.Verb(heading: "Withdrawal", past: "withdrew"))
    XCTAssertEqual(Words.verb("annotate"), Words.Verb(heading: "Comment", past: "commented"))
    XCTAssertEqual(Words.verb("publish"), Words.Verb(heading: "Publish", past: "published"))
    XCTAssertEqual(Words.verb("restyle"), Words.Verb(heading: "Restyle", past: "restyled"))
    XCTAssertEqual(Words.verb("export"), Words.Verb(heading: "Export", past: "exported"))

    // A read is not a write and has no word here; neither has a command this
    // app has never heard of.
    XCTAssertNil(Words.verb("read"))
    XCTAssertNil(Words.verb("comments"))
    XCTAssertNil(Words.verb(""))
    XCTAssertNil(Words.verb(nil))
  }

  /// The same seven as present participles, which a job card and a
  /// document that is being written to both say. `Publishing · Chapter 4`
  /// is scene r1 and `publishing` is that scene's document row.
  func testTheSevenRunningWordsAreTheParticiples() {
    XCTAssertEqual(Words.participles.count, 7)
    XCTAssertEqual(Set(Words.participles.keys), Now.stuckCommands)

    XCTAssertEqual(Words.running("publish"), "Publishing")
    XCTAssertEqual(Words.running("restyle"), "Restyling")
    XCTAssertEqual(Words.running("export"), "Exporting")
    XCTAssertEqual(Words.running("reply"), "Replying")
    XCTAssertEqual(Words.running("propose"), "Proposing")
    XCTAssertEqual(Words.running("withdraw"), "Withdrawing")
    XCTAssertEqual(Words.running("annotate"), "Commenting")

    XCTAssertNil(Words.running("read"))
    XCTAssertNil(Words.running(nil))
  }

  /// `since 13:10` today, Yesterday the day before, `Mon` inside the week,
  /// and a date past it, because a weekday seven days back names two days.
  func testTimesAreClockYesterdayWeekdayThenDate() throws {
    let calendar = try utc()
    let now = try moment("2026-10-17T13:10:00Z")

    XCTAssertEqual(Words.time(now, now: now, calendar: calendar), "13:10")
    XCTAssertEqual(
      Words.time(try moment("2026-10-17T09:05:00Z"), now: now, calendar: calendar), "09:05")
    XCTAssertEqual(
      Words.time(try moment("2026-10-16T23:59:00Z"), now: now, calendar: calendar), "Yesterday")
    XCTAssertEqual(
      Words.time(try moment("2026-10-16T00:00:00Z"), now: now, calendar: calendar), "Yesterday")
    // 2026-10-12 is a Monday, which is the `published Mon` of scene e1.
    XCTAssertEqual(
      Words.time(try moment("2026-10-12T08:00:00Z"), now: now, calendar: calendar), "Mon")
    XCTAssertEqual(
      Words.time(try moment("2026-10-15T08:00:00Z"), now: now, calendar: calendar), "Thu")
    // Six days back is the last day a weekday names once.
    XCTAssertEqual(
      Words.time(try moment("2026-10-11T08:00:00Z"), now: now, calendar: calendar), "Sun")
    XCTAssertEqual(
      Words.time(try moment("2026-10-10T08:00:00Z"), now: now, calendar: calendar), "10 Oct")
    XCTAssertEqual(
      Words.time(try moment("2026-10-03T08:00:00Z"), now: now, calendar: calendar), "3 Oct")

    // The day is the local day and never the clock's own 24 hours: 01:30 in
    // Amsterdam is yesterday to a panel drawn at 13:10 UTC the day before.
    var amsterdam = Calendar(identifier: .gregorian)
    amsterdam.timeZone = try XCTUnwrap(TimeZone(identifier: "Europe/Amsterdam"))
    amsterdam.locale = Locale(identifier: "en_US_POSIX")
    XCTAssertEqual(
      Words.time(try moment("2026-10-17T01:30:00Z"), now: now, calendar: amsterdam), "03:30")
    XCTAssertEqual(
      Words.time(try moment("2026-10-16T23:30:00Z"), now: now, calendar: amsterdam), "01:30",
      "01:30 in Amsterdam is the same local day the panel is drawn in")
  }

  /// `quiet 52 min`, `quiet 2 h`, `quiet 1 h 20 min`, `5 h 45 min`.
  func testDurationsAreMinutesAndHours() {
    XCTAssertEqual(Words.duration(52 * 60), "52 min")
    XCTAssertEqual(Words.duration(14 * 60), "14 min")
    XCTAssertEqual(Words.duration(6 * 60), "6 min")
    XCTAssertEqual(Words.duration(30 * 60), "30 min")
    XCTAssertEqual(Words.duration(2 * 3_600), "2 h")
    XCTAssertEqual(Words.duration(3_600 + 20 * 60), "1 h 20 min")
    XCTAssertEqual(Words.duration(5 * 3_600 + 45 * 60), "5 h 45 min")

    // Seconds are a fact nobody asked for: the minute it is inside is the
    // measurement, rounded down, and under one minute that is none of them.
    XCTAssertEqual(Words.duration(59), "0 min")
    XCTAssertEqual(Words.duration(0), "0 min")
    XCTAssertEqual(Words.duration(-120), "0 min", "a clock that went backwards says nothing")
    XCTAssertEqual(Words.duration(3_599), "59 min")
    XCTAssertEqual(Words.duration(3_600), "1 h")
    XCTAssertEqual(
      Words.duration(24 * 3_600), "24 h", "a day is hours, because the panel has no day")
  }

  /// The record's two vias, in the panel's words. `code` is M24b's and has no
  /// word until the task that reads it.
  func testViaIsFromTheTerminalOrFromDesktopChat() {
    XCTAssertEqual(Words.via("terminal"), "from the terminal")
    XCTAssertEqual(Words.via("chat"), "from Desktop chat")
    XCTAssertNil(Words.via(nil))
    XCTAssertNil(Words.via(""))
    XCTAssertNil(Words.via("code"))
  }

  /// A read-back that confirmed the write, and one that did not. A false is
  /// a hollow check and never a failure, so the word says what happened and
  /// not that something is wrong.
  func testVerifiedIsVerifiedOrNotReadBack() {
    XCTAssertEqual(Words.verified(true), "verified")
    XCTAssertEqual(Words.verified(false), "not read back")
    XCTAssertNil(Words.verified(nil))
    XCTAssertFalse(try XCTUnwrap(Words.verified(false)).contains("failed"))
  }

  /// Principle 5: facts and questions, never a prediction. No word this file
  /// can produce says when something finishes, and the file that produces
  /// them carries none either.
  func testNothingPredictsATime() throws {
    // Spelled in pieces so this test is not its own failure.
    let banned = [
      "wi" + "ll", "so" + "on", "esti" + "mate", "remai" + "ning", "e" + "ta",
      "alm" + "ost", "short" + "ly", "expec" + "ted", "any min" + "ute", "le" + "ft",
    ]

    var produced: [String] = []
    for command in Now.stuckCommands {
      let verb = try XCTUnwrap(Words.verb(command))
      produced.append(verb.heading)
      produced.append(verb.past)
    }
    for via in ["terminal", "chat"] { produced.append(try XCTUnwrap(Words.via(via))) }
    for state in [true, false] { produced.append(try XCTUnwrap(Words.verified(state))) }
    for seconds in stride(from: 0, through: 30 * 3_600, by: 37) {
      produced.append(Words.duration(TimeInterval(seconds)))
    }
    let calendar = try utc()
    let now = try moment("2026-10-17T13:10:00Z")
    for days in 0...400 {
      produced.append(
        Words.time(
          now.addingTimeInterval(TimeInterval(-days * 24 * 3_600)), now: now,
          calendar: calendar))
    }

    for word in produced {
      for needle in banned {
        XCTAssertFalse(
          word.lowercased().contains(needle), "\"\(word)\" carries \"\(needle)\"")
      }
    }

    // The file that produces them, read as whole words, so an ordinary word
    // that happens to hold one of these inside it is not a failure.
    let source = try String(
      contentsOf: sourceRoot().appendingPathComponent("PresenceUI/Words.swift"),
      encoding: .utf8)
    for needle in banned + ["esti" + "mated", "expe" + "ct"] {
      XCTAssertFalse(word(needle, isIn: source), "Words.swift carries \"\(needle)\"")
    }
  }

  /// The needle on its own, never inside a longer word.
  private func word(_ needle: String, isIn text: String) -> Bool {
    let lower = text.lowercased()
    var search = lower.startIndex..<lower.endIndex
    while let found = lower.range(of: needle, range: search) {
      let before =
        found.lowerBound == lower.startIndex ? nil : lower[lower.index(before: found.lowerBound)]
      let after = found.upperBound == lower.endIndex ? nil : lower[found.upperBound]
      if !wordish(before) && !wordish(after) { return true }
      search = found.upperBound..<lower.endIndex
    }
    return false
  }

  private func wordish(_ character: Character?) -> Bool {
    guard let character else { return false }
    return character.isLetter || character.isNumber || character == "_"
  }

  private func utc() throws -> Calendar {
    var calendar = Calendar(identifier: .gregorian)
    calendar.timeZone = try XCTUnwrap(TimeZone(identifier: "UTC"))
    calendar.locale = Locale(identifier: "en_US_POSIX")
    return calendar
  }

  private func moment(_ text: String) throws -> Date {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return try XCTUnwrap(formatter.date(from: text), "\(text) is not a moment")
  }

  /// This file is Tests/PresenceUITests/WordsTests.swift, so Sources is three
  /// directories up and one across.
  private func sourceRoot() -> URL {
    URL(fileURLWithPath: #filePath)
      .deletingLastPathComponent()
      .deletingLastPathComponent()
      .deletingLastPathComponent()
      .appendingPathComponent("Sources")
  }
}
