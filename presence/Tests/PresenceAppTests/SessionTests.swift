// The session the shell is wired out of.
//
// Every room it reaches is handed to it here: a gdoc of this file's own
// making, a desk that writes down what it was asked, a login register that
// registers nothing, a store in memory and a directory of records the test
// wrote. So the whole of the wiring is a test: no browser opens, no login
// item is touched, nothing is written into anybody's defaults and the gdoc
// that answers is a shell script that logs its argv.
//
// The record is written before the session starts, so the first pass of the
// tail is what delivers it, which is the route every later line takes too.

import Foundation
import PresenceCore
import PresenceUI
import XCTest

@testable import PresenceApp

@MainActor
final class SessionTests: XCTestCase {
  private var root: URL!
  private var session: Session?

  /// One document, in the Drive shape, so its row carries the four icons.
  private static let doc = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789"

  /// A second document, for the tests that need two.
  private static let other = "1ZzYyXxWwVvUuTtSsRrQqPpOoNnMmLlKkJjIi"

  override func setUp() async throws {
    root = URL(fileURLWithPath: NSTemporaryDirectory())
      .appendingPathComponent("gdoc-session-\(UUID().uuidString)")
    try FileManager.default.createDirectory(
      at: records, withIntermediateDirectories: true)
    try FileManager.default.createDirectory(
      at: root.appendingPathComponent(".local/bin"), withIntermediateDirectories: true)
    try writeTheFakeGdoc()
  }

  override func tearDown() async throws {
    session?.stop()
    session = nil
    try? FileManager.default.removeItem(at: root)
  }

  // MARK: what the shell is handed

  /// The Network row is the record's own word and never a connection this
  /// app opened: `stop: "network"` is what gdoc writes when a request did
  /// not get to Google, and a later command that worked is the answer to it.
  ///
  /// Two commands say nothing about Google and are read past: `auth logout`
  /// opens no connection, and `update` reads GitHub. A `not_found` is
  /// Google answering, so it clears the row the way a success does.
  func testTheNetworkRowIsTheRecordsOwnWord() throws {
    let failed = line(
      kind: "end", command: "reply", call: 1, doc: Self.doc,
      extra: "\"ok\":false,\"stop\":\"network\"")
    try writeTheRecord([line(kind: "start", command: "reply", call: 1, doc: Self.doc), failed])

    let session = made()
    answered(session)
    XCTAssertEqual(
      session.facts.network, NetworkFact(reachable: false), "the last thing gdoc said")
    XCTAssertTrue(
      session.model.settings.contains { $0.kind == .network },
      "which the panel draws as a row of its own")

    // A command that worked afterwards answers it, and the row goes.
    let settled = expectation(description: "the later end reached the panel")
    settled.assertForOverFulfill = false
    session.start { [weak session] in
      if session?.facts.network == NetworkFact(reachable: true) { settled.fulfill() }
    }
    try writeTheRecord([
      line(kind: "start", command: "reply", call: 1, doc: Self.doc), failed,
      line(kind: "start", command: "reply", call: 2, doc: Self.doc),
      line(kind: "end", command: "reply", call: 2, doc: Self.doc, extra: "\"ok\":true"),
    ])
    wait(for: [settled], timeout: 20)
    XCTAssertFalse(session.model.settings.contains { $0.kind == .network })

    // And the ends that are not about Google, read as the fold reads them.
    let logout = try Event.decode(
      line(kind: "end", command: "auth logout", call: nil, doc: nil, extra: "\"ok\":true"))
    XCTAssertEqual(
      Reachability.latest(in: [try Event.decode(failed), logout]), false,
      "a logout opens no connection, so it answers nothing")

    let github = try Event.decode(
      line(
        kind: "end", command: "update", call: nil, doc: nil,
        extra: "\"ok\":false,\"stop\":\"network\""))
    XCTAssertNil(
      Reachability.latest(in: [github]), "update reads GitHub, and the row names Google")

    let notFound = try Event.decode(
      line(
        kind: "end", command: "read", call: 2, doc: Self.doc,
        extra: "\"ok\":false,\"stop\":\"not_found\""))
    XCTAssertEqual(
      Reachability.latest(in: [try Event.decode(failed), notFound]), true,
      "a 404 is Google answering")
  }

  /// The Claude Desktop row says the one thing the app can prove: gdoc never
  /// wrote the extension here, because there is no `gdoc.mcpb` beside the
  /// binary. Once there is one, the app has nothing left to say: whether
  /// Desktop loaded it is not a question it has a way to ask.
  func testTheExtensionRowAppearsOnlyUntilGdocHasWrittenOne() throws {
    try writeTheRecord([])

    let session = made()
    answered(session)
    XCTAssertEqual(session.facts.extensionState, .notInstalled)
    let row = try XCTUnwrap(session.model.settings.first { $0.kind == .desktop })
    XCTAssertEqual(row.value, "not installed")
    XCTAssertEqual(row.actions.map(\.kind), [.installExtension])

    try Data("not really an extension".utf8).write(
      to: gdoc.deletingLastPathComponent().appendingPathComponent("gdoc.mcpb"))
    let written = made()
    answered(written)
    XCTAssertEqual(written.facts.extensionState, .written)
    XCTAssertNil(
      written.model.settings.first { $0.kind == .desktop },
      "a file that is there says nothing about what Desktop loaded")
  }

  /// A run that wrote the extension and could not hand it to Claude Desktop
  /// leaves the row and its button where they were.
  ///
  /// `update --desktop` writes `gdoc.mcpb` before it opens it, and an open
  /// that refused is a warning with the file still there. So the file alone
  /// would take the row away on a machine that was never offered anything,
  /// and the person who pressed the button would be left with nothing to
  /// press and nothing said. `extension_opened` is what the row goes on, and
  /// the fake gdoc here writes the file and prints no such field, which is
  /// what the real one prints when the open failed.
  func testARunThatOpenedNothingLeavesTheRowWhereItWas() throws {
    try writeTheRecord([])
    let session = made()
    answered(session)
    XCTAssertEqual(session.facts.extensionState, .notInstalled)

    let mcpb = gdoc.deletingLastPathComponent().appendingPathComponent("gdoc.mcpb")
    let ended = expectation(description: "the extension run reached the panel")
    ended.assertForOverFulfill = false
    session.start { [weak session] in
      guard let session, !session.extensionHandedOver else { return }
      ended.fulfill()
    }
    session.perform(.installExtension, "desktop")
    wait(for: [ended], timeout: 20)

    XCTAssertTrue(
      FileManager.default.fileExists(atPath: mcpb.path), "the run wrote the file either way")
    XCTAssertEqual(
      session.facts.extensionState, .notInstalled,
      "and Desktop was never offered it, whatever is on the disk")
    let row = try XCTUnwrap(session.model.settings.first { $0.kind == .desktop })
    XCTAssertEqual(row.actions.map(\.kind), [.installExtension], "so the button stays")
    XCTAssertTrue(try argv().contains("update --desktop"), "one run, and the flag it was for")
  }

  /// A run that installed nothing says why, and the sentence stays until
  /// somebody presses one of the two buttons again.
  ///
  /// The fake gdoc here writes the extension and prints no
  /// `extension_opened`, which is the real one's answer when Claude Desktop
  /// refused the open: the row keeps its button and gains the one line that
  /// says what happened. Nothing writes the sentence down, so a launch
  /// starts with none, and a press of either button takes whichever is
  /// there off: a line about a run of five minutes ago under a run that is
  /// starting now reads as a line about this one.
  func testASentenceAboutARunStaysUntilTheNextPress() throws {
    try writeTheRecord([])
    let session = made()
    answered(session)
    XCTAssertNil(session.facts.extensionRun, "a launch starts with nothing to say")
    XCTAssertNil(session.facts.updateRun)

    let saidWhy = expectation(description: "the extension run reached the panel")
    saidWhy.assertForOverFulfill = false
    session.start { [weak session] in
      guard let session, session.facts.extensionRun != nil else { return }
      saidWhy.fulfill()
    }
    session.perform(.installExtension, "desktop")
    wait(for: [saidWhy], timeout: 20)

    XCTAssertEqual(session.facts.extensionRun?.miss, .handedNothing)
    let said = try XCTUnwrap(session.model.settings.first { $0.kind == .desktop })
    XCTAssertEqual(
      said.actions.map(\.kind), [.installExtension], "the button is still the one thing to do")
    XCTAssertNotNil(said.quiet, "and the row says what the run did")

    let pressedAgain = expectation(description: "the update run reached the panel")
    pressedAgain.assertForOverFulfill = false
    session.start { [weak session] in
      guard let session, session.updateChecked != nil else { return }
      pressedAgain.fulfill()
    }
    session.perform(.update, "gdoc")
    wait(for: [pressedAgain], timeout: 20)

    XCTAssertNil(session.facts.extensionRun, "the next press took the sentence off")
    XCTAssertNil(
      session.facts.updateRun, "and this run is already on the newest release")
    let quiet = try XCTUnwrap(session.model.settings.first { $0.kind == .desktop })
    XCTAssertNil(quiet.quiet, "so the row is back to what it was")
  }

