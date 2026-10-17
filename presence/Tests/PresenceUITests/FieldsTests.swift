// What the panel says now that M24b's fields are in the record.
//
// Five scenes of the screens file drew a fallback through M24a, because the
// v1 record did not carry the fact the screen was about: a publish with no
// title and no steps, a stuck card with no cause, a failed poll with no
// reason, a watch that stopped with nobody saying who stopped it, and a run
// from Claude Code that read as a run from the terminal. Each of those is
// one field, and each test here is the field reaching the words.
//
// The events are values rather than a fixture: four lines a test can read
// beside its assertion say more than a file it has to open. The scenes
// themselves are the fixtures, and RenderTests draws them.

import Foundation
import PresenceCore
import XCTest

@testable import PresenceUI

@MainActor
final class FieldsTests: XCTestCase {
  private let chapter4 = "1Ch4BbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQq"
  private let chapter5 = SceneDocuments.chapter5
  private let policy = SceneDocuments.policy

  // Scene r2: `stop` is one of gdoc's seven codes and `line` is where the
  // cause sits, so the card says the cause with its place first and the
  // state of the world second, which is the screens file's own rule and its
  // own sentence.
  func testTheStuckReasonIsTheCodesWordsAndTheLine() throws {
    let panel = try drawn(
      [
        event("2026-10-17T09:03:10Z", .start, command: "publish"),
        event(
          "2026-10-17T09:05:00Z", .end, command: "publish", ok: false, stop: "svg_picture",
          line: 41),
      ], at: "2026-10-17T09:06:00Z")

    guard case .stuck(_, let card, _)? = panel.now.first else {
      return XCTFail("a failed publish is a stuck card")
    }
    XCTAssertEqual(
      card.reason, "SVG picture at line 41. Nothing was created; the note is unchanged.")
    XCTAssertTrue(StuckCard.fits(card.reason), "the two lines the card has room for")

    XCTAssertEqual(
      panel.today?.rows.first?.sub, "from the terminal · SVG picture at line 41",
      "the log row carries the cause as well, and the red glyph says it stopped")

    // A code this version has not heard of, and `other`, which is gdoc
    // saying nothing matched: the panel says where the reason is and never
    // guesses at one.
    for code in ["other", "a_word_from_a_newer_gdoc"] {
      let unknown = try drawn(
        [
          event("2026-10-17T09:03:10Z", .start, command: "reply", doc: chapter5),
          event(
            "2026-10-17T09:05:00Z", .end, command: "reply", doc: chapter5, ok: false, stop: code),
        ], at: "2026-10-17T09:06:00Z")
      XCTAssertEqual(
        unknown.reasons, ["Stopped at 09:05. The terminal that ran it has the reason."],
        "\(code) names no cause, so the card says where to look")
    }
  }

  // Scene r1: the steps are the record's own four, the one that was written
  // last is the one running, and the ring is that step's place in the list.
  // Nothing here is a percentage of work done: gdoc writes none and the
  // panel invents none.
  func testStepEventsDriveTheJobCardAndTheBarRing() throws {
    let events = [
      event("2026-10-17T09:03:10Z", .start, command: "publish"),
      event("2026-10-17T09:03:12Z", .step, command: "publish", step: "build", index: 1, count: 4),
      event(
        "2026-10-17T09:03:50Z", .step, command: "publish", step: "upload", index: 2, count: 4,
        title: "Chapter 4"),
    ]
    let panel = try drawn(events, at: "2026-10-17T09:04:00Z", alive: [7101])

    guard case .job(_, let card)? = panel.now.first else {
      return XCTFail("a publish that is running is a job card")
    }
    XCTAssertEqual(card.title, "Publishing · Chapter 4", "the title the first step carried")
    XCTAssertEqual(card.sub, "step 2 of 4 · started 09:03 · from the terminal")
    XCTAssertEqual(card.progress, 0.5, "two of four, which is a place and not a forecast")
    XCTAssertEqual(
      card.steps.map(\.title),
      ["Built the docx", "Uploading", "Read it back", "Link into the note"])
    XCTAssertEqual(
      card.steps.map(\.state), [.done, .running(nil), .waiting, .waiting],
      "a step line is written before its part begins, so the last one is the one running")
    XCTAssertTrue(card.steps.allSatisfy { $0.value == nil }, "the record carries no step counts")

    let fold = Now.fold(events, now: try at("2026-10-17T09:04:00Z"), alive: { $0 == 7101 })
    XCTAssertEqual(BarImage.ringProgress(now: fold), 0.5, "the badge turns with the job")
    XCTAssertEqual(
      BarImage.ringFraction(for: nil), RingView.indeterminateFraction,
      "a job whose steps are not known is the indeterminate ring")
  }

