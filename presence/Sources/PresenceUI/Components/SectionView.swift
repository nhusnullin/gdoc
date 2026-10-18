// A section of the panel: a label, and the rows under it.
//
// `.section` in docs/design/2026-10-17-presence-screens-v3.html is 8 pt of
// air, a half-pixel hairline, 4 pt more air and then its label, except for
// the first section in the panel, which has none of the three because the
// header is already above it.
//
// `.section-label` is 28 pt tall and holds one thing: the name of the
// section in small uppercase letters. It carried a count on its right until
// 2026-10-18, "14 days · 12" over DOCUMENTS, and the third look took it off
// with the lists it was a number about: every row the panel has is on the
// screen, so a number saying how many there are is a count of what a person
// can see.
//
// RowTests states the height and the case.

import SwiftUI

/// The label at the top of a section.
public struct SectionLabel: View {
  public let label: String

  @Environment(\.colorScheme) private var scheme

  public init(_ label: String) {
    self.label = label
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
    }
    .foregroundStyle(Self.ink(in: scheme).color)
    .padding(.horizontal, Grid.rowInset)
    .frame(maxWidth: .infinity, minHeight: Grid.sectionLabelHeight, alignment: .leading)
    .metricProbe("section-label")
  }
}

/// One section: its label and its rows.
public struct SectionView<Content: View>: View {
  public let label: String
  /// `.section:first-of-type`: no air, no hairline, no padding.
  public let isFirst: Bool
  public let content: Content

  @Environment(\.colorScheme) private var scheme

  public init(
    _ label: String, isFirst: Bool = false, @ViewBuilder content: () -> Content
  ) {
    self.label = label
    self.isFirst = isFirst
    self.content = content()
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return VStack(alignment: .leading, spacing: 0) {
      SectionLabel(label)
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