  /// Update and Install extension take one slot. Both argv lists are `gdoc
  /// update` and `--desktop` installs the release it writes the extension
  /// of, so two children at once are two processes staging and renaming one
  /// binary: the `.previous` copy can end up being the new binary rather
  /// than the old one. The runner's own rule is per argv list and would let
  /// these two through.
  func testTheExtensionAndTheUpdateCannotRunAtOnce() throws {
    try writeTheRecord([])
    let session = made()
    answered(session)

    let ended = expectation(description: "the extension run reached the panel")
    ended.assertForOverFulfill = false
    session.start { [weak session] in
      guard let session, !session.extensionHandedOver else { return }
      ended.fulfill()
    }
    session.perform(.installExtension, "desktop")
    XCTAssertTrue(session.installing, "the slot is taken by the press and not by the child")
    session.perform(.update, "gdoc")
    wait(for: [ended], timeout: 20)

    let lists = try argv()
    XCTAssertTrue(lists.contains("update --desktop"), "the run the person asked for went")
    XCTAssertFalse(
      lists.contains("update"), "and the second press started no second update")
    XCTAssertFalse(session.installing, "the slot is free again once the child ended")
  }

  /// A press on the update glyph is owed an answer even where there was
  /// nothing to install. `gdoc update` asks GitHub on every run, and the
  /// one word for a machine already on the newest release is `up_to_date`:
  /// the glyph shows the check for a second and goes quiet by itself.
  ///
  /// A run that did replace the binary says nothing here. The version moves
  /// when `help --json` is asked again, and where the release carried a new
  /// app the bundle under this process changes and it reopens itself, which
  /// is M24's: testAChildOfThisAppsOwnHoldsTheReopenAndThenLetsItGo.
  func testAnUpdateThatInstalledNothingShowsTheCheckForASecond() throws {
    try writeTheRecord([])
    let session = made()
    answered(session)
    XCTAssertNil(session.updateChecked, "nothing has been pressed")
    XCTAssertEqual(
      session.model.header.update?.state, .live, "and the glyph is there to press")

    let ended = expectation(description: "the update run reached the panel")
    ended.assertForOverFulfill = false
    session.start { [weak session] in
      guard let session, session.updateChecked != nil else { return }
      ended.fulfill()
    }
    session.perform(.update, "gdoc")
    wait(for: [ended], timeout: 20)

    XCTAssertTrue(try argv().contains("update"), "the run went with no flag at all")
    XCTAssertFalse(session.installing, "the slot is free again")
    XCTAssertEqual(
      session.model.header.update?.state, .done,
      "the press is answered with the check")
    XCTAssertEqual(session.model.header.update?.tip, "This is the newest release")
    let armed = try XCTUnwrap(session.revertTimer, "and its end is on the one revert slot")
    XCTAssertLessThanOrEqual(
      armed.fireDate.timeIntervalSinceNow, Timing.doneFlash,
      "which is the copied link's own second")
  }

  /// The check the glyph shows lasts one second from the child's end, and
  /// the two polls that follow the run are two processes. So the facts carry
  /// it before those are asked, not after they answer: on a slow pair the
  /// one redraw the second is up would otherwise draw the ring of a run that
  /// had already ended, and the check would never reach the screen at all.
  func testTheCheckIsOnTheScreenBeforeTheTwoPollsComeBack() throws {
    try writeTheRecord([])
    let session = made()
    answered(session)

    var askedWhenTheCheckLanded: [String] = []
    let landed = expectation(description: "the check reached the facts")
    landed.assertForOverFulfill = false
    session.start { [weak session] in
      guard let session, session.facts.updateChecked != nil else { return }
      if askedWhenTheCheckLanded.isEmpty {
        askedWhenTheCheckLanded = (try? self.argv()) ?? []
      }
      landed.fulfill()
    }
    session.perform(.update, "gdoc")
    wait(for: [landed], timeout: 20)

    XCTAssertEqual(
      askedWhenTheCheckLanded.last, "update",
      "the facts carried the check before help --json and auth status were asked")
    XCTAssertEqual(
      session.model.header.update?.state, .done, "and the glyph draws it")
  }

  /// The banner after a sign-in is five seconds of the clock and no fact of
  /// its own, so the session schedules its own end. Nothing else would take
  /// it off: a refold that did not move redraws nothing, and the next fact
  /// may be minutes away.
  func testTheSignedInBannerTakesItselfOff() throws {
    try writeTheRecord([])
    let session = made()
    answered(session)

    let signedIn = expectation(description: "the sign-in reached the panel")
    signedIn.assertForOverFulfill = false
    session.start { [weak session] in
      guard let session, session.signedInAt != nil else { return }
      signedIn.fulfill()
    }
    session.perform(.signIn, nil)
    wait(for: [signedIn], timeout: 20)

    XCTAssertNotNil(
      session.model.header.banner,
      "the address is on the screen at the size a glance catches")
    let armed = try XCTUnwrap(session.bannerTimer, "and its end is on a timer of its own")
    XCTAssertGreaterThan(armed.fireDate.timeIntervalSinceNow, 0)
    XCTAssertLessThanOrEqual(
      armed.fireDate.timeIntervalSinceNow, SignedInBanner.life,
      "which is the banner's own five seconds")

    // And nothing a person does inside those five seconds takes it away: the
    // copied icon and the questions have the one revert slot, and the banner
    // is neither of those.
    session.revert(after: 1)
    XCTAssertTrue(armed.isValid, "a copied icon does not take the banner off early")
    XCTAssertEqual(
      armed.fireDate, try XCTUnwrap(session.bannerTimer).fireDate,
      "and the banner is still waiting on its own moment")
  }

  /// The facts are the polls', the stores' and the record's, and the model is
  /// built out of all three.
  func testTheFactsComeFromThePollsTheStoresAndTheRecord() throws {
    try writeTheRecord([
      line(kind: "start", command: "comments", call: 1, doc: Self.doc, wait: true),
      line(
        kind: "end", command: "comments", call: 1, doc: Self.doc,
        extra: "\"title\":\"Supplier register\",\"ok\":true"),
    ])
    let store = Store()
    let preferences = Preferences(store: store)
    preferences.hub = root.appendingPathComponent("hub").path

    let session = made(preferences: preferences)
    answered(session)

    XCTAssertEqual(session.facts.account, "someone@example.com", "`auth status` answered")
    XCTAssertEqual(session.facts.auth, .ok)
    XCTAssertEqual(session.facts.mark, "⁘", "the mark is the binary's setting")
    XCTAssertEqual(session.facts.versions.gdoc, "v2.14.0", "`help --json` answered")
    XCTAssertEqual(session.facts.versions.latestStable, "v2.14.1")
    XCTAssertEqual(session.facts.versions.app, "2.15.0", "the app's own version is the places'")
    XCTAssertEqual(session.facts.gdocPath, gdoc.path, "the binary this session resolved")
    XCTAssertEqual(
      session.facts.loginItem, .off, "the register says this app is not a login item")

    // And the record reached the model, through the tail and the four folds.
    let row = session.model.documents?.rows.first
    XCTAssertEqual(row?.doc, Self.doc)
    XCTAssertEqual(row?.title, "Supplier register")
    // A wait that ended seconds ago is a watch between two polls, which is
    // the stretch the menu bar calls live.
    XCTAssertEqual(session.bar, .live)
    XCTAssertEqual(
      store.object(forKey: Preferences.Key.lastAccount.rawValue) as? String,
      "someone@example.com", "the address the app saw is the app's own memory")
  }

  /// A redraw is one fold of the record, not three.
  ///
  /// The glyph, its ring and the panel were three computed properties, and
  /// each of them took the folds again and read the clock again: nine walks
  /// and sorts of every line the app holds for one redraw, on the main
  /// actor, once a second while a child of the app's own is up. `Drawn` is
  /// those four folds, and the question each of them puts to the kernel
  /// about a start's pid is what counts them here, so one screen must cost
  /// exactly one `Drawn` and nothing over it.
  ///
  /// Reading the clock once is the other half: the glyph and the panel are
  /// in front of a person at the same time, and two readings a moment apart
  /// could have let one of them call a watch live while the other had
  /// already let it go quiet.
  func testOneRedrawFoldsTheRecordOnce() throws {
    try writeTheRecord([
      line(kind: "start", command: "publish", call: 1, doc: nil),
      line(kind: "start", command: "comments", call: 1, doc: Self.doc, wait: true),
    ])

    let asked = Tally()
    let session = made(alive: { _ in asked.bump() })
    answered(session)

    asked.count = 0
    _ = session.screen
    let redraw = asked.count
    XCTAssertGreaterThan(redraw, 0, "the record holds starts whose pid is a question")

    asked.count = 0
    _ = Drawn(
      session.events, now: Date(), dismissed: session.ui.dismissed,
      alive: { _ in asked.bump() })
    XCTAssertEqual(redraw, asked.count, "one redraw is the three folds and nothing over them")
  }

