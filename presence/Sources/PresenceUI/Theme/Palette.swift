// The panel's colours, as the screens file states them.
//
// The stylesheet of docs/design/2026-10-17-presence-screens-v3.html is the
// source of truth, and these are its two blocks turned into Swift: `:root`
// is the dark palette, `.screen.light` is the light one. No semantic system
// colour stands in for any of them, because the native look is the complaint
// that started this milestone, and no setting picks between them: the
// appearance SwiftUI draws in is the whole of the choice.
//
// ThemeTests states every value here again as a literal read off the CSS.

import SwiftUI

/// One colour as the stylesheet writes one: a hex triple and an alpha.
///
/// The hex stays a hex so a value can be compared with the line it was copied
/// from. A `Color` is made at the moment of drawing and never stored, so
/// nothing here depends on a colour space the renderer chose.
public struct Swatch: Sendable, Equatable {
  public let hex: UInt32
  public let alpha: Double

  public init(_ hex: UInt32, alpha: Double = 1) {
    self.hex = hex
    self.alpha = alpha
  }

  /// The CSS's eight-digit form, `#0000001a`: the alpha is a byte there, so a
  /// value written that way keeps the byte rather than a decimal somebody
  /// converted by hand.
  public init(_ hex: UInt32, alpha8: UInt8) {
    self.init(hex, alpha: Double(alpha8) / 255)
  }

  public var red: Double { Double((hex >> 16) & 0xFF) / 255 }
  public var green: Double { Double((hex >> 8) & 0xFF) / 255 }
  public var blue: Double { Double(hex & 0xFF) / 255 }

  public var color: Color {
    Color(.sRGB, red: red, green: green, blue: blue, opacity: alpha)
  }
}

/// One shadow of a CSS stack: the offset, the blur and the colour.
///
/// The blur is the stylesheet's own number. SwiftUI's radius is half of it,
/// which is the one conversion, done here and not at each call site.
public struct Shadow: Sendable, Equatable {
  public let swatch: Swatch
  public let x: CGFloat
  public let y: CGFloat
  public let blur: CGFloat

  public init(_ swatch: Swatch, x: CGFloat = 0, y: CGFloat, blur: CGFloat) {
    self.swatch = swatch
    self.x = x
    self.y = y
    self.blur = blur
  }

  public var radius: CGFloat { blur / 2 }
}

/// Every colour the panel draws with, in one appearance.
///
/// The field names are what the thing is in the panel, not what the CSS
/// variable was called, so a view asks for `textTertiary` rather than `t3`.
/// `glyphStroke` is a width and sits here anyway, because it is the one size
/// the light block overrides.
public struct Palette: Sendable, Equatable {
  // The surfaces: --s0 the panel, --s1 a raised plate, --s2 a hovered row,
  // and --wall the desktop the mock draws the panel over.
  public let surface: Swatch
  public let raised: Swatch
  public let hover: Swatch
  public let wall: Swatch

  // The three text levels, --t1 to --t3.
  public let text: Swatch
  public let textSecondary: Swatch
  public let textTertiary: Swatch

  // `.act`'s own colour, which is --t3 in the dark block and a grey of its
  // own in the light one, so an action icon is not the same grey as the
  // words beside it there.
  public let actionIcon: Swatch

  // The hairlines, --hair and --hair-strong, each one device pixel wide.
  public let hairline: Swatch
  public let hairlineStrong: Swatch

  // The three status colours, the only colour in the panel.
  public let ok: Swatch
  public let hold: Swatch
  public let bad: Swatch

  // The hold card: its tint, the half-pixel edge inside it, and the ink the
  // light block gives it so a warm plate keeps its contrast.
  public let holdTint: Swatch
  public let holdTintEdge: Swatch
  public let holdInk: Swatch
  public let holdInkQuiet: Swatch

  // The stuck card, the same two values in red.
  public let stuckTint: Swatch
  public let stuckTintEdge: Swatch

  // The controls: the ring's track, the switch, the one filled button, the
  // confirm pair's fill and the danger edge.
  public let ringTrack: Swatch
  public let switchTrack: Swatch
  public let switchKnob: Swatch
  public let primaryFill: Swatch
  public let primaryInk: Swatch
  public let confirmFill: Swatch
  public let dangerEdge: Swatch

  // The tooltip the app draws itself, `[data-tip]::after` in the stylesheet.
  public let tooltipSurface: Swatch
  public let tooltipEdge: Swatch
  public let tooltipShadow: Shadow

  // The panel's own edge, its inset highlight and its drop shadows, in the
  // stylesheet's order.
  public let panelEdge: Swatch
  public let panelHighlight: Swatch
  public let panelShadows: [Shadow]

  // .glyph's border-width: 1.25 on black, 1.5 on white.
  public let glyphStroke: CGFloat

