// What `gdoc help --json` and `gdoc auth status` say, as values.
//
// Two commands answer everything HEALTH knows that the record does not: the
// mark in effect, the installed version and the one published, whether the
// token works, and whose it is. This file reads those two objects and nothing
// else. It starts no program and opens no file: the bytes arrive from the
// app library, which is the only room that runs anything.
//
// Strict, by the same rule the record's own decoder follows. A field a newer
// gdoc prints is read past, because an app that refused one would stop
// working on the release after this one; a field this app does know and that
// holds the wrong kind of value is refused by name, because something other
// than gdoc wrote it.
//
// Nothing here judges. The state is the four words the token can be in, the
// domain is a hint that blocks nothing, and "Account changed" is two
// addresses this app saw. No answer here is ever taken from the record: a
// line of the record is forgeable and carries no account at all.

import Foundation

/// Which strict reading of a health answer did not hold. The panel shows
/// none of these words; a test names the rule it is holding.
public enum HealthFault: Error, Equatable, Sendable {
  case undecodable(String)
}

/// What `gdoc help --json` says about this installation.
///
/// The mark is a setting and may be empty, so it is a string that is there
/// rather than one that is non-empty. The two versions are absent on a
/// checkout build, which names no release.
public struct HelpFacts: Sendable, Equatable {
  /// The mark in effect, which every thread gdoc writes opens with.
  public let mark: String
  /// Where that mark came from: the default, or the settings file.
  public let markFrom: String?
  /// The version of the binary that answered.
  public let installed: String?
  /// The newest release gdoc's own daily check heard about.
  public let latestStable: String?
  /// Where gdoc keeps the house logo on this machine. Nothing on a machine
  /// that has never built a document, which has no picture to draw.
  public let logoPath: String?

  public init(
    mark: String, markFrom: String? = nil, installed: String? = nil,
    latestStable: String? = nil, logoPath: String? = nil
  ) {
    self.mark = mark
    self.markFrom = markFrom
    self.installed = installed
    self.latestStable = latestStable
    self.logoPath = logoPath
  }

  /// The envelope's `data` object, read.
  public static func decode(_ data: Data) throws -> HelpFacts {
    let object = try HealthObject(data)
    guard let settings = object.object("settings") else {
      throw HealthFault.undecodable("no settings")
    }
    guard let mark = try settings.string("mark", or: "no settings.mark") else {
      throw HealthFault.undecodable("no settings.mark")
    }
    let update = object.object("update")
    return HelpFacts(
      mark: mark,
      markFrom: try settings.string("mark_from", or: "no settings.mark_from"),
      installed: try update?.string("installed", or: "no update.installed"),
      latestStable: try update?.string("latest_stable", or: "no update.latest_stable"),
      logoPath: try object.string("logo_path", or: "no logo_path"))
  }
}

/// The four states a token can be in. The panel words three of them; the
/// fourth is a machine that has no token at all.
public enum AuthState: Sendable, Equatable, CaseIterable {
  case ok
  case needsSignIn
  case missingScopes
  case signedOut
}

/// What `gdoc auth status` says about the token.
public struct AuthFacts: Sendable, Equatable {
  public let tokenPresent: Bool
  /// What the token file said when the command read it. Nothing where there
  /// is no token to describe.
  public let expired: Bool?
  /// What gdoc asks for that the token does not carry.
  public let missingScopes: [String]
  /// The address the token signs in as, or nothing where the read did not
  /// answer.
  public let account: String?
  /// The binary's own version, which this command reports beside the token.
  public let version: String?

  public init(
    tokenPresent: Bool,
    expired: Bool? = nil,
    missingScopes: [String] = [],
    account: String? = nil,
    version: String? = nil
  ) {
    self.tokenPresent = tokenPresent
    self.expired = expired
    self.missingScopes = missingScopes
    self.account = account
    self.version = version
  }

  /// The one word the row says.
  ///
  /// A missing scope comes before an expired token: an expired access token
  /// refreshes itself on the next request, and a scope the token never
  /// carried needs a new consent, which is a different sign-in.
  public var state: AuthState {
    guard tokenPresent else { return .signedOut }
    if !missingScopes.isEmpty { return .missingScopes }
    return expired == true ? .needsSignIn : .ok
  }

  /// A token that works beside an account that could not be read. The row
  /// says so in those words rather than leaving the line blank.
  public var isAccountUnread: Bool { tokenPresent && account == nil }