  // MARK: what a press reaches

  /// The four that open, copy or dismiss, each through the one room that does
  /// it, and the
  /// strip of labels that goes after the first of them.
  func testEachIconLeavesThroughTheOneRoomThatDoesIt() throws {
    try writeTheRecord([line(kind: "start", command: "reply", call: 1, doc: Self.doc)])
    let store = Store()
    let desk = FakeDesk()
    let session = made(preferences: Preferences(store: store), desk: desk)
    answered(session)
    XCTAssertFalse(session.ui.stripSeen, "the labels have not been read yet")

    session.perform(.openDoc, Self.doc)
    XCTAssertEqual(
      desk.opened.map(\.absoluteString),
      ["https://docs.google.com/document/d/\(Self.doc)/edit"])
    XCTAssertTrue(session.ui.stripSeen, "the first click on an icon is the labels read")
    XCTAssertTrue(
      store.object(forKey: Preferences.Key.stripSeen.rawValue) as? Bool ?? false,
      "and a later launch does not show them again")

    session.perform(.copyLink, Self.doc)
    XCTAssertEqual(
      desk.copied, ["https://docs.google.com/document/d/\(Self.doc)/edit"],
      "the plain URL and nothing else")
    XCTAssertEqual(session.ui.copied, Self.doc, "the icon is green for the second")

    session.perform(.openChat, Self.doc)
    XCTAssertEqual(desk.activated, [Opener.claudeDesktop], "one activation, no prefill")

    // No note names this document, so there is nothing to open. The click
    // keeps why, which is what greys the icon.
    session.perform(.openNote, Self.doc)
    XCTAssertEqual(desk.opened.count, 1, "the note icon opened nothing")
    XCTAssertEqual(session.ui.noteFault[Self.doc], .notPaired, "and the click kept why")
  }

  /// A press on a document row opens the document, and leaves the four
  /// labels where they are.
  ///
  /// The row is the press since the fifth look (docs/v2/DECISIONS.md,
  /// 2026-10-28), through the one room that opens documents. The labels say
  /// what the icons are, and a click on a row is not a click on an icon: a
  /// person who has opened a document by clicking its row has read nothing
  /// about the four icons beside it, so the strip waits for a click on one.
  func testARowPressOpensTheDocumentAndLeavesTheStrip() throws {
    try writeTheRecord([line(kind: "start", command: "reply", call: 1, doc: Self.doc)])
    let store = Store()
    let desk = FakeDesk()
    let session = made(preferences: Preferences(store: store), desk: desk)
    answered(session)
    XCTAssertFalse(session.ui.stripSeen, "the labels have not been read yet")

    session.perform(.openRow, Self.doc)
    XCTAssertEqual(
      desk.opened.map(\.absoluteString),
      ["https://docs.google.com/document/d/\(Self.doc)/edit"],
      "the row opens the document through the one room that opens documents")
    XCTAssertFalse(session.ui.stripSeen, "a click on a row is not a click on an icon")
    XCTAssertNil(
      store.object(forKey: Preferences.Key.stripSeen.rawValue),
      "and nothing is written down about labels nobody has read")

    // The first click on an icon is still what retires them.
    session.perform(.openChat, Self.doc)
    XCTAssertTrue(session.ui.stripSeen, "the first click on an icon is the labels read")
  }

  /// A press on the firm's mark opens the picture's page, and moves nothing
  /// else.
  ///
  /// The mark sits in a section's label, so a click on it is a click on
  /// neither a row nor one of a row's icons: the four labels stay, the card's
  /// hint stays, and nothing is written down. The address is the one the
  /// facts carry, and the session hands it over without reading it.
  func testTheMarkPressOpensItsPageAndMovesNothingElse() throws {
    try writeTheRecord([line(kind: "start", command: "reply", call: 1, doc: Self.doc)])
    let store = Store()
    let desk = FakeDesk()
    let session = made(preferences: Preferences(store: store), desk: desk)
    answered(session)

    let page = "https://drive.google.com/file/d/1MaRkStAnDiN0PaGe0iD00000000/view"
    session.perform(.openMark, page)

    XCTAssertEqual(
      desk.opened.map(\.absoluteString), [page],
      "the mark opens the picture's own page and nothing else")
    XCTAssertFalse(session.ui.stripSeen, "a click on the mark is not a click on an icon")
    XCTAssertFalse(session.ui.cardHintSeen, "and it is not a click on a document row")
    XCTAssertNil(
      store.object(forKey: Preferences.Key.stripSeen.rawValue),
      "nothing is written about labels nobody has read")
    XCTAssertNil(
      store.object(forKey: Preferences.Key.cardHintSeen.rawValue),
      "and nothing about a hint nobody has put away")
  }

  /// The card's hint is taught once: the person's own first press of a
  /// document row puts it away, and it never comes back.
  ///
  /// The ninth key is the whole of what that costs (docs/v2/DECISIONS.md,
  /// 2026-10-28). A press of a NOW row or the stuck card is `.openDoc` and
  /// writes nothing: the hint says a click on a document row opens the
  /// document, and the rows of NOW have never needed telling.
  func testTheFirstRowPressPutsTheHintAway() throws {
    try writeTheRecord([line(kind: "start", command: "reply", call: 1, doc: Self.doc)])
    let store = Store()
    let session = made(preferences: Preferences(store: store))
    answered(session)
    XCTAssertFalse(
      session.ui.cardHintSeen, "the launch state reads the store, where nothing is written yet")

    // A NOW row is not a document row, and the hint is not about it.
    session.perform(.openDoc, Self.doc)
    XCTAssertFalse(session.ui.cardHintSeen, "a NOW row's press teaches nothing about a row")
    XCTAssertNil(
      store.object(forKey: Preferences.Key.cardHintSeen.rawValue),
      "and writes nothing down")

    session.perform(.openRow, Self.doc)
    XCTAssertTrue(session.ui.cardHintSeen, "the person's own first row press is the hint read")
    XCTAssertEqual(
      store.object(forKey: Preferences.Key.cardHintSeen.rawValue) as? Bool, true,
      "and the store keeps it, so a later launch shows no hint")

    // The next launch off the same store starts with it away.
    let again = made(preferences: Preferences(store: store))
    XCTAssertTrue(again.ui.cardHintSeen, "a hint read once stays read")
  }

  /// A fault on Open md greys the icon, and a greyed icon cannot be pressed,
  /// so the panel going away is the one thing that ends it. Without that, a
  /// note written after the click would leave the icon grey until the app
  /// quit, and the comment about the next click working would be about a
  /// click nobody can make.
  func testAFaultGoesWhenThePanelOpensAgain() throws {
    try writeTheRecord([line(kind: "start", command: "reply", call: 1, doc: Self.doc)])
    let session = made()
    answered(session)

    // No note names this document, so the click goes nowhere and the icon
    // says why.
    session.perform(.openNote, Self.doc)
    XCTAssertEqual(session.ui.noteFault[Self.doc], .notPaired)
    let greyed = try noteIcon(of: session)
    XCTAssertEqual(greyed.state, .off(NoteFault.notPaired.tip))
    XCTAssertFalse(
      ActionButton.isPressable(greyed.state), "so no second click can take it away")

    // The panel opening again is what takes it away.
    session.opened()
    XCTAssertTrue(session.ui.noteFault.isEmpty, "the fault died with the panel it was raised in")
  }

  /// A watch going quiet is three minutes passing and no new line, and the
  /// four folds the screen is built from are the only things that notice it.
  func testAQuietWatchRedrawsWithNoNewLine() throws {
    let started = try Event.decode(
      line(kind: "start", command: "comments", call: 1, doc: Self.doc, wait: true))
    let ended = try Event.decode(
      line(kind: "end", command: "comments", call: 1, doc: Self.doc, extra: "\"ok\":true"))
    let events = [started, ended]

    let between = Drawn(
      events, now: ended.at.addingTimeInterval(10), dismissed: [],
      alive: { _ in false })
    let quiet = Drawn(
      events, now: ended.at.addingTimeInterval(Live.gap + 1), dismissed: [],
      alive: { _ in false })

    XCTAssertTrue(between.sessions.allSatisfy(\.isOpen), "a wait between two polls is a watch")
    XCTAssertTrue(quiet.sessions.allSatisfy { !$0.isOpen }, "three minutes later it is not")
    XCTAssertNotEqual(quiet, between, "so the screen is drawn again with no line to read")
  }

