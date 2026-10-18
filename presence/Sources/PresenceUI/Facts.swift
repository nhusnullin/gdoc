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
// printed, `network` is what a command answered, `pairing` is what the
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

  /// Whether the token is good for the work. Everything else is the header
  /// asking the person for something, and a machine with no token at all is
  /// why SETTINGS is pinned above everything under it.
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

/// Where the Claude Desktop extension stands, as far as this app can know.
///
/// `gdoc update --desktop` writes `gdoc.mcpb` beside the binary and hands it
/// to Claude Desktop. The file being absent is proof that gdoc never wrote
/// one on this machine, so Claude Desktop cannot have it. The file being
/// there proves nothing about Claude Desktop, which this app may not ask:
/// nothing published says where Desktop keeps what it loaded. So there is no
/// "connected", and the row is left out rather than guessed at.
/// docs/backlog/the-panel-cannot-see-the-desktop-extension.md.
public enum ExtensionState: Sendable, Equatable {
  /// No extension Claude Desktop can have taken: no `gdoc.mcpb` beside the
  /// binary, or a run of this app wrote one and could not hand it over.
  case notInstalled
  /// It was written. Whether Desktop loaded it is not a question this app
  /// has an answer to, and nothing is drawn.
  case written
  /// Nothing was looked for, because no gdoc was found at all.
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

/// Whether Google answered the last command that tried. It changes no icon
/// in the menu bar, by decision: a watch that cannot reach Google is still a
/// watch.
///
/// It is read off the record and never measured here: this app opens no
/// connection, so the only thing it can know about the network is what gdoc
/// wrote down. The fold that reads it is `PresenceCore.Reachability`, beside
/// every other fold of the record; this is the value it hands back. There is
/// no sentence with it, because the record carries one word and a sentence
/// here would be this app guessing at what that word meant.
public struct NetworkFact: Sendable, Equatable {
  public let reachable: Bool

  public init(reachable: Bool) {
    self.reachable = reachable
  }
}

/// The three version strings the header can say.
///
/// The skills are not among them. Nothing gdoc prints names a skills
/// version, and the folders are symlinks into somebody's `~/.claude`, which
/// this app does not read:
/// docs/backlog/the-panel-cannot-see-the-skills-version.md.
public struct Versions: Sendable, Equatable {
  /// The app's own, off its plist.
  public let app: String
  /// The installed binary's, from `help --json`. Nothing where gdoc was not
  /// found at all.
  public let gdoc: String?
  /// The newest stable release gdoc's own daily check heard about. It is a
  /// fact and not a judgement: gdoc prints what is published whatever is
  /// installed, so it may name a release older than this machine's.
  public let latestStable: String?

  public init(app: String, gdoc: String? = nil, latestStable: String? = nil) {
    self.app = app
    self.gdoc = gdoc
    self.latestStable = latestStable
  }

  /// Whether the row offers an update: a published release that is newer
  /// than the one installed.
  ///
  /// It is the three numbers and never the two strings. A nightly is an
  /// ordinary `x.y.z` tag whose patch is not zero, and this repository cuts
  /// one after every merge, so a machine on v2.14.1 against a published
  /// v2.14.0 would be offered an update that does not exist and a button
  /// `gdoc update` answers with "up to date".
  /// testAnOlderPublishedReleaseIsNoUpdate.
  public var updateAvailable: String? {
    guard let latestStable, let gdoc else { return nil }
    guard let published = Release(latestStable), let installed = Release(gdoc) else {
      // A version neither side can read is a string, and two strings that
      // differ are the most that can honestly be said about them.
      return latestStable == gdoc ? nil : latestStable
    }
    return published > installed ? latestStable : nil
  }
}

/// A release tag as the three numbers it is, for the one comparison the
/// header makes. It is `go/internal/update`'s own rule: `vX.Y.Z`, nothing
/// else.
struct Release: Comparable {
  let major: Int
  let minor: Int
  let patch: Int

  init?(_ tag: String) {
    let parts = tag.hasPrefix("v") ? String(tag.dropFirst()) : tag
    let numbers = parts.split(separator: ".", omittingEmptySubsequences: false)
    guard numbers.count == 3 else { return nil }
    guard let major = Int(numbers[0]), let minor = Int(numbers[1]), let patch = Int(numbers[2])
    else { return nil }
    self.major = major
    self.minor = minor
    self.patch = patch
  }

