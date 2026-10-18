// What the app puts in front of somebody, and what it never does.
//
// No test here opens a browser, activates an application or writes to the
// pasteboard: the opener is handed a desk that writes down what it was asked
// for. Every URL is a literal in this file.

import Foundation
import PresenceCore
import PresenceUI
import XCTest

@testable import PresenceApp

final class OpenerTests: XCTestCase {

  private static let id = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"
  private static let link =
    "https://docs.google.com/document/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd/edit"
  /// A title of somebody's own, which must reach nothing here.
  private static let title = "Supplier register policy, Q4 draft"

  func testCopyLinkIsThePlainURL() {
    let desk = NoteDesk()
    let opener = Opener(desk: desk)

    XCTAssertEqual(opener.copyLink(Self.id), Self.link)
    XCTAssertEqual(desk.copied, [Self.link], "the plain URL, with nothing around it")
    XCTAssertTrue(desk.opened.isEmpty, "a copy opens nothing")

    XCTAssertNil(opener.copyLink(nil), "no id, nothing copied")
    XCTAssertNil(opener.copyLink("short"), "not a Drive id, nothing copied")
    XCTAssertEqual(desk.copied.count, 1, "the pasteboard was touched once")

    // And the open, which is the same URL through the other verb.
    XCTAssertTrue(opener.openDocument(Self.id))
    XCTAssertEqual(desk.opened.map(\.absoluteString), [Self.link])
    XCTAssertFalse(opener.openDocument(nil), "a greyed icon opens nothing")
    XCTAssertEqual(desk.opened.count, 1)
  }

  func testNoTitleEverReachesAURLOrThePasteboard() throws {
    let hub = try temporaryHub()
    try note(in: hub, at: "notes/supplier-register.md")

    let desk = NoteDesk()
    let opener = Opener(desk: desk)
    opener.openDocument(Self.id)
    opener.copyLink(Self.id)
    _ = opener.openNote(at: hub + "/notes/supplier-register.md")
    opener.continueWithClaude()

    let everything =
      desk.opened.map(\.absoluteString) + desk.copied + desk.activated
    XCTAssertFalse(everything.isEmpty, "something was asked for, so this test guards something")
    for said in everything {
      XCTAssertFalse(said.contains(Self.title), "\(said) carries somebody's document title")
      XCTAssertFalse(said.contains("Supplier"), "\(said) carries a word of the document")
    }
  }

  func testAnUnpairedDocumentGreysTheIconWithItsTooltip() throws {
    let desk = NoteDesk()
    let opener = Opener(desk: desk)

    let unpaired = opener.openNote(at: nil)
    XCTAssertEqual(unpaired.fault, .notPaired)
    XCTAssertEqual(NoteFault.notPaired.tip, "No note is paired with this document")

    XCTAssertTrue(desk.opened.isEmpty, "a greyed icon opens nothing")

    // The panel's own reading of the same fact: the icon is off, and its
    // words are the tooltip.
    let moment = Date(timeIntervalSince1970: 1_792_000_000)
    let written = try Event.decode(
      """
      {"v":1,"at":"\(ISO8601DateFormatter().string(from: moment))","gdoc":"v2.14.0",\
      "pid":4201,"run":"0d1a2b3c4d5e6f70","call":1,"event":"end","command":"propose",\
      "via":"terminal","doc":"\(Self.id)","title":"\(Self.title)","ok":true,"ms":10}
      """)
    // No note names this document, which is the one reason the icon is
    // grey since the third look.
    let panel = PanelModel.build(
      events: [written], facts: Facts(versions: Versions(app: "1.0")), now: moment,
      calendar: Calendar(identifier: .gregorian), alive: { _ in false })
    let icons = panel.sections.compactMap { section -> [ActionModel]? in
      guard case .documents(let documents) = section else { return nil }
      return documents.rows.first?.actions
    }.first
    XCTAssertEqual(
      icons?.first(where: { $0.kind == .openNote })?.state, .off(Tips.noNote),
      "nothing is paired, so the panel greys the icon with the opener's own words")
  }

  func testAMovedNoteSaysMovedOrDeleted() throws {
    let hub = try temporaryHub()
    let desk = NoteDesk()
    let opener = Opener(desk: desk)
    let path = hub + "/notes/supplier-register.md"

    // The finder named it, and it is not there any more.
    let gone = opener.openNote(at: path)
    XCTAssertEqual(gone.fault, .movedOrDeleted)
    XCTAssertEqual(NoteFault.movedOrDeleted.tip, "Note moved or deleted")
    XCTAssertTrue(desk.opened.isEmpty)

    try note(in: hub, at: "notes/supplier-register.md")
    let opened = opener.openNote(at: path)
    XCTAssertTrue(
      try opened.get().absoluteString.hasPrefix("obsidian://open?path=%2F"),
      "the absolute path, percent-encoded to the unreserved set")
    XCTAssertEqual(desk.opened.count, 1)

    // A path that is not an absolute Markdown path is refused before
    // anything opens, and reads as the same sentence.
    let relative = opener.openNote(at: "../../somewhere-else.md")
    XCTAssertEqual(relative.fault, .notMarkdown)
    XCTAssertEqual(NoteFault.notMarkdown.tip, "Note moved or deleted")
    XCTAssertEqual(desk.opened.count, 1, "nothing more was opened")
  }