  /// A minute passing over a hold or a live watch is a redraw with no line
  /// behind it: both rows say how long something has been going, and
  /// `Words.duration` counts in whole minutes. A screen with neither on it
  /// is drawn again only when the record or the facts move.
  func testAQuietMinuteRedrawsAHoldAndAWatchAndNothingElse() throws {
    let watching = try Event.decode(
      line(kind: "start", command: "comments", call: 1, doc: Self.doc, wait: true))
    let held = try Event.decode(
      line(
        kind: "held", command: "reply", call: nil, doc: Self.doc,
        extra: "\"hold\":\"h_ab12\",\"rule\":\"echo\",\"until\":\"\(stamp(inAnHour: true))\""))
    // Pinned to the middle of a UTC day, so the moments below are literals
    // like every other moment in this file.
    let over = try Event.decode(
      line(
        kind: "end", command: "reply", call: 4, doc: Self.doc, extra: "\"ok\":true",
        at: "2026-10-17T12:00:00Z"))

    for (what, events) in [("a watch", [watching]), ("a hold", [held])] {
      let now = Drawn(
        events, now: watching.at, dismissed: [], alive: { _ in true })
      let later = Drawn(
        events, now: watching.at.addingTimeInterval(60), dismissed: [],
        alive: { _ in true })
      XCTAssertNotEqual(later, now, "\(what) counts the minutes, so the screen is owed one")
    }

    // The minute is each stretch's own and not the clock's. A hold raised at
    // half past the minute shows its first minute half a minute later, and a
    // key cut at the wall clock noticed nothing until the minute after that.
    let raised = try Event.decode(
      line(
        kind: "held", command: "reply", call: nil, doc: Self.doc,
        extra: "\"hold\":\"h_cd34\",\"rule\":\"echo\",\"until\":\"2026-10-17T11:00:00Z\"",
        at: "2026-10-17T10:00:30Z"))
    let atTheWallMinute = Drawn(
      [raised], now: raised.at.addingTimeInterval(30), dismissed: [],
      alive: { _ in true })
    let atItsOwnMinute = Drawn(
      [raised], now: raised.at.addingTimeInterval(60), dismissed: [],
      alive: { _ in true })
    XCTAssertEqual(atTheWallMinute.counting, [0], "nothing is a minute old yet")
    XCTAssertEqual(atItsOwnMinute.counting, [1], "and a minute after it was raised, one is")
    XCTAssertNotEqual(
      atItsOwnMinute, atTheWallMinute, "so the screen is owed the redraw that says 1 min")

    // Nothing on this screen counts a stretch, so nothing moved.
    let still = Drawn(
      [over], now: over.at, dismissed: [], alive: { _ in false })
    let stillLater = Drawn(
      [over], now: over.at.addingTimeInterval(600), dismissed: [],
      alive: { _ in false })
    XCTAssertEqual(stillLater, still, "a finished day redraws on nothing")
    XCTAssertNil(still.counting)
  }

  /// The `hubPath` override is asked before Spotlight, and the finder is
  /// never asked about a document it already pairs.
  ///
  /// The override is a `defaults write` since the third look: there is no
  /// row that sets it, and a machine nobody has written one on has none.
  /// Where there is one, the notes under it are read at start, so the md
  /// icon of a document it pairs is live on the first draw.
  func testTheOverrideIsAskedBeforeSpotlight() throws {
    try writeTheRecord([
      line(kind: "end", command: "reply", call: 1, doc: Self.doc, extra: "\"ok\":true")
    ])
    let hub = root.appendingPathComponent("hub")
    try FileManager.default.createDirectory(
      at: hub.appendingPathComponent("notes"), withIntermediateDirectories: true)
    try note(pairing: Self.doc).write(
      to: hub.appendingPathComponent("notes/scope.md"), atomically: true, encoding: .utf8)

    let store = Store()
    let preferences = Preferences(store: store)
    preferences.hub = hub.path
    let search = StatedSearch()
    let session = made(preferences: preferences, noteSearch: search)
    answered(session)

    try waitUntil("the override was read") { session.facts.pairing[Self.doc] != nil }
    XCTAssertEqual(session.index.notes[Self.doc], "notes/scope.md")
    XCTAssertEqual(
      session.facts.pairing[Self.doc], hub.appendingPathComponent("notes/scope.md").path,
      "the pairing is the absolute path, whichever route found it")
    XCTAssertEqual(try noteIcon(of: session).state, .live, "so Open md works")
    XCTAssertEqual(search.asked, [], "and Spotlight was never asked about it")

    XCTAssertFalse(
      SettingsRowKind.allCases.contains { $0.label == "Hub" },
      "and no row offers to choose a folder any more")
  }

  /// With no override the finder answers, and the md icon is grey only
  /// while no note names the document.
  func testTheMdIconIsGreyOnlyWhenNoNoteNamesTheDocument() throws {
    try writeTheRecord([
      line(kind: "end", command: "reply", call: 1, doc: Self.doc, extra: "\"ok\":true")
    ])
    let search = StatedSearch()
    let paired = note(pairing: Self.doc)
    let session = made(noteSearch: search, noteRead: { _ in paired })
    answered(session)

    try waitUntil("the finder answered") { session.notes[Self.doc] != nil }
    XCTAssertEqual(search.asked, [Self.doc], "the one document on the screen, once")
    XCTAssertEqual(try noteIcon(of: session).state, .off(Tips.noNote), "nothing names it yet")

    // Spotlight finds the note, says so, and the next draw is a live icon.
    search.willAnswer([Self.doc: ["/notes/scope.md"]])
    let told = expectation(description: "the finder threw its answer away")
    Task { @MainActor in
      await session.finder.forget(Self.doc)
      told.fulfill()
    }
    wait(for: [told], timeout: 20)
    try waitUntil("the second answer landed") { session.facts.pairing[Self.doc] != nil }
    XCTAssertEqual(session.facts.pairing[Self.doc], "/notes/scope.md")
    XCTAssertEqual(try noteIcon(of: session).state, .live)
    XCTAssertEqual(search.asked, [Self.doc, Self.doc], "the notice is what made it ask again")
  }

  /// Two notes naming one document is the icon's own sentence, not "no note
  /// is paired with this document". The finder will not pick between them,
  /// and the panel has to say which of the two greys it is: the pairing
  /// carries the notes that were found, so the fault travels beside it.
  func testTwoNotesNamingOneDocumentSaySoOnTheIcon() throws {
    try writeTheRecord([
      line(kind: "end", command: "reply", call: 1, doc: Self.doc, extra: "\"ok\":true")
    ])
    let search = StatedSearch([Self.doc: ["/notes/a.md", "/notes/b.md"]])
    let paired = note(pairing: Self.doc)
    let session = made(noteSearch: search, noteRead: { _ in paired })
    answered(session)

    try waitUntil("the facts took the fault") {
      session.facts.noteFaults[Self.doc] != nil
    }
    XCTAssertEqual(session.facts.noteFaults[Self.doc], .twoNotes)
    XCTAssertNil(session.facts.pairing[Self.doc], "neither of the two is picked")
    XCTAssertEqual(
      try noteIcon(of: session).state, .off(Tips.twoNotes),
      "and the icon says which grey it is")
  }

  /// A query that did not answer inside the finder's ceiling is no answer,
  /// and the draw that asked gives up rather than writing "no note" down.
  ///
  /// Writing it down would be the one wrong answer nothing could take back:
  /// a query that hit the ceiling was stopped, so no Spotlight notice is
  /// coming for that document, and a notice is the only thing that frees a
  /// cached answer. The icon would stay grey until the app quit over a note
  /// sitting in the vault. A cold index right after login is where this
  /// happens.
  func testADrawThatHitTheCeilingAsksAgainOnTheNext() throws {
    try writeTheRecord([
      line(kind: "end", command: "reply", call: 1, doc: Self.doc, extra: "\"ok\":true")
    ])
    let search = StatedSearch()
    search.willSayNothing(about: [Self.doc])
    let paired = note(pairing: Self.doc)
    let session = made(noteSearch: search, noteRead: { _ in paired })
    answered(session)

    try waitUntil("the first lookup gave up") {
      search.asked == [Self.doc] && !session.looking.contains(Self.doc)
    }
    XCTAssertNil(session.notes[Self.doc], "the ceiling wrote nothing down")
    XCTAssertEqual(
      try noteIcon(of: session).state, .off(Tips.noNote), "the icon is grey for this draw")

    // And opening the panel asks again, which is the whole of the
    // difference: there is no notice coming for a query that was stopped.
    search.willAnswer([Self.doc: ["/notes/scope.md"]])
    session.opened()
    try waitUntil("the second lookup landed") { session.facts.pairing[Self.doc] != nil }
    XCTAssertEqual(search.asked, [Self.doc, Self.doc], "the draw is what asked again")
    XCTAssertEqual(try noteIcon(of: session).state, .live)
  }

