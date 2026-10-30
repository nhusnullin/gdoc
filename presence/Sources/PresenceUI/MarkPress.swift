// What a press on the firm's mark means, which is one thing: open the page
// the picture itself has in Drive.
//
// The mark sat beside DOCUMENTS without a press from 2026-10-29. Nail looked
// at it on his own screen the next day and said what he expected of it: "when
// i click i expect default browser with picture from gdrive to be opened".
// So the mark took a press (docs/v2/DECISIONS.md, 2026-10-30).
//
// It takes two things to press, and either half missing is no press at all.
// No file, no mark, so there is nothing on the screen to click. Bytes and no
// address, the mark and no press: the picture is drawn and nothing says it
// can be opened, which is principle 4 of
// docs/design/2026-10-17-presence-design-principles.md, a dead control is
// not shown as live. The address is the binary's own, held to its one shape
// before it reached the facts; nothing here builds one.
//
// RowTests reads the press through a recording `PanelActions`; the click
// itself is Nail's hand check.

import Foundation

/// The mark's own press, where the mark has one.
public enum MarkPress {
  /// The press the firm's mark carries, or nothing where there is no
  /// picture to click or no page to open.
  public static func open(_ model: PanelModel, actions: PanelActions) -> (() -> Void)? {
    guard model.logo != nil, let page = model.logoURL else { return nil }
    return { actions.perform(.openMark, page) }
  }
}
