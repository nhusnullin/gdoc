// SETTINGS: the last section, and the one a person comes to on purpose.
//
// The group was HEALTH until 2026-10-18. Three of its rows left that
// morning and the Hub row left that evening, and what is left is two facts
// about the machine and the two settings the panel has, so the label says
// what the rows are.
//
// Every fact here is one somebody else produced. The network is the
// record's own word for a run that did not reach Google, the extension is a
// file beside the binary, and the switch is this machine's login items.
// Nothing in this file asks anything: it is handed Facts and it words them.
//
// The account and the version are not here. They moved into the header on
// 2026-10-18, which is PanelHeaderModel.swift, and the Thread mark row went
// with them: the mark is drawn in that header, and a row repeating it is
// one fact in two places.
//
// A row with nothing to say is left out, and the rest keep their places: a
// section whose rows move is a section nobody learns. The order is
// SettingsRowKind's and never this file's.

import Foundation
import PresenceCore

extension PanelModel {
  /// The rows, in the order `SettingsRowKind` lists them.
  ///
  /// A row with nothing to say is left out and the rest keep their places.
  /// When every fact is good the list is the short one, because a screen of
  /// green rows is a screen nobody reads; the two that are not facts stay,
  /// so Quit is always one click away: testSettingsKeepsItsOrderOnEveryScreen.
  ///
  /// The Claude Desktop row is not one of the green ones: it is drawn only
  /// where Claude Desktop can have no extension, which is no `gdoc.mcpb`
  /// beside the binary or a run of this process that wrote one and could
  /// not hand it over, and it carries the one thing to do about that, so it
  /// does not wait for something else to go wrong first:
  /// testTheExtensionRowAppearsOnlyUntilGdocHasWrittenOne and
  /// testARunThatOpenedNothingLeavesTheRowWhereItWas.
  static func settingsRows(facts: Facts) -> [SettingsRowModel] {
    var rows: [SettingsRowModel] = []

    if let network = facts.network, !network.reachable {
      rows.append(
        SettingsRowModel(
          .network, value: "Google not reachable", glyph: .exclamation, tone: .bad))
    }

    rows.append(contentsOf: desktop(facts: facts))
    rows.append(loginItem(facts: facts))
    rows.append(SettingsRowModel(.quit, value: SettingsRows.quitShortcut, select: .quit))
    return rows
  }

  /// The Claude Desktop row, which is drawn only when the app can prove
  /// Claude Desktop has no extension to load: no `gdoc.mcpb` beside the
  /// binary means gdoc never wrote one, and a run of this process that
  /// wrote one and could not hand it over is the same thing with a file
  /// left behind. A file that is there says nothing about what Desktop
  /// loaded, and a row claiming it was connected would be a guess.
  static func desktop(facts: Facts) -> [SettingsRowModel] {
    guard facts.extensionState == .notInstalled else { return [] }
    // While an update child is up the row offers nothing to press. Both
    // argv lists are `gdoc update` and `--desktop` installs the release it
    // writes the extension of, so a second press is a second process
    // renaming the binary the first one is replacing:
    // testNeitherUpdateIsOfferedWhileOneIsRunning.
    guard !facts.updating else {
      return [SettingsRowModel(.desktop, value: "not installed", glyph: .empty)]
    }
    return [
      SettingsRowModel(
        .desktop, value: "not installed", glyph: .empty,
        actions: [
          ActionModel(
            .installExtension, icon: .down, tip: Tips.installExtension, target: "desktop")
        ])
    ]
  }

  static func loginItem(facts: Facts) -> SettingsRowModel {
    switch facts.loginItem {
    case .on, .off:
      return SettingsRowModel(
        .launchAtLogin, isOn: facts.loginItem == .on, select: .toggleLoginItem)
    case .needsApproval:
      return SettingsRowModel(
        .launchAtLogin, quiet: Sentences.loginItemApproval, isOn: false,
        select: .openLoginItems)
    }
  }
}
