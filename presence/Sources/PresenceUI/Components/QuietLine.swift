// A line under a row, in the quietest grey the panel has.
//
// `.quiet` in docs/design/2026-10-17-presence-screens-v3.html is a 12 pt
// line inset 44 pt from the left, which is the row's own 12 pt plus the
// glyph's 16 pt and the 8 pt gap plus 8 pt again: it starts where the title
// above it starts, so a sentence under a row reads as that row's and not as
// a row of its own.
//
// It carries words and never an action, which is why it is not a row.
//
// RowTests states the inset and the size.

import SwiftUI

/// One quiet line.
public struct QuietLine: View {
  public let text: String
  /// `padding-left`. The hold card hands over its own, because its plate is
  /// already inset.
  public let inset: CGFloat
  /// The hold card's ink, which is the one place this line is not tertiary.
  public let ink: Swatch?

  @Environment(\.colorScheme) private var scheme

  public init(_ text: String, inset: CGFloat = Grid.quietInset, ink: Swatch? = nil) {
    self.text = text
    self.inset = inset
    self.ink = ink
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return Text(text)
      .font(Typography.rowValue.font)
      .foregroundStyle((ink ?? palette.textTertiary).color)
      .fixedSize(horizontal: false, vertical: true)
      .frame(maxWidth: .infinity, alignment: .leading)
      .padding(.leading, inset)
      .padding(.trailing, Grid.rowInset)
      .padding(.top, 4)
      .padding(.bottom, Grid.unit)
      .metricProbe("quiet")
  }
}
