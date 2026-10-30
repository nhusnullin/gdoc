// Update, and Install extension.
//
// The button says "Update" and not "Check for updates", because this app
// never asks GitHub what is published. `gdoc help --json` carries gdoc's own
// once-a-day check, the header says what that check heard, and pressing
// Update runs `update` with no flag at all: what the person would have typed,
// and nothing the app invented. A flag here would be this app choosing a
// channel or a major on somebody's behalf.
//
// While the child is up the header draws a ring beside the version, and the
// version it names is the one still installed: a line that said the new
// number before the bytes were in place would be a line that lied for a few
// seconds. Only `action: "updated"` moves it, because that is the one word
// the command prints for a run that replaced something; `major_available` is
// a version this app declined to install by passing no flag. It moves the row
// until the next reading disagrees with it and no longer: the row belongs to
// `help --json`, and this run's answer stands in for one stale reading rather
// than for the rest of the process.
//
// Install extension is the second list, `update --desktop`. It is not a new
// gdoc, so it moves no version: the standing stays idle and the row's number
// is the one `help --json` read. It hands back a standing of its own all the
// same, because since the fourth look a run that handed Claude Desktop
// nothing says so on the row's quiet line.
//
// It does take the same slot, though, and while either child is up both rows
// wait. `update --desktop` installs the release it writes the extension of,
// so two of them at once are two processes staging and renaming one binary,
// and a button offered under the one that is running is an offer to do that.

import Foundation
import PresenceCore
import PresenceUI

/// Where an update run stands.
public enum UpdateStanding: Sendable, Equatable {
  case idle
  /// The child is up, which is the ring beside the header's version.
  case running
  /// It replaced the binary, with the version it put there.
  case installed(String)
  /// It finished and installed nothing, carrying the word it printed.
  case unchanged(String)

  /// Whether the run answered that this machine is on the newest release.
  /// It is the one word of the six that says so: `major_available` is a
  /// release this app declined to install by passing no flag, and a glyph
  /// that showed a check for it would be saying there is nothing to take.
  public var isCurrent: Bool {
    self == .unchanged(UpdateReport.upToDate)
  }
  /// Nothing was installed and the reason is the envelope's own.
  case failed(String)

  /// What the header's quiet line says about this run, or nothing where the
  /// run needs no sentence: a machine already on the newest release has the
  /// check for a second, and one that installed something has the version.
  ///
  /// The reading is `ok` and the action word, and nothing else. `checked`
  /// and `rolled_back` are the two words the app never asks for, because
  /// neither flag is ever passed, so a run that printed one of them is not
  /// the run this button started: it falls to the general sentence with
  /// every other word this version has no case for.
  public var miss: UpdateMiss? {
    switch self {
    case .idle, .running, .installed: nil
    case .unchanged(let action):
      switch action {
      case UpdateReport.upToDate: nil
      case UpdateReport.majorAvailable: .major
      case UpdateReport.unreachable: .unreachable
      default: .stopped
      }
    case .failed: .stopped
    }
  }
}

/// Where a run of Install extension ended. It is the extension's own
/// standing and not the update's: no version moves either way.
public enum ExtensionStanding: Sendable, Equatable {
  /// gdoc wrote the extension and Claude Desktop was handed it.
  case handedOver
  /// The run ended `ok: true` and handed nothing over, carrying the word it
  /// printed.
  case unchanged(String)
  /// It refused, or answered something gdoc would not print.
  case stopped

  /// What the Claude Desktop row's quiet line says about it.
  public var miss: ExtensionMiss? {
    switch self {
    case .handedOver: nil
    case .unchanged(let action): action == UpdateReport.majorAvailable ? .major : .handedNothing
    case .stopped: .stopped
    }
  }
}

