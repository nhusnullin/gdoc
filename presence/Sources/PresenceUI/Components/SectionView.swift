// A section of the panel: a label, a count, and the rows under it.
//
// `.section` in docs/design/2026-10-17-presence-screens-v3.html is 8 pt of
// air, a half-pixel hairline, 4 pt more air and then its label, except for
// the first section in the panel, which has none of the three because the
// header is already above it.
//
// `.section-label` is 28 pt tall and holds two things: the name of the
// section in small uppercase letters, and, pushed to the right, a count in
// ordinary letters. The count is the one place a number says how much is
// not on the screen, so it is never a label of its own and never a badge.
//
// RowTests states the height, the case and the count.

import SwiftUI

/// The label at the top of a section.
public struct SectionLabel: View {
  public let label: String
  /// `.section-label span`: the plain-letters count on the right, or nothing.
  public let count: String?

  @Environment(\.colorScheme) private var scheme

  public init(_ label: String, count: String? = nil) {
    self.label = label
    self.count = count
  }

  /// `text-transform:uppercase`, done here so a test can read the words that
  /// reach the screen rather than the ones that were passed in.
  public static func text(_ label: String) -> String {
    label.uppercased()
  }

  /// `--t3` on black, and `.screen.light .section-label{color:var(--t2)}`,
  /// because a tertiary grey on white is too faint to read at 11 pt.
  public static func ink(in scheme: ColorScheme) -> Swatch {
    let palette = Palette.of(scheme)
    return scheme == .dark ? palette.textTertiary : palette.textSecondary
  }

  public var body: some View {
    let role = Typography.sectionLabel(in: scheme)
    return HStack(spacing: Grid.columnGap) {
      Text(Self.text(label))
        .font(role.font)
        .tracking(role.letterSpacing)
      Spacer(minLength: 0)
      if let count {
        Text(count)
          .font(countFont)
      }
    }
    .foregroundStyle(Self.ink(in: scheme).color)
    .padding(.horizontal, Grid.rowInset)
    .frame(maxWidth: .infinity, minHeight: Grid.sectionLabelHeight, alignment: .leading)
    .metricProbe("section-label")
  }

  /// `.section-label span{letter-spacing:0;text-transform:none;
  /// font-weight:400}`: the count keeps the label's size and colour and
  /// loses its three other rules.
  private var countFont: Font {
    .system(size: Typography.sectionLabel.size, weight: .regular)
  }
}

/// One section: its label and its rows.
public struct SectionView<Content: View>: View {
  public let label: String
  public let count: String?
  /// `.section:first-of-type`: no air, no hairline, no padding.
  public let isFirst: Bool
  public let content: Content

  @Environment(\.colorScheme) private var scheme

  public init(
    _ label: String, count: String? = nil, isFirst: Bool = false,
    @ViewBuilder content: () -> Content
  ) {
    self.label = label
    self.count = count
    self.isFirst = isFirst
    self.content = content()
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return VStack(alignment: .leading, spacing: 0) {
      SectionLabel(label, count: count)
      content
    }
    .padding(.top, isFirst ? 0 : 4)
    .overlay(alignment: .top) {
      if !isFirst {
        Rectangle()
          .fill(palette.hairline.color)
          .frame(height: Grid.hairline)
      }
    }
    .padding(.top, isFirst ? 0 : Grid.unit)
    .metricProbe("section")
  }
}
