// What the person's own state does to the panel, and what it may not do.
//
// UIState is the three things a person did to the panel that the record knows
// nothing about: a list opened, a link copied, a card dismissed. Each is a
// value outside every view, so each is a test that states the value and reads
// the model back. The three here are the ones a drawn panel would be wrong
// about without them: a green check that never goes back, a card nobody
// dismissed, and the same request made twice at once.

import Foundation
import PresenceCore
import XCTest

@testable import PresenceUI

@MainActor
final class StateTests: XCTestCase {
  private let chapter4 = SceneDocuments.chapter4

  /// The whole day is in the model, and the whole day is drawn.
  ///
  /// TODAY was eight rows with the rest behind a Show more until
  /// 2026-10-18, and the word was the one piece of panel state a person's
  /// press changed. Nothing is folded now: the fold hands over the day and
  /// the list draws it.
  func testTheWholeDayIsInTheModel() throws {
    var events: [Event] = []
    for minute in 0..<11 {
      let at = String(format: "2026-10-17T09:%02d:00Z", minute)
      // Each line is its own call: the shared walk keeps one row per
      // run and call, so eleven ends of one call would be one row.
      events.append(
        event(at, .end, command: "reply", doc: chapter4, call: Int64(minute + 1), ok: true))
    }

    let drawn = try drawn(events, at: "2026-10-17T10:00:00Z")
    let today = try XCTUnwrap(drawn.today)
    XCTAssertEqual(today.rows.count, 11, "the day's rows are all in the model")
    XCTAssertNil(today.empty, "a day with eleven rows says nothing about being empty")
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
    XCTAssertTrue(
      inFlight.now.isEmpty, "the countdown in the header is the whole of the request")
    XCTAssertTrue(inFlight.header.waiting, "and the header is the thing waiting for Google")
    XCTAssertEqual(inFlight.header.account, "Waiting for Google")
    XCTAssertEqual(inFlight.header.signInURL, Scenes.signInURL)
    XCTAssertEqual(
      inFlight.header.quiet, ["2:43 left of the three minutes gdoc waits"],
      "the child's own three minutes, as a clock")
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

    // The header itself, so the rule reaches the screen it is about. The
    // version says the version and the icon says there is a newer one: the
    // row said both in one string, and the header has no room for it.
    let behind = try drawn(
      [], at: "2026-10-17T09:00:00Z", facts: Scenes.machine(latestStable: "v2.15.0"))
    XCTAssertEqual(behind.header.version, "v2.14.0")
    XCTAssertEqual(behind.header.update?.kind, .update)
    XCTAssertEqual(behind.header.update?.tip, "Update to v2.15.0")
    let ahead = try drawn(
      [], at: "2026-10-17T09:00:00Z", facts: Scenes.machine(latestStable: "v2.13.0"))
    XCTAssertEqual(ahead.header.version, "v2.14.0")
    XCTAssertNil(ahead.header.update, "and there is no button to press")
  }

  /// The Hub row, which is the one row in the panel that opens a window of
  /// the system's: the folder the person chose, written with a tilde where
  /// it is inside their home, or the word that offers to choose one.
  ///
  /// The md icon says the same thing from the other side. Without a hub
  /// there is no pairing, and a greyed icon that said "No note is paired
  /// with this document" sent the person looking for a note, which is where
  /// Nail went on the second morning: the tooltip names the row that fixes
  /// it instead.
  func testTheHubRowIsTheFolderOrTheWayToChooseOne() throws {
    let chosen = try drawn([], at: "2026-10-17T10:00:00Z")
    let row = try XCTUnwrap(chosen.settings.first { $0.kind == .hub }, "SETTINGS has no Hub row")
    XCTAssertEqual(row.kind.label, "Hub")
    XCTAssertEqual(row.value, "~/hub", "the folder, with the home directory as a tilde")
    XCTAssertEqual(row.select, .chooseHub, "and pressing the row is what opens the panel")
    XCTAssertNil(row.isOn, "it is not a switch")
    XCTAssertTrue(row.actions.isEmpty, "and it carries no icon")

    let none = try drawn(
      [], at: "2026-10-17T10:00:00Z", facts: Scenes.machine(hub: nil))
    let offer = try XCTUnwrap(none.settings.first { $0.kind == .hub })
    XCTAssertEqual(offer.value, "Choose…")
    XCTAssertEqual(offer.select, .chooseHub, "the same press either way")

    // A hub outside the home directory is written in full, because a tilde
    // there would name a folder that is not the one chosen.
    XCTAssertEqual(
      Sentences.folder("/Volumes/work/hub", home: "/Users/nail"), "/Volumes/work/hub")
    XCTAssertEqual(Sentences.folder("/Users/nail", home: "/Users/nail"), "~")
    XCTAssertEqual(
      Sentences.folder("/Users/nail/hub", home: ""), "/Users/nail/hub",
      "and no home to compare against is no tilde either")

    // The other side of the same fact: the md icon with no hub names the row.
    let paired = event(
      "2026-10-17T09:00:00Z", .end, command: "reply", doc: chapter4, ok: true)
    let greyed = try drawn([paired], at: "2026-10-17T10:00:00Z", facts: Scenes.machine(hub: nil))
    let icon = try XCTUnwrap(
      greyed.documents?.rows.first?.actions.first { $0.kind == .openNote })
    XCTAssertEqual(icon.state, .off("Choose the hub in SETTINGS"))
    XCTAssertEqual(NoteFault.noHub.tip, "Choose the hub in SETTINGS", "and a click says it too")

    let live = try drawn([paired], at: "2026-10-17T10:00:00Z")
    XCTAssertEqual(
      live.documents?.rows.first?.actions.first { $0.kind == .openNote }?.state, .live,
      "with a hub and a note the icon works")
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
      idle.header.update?.kind, .update,
      "with nothing running the update is there to take")
    XCTAssertEqual(
      idle.settings.first { $0.kind == .desktop }?.actions.map(\.kind), [.installExtension])