  // Scene r3: `via: code` is gdoc under Claude Code, which is where most of
  // a day's writes come from. It was the one via with no words.
  func testViaCodeReadsFromClaudeCode() throws {
    let panel = try drawn(
      [
        event("2026-10-17T11:37:40Z", .start, command: "annotate", via: "code", doc: chapter5),
        event(
          "2026-10-17T11:38:00Z", .end, command: "annotate", via: "code", doc: chapter5,
          title: "Ch 5. Commercial Model: review draft v0.1", ok: true, verified: true),
      ], at: "2026-10-17T11:39:00Z")

    XCTAssertEqual(Words.via("code"), "from Claude Code")
    XCTAssertEqual(panel.today?.rows.first?.sub, "from Claude Code · verified")
  }

  // Scene r9: a wait somebody stopped carries `interrupted`, so the log says
  // who ended the watch. Without it the row read as a watch that stopped by
  // itself, which is what a lost process looks like.
  func testAnInterruptedWaitIsLiveStoppedByYou() throws {
    let stopped = try drawn(
      [
        event("2026-10-17T18:10:50Z", .start, command: "comments", doc: chapter5, wait: true),
        event(
          "2026-10-17T18:40:40Z", .end, command: "comments", doc: chapter5, wait: true,
          interrupted: true, title: "Ch 5. Commercial Model: review draft v0.1", ok: true),
      ], at: "2026-10-17T18:59:00Z")

    let row = try XCTUnwrap(stopped.today?.rows.first)
    XCTAssertEqual(row.title, "Live stopped by you · Ch 5. Commercial Model: review draft v0.1")

    let lost = try drawn(
      [
        event("2026-10-17T18:10:50Z", .start, command: "comments", doc: chapter5, wait: true),
        event(
          "2026-10-17T18:40:40Z", .end, command: "comments", doc: chapter5, wait: true,
          title: "Ch 5. Commercial Model: review draft v0.1", ok: true),
      ], at: "2026-10-17T18:59:00Z")
    XCTAssertEqual(
      lost.today?.rows.first?.title, "Live stopped · Ch 5. Commercial Model: review draft v0.1",
      "nobody said they stopped it, so the panel does not say so either")
  }

  // Scene r8 and scene e9: a poll that failed carries its code, so the row
  // says what kind of trouble it is. The one code the row words differently
  // is auth: a watch nobody can sign in for is paused and not broken.
  func testAFailedPollCarriesItsCode() throws {
    let network = try drawn(
      [
        event("2026-10-17T16:34:00Z", .start, command: "comments", doc: chapter5, wait: true),
        event(
          "2026-10-17T17:04:00Z", .end, command: "comments", doc: chapter5, wait: true,
          title: "Chapter 5 storyline", ok: false, stop: "network"),
      ], at: "2026-10-17T17:05:00Z")
    guard case .live(_, let row, _, _)? = network.now.first else {
      return XCTFail("an open watch is a live row")
    }
    XCTAssertEqual(row.sub, "last poll failed 17:04 · network")

    let expired = try drawn(
      [
        event("2026-10-17T15:10:00Z", .start, command: "comments", doc: chapter5, wait: true),
        event(
          "2026-10-17T15:29:00Z", .end, command: "comments", doc: chapter5, wait: true,
          title: "Chapter 5 storyline", ok: false, stop: "auth"),
      ], at: "2026-10-17T15:30:00Z", facts: Scenes.machine(auth: .needsSignIn))
    XCTAssertEqual(
      expired.now.compactMap(liveSub), ["paused 15:29 · sign-in expired"],
      "the watch is not broken, it is waiting for a sign-in")
  }

