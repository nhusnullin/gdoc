// What the app remembers between two launches, and the six keys it keeps it
// under.
//
// The record is the whole of what gdoc did, and it is the Go binary's file.
// This app writes nothing into it, ever. What it does write down is six
// things about itself: the cards a person hid, the hub folder they chose,
// whether the four labels under the first document row have been read, their
// launch-at-login answer, the last self-relaunch it did, and the address it
// last saw signed in. Nothing else persists, which is the specification's
// own row, and a source scan in PresenceCoreTests holds the word
// UserDefaults to this file so a seventh key cannot appear while something
// else is being fixed.
//
// A dismissal names a run and a call, or a hold's own id, and nothing about
// the document: a title or a document id in UserDefaults would be a word of
// somebody's document kept outside the record, where nothing prunes it.
// Sixty-two days is what prunes the two keys that are about a moment, the
// dismissals and the relaunch stamp: the tail reads two month files, so a
// key older than that is about a line this app can no longer see. The other
// four are answers rather than moments, and an answer does not go stale: an
// "off" dropped after two months would register somebody at login again,
// which is the one thing the specification says never happens.
//
// The store is handed in. UserDefaults is what the app passes; a test passes
// a dictionary of its own, so no test writes to the defaults of the machine
// it runs on.

import Foundation
import PresenceCore

/// Where keys and values are kept.
public protocol DefaultsStore: AnyObject {
  func object(forKey key: String) -> Any?
  func set(_ value: Any?, forKey key: String)
  func removeObject(forKey key: String)
}

extension UserDefaults: DefaultsStore {}

/// The person's answer about launching at login. It is their answer and not
/// the system's state: an "off" is never overridden by a relaunch or an
/// update, which is the only reason it is written down at all.
public enum LoginItemChoice: String, Sendable, Equatable, CaseIterable {
  case on
  case off
}

/// The app's own state, on disk, and the whole of it.
///
/// Unchecked Sendable: the store is reached under a lock, because a dismissal
/// comes off the panel and a relaunch stamp off a watcher.
public final class Preferences: @unchecked Sendable {
  /// The six keys. A seventh is a decision in `docs/v2/DECISIONS.md`, never a
  /// line added here.
  public enum Key: String, CaseIterable, Sendable {
    case dismissed
    case hubPath
    case stripSeen
    case loginItemChoice = "loginItem.choice"
    case relaunch
    case lastAccount
  }

  /// How long the two keys that are about a moment are kept: sixty-two
  /// days, which is past both month files the tail reads. It is the
  /// dismissals and the relaunch stamp, and no other key: the hub, the
  /// strip, the launch-at-login answer and the last account are answers
  /// rather than moments, and nothing prunes an answer.
  public static let keepFor: TimeInterval = 62 * 24 * 60 * 60

  private let store: DefaultsStore
  private let lock = NSLock()

  public init(store: DefaultsStore = UserDefaults.standard) {
    self.store = store
  }

  // MARK: the dismissals

  /// What a dismissal of this card names, or nothing where the card is not
  /// one a person can dismiss. A running job and a live watch are not: what
  /// ends them is the work finishing or Stop.
  public static func key(of entry: NowEntry) -> String? {
    switch entry {
    case .hold(let card): card.id
    case .stuck(let write): write.key
    case .job, .watch: nil
    }
  }

  /// The keys still in force, with anything past the horizon dropped from
  /// the store as it is read.
  public func dismissals(now: Date = Date()) -> Set<String> {
    lock.lock()
    defer { lock.unlock() }
    return Set(kept(now: now).keys)
  }

  /// Hide one card. The row in TODAY stays: a dismissal hides a card and
  /// rewrites nothing.
  public func dismiss(_ key: String, at moment: Date = Date()) {
    lock.lock()
    defer { lock.unlock() }
    var stamps = kept(now: moment)
    stamps[key] = moment.timeIntervalSince1970
    store.set(stamps, forKey: Key.dismissed.rawValue)
  }

  // MARK: the hub, the strip, the choice, the account

  /// The hub folder a person chose, as an absolute path.
  public var hub: String? {
    get { string(.hubPath) }
    set { write(newValue, .hubPath) }
  }

  /// Whether the four labels under the first document row have been read.
  public var stripSeen: Bool {
    get {
      lock.lock()
      defer { lock.unlock() }
      return store.object(forKey: Key.stripSeen.rawValue) as? Bool ?? false
    }
    set {
      lock.lock()
      defer { lock.unlock() }
      store.set(newValue, forKey: Key.stripSeen.rawValue)
    }
  }

  /// The person's launch-at-login answer, or nothing before the first run
  /// ever has recorded one.
  public var loginItemChoice: LoginItemChoice? {
    get { string(.loginItemChoice).flatMap(LoginItemChoice.init(rawValue:)) }
    set { write(newValue?.rawValue, .loginItemChoice) }
  }

  /// The address the app last saw signed in, which is the whole of how it
  /// knows an account changed.
  public var lastAccount: String? {
    get { string(.lastAccount) }
    set { write(newValue, .lastAccount) }
  }

  // MARK: the relaunch

  /// The last relaunch, where there is one inside the horizon.
  public func lastRelaunch(now: Date = Date()) -> RelaunchStamp? {
    lock.lock()
    defer { lock.unlock() }
    guard let written = store.object(forKey: Key.relaunch.rawValue) as? [String: Any],
      let version = written["version"] as? String,
      let seconds = written["at"] as? Double
    else { return nil }
    let at = Date(timeIntervalSince1970: seconds)
    guard now.timeIntervalSince(at) < Self.keepFor else {
      store.removeObject(forKey: Key.relaunch.rawValue)
      return nil
    }
    return RelaunchStamp(version: version, at: at)
  }

  /// Write down that the app opened its own bundle again.
  public func noteRelaunch(version: String, at moment: Date = Date()) {
    lock.lock()
    defer { lock.unlock() }
    store.set(
      ["version": version, "at": moment.timeIntervalSince1970] as [String: Any],
      forKey: Key.relaunch.rawValue)
  }

  // MARK: the store

  /// The dismissals inside the horizon, written back where the read dropped
  /// something. The caller holds the lock.
  private func kept(now: Date) -> [String: Double] {
    let stamps = store.object(forKey: Key.dismissed.rawValue) as? [String: Double] ?? [:]
    let inside = stamps.filter {
      now.timeIntervalSince(Date(timeIntervalSince1970: $0.value)) < Self.keepFor
    }
    if inside.count != stamps.count {
      store.set(inside, forKey: Key.dismissed.rawValue)
    }
    return inside
  }

  private func string(_ key: Key) -> String? {
    lock.lock()
    defer { lock.unlock() }
    return store.object(forKey: key.rawValue) as? String
  }

  private func write(_ value: String?, _ key: Key) {
    lock.lock()
    defer { lock.unlock() }
    guard let value else {
      store.removeObject(forKey: key.rawValue)
      return
    }
    store.set(value, forKey: key.rawValue)
  }
}