    let running = try drawn([], at: at, facts: updating(true))
    XCTAssertTrue(
      running.header.versionWaiting, "the header is a ring while the child is up")
    XCTAssertEqual(
      running.header.version, "v2.14.0", "and the version is the one still installed")
    XCTAssertNil(running.header.update)
    let desktop = try XCTUnwrap(running.settings.first { $0.kind == .desktop })
    XCTAssertEqual(desktop.value, "not installed", "the row is where it was")
    XCTAssertTrue(desktop.actions.isEmpty, "and there is nothing on it to press")
  }

  /// The one fact the panel may not invent is whose account this is. `auth`
  /// is nothing until `gdoc auth status` has answered, which is every
  /// panel's first draw and the whole of a machine with no gdoc on it: the
  /// header is a ring over no address, and it offers no Sign out.
  func testTheHeaderClaimsNothingAboutTheAccountBeforeTheFirstAnswer() throws {
    let launching = Facts(versions: Versions(app: "2.15.0"), gdocPath: Scenes.gdocPath)
    XCTAssertNil(launching.auth, "nobody has asked")

    let first = try drawn([], at: "2026-10-17T10:00:00Z", facts: launching).header
    XCTAssertTrue(first.waiting, "the ring says the question is out")
    XCTAssertNil(first.account, "and no sign-in is claimed")
    XCTAssertEqual(first.glyph, .blank, "no green check over a token nobody read")
    XCTAssertTrue(first.actions.isEmpty, "there is nothing to sign out of yet")

    let answered = try drawn([], at: "2026-10-17T10:00:00Z", facts: Scenes.machine()).header
    XCTAssertFalse(answered.waiting)
    XCTAssertEqual(answered.glyph, .check)
    XCTAssertEqual(answered.account, Scenes.account)
    XCTAssertEqual(answered.actions.map(\.kind), [.signOut], "now there is")

    // And no SETTINGS row says any of it, which is where all of this was
    // until the second morning.
    XCTAssertFalse(
      SettingsRowKind.allCases.map(\.rawValue).contains("account"),
      "the account is the header's, and is said in one place")
    XCTAssertFalse(
      SettingsRowKind.allCases.map(\.rawValue).contains("gdoc"),
      "and so is the version")
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
      try drawn([], at: "2026-10-17T10:00:00Z", facts: signedIn).header.quiet,
      ["Account changed: first@example.com → second@example.com"])

    let signedOut = Facts(
      auth: .signedOut, previousAccount: before, versions: Versions(app: "2.15.0"),
      gdocPath: Scenes.gdocPath)
    XCTAssertEqual(
      try drawn([], at: "2026-10-17T10:00:00Z", facts: signedOut).header.quiet, [],
      "there is no second address, so there is no sentence")
  }

  /// The question is the only thing said under itself, and it keeps the
  /// account's own glyph.
  ///
  /// Two things went wrong on scene e10 at once. The glyph went to the empty
  /// ring, which is the panel's "nobody has asked" and the glyph beside
  /// "Claude Desktop, not installed", so the panel stopped saying whose
  /// account it is at the one moment a person is deciding about it. And the
  /// quiet line about a changed address was drawn under the question in the
  /// same ink and at the same indent as its consequence, which reads as a
  /// second consequence: press this and the account changes.
  func testTheQuestionKeepsTheAccountsGlyphAndSaysNothingElse() throws {
    let facts = Facts(
      account: "second@example.com", auth: .ok, previousAccount: "first@example.com",
      versions: Versions(app: "2.15.0"), gdocPath: Scenes.gdocPath)
    let quiet = try drawn([], at: "2026-10-17T10:00:00Z", facts: facts).header
    XCTAssertEqual(quiet.glyph, .check, "the token works, so the header says so")
    XCTAssertEqual(
      quiet.quiet, ["Account changed: first@example.com → second@example.com"],
      "and the memory of the address before is a line of its own")

    let asked = try drawn(
      [], at: "2026-10-17T10:00:00Z", facts: facts,
      ui: UIState(question: .signOut, questionAt: try at("2026-10-17T10:00:00Z"))
    ).header
    XCTAssertEqual(asked.glyph, quiet.glyph, "the question keeps the account's own glyph")
    XCTAssertEqual(asked.tone, quiet.tone)
    XCTAssertEqual(asked.quiet, [], "and nothing but its own consequence is said under it")
    XCTAssertNil(asked.version, "no version and no address beside a question")
    XCTAssertNil(asked.account)

    // A sign-out asked inside the banner's five seconds: the banner carries
    // a Sign out of its own, and two of them under one question is the
    // panel asking twice.
    let fresh = Facts(
      account: "second@example.com", auth: .ok, versions: Versions(app: "2.15.0"),
      gdocPath: Scenes.gdocPath, signedInAt: try at("2026-10-17T09:59:58Z"))
    let banner = try drawn([], at: "2026-10-17T10:00:00Z", facts: fresh).header
    XCTAssertNotNil(banner.banner, "the banner is on the screen for its five seconds")
    let asking = try drawn(
      [], at: "2026-10-17T10:00:00Z", facts: fresh,
      ui: UIState(question: .signOut, questionAt: try at("2026-10-17T10:00:00Z"))
    ).header
    XCTAssertNil(asking.banner, "and the question is the only thing under itself")
  }

  // MARK: - The pieces a test states

  private func question(of panel: PanelModel) throws -> QuestionModel {
    try XCTUnwrap(
      panel.header.question, "the header is the question while it is asked")
  }

  /// A document row's own menu, in both its states.
  ///
  /// The menu is where a pin and a hide are, so the four icons on the row
  /// stay four. A pinned row says Unpin and nothing else, because the one
  /// thing a person wants from a pin they have already made is to take it
  /// back, and a hide of a pinned document hides nothing: the press would
  /// look dead and then take the row away at the Unpin. There is no Unhide
  /// anywhere: a hidden document has no row to open a menu on.
  func testTheRowMenuSaysPinOrUnpinAndHide() throws {
    let events = [
      event(
        "2026-10-17T09:00:00Z", .end, command: "reply", doc: chapter4, title: "Ch 4", ok: true)
    ]

    let plain = try drawn(events, at: "2026-10-17T09:10:00Z")
    let row = try XCTUnwrap(plain.documents?.rows.first)
    XCTAssertFalse(row.isPinned, "nothing is pinned until somebody pins it")
    XCTAssertEqual(row.menu.map(\.word), ["Pin", "Hide from the list"])
    XCTAssertEqual(row.menu.map(\.kind), [.pinDoc, .hideDoc])
    XCTAssertEqual(row.menu.map(\.target), [chapter4, chapter4])
    XCTAssertEqual(row.actions.count, 4, "the four icons are four")

    let moment = try at("2026-10-17T09:05:00Z")
    let pinned = try drawn(
      events, at: "2026-10-17T09:10:00Z",
      ui: UIState(pinned: [DocumentChoice(doc: chapter4, at: moment, title: "Ch 4")]))
    let kept = try XCTUnwrap(pinned.documents?.rows.first)
    XCTAssertTrue(kept.isPinned, "which the row draws in its own column before the glyph")
    XCTAssertEqual(
      kept.menu.map(\.word), ["Unpin"],
      "and a pinned row offers no hide, because the pin would win and the press look dead")
    XCTAssertEqual(kept.menu.map(\.kind), [.unpinDoc])
    XCTAssertEqual(kept.actions.count, 4, "and the icons are still four")

    let hidden = try drawn(
      events, at: "2026-10-17T09:10:00Z",
      ui: UIState(hidden: [DocumentChoice(doc: chapter4, at: moment, title: "Ch 4")]))
    XCTAssertEqual(hidden.documents?.rows.count, 0, "a hidden document is not a row")
    XCTAssertNil(hidden.documents?.count, "and the label counts no rows")
    XCTAssertEqual(
      hidden.today?.rows.count, 1, "TODAY still carries its run, because the log is the log")
  }

  /// A pinned document the record has no line for any more: the row is the
  /// pin's own, and it says nothing about an action, because there is none
  /// to say.
  func testAPinnedDocumentWithNoLinesDrawsThePinsOwnRow() throws {
    let pin = DocumentChoice(
      doc: chapter4, at: try at("2026-08-01T09:00:00Z"), title: "Chapter 4. Pricing")
    let panel = try drawn([], at: "2026-10-17T09:10:00Z", ui: UIState(pinned: [pin]))

    let row = try XCTUnwrap(panel.documents?.rows.first, "the pin is the row")
    XCTAssertEqual(row.title, "Chapter 4. Pricing", "the title the pin kept")
    XCTAssertTrue(row.isPinned)
    XCTAssertEqual(
      row.glyph, .blank, "no state is invented for a document the record is silent about")
    XCTAssertEqual(
      row.sub, "falcon/chapters/04.md",
      "the sub-line says where the note is and nothing about an action")
    XCTAssertEqual(
      row.actions.first { $0.kind == .openDoc }?.state, .live, "and its open action works")
    XCTAssertEqual(panel.documents?.count, "14 days · 1", "the label counts it")
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
