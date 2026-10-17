// Update, and Install extension.
//
// The button says "Update" and not "Check for updates", because this app
// never asks GitHub what is published. `gdoc help --json` carries gdoc's own
// once-a-day check, the HEALTH row says what that check heard, and pressing
// Update runs `update` with no flag at all: what the person would have typed,
// and nothing the app invented. A flag here would be this app choosing a
// channel or a major on somebody's behalf.
//
// While the child is up the gdoc row draws a ring in its glyph's place, and
// the version it names is the one still installed: a row that said the new
// number before the bytes were in place would be a row that lied for a few
// seconds. Only `action: "updated"` moves it, because that is the one word
// the command prints for a run that replaced something; `major_available` is
// a version this app declined to install by passing no flag.
//
// Install extension is the second list, `update --desktop`. It is not a new
// gdoc, so it moves no version and the gdoc row's ring is not its.

import Foundation
import PresenceCore
import PresenceUI

/// Where an update run stands.
public enum UpdateStanding: Sendable, Equatable {
  case idle
  /// The child is up, which is the ring on the gdoc row.
  case running
  /// It replaced the binary, with the version it put there.
  case installed(String)
  /// It finished and installed nothing, carrying the word it printed.
  case unchanged(String)
  /// Nothing was installed and the reason is the envelope's own.
  case failed(String)
}

/// The two runs behind the HEALTH rows: the update, and the extension.
///
/// An actor because the standing is read on the main thread while the child
/// is answering off it. The child comes from a closure, which is the runner's
/// own `run` in the app and an object written in a test under test.
public actor UpdateAction {
  /// How a call is answered. The app hands over the runner's own `run`.
  public typealias Answer = @Sendable (Call) async throws -> Outcome

  /// Where the gdoc row stands. Install extension never moves it.
  public private(set) var standing: UpdateStanding = .idle

  private let answer: Answer

  public init(answer: @escaping Answer) {
    self.answer = answer
  }

  /// Run `update`, with no flag, and hand back where the row ends up.
  public func update() async -> UpdateStanding {
    standing = .running
    standing = await ran(.update)
    return standing
  }

  /// Run `update --desktop`, which writes the Claude Desktop extension and
  /// hands it to Desktop. Whether Desktop then installs it is not something
  /// this app is told, so no fact here changes.
  public func installExtension() async -> UpdateStanding {
    await ran(.updateDesktop)
  }

  /// The panel's facts with the run written into them: the ring while the
  /// child is up, and the installed version once it is there.
  ///
  /// Every other field is the one it was handed. The newest release stays as
  /// the daily check reported it, so a version that matches it stops
  /// offering the update by itself.
  public func applied(to facts: Facts) -> Facts {
    Facts(
      account: facts.account,
      auth: facts.auth,
      previousAccount: facts.previousAccount,
      versions: Versions(
        app: facts.versions.app,
        gdoc: installed ?? facts.versions.gdoc,
        latestStable: facts.versions.latestStable,
        skills: facts.versions.skills),
      mark: facts.mark,
      gdocPath: facts.gdocPath,
      gdocPathTried: facts.gdocPathTried,
      hub: facts.hub,
      extensionState: facts.extensionState,
      network: facts.network,
      loginItem: facts.loginItem,
      signIn: facts.signIn,
      pairing: facts.pairing,
      signedInAt: facts.signedInAt,
      bundle: facts.bundle,
      newerRecord: facts.newerRecord,
      unstoppable: facts.unstoppable,
      updating: standing == .running)
  }

  /// The version this app installed, where it installed one.
  private var installed: String? {
    guard case .installed(let version) = standing else { return nil }
    return version
  }

  /// One child, and what its one object says about what it did.
  private func ran(_ call: Call) async -> UpdateStanding {
    guard let outcome = try? await answer(call) else {
      return .failed(UpdateWords.didNotRun)
    }
    guard outcome.envelope.ok else {
      return .failed(outcome.envelope.error ?? UpdateWords.didNotRun)
    }
    guard let data = outcome.envelope.data, let report = try? UpdateReport.decode(data) else {
      return .failed(UpdateWords.answerUnread)
    }
    guard report.action == UpdateReport.updated else {
      return .unchanged(report.action ?? "")
    }
    guard let version = report.to else { return .failed(UpdateWords.answerUnread) }
    return .installed(version)
  }
}

/// What `gdoc update` printed, as the two fields the panel reads.
///
/// Strict, by the rule the record's own decoder follows: a field a newer gdoc
/// prints is read past, and one this app knows that holds the wrong kind of
/// value is refused, because something other than gdoc wrote it.
struct UpdateReport: Equatable {
  /// The one word that means something was replaced.
  static let updated = "updated"

  let action: String?
  let to: String?

  static func decode(_ data: Data) throws -> UpdateReport {
    guard let top = try? JSONSerialization.jsonObject(with: data, options: []),
      let object = top as? [String: Any]
    else { throw UpdateFault.undecodable("the update's data is not an object") }
    return UpdateReport(action: try string("action", in: object), to: try string("to", in: object))
  }

  private static func string(_ name: String, in object: [String: Any]) throws -> String? {
    guard let held = object[name] else { return nil }
    guard let text = held as? String else {
      throw UpdateFault.undecodable("\(name) is not a string")
    }
    return text
  }
}

/// Which strict reading of the update's answer did not hold. The panel shows
/// none of these words; a test names the rule it is holding.
enum UpdateFault: Error, Equatable {
  case undecodable(String)
}

/// The two sentences this room can put in front of somebody. Neither names a
/// document, because an update is about this machine.
enum UpdateWords {
  static let didNotRun = "the update did not run"
  static let answerUnread = "the update answered something gdoc would not print"
}
