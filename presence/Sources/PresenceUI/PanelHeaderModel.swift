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

    let version = versionPieces(facts, now: now)
    let account = accountPieces(facts)

    // The question is asked in place of the line. The pair of buttons takes
    // the right-hand side, and a version and an address beside them is one
    // line asking one thing and saying two others.
    //
    // It keeps the account's own glyph, as the hold card's question keeps
    // the hold's pause: a check that became an empty ring while the question
    // stood said the panel had stopped knowing whose account it is, at the
    // one moment a person is deciding about it.
    //
    // And it is the only line drawn under itself. The quiet line about a
    // changed address is something that already happened, and under "Sign
    // out?" in the same ink and at the same indent as the consequence it
    // read as a second consequence: press this and the account changes.
    // The banner of a sign-in five seconds old goes the same way, and it
    // carries a Sign out of its own: two of them under one question is the
    // panel asking twice. Both come back when the question is answered.
    if ui.isAsking(.signOut, now: now) {
      return HeaderModel(
        mark: facts.mark,
        glyph: account.glyph,
        tone: account.tone,
        waiting: account.waiting,
        question: QuestionModel(
          title: "Sign out?", consequence: stopping.sentence, danger: .signOut))
    }

    return HeaderModel(
      mark: facts.mark,
      version: version.text,
      versionTone: version.tone,
      versionWaiting: version.waiting,
      update: version.update,
      updateLook: version.look,
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
    var look: UpdateLook = .quiet
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

  /// What the header says about which gdoc this is, and what pressing the
  /// glyph beside it would do.
  ///
  /// The glyph is there whenever there is a binary to update, because
  /// `gdoc update` is the command that asks GitHub: a header that offered
  /// nothing until the app's own six hours had passed would make a person
  /// wait for a release they already know is out. What the states differ in
  /// is what this app has been told, never what the press does.
  /// testTheUpdateGlyphIsAlwaysThereInItsThreeStates.
  private static func versionPieces(_ facts: Facts, now: Date) -> VersionPieces {
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

    let latest = facts.versions.updateAvailable
    // The published version is the line under the header and not a second
    // number on it. Both of them on the line is what the decision of the
    // third look asks for, and it does not fit: 320 pt holds the mark, the
    // name, one version, the glyph, the account's own glyph, its address
    // and Sign out with 4 pt to spare, and a second version squeezes the
    // address from 98 pt to 49, which is "nail@ex…" on every screen. That
    // is the one thing the morning before this evening was about. So the
    // number is directly under the one installed, the dot on the glyph is
    // what catches an eye, and the address stays whole:
    // testTheHeaderIs36WithTheMarkTheNameTheVersionAndTheAccount measures
    // both. Nail may still want it on the line, and that is his word to
    // say.
    let look: UpdateLook = latest == nil ? .quiet : .available
    let published = latest.map { "\($0) is published" }

    // The second after a run that installed nothing. The check stands in
    // for the arrow, as a copied link's does, and it goes by itself.
    if ActionButton.isDone(copiedAt: facts.updateChecked, now: now) {
      return VersionPieces(
        text: version,
        update: ActionModel(
          .update, icon: .down, tip: Tips.newestRelease, state: .done, target: "gdoc"),
        look: look, quiet: published)
    }

    guard let latest else {
      return VersionPieces(
        text: version,
        update: ActionModel(.update, icon: .down, tip: Tips.checkUpdate, target: "gdoc"),
        look: .quiet)
    }
    return VersionPieces(
      text: version,
      update: ActionModel(.update, icon: .down, tip: Tips.updateTo(latest), target: "gdoc"),
      look: .available, quiet: published)
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