  /// The clock does not ask again about a document whose query hit the
  /// ceiling, for a minute.
  ///
  /// Nothing is written into `notes` for one of those, so every refold
  /// wants it: the five-second tick and every batch of new lines. A
  /// Spotlight that is rebuilding its index answers nothing for hours, and
  /// that would be a two-second query every five seconds for the life of
  /// the app. The backoff is what makes the recovery above a redraw
  /// somebody asked for rather than a loop.
  func testACeilingIsNotAskedAgainUntilTheBackoffIsUp() throws {
    try writeTheRecord([
      line(kind: "end", command: "reply", call: 1, doc: Self.doc, extra: "\"ok\":true")
    ])
    let search = StatedSearch()
    search.willSayNothing(about: [Self.doc])
    let paired = note(pairing: Self.doc)
    let session = made(noteSearch: search, noteRead: { _ in paired })
    answered(session)

    try waitUntil("the first lookup gave up") {
      search.asked == [Self.doc] && !session.looking.contains(Self.doc)
    }
    XCTAssertNotNil(session.ceilings[Self.doc], "the moment it gave up is what is kept")

    // What a tick does, and what a new line does, twice over.
    search.willAnswer([Self.doc: ["/notes/scope.md"]])
    session.lookUpTheNotes()
    session.lookUpTheNotes()
    XCTAssertEqual(search.asked, [Self.doc], "the clock leaves that document alone")
    XCTAssertFalse(session.looking.contains(Self.doc), "and nothing is out for it")

    // A minute later it is asked again, which is the icon recovering on its
    // own for somebody who never opens the panel.
    session.ceilings[Self.doc] = Date(timeIntervalSinceNow: -61)
    session.lookUpTheNotes()
    try waitUntil("the lookup after the backoff landed") {
      session.facts.pairing[Self.doc] != nil
    }
    XCTAssertEqual(search.asked, [Self.doc, Self.doc], "once the minute is up")
    XCTAssertNil(session.ceilings[Self.doc], "and the answer took the backoff away")
  }

  /// A lookup that was out when the notice arrived is dropped.
  ///
  /// The query walks an index off the main actor, so an answer about the
  /// files as they were can land after the notice that says they changed.
  /// Writing it back would put the stale answer in with nothing left to
  /// take it away again, the same bug an older hub scan replacing a newer
  /// one is: testAnOlderHubScanDoesNotReplaceANewerOne.
  func testANoticeDropsTheLookupItInterrupted() throws {
    let session = made()
    answered(session)

    session.noteAsks[Self.doc] = 1
    session.looking.insert(Self.doc)
    session.noteWentStale(Self.doc)
    XCTAssertEqual(session.noteAsks[Self.doc], 2, "the notice asked for a fresher answer")

    session.tookTheNote(Self.doc, 1, .success("/notes/stale.md"))
    XCTAssertNil(session.notes[Self.doc], "the older lookup's answer is dropped")

    session.tookTheNote(Self.doc, 2, .success("/notes/fresh.md"))
    XCTAssertEqual(try session.notes[Self.doc]?.get(), "/notes/fresh.md")
  }

  /// The watcher follows the folder the override names. A second override
  /// is a different vault, and a watcher left on the first one would
  /// rescan the folder nobody named on its next file event: that scan is
  /// the newest one, so the pairing would flip back to the old notes.
  func testASecondHubIsTheOneTheWatcherWatches() throws {
    let first = root.appendingPathComponent("first")
    let second = root.appendingPathComponent("second")
    for folder in [first, second] {
      try FileManager.default.createDirectory(
        at: folder.appendingPathComponent("notes"), withIntermediateDirectories: true)
    }
    try note(pairing: Self.doc).write(
      to: first.appendingPathComponent("notes/first.md"), atomically: true, encoding: .utf8)

    let session = made()
    answered(session)

    session.preferences.hub = first.path
    session.readTheHub()
    try waitUntil("the first hub was read") { session.index.notes[Self.doc] != nil }
    XCTAssertEqual(session.hubWatched, first.path, "the watcher is on the folder named")

    session.preferences.hub = second.path
    session.readTheHub()
    XCTAssertEqual(session.hubWatched, second.path, "and moves with the second")

    // A note written into the new folder reaches the pairing, which is the
    // whole of what watching it is for.
    let path = second.appendingPathComponent("notes/second.md")
    let written = note(pairing: Self.other)
    let landed = expectation(description: "the new folder's own note was read")
    Task { @MainActor in
      // One write a second, which is longer than the watcher's own
      // debounce: a write every quarter second would keep putting the one
      // signal off, and the test would be measuring the debounce.
      for _ in 0..<20 where session.index.notes[Self.other] == nil {
        try? written.write(to: path, atomically: true, encoding: .utf8)
        try? await Task.sleep(nanoseconds: 1_000_000_000)
      }
      if session.index.notes[Self.other] != nil { landed.fulfill() }
    }
    wait(for: [landed], timeout: 30)
    XCTAssertEqual(session.index.notes[Self.other], "notes/second.md")
    XCTAssertNil(
      session.index.notes[Self.doc],
      "and the old vault's own notes are not in the index any more")
  }

  /// A note whose front matter pairs it with one document and says nothing
  /// else.
  private func note(pairing doc: String) -> String {
    """
    ---
    gdoc:
      schema: 1
      document_id: \(doc)
    ---

    # Scope
    """
  }

  /// Wait for something the session does off the main thread and comes back
  /// to it with. The scan of the hub is the one of them a press starts.
  private func waitUntil(
    _ what: String, _ settled: @escaping @MainActor () -> Bool
  ) throws {
    let done = expectation(description: what)
    Task { @MainActor in
      for _ in 0..<200 where !settled() {
        try? await Task.sleep(nanoseconds: 50_000_000)
      }
      if settled() { done.fulfill() }
    }
    wait(for: [done], timeout: 20)
  }

  /// The Open md icon of the one document row the panel draws.
  private func noteIcon(of session: Session) throws -> ActionModel {
    let row = try XCTUnwrap(session.model.documents?.rows.first)
    return try XCTUnwrap(row.actions.first { $0.kind == .openNote })
  }

  /// A hold asks before its card goes and a failed write does not. Both end
  /// up in the one store that writes anything down.
  func testAHoldIsAskedAboutAndAFailedWriteIsNot() throws {
    try writeTheRecord([
      line(
        kind: "held", command: "reply", call: nil, doc: Self.doc,
        extra: "\"hold\":\"h_ab12\",\"rule\":\"echo\",\"until\":\"\(stamp(inAnHour: true))\""),
      line(kind: "start", command: "propose", call: 7, doc: Self.doc),
      line(
        kind: "end", command: "propose", call: 7, doc: Self.doc, extra: "\"ok\":false"),
    ])
    let store = Store()
    let session = made(preferences: Preferences(store: store))
    answered(session)

    session.perform(.dismiss, "h_ab12")
    XCTAssertEqual(
      session.ui.question, .dismissHold("h_ab12"), "a hold asks first")
    XCTAssertFalse(session.ui.dismissed.contains("h_ab12"), "and nothing is hidden yet")

    session.perform(.dismiss, "h_ab12")
    XCTAssertNil(session.ui.question, "the second press is the answer")
    XCTAssertTrue(session.ui.dismissed.contains("h_ab12"))

    // The failed write's own key, which is `run|call`, goes at once.
    let key = "\(Self.run)|7"
    session.perform(.dismiss, key)
    XCTAssertNil(session.ui.question, "a write that failed is over, so there is no question")
    XCTAssertTrue(session.ui.dismissed.contains(key))
    let written = store.object(forKey: Preferences.Key.dismissed.rawValue)
    XCTAssertEqual(
      Set((written as? [String: Double] ?? [:]).keys), ["h_ab12", key],
      "both are in the one store that writes anything down")
  }

  /// The copied check and the question share one redraw, and they are not
  /// the same length. A Copy link pressed inside the question's three
  /// seconds may not move that redraw forward to its own second: nothing
  /// would then be scheduled for the moment the question ran out, and the
  /// expired question would sit on the screen until some fact moved, where a
  /// press on it asks again instead of acting.
  func testACopiedCheckDoesNotCutTheQuestionsRedrawShort() throws {
    let now = Date()
    XCTAssertTrue(
      Session.takesTheSlot(waiting: nil, due: now.addingTimeInterval(Timing.doneFlash)),
      "an empty slot is taken")
    XCTAssertFalse(
      Session.takesTheSlot(
        waiting: now.addingTimeInterval(Timing.checkFlash),
        due: now.addingTimeInterval(Timing.doneFlash)),
      "the banner's five seconds outlive the copied second, so the banner keeps the slot")
    XCTAssertTrue(
      Session.takesTheSlot(
        waiting: now.addingTimeInterval(Timing.doneFlash),
        due: now.addingTimeInterval(Timing.checkFlash)),
      "and a banner raised inside a copied second takes it, because it is the later moment")
  }

