// The panel's one button with a word in it.
//
// `.btn` in docs/design/2026-10-17-presence-screens-v3.html is a 24 pt box
// with 10 pt either side, and the panel holds three of them: Open chat on
// the hold card, and the pair of an inline confirm. Everything else a person
// can press is an icon with a tooltip, which is the design principles' one
// vocabulary: an action is a glyph.
//
// So the words are an enum and not a string. A view that could write its own
// label would grow a fourth button the week after, and the scan in
// ControlTests counts these four and nothing else. Release is not among
// them: the screens file labels the hold card's second button "Release
// without writing", and the specification's decision table says the panel
// never says Release and carries Dismiss instead, and a decision wins over
// a label.
//
// ControlTests states the heights, the paddings and the three kinds.

import SwiftUI

/// Every word the panel puts inside a button.
public enum ButtonWord: String, Sendable, CaseIterable {
  /// The hold card's first button: the chat is where the writing is held, so
  /// this is the one action that leaves the panel for another app.
  case openChat = "Open chat"
  /// The hold card's second: the card goes, the hold stays in the chat.
  case dismiss = "Dismiss"
  /// The quiet half of an inline confirm, which is always the safe one.
  case keep = "Keep"
  /// The loud half of the one confirm the panel asks for.
  case signOut = "Sign out"
}

/// The three plates a button is drawn on.
public enum ButtonKind: Sendable, CaseIterable {
  /// `.btn`: a hairline and the panel's own ink.
  case plain
  /// `.btn-primary`: the one filled button, which is Open chat.
  case primary
  /// `.confirm .btn.dz`: the red word of a confirm pair, with no plate of
  /// its own, because a red fill beside a red word reads as an alarm.
  case danger
}

/// The two heights. A confirm pair sits inside a row, so it is the short one.
public enum ButtonSize: Sendable, CaseIterable {
  case regular
  case compact
}

/// One button with a word in it.
public struct TextButton: View {
  /// `.screen.light .btn-primary{box-shadow:inset 0 .5px 0 #ffffff26}`: the
  /// half-pixel highlight along the top of the one filled button, on white
  /// only. It is inline in the stylesheet rather than a token, so it is a
  /// literal here beside the rule it came from.
  public static let primaryHighlight = Swatch(0xFF_FF_FF, alpha8: 0x26)

  public let word: ButtonWord
  public let kind: ButtonKind
  public let size: ButtonSize
  public let action: () -> Void

  @Environment(\.colorScheme) private var scheme

  public init(
    _ word: ButtonWord, kind: ButtonKind = .plain, size: ButtonSize = .regular,
    action: @escaping () -> Void = {}
  ) {
    self.word = word
    self.kind = kind
    self.size = size
    self.action = action
  }

  /// `.btn{height:24px}` and `.confirm .btn{height:20px}`.
  public static func height(_ size: ButtonSize) -> CGFloat {
    size == .compact ? 20 : 24
  }

  /// `padding:0 10px`, and `0 8px` in a confirm pair.
  public static func paddingH(_ size: ButtonSize) -> CGFloat {
    size == .compact ? 8 : 10
  }

  /// The plate. The plain and dangerous kinds are transparent on black, so a
  /// button on the hold card's warm plate lets the plate through; on white
  /// `.screen.light .btn{background:var(--s0)}` gives them the panel's own
  /// surface, where transparent would read as a hole in the tint.
  public static func fill(_ kind: ButtonKind, in palette: Palette, scheme: ColorScheme) -> Swatch? {
    switch kind {
    case .primary: return palette.primaryFill
    case .plain, .danger: return scheme == .light ? palette.surface : nil
    }
  }

  /// The ink. `.btn{color:var(--t1)}`, `.btn-primary{color:#111}` which is
  /// the palette's `primaryInk`, and `.confirm .btn.dz{color:var(--bad)}`.
  public static func ink(_ kind: ButtonKind, in palette: Palette) -> Swatch {
    switch kind {
    case .plain: return palette.text
    case .primary: return palette.primaryInk
    case .danger: return palette.bad
    }
  }

  /// `border:.5px solid var(--hair-strong)`, and transparent for the two
  /// kinds that state their own border-color.
  public static func edge(_ kind: ButtonKind, in palette: Palette) -> Swatch? {
    kind == .plain ? palette.hairlineStrong : nil
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return Button(action: action) {
      Text(word.rawValue)
        .font(Typography.button.font)
        .tracking(Typography.button.letterSpacing)
        .lineLimit(1)
        .foregroundStyle(Self.ink(kind, in: palette).color)
        .padding(.horizontal, Self.paddingH(size))
        .frame(height: Self.height(size))
        .background(plate(in: palette))
        .contentShape(shape)
    }
    .buttonStyle(.plain)
    .accessibilityLabel(word.rawValue)
  }

  private var shape: RoundedRectangle {
    RoundedRectangle(cornerRadius: Grid.controlRadius, style: .continuous)
  }

  @ViewBuilder
  private func plate(in palette: Palette) -> some View {
    ZStack {
      if let fill = Self.fill(kind, in: palette, scheme: scheme) {
        shape.fill(fill.color)
      }
      if let edge = Self.edge(kind, in: palette) {
        shape.strokeBorder(edge.color, lineWidth: Grid.hairline)
      }
      if kind == .primary, scheme == .light {
        shape
          .strokeBorder(Self.primaryHighlight.color, lineWidth: Grid.hairline)
          .mask(
            VStack(spacing: 0) {
              Color.black.frame(height: Grid.hairline * 2)
              Color.clear
            }
          )
      }
    }
  }
}
