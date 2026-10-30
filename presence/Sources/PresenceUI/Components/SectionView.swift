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
// the words alone, which is most of what a first run is. Since 2026-10-30
// the bytes go through MarkPicture, which makes one bitmap of the pixels the
// mark occupies: the label draws that bitmap one for one and scales nothing.
//
// Since 2026-10-30 the mark is also a press, where the caller hands one over:
// it opens the picture's own page in Drive, which is what Nail expected of it
// the first time he saw it (docs/v2/DECISIONS.md, 2026-10-30). The press is
// the caller's and not this room's: a label handed none draws exactly what it
// drew before, with no plate, no words and nothing read out, because a
// picture nothing can open is not a control.
//
// RowTests states the height, the case, the mark and the press.

import AppKit
import SwiftUI

/// The label at the top of a section.
public struct SectionLabel: View {
  public let label: String
  /// The firm's mark as the facts handed it over, or nothing. Bytes and
  /// never a path: this room reads no file.
  public let mark: Data?
  /// What a click on the mark does, or nothing where it does nothing. This
  /// room decides no address and reaches no page: MarkPress hands over a
  /// closure or nil, and nil is a picture with no press on it.
  public let press: (() -> Void)?
  /// Hold the mark's words open with no pointer, which is the component
  /// sheet's own ask and no panel's.
  public let forcedTooltip: Bool

  @Environment(\.colorScheme) private var scheme
  /// The pixels this screen puts in a point, which the bitmap is made at.
  @Environment(\.displayScale) private var displayScale
  @State private var hovering = false

  public init(
    _ label: String, mark: Data? = nil, press: (() -> Void)? = nil,
    forcedTooltip: Bool = false
  ) {
    self.label = label
    self.mark = mark
    self.press = press
    self.forcedTooltip = forcedTooltip
  }

  /// The bitmap to draw, where the bytes are a picture.
  ///
  /// It is made at the height the mark is drawn at and the scale of the
  /// screen it is drawn on, so this view scales nothing: a shrink in the
  /// compositor is what left the letters ragged before 2026-10-30.
  /// MarkPicture keeps it, so an evaluation of the body costs a comparison
  /// and not a draw.
  @MainActor
  public static func picture(mark: Data?, scale: CGFloat) -> CGImage? {
    guard let mark else { return nil }
    return MarkPicture.make(mark, height: Grid.markHeight, scale: scale)
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
      // it, so there is no offset here. The bitmap is already the size it is
      // drawn at, so its width is the picture's own aspect and no number in
      // this package, and the frame states the height it was made for rather
      // than asking for a shrink.
      //
      // The picture is a template, so the ink is the label's and the file
      // carries no colour into the panel. The ink is taken opaque with its
      // alpha on the view, because a translucent tint on a template image
      // composites twice and drew the mark at half the weight of the words
      // beside it: `Swatch.opaque`.
      //
      // With no press it is not a control and says nothing a row does not,
      // so nothing reads it out. With one it is a plain button: the hover
      // plate reaches 4 pt beyond the picture on every side and is taken
      // back off the layout by the matching negative padding, so the box
      // the panel measures is the picture's own either way and the mark
      // stays on the content edge. The words hang off its right edge, as a
      // row's actions do, and the pointer stays the arrow: no press in this
      // panel changes it.
      if let picture = Self.picture(mark: mark, scale: displayScale) {
        let drawn =
          Image(decorative: picture, scale: displayScale)
          .renderingMode(.template)
          .frame(height: Grid.markHeight)
          .foregroundStyle(Self.ink(in: scheme).opaque)
          .opacity(Self.ink(in: scheme).alpha)
        if let press {
          Button(action: press) {
            drawn
              .padding(Grid.unit / 2)
              .background(
                RoundedRectangle(cornerRadius: Grid.controlRadius, style: .continuous)
                  .fill(hovering ? Palette.of(scheme).hover.color : Color.clear)
              )
              .contentShape(
                RoundedRectangle(cornerRadius: Grid.controlRadius, style: .continuous)
              )
              .padding(-Grid.unit / 2)
          }
          .buttonStyle(.plain)
          .onHover { hovering = $0 }
          .tooltip(Tips.openMark, align: .trailing, forced: forcedTooltip)
          .accessibilityLabel(Tips.openMark)
          .metricProbe("section-label.mark")
        } else {
          drawn
            .accessibilityHidden(true)
            .metricProbe("section-label.mark")
        }
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
  /// What a click on that mark does, which only DOCUMENTS is handed either.
  /// `forcedTooltip` is not here: no scene holds the mark's words open, so
  /// it stays the component sheet's own parameter on the label.
  public let press: (() -> Void)?
  public let content: Content

  @Environment(\.colorScheme) private var scheme

  public init(
    _ label: String, isFirst: Bool = false, mark: Data? = nil, press: (() -> Void)? = nil,
    @ViewBuilder content: () -> Content
  ) {
    self.label = label
    self.isFirst = isFirst
    self.mark = mark
    self.press = press
    self.content = content()
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return VStack(alignment: .leading, spacing: 0) {
      SectionLabel(label, mark: mark, press: press)
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
