// The window the panel is drawn in.
//
// A borderless NSPanel that can take key, so Esc reaches it and so it closes
// the moment the person clicks anywhere else. It hangs under the status
// item's button, never over the menu bar, and it holds a SwiftUI view in an
// NSHostingView and nothing else.
//
// It is closed, never hidden behind another window: an always-visible surface
// is the one thing the design principles rule out, and a panel left open
// behind Safari is exactly that.

import AppKit
import PresenceUI
import SwiftUI

/// How far under the menu bar the panel sits.
private let gapUnderTheBar: CGFloat = 6

/// A panel that takes key, so Esc closes it.
///
/// A borderless window is not key-capable by default, which would leave the
/// person with no keyboard way out of a window that has no close button.
private final class KeyPanel: NSPanel {
  override var canBecomeKey: Bool { true }

  override func cancelOperation(_ sender: Any?) {
    performClose(sender)
  }

  override func performClose(_ sender: Any?) {
    close()
  }
}

@MainActor
final class Panel: NSObject, NSWindowDelegate {
  private let panel: KeyPanel
  private let host: NSHostingView<AnyView>
  /// The button the open panel hangs from, so a redraw that changes the
  /// height can hang it there again. Weak because the status item owns it.
  private weak var anchor: NSStatusBarButton?

  override init() {
    self.panel = KeyPanel(
      contentRect: NSRect(x: 0, y: 0, width: Grid.panelWidth, height: 1),
      styleMask: [.borderless, .nonactivatingPanel],
      backing: .buffered,
      defer: true)
    self.host = NSHostingView(rootView: AnyView(EmptyView()))
    super.init()

    panel.isFloatingPanel = true
    panel.hidesOnDeactivate = false
    panel.level = .popUpMenu
    panel.isMovable = false
    panel.hasShadow = true
    panel.isOpaque = false
    panel.backgroundColor = .clear
    panel.collectionBehavior = [.canJoinAllSpaces, .transient]
    panel.delegate = self
    panel.contentView = host
  }

  var isOpen: Bool { panel.isVisible }

  /// Called once each time the panel closes, however it closed: Esc, a click
  /// elsewhere, the item's own click. The session puts its question away.
  var onClose: (() -> Void)?

  /// Draw this view and open under that button. Opening again with the panel
  /// already up closes it, which is what a second click on a menu bar item
  /// means everywhere else on this system.
  func toggle<Content: View>(_ content: Content, under button: NSStatusBarButton?) {
    if isOpen {
      close()
      return
    }
    show(content, under: button)
  }

  /// Redraw an open panel. A closed one is left closed: a new line in the
  /// record is not a reason to put a window over somebody's work.
  ///
  /// It is placed again, because `setContentSize` keeps the frame's bottom
  /// left corner: a redraw that opened a fold would move the top edge up
  /// over the menu bar, and one that lost a row would leave a gap under the
  /// button. The placement is the one `show` uses, so an open panel hangs
  /// where it would if it had just been opened.
  func refresh<Content: View>(_ content: Content) {
    guard isOpen else { return }
    host.rootView = AnyView(content)
    size()
    place(under: anchor)
  }

  func close() {
    panel.close()
  }

  func windowDidResignKey(_ notification: Notification) {
    close()
  }

  func windowWillClose(_ notification: Notification) {
    onClose?()
  }

  private func show<Content: View>(_ content: Content, under button: NSStatusBarButton?) {
    anchor = button
    host.rootView = AnyView(content)
    size()
    place(under: button)
    panel.makeKeyAndOrderFront(nil)
  }

  private func size() {
    host.layoutSubtreeIfNeeded()
    var fitting = host.fittingSize
    fitting.width = max(fitting.width, Grid.panelWidth)
    panel.setContentSize(fitting)
  }

  /// Under the button's own screen position, pulled left so the panel's edge
  /// is inside the screen when the item sits near the right of the bar.
  private func place(under button: NSStatusBarButton?) {
    guard let window = button?.window else { return }
    let anchor = window.convertToScreen(button?.bounds ?? .zero)
    let size = panel.frame.size
    var origin = NSPoint(
      x: anchor.midX - size.width / 2, y: anchor.minY - size.height - gapUnderTheBar)
    if let visible = window.screen?.visibleFrame {
      origin.x = min(max(origin.x, visible.minX + 8), visible.maxX - size.width - 8)
      origin.y = max(origin.y, visible.minY + 8)
    }
    panel.setFrameOrigin(origin)
  }
}