  // The hold the sign-in changed under: the card goes, and the one line in
  // TODAY says what happened to the text. It is the record's own word and
  // no reading on top of it.
  func testADroppedHoldEndsTheCard() throws {
    let panel = try drawn(
      [
        event(
          "2026-10-17T16:19:01Z", .end, command: "read", doc: policy,
          title: "Altery Group Channel Management Policy", ok: true),
        event(
          "2026-10-17T16:20:00Z", .held, command: "reply", via: "chat", doc: policy,
          hold: "h_1", rule: "Dictated"),
        event(
          "2026-10-17T16:41:00Z", .dropped, command: "reply", via: "chat", doc: policy,
          hold: "h_1", rule: "Dictated"),
      ], at: "2026-10-17T17:00:00Z")

    XCTAssertTrue(
      panel.now.isEmpty, "nothing is waiting for anybody, so NOW is not drawn at all")
    let closed = try XCTUnwrap(panel.today?.rows.first)
    XCTAssertEqual(closed.title, "Hold closed · Altery Group Channel Management Policy")
    XCTAssertEqual(closed.sub, "the sign-in changed, nothing was sent")
  }

  // MARK: the record, as values

  /// The panel one stretch of the record draws.
  private func drawn(
    _ events: [Event], at text: String, facts: Facts = Scenes.machine(),
    ui: UIState = UIState(), alive: Set<Int32> = []
  ) throws -> PanelModel {
    var calendar = Calendar(identifier: .gregorian)
    calendar.timeZone = TimeZone(secondsFromGMT: 0) ?? .gmt
    calendar.locale = Locale(identifier: "en_US_POSIX")
    return PanelModel.build(
      events: events, facts: facts, ui: ui, now: try at(text), calendar: calendar,
      alive: { alive.contains($0) })
  }

  /// A moment the way the record writes one.
  private func at(_ text: String) throws -> Date {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return try XCTUnwrap(formatter.date(from: text), "\(text) is not a moment")
  }

  /// One line of the record. Every field is the wire's own name, so a test
  /// reads the way the file reads and no builder decides anything.
  private func event(
    _ at: String, _ kind: EventKind, command: String? = nil, via: String = "terminal",
    doc: String? = nil, wait: Bool = false, interrupted: Bool = false, step: String? = nil,
    index: Int? = nil, count: Int? = nil, title: String? = nil, made: String? = nil,
    ok: Bool? = nil, verified: Bool? = nil, stop: String? = nil, line: Int? = nil,
    hold: String? = nil, rule: String? = nil
  ) -> Event {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return Event(
      schema: 1, at: formatter.date(from: at) ?? Date(timeIntervalSince1970: 0),
      gdoc: "v2.15.0", pid: 7101, run: "3000000000000001", call: hold == nil ? 1 : nil,
      kind: kind, command: command, via: via, doc: doc, wait: wait, interrupted: interrupted,
      step: step, index: index, count: count, title: title, made: made, ok: ok,
      verified: verified, stop: stop, line: line, ms: nil, hold: hold, rule: rule, until: nil)
  }

  /// The sub-line of a live row, where an item is one.
  private func liveSub(_ item: NowItem) -> String? {
    if case .live(_, let row, _, _) = item { return row.sub }
    return nil
  }
}
