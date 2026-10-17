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

  /// `latest_stable` is a fact and not a judgement: gdoc prints the newest
  /// stable release whatever is installed. A nightly is an ordinary tag whose
  /// patch is not zero, and this repository cuts one after every merge, so a
  /// machine ahead of the stable line must not be offered a release behind
  /// it: the row would claim an update and the button would do nothing.
  func testAnOlderPublishedReleaseIsNoUpdate() throws {
    let cases: [(installed: String, published: String, offered: String?)] = [
      ("v2.14.0", "v2.15.0", "v2.15.0"),
      ("v2.14.0", "v2.14.1", "v2.14.1"),
      ("v2.14.0", "v2.14.0", nil),
      ("v2.14.1", "v2.14.0", nil),
      ("v2.15.0", "v2.14.9", nil),
      ("v2.9.0", "v2.10.0", "v2.10.0"),
      // Neither is a tag this app can read, so the most it can say is that
      // the two strings differ.
      ("dev", "v2.15.0", "v2.15.0"),
      ("dev", "dev", nil),
    ]
    for (installed, published, offered) in cases {
      let versions = Versions(app: "2.14.0", gdoc: installed, latestStable: published)
      XCTAssertEqual(
        versions.updateAvailable, offered,
        "installed \(installed) against published \(published)")
    }

    XCTAssertNil(
      Versions(app: "2.14.0", gdoc: nil, latestStable: "v2.15.0").updateAvailable,
      "a binary nobody found is no update")
    XCTAssertNil(
      Versions(app: "2.14.0", gdoc: "v2.14.0").updateAvailable,
      "and a check that heard nothing offers nothing")

    // The row itself, so the rule reaches the screen it is about.
    let behind = try drawn(
      [], at: "2026-10-17T09:00:00Z", facts: Scenes.machine(latestStable: "v2.15.0"))
    XCTAssertEqual(
      behind.health.first { $0.kind == .gdoc }?.value, "v2.14.0 · v2.15.0 available")
    let ahead = try drawn(
      [], at: "2026-10-17T09:00:00Z", facts: Scenes.machine(latestStable: "v2.13.0"))
    XCTAssertEqual(ahead.health.first { $0.kind == .gdoc }?.value, "v2.14.0")
    XCTAssertTrue(
      ahead.health.first { $0.kind == .gdoc }?.actions.isEmpty ?? false,
      "and there is no button to press")
  }

  /// While an update child of the app's own is up, neither row offers a
  /// button. Both argv lists are `gdoc update` and `update --desktop`
  /// installs the release it writes the extension of, so a second press is a
  /// second process staging and renaming the binary the first one is
  /// replacing: the rollback copy can end up being the new binary.
  ///
  /// The rows stay where they are. A row that left while a child was up
  /// would be the screen moving under the person who pressed it.
  func testNeitherUpdateIsOfferedWhileOneIsRunning() throws {
    let at = "2026-10-17T09:00:00Z"
    let idle = try drawn([], at: at, facts: updating(false))
    XCTAssertEqual(
      idle.health.first { $0.kind == .gdoc }?.actions.map(\.kind), [.update],
      "with nothing running the update is there to take")
    XCTAssertEqual(
      idle.health.first { $0.kind == .desktop }?.actions.map(\.kind), [.installExtension])

    let running = try drawn([], at: at, facts: updating(true))
    let gdoc = try XCTUnwrap(running.health.first { $0.kind == .gdoc })
    XCTAssertTrue(gdoc.waiting, "the gdoc row is a ring while the child is up")
    XCTAssertEqual(gdoc.value, "v2.14.0", "and the version is the one still installed")
    XCTAssertTrue(gdoc.actions.isEmpty)
    let desktop = try XCTUnwrap(running.health.first { $0.kind == .desktop })
    XCTAssertEqual(desktop.value, "not installed", "the row is where it was")
    XCTAssertTrue(desktop.actions.isEmpty, "and there is nothing on it to press")
  }

  /// The one fact the panel may not invent is whose account this is. `auth`
  /// is nothing until `gdoc auth status` has answered, which is every
  /// panel's first draw and the whole of a machine with no gdoc on it: the
  /// row is a ring over no value, and it offers no Sign out.
  func testTheAccountRowClaimsNothingBeforeTheFirstAnswer() throws {
    let launching = Facts(versions: Versions(app: "2.15.0"), gdocPath: Scenes.gdocPath)
    XCTAssertNil(launching.auth, "nobody has asked")

    let first = try accountRow(of: try drawn([], at: "2026-10-17T10:00:00Z", facts: launching))
    XCTAssertTrue(first.waiting, "the ring says the question is out")
    XCTAssertEqual(first.value, "", "and no sign-in is claimed")
    XCTAssertEqual(first.glyph, .blank, "no green check over a token nobody read")
    XCTAssertTrue(first.actions.isEmpty, "there is nothing to sign out of yet")

    let answered = try accountRow(
      of: try drawn([], at: "2026-10-17T10:00:00Z", facts: Scenes.machine()))
    XCTAssertFalse(answered.waiting)
    XCTAssertEqual(answered.glyph, .check)
    XCTAssertEqual(answered.value, Scenes.account)
    XCTAssertEqual(answered.actions.map(\.kind), [.signOut], "now there is")
  }

  /// A sign-out drops every hold the chat is keeping, so the one destructive
  /// confirm in the panel counts what the record has open and not the cards
  /// on the screen: a dismissed card is a hold all the same, and a hold
  /// whose life ran out is waiting for nobody.
  func testTheSignOutCountsTheHoldsTheRecordHasOpen() throws {
    let asking = UIState(question: .signOut, questionAt: try at("2026-10-17T10:00:00Z"))
    let hidden = try drawn(
      [held("2026-10-17T09:50:00Z", hold: "h_1", until: "2026-10-17T10:50:00Z")],
      at: "2026-10-17T10:00:00Z",
      ui: UIState(
        dismissed: ["h_1"], question: .signOut,
        questionAt: try at("2026-10-17T10:00:00Z")))
    XCTAssertTrue(hidden.now.isEmpty, "the card was dismissed, so NOW draws nothing")
    XCTAssertEqual(
      try question(of: hidden).consequence, "1 hold will stop.",
      "the hold is still waiting in the chat, and the sign-out drops it")

    let ranOut = try drawn(
      [held("2026-10-17T08:00:00Z", hold: "h_2", until: "2026-10-17T09:00:00Z")],
      at: "2026-10-17T10:00:00Z", ui: asking)
    XCTAssertEqual(ranOut.now.count, 1, "the card is on the screen")
    XCTAssertEqual(
      try question(of: ranOut).consequence, "Nothing is running. Only the token goes.",
      "a hold whose life ran out is waiting for nobody, so the sign-out stops nothing")
  }

  /// An id Drive could not have written is a line gdoc did not write. Both
  /// icons that would build a URL out of it say so before the press, because
  /// `Opener` refuses to build one and a live icon that does nothing says
  /// nothing.
  func testAnIdDriveCouldNotHaveWrittenGreysItsTwoIcons() throws {
    let planted = "not-a-drive-id"
    let panel = try drawn(
      [event("2026-10-17T09:00:00Z", .end, command: "reply", doc: planted, ok: true)],
      at: "2026-10-17T09:10:00Z")

    let row = try XCTUnwrap(panel.documents?.rows.first, "the row is drawn, as the record says")
    XCTAssertEqual(row.doc, planted)
    XCTAssertEqual(
      row.actions.first { $0.kind == .openDoc }?.state, .off(Tips.noDoc),
      "there is no document behind it to open")
    XCTAssertEqual(
      row.actions.first { $0.kind == .copyLink }?.state, .off(Tips.noLink),
      "and no link to copy")

    let real = try drawn(
      [event("2026-10-17T09:00:00Z", .end, command: "reply", doc: chapter4, ok: true)],
      at: "2026-10-17T09:10:00Z")
    XCTAssertEqual(
      real.documents?.rows.first?.actions.first { $0.kind == .openDoc }?.state, .live,
      "an id of Drive's own shape is live")
  }

  /// "Account changed" is a sentence about two addresses, so it is written
  /// only where there are two. The memory of the one before is kept for the
  /// life of the app, and a sign-out leaves no address at all: the line
  /// would end at the arrow with nothing after it.
  func testASignOutLeavesNoHalfSentenceAboutTheAccount() throws {
    let before = "first@example.com"
    let signedIn = Facts(
      account: "second@example.com", auth: .ok, previousAccount: before,
      versions: Versions(app: "2.15.0"), gdocPath: Scenes.gdocPath)
    XCTAssertEqual(
      try accountRow(of: try drawn([], at: "2026-10-17T10:00:00Z", facts: signedIn)).quiet,
      "Account changed: first@example.com → second@example.com")

    let signedOut = Facts(
      auth: .signedOut, previousAccount: before, versions: Versions(app: "2.15.0"),
      gdocPath: Scenes.gdocPath)
    XCTAssertNil(
      try accountRow(of: try drawn([], at: "2026-10-17T10:00:00Z", facts: signedOut)).quiet,
      "there is no second address, so there is no sentence")
  }

  // MARK: - The pieces a test states

  private func accountRow(of panel: PanelModel) throws -> HealthRowModel {
    try XCTUnwrap(panel.health.first { $0.kind == .account }, "HEALTH opens with the account")
  }

  private func question(of panel: PanelModel) throws -> QuestionModel {
    try XCTUnwrap(try accountRow(of: panel).question, "the row is the question for three seconds")
  }

  /// A write a rule held, as gdoc writes it: one line, and no call.
  private func held(_ at: String, hold: String, until: String) -> Event {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return Event(
      schema: 1, at: formatter.date(from: at) ?? Date(timeIntervalSince1970: 0),
      gdoc: "v2.15.0", pid: 7101, run: "3000000000000002", call: nil, kind: .held,
      command: "reply", via: "chat", doc: chapter4, wait: false, title: nil, ok: nil,
      verified: nil, ms: nil, hold: hold, rule: "dictated",
      until: formatter.date(from: until))
  }

  /// A machine with an update to take and no extension written, with and
  /// without a child of the app's own up.
  private func updating(_ running: Bool) -> Facts {
    Facts(
      account: Scenes.account,
      versions: Versions(app: "2.14.0", gdoc: "v2.14.0", latestStable: "v2.15.0"),
      gdocPath: Scenes.gdocPath,
      hub: Scenes.hub,
      extensionState: .notInstalled,
      updating: running)
  }

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
