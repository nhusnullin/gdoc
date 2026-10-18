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

  /// A write whose process went away has no row in TODAY while its card is
  /// up, because `Days.every` leaves out what NOW is carding, and NOW keeps
  /// such a card for fourteen days. So the card is the one thing on the
  /// screen that can say when it started, and it does.
  ///
  /// A write whose end said it failed is the other way round: the row in
  /// TODAY carries the moment, so the card says the cause and the state of
  /// the world and nothing a second time.
  func testACardWithNoRowBehindItCarriesTheMoment() throws {
    let went = try drawn(
      [event("2026-10-17T12:00:00Z", .start, command: "restyle", doc: chapter5)],
      at: "2026-10-17T14:00:00Z")
    XCTAssertEqual(
      went.reasons, ["Started 12:00. The terminal that ran it has the reason."],
      "nothing else on the screen says when it was")
    XCTAssertTrue(went.today?.rows.isEmpty ?? false, "and TODAY draws no row under the card")
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

    // An end that failed with a code nobody knows has its row in TODAY, so
    // the card says where to look and no moment.
    let failed = try drawn(
      [
        event("2026-10-17T12:00:00Z", .start, command: "restyle", doc: chapter5),
        event(
          "2026-10-17T12:01:00Z", .end, command: "restyle", doc: chapter5, ok: false,
          stop: "other"),
      ], at: "2026-10-17T14:00:00Z")
    XCTAssertEqual(failed.reasons, ["The terminal that ran it has the reason."])
    XCTAssertEqual(failed.today?.rows.count, 1, "the row is what carries the moment")
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

  // The two endings a write happened under, each the line gdoc actually
  // writes for it. `mcprelease.go` writes `released` when the person presses
  // the card, before the send, so the approval is the row of the write it
  // sent; `mcphold.go` writes `landed` for a held write the model sent again
  // with no card, so that one is the write's own row and a second row saying
  // the card went with it. Neither of them is a person saying no: nothing in
  // gdoc writes a refusal, and an unanswered card expires.
  func testTheApprovalAndTheLandingAreTheLinesGdocWrites() throws {
    let read = event(
      "2026-10-17T16:19:01Z", .end, command: "read", doc: policy,
      title: "Altery Group Channel Management Policy", ok: true)
    let held = event(
      "2026-10-17T16:20:00Z", .held, command: "reply", via: "chat", doc: policy,
      hold: "h_1", rule: "Dictated")
    // The chat is one run and a dozen calls, so the reply is the second
    // call of the process the read was the first call of.
    let wrote = [
      event(
        "2026-10-17T16:40:55Z", .start, command: "reply", via: "chat", doc: policy, call: 2),
      event(
        "2026-10-17T16:41:00Z", .end, command: "reply", via: "chat", doc: policy, ok: true,
        verified: true, call: 2),
    ]

    let approved = try drawn(
      [
        read, held,
        event(
          "2026-10-17T16:40:55Z", .released, command: "reply", via: "chat", doc: policy,
          hold: "h_1"),
      ] + wrote, at: "2026-10-17T17:00:00Z")
    let rows = try XCTUnwrap(approved.today).rows
    XCTAssertEqual(
      rows.first?.title, "Reply · Altery Group Channel Management Policy",
      "one row for the write the approval sent")
    XCTAssertEqual(rows.first?.sub, "approved by you in chat · verified")
    XCTAssertFalse(
      rows.contains { $0.title.hasPrefix("Hold closed") }, "the write carries the ending")

    let landed = try drawn(
      [read, held] + wrote + [
        event(
          "2026-10-17T16:41:01Z", .landed, command: "reply", via: "chat", doc: policy,
          hold: "h_1")
      ], at: "2026-10-17T17:00:00Z")
    let after = try XCTUnwrap(landed.today).rows
    XCTAssertEqual(after.first?.title, "Hold closed · Altery Group Channel Management Policy")
    XCTAssertEqual(
      after.first?.sub, "the write went out without the card",
      "nobody approved anything, so the row says what did happen")
    XCTAssertEqual(
      after.dropFirst().first?.sub, "from Desktop chat · verified",
      "and the write is the ordinary row of a write")
  }

  // The TODAY row of a hold names the write by the command's heading, the
  // way every other row in the panel names it: the record writes `propose`
  // and a person reads "Proposal". A raw command name in one row and a
  // heading in the next is two vocabularies for one thing.
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

      let raised = try XCTUnwrap(
        panel.today?.rows.first { $0.title.hasPrefix("Hold ·") },
        "a held \(command) is one row in TODAY")
      XCTAssertEqual(
        raised.sub, "\(verb.heading) held by rule Unread target · waiting in Desktop chat",
        "the row says \(verb.heading) and never \(command)")
    }
  }

  // One rule for the DOCUMENTS header, and one number in it: the day's own
  // count where every row in the list is from today, and "14 days" with the
  // whole list's count where one row is older. Scene e2 is the mixed one,
  // and it reads "14 days · 4".
  func testTheDocumentsHeaderCountsTheWholeList() throws {
    var runs = 0
    func wrote(_ at: String, _ doc: String) -> Event {
      runs += 1
      return event(
        at, .end, command: "reply", doc: doc, title: "A note", ok: true, verified: true,
        run: "300000000000000\(runs)")
    }

    let bare = try drawn([], at: "2026-10-17T18:00:00Z")
    XCTAssertNil(bare.documents?.count, "an empty list has no count at all")

    let day = try drawn(
      [
        wrote("2026-10-17T09:00:00Z", chapter4),
        wrote("2026-10-17T10:00:00Z", chapter5),
        wrote("2026-10-17T11:00:00Z", policy),
      ], at: "2026-10-17T18:00:00Z")
    XCTAssertEqual(day.documents?.count, "3 today", "every row is from today")

    let older = try drawn(
      [
        wrote("2026-10-16T09:00:00Z", SceneDocuments.northwind),
        wrote("2026-10-17T09:00:00Z", chapter4),
        wrote("2026-10-17T10:00:00Z", chapter5),
        wrote("2026-10-17T11:00:00Z", policy),
      ], at: "2026-10-17T18:00:00Z")
    XCTAssertEqual(
      older.documents?.count, "14 days · 4",
      "one row is from yesterday, so the header is the stretch and the whole count")

    let long = try drawn(
      [
        wrote("2026-10-17T09:00:00Z", chapter4),
        wrote("2026-10-17T10:00:00Z", chapter5),
        wrote("2026-10-17T11:00:00Z", policy),
        wrote("2026-10-17T12:00:00Z", SceneDocuments.northwind),
        wrote("2026-10-17T13:00:00Z", SceneDocuments.partner),
        wrote("2026-10-17T14:00:00Z", SceneDocuments.concept),
      ], at: "2026-10-17T18:00:00Z")
    XCTAssertEqual(long.documents?.rows.count, 6)
    XCTAssertEqual(
      long.documents?.count, "6 today",
      "six rows, and the header counts all of them")
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