  /// Sign out asks first, and the answer runs `auth logout` and asks gdoc
  /// again. Every argv the session ran is the log the fake gdoc kept.
  func testSigningOutAsksFirstAndThenRunsTheLogout() throws {
    try writeTheRecord([line(kind: "start", command: "read", call: 1, doc: Self.doc)])
    let session = made()
    answered(session)
    XCTAssertEqual(try argv(), ["help --json", "auth status"], "the open asked the two")

    session.perform(.signOut, nil)
    XCTAssertEqual(session.ui.question, .signOut, "the first press is the question")
    XCTAssertEqual(try argv(), ["help --json", "auth status"], "and it ran nothing")

    let out = expectation(description: "the logout ran and gdoc was asked again")
    out.assertForOverFulfill = false
    session.start { [weak self] in
      guard let self, let ran = try? self.argv() else { return }
      if ran.contains("auth logout"), ran.filter({ $0 == "auth status" }).count > 1 {
        out.fulfill()
      }
    }
    session.perform(.signOut, nil)
    wait(for: [out], timeout: 20)
    XCTAssertNil(session.ui.question, "the question went with the press")

    let ran = try argv()
    XCTAssertTrue(
      ran.allSatisfy { Call.allCases.map { $0.arguments.joined(separator: " ") }.contains($0) },
      "every argv the session ran is one of the six: \(ran)")
  }

  /// The question waits for its answer. Nail pressed Sign out on 2026-10-18,
  /// read the sentence under "Sign out?", pressed again and the icon was back
  /// under the pointer: three seconds had passed and the question had
  /// reverted, so the press asked again and `auth logout` never ran. A
  /// question goes with Keep, with the confirm, or with the panel closing.
  func testTheQuestionWaitsForItsAnswerAndGoesWithThePanel() throws {
    try writeTheRecord([line(kind: "start", command: "read", call: 1, doc: Self.doc)])
    var now = Date()
    let session = made(clock: { now })
    answered(session)

    session.perform(.signOut, nil)
    XCTAssertEqual(session.ui.question, .signOut)
    now = now.addingTimeInterval(45)
    XCTAssertTrue(
      session.ui.isAsking(.signOut, now: now), "forty-five seconds later it is still asked")

    session.closed()
    XCTAssertNil(session.ui.question, "the panel closing is the Keep nobody pressed")
    XCTAssertEqual(try argv(), ["help --json", "auth status"], "and nothing ran")

    session.perform(.signOut, nil)
    now = now.addingTimeInterval(45)
    let out = expectation(description: "the second press, forty-five seconds later, ran the logout")
    out.assertForOverFulfill = false
    session.start { [weak self] in
      guard let self, let ran = try? self.argv() else { return }
      if ran.contains("auth logout") { out.fulfill() }
    }
    session.perform(.signOut, nil)
    wait(for: [out], timeout: 20)
    XCTAssertNil(session.ui.question)
  }

  /// Quit is the shell's own closure, because the run loop is the shell's.
  func testQuitIsTheShellsClosure() throws {
    try writeTheRecord([])
    var quits = 0
    let session = made(quit: { quits += 1 })
    answered(session)

    session.perform(.quit, nil)
    XCTAssertEqual(quits, 1)
  }

  /// The switch records the person's answer and the register is told once.
  /// Starting the session tells it nothing: no run of this app registers a
  /// login item by itself, which is the 2026-10-18 decision.
  func testTheSwitchRecordsThePersonsAnswer() throws {
    try writeTheRecord([])
    let store = Store()
    let register = FakeRegister()
    let session = made(preferences: Preferences(store: store), loginService: register)
    answered(session)
    XCTAssertEqual(register.registrations, 0, "a run registers nothing")
    XCTAssertNil(
      store.object(forKey: Preferences.Key.loginItemChoice.rawValue),
      "and answers nothing for the person")

    session.perform(.toggleLoginItem, nil)
    refreshed(session)
    XCTAssertEqual(
      store.object(forKey: Preferences.Key.loginItemChoice.rawValue) as? String, "on")
    XCTAssertEqual(register.registrations, 1, "the switch is the one that registers")
    XCTAssertEqual(session.facts.loginItem, .on, "and the row shows what macOS says")

    session.perform(.toggleLoginItem, nil)
    refreshed(session)
    XCTAssertEqual(
      store.object(forKey: Preferences.Key.loginItemChoice.rawValue) as? String, "off")
    XCTAssertEqual(register.unregistrations, 1)

    session.perform(.openLoginItems, nil)
    XCTAssertEqual(register.settingsOpened, 1, "the approval row opens the pane")
  }

  /// macOS took the registration and is waiting for the person to allow it
  /// in System Settings. The switch is drawn off, because the item is not
  /// running yet, and the one thing a press may do is open that pane:
  /// registering again is a second notice in somebody's menu bar for a
  /// choice this app already wrote down, which LoginItem's own page says is
  /// never retried.
  func testTheSwitchInTheApprovalStateOpensThePaneAndRegistersNothingAgain() throws {
    try writeTheRecord([])
    let register = FakeRegister(after: .requiresApproval)
    let session = made(loginService: register)
    answered(session)

    session.perform(.toggleLoginItem, nil)
    refreshed(session)
    XCTAssertEqual(session.facts.loginItem, .needsApproval, "the row macOS left")
    XCTAssertEqual(register.registrations, 1, "the switch registered once")

    session.perform(.toggleLoginItem, nil)

    XCTAssertEqual(register.registrations, 1, "the press registered nothing again")
    XCTAssertEqual(register.unregistrations, 0, "and unregistered nothing")
    XCTAssertEqual(register.settingsOpened, 1, "it opened the pane that allows it")
  }

  /// A redraw is decided by the folds the screen is drawn from, so a line
  /// only one of them reads still moves the panel. A `step` is that line:
  /// nothing but the ring and the job card changes when one arrives, and a
  /// publish whose step never reached the panel would sit at its first part
  /// until the run ended.
  func testAStepLineRedrawsTheJobCard() throws {
    let start = line(kind: "start", command: "publish", call: 1, doc: Self.doc)
    try writeTheRecord([start])

    let session = made(alive: { _ in true })
    let settled = expectation(description: "the start reached the panel")
    settled.assertForOverFulfill = false
    let moved = expectation(description: "the step reached the panel")
    moved.assertForOverFulfill = false
    session.start { [weak session] in
      guard let session else { return }
      let jobs = Now.fold(
        session.events, now: Date(), dismissed: [], alive: { _ in true }
      ).jobs
      if jobs.count == 1 { settled.fulfill() }
      if jobs.compactMap(\.step) == ["upload"] { moved.fulfill() }
    }
    wait(for: [settled], timeout: 20)

    try writeTheRecord([
      start,
      line(
        kind: "step", command: "publish", call: 1, doc: Self.doc,
        extra: "\"step\":\"upload\",\"index\":2,\"count\":4"),
    ])
    wait(for: [moved], timeout: 20)
  }

  /// The sign-in's countdown is read off an actor, so the answer comes back
  /// after a hop. A sign-in that finished during that hop has already put
  /// the row away, and writing the phase it asked about back would put a
  /// frozen countdown and a dead Cancel on the Account row that nothing
  /// clears: the next read returns at the `signingIn` guard for ever.
  func testAPhaseThatCameBackAfterTheSignInEndedIsDropped() throws {
    let session = made()
    answered(session)
    let read = SignInPhase(url: "https://accounts.google.com/o/oauth2/auth", secondsLeft: 142)

    session.signingIn = true
    XCTAssertTrue(session.tookThePhase(read), "a sign-in is running, so the read is the row")
    XCTAssertEqual(session.phase, read)

    // What the sign-in's own closure does when the child comes back.
    session.signingIn = false
    session.phase = nil

    XCTAssertFalse(
      session.tookThePhase(read), "the sign-in it asked about is over, so the read is dropped")
    XCTAssertNil(session.phase, "and the Account row keeps no countdown")
  }

  /// A rescan the watcher on the folder nobody chose any more asks for is
  /// dropped.
  ///
  /// The watcher's callback arrives on a queue of its own and hops to the
  /// main actor, so one that fired just before the Hub row was answered
  /// lands after the new hub was already read. Its walk would be the newest
  /// one asked for, and the pairing would flip back to the old vault's
  /// notes. The folder it names decides: it is not the one being watched,
  /// so nobody wants its scan.
  func testAScanFromTheWatcherOnTheOldHubIsDropped() throws {
    try writeTheRecord([])
    let session = made()
    session.hubWatched = "/second"

    session.rescanIfWatched("/first")
    XCTAssertEqual(session.hubScans, 0, "the folder nobody chose starts no walk")

    session.rescanIfWatched("/second")
    XCTAssertEqual(session.hubScans, 1, "and the folder being watched starts one")
  }

