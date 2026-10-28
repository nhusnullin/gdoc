// A row's menu as the system draws it, and the view it is measured from.
//
// A held press on a document row drew its words in a SwiftUI popover until
// 2026-10-26. A popover is a window of its own, and the panel is a
// borderless `KeyPanel` whose delegate closes it the moment it loses key: a
// menu that took key would have taken the panel it belongs to with it. What
// was measured on 2026-10-18 stopped one question short of that, because the
// probe never got the app active, so the mechanism stood as a risk nobody
// could rule out. The fourth look's answer is the second of the three the
// backlog item named: a real `NSMenu`, popped up from the row's own view, so
// no second window exists to take anything.
//
// Three small pieces. `MenuEntry` is one word and what pressing it does, so
// the thing that draws the menu judges nothing. `MenuPresenter` is how a
// menu reaches the screen, which is `NSMenu` in the app and a recorder in a
// test. `MenuAnchorView` is where it pops up from: a view behind the row's
// title whose `hitTest` is nil, so a click meant for the row or for one of
// its four icons goes where it was going.
//
// The gesture itself is the title area's, through `RowView.titleHeld`: it
// sat on the whole row, icons included, so a slow click on an icon both did
// the icon's thing and opened the menu. The row's own click sits on that
// same view since the fifth look, because a long press swallows the click
// under it: two views would leave the name of a row with a menu opening
// nothing.
//
// CardTests holds the words the press offers and that the anchor catches
// nothing; SourceRulesTests holds that no file under Sources opens a window
// of its own.

import AppKit
import SwiftUI

/// One word of a menu, and what a press on it does.
struct MenuEntry {
  let word: String
  let press: () -> Void
}

/// How a row's held press puts its menu on the screen.
///
/// The app's own pops up an `NSMenu` from the anchor behind the row's title.
/// A test hands its own and reads what the press offered.
struct MenuPresenter {
  let open: @MainActor ([MenuEntry], MenuAnchorView?) -> Void

  /// The system's menu, from the row's own view.
  ///
  /// `popUp(positioning:at:in:)` tracks the menu itself and returns when it
  /// has closed, so the targets `NSMenuItem` holds unowned stay alive for
  /// every press there can be. A row whose anchor is not in a window yet
  /// opens nothing: there is no place to put a menu.
  static func system() -> MenuPresenter {
    MenuPresenter { entries, anchor in
      guard let anchor, anchor.window != nil else { return }
      let menu = NSMenu()
      let targets = entries.map { MenuTarget(press: $0.press) }
      for (entry, target) in zip(entries, targets) {
        let item = NSMenuItem(
          title: entry.word, action: #selector(MenuTarget.fire), keyEquivalent: "")
        item.target = target
        menu.addItem(item)
      }
      withExtendedLifetime(targets) {
        // The anchor's origin, which in a view that is not flipped is its
        // bottom left corner: the menu opens under the title it was asked
        // from.
        menu.popUp(positioning: nil, at: .zero, in: anchor)
      }
    }
  }
}

/// What one word of the menu is wired to, because an `NSMenuItem` carries a
/// target and a selector and not a closure.
final class MenuTarget: NSObject {
  private let press: () -> Void

  init(press: @escaping () -> Void) {
    self.press = press
  }

  @objc func fire() {
    press()
  }
}

/// The view a row's menu is measured from, which catches nothing.
final class MenuAnchorView: NSView {
  /// Nothing lands here. The anchor is a place to put a menu and never a
  /// control, so the row behind it keeps its hover, its own press and the
  /// presses of its four icons.
  override func hitTest(_ point: NSPoint) -> NSView? {
    nil
  }
}

/// Where a row's menu pops up from, once the tree has made the view.
///
/// The box is how a closure built in a `body` reaches a view AppKit owns.
/// It is filled in on the way up and on every update, so a press that
/// happens after the row is on the screen has an anchor to open from.
final class MenuAnchorBox {
  weak var view: MenuAnchorView?
}

/// The anchor, in the tree.
struct MenuAnchor: NSViewRepresentable {
  let box: MenuAnchorBox

  func makeNSView(context: Context) -> MenuAnchorView {
    let view = MenuAnchorView()
    box.view = view
    return view
  }

  func updateNSView(_ view: MenuAnchorView, context: Context) {
    box.view = view
  }
}

extension View {
  /// The held press of a row's title, where the row has a menu.
  ///
  /// A row without one carries no gesture at all: a long press that does
  /// nothing still swallows the click under it, and a row of NOW is a press
  /// of its own.
  func heldPress(_ press: (() -> Void)?) -> some View {
    modifier(HeldPress(press: press))
  }
}

/// The one gesture, on the one area.
private struct HeldPress: ViewModifier {
  let press: (() -> Void)?

  func body(content: Content) -> some View {
    if let press {
      content.onLongPressGesture(minimumDuration: Timing.menuHold, perform: press)
    } else {
      content
    }
  }
}
