// The sign-in URL, as a line a person can read and copy.
//
// `.urlline` in docs/design/2026-10-17-presence-screens-v3.html is the one
// monospaced thing in the panel: a faint plate under the Account row holding
// the address the browser was sent to. It is there for the case the default
// browser opened the wrong profile, which is the ordinary case on a machine
// with a work account and a personal one.
//
// It says the URL and nothing about it. Copying it is the icon on the row
// above, not this line.
//
// CardTests renders it; the plate's two values are literals here beside the
// rules they came from, because the stylesheet writes them inline rather
// than as tokens.

import SwiftUI

/// One URL on its plate.
public struct URLLine: View {
  /// `background:rgba(255,255,255,.06)` and `.screen.light .urlline
  /// {background:#00000008}`.
  public static let plateDark = Swatch(0xFF_FF_FF, alpha: 0.06)
  public static let plateLight = Swatch(0x00_00_00, alpha8: 0x08)
  /// `padding:5px 8px;border-radius:6px`.
  public static let paddingH: CGFloat = Grid.unit
  public static let paddingV: CGFloat = 5
  public static let radius: CGFloat = Grid.controlRadius
  /// `margin:4px 12px 6px 44px`: it starts where the row above it starts.
  public static let topMargin: CGFloat = 4
  public static let bottomMargin: CGFloat = 6

  public let url: String

  @Environment(\.colorScheme) private var scheme

  public init(url: String) {
    self.url = url
  }

  public static func plate(in scheme: ColorScheme) -> Swatch {
    scheme == .dark ? plateDark : plateLight
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return Text(url)
      .font(Typography.mono.font)
      .lineLimit(1)
      .truncationMode(.tail)
      .foregroundStyle(palette.textSecondary.color)
      .padding(.horizontal, Self.paddingH)
      .padding(.vertical, Self.paddingV)
      .frame(maxWidth: .infinity, alignment: .leading)
      .background(
        RoundedRectangle(cornerRadius: Self.radius, style: .continuous)
          .fill(Self.plate(in: scheme).color)
      )
      .metricProbe("urlline")
      .padding(.top, Self.topMargin)
      .padding(.bottom, Self.bottomMargin)
      .padding(.leading, Grid.quietInset)
      .padding(.trailing, Grid.rowInset)
      .accessibilityLabel(url)
  }
}
