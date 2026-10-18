// The header's own words: the version, the account, and the question.
//
// Both of these were HEALTH rows until 2026-10-18, at the foot of a panel
// that scrolls. Nail used the panel for a morning and said the account and
// its sign-out belong at the top; the version followed it, because the two
// things the header is about are which gdoc this is and who it is signed in
// as. The decision is docs/v2/DECISIONS.md, 2026-10-18, "The second morning".
//
// Nothing here asks anything: it is handed Facts and it words them. The one
// reading it makes is the glyph, which is the `auth` word in one shape.
//
// RowTests measures the line the pieces are drawn on, and StateTests states
// each of these states out of a set of facts.

import Foundation
import PresenceCore

extension PanelModel {
  /// The header, out of the machine's facts and what the person asked.
  static func header(
    facts: Facts, ui: UIState, now: Date, stopping: Stopping
  ) -> HeaderModel {
    let banner: BannerModel? =
      if let account = facts.account,
        SignedInBanner.isShowing(signedInAt: facts.signedInAt, now: now)
      {
        BannerModel(account: account)
      } else {
        nil
      }

    // The question is asked in place of the line. The pair of buttons takes
    // the right-hand side, and a version and an address beside them is one
    // line asking one thing and saying two others.
    if ui.isAsking(.signOut, now: now) {
      return HeaderModel(
        mark: facts.mark,
        question: QuestionModel(
          title: "Sign out?", consequence: stopping.sentence, danger: .signOut),
        quiet: [accountChanged(facts)].compactMap { $0 },
        banner: banner)
    }

    let version = versionPieces(facts)
    let account = accountPieces(facts)
    return HeaderModel(
      mark: facts.mark,
      version: version.text,
      versionTone: version.tone,
      versionWaiting: version.waiting,
      update: version.update,
      account: account.text,
      glyph: account.glyph,
      tone: account.tone,
      waiting: account.waiting,
      actions: account.actions,
      quiet: [account.quiet, version.quiet].compactMap { $0 },
      signInURL: account.signInURL,
      banner: banner)
  }

  /// What the header says about which gdoc this is.
  private struct VersionPieces {
    var text: String?
    var tone: GlyphTone = .dim
    var waiting = false
    var update: ActionModel?
    var quiet: String?
  }

  /// What the header says about the account.
  private struct AccountPieces {
    var text: String?
    var glyph: GlyphKind = .blank
    var tone: GlyphTone = .quiet
    var waiting = false
    var actions: [ActionModel] = []
    var quiet: String?
    var signInURL: String?
  }

  private static func versionPieces(_ facts: Facts) -> VersionPieces {
    guard facts.gdocFound else {
      return VersionPieces(
        text: "not found", tone: .bad, quiet: "looked for it at \(facts.gdocPathTried)")
    }
    // The binary is there and `help --json` has not answered yet, which is
    // every panel's first draw: the ring says the question is out rather
    // than claiming a version nobody read.
    guard let version = facts.versions.gdoc else {
      return VersionPieces(waiting: true)
    }
    // While an update child is up the ring is beside the version still
    // installed: there is nothing to press until the end arrives, and
    // nothing honest to say about what will be there. It is either child,
    // because `update --desktop` installs the release it writes the
    // extension of.
    if facts.updating {
      return VersionPieces(text: version, waiting: true)
    }
    guard let latest = facts.versions.updateAvailable else {
      return VersionPieces(text: version)
    }
    return VersionPieces(
      text: version,
      update: ActionModel(.update, icon: .down, tip: "Update to \(latest)", target: "gdoc"))
  }

  private static func accountPieces(_ facts: Facts) -> AccountPieces {
    if let phase = facts.signIn {
      return AccountPieces(
        text: "Waiting for Google", waiting: true,
        actions: [
          ActionModel(.copySignInLink, icon: .link, tip: Tips.copySignIn, target: "auth"),
          ActionModel(.cancelSignIn, icon: .x, tip: Tips.cancel, target: "auth"),
        ],
        quiet: "\(countdown(phase.secondsLeft)) left of the three minutes gdoc waits",
        signInURL: phase.url)
    }

    let changed = accountChanged(facts)

    // Nobody has asked yet, which is every panel's first draw and the whole
    // of a machine with no gdoc on it: the ring stands where the glyph is,
    // and there is no Sign out for a token nobody read.
    guard let auth = facts.auth else {
      return AccountPieces(text: facts.account, waiting: true, quiet: changed)
    }

    guard auth.isWorking else {
      return AccountPieces(
        text: facts.account ?? auth.rawValue, glyph: .exclamation, tone: .bad,
        actions: [ActionModel(.signIn, icon: .signin, tip: Tips.signIn, target: "auth")],
        quiet: changed)
    }

    return AccountPieces(
      text: facts.account ?? Sentences.accountNotRead, glyph: .check, tone: .ok,
      actions: [ActionModel(.signOut, icon: .signout, tip: Tips.signOut, target: "auth")],
      quiet: changed)
  }

  /// Both addresses or neither: the memory of the one before is kept for
  /// the life of the app, so a sign-out would otherwise leave the line
  /// "Account changed: someone@example.com → " with nothing after the
  /// arrow: testASignOutLeavesNoHalfSentenceAboutTheAccount.
  private static func accountChanged(_ facts: Facts) -> String? {
    guard let previous = facts.previousAccount, let account = facts.account else { return nil }
    return "Account changed: \(previous) → \(account)"
  }

  /// The sign-in's own countdown, as a clock rather than as a number of
  /// seconds: 2:43 is read at a glance and 163 s is counted.
  static func countdown(_ seconds: Int) -> String {
    let left = max(0, seconds)
    return String(format: "%d:%02d", left / 60, left % 60)
  }

  /// What signing out would stop, which is what the question names.
  struct Stopping: Equatable {
    let live: Int
    let holds: Int

    /// The sentence under the question. Nothing running is said as plainly
    /// as two watches are: a person pressing Sign out is owed the
    /// consequence either way.
    var sentence: String {
      var parts: [String] = []
      if live > 0 { parts.append("\(live) live \(live == 1 ? "watch" : "watches")") }
      if holds > 0 { parts.append("\(holds) \(holds == 1 ? "hold" : "holds")") }
      guard !parts.isEmpty else { return "Nothing is running. Only the token goes." }
      return parts.joined(separator: " and ") + " will stop."
    }
  }

  /// It counts the holds the record has open and not the cards NOW draws: a
  /// dismissed card is a card nobody can see and a hold the sign-out still
  /// drops, and a hold whose life ran out is waiting for nobody, so it is
  /// not one the sign-out stops either:
  /// testTheSignOutCountsTheHoldsTheRecordHasOpen.
  static func stopping(_ fold: NowFold, sessions: [LiveSession]) -> Stopping {
    Stopping(
      live: sessions.filter(\.isOpen).count,
      holds: fold.everyHold.filter { !$0.ranOut }.count)
  }
}
