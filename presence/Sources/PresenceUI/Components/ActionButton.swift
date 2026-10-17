// One action in a row: an icon, 24 px, with a tooltip and three states.
//
// `.act` in docs/design/2026-10-17-presence-screens-v3.html is a 24 px box
// with a 16 px icon inside it and no label, because the actions of a row sit
// four across in 108 px and words would not fit. The tooltip is the whole of
// the label, so every button carries one.
//
// The three states are the stylesheet's: the plain `.act`, `.act.off` at
// 0.38 with words of its own saying why it cannot be pressed, and
// `.act.is-done` in green with its tooltip held open, which is Copy link for
// the second after a press. Nothing here judges which state a button is in:
// the caller hands one over.
//
// The tooltip is not drawn here. The panel draws one layer of its own after
// half a second, which is TooltipLayer; this button carries the words, the
// rule for which words, and the request that puts them on that layer.
//
// IconTests states the sizes, the dimming, the words and the second.

import Foundation
import SwiftUI

/// What an action can be: pressable, off with its reason, or just pressed.
public enum ActionState: Equatable, Sendable {
  /// `.act`: the icon in the panel's grey, pressable.
  case live
  /// `.act.off`: dimmed, with the words that say why.
  case off(String)
  /// `.act.is-done show-tip`: the green check, for one second.
  case done
}

/// One icon button.
public struct ActionButton: View {
  /// `.act`'s width and height.
  public static let size: CGFloat = Grid.actionSize
  /// `border-radius:var(--r)`.
  public static let radius: CGFloat = Grid.controlRadius
  /// `.act.off{opacity:.38}`.
  public static let offOpacity: Double = 0.38

  public let icon: Icon
  public let tip: String
  public let state: ActionState
  /// Held open with no pointer, which is what a still of the panel needs and
  /// `.show-tip` is in the screens file.
  public let forcedTooltip: Bool
  public let action: () -> Void

  @Environment(\.colorScheme) private var scheme
  @State private var hovering = false

  public init(
    _ icon: Icon, tip: String, state: ActionState = .live, forcedTooltip: Bool = false,
    action: @escaping () -> Void = {}
  ) {
    self.icon = icon
    self.tip = tip
    self.state = state
    self.forcedTooltip = forcedTooltip
    self.action = action
  }

  /// Which shape is drawn: the check stands in for the icon while it is
  /// green, which is how Copy link says Copied without a word.
  public static func shown(icon: Icon, state: ActionState) -> Icon {
    state == .done ? .done : icon
  }

  /// The words under the pointer. An off button says why it is off, and
  /// never the action it is not doing.
  public static func tooltip(_ tip: String, state: ActionState) -> String {
    if case .off(let reason) = state { return reason }
    return tip
  }

  public static func opacity(for state: ActionState) -> Double {
    if case .off = state { return offOpacity }
    return 1
  }

  /// `.act.is-done{color:var(--ok)}`, and `.act`'s own grey otherwise, which
  /// the light block is the one to change.
  public static func ink(state: ActionState, in palette: Palette) -> Swatch {
    state == .done ? palette.ok : palette.actionIcon
  }

  /// `show-tip`: a person who just pressed Copy link is not hovering for
  /// half a second to learn that it worked.
  public static func showsTooltipAlways(_ state: ActionState) -> Bool {
    state == .done
  }

  public static func isPressable(_ state: ActionState) -> Bool {
    if case .off = state { return false }
    return true
  }

  /// How long after the press the check is still green. A press in the
  /// future is no press: a clock that stepped backwards would otherwise
  /// leave the check green for ever.
  public static func isDone(copiedAt: Date?, now: Date) -> Bool {
    guard let copiedAt else { return false }
    let since = now.timeIntervalSince(copiedAt)
    return since >= 0 && since < Timing.doneFlash
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    let pressable = Self.isPressable(state)
    return Button {
      if pressable { action() }
    } label: {
      IconView(Self.shown(icon: icon, state: state), ink: Self.ink(state: state, in: palette))
        .frame(width: Self.size, height: Self.size)
        .background(
          RoundedRectangle(cornerRadius: Self.radius, style: .continuous)
            // The mock is a still, so it draws no hover on an action. The
            // row's own hover colour is the only one in the sheet, and a
            // button that answers the pointer is why `.act` carries a
            // radius with nothing behind it.
            .fill(hovering && pressable ? palette.hover.color : Color.clear)
        )
        .opacity(Self.opacity(for: state))
        .contentShape(RoundedRectangle(cornerRadius: Self.radius, style: .continuous))
    }
    .buttonStyle(.plain)
    .onHover { hovering = $0 }
    // `.acts [data-tip]::after{right:0}`: an action's words hang off its own
    // right edge, so the last of four icons keeps them inside the panel.
    .tooltip(
      Self.tooltip(tip, state: state), align: .trailing,
      forced: forcedTooltip || Self.showsTooltipAlways(state)
    )
    .accessibilityLabel(Self.tooltip(tip, state: state))
  }
}
