// What the app writes down, and what it refuses to write down.
//
// Three rules are held here. A dismissal names the run and the call of a
// write, or a hold's own id, and nothing about the document: a title or a
// document id in UserDefaults would be a word of somebody's document kept
// outside the record. A dismissal older than sixty-two days is gone, because
// the record it was about is gone. And the store writes eight keys, which is
// the whole of what this app remembers between two launches.
//
// The two keys of the second morning are the one written-down exception to
// the first rule: a pin and a hide name a document, and a pin keeps its last
// title, because a pinned document whose lines are past the horizon has its
// name nowhere else. The decision of 2026-10-18 is where that is written.
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

  /// Two of the six keys are stamps and four are answers, and only a stamp
  /// is pruned. An answer that went stale after two months would be the
  /// app changing its mind about something the person said: a dropped "off"
  /// puts the app back in their login items, and a dropped hub takes the
  /// pairing away from somebody who chose a folder once.
  func testOnlyTheTwoStampsArePruned() {
    let now = Self.at("2026-10-17T09:30:00Z")
    let long = now.addingTimeInterval(-63 * 24 * 60 * 60)
    let store = Store()
    let preferences = Preferences(store: store)

    preferences.hub = "/Users/someone/hub"
    preferences.stripSeen = true
    preferences.loginItemChoice = .off
    preferences.lastAccount = "someone@example.com"
    preferences.dismiss("01OLD|1", at: long)
    preferences.noteRelaunch(version: "2.14.0", at: long)

    XCTAssertTrue(preferences.dismissals(now: now).isEmpty, "a stamp past the horizon goes")
    XCTAssertNil(preferences.lastRelaunch(now: now), "and so does the relaunch's")

    XCTAssertEqual(preferences.hub, "/Users/someone/hub", "the folder they chose stays")
    XCTAssertTrue(preferences.stripSeen, "labels read once are read")
    XCTAssertEqual(preferences.loginItemChoice, .off, "an off is never turned back on")
    XCTAssertEqual(preferences.lastAccount, "someone@example.com")

    preferences.pin("1AbCdEfGhIjK", title: "Q3 plan", at: long)
    preferences.hide("1ZzYyXxWwVvU", at: long)
    XCTAssertEqual(preferences.pins.map(\.doc), ["1AbCdEfGhIjK"], "a pin is an answer too")
    XCTAssertEqual(preferences.hides.map(\.doc), ["1ZzYyXxWwVvU"], "and so is a hide")
  }

  // MARK: the pins and the hides

  /// A pin is a document's id, the moment, and the title the list was
  /// drawing. The list comes back newest pin first, which is the order
  /// DOCUMENTS draws them in, and taking a pin back leaves nothing behind.
  func testPinsComeBackNewestFirstAndUnpinLeavesNothing() {
    let store = Store()
    let preferences = Preferences(store: store)
    let morning = Self.at("2026-10-17T09:00:00Z")

    preferences.pin("1AbCdEfGhIjK", title: "Q3 plan", at: morning)
    preferences.pin("1ZzYyXxWwVvU", title: "Chapter 4", at: morning.addingTimeInterval(60))

    XCTAssertEqual(preferences.pins.map(\.doc), ["1ZzYyXxWwVvU", "1AbCdEfGhIjK"])
    XCTAssertEqual(preferences.pins.map(\.title), ["Chapter 4", "Q3 plan"])
    XCTAssertEqual(preferences.pins.first?.at, morning.addingTimeInterval(60))

    // Pinning one that is pinned moves it to the front: the moment is the
    // one the person just made.
    preferences.pin("1AbCdEfGhIjK", title: "Q3 plan, again", at: morning.addingTimeInterval(120))
    XCTAssertEqual(preferences.pins.map(\.doc), ["1AbCdEfGhIjK", "1ZzYyXxWwVvU"])
    XCTAssertEqual(preferences.pins.first?.title, "Q3 plan, again")

    preferences.unpin("1AbCdEfGhIjK")
    XCTAssertEqual(preferences.pins.map(\.doc), ["1ZzYyXxWwVvU"])
    XCTAssertFalse(store.text.contains("1AbCdEfGhIjK"), "the pin that went leaves the store")

    preferences.unpin("1ZzYyXxWwVvU")
    XCTAssertTrue(preferences.pins.isEmpty)
    XCTAssertFalse(
      store.keys.contains(Preferences.Key.pinned.rawValue),
      "the last pin takes the key with it")
  }

  /// A hide names the document and the moment it was hidden, which is what
  /// the fold reads the next action against, and never a title: the row a
  /// hide takes away is drawn nowhere, and the last title in UserDefaults is
  /// the pin's own written-down exception.
  func testAHideKeepsItsMomentAndIsNeverPruned() {
    let now = Self.at("2026-10-17T09:30:00Z")
    let longAgo = now.addingTimeInterval(-200 * 24 * 60 * 60)
    let store = Store()
    let preferences = Preferences(store: store)

    preferences.hide("1AbCdEfGhIjK", at: longAgo)

    XCTAssertEqual(preferences.hides.map(\.doc), ["1AbCdEfGhIjK"])
    XCTAssertEqual(preferences.hides.first?.at, longAgo)
    XCTAssertNil(preferences.hides.first?.title, "a hide keeps no word of the document")
    XCTAssertFalse(store.text.contains("title"), "and writes none into the store")
    XCTAssertEqual(
      preferences.dismissals(now: now).count, 0,
      "the read that prunes the stamps leaves the hide where it is")
    XCTAssertEqual(
      preferences.hides.count, 1,
      "two hundred days on, a document somebody took away is still away")
  }

  // MARK: the eight keys

  func testOnlyTheEightKeysPersist() {
    XCTAssertEqual(
      Preferences.Key.allCases.map(\.rawValue),
      [
        "dismissed", "hubPath", "stripSeen", "loginItem.choice", "relaunch", "lastAccount",
        "pinned", "hidden",
      ])

    let now = Self.at("2026-10-17T09:30:00Z")
    let store = Store()
    let preferences = Preferences(store: store)

    preferences.dismiss("01JAB|7", at: now)
    preferences.hub = "/Users/someone/hub"
    preferences.stripSeen = true
    preferences.loginItemChoice = .off
    preferences.noteRelaunch(version: "v2.15.0", at: now)
    preferences.lastAccount = "someone@example.com"
    preferences.pin("1AbCdEfGhIjK", title: "Q3 plan", at: now)
    preferences.hide("1ZzYyXxWwVvU", at: now)

    XCTAssertEqual(store.keys, Set(Preferences.Key.allCases.map(\.rawValue)))

    // And each reads back as it was written, which is what persisting means.
    XCTAssertEqual(preferences.dismissals(now: now), ["01JAB|7"])
    XCTAssertEqual(preferences.hub, "/Users/someone/hub")
    XCTAssertTrue(preferences.stripSeen)
    XCTAssertEqual(preferences.loginItemChoice, .off)
    XCTAssertEqual(preferences.lastRelaunch(now: now), RelaunchStamp(version: "v2.15.0", at: now))
    XCTAssertEqual(preferences.lastAccount, "someone@example.com")
    XCTAssertEqual(
      preferences.pins, [DocumentChoice(doc: "1AbCdEfGhIjK", at: now, title: "Q3 plan")])
    XCTAssertEqual(
      preferences.hides, [DocumentChoice(doc: "1ZzYyXxWwVvU", at: now, title: nil)])
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
