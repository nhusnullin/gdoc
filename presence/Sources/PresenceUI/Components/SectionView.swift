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
// Since 2026-10-29 the DOCUMENTS label carries the firm's mark at its right
// end, where the count used to be (docs/v2/DECISIONS.md, "The firm's mark
// beside DOCUMENTS"). The label is handed bytes and never a path: the one
// read of the cached file is the app's, and a machine with no picture draws
// the words alone, which is most of what a first run is.
//
// RowTests states the height, the case and the mark.

import AppKit
import SwiftUI

/// The label at the top of a section.
public struct SectionLabel: View {
  public let label: String
  /// The firm's mark as the facts handed it over, or nothing. Bytes and
  /// never a path: this room reads no file.
  public let mark: Data?

  @Environment(\.colorScheme) private var scheme

  public init(_ label: String, mark: Data? = nil) {
    self.label = label
    self.mark = mark
  }

  /// The picture, where the bytes are one.
  ///
  /// `NSImage(data:)` runs on each evaluation of the body. The bytes are a
  /// few kilobytes, and that cost is accepted: a stored image would be a
  /// second place the picture lives. An image with no size is the answer
  /// AppKit gives for a file that is half written, and it draws nothing.
  private var picture: NSImage? {
    guard let mark, let image = NSImage(data: mark), image.size.width > 0,
      image.size.height > 0
    else { return nil }
    image.isTemplate = true
    return image
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
      // The label's own 12 pt padding lands the right edge on the content
      // edge every value in the panel ends on, and the 28 pt row centres
      // it, so there is no offset here. The width is the picture's own
      // aspect and no number in this package.
      //
      // The picture is a template, so the ink is the label's and the file
      // carries no colour into the panel. The ink is taken opaque with its
      // alpha on the view, because a translucent tint on a template image
      // composites twice and drew the mark at half the weight of the words
      // beside it: `Swatch.opaque`.
      //
      // It is not a control and says nothing a row does not, so nothing
      // reads it out.
      if let picture {
        Image(nsImage: picture)
          .renderingMode(.template)
          .resizable()
          .aspectRatio(contentMode: .fit)
          .frame(height: Grid.markHeight)
          .foregroundStyle(Self.ink(in: scheme).opaque)
          .opacity(Self.ink(in: scheme).alpha)
          .accessibilityHidden(true)
          .metricProbe("section-label.mark")
      }
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
  /// The firm's mark, which only DOCUMENTS is handed.
  public let mark: Data?
  public let content: Content

  @Environment(\.colorScheme) private var scheme

  public init(
    _ label: String, isFirst: Bool = false, mark: Data? = nil,
    @ViewBuilder content: () -> Content
  ) {
    self.label = label
    self.isFirst = isFirst
    self.mark = mark
    self.content = content()
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return VStack(alignment: .leading, spacing: 0) {
      SectionLabel(label, mark: mark)
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
