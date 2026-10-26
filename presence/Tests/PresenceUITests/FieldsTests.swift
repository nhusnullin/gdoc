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
        unknown.reasons, ["The terminal that ran it has the reason."],
        "\(code) names no cause, so the card says where to look and nothing else")
    }
  }

  // The second half of the sentence is a claim, so it is said only where
  // the record carries it. A restyle is a dozen batches, a propose is a
  // suggestion and then its own comment, and an export writes a file for
  // every tab, so a run of any of the three may have written part of what
  // it was going to; internal/restyle refuses the same sentence on gdoc's
  // own side. A request that never got an answer is the same whatever the
  // command, because a deadline may have passed over a request Google had
  // already taken.
  func testAStoppedWriteClaimsTheWorldOnlyWhereItCan() throws {
    func reason(_ command: String, stop: String) throws -> String {
      let panel = try drawn(
        [
          event("2026-10-17T09:03:10Z", .start, command: command, doc: chapter5),
          event(
            "2026-10-17T09:05:00Z", .end, command: command, doc: chapter5, ok: false, stop: stop),
        ], at: "2026-10-17T09:06:00Z")
      return try XCTUnwrap(panel.reasons.first, "a failed \(command) is a stuck card")
    }

    // The three writes that are one request each say it in the document's
    // words, and a publish says it in the note's.
    for command in ["reply", "annotate", "withdraw"] {
      XCTAssertEqual(
        try reason(command, stop: "refused"),
        "Refused by the guard. Nothing was written; the document is unchanged.",
        "a \(command) that never left the machine wrote nothing")
    }
    XCTAssertEqual(
      try reason("publish", stop: "refused"),
      "Refused by the guard. Nothing was created; the note is unchanged.")

    // The three that write more than once say where the reason is instead.
    for command in ["restyle", "propose", "export"] {
      let said = try reason(command, stop: "refused")
      XCTAssertEqual(said, "Refused by the guard. The terminal that ran it has the reason.")
      XCTAssertFalse(said.contains("unchanged"), "a \(command) cannot make that claim")
      XCTAssertTrue(StuckCard.fits(said), "and the card still has the room")
    }

    // And a request with no answer behind it claims nothing at all, not
    // even for the writes that are one request.
    for command in ["reply", "publish", "restyle"] {
      XCTAssertEqual(
        try reason(command, stop: "network"),
        "Network. The terminal that ran it has the reason.",
        "the deadline may have passed over a request Google had taken")
    }
  }

  // The place a person is sent to look is the record's own `via`, and the
  // record writes three of them. A run from Claude Code left nothing in a
  // terminal, so naming one would be a sentence that is not true.
  func testAStoppedWriteNamesThePlaceItRanFrom() throws {
    let places = [
      "terminal": "terminal", "chat": "Desktop chat", "code": "Claude Code session",
    ]
    for (via, place) in places {
      let panel = try drawn(
        [
          event("2026-10-17T09:03:10Z", .start, command: "reply", via: via, doc: chapter5),
          event(
            "2026-10-17T09:05:00Z", .end, command: "reply", via: via, doc: chapter5, ok: false,
            stop: "other"),
        ], at: "2026-10-17T09:06:00Z")
      XCTAssertEqual(
        panel.reasons, ["The \(place) that ran it has the reason."],
        "a write from \(via) is sent to the place it ran in")
      XCTAssertTrue(StuckCard.fits(panel.reasons[0]), "it still fits the card")
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

  /// A write whose process went away said nothing about how it went, so the
  /// card says when it started: that is the one fact the record carries
  /// about it, and NOW keeps such a card for fourteen days.
  ///
  /// A write whose end said it failed is the other way round: the card says
  /// the cause and the state of the world, and never a moment on top of
  /// them. The card is two lines, and a moment there costs the half of the
  /// sentence that says what to do.
  func testACardWithNoCauseCarriesTheMoment() throws {
    let went = try drawn(
      [event("2026-10-17T12:00:00Z", .start, command: "restyle", doc: chapter5)],
      at: "2026-10-17T14:00:00Z")
    XCTAssertEqual(
      went.reasons, ["Started 12:00. The terminal that ran it has the reason."],
      "the moment is the one thing the record says about it")
    XCTAssertTrue(StuckCard.fits(went.reasons[0]), "it still fits the card")

    // An older card says the day and not the clock, which is the words
    // every row in the panel says a moment in: a card inventing a second
    // vocabulary for one thing is the thing this file is against.
    let yesterday = try drawn(
      [event("2026-10-16T12:00:00Z", .start, command: "restyle", doc: chapter5)],
      at: "2026-10-17T14:00:00Z")
    XCTAssertEqual(
      yesterday.reasons, ["Started yesterday. The terminal that ran it has the reason."])

    let older = try drawn(
      [event("2026-10-13T12:00:00Z", .start, command: "restyle", doc: chapter5)],
      at: "2026-10-17T14:00:00Z")
    XCTAssertEqual(
      older.reasons, ["Started Tue. The terminal that ran it has the reason."],
      "the weekday, as the value column of a row says it")

    // An end that failed with a code nobody knows says where to look and no
    // moment: the document's own row carries the moment it stopped at.
    let failed = try drawn(
      [
        event("2026-10-17T12:00:00Z", .start, command: "restyle", doc: chapter5),
        event(
          "2026-10-17T12:01:00Z", .end, command: "restyle", doc: chapter5, ok: false,
          stop: "other"),
      ], at: "2026-10-17T14:00:00Z")
    XCTAssertEqual(failed.reasons, ["The terminal that ran it has the reason."])
    XCTAssertEqual(
      failed.documents?.rows.first?.sub, "restyle failed 12:01 · ~/hub/falcon/chapters/05.md",
      "the document's row is what carries the moment, and the write that failed")
  }

  // Scene r3: `via: code` is gdoc under Claude Code, which is where most of
  // a day's writes come from. It was the one via with no words, and the job
  // card is where a via reaches a person now that the day's log is gone.
  func testViaCodeReadsFromClaudeCode() throws {
    let panel = try drawn(
      [event("2026-10-17T11:37:40Z", .start, command: "annotate", via: "code", doc: chapter5)],
      at: "2026-10-17T11:39:00Z", alive: [7101])

    XCTAssertEqual(Words.via("code"), "from Claude Code")
    guard case .job(_, let card)? = panel.now.first else {
      return XCTFail("a command that is still running is a job card")
    }
    XCTAssertEqual(card.sub, "started 11:37 · from Claude Code")
  }

  // Scene r8 and scene e9: a poll that failed carries its code, so the row
  // says what kind of trouble it is. The one code the row words differently
  // is auth: a watch nobody can sign in for is stopped, because no poll
  // follows a token Google refused until somebody signs in again.
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
      expired.now.compactMap(liveSub), ["15:29 · sign-in expired"],
      "the moment and the cause, with the word in the title above them")
    XCTAssertEqual(
      expired.now.compactMap(liveTitle), ["Live stopped · Chapter 5 storyline"],
      "nothing is polling that document any more, so the row says stopped")
  }

  // Scene e9's card: the number it says is the watches that sign-in stopped,
  // and not the rows NOW is drawing. A token that is only short a scope
  // leaves a watch polling, and the card calling that one stopped would
  // contradict the live row with its Stop icon right under it.
  func testTheSignInCardCountsOnlyTheWatchesItStopped() throws {
    let panel = try drawn(
      [
        event("2026-10-17T15:10:00Z", .start, command: "comments", doc: chapter5, wait: true),
        event(
          "2026-10-17T15:29:00Z", .end, command: "comments", doc: chapter5, wait: true,
          title: "Chapter 5 storyline", ok: false, stop: "auth"),
        event(
          "2026-10-17T15:29:30Z", .start, command: "comments", doc: policy, wait: true,
          call: 2),
      ], at: "2026-10-17T15:30:00Z", facts: Scenes.machine(auth: .missingScopes),
      alive: [7101])

    guard case .stuck(_, let card, _)? = panel.now.first else {
      return XCTFail("a token that is not working raises its card")
    }
    XCTAssertEqual(
      card.reason, "1 live watch stopped. Sign in and start it again.",
      "one watch stopped, and the one still polling is not counted with it")
    XCTAssertEqual(
      panel.now.compactMap(liveTitle),
      ["Live stopped · Chapter 5 storyline", "Live · \(policy)"],
      "the other watch is live on the screen, so the card may not call it stopped")
  }

  // Whether the token works is a fact about the computer, and a watch's
  // sub-line is a fact about that watch. A token short a scope stops the
  // watch whose poll it was refused on and leaves the other polling, so the
  // one still live keeps since, quiet and its two counts: writing the
  // machine's word over every row would take four facts off a watch that is
  // live on the screen with its Stop icon beside it.
  func testAnOpenWatchKeepsItsFactsWhenAnotherOneLostTheSignIn() throws {
    let panel = try drawn(
      [
        event("2026-10-17T15:10:00Z", .start, command: "comments", doc: chapter5, wait: true),
        event(
          "2026-10-17T15:29:00Z", .end, command: "comments", doc: chapter5, wait: true,
          title: "Chapter 5 storyline", ok: false, stop: "auth"),
        event(
          "2026-10-17T15:20:00Z", .start, command: "comments", doc: policy, wait: true, call: 2),
        event(
          "2026-10-17T15:25:00Z", .end, command: "reply", doc: policy, ok: true, verified: true,
          call: 3),
      ], at: "2026-10-17T15:30:00Z", facts: Scenes.machine(auth: .missingScopes),
      alive: [7101])

    XCTAssertEqual(
      panel.now.compactMap(liveSub),
      ["15:29 · sign-in expired", "since 15:20 · quiet 5 min · replied 1"],
      "the stopped watch says why it stopped, and the open one still says what it has done")
  }

  // The hold the sign-in changed under: the card goes, and nothing is left
  // on the screen. The hold was the whole of what that document had: a
  // document the chat only held a write for has no action on it, so no row
  // in DOCUMENTS either.
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
    XCTAssertEqual(
      panel.documents?.rows.map(\.doc), [], "a hold that ended is no action on the document")
  }

  // The card of a hold names the write by the command's heading, the way
  // every other word in the panel names it: the record writes `propose` and
  // a person reads "Proposal". A raw command name on the card and a heading
  // in the list is two vocabularies for one thing.
  func testAHeldWriteIsNamedByItsHeading() throws {
    for (command, verb) in Words.verbs {
      let panel = try drawn(
        [
          event(
            "2026-10-17T16:19:01Z", .end, command: "read", doc: policy,
            title: "Altery Group Channel Management Policy", ok: true),
          event(
            "2026-10-17T16:20:00Z", .held, command: command, via: "chat", doc: policy,
            hold: "h_1", rule: "Unread target"),
        ], at: "2026-10-17T16:34:00Z")

      guard case .hold(_, let card, _)? = panel.now.first else {
        return XCTFail("a held \(command) is one card in NOW")
      }
      XCTAssertEqual(
        card.sub,
        "\(verb.heading) · Altery Group Channel Management Policy · rule Unread target",
        "the card says \(verb.heading) and never \(command)")
    }
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
    hold: String? = nil, rule: String? = nil, run: String = "3000000000000001",
    call: Int64? = nil
  ) -> Event {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return Event(
      schema: 1, at: formatter.date(from: at) ?? Date(timeIntervalSince1970: 0),
      gdoc: "v2.15.0", pid: 7101, run: run, call: call ?? (hold == nil ? 1 : nil),
      kind: kind, command: command, via: via, doc: doc, wait: wait, interrupted: interrupted,
      step: step, index: index, count: count, title: title, made: made, ok: ok,
      verified: verified, stop: stop, line: line, ms: nil, hold: hold, rule: rule, until: nil)
  }

  /// The sub-line of a live row, where an item is one.
  private func liveSub(_ item: NowItem) -> String? {
    if case .live(_, let row, _, _) = item { return row.sub }
    return nil
  }

  private func liveTitle(_ item: NowItem) -> String? {
    if case .live(_, let row, _, _) = item { return row.title }
    return nil
  }
}