  /// Spotlight is asked nothing at all while the first walk of the override
  /// is still out, and the walk landing is what asks about the rest.
  ///
  /// This is the race `testTheOverrideIsAskedBeforeSpotlight` was failing
  /// on a slow runner: both routes started together and the override only
  /// won when its walk finished first. Here the walk is held on a gate of
  /// this test's own, so the order is the one the rule names and not the one
  /// the machine happened to run.
  func testSpotlightWaitsWhileTheOverrideIsBeingRead() throws {
    try writeTheRecord([
      line(kind: "end", command: "reply", call: 1, doc: Self.doc, extra: "\"ok\":true")
    ])
    let store = Store()
    let preferences = Preferences(store: store)
    preferences.hub = "/hub"
    let search = StatedSearch([Self.doc: ["/notes/spotlight.md"]])
    let held = HeldScan(HubIndex(hub: "/hub", notes: [Self.doc: "notes/scope.md"]))
    let session = made(preferences: preferences, noteSearch: search, hubScan: held.scan)

    let drew = expectation(description: "the record reached the panel")
    drew.assertForOverFulfill = false
    session.start { if session.drawn.documents != nil { drew.fulfill() } }
    wait(for: [drew], timeout: 20)
    XCTAssertNotNil(session.hubFirstScan, "the first walk of the hub is out")
    XCTAssertEqual(search.asked, [], "so Spotlight has been asked nothing")

    held.open()
    try waitUntil("the walk landed") { session.facts.pairing[Self.doc] != nil }
    XCTAssertNil(session.hubFirstScan, "the gate the walk held is open")
    XCTAssertEqual(
      session.facts.pairing[Self.doc], "/hub/notes/scope.md",
      "the pairing is the override's own note")
    XCTAssertEqual(search.asked, [], "and Spotlight was never asked about it")
  }

  /// The gate is the first walk of the folder and no later one. A watcher
  /// rescan holds nothing back, and a walk cancelled by a newer one lets the
  /// gate go rather than holding Spotlight off for the life of the app.
  func testOnlyTheFirstScanHoldsSpotlight() throws {
    try writeTheRecord([])
    let session = made()
    session.preferences.hub = "/hub"

    session.readTheHub()
    XCTAssertEqual(session.hubFirstScan, 1, "the first walk of this folder holds the gate")

    session.rescanIfWatched("/hub")
    XCTAssertEqual(session.hubWatched, "/hub", "the watcher is on the folder named")
    XCTAssertEqual(session.hubScans, 2, "a file event asked for a second walk")
    XCTAssertNil(
      session.hubFirstScan,
      "which cancelled the first, so nothing is left to wait for")

    // And the walk that was cancelled landing later takes nothing back.
    session.tookTheWalk(1, "/hub", HubIndex(hub: "/hub", notes: [:], stop: .cancelled))
    XCTAssertNil(session.hubFirstScan, "the gate stays open")
  }

  /// A folder larger than the walk reads is set aside: no pairing, no
  /// watcher over it, and the finder answers every document the way it does
  /// on a machine with no override at all.
  func testAHubOverTheCeilingIsSetAsideAndSpotlightAnswers() throws {
    try writeTheRecord([
      line(kind: "end", command: "reply", call: 1, doc: Self.doc, extra: "\"ok\":true")
    ])
    let store = Store()
    let preferences = Preferences(store: store)
    preferences.hub = "/hub"
    let search = StatedSearch([Self.doc: ["/notes/spotlight.md"]])
    let over = HubIndex(hub: "/hub", notes: [:], stop: .overTheCeiling)
    let paired = note(pairing: Self.doc)
    let session = made(
      preferences: preferences, noteSearch: search, noteRead: { _ in paired },
      hubScan: { _ in over })
    answered(session)

    try waitUntil("the walk came back over the ceiling") {
      session.hubFirstScan == nil && session.index.notes.isEmpty
    }
    XCTAssertEqual(session.index, HubIndex.empty, "the override pairs nothing")
    XCTAssertNil(session.hubWatched, "and no watcher is left on that folder")
    XCTAssertNil(session.hubWatcher)

    try waitUntil("the finder answered instead") { session.facts.pairing[Self.doc] != nil }
    XCTAssertEqual(search.asked, [Self.doc], "the one document on the screen, once")
    XCTAssertEqual(session.facts.pairing[Self.doc], "/notes/spotlight.md")
  }

  /// One walk of the hub at a time. A newer one cancels the walk before it,
  /// and the cancelled walk's answer is nobody's.
  ///
  /// Every burst of file events in a vault asked for its own full walk, and
  /// nothing stopped the one before it: a vault syncing for a minute left a
  /// pile of walks on the utility queue, each of them reading files whose
  /// answer was already owed to a newer walk.
  func testOnlyOneWalkOfTheHubRunsAtATime() throws {
    try writeTheRecord([])
    let first = HeldScan(HubIndex(hub: "/hub", notes: [Self.doc: "first.md"]))
    let session = made(hubScan: first.scan)
    session.preferences.hub = "/hub"

    session.readTheHub()
    try waitUntil("the first walk started") { first.started }

    session.rescanIfWatched("/hub")
    XCTAssertEqual(session.hubScans, 2, "a second walk was asked for")
    first.open()
    try waitUntil("the first walk let go") { first.wasCancelled }
    XCTAssertTrue(first.wasCancelled, "the walk before it was asked to stop")
    XCTAssertEqual(
      session.index, HubIndex.empty, "and its pairing never reached the session")

    // Which is the rule stated over the one function that takes an answer.
    XCTAssertFalse(
      session.tookTheScan(2, HubIndex(hub: "/hub", notes: [Self.doc: "x.md"], stop: .cancelled)),
      "a walk that let go carries no pairing, whatever its number says")
    XCTAssertEqual(session.index, HubIndex.empty)
  }

  /// A hub scan that comes back after a newer one is dropped.
  ///
  /// Every burst of file events in the hub starts its own walk, off the main
  /// thread, and they come back in whatever order they finish in. The walk
  /// of a big vault that started first may land last, and the pairing it
  /// carries is the hub as it was: Open md would then open yesterday's note
  /// until some later file event.
  func testAnOlderHubScanDoesNotReplaceANewerOne() throws {
    try writeTheRecord([])
    let session = made()

    let old = HubIndex(hub: "/hub", notes: [Self.doc: "old.md"])
    let new = HubIndex(hub: "/hub", notes: [Self.doc: "new.md"])
    // Two walks asked for, the second after the first.
    session.hubScans = 2

    XCTAssertTrue(session.tookTheScan(2, new), "the newest walk is the one that stands")
    XCTAssertEqual(session.index.notes, new.notes)

    XCTAssertFalse(
      session.tookTheScan(1, old), "the older walk came back last and is dropped")
    XCTAssertEqual(session.index.notes, new.notes, "so the pairing is still the newer one")

    XCTAssertFalse(
      session.tookTheScan(2, new), "a walk that found what is already there redraws nothing")
  }

  /// The last icon on a row pins the document and takes the pin back, and
  /// it retires the labels like the other three.
  ///
  /// Pin was a word in the row's menu until the third look, and Nail never
  /// found the menu. The strip of labels now has four words, so the press
  /// that reads the last one is the press that takes them away.
  func testThePinIconPinsTheDocumentAndRetiresTheStrip() throws {
    try writeTheRecord([line(kind: "start", command: "reply", call: 1, doc: Self.doc)])
    let store = Store()
    let session = made(preferences: Preferences(store: store))
    answered(session)
    XCTAssertFalse(session.ui.stripSeen, "the labels have not been read yet")

    session.perform(.pinDoc, Self.doc)
    XCTAssertEqual(session.ui.pinned.map(\.doc), [Self.doc], "the pin is written down")
    XCTAssertTrue(session.ui.stripSeen, "and the pin is an icon, so the labels go")
    XCTAssertTrue(
      store.object(forKey: Preferences.Key.stripSeen.rawValue) as? Bool ?? false,
      "once and for every later launch")

    session.perform(.unpinDoc, Self.doc)
    XCTAssertTrue(session.ui.pinned.isEmpty, "and the filled tack takes it back")
  }

  // MARK: a session over this test's own machine

