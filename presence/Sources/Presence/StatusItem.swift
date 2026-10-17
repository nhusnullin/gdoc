// The item in the menu bar.
//
// One button, one template image, one tooltip. Both mouse buttons send the
// same action, because there is one surface: a right-click opens the same
// panel a left-click does, and there is no second menu anywhere.
//
// The item is of a variable length, because the image is not square: four of
// the six states are the mark with a badge beside it. An idle app still keeps
// its item, with the mark at 0.45, so the panel stays reachable when nothing
// is happening. The item is the only thing the app shows when it is not
// open.

import AppKit
import PresenceUI

@MainActor
final class StatusItem: NSObject {
  private let item: NSStatusItem
  private let onClick: () -> Void

  init(onClick: @escaping () -> Void) {
    self.item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
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

  /// The image for a state, and the tooltip under it. The dimming of idle is
  /// inside the image, so nothing here touches the button's own opacity.
  func show(_ state: BarState) {
    guard let button = item.button else { return }
    button.image = BarImage.image(for: state)
    button.toolTip = BarImage.tooltip(for: state)
  }

  @objc private func clicked() {
    onClick()
  }
}
