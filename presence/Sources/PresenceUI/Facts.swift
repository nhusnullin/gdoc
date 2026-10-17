// What the panel knows that the record does not say.
//
// The record is the whole of what gdoc did. It says nothing about which
// account is signed in, which version is installed, whether a note is paired
// with a document, or whether a sign-in is in flight right now. Those are
// facts about the machine, and this file is the shape they arrive in.
//
// Every field is a plain value. Nothing here runs a command, reads a file or
// asks the kernel: Tasks 15 to 21 produce these values and hand them over, so
// every state the panel can be in is reachable from a literal in a test. That
// is the whole reason the type exists.
//
// A fact is never a judgement either. `auth` is the word `gdoc auth status`
// printed, `network` is what a command answered, `pairing` is what the hub's
// front matter said. The panel's reading of them is in PanelModel.

import Foundation
import PresenceCore

/// What `gdoc auth status` says about the token, in the three words the
/// specification fixed, and the fourth for a machine that has no token at
/// all.
public enum AuthWord: String, Sendable, Equatable, CaseIterable {
  case ok = "OK"
  case needsSignIn = "Needs sign-in"
  case missingScopes = "Missing scopes"
  case signedOut = "Not signed in"

  /// Whether the token is good for the work. Everything else is a row a
  /// person has to answer, which is why HEALTH is pinned first under it.
  public var isWorking: Bool { self == .ok }
}

/// A sign-in that is running: the URL the child printed, and what is left of
/// the child's own patience.
public struct SignInPhase: Equatable, Sendable {
  public let url: String
  /// The countdown, in whole seconds. It is the child's three minutes and
  /// never a guess at how long Google will take.
  public let secondsLeft: Int

  public init(url: String, secondsLeft: Int) {
    self.url = url
    self.secondsLeft = secondsLeft
  }
}

/// Whether the Claude Desktop extension is connected.
public enum ExtensionState: Sendable, Equatable {
  case connected
  case notInstalled
  /// Nothing was asked, so the row is left out rather than guessed at.
  case unknown
}

/// Where launch at login stands.
public enum LoginItemState: Sendable, Equatable {
  case on
  case off
  /// macOS wants the person to allow it in Login Items, which is a row and
  /// never a retry.
  case needsApproval
}

/// Whether Google answered. It changes no icon in the menu bar, by decision:
/// a watch that cannot reach Google is still a watch.
public struct NetworkFact: Sendable, Equatable {
  public let reachable: Bool
  /// What was seen, in the panel's own words, or nothing.
  public let detail: String?

  public init(reachable: Bool, detail: String? = nil) {
    self.reachable = reachable
    self.detail = detail
  }
}

/// The four version strings HEALTH can say.
public struct Versions: Sendable, Equatable {
  /// The app's own, off its plist.
  public let app: String
  /// The installed binary's, from `help --json`. Nothing where gdoc was not
  /// found at all.
  public let gdoc: String?
  /// The newest release gdoc's own daily check heard about, where it is not
  /// the installed one.
  public let latestStable: String?
  /// The skills, and where they came from.
  public let skills: String?

  public init(app: String, gdoc: String? = nil, latestStable: String? = nil, skills: String? = nil)
  {
    self.app = app
    self.gdoc = gdoc
    self.latestStable = latestStable
    self.skills = skills
  }

  /// Whether the row offers an update: a release that is named and is not
  /// the one installed.
  public var updateAvailable: String? {
    guard let latestStable, let gdoc, latestStable != gdoc else { return nil }
    return latestStable
  }
}

/// Everything about the machine the panel draws beside the record.
public struct Facts: Sendable, Equatable {
  /// The signed-in address, or nothing where the token carries none.
  public let account: String?
  public let auth: AuthWord
  /// The address the app saw before this one, which is its own memory and
  /// never a line of the log.
  public let previousAccount: String?
  public let versions: Versions
  /// The mark in effect, a setting of the binary's. It may be empty.
  public let mark: String
  /// The resolved absolute path of the binary, or nothing where neither
  /// place held one.
  public let gdocPath: String?
  /// Where it was looked for, which is what the not-found row says.
  public let gdocPathTried: String
  /// The hub folder the person chose, or nothing while none is chosen.
  public let hub: String?
  public let extensionState: ExtensionState
  /// What the last command said about reaching Google, or nothing.
  public let network: NetworkFact?
  public let loginItem: LoginItemState
  /// A sign-in in flight.
  public let signIn: SignInPhase?
  /// The hub index: a document id to the note paired with it, as a path
  /// relative to the hub.
  public let pairing: [String: String]
  /// When this account signed in, which the banner's five seconds run from.
  public let signedInAt: Date?
  /// Whether the bundle under the running app is still the one it launched
  /// from.
  public let bundle: BundleState
  /// Whether the record has held a line from a gdoc at a schema this app
  /// does not read. The line was skipped, so the one thing left to say is
  /// that it was there.
  public let newerRecord: Bool

  public init(
    account: String? = nil,
    auth: AuthWord = .ok,
    previousAccount: String? = nil,
    versions: Versions,
    mark: String = "⁘",
    gdocPath: String? = nil,
    gdocPathTried: String = "~/.local/bin/gdoc",
    hub: String? = nil,
    extensionState: ExtensionState = .unknown,
    network: NetworkFact? = nil,
    loginItem: LoginItemState = .on,
    signIn: SignInPhase? = nil,
    pairing: [String: String] = [:],
    signedInAt: Date? = nil,
    bundle: BundleState = .same,
    newerRecord: Bool = false
  ) {
    self.account = account
    self.auth = auth
    self.previousAccount = previousAccount
    self.versions = versions
    self.mark = mark
    self.gdocPath = gdocPath
    self.gdocPathTried = gdocPathTried
    self.hub = hub
    self.extensionState = extensionState
    self.network = network
    self.loginItem = loginItem
    self.signIn = signIn
    self.pairing = pairing
    self.signedInAt = signedInAt
    self.bundle = bundle
    self.newerRecord = newerRecord
  }

  /// What the app knows before it has asked anything: its own version, and
  /// nothing else. The shell starts here and the pollers fill it in.
  public static func launching(appVersion: String) -> Facts {
    Facts(auth: .ok, versions: Versions(app: appVersion))
  }

  /// The note paired with a document, as a path inside the hub.
  public func note(for doc: String?) -> String? {
    guard let doc else { return nil }
    return pairing[doc]
  }
}