  static func < (left: Release, right: Release) -> Bool {
    (left.major, left.minor, left.patch) < (right.major, right.minor, right.patch)
  }
}

/// Everything about the machine the panel draws beside the record.
public struct Facts: Sendable, Equatable {
  /// The signed-in address, or nothing where the token carries none.
  public let account: String?
  /// What `gdoc auth status` last said, or nothing because nobody has asked
  /// yet. A machine with no gdoc on it is asked nothing at all, so this
  /// stays nothing there: the panel draws a ring over the row rather than a
  /// green check over a token nobody read.
  /// testTheAccountRowClaimsNothingBeforeTheFirstAnswer.
  public let auth: AuthWord?
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
  public let extensionState: ExtensionState
  /// What the last command said about reaching Google, or nothing.
  public let network: NetworkFact?
  public let loginItem: LoginItemState
  /// A sign-in in flight.
  public let signIn: SignInPhase?
  /// This machine's home directory, which is the one thing a note's path
  /// needs to be written with a tilde. It is handed over like every other
  /// fact here, so a render of a scene says the same thing on every
  /// machine, and it is empty where nobody asked.
  public let home: String
  /// A document id to the note paired with it, as an absolute path. Since
  /// the third look it is what the finder answered, with the `hubPath`
  /// override's own notes first.
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
  /// The documents whose live watch the app may not signal, because the
  /// kernel's answer about that pid did not agree with the record. It is
  /// `Stopper.unstoppable`, asked once per refold, and it greys one icon.
  public let unstoppable: Set<String>
  /// Whether the app's own `gdoc update` child is still up. It draws the
  /// ring in the header, and the version beside it stays the installed one
  /// until the child's end says what was put there.
  public let updating: Bool
  /// When an update run of the app's own came back having installed
  /// nothing because this is the newest release. The glyph draws the check
  /// for a second from that moment, which is the copied link's own second:
  /// a press that did something is owed an answer, and "nothing to do" is
  /// an answer.
  public let updateChecked: Date?

  public init(
    account: String? = nil,
    auth: AuthWord? = nil,
    previousAccount: String? = nil,
    versions: Versions,
    mark: String = "⁘",
    gdocPath: String? = nil,
    gdocPathTried: String = "~/.local/bin/gdoc",
    home: String = "",
    extensionState: ExtensionState = .unknown,
    network: NetworkFact? = nil,
    loginItem: LoginItemState = .off,
    signIn: SignInPhase? = nil,
    pairing: [String: String] = [:],
    signedInAt: Date? = nil,
    bundle: BundleState = .same,
    newerRecord: Bool = false,
    unstoppable: Set<String> = [],
    updating: Bool = false,
    updateChecked: Date? = nil
  ) {
    self.account = account
    self.auth = auth
    self.previousAccount = previousAccount
    self.versions = versions
    self.mark = mark
    self.gdocPath = gdocPath
    self.gdocPathTried = gdocPathTried
    self.home = home
    self.extensionState = extensionState
    self.network = network
    self.loginItem = loginItem
    self.signIn = signIn
    self.pairing = pairing
    self.signedInAt = signedInAt
    self.bundle = bundle
    self.newerRecord = newerRecord
    self.unstoppable = unstoppable
    self.updating = updating
    self.updateChecked = updateChecked
  }

  /// What the app knows before it has asked anything: its own version, and
  /// nothing else. The shell starts here and the pollers fill it in.
  public static func launching(appVersion: String) -> Facts {
    Facts(versions: Versions(app: appVersion))
  }

  /// Whether gdoc itself was there to be asked. It is the header's own
  /// question and the menu bar's: an app that cannot find the binary can do
  /// nothing at all, which is one of the three things that are stuck without
  /// a line in the record.
  ///
  /// The path alone answers it, because the path is the location's own fact
  /// and the version is an answer that has to come back. The two questions
  /// are asked when the panel opens, so a version read as a not-found would
  /// badge the menu bar from login until the first click:
  /// testTheBarIsNotStuckBeforeTheFirstAnswer.
  public var gdocFound: Bool { gdocPath != nil }

  /// The note paired with a document, as an absolute path.
  public func note(for doc: String?) -> String? {
    guard let doc else { return nil }
    return pairing[doc]
  }
}