  /// The `:root` block.
  public static let dark = Palette(
    surface: Swatch(0x1C_1C_1E),
    raised: Swatch(0x24_24_26),
    hover: Swatch(0x2C_2C_2E),
    wall: Swatch(0x3B_43_52),
    text: Swatch(0xF2_F2_F7),
    textSecondary: Swatch(0xEB_EB_F5, alpha: 0.60),
    textTertiary: Swatch(0xEB_EB_F5, alpha: 0.32),
    actionIcon: Swatch(0xEB_EB_F5, alpha: 0.32),
    hairline: Swatch(0xFF_FF_FF, alpha: 0.08),
    hairlineStrong: Swatch(0xFF_FF_FF, alpha: 0.14),
    ok: Swatch(0x5C_C4_89),
    hold: Swatch(0xD9_A0_3C),
    bad: Swatch(0xE5_48_4D),
    holdTint: Swatch(0xD9_A0_3C, alpha: 0.07),
    holdTintEdge: Swatch(0xD9_A0_3C, alpha: 0.22),
    holdInk: Swatch(0xF2_F2_F7),
    holdInkQuiet: Swatch(0xEB_EB_F5, alpha: 0.32),
    stuckTint: Swatch(0xE5_48_4D, alpha: 0.07),
    stuckTintEdge: Swatch(0xE5_48_4D, alpha: 0.22),
    ringTrack: Swatch(0xFF_FF_FF, alpha: 0.10),
    switchTrack: Swatch(0xFF_FF_FF, alpha: 0.24),
    switchKnob: Swatch(0xFF_FF_FF),
    primaryFill: Swatch(0xFF_FF_FF, alpha: 0.92),
    primaryInk: Swatch(0x11_11_11),
    confirmFill: Swatch(0xE5_48_4D, alpha: 0.10),
    dangerEdge: Swatch(0x00_00_00, alpha: 0),
    tooltipSurface: Swatch(0x2C_2C_2E),
    tooltipEdge: Swatch(0xFF_FF_FF, alpha: 0.12),
    tooltipShadow: Shadow(Swatch(0x00_00_00, alpha: 0.4), y: 2, blur: 8),
    panelEdge: Swatch(0xFF_FF_FF, alpha: 0.14),
    panelHighlight: Swatch(0xFF_FF_FF, alpha: 0.06),
    panelShadows: [
      Shadow(Swatch(0x00_00_00, alpha: 0.45), y: 12, blur: 40),
      Shadow(Swatch(0x00_00_00, alpha: 0.4), y: 1, blur: 2),
    ],
    glyphStroke: 1.25
  )

  /// The `.screen.light` block. The light panel carries no border: its
  /// half-pixel ring is a shadow, and `panelEdge` is that ring's colour.
  public static let light = Palette(
    surface: Swatch(0xFF_FF_FF),
    raised: Swatch(0xDF_DF_E4),
    hover: Swatch(0xEC_EC_EF),
    wall: Swatch(0x7E_93_AD),
    text: Swatch(0x11_11_13),
    textSecondary: Swatch(0x6B_6B_70),
    textTertiary: Swatch(0x96_96_9C),
    actionIcon: Swatch(0x7B_7B_81),
    hairline: Swatch(0x00_00_00, alpha8: 0x1A),
    hairlineStrong: Swatch(0x00_00_00, alpha8: 0x29),
    ok: Swatch(0x1B_9E_4B),
    hold: Swatch(0xE6_87_0A),
    bad: Swatch(0xE0_31_2E),
    holdTint: Swatch(0xFF_F6_E5),
    holdTintEdge: Swatch(0xE6_87_0A, alpha8: 0x66),
    holdInk: Swatch(0x8A_55_00),
    holdInkQuiet: Swatch(0x8A_55_00),
    stuckTint: Swatch(0xC9_38_38, alpha: 0.07),
    stuckTintEdge: Swatch(0xC9_38_38, alpha: 0.28),
    ringTrack: Swatch(0x00_00_00, alpha8: 0x1F),
    switchTrack: Swatch(0x00_00_00, alpha: 0.12),
    switchKnob: Swatch(0xFF_FF_FF),
    primaryFill: Swatch(0x11_11_13),
    primaryInk: Swatch(0xFF_FF_FF),
    // The stylesheet leaves .btn.confirm's fill alone in the light block, so
    // the confirm pair carries the dark red there too.
    confirmFill: Swatch(0xE5_48_4D, alpha: 0.10),
    dangerEdge: Swatch(0xE0_31_2E, alpha8: 0x33),
    tooltipSurface: Swatch(0xF2_F2_F3),
    tooltipEdge: Swatch(0x00_00_00, alpha: 0.14),
    tooltipShadow: Shadow(Swatch(0x00_00_00, alpha: 0.14), y: 2, blur: 8),
    panelEdge: Swatch(0x00_00_00, alpha8: 0x29),
    panelHighlight: Swatch(0xFF_FF_FF),
    panelShadows: [
      Shadow(Swatch(0x00_00_00, alpha8: 0x1F), y: 1, blur: 2),
      Shadow(Swatch(0x00_00_00, alpha8: 0x29), y: 12, blur: 36),
    ],
    glyphStroke: 1.5
  )

  /// The one way a view picks a palette.
  public static func of(_ scheme: ColorScheme) -> Palette {
    scheme == .dark ? .dark : .light
  }
}
