// What the person's own state does to the panel, and what it may not do.
//
// UIState is the three things a person did to the panel that the record knows
// nothing about: a list opened, a link copied, a card dismissed. Each is a
// value outside every view, so each is a test that states the value and reads
// the model back. The three here are the ones a drawn panel would be wrong
// about without them: a Show more that opens nothing, a green check that
// never goes back, and the same request made twice at once.

import Foundation
import PresenceCore
import XCTest

@testable import PresenceUI

@MainActor
final class StateTests: XCTestCase {
  private let chapter4 = SceneDocuments.chapter4

  /// `MORE('Show 3 more')` over TODAY: the row is drawn from the day's own
  /// count, so the list has to read the state the chevron writes. Without it
  /// the word is there, the press is wired, and the panel comes back the
  /// same.
  func testTodaysShowMoreOpensTheList() throws {
    var events: [Event] = []
    for minute in 0..<11 {
      let at = String(format: "2026-10-17T09:%02d:00Z", minute)
      // Each line is its own call: the shared walk keeps one row per
      // run and call, so eleven ends of one call would be one row.
      events.append(
        event(at, .end, command: "reply", doc: chapter4, call: Int64(minute + 1), ok: true))
    }

    let closed = try drawn(events, at: "2026-10-17T10:00:00Z")
    let today = try XCTUnwrap(closed.today)
    XCTAssertEqual(today.rows.count, 11, "the day's rows are all in the model")
    XCTAssertEqual(today.collapsed, 8)
    XCTAssertFalse(today.isOpen, "the list starts closed")

    let opened = try drawn(
      events, at: "2026-10-17T10:00:00Z", ui: UIState(expanded: [.today]))
    XCTAssertTrue(
      try XCTUnwrap(opened.today).isOpen,
      "the chevron the shell wires to .today is the state this list reads")
  }

  /// `.act.is-done` for one second: the state is the document and the moment
  /// together. A done icon holds its own tooltip open, so a check that never
  /// went back would leave "Copied" on the screen and no other icon in the
  /// panel could show its words again.
  func testTheCopiedCheckGoesBackAfterItsSecond() throws {
    let events = [
      event(
        "2026-10-17T09:00:00Z", .end, command: "reply", doc: chapter4, title: "Ch 4", ok: true)
    ]
    let pressed = try at("2026-10-17T09:10:00Z")

    let green = try drawn(
      events, at: "2026-10-17T09:10:00Z",
      ui: UIState(copied: chapter4, copiedAt: pressed))
    XCTAssertEqual(copyState(green), .done, "the second after the press")

    let after = try drawn(
      events, at: "2026-10-17T09:10:01Z",
      ui: UIState(copied: chapter4, copiedAt: pressed))
    XCTAssertEqual(after.documents?.rows.count, 1)
    XCTAssertEqual(copyState(after), .live, "one second on it is the link icon again")

    let forgotten = try drawn(
      events, at: "2026-10-17T09:10:00Z", ui: UIState(copied: chapter4))
    XCTAssertEqual(copyState(forgotten), .live, "a document with no moment is nothing pressed")
  }

  /// A sign-in the app started is the Account row with its countdown. The
  /// last `auth status` still says the token does not work until that child
  /// answers, so the card would ask for the same sign-in a second time, with
  /// a button the running sign-in refuses.
  func testNoSecondSignInCardWhileOneIsInFlight() throws {
    let phase = SignInPhase(url: Scenes.signInURL, secondsLeft: 163)

    let asking = try drawn(
      [], at: "2026-10-17T09:00:00Z", facts: Scenes.machine(auth: .needsSignIn, account: nil))
    XCTAssertEqual(asking.now.count, 1, "a token that stopped working is one card")

    let inFlight = try drawn(
      [], at: "2026-10-17T09:00:00Z",
      facts: Scenes.machine(auth: .needsSignIn, account: nil, signIn: phase))
    XCTAssertTrue(inFlight.now.isEmpty, "the countdown in HEALTH is the whole of the request")
    XCTAssertTrue(
      inFlight.health.contains { $0.kind == .account && $0.waiting },
      "and that row is the one waiting for Google")
  }

  // MARK: - The pieces a test states

  /// The state of Copy link on the one document row.
  private func copyState(_ panel: PanelModel) -> ActionState? {
    panel.documents?.rows.first?.actions.first { $0.kind == .copyLink }?.state
  }

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

  private func at(_ text: String) throws -> Date {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return try XCTUnwrap(formatter.date(from: text), "\(text) is not a moment")
  }

  /// One line of the record, by the wire's own names.
  private func event(
    _ at: String, _ kind: EventKind, command: String? = nil, via: String = "terminal",
    doc: String? = nil, call: Int64 = 1, title: String? = nil, made: String? = nil,
    ok: Bool? = nil, verified: Bool? = nil
  ) -> Event {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return Event(
      schema: 1, at: formatter.date(from: at) ?? Date(timeIntervalSince1970: 0),
      gdoc: "v2.15.0", pid: 7101, run: "3000000000000002", call: call, kind: kind,
      command: command, via: via, doc: doc, wait: false, interrupted: false, step: nil,
      index: nil, count: nil, title: title, made: made, ok: ok, verified: verified, stop: nil,
      line: nil, ms: nil, hold: nil, rule: nil, until: nil)
  }
}
