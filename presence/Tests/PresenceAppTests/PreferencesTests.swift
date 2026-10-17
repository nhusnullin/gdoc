// What the app writes down, and what it refuses to write down.
//
// Three rules are held here. A dismissal names the run and the call of a
// write, or a hold's own id, and nothing about the document: a title or a
// document id in UserDefaults would be a word of somebody's document kept
// outside the record. A dismissal older than sixty-two days is gone, because
// the record it was about is gone. And the store writes six keys, which is
// the whole of what this app remembers between two launches.
//
// The store is handed in, so no test here touches the defaults of the
// machine it runs on.

import Foundation
import PresenceCore
import XCTest

@testable import PresenceApp

final class PreferencesTests: XCTestCase {

  // MARK: the dismissals

  func testDismissalsAreKeyedByRunAndCallOrHoldId() {
    // Arrange: a failed propose and an open hold, as the fold reads them.
    let day = Self.at("2026-10-17T09:30:00Z")
    let events = [
      Self.event(
        at: "2026-10-17T09:10:00Z", kind: .end, command: "propose", run: "01JAB", call: 7,
        doc: "1AbCdEfGhIjK", title: "Q3 plan", ok: false),
      Self.event(
        at: "2026-10-17T09:20:00Z", kind: .held, command: "propose", run: "01JXY", call: nil,
        doc: "1AbCdEfGhIjK", hold: "h-42"),
      Self.event(
        at: "2026-10-17T09:25:00Z", kind: .start, command: "publish", run: "01JOB", call: 3,
        doc: "1AbCdEfGhIjK"),
    ]
    let fold = Now.fold(events, now: day, alive: { _ in true })
    let store = Store()
    let preferences = Preferences(store: store)

    // Act
    for entry in fold.entries {
      guard let key = Preferences.key(of: entry) else { continue }
      preferences.dismiss(key, at: day)
    }

    // Assert: the two keys the fold's own `dismissed` argument takes.
    XCTAssertEqual(preferences.dismissals(now: day), ["01JAB|7", "h-42"])
    XCTAssertNil(
      Preferences.key(of: fold.entries.first { if case .job = $0 { true } else { false } }!),
      "a running job is nothing to dismiss")

    // And the fold with them is a panel with no card left.
    let after = Now.fold(
      events, now: day, dismissed: preferences.dismissals(now: day), alive: { _ in true })
    XCTAssertEqual(after.holds.count, 0)
    XCTAssertEqual(after.stuck.count, 0)

    // No word of the document is written down.
    XCTAssertFalse(store.text.contains("1AbCdEfGhIjK"), "a document id is not the app's to keep")
    XCTAssertFalse(store.text.contains("Q3 plan"), "a title is not the app's to keep")
  }

  func testDismissalsArePrunedPastSixtyTwoDays() {
    XCTAssertEqual(Preferences.keepFor, 62 * 24 * 60 * 60)

    let now = Self.at("2026-10-17T09:30:00Z")
    let store = Store()
    let preferences = Preferences(store: store)

    preferences.dismiss("01OLD|1", at: now.addingTimeInterval(-63 * 24 * 60 * 60))
    preferences.dismiss("01NEW|2", at: now.addingTimeInterval(-61 * 24 * 60 * 60))

    XCTAssertEqual(preferences.dismissals(now: now), ["01NEW|2"])
    XCTAssertFalse(store.text.contains("01OLD|1"), "the pruned one leaves the store too")
  }

  // MARK: the six keys

  func testOnlyTheSixKeysPersist() {
    XCTAssertEqual(
      Preferences.Key.allCases.map(\.rawValue),
      ["dismissed", "hubPath", "stripSeen", "loginItem.choice", "relaunch", "lastAccount"])

    let now = Self.at("2026-10-17T09:30:00Z")
    let store = Store()
    let preferences = Preferences(store: store)

    preferences.dismiss("01JAB|7", at: now)
    preferences.hub = "/Users/someone/hub"
    preferences.stripSeen = true
    preferences.loginItemChoice = .off
    preferences.noteRelaunch(version: "v2.15.0", at: now)
    preferences.lastAccount = "someone@example.com"

    XCTAssertEqual(store.keys, Set(Preferences.Key.allCases.map(\.rawValue)))

    // And each reads back as it was written, which is what persisting means.
    XCTAssertEqual(preferences.dismissals(now: now), ["01JAB|7"])
    XCTAssertEqual(preferences.hub, "/Users/someone/hub")
    XCTAssertTrue(preferences.stripSeen)
    XCTAssertEqual(preferences.loginItemChoice, .off)
    XCTAssertEqual(preferences.lastRelaunch(now: now), Relaunch(version: "v2.15.0", at: now))
    XCTAssertEqual(preferences.lastAccount, "someone@example.com")
  }

  // MARK: a store of its own

  /// Everything the app may keep, in memory, with the keys it was asked for.
  private final class Store: DefaultsStore {
    private var values: [String: Any] = [:]

    var keys: Set<String> { Set(values.keys) }

    /// Everything in the store as one string, for a test that holds what is
    /// not in it.
    var text: String { String(describing: values) }

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

  private static func at(_ text: String) -> Date {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return formatter.date(from: text) ?? Date()
  }

  private static func event(
    at text: String, kind: EventKind, command: String, run: String, call: Int64?, doc: String?,
    title: String? = nil, ok: Bool? = nil, hold: String? = nil
  ) -> Event {
    Event(
      schema: 1, at: at(text), gdoc: "v2.14.0", pid: 6000, run: run, call: call, kind: kind,
      command: command, via: "terminal", doc: doc, wait: false, title: title, ok: ok,
      verified: nil, ms: nil, hold: hold, rule: nil, until: nil)
  }
}