  /// The sentence only a failed open can produce. The index is rebuilt on a
  /// file event, so an unpaired document greys itself; a note the index
  /// still names and Obsidian will not open is a fault nothing else in the
  /// panel can see, so the click that found it is what the icon carries.
  func testAFailedOpenGreysTheIconWithItsReason() throws {
    let moment = Date(timeIntervalSince1970: 1_792_000_000)
    let written = try Event.decode(
      """
      {"v":1,"at":"\(ISO8601DateFormatter().string(from: moment))","gdoc":"v2.14.0",\
      "pid":4201,"run":"0d1a2b3c4d5e6f70","call":1,"event":"end","command":"propose",\
      "via":"terminal","doc":"\(Self.id)","title":"\(Self.title)","ok":true,"ms":10}
      """)
    let facts = Facts(
      versions: Versions(app: "1.0"),
      pairing: [Self.id: "/somewhere/notes/supplier-register.md"])

    let paired = try noteIcon(in: written, facts: facts, ui: UIState())
    XCTAssertEqual(paired.state, .live, "a note names the document, so the icon is live")

    let failed = try noteIcon(
      in: written, facts: facts, ui: UIState(noteFault: [Self.id: .movedOrDeleted]))
    XCTAssertEqual(
      failed.state, .off(Tips.noteMoved),
      "the last click found nothing there, and the icon says so")
  }

  /// The Open md icon of the one document row the panel draws.
  private func noteIcon(in event: Event, facts: Facts, ui: UIState) throws -> ActionModel {
    let panel = PanelModel.build(
      events: [event], facts: facts, ui: ui, now: event.at,
      calendar: Calendar(identifier: .gregorian), alive: { _ in false })
    let row = try XCTUnwrap(panel.documents?.rows.first)
    return try XCTUnwrap(row.actions.first { $0.kind == .openNote })
  }

  func testContinueWithClaudeOnlyActivatesTheApp() {
    let desk = NoteDesk()
    let opener = Opener(desk: desk)

    XCTAssertTrue(opener.continueWithClaude())
    XCTAssertEqual(desk.activated, ["com.anthropic.claudefordesktop"])
    XCTAssertTrue(desk.opened.isEmpty, "no URL, so no prefill")
    XCTAssertTrue(desk.copied.isEmpty, "and nothing on the pasteboard either")
    XCTAssertEqual(Opener.claudeDesktop, "com.anthropic.claudefordesktop")
  }

  func testTheStripShowsOnceUntilAnIconIsClicked() {
    let first = UIState(stripSeen: false)
    XCTAssertFalse(first.stripSeen, "the labels are under the first row of a first open")

    let clicked = first.afterAnIconClick()
    XCTAssertTrue(clicked.stripSeen, "the first click on any icon is the proof they were read")
    XCTAssertTrue(clicked.afterAnIconClick().stripSeen, "and they never come back")
    XCTAssertEqual(
      UIState().stripSeen, true, "a panel that remembers nothing else still remembers this")
  }

  // MARK: a hub of this test's own

  private func temporaryHub() throws -> String {
    let base = URL(fileURLWithPath: NSTemporaryDirectory())
      .appendingPathComponent("opener-\(UUID().uuidString)")
      .appendingPathComponent("hub")
    try FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
    addTeardownBlock {
      try? FileManager.default.removeItem(at: base.deletingLastPathComponent())
    }
    return base.path
  }

  private func note(in hub: String, at relative: String) throws {
    let url = URL(fileURLWithPath: hub).appendingPathComponent(relative)
    try FileManager.default.createDirectory(
      at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
    try """
    ---
    title: \(Self.title)
    gdoc:
      schema: 1
      document_id: \(Self.id)
    ---

    # Scope
    """.write(to: url, atomically: true, encoding: .utf8)
  }
}

/// A desk that opens nothing and writes down what it was asked for.
final class NoteDesk: Desk, @unchecked Sendable {
  private let lock = NSLock()
  private var openedURLs: [URL] = []
  private var copiedText: [String] = []
  private var activatedApps: [String] = []

  var opened: [URL] { read { openedURLs } }
  var copied: [String] { read { copiedText } }
  var activated: [String] { read { activatedApps } }

  @discardableResult func open(_ url: URL) -> Bool {
    lock.lock()
    openedURLs.append(url)
    lock.unlock()
    return true
  }

  @discardableResult func activate(_ bundleIdentifier: String) -> Bool {
    lock.lock()
    activatedApps.append(bundleIdentifier)
    lock.unlock()
    return true
  }

  func copy(_ text: String) {
    lock.lock()
    copiedText.append(text)
    lock.unlock()
  }

  private func read<T>(_ body: () -> T) -> T {
    lock.lock()
    defer { lock.unlock() }
    return body()
  }
}

/// Why a result failed, where it did. The same one line as the core tests'
/// own: two test targets share no source.
extension Result {
  var fault: Failure? {
    guard case .failure(let why) = self else { return nil }
    return why
  }
}
