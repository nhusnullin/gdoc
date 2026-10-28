// What a press on a DOCUMENTS row means, which is one thing: open the
// document the row names.
//
// The row was opened by its first icon until the fifth look
// (docs/v2/DECISIONS.md, 2026-10-28). Nail asked for a click on the row to
// open it, and one control per action, so the icon went and the row took its
// press: a NOW row has worked this way since the fourth look, and this is
// `NowPress.open` for the other section.
//
// The kind is `.openRow` and not `.openDoc`, because the two presses differ
// in one thing a person sees. The four labels under the first row go after
// the first click on an icon, and a click on a row is not a click on an
// icon: the labels wait for one.
//
// The icons keep their own presses over the row's, which is the platform's
// rule about a child and the view behind it, so a hand reaching for Pin
// never opens a browser tab instead.
//
// CardTests reads the press through a recording `PanelActions`; the gesture
// itself is Nail's hand check.

import Foundation

/// The row's own press, where the row has one.
public enum DocumentPress {
  /// The press a document row carries, or nothing where it names no
  /// document Drive could have written.
  public static func open(_ row: DocumentRowModel, actions: PanelActions) -> (() -> Void)? {
    guard let doc = row.opens else { return nil }
    return { actions.perform(.openRow, doc) }
  }
}