  private func made(
    preferences: Preferences = Preferences(store: Store()),
    desk: Desk = FakeDesk(),
    loginService: LoginService = FakeRegister(),
    noteSearch: NoteSearch = StatedSearch(),
    noteRead: @escaping @Sendable (String) -> String = { _ in "" },
    hubScan: @escaping @Sendable (String) -> HubIndex = { HubIndex.scan(hub: $0) },
    alive: @escaping (Int32) -> Bool = { _ in false },
    quit: @escaping () -> Void = {},
    clock: @escaping () -> Date = Date.init
  ) -> Session {
    let made = Session(
      places: Places(
        records: records, bundle: nil, version: "2.15.0", identifier: "", home: root.path,
        arguments: []),
      calendar: Calendar.current,
      preferences: preferences,
      desk: desk,
      loginService: loginService,
      noteSearch: noteSearch,
      noteRead: noteRead,
      hubScan: hubScan,
      environment: ["HOME": root.path, "PATH": "/usr/bin:/bin"],
      alive: alive,
      clock: clock,
      quit: quit)
    session = made
    return made
  }

  /// Take the facts again and wait for them, which is what a press does on
  /// its own in a Task nothing here can wait on.
  private func refreshed(_ session: Session) {
    let done = expectation(description: "the facts were taken again")
    Task { @MainActor in
      _ = await session.recompose()
      done.fulfill()
    }
    wait(for: [done], timeout: 20)
  }

  /// Start the session and wait for the two polls to answer, which is the
  /// state every test here begins in.
  private func answered(_ session: Session) {
    let settled = expectation(description: "gdoc answered both questions")
    settled.assertForOverFulfill = false
    session.start { [weak session] in
      guard let session else { return }
      if session.facts.account != nil, session.facts.versions.gdoc != nil { settled.fulfill() }
    }
    session.opened()
    wait(for: [settled], timeout: 20)
  }

  // MARK: the record this test wrote

  private var records: URL { root.appendingPathComponent("presence") }
  private var gdoc: URL { root.appendingPathComponent(".local/bin/gdoc") }
  private var log: URL { root.appendingPathComponent("argv.log") }

  private static let run = "9f1cb3a7d2e45608"

  private func writeTheRecord(_ lines: [String]) throws {
    let name = MonthFile.name(for: Date())
    let text = lines.isEmpty ? "" : lines.joined(separator: "\n") + "\n"
    try Data(text.utf8).write(to: records.appendingPathComponent(name))
  }

  /// One line of the record, at this moment, so the folds keep it.
  private func line(
    kind: String, command: String, call: Int64?, doc: String?, wait: Bool = false,
    extra: String = "", at: String? = nil
  ) -> String {
    var fields = [
      "\"v\":1", "\"at\":\"\(at ?? stamp())\"", "\"gdoc\":\"v2.14.0\"", "\"pid\":4242",
      "\"run\":\"\(Self.run)\"", "\"event\":\"\(kind)\"", "\"command\":\"\(command)\"",
      "\"via\":\"terminal\"",
    ]
    if let call { fields.append("\"call\":\(call)") }
    if let doc { fields.append("\"doc\":\"\(doc)\"") }
    if wait { fields.append("\"wait\":true") }
    if !extra.isEmpty { fields.append(extra) }
    return "{" + fields.joined(separator: ",") + "}"
  }

  private func stamp(inAnHour: Bool = false) -> String {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    formatter.timeZone = TimeZone(identifier: "UTC")
    return formatter.string(from: Date().addingTimeInterval(inAnHour ? 3600 : -5))
  }

  // MARK: the gdoc this test wrote

  /// A gdoc of this file's own: it writes down the argv it was handed and
  /// answers the three lists these tests ask for. The real binary is never
  /// run, because a test that ran it would read somebody's token.
  private func writeTheFakeGdoc() throws {
    let script = """
      #!/bin/sh
      printf '%s\\n' "$*" >> "\(log.path)"
      case "$*" in
      "help --json")
        printf '{"ok":true,"data":{"settings":{"mark":"⁘"},\
      "update":{"installed":"v2.14.0","latest_stable":"v2.14.1"}}}\\n'
        ;;
      "auth status")
        printf '{"ok":true,"data":{"token_present":true,"expired":false,\
      "missing_scopes":[],"account":"someone@example.com"}}\\n'
        ;;
      "auth logout")
        printf '{"ok":true,"data":{"removed":true}}\\n'
        ;;
      "auth login")
        printf 'Open this link in your browser to sign in:\\n' >&2
        printf 'https://accounts.google.com/o/oauth2/auth?client_id=x\\n' >&2
        printf '{"ok":true,"data":{"account":"someone@example.com"}}\\n'
        ;;
      "update")
        printf '{"ok":true,"data":{"action":"up_to_date","installed":"v2.14.0"}}\\n'
        ;;
      "update --desktop")
        printf 'not really an extension' > "$(dirname "$0")/gdoc.mcpb"
        printf '{"ok":true,"data":{"installed":"v2.14.0",\
      "extension":"gdoc.mcpb"},"warnings":["the open refused"]}\\n'
        ;;
      *)
        printf '{"ok":false,"error":"this gdoc was not asked for that"}\\n'
        exit 1
        ;;
      esac
      """
    try Data(script.utf8).write(to: gdoc)
    try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: gdoc.path)
  }

  /// Every argv list the session ran, in order.
  private func argv() throws -> [String] {
    guard let text = try? String(contentsOf: log, encoding: .utf8) else { return [] }
    return text.split(separator: "\n").map(String.init)
  }

  // MARK: the outside, written down instead of done

  /// The system, as what it was asked rather than as what it did.
  private final class FakeDesk: Desk, @unchecked Sendable {
    private let lock = NSLock()
    private var urls: [URL] = []
    private var strings: [String] = []
    private var apps: [String] = []

    var opened: [URL] { lock.withLock { urls } }
    var copied: [String] { lock.withLock { strings } }
    var activated: [String] { lock.withLock { apps } }

    func open(_ url: URL) -> Bool {
      lock.withLock { urls.append(url) }
      return true
    }

    func activate(_ bundleIdentifier: String) -> Bool {
      lock.withLock { apps.append(bundleIdentifier) }
      return true
    }

    func copy(_ text: String) {
      lock.withLock { strings.append(text) }
    }

  }

  /// This machine's login items, untouched.
  private final class FakeRegister: LoginService, @unchecked Sendable {
    private let lock = NSLock()
    private var state: LoginRegistration = .notRegistered
    /// What macOS says once it has taken the registration.
    private let after: LoginRegistration
    private(set) var registrations = 0
    private(set) var unregistrations = 0
    private(set) var settingsOpened = 0

    init(after: LoginRegistration = .enabled) {
      self.after = after
    }

    var registration: LoginRegistration { lock.withLock { state } }

    func register() throws {
      lock.withLock {
        registrations += 1
        state = after
      }
    }

    func unregister() throws {
      lock.withLock {
        unregistrations += 1
        state = .notRegistered
      }
    }

    func openSettings() {
      lock.withLock { settingsOpened += 1 }
    }
  }

  /// Everything the app may keep, in memory, so no test writes to the
  /// defaults of the machine it runs on.
  private final class Store: DefaultsStore {
    private var values: [String: Any] = [:]

    func object(forKey key: String) -> Any? { values[key] }

    func set(_ value: Any?, forKey key: String) {
      guard let value else {
        values.removeValue(forKey: key)
        return
      }
      values[key] = value
    }

    func removeObject(forKey key: String) { values.removeValue(forKey: key) }
  }
}

/// How many times an injected answer was asked for. A class, so the closure
/// that counts and the test that reads the count are looking at one number.
@MainActor
private final class Tally {
  var count = 0

  /// Counted, and the answer every caller here wants: the pid is alive.
  func bump() -> Bool {
    count += 1
    return true
  }
}

/// A walk of the hub this test opens and closes by hand.
///
/// The real walk is a file system and a clock, and the race it was in is the
/// thing under test: so the walk here waits on a gate until the test opens
/// it, and says afterwards whether it was asked to let go while it waited.
/// It runs on the session's own detached task, which is why `Task.isCancelled`
/// is the question it asks.
final class HeldScan: @unchecked Sendable {
  private let gate = DispatchSemaphore(value: 0)
  private let lock = NSLock()
  private let answer: HubIndex
  private var walks = 0
  private var cancelled = false

  init(_ answer: HubIndex) {
    self.answer = answer
  }

  /// Whether a walk has reached the gate, so a test knows the session got
  /// that far before it does anything else.
  var started: Bool {
    lock.lock()
    defer { lock.unlock() }
    return walks > 0
  }

  /// Whether a walk was cancelled while it waited.
  var wasCancelled: Bool {
    lock.lock()
    defer { lock.unlock() }
    return cancelled
  }

  /// Let one waiting walk through.
  func open() { gate.signal() }

  /// The closure the session is handed.
  var scan: @Sendable (String) -> HubIndex {
    { [self] _ in
      lock.withLock { walks += 1 }
      gate.wait()
      if Task.isCancelled {
        lock.withLock { cancelled = true }
        return HubIndex(hub: answer.hub, notes: [:], stop: .cancelled)
      }
      return answer
    }
  }
}
