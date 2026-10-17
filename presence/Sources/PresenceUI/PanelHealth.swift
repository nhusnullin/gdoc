// HEALTH: the last section, and the one that is about the machine.
//
// Every row here is a fact somebody else produced. The account and the
// versions are `gdoc auth status` and `gdoc help --json`, the network is the
// record's own word for a run that did not reach Google, the extension is a
// file beside the binary and what the run that wrote it said, the mark is a
// setting and the switch is this machine's login items. Nothing in this file
// asks anything: it is handed Facts and it words them.
//
// A row with nothing to say is left out, and the rest keep their places: a
// section whose rows move is a section nobody learns. The order is
// HealthRowKind's and never this file's.

import Foundation
import PresenceCore

extension PanelModel {
  /// The rows, in the order `HealthRowKind` lists them.
  ///
  /// A row with nothing to say is left out and the rest keep their places.
  /// When every fact is good the list is the short one, because a screen of
  /// green rows is a screen nobody reads; the two that are not facts stay,
  /// so Quit is always one click away.
  ///
  /// The Claude Desktop row is not one of the green ones: it is drawn only
  /// where Claude Desktop can have no extension, which is no `gdoc.mcpb`
  /// beside the binary or a run of this process that wrote one and could
  /// not hand it over, and it carries the one thing to do about that, so it
  /// does not wait for something else to go wrong first:
  /// testTheExtensionRowAppearsOnlyUntilGdocHasWrittenOne and
  /// testARunThatOpenedNothingLeavesTheRowWhereItWas.
  static func healthRows(
    facts: Facts, ui: UIState, now: Date, stopping: Stopping
  ) -> [HealthRowModel] {
    var rows: [HealthRowModel] = [account(facts: facts, ui: ui, now: now, stopping: stopping)]

    if let network = facts.network, !network.reachable {
      rows.append(
        HealthRowModel(
          .network, value: "Google not reachable", glyph: .exclamation, tone: .bad))
    }

    rows.append(gdoc(facts: facts))
    rows.append(contentsOf: desktop(facts: facts))
    rows.append(HealthRowModel(.mark, value: facts.mark))
    rows.append(loginItem(facts: facts))
    rows.append(HealthRowModel(.quit, value: HealthRows.quitShortcut, select: .quit))
    return rows
  }

  /// What signing out would stop, which is what the confirm names.
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

  static func account(
    facts: Facts, ui: UIState, now: Date, stopping: Stopping
  ) -> HealthRowModel {
    if let phase = facts.signIn {
      return HealthRowModel(
        .account, value: "Waiting for Google", waiting: true,
        quiet: "\(countdown(phase.secondsLeft)) left of the three minutes gdoc waits",
        actions: [
          ActionModel(.copySignInLink, icon: .link, tip: Tips.copySignIn, target: "auth"),
          ActionModel(.cancelSignIn, icon: .x, tip: Tips.cancel, target: "auth"),
        ])
    }

    // Both addresses or neither: the memory of the one before is kept for
    // the life of the app, so a sign-out would otherwise leave the line
    // "Account changed: someone@example.com → " with nothing after the
    // arrow: testASignOutLeavesNoHalfSentenceAboutTheAccount.
    let changed: String? =
      if let previous = facts.previousAccount, let account = facts.account {
        "Account changed: \(previous) → \(account)"
      } else {
        nil
      }

    if ui.isAsking(.signOut, now: now) {
      return HealthRowModel(
        .account, glyph: .empty, quiet: changed,
        question: QuestionModel(
          title: "Sign out?", consequence: stopping.sentence, danger: .signOut))
    }

    // Nobody has asked yet, which is every panel's first draw and the whole
    // of a machine with no gdoc on it: the row is a ring, as the gdoc row
    // is, and it offers no Sign out for a token nobody read.
    guard let auth = facts.auth else {
      return HealthRowModel(.account, value: facts.account ?? "", waiting: true, quiet: changed)
    }

    guard auth.isWorking else {
      return HealthRowModel(
        .account, value: facts.account ?? auth.rawValue, glyph: .exclamation, tone: .bad,
        quiet: changed,
        actions: [ActionModel(.signIn, icon: .signin, tip: Tips.signIn, target: "auth")])
    }

    return HealthRowModel(
      .account, value: facts.account ?? Sentences.accountNotRead, glyph: .check, tone: .ok,
      quiet: changed,
      actions: [ActionModel(.signOut, icon: .signout, tip: Tips.signOut, target: "auth")])
  }

  /// The sign-in's own countdown, as a clock rather than as a number of
  /// seconds: 2:43 is read at a glance and 163 s is counted.
  static func countdown(_ seconds: Int) -> String {
    let left = max(0, seconds)
    return String(format: "%d:%02d", left / 60, left % 60)
  }

  static func gdoc(facts: Facts) -> HealthRowModel {
    guard facts.gdocFound else {
      return HealthRowModel(
        .gdoc, value: "not found", glyph: .exclamation, tone: .bad,
        quiet: "looked for it at \(facts.gdocPathTried)")
    }
    // The binary is there and `help --json` has not answered yet, which is
    // every panel's first draw: the ring says the question is out rather
    // than claiming a version nobody read or a binary nobody found.
    guard let version = facts.versions.gdoc else {
      return HealthRowModel(.gdoc, waiting: true)
    }
    // While an update child is up the row is a ring and the version it
    // names is the one still installed: there is nothing to press until the
    // end arrives, and nothing honest to say about what will be there. It is
    // either child, because `update --desktop` installs the release it
    // writes the extension of.
    if facts.updating {
      return HealthRowModel(.gdoc, value: version, waiting: true)
    }
    guard let latest = facts.versions.updateAvailable else {
      return HealthRowModel(.gdoc, value: version, glyph: .check, tone: .ok)
    }
    return HealthRowModel(
      .gdoc, value: "\(version) · \(latest) available", glyph: .check, tone: .ok,
      actions: [
        ActionModel(.update, icon: .down, tip: "Update to \(latest)", target: "gdoc")
      ])
  }

  /// The Claude Desktop row, which is drawn only when the app can prove
  /// Claude Desktop has no extension to load: no `gdoc.mcpb` beside the
  /// binary means gdoc never wrote one, and a run of this process that
  /// wrote one and could not hand it over is the same thing with a file
  /// left behind. A file that is there says nothing about what Desktop
  /// loaded, and a row claiming it was connected would be a guess.
  static func desktop(facts: Facts) -> [HealthRowModel] {
    guard facts.extensionState == .notInstalled else { return [] }
    // While an update child is up the row offers nothing to press. Both
    // argv lists are `gdoc update` and `--desktop` installs the release it
    // writes the extension of, so a second press is a second process
    // renaming the binary the first one is replacing:
    // testNeitherUpdateIsOfferedWhileOneIsRunning.
    guard !facts.updating else {
      return [HealthRowModel(.desktop, value: "not installed", glyph: .empty)]
    }
    return [
      HealthRowModel(
        .desktop, value: "not installed", glyph: .empty,
        actions: [
          ActionModel(
            .installExtension, icon: .down, tip: Tips.installExtension, target: "desktop")
        ])
    ]
  }

  static func loginItem(facts: Facts) -> HealthRowModel {
    switch facts.loginItem {
    case .on, .off:
      return HealthRowModel(
        .launchAtLogin, isOn: facts.loginItem == .on, select: .toggleLoginItem)
    case .needsApproval:
      return HealthRowModel(
        .launchAtLogin, quiet: Sentences.loginItemApproval, isOn: false,
        select: .openLoginItems)
    }
  }
}