  /// The envelope's `data` object, read.
  public static func decode(_ data: Data) throws -> AuthFacts {
    let object = try HealthObject(data)
    guard let present = try object.bool("token_present", or: "no token_present") else {
      throw HealthFault.undecodable("no token_present")
    }
    return AuthFacts(
      tokenPresent: present,
      expired: try object.bool("expired", or: "no expired"),
      missingScopes: try object.strings("missing_scopes", or: "no missing_scopes"),
      account: try object.string("account", or: "no account"),
      version: try object.string("version", or: "no version"))
  }
}

/// The domain the person expects to be signed in as.
///
/// An app preference, empty by default, and a hint and nothing more: it
/// gates no action, changes no word of the token's state, and stops no poll.
/// A firm that signs in as one domain gets a line when it is signed in as
/// another; everybody else never sees it.
public struct DomainHint: Sendable, Equatable {
  /// Nothing expected, which is how the app ships.
  public static let unset = DomainHint(expected: "")

  public let expected: String

  /// The preference as typed, with a leading at sign taken off: somebody
  /// writing `@example.com` means the same thing as `example.com`.
  public init(expected: String) {
    self.expected = expected.hasPrefix("@") ? String(expected.dropFirst()) : expected
  }

  /// The line under the account row, or nothing. An address that could not
  /// be read and a word with no domain in it are not mismatches.
  public func sentence(for account: String?) -> String? {
    guard !expected.isEmpty, let account else { return nil }
    guard let at = account.lastIndex(of: "@") else { return nil }
    let domain = account[account.index(after: at)...]
    guard domain.lowercased() != expected.lowercased() else { return nil }
    return "the expected domain is \(expected)"
  }
}

/// The addresses this app has seen, and nothing else.
///
/// "Account changed" is the app's own memory: the record carries no account,
/// and a line in it is forgeable, so the only honest source for the sentence
/// is the two answers the app read itself.
public struct AccountMemory: Sendable, Equatable {
  /// The address the app saw last, or nothing before the first reading.
  public let seen: String?

  public init(seen: String? = nil) {
    self.seen = seen
  }

  /// The address the app saw before this one, where this one is another. The
  /// first reading is never a change, and an account that could not be read
  /// is not a sign-in as somebody else.
  public func change(to account: String?) -> String? {
    guard let seen, let account, seen != account else { return nil }
    return seen
  }

  /// The memory after this reading. An account that could not be read
  /// forgets nothing: the token is still the one the app knows.
  public func noting(_ account: String?) -> AccountMemory {
    guard let account else { return self }
    return AccountMemory(seen: account)
  }
}

/// One JSON object, read field by field.
///
/// A small reader rather than `Decodable`, because the two answers are read
/// the way the record is read: a key this app knows is held to its type and
/// named when it is wrong, and every other key is left alone.
struct HealthObject {
  private let fields: [String: Any]

  init(_ data: Data) throws {
    guard let top = try? JSONSerialization.jsonObject(with: data, options: []),
      let object = top as? [String: Any]
    else { throw HealthFault.undecodable("not one JSON object") }
    fields = object
  }

  private init(_ fields: [String: Any]) {
    self.fields = fields
  }

  /// A field that is there and is not null. The contract leaves a field out
  /// rather than printing an empty one, so null is absent either way.
  private func present(_ name: String) -> Any? {
    guard let value = fields[name], !(value is NSNull) else { return nil }
    return value
  }

  func object(_ name: String) -> HealthObject? {
    guard let inside = present(name) as? [String: Any] else { return nil }
    return HealthObject(inside)
  }

  func string(_ name: String, or fault: String) throws -> String? {
    guard let value = present(name) else { return nil }
    guard let text = value as? String else { throw HealthFault.undecodable(fault) }
    return text
  }

  func bool(_ name: String, or fault: String) throws -> Bool? {
    guard let value = present(name) else { return nil }
    guard let number = value as? NSNumber, CFGetTypeID(number) == CFBooleanGetTypeID() else {
      throw HealthFault.undecodable(fault)
    }
    return number.boolValue
  }

  func strings(_ name: String, or fault: String) throws -> [String] {
    guard let value = present(name) else { return [] }
    guard let list = value as? [String] else { throw HealthFault.undecodable(fault) }
    return list
  }
}
