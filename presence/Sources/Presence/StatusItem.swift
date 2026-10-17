// The item in the menu bar.
//
// One button, one template image, one tooltip. Both mouse buttons send the
// same action, because there is one surface: a right-click opens the same
// panel a left-click does, and there is no second menu anywhere.
//
// An idle app still keeps its item, with no image and the mark's own width,
// so the panel stays reachable when nothing is happening. The item is the
// only thing the app shows when it is not open.

import AppKit
import PresenceUI

@MainActor
final class StatusItem: NSObject {
  private let item: NSStatusItem
  private let onClick: () -> Void

  init(onClick: @escaping () -> Void) {
    self.item = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
    self.onClick = onClick
    super.init()

    guard let button = item.button else { return }
    button.target = self
    button.action = #selector(clicked)
    button.sendAction(on: [.leftMouseUp, .rightMouseUp])
    show(.idle)
  }

  /// Where the panel hangs from.
  var button: NSStatusBarButton? { item.button }

  /// The glyph for a state, and the tooltip under it.
  func show(_ state: GlyphState) {
    guard let button = item.button else { return }
    button.image = Glyph.image(for: state)
    button.title = Glyph.image(for: state) == nil ? Glyph.mark : ""
    button.alphaValue = state == .idle ? 0.45 : 1
    button.toolTip = Glyph.tooltip(for: state)
  }

  @objc private func clicked() {
    onClick()
  }
}