/// The two runs the panel offers: the update in the header, and the
/// extension in SETTINGS.
///
/// An actor because the standing is read on the main thread while the child
/// is answering off it. The child comes from a closure, which is the runner's
/// own `run` in the app and an object written in a test under test.
public actor UpdateAction {
  /// How a call is answered. The app hands over the runner's own `run`.
  public typealias Answer = @Sendable (Call) async throws -> Outcome

  /// Where the update stands. Install extension never moves it.
  public private(set) var standing: UpdateStanding = .idle

  private let answer: Answer
  /// The version the last run replaced, as that run reported it. It is what
  /// bounds the override in `installed(over:)` to the one reading it is for.
  private var replaced: String?

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
  /// hands it to Desktop, and say whether the handing over happened.
  ///
  /// Whether Desktop then installs it is not something this app is told, so
  /// no version and no standing moves. What it is told is `extension_opened`:
  /// the command writes the file and then opens it, and an open that failed
  /// is a warning with the file left where it is, by the house rule that
  /// nothing raises over a write that happened. So the file beside the
  /// binary is no proof that Desktop was ever offered one, and this answer
  /// is what the panel keeps instead:
  /// testTheExtensionRunSaysWhetherDesktopWasHandedIt.
  ///
  /// A run that handed nothing over says why on the row's quiet line since
  /// the fourth look, and the three cases are told apart by `ok` and the
  /// action word alone: a declined major, a refusal, and every other run
  /// that ended with `extension_opened` absent. The child's own warning is
  /// never the sentence.
  public func installExtension() async -> ExtensionStanding {
    guard let outcome = try? await answer(.updateDesktop), outcome.envelope.ok else {
      return .stopped
    }
    guard let data = outcome.envelope.data, let report = try? UpdateReport.decode(data) else {
      return .stopped
    }
    return report.extensionOpened ? .handedOver : .unchanged(report.action ?? "")
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
        gdoc: installed(over: facts.versions.gdoc) ?? facts.versions.gdoc,
        latestStable: facts.versions.latestStable),
      mark: facts.mark,
      logo: facts.logo,
      gdocPath: facts.gdocPath,
      gdocPathTried: facts.gdocPathTried,
      home: facts.home,
      extensionState: facts.extensionState,
      network: facts.network,
      loginItem: facts.loginItem,
      signIn: facts.signIn,
      pairing: facts.pairing,
      noteFaults: facts.noteFaults,
      signedInAt: facts.signedInAt,
      bundle: facts.bundle,
      newerRecord: facts.newerRecord,
      unstoppable: facts.unstoppable,
      // Either run closes both buttons, so this never puts the flag down:
      // the session raises it on the press, for the extension's run as well,
      // and this raises it again while the child of a plain update is up.
      updating: facts.updating || standing == .running,
      updateChecked: facts.updateChecked,
      updateRun: facts.updateRun,
      extensionRun: facts.extensionRun)
  }

  /// The version this app installed, while it is newer than the reading
  /// beside it.
  ///
  /// The override is for the second between the child's end and the next
  /// `help --json`, where the last reading still names the version that was
  /// replaced. It is bounded by that version: a reading naming anything else
  /// came after this run, which is what a `gdoc update` typed in a terminal
  /// later leaves behind. Unbounded, the row would name this run's version
  /// for the life of the process and offer an update for a release already
  /// installed: testTheInstalledVersionStopsOverridingALaterReading.
  private func installed(over reported: String?) -> String? {
    guard case .installed(let version) = standing else { return nil }
    guard let reported, reported != version else { return version }
    return reported == replaced ? version : nil
  }

  /// One child, and what its one object says about what it did.
  ///
  /// `.failed` and `.unchanged` carry the word the run printed, and `miss`
  /// above turns it into one of the three the header has a sentence for.
  /// The version stays where `help --json` drew it either way: a run that
  /// installed nothing moved nothing.
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
    replaced = report.installed
    return .installed(version)
  }
}

/// What `gdoc update` printed, as the three fields the panel reads.
///
/// Strict, by the rule the record's own decoder follows: a field a newer gdoc
/// prints is read past, and one this app knows that holds the wrong kind of
/// value is refused, because something other than gdoc wrote it.
struct UpdateReport: Equatable {
  /// The one word that means something was replaced.
  static let updated = "updated"
  /// The one word that means there was nothing to replace. It is
  /// `update.UpToDate` in the Go, printed by a run that reached GitHub and
  /// found this machine already on the newest release.
  static let upToDate = "up_to_date"
  /// A newer release across a major boundary, which gdoc names and never
  /// takes without `--major`, and the app passes no flag at all.
  static let majorAvailable = "major_available"
  /// GitHub did not answer. gdoc raises nothing for it, by its own rule
  /// that a person on a train is owed an answer: the binary they have is
  /// still the binary they have.
  static let unreachable = "unreachable"

  let action: String?
  let to: String?
  /// The version that was there when the run started, which is what the
  /// installed version is allowed to stand over.
  let installed: String?
  /// Whether the extension reached Claude Desktop. gdoc prints the field
  /// only where it did, so an absent field is a run that opened nothing,
  /// which is the same silence a run without `--desktop` leaves.
  let extensionOpened: Bool

  static func decode(_ data: Data) throws -> UpdateReport {
    guard let top = try? JSONSerialization.jsonObject(with: data, options: []),
      let object = top as? [String: Any]
    else { throw UpdateFault.undecodable("the update's data is not an object") }
    return UpdateReport(
      action: try string("action", in: object), to: try string("to", in: object),
      installed: try string("installed", in: object),
      extensionOpened: try flag("extension_opened", in: object) ?? false)
  }

  private static func string(_ name: String, in object: [String: Any]) throws -> String? {
    guard let held = object[name] else { return nil }
    guard let text = held as? String else {
      throw UpdateFault.undecodable("\(name) is not a string")
    }
    return text
  }

  /// A boolean held to being one, the way the envelope holds `ok`: a number
  /// or a string in its place is something other than gdoc writing the
  /// field, and `as? Bool` would take a 1 for a yes.
  private static func flag(_ name: String, in object: [String: Any]) throws -> Bool? {
    guard let held = object[name] else { return nil }
    guard let number = held as? NSNumber, CFGetTypeID(number) == CFBooleanGetTypeID() else {
      throw UpdateFault.undecodable("\(name) is not a boolean")
    }
    return number.boolValue
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
