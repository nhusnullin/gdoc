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
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class StateTests: XCTestCase {
  private let chapter4 = SceneDocuments.chapter4

  /// Every row the fold hands over is drawn, and the fold is what decides
  /// how many there are.
  ///
  /// DOCUMENTS drew the first five with the rest behind a Show more until
  /// 2026-10-18, and the word was the one piece of panel state a person's
  /// press changed. The five are the fold's own now: seven documents
  /// written to today are five rows in the model, newest first, and the
  /// list draws every one of them.
  func testTheListTheFoldHandsOverIsTheListDrawn() throws {
    let documents = [
      SceneDocuments.chapter4, SceneDocuments.chapter5, SceneDocuments.policy,
      SceneDocuments.northwind, SceneDocuments.partner, SceneDocuments.concept,
      SceneDocuments.proposal,
    ]
    var events: [Event] = []
    for (index, doc) in documents.enumerated() {
      let at = String(format: "2026-10-17T09:%02d:00Z", index)
      events.append(
        event(at, .end, command: "reply", doc: doc, call: Int64(index + 1), ok: true))
    }

    let drawn = try drawn(events, at: "2026-10-17T10:00:00Z")
    let list = try XCTUnwrap(drawn.documents)
    XCTAssertEqual(
      list.rows.map(\.doc), documents.suffix(5).reversed(),
      "the last five by their latest action, newest first")
    XCTAssertNil(list.empty, "a list with five rows says nothing about being empty")
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
    XCTAssertEqual(behind.header.quiet, [], "and says nothing under the header about it")
    XCTAssertEqual(behind.header.updateLook, .available)
    XCTAssertEqual(behind.header.update?.tip, "Update to v2.15.0")
    let ahead = try drawn(
      [], at: "2026-10-17T09:00:00Z", facts: Scenes.machine(latestStable: "v2.13.0"))
    XCTAssertEqual(ahead.header.version, "v2.14.0", "the line names one version")
    XCTAssertEqual(
      ahead.header.updateLook, .quiet, "and the glyph offers a check and no release")
  }

  /// A published release writes no line under the header. The number is on
  /// the glyph: its dot catches the eye and its tooltip says which version.
  ///
  /// It was a quiet line, "v2.15.0 is published", until 2026-10-28. Nail
  /// used the panel for ten days and said he does not need it: the dot is
  /// what he looks at, and a line that says the same thing in words is a
  /// line he reads past. The line under the header is for the answer to a
  /// press now, which is the one thing a person asked for words about:
  /// testEachRunThatInstalledNothingSaysWhyInAFixedSentence.
  func testAPublishedReleaseWritesNoLineUnderTheHeader() throws {
    let panel = try drawn(
      [], at: "2026-10-17T09:00:00Z", facts: Scenes.machine(latestStable: "v2.15.0"))
    XCTAssertEqual(panel.header.version, "v2.14.0")
    XCTAssertEqual(panel.header.quiet, [], "the release itself is not news in words")
    XCTAssertEqual(panel.header.updateLook, .available, "the dot is what says it")
    XCTAssertEqual(
      panel.header.update?.tip, "Update to v2.15.0",
      "and the tooltip is where the number lives")
  }

  /// The update glyph is on the line whenever there is a binary to update,
  /// in the states of the third look. Pressing it runs `gdoc update` in
  /// every one of them, because that command is the one that asks GitHub:
  /// the states differ in what this app has been told, never in what the
  /// press does.
  func testTheUpdateGlyphIsAlwaysThereInItsThreeStates() throws {
    let when = "2026-10-17T09:00:00Z"

    // Quiet: nothing newer is known, so the words are about asking.
    let quiet = try drawn([], at: when, facts: Scenes.machine())
    XCTAssertEqual(quiet.header.version, "v2.14.0")
    XCTAssertEqual(quiet.header.updateLook, .quiet)
    let glyph = try XCTUnwrap(quiet.header.update, "the glyph is there with nothing published")
    XCTAssertEqual(glyph.kind, .update)
    XCTAssertEqual(glyph.icon, .down)
    XCTAssertEqual(glyph.state, .live)
    XCTAssertEqual(glyph.tip, "Check for a new version")

    // Available: `help --json` listed a newer stable, so the glyph takes the
    // accent ink and its dot, and the line names the one version installed.
    let available = try drawn([], at: when, facts: Scenes.machine(latestStable: "v2.15.0"))
    XCTAssertEqual(available.header.version, "v2.14.0", "the line names the one installed")
    XCTAssertEqual(
      available.header.quiet, [],
      "and the one published is on the glyph's tooltip, not in words under the header")
    XCTAssertEqual(available.header.updateLook, .available)
    XCTAssertEqual(available.header.update?.tip, "Update to v2.15.0")
    XCTAssertEqual(available.header.update?.state, .live)

    // Running: the ring stands where the glyph is, and the version beside
    // it is the one still installed.
    let running = try drawn([], at: when, facts: updating(true))
    XCTAssertTrue(running.header.versionWaiting)
    XCTAssertEqual(running.header.version, "v2.14.0")
    XCTAssertNil(running.header.update, "there is nothing to press while the child is up")

    // And the second after a run that installed nothing: the check, with
    // the one thing the run said.
    let answered = try at(when)
    let checked = try drawn(
      [], at: when, facts: Scenes.machine(updateChecked: answered))
    XCTAssertEqual(checked.header.update?.state, .done)
    XCTAssertEqual(checked.header.update?.tip, "This is the newest release")
    let after = try drawn(
      [], at: "2026-10-17T09:00:02Z", facts: Scenes.machine(updateChecked: answered))
    XCTAssertEqual(after.header.update?.state, .live, "a second later it is quiet again")
    XCTAssertEqual(after.header.update?.tip, "Check for a new version")
  }

  /// Every run of either button that installed nothing says why, in a
  /// sentence this file holds as a literal.
  ///
  /// Three cases each, told apart by `ok` and the action word alone. The
  /// clock in the sentence is the moment the run ended, because a sentence
  /// stays until the next press and a person reading it later is owed to
  /// know which press it is about. Two of them name the command to type:
  /// the child's own warning is a paragraph about this machine, and a quiet
  /// line in a menu bar is not where anybody reads one.
  func testEachRunThatInstalledNothingSaysWhyInAFixedSentence() throws {
    let ended = try at("2026-10-17T14:02:00Z")
    let later = "2026-10-17T14:05:00Z"

    let update: [(UpdateMiss, String)] = [
      (.major, "Only a major version is newer. gdoc update --major installs it."),
      (.unreachable, "GitHub did not answer at 14:02. Nothing changed."),
      (.stopped, "The update stopped at 14:02. gdoc update in a terminal says why."),
    ]
    XCTAssertEqual(update.count, UpdateMiss.allCases.count, "every case has its sentence")
    for (miss, sentence) in update {
      let panel = try drawn(
        [], at: later,
        facts: Scenes.machine(
          latestStable: "v3.0.0", updateRun: UpdateRun(miss: miss, at: ended)))
      XCTAssertEqual(
        panel.header.quiet, [sentence],
        "the answer to the press takes the quiet line under the version")
      XCTAssertEqual(
        panel.header.update?.state, .live, "and the glyph is there to press again")
    }

    let desktop: [(ExtensionMiss, String)] = [
      (.major, "Nothing was installed at 14:02. It needs gdoc update --major first."),
      (
        .stopped,
        "The install stopped at 14:02. gdoc update --desktop in a terminal says why."
      ),
      (.handedNothing, "Claude Desktop was not handed the extension at 14:02."),
    ]
    XCTAssertEqual(desktop.count, ExtensionMiss.allCases.count)
    for (miss, sentence) in desktop {
      let panel = try drawn(
        [], at: later,
        facts: Scenes.machine(
          extensionState: .notInstalled, extensionRun: ExtensionRun(miss: miss, at: ended)))
      let row = try XCTUnwrap(panel.settings.first { $0.kind == .desktop })
      XCTAssertEqual(row.quiet, sentence)
      XCTAssertEqual(row.value, "not installed", "the row itself says what it always said")
      XCTAssertEqual(
        row.actions.map(\.kind), [.installExtension], "and the button stays where it was")
    }

    // Each of them fits two lines in the room the panel has, which is its
    // 360 less the 8 either side.
    let room = Grid.panelWidth - Grid.unit * 2
    let one = try XCTUnwrap(probes(QuietLine("one line"), scheme: .dark, width: room))
    let line = try XCTUnwrap(one["quiet.text"]).height
    for sentence in update.map(\.1) + desktop.map(\.1) {
      let measured = try XCTUnwrap(
        probes(QuietLine(sentence), scheme: .dark, width: room)["quiet.text"])
      XCTAssertLessThanOrEqual(measured.height, line * 2, "\(sentence) runs past two lines")
    }
  }

  /// A run that did something says nothing: the version that moved and the
  /// check for a second are the answers to those two presses, and a
  /// sentence beside either would be the panel saying it twice. So the line
  /// under the header is empty until the next press that installs nothing.
  func testARunThatDidSomethingLeavesNoLineUnderTheHeader() throws {
    let panel = try drawn(
      [], at: "2026-10-17T14:05:00Z", facts: Scenes.machine(latestStable: "v2.15.0"))
    XCTAssertEqual(panel.header.quiet, [])

    let handedOver = try drawn(
      [], at: "2026-10-17T14:05:00Z", facts: Scenes.machine(extensionState: .notInstalled))
    let row = try XCTUnwrap(handedOver.settings.first { $0.kind == .desktop })
    XCTAssertNil(row.quiet, "a row nobody has pressed has nothing to say")
  }

  /// A machine with no gdoc on it has nothing to update, so there is no
  /// glyph: the header says where it looked instead.
  func testAMachineWithNoGdocOffersNoUpdate() throws {
    let missing = try drawn(
      [], at: "2026-10-17T09:00:00Z", facts: Scenes.machine(gdocFound: false))
    XCTAssertEqual(missing.header.version, "not found")
    XCTAssertNil(missing.header.update)
  }

  /// The md icon is grey because no note names the document, which is the
  /// reason it carries where nothing else was found.
  ///
  /// The Hub row was the other half of this test until that evening. The
  /// note is found now, so there is no folder to choose, and "Choose the hub
  /// in SETTINGS" named a row that no longer exists. The other sentence the
  /// icon can carry is the finder's own, two notes naming one document:
  /// testTwoNotesNamingOneDocumentSaySoOnTheIcon.
  func testTheMdIconIsGreyOnlyWhenNoNoteNamesTheDocument() throws {
    let paired = event(
      "2026-10-17T09:00:00Z", .end, command: "reply", doc: chapter4, ok: true)

    let greyed = try drawn(
      [paired], at: "2026-10-17T10:00:00Z", facts: Scenes.machine(pairing: [:]))
    let icon = try XCTUnwrap(
      greyed.documents?.rows.first?.actions.first { $0.kind == .openNote })
    XCTAssertEqual(icon.state, .off("No note is paired with this document"))
    XCTAssertEqual(
      NoteFault.notPaired.tip, "No note is paired with this document",
      "and a click says the same")

    let live = try drawn([paired], at: "2026-10-17T10:00:00Z")
    XCTAssertEqual(
      live.documents?.rows.first?.actions.first { $0.kind == .openNote }?.state, .live,
      "with a note naming it the icon works")

    XCTAssertFalse(
      SettingsRowKind.allCases.contains { $0.label == "Hub" },
      "and no row in SETTINGS offers to choose a folder")
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
      [held("2026-10-17T09:50:00Z", hold: "h_1", until: "2026-10-17T10:50:00Z", doc: chapter4)],
      at: "2026-10-17T10:00:00Z",
      ui: UIState(
        dismissed: ["h_1"], question: .signOut,
        questionAt: try at("2026-10-17T10:00:00Z")))
    XCTAssertTrue(hidden.now.isEmpty, "the card was dismissed, so NOW draws nothing")
    XCTAssertEqual(
      try question(of: hidden).consequence, "1 hold will stop.",
      "the hold is still waiting in the chat, and the sign-out drops it")

    let ranOut = try drawn(
      [held("2026-10-17T08:00:00Z", hold: "h_2", until: "2026-10-17T09:00:00Z", doc: chapter4)],
      at: "2026-10-17T10:00:00Z", ui: asking)
    XCTAssertEqual(ranOut.now.count, 1, "the card is on the screen")
    XCTAssertEqual(
      try question(of: ranOut).consequence, "Nothing is running. Only the token goes.",
      "a hold whose life ran out is waiting for nobody, so the sign-out stops nothing")
  }

  /// An id Drive could not have written is a line gdoc did not write. The
  /// row carries no press and the icon that would build a URL out of it says
  /// so before the press, because `Opener` refuses to build one and a press
  /// that does nothing says nothing.
  func testAnIdDriveCouldNotHaveWrittenGreysItsTwoIcons() throws {
    let planted = "not-a-drive-id"
    let panel = try drawn(
      [event("2026-10-17T09:00:00Z", .end, command: "reply", doc: planted, ok: true)],
      at: "2026-10-17T09:10:00Z")

    let row = try XCTUnwrap(panel.documents?.rows.first, "the row is drawn, as the record says")
    XCTAssertEqual(row.doc, planted)
    XCTAssertNil(row.opens, "there is no document behind it to open")
    XCTAssertEqual(
      row.actions.first { $0.kind == .copyLink }?.state, .off(Tips.noLink),
      "and no link to copy")

    let real = try drawn(
      [event("2026-10-17T09:00:00Z", .end, command: "reply", doc: chapter4, ok: true)],
      at: "2026-10-17T09:10:00Z")
    XCTAssertEqual(
      real.documents?.rows.first?.opens, chapter4,
      "an id of Drive's own shape is the press")
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

  /// The row opens the document it names, and the four icons left on it are
  /// the note, the chat, the link and the pin.
  ///
  /// The Google Docs icon went at the fifth look (docs/v2/DECISIONS.md,
  /// 2026-10-28): the row itself is the press, as a NOW row's has been
  /// since the fourth, so the icon was a second control for one action. The
  /// row carries the id it opens under its own rule, the one a NOW row
  /// follows: an id Drive could not have written opens nothing, because
  /// `Opener` refuses to build a URL out of it.
  ///
  /// The pin is the last of the four since the third look
  /// (docs/v2/DECISIONS.md, 2026-10-18): it was a word in a menu Nail never
  /// found, and a menu nobody finds is no affordance. The outline tack pins
  /// and says Pin, the filled one takes it back and says Unpin, so the row
  /// says which it is without a word.
  func testADocumentRowOpensItselfAndCarriesFourIcons() throws {
    for id in [chapter4, "not-a-drive-id"] {
      let events = [
        event("2026-10-17T09:00:00Z", .end, command: "reply", doc: id, title: "Ch 4", ok: true)
      ]

      let plain = try drawn(events, at: "2026-10-17T09:10:00Z")
      let row = try XCTUnwrap(plain.documents?.rows.first)
      XCTAssertEqual(
        row.opens, id == chapter4 ? chapter4 : nil,
        "the row opens an id of Drive's own shape and nothing else")
      XCTAssertFalse(row.isPinned, "nothing is pinned until somebody pins it")
      XCTAssertEqual(
        row.actions.map(\.icon), [.md, .branch, .link, .pin],
        "the note, the chat, the link and the pin at the end")
      XCTAssertEqual(row.actions.map(\.kind), [.openNote, .openChat, .copyLink, .pinDoc])
      XCTAssertNil(
        row.actions.first { $0.kind == .openDoc },
        "the Google Docs icon went: the row is that press now")
      let pin = try XCTUnwrap(row.actions.last)
      XCTAssertEqual(pin.tip, "Pin")
      XCTAssertEqual(pin.state, .live, "a pin is a press on every row there is")
      XCTAssertEqual(pin.target, id)

      let moment = try at("2026-10-17T09:05:00Z")
      let pinned = try drawn(
        events, at: "2026-10-17T09:10:00Z",
        ui: UIState(pinned: [DocumentChoice(doc: id, at: moment, title: "Ch 4")]))
      let kept = try XCTUnwrap(pinned.documents?.rows.first)
      XCTAssertTrue(kept.isPinned)
      XCTAssertEqual(
        kept.actions.map(\.icon), [.md, .branch, .link, .pinOn],
        "the filled tack is the one thing that changes")
      XCTAssertEqual(kept.actions.map(\.kind), [.openNote, .openChat, .copyLink, .unpinDoc])
      let unpin = try XCTUnwrap(kept.actions.last)
      XCTAssertEqual(unpin.tip, "Unpin")
      XCTAssertEqual(unpin.target, id)
    }
  }

  /// A document row's own menu is one word, and a pinned row has no menu.
  ///
  /// Hide is the one thing left in it: the pin is the last of the row's four
  /// icons since the third look. A pinned row offers nothing, because the pin is the
  /// louder answer and a hide of a pinned document hides nothing: the press
  /// would look dead and then take the row away at the unpin. There is no
  /// Unhide anywhere, because a hidden document has no row to open a menu on.
  func testTheRowMenuIsHideAndAPinnedRowHasNone() throws {
    let events = [
      event(
        "2026-10-17T09:00:00Z", .end, command: "reply", doc: chapter4, title: "Ch 4", ok: true)
    ]

    let plain = try drawn(events, at: "2026-10-17T09:10:00Z")
    let row = try XCTUnwrap(plain.documents?.rows.first)
    XCTAssertEqual(row.menu.map(\.word), ["Hide from the list"])
    XCTAssertEqual(row.menu.map(\.kind), [.hideDoc])
    XCTAssertEqual(row.menu.map(\.target), [chapter4])

    let moment = try at("2026-10-17T09:05:00Z")
    let pinned = try drawn(
      events, at: "2026-10-17T09:10:00Z",
      ui: UIState(pinned: [DocumentChoice(doc: chapter4, at: moment, title: "Ch 4")]))
    let kept = try XCTUnwrap(pinned.documents?.rows.first)
    XCTAssertTrue(kept.menu.isEmpty, "a pinned row has nothing to offer, so it opens no menu")

    let hidden = try drawn(
      events, at: "2026-10-17T09:10:00Z",
      ui: UIState(hidden: [DocumentChoice(doc: chapter4, at: moment, title: "Ch 4")]))
    XCTAssertEqual(hidden.documents?.rows.count, 0, "a hidden document is not a row")
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
      row.sub, "~/hub/falcon/chapters/04.md",
      "the sub-line says where the note is, with the home as a tilde, and "
        + "nothing about an action")
    XCTAssertEqual(row.opens, chapter4, "and the row itself opens it")
    XCTAssertEqual(panel.documents?.rows.count, 1, "and it is the whole of the list")
  }

  /// The mark in front of a document row means one thing: the last run on
  /// that document failed.
  ///
  /// It carried five marks until the fourth look: a green check for a run
  /// that worked, a dot for a live watch, a dim dot for one that stopped, a
  /// pause for a hold, the red exclamation for a failure. Nail read a list
  /// of grey dots and green checks on 2026-10-26 and nothing on the panel
  /// said why, and the second line already says what the last run was. So
  /// the check, the dots and the pause go, the exclamation stays, and the
  /// column stays with it: a row with nothing to say still keeps the 16 pt
  /// every title starts after.
  ///
  /// The table is `DocumentState.allCases`, so a ninth state cannot be
  /// added without an answer here. Each record is built through the fold,
  /// and the fold's own state is read back before the glyph is, so the
  /// table is the states and not the lines that make them.
  func testADocumentRowIsMarkedOnlyWhenItsLastRunFailed() throws {
    XCTAssertEqual(DocumentState.allCases.count, 8, "the eight states a row can be in")

    for state in DocumentState.allCases {
      let made = try record(for: state)
      let fold = Documents.fold(
        made.events, now: try at(drawnAt), pinned: made.ui.pinned,
        alive: { made.alive.contains($0) })
      XCTAssertEqual(
        fold.rows.first?.state, state, "the record built for \(state) folds to it")

      let panel = try drawn(made.events, at: drawnAt, ui: made.ui, alive: made.alive)
      let row = try XCTUnwrap(panel.documents?.rows.first, "\(state) draws a row")
      if state == .failed {
        XCTAssertEqual(row.glyph, .exclamation, "a run that failed is the one marked row")
        XCTAssertEqual(row.tone, .bad)
      } else {
        XCTAssertEqual(row.glyph, .blank, "\(state) is not a failure, so nothing is drawn")
        XCTAssertEqual(row.tone, .quiet, "\(state) draws no colour either")
      }
    }

    // Scene e1, which is the render of the list: the reply on Chapter 5
    // failed, the person dismissed its card in NOW, and the one mark the
    // list can have is on that one row.
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e1" })
    let drawn = try XCTUnwrap(try scene.model().documents).rows
    XCTAssertEqual(
      drawn.filter { $0.glyph == .exclamation }.map(\.doc), [SceneDocuments.chapter5],
      "e1 draws one mark, on the document whose last run failed")
    XCTAssertEqual(
      drawn.filter { $0.glyph == .blank }.count, 5, "and nothing in front of the other five")
    XCTAssertTrue(try scene.model().now.isEmpty, "with the failure's own card dismissed")
  }

  /// The card under a rested pointer is the row uncut, and never the row
  /// plus.
  ///
  /// Four fields and no fifth: the document's name, what happened to it and
  /// when, where its note is, and the one hint that retires itself. Each is
  /// a fact the row beside it already draws. The card draws them whole,
  /// where the row has one line of 360 points and cuts; the one thing it
  /// adds is the moment's own day, date and year, which is what a cut line
  /// has no room for and what the row's "Sat" or "17:00" cannot say.
  ///
  /// The table is `DocumentState.allCases` through the same records the
  /// mark's table is built from, so a ninth state cannot be added without
  /// an answer here, and the words are the second line's own.
  func testTheCardIsTheRowUncutAndNeverTheRowPlus() throws {
    let table: [(DocumentState, String?)] = [
      (.hold, "hold"),
      (.running, "replying"),
      (.live, "live"),
      (.liveStopped, "live stopped Sat 17 Oct 2026, 17:05"),
      (.done, "replied Sat 17 Oct 2026, 17:00"),
      (.failed, "reply failed Sat 17 Oct 2026, 17:00"),
      (.unknown, "outcome unknown · Sat 17 Oct 2026, 17:00"),
      (.pinnedOnly, nil),
    ]
    XCTAssertEqual(
      table.map(\.0), DocumentState.allCases, "every state a row can be in has a card")

    for (state, expected) in table {
      let made = try record(for: state)
      let panel = try drawn(made.events, at: drawnAt, ui: made.ui, alive: made.alive)
      let row = try XCTUnwrap(panel.documents?.rows.first, "\(state) draws a row")
      let card = row.card

      XCTAssertEqual(
        Mirror(reflecting: card).children.compactMap(\.label),
        ["title", "when", "note", "hint"], "four fields and no fifth")
      XCTAssertEqual(card.title, row.title, "\(state): the card's name is the row's name")
      XCTAssertEqual(card.when, expected, "\(state): the second line's words, uncut")
      XCTAssertEqual(
        card.note, "~/hub/falcon/chapters/04.md",
        "\(state): the whole path, with the home as a tilde")
      XCTAssertNil(card.hint, "\(state): the hint is away once a row has been opened")
    }

    // A record that named the document is the name on the card, and a
    // record that named none is the id on both: the card never knows a name
    // the row does not.
    let unnamed = try drawn(
      try record(for: .live).events, at: drawnAt, ui: UIState(), alive: [7101])
    XCTAssertEqual(try XCTUnwrap(unnamed.documents?.rows.first).card.title, chapter4)

    // A document with no note says so in the row's own words, which is the
    // one place the card is shorter than the path it would have written.
    let nameless = try drawn(
      [
        event(
          "2026-10-17T17:00:00Z", .end, command: "reply", doc: SceneDocuments.northwind,
          title: "Northwind", ok: true)
      ], at: drawnAt)
    XCTAssertEqual(try XCTUnwrap(nameless.documents?.rows.first).card.note, "no note")

    // The hint is the one line on the card that is not about the document.
    // It stands until the person's own first click on a row, and it is
    // never on a row a click would not open.
    let events = [
      event(
        "2026-10-17T17:00:00Z", .end, command: "reply", doc: chapter4, title: "Ch 4", ok: true)
    ]
    let first = try drawn(events, at: drawnAt, ui: UIState(cardHintSeen: false))
    XCTAssertEqual(
      try XCTUnwrap(first.documents?.rows.first).card.hint, "Click opens in Google Docs")

    let forged = try drawn(
      [
        event(
          "2026-10-17T17:00:00Z", .end, command: "reply", doc: "not-a-drive-id", title: "Ch 4",
          ok: true)
      ], at: drawnAt, ui: UIState(cardHintSeen: false))
    let planted = try XCTUnwrap(forged.documents?.rows.first)
    XCTAssertNil(planted.opens, "an id Drive could not have written opens nothing")
    XCTAssertNil(planted.card.hint, "so the card never offers the words for a press")
  }

  /// A run that failed and a watch that was stopped read as two different
  /// things, in the words as well as in the mark.
  ///
  /// They read alike until the fourth look: the row of a failed reply said
  /// "stopped Tue" and the row of a stopped watch "live stopped 12:26", and
  /// both drew a dot or a check nobody could tell apart. A failure now
  /// names the write and says failed, and neither line carries the other's
  /// word.
  func testAFailedRunDoesNotReadAsAStoppedOne() throws {
    let failed = try drawn(
      [
        event(
          "2026-10-13T12:00:00Z", .end, command: "reply", doc: chapter4, title: "Ch 4",
          ok: false)
      ], at: drawnAt)
    let failedRow = try XCTUnwrap(failed.documents?.rows.first)
    XCTAssertEqual(
      failedRow.sub, "reply failed Tue · ~/hub/falcon/chapters/04.md",
      "the write's own name, and the one word for how it went")
    XCTAssertEqual(failedRow.glyph, .exclamation)

    let watch = try drawn(
      [
        event(
          "2026-10-17T12:00:00Z", .start, command: "comments", doc: chapter4, wait: true),
        event(
          "2026-10-17T12:26:00Z", .end, command: "comments", doc: chapter4, wait: true,
          ok: true),
      ], at: drawnAt)
    let watchRow = try XCTUnwrap(watch.documents?.rows.first)
    XCTAssertEqual(
      watchRow.sub, "live stopped 12:26 · ~/hub/falcon/chapters/04.md",
      "a watch somebody stopped kept its words")
    XCTAssertEqual(watchRow.glyph, .blank, "and nothing in front of them")

    XCTAssertFalse(
      failedRow.sub.contains("stopped"), "a failure never says the word a watch says")
    XCTAssertFalse(
      watchRow.sub.contains("failed"), "and a watch that stopped never says failed")
  }

  /// A write the read-back did not confirm is a run that worked: no mark,
  /// and the word it has today.
  ///
  /// `verified: false` is the hollow check of principle 5, which is a write
  /// that happened and a read-back that did not say so. The mark is for
  /// `ok: false` and nothing else, so the two rows here are one drawing.
  func testAWriteThatWasNotReadBackIsStillARunThatWorked() throws {
    for verified in [true, false] {
      let panel = try drawn(
        [
          event(
            "2026-10-13T12:00:00Z", .end, command: "reply", doc: chapter4, title: "Ch 4",
            ok: true, verified: verified)
        ], at: drawnAt)
      let row = try XCTUnwrap(panel.documents?.rows.first)
      XCTAssertEqual(row.glyph, .blank, "verified: \(verified) is no failure")
      XCTAssertEqual(
        row.sub, "replied Tue · ~/hub/falcon/chapters/04.md",
        "and the word is the write's own past, as it was before the fourth look")
    }
  }

  /// The moment every table above is drawn at.
  private let drawnAt = "2026-10-17T18:00:00Z"

  /// One record for each of the eight states, with the person's own state
  /// and the pids the kernel knows about, all on the one document.
  private func record(for state: DocumentState) throws -> (
    events: [Event], ui: UIState, alive: Set<Int32>
  ) {
    switch state {
    case .hold:
      return (
        [held("2026-10-17T17:00:00Z", hold: "h-1", until: "2026-10-17T19:00:00Z", doc: chapter4)],
        UIState(), []
      )
    case .running:
      return (
        [event("2026-10-17T17:00:00Z", .start, command: "reply", doc: chapter4)], UIState(), [7101]
      )
    case .live:
      return (
        [
          event(
            "2026-10-17T17:59:00Z", .start, command: "comments", doc: chapter4, wait: true)
        ], UIState(), [7101]
      )
    case .liveStopped:
      return (
        [
          event(
            "2026-10-17T17:00:00Z", .start, command: "comments", doc: chapter4, wait: true),
          event(
            "2026-10-17T17:05:00Z", .end, command: "comments", doc: chapter4, wait: true,
            ok: true),
        ], UIState(), []
      )
    case .done:
      return (
        [
          event(
            "2026-10-17T17:00:00Z", .end, command: "reply", doc: chapter4, title: "Ch 4",
            ok: true, verified: true)
        ], UIState(), []
      )
    case .failed:
      return (
        [
          event(
            "2026-10-17T17:00:00Z", .end, command: "reply", doc: chapter4, title: "Ch 4",
            ok: false)
        ], UIState(), []
      )
    case .unknown:
      // The same start as `.running`, with a pid the kernel does not know.
      return (
        [event("2026-10-17T17:00:00Z", .start, command: "reply", doc: chapter4)], UIState(), []
      )
    case .pinnedOnly:
      return (
        [],
        UIState(
          pinned: [
            DocumentChoice(doc: chapter4, at: try at("2026-10-17T17:00:00Z"), title: "Ch 4")
          ]), []
      )
    }
  }

  /// Every row in NOW that names a document opens it, and no row opens
  /// anything else.
  ///
  /// The fourth look made the row itself the press (docs/v2/DECISIONS.md,
  /// 2026-10-26): a watch, a hold, a write that stopped and a job each say
  /// which document they are about, and until now only the stuck card
  /// offered a way into it. The id is the record's, so it is held to the
  /// shape Drive writes before a press is offered at all: a line somebody
  /// planted opens nothing, and so does a publish that has not created a
  /// document yet.
  func testEveryNowRowThatNamesADocumentOpensIt() throws {
    let planted = "not-a-drive-id"
    for shape in NowShape.allCases {
      let named = try row(shape, doc: chapter4)
      XCTAssertEqual(
        named.opens, chapter4, "a \(shape.rawValue) row names a document, so it opens it")
      XCTAssertEqual(NowPress.opens(named), chapter4, "and the press is offered")
      XCTAssertEqual(
        NowPress.tip(for: named), Tips.openDoc,
        "and the title says so in the Google Docs icon's own words")

      let forged = try row(shape, doc: planted)
      XCTAssertNil(
        forged.opens, "a \(shape.rawValue) row whose id Drive could not have written opens none")
      XCTAssertNil(NowPress.opens(forged))
      XCTAssertNil(NowPress.tip(for: forged), "and its title offers nothing")

      guard shape != .live else { continue }
      let nameless = try row(shape, doc: nil)
      XCTAssertNil(
        nameless.opens, "a \(shape.rawValue) row that names no document opens none")
      XCTAssertNil(NowPress.tip(for: nameless))
    }
  }

  /// A hold that is asking its question opens nothing.
  ///
  /// The card is two buttons and the sentence that says what follows from
  /// them, so a click anywhere on it that opened a browser tab would be an
  /// answer nobody gave. The document is still on the item: the card goes
  /// back to offering when the question runs out, and the press comes back
  /// with it.
  func testAHoldThatIsAskingOpensNothing() throws {
    let asked = try at("2026-10-17T17:30:00Z")
    let ui = UIState(question: .dismissHold("h-1"), questionAt: asked)
    let panel = try drawn(
      [held("2026-10-17T17:00:00Z", hold: "h-1", until: "2026-10-17T19:00:00Z", doc: chapter4)],
      at: "2026-10-17T17:30:01Z", ui: ui)
    let item = try XCTUnwrap(panel.now.first, "the hold is the one row in NOW")
    guard case .hold(_, _, let asking, _) = item else {
      return XCTFail("a held write is a hold card")
    }
    XCTAssertTrue(asking, "the card is asking")
    XCTAssertEqual(item.opens, chapter4, "the document is still what the row is about")
    XCTAssertNil(NowPress.opens(item), "and the card offers no press while it asks")
    XCTAssertNil(NowPress.tip(for: item), "so its title promises none")
  }

  /// The four shapes NOW draws, each built from the record.
  private enum NowShape: String, CaseIterable {
    case hold
    case stuck
    case job
    case live
  }

  /// One row of NOW in the shape asked for, about the document named.
  private func row(_ shape: NowShape, doc: String?) throws -> NowItem {
    let events: [Event]
    var alive: Set<Int32> = []
    switch shape {
    case .hold:
      events = [
        held("2026-10-17T17:00:00Z", hold: "h-1", until: "2026-10-17T19:00:00Z", doc: doc)
      ]
    case .stuck:
      events = [
        event("2026-10-17T17:00:00Z", .start, command: "publish", doc: doc),
        event("2026-10-17T17:01:00Z", .end, command: "publish", doc: doc, ok: false),
      ]
    case .job:
      events = [event("2026-10-17T17:00:00Z", .start, command: "publish", doc: doc)]
      alive = [7101]
    case .live:
      events = [
        event("2026-10-17T17:59:00Z", .start, command: "comments", doc: doc, wait: true)
      ]
      alive = [7101]
    }
    let panel = try drawn(events, at: drawnAt, alive: alive)
    return try XCTUnwrap(
      panel.now.first, "the record above draws no \(shape.rawValue) row")
  }

  /// A write a rule held, as gdoc writes it: one line, and no call.
  private func held(_ at: String, hold: String, until: String, doc: String? = nil) -> Event {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return Event(
      schema: 1, at: formatter.date(from: at) ?? Date(timeIntervalSince1970: 0),
      gdoc: "v2.15.0", pid: 7101, run: "3000000000000002", call: nil, kind: .held,
      command: "reply", via: "chat", doc: doc, wait: false, title: nil, ok: nil,
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
    doc: String? = nil, call: Int64 = 1, wait: Bool = false, title: String? = nil,
    made: String? = nil, ok: Bool? = nil, verified: Bool? = nil
  ) -> Event {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return Event(
      schema: 1, at: formatter.date(from: at) ?? Date(timeIntervalSince1970: 0),
      gdoc: "v2.15.0", pid: 7101, run: "3000000000000002", call: call, kind: kind,
      command: command, via: via, doc: doc, wait: wait, interrupted: false, step: nil,
      index: nil, count: nil, title: title, made: made, ok: ok, verified: verified, stop: nil,
      line: nil, ms: nil, hold: nil, rule: nil, until: nil)
  }
}
