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

  /// The facts are the polls', the stores' and the record's, and the model is
  /// built out of all three.
  func testTheFactsComeFromThePollsTheStoresAndTheRecord() throws {
    try writeTheRecord([
      line(kind: "start", command: "comments", call: 1, doc: Self.doc, wait: true),
      line(
        kind: "end", command: "comments", call: 1, doc: Self.doc, wait: true,
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
    XCTAssertEqual(session.facts.hub, preferences.hub, "the hub is the store's")
    XCTAssertEqual(session.facts.loginItem, .on, "the register answered")

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

  // MARK: what a press reaches

  /// The four icons on a row, each through the one room that does it, and the
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

    // No hub is chosen, so there is no note to open and nothing is opened.
    session.perform(.openNote, Self.doc)
    XCTAssertEqual(desk.opened.count, 1, "the note icon opened nothing")
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
  func testTheSwitchRecordsThePersonsAnswer() throws {
    try writeTheRecord([])
    let store = Store()
    let register = FakeRegister()
    let session = made(preferences: Preferences(store: store), loginService: register)
    answered(session)
    XCTAssertEqual(register.registrations, 1, "the first run ever registers once")

    session.perform(.toggleLoginItem, nil)
    XCTAssertEqual(
      store.object(forKey: Preferences.Key.loginItemChoice.rawValue) as? String, "off")
    XCTAssertEqual(register.unregistrations, 1)

    session.perform(.openLoginItems, nil)
    XCTAssertEqual(register.settingsOpened, 1, "the approval row opens the pane")
  }

  /// A section a person opened stays open, because what is open lives outside
  /// every view.
  func testASectionAPersonOpenedStaysOpen() throws {
    try writeTheRecord([])
    let session = made()
    answered(session)

    session.toggle(.documents)
    XCTAssertTrue(session.ui.isOpen(.documents))
    XCTAssertEqual(session.model.documents?.isOpen, true)
    session.toggle(.documents)
    XCTAssertFalse(session.ui.isOpen(.documents))
  }

  // MARK: a session over this test's own machine

  private func made(
    preferences: Preferences = Preferences(store: Store()),
    desk: Desk = FakeDesk(),
    loginService: LoginService = FakeRegister(),
    quit: @escaping () -> Void = {}
  ) -> Session {
    let made = Session(
      places: Places(
        records: records, bundle: nil, version: "2.15.0", identifier: "", home: root.path,
        arguments: []),
      calendar: Calendar.current,
      preferences: preferences,
      desk: desk,
      loginService: loginService,
      environment: ["HOME": root.path, "PATH": "/usr/bin:/bin"],
      alive: { _ in false },
      quit: quit)
    session = made
    return made
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
    extra: String = ""
  ) -> String {
    var fields = [
      "\"v\":1", "\"at\":\"\(stamp())\"", "\"gdoc\":\"v2.14.0\"", "\"pid\":4242",
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
    private(set) var registrations = 0
    private(set) var unregistrations = 0
    private(set) var settingsOpened = 0

    var registration: LoginRegistration { lock.withLock { state } }

    func register() throws {
      lock.withLock {
        registrations += 1
        state = .enabled
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
