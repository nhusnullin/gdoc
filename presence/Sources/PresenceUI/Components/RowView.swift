// One row of the panel, in the five shapes the panel has.
//
// `.row` in docs/design/2026-10-17-presence-screens-v3.html is a grid of
// three columns, 16 pt for the glyph, the rest for the title, and as much as
// the right-hand side asks for; and of up to three rows, the title with its
// value, a sub-line under both, and a line of text buttons under that. The
// five shapes are that grid's five variants: `.row`, `.row.two`,
// `.row.compact`, `.row.docrow` and `.row-more`.
//
// Nothing here decides anything. The caller hands over the words, the shape,
// what sits in the glyph column and what sits on the right, and the row puts
// them on the grid. The two states it does draw are the stylesheet's:
// `.row.hov` fills behind a hovered row, and `.row.is-busy` hides its
// actions, because an icon that would start a second write while the first
// is in flight is an icon nobody should be able to press.
//
// RowTests states every height and the action column as literals and reads
// them back off the drawing through MetricProbes.

import SwiftUI

/// The five grids a row is laid out on.
public enum RowShape: String, Sendable, CaseIterable {
  /// `.row`: one line, 32 pt.
  case plain
  /// `.row.two`: a title and a sub-line, 44 pt.
  case two
  /// `.row.compact`: one line, 28 pt, for a list of short facts.
  case compact
  /// `.row.docrow`: a title, a sub-line of two, and 108 pt of actions.
  case doc
  /// `.row-more`: the quiet 12 pt line that opens a longer list.
  case more

  /// `min-height`.
  public var minHeight: CGFloat {
    switch self {
    case .plain: return Grid.rowHeight
    case .two, .doc: return Grid.twoLineRowHeight
    case .compact, .more: return Grid.compactRowHeight
    }
  }

  /// The vertical half of `padding`. Every shape keeps the row's 12 pt
  /// either side; only the two-line ones pad above and below.
  public var verticalPadding: CGFloat {
    switch self {
    case .two, .doc: return 6
    case .plain, .compact, .more: return 0
    }
  }

  /// `grid-template-columns`' third column, where the document row is the
  /// one shape that fixes it: four icons in 108 pt, so every document row
  /// in a list breaks its title at the same place.
  public var trailingColumn: CGFloat? {
    self == .doc ? Grid.actionColumn : nil
  }

  /// The title's type role. `.row-more .row-title{font-size:12px}`.
  public var titleRole: TypeRole {
    self == .more ? Typography.rowValue : Typography.rowTitle
  }

  /// `.row-more .row-title{color:var(--t3)}`: the row that opens a list is
  /// not one of the list's own rows.
  public func titleInk(in palette: Palette) -> Swatch? {
    self == .more ? palette.textTertiary : nil
  }

  /// `-webkit-line-clamp`: a document's reason gets two lines, and every
  /// other sub-line one.
  public var subLines: Int {
    self == .doc ? 2 : 1
  }
}

/// One row.
public struct RowView<Leading: View, Trailing: View>: View {
  public let shape: RowShape
  public let title: String
  public let value: String?
  public let sub: String?
  public let gone: Bool
  public let hovering: Bool
  public let busy: Bool
  public let leading: Leading
  public let trailing: Trailing

  @Environment(\.colorScheme) private var scheme
  @State private var pointerInside = false

  public init(
    _ shape: RowShape = .plain,
    title: String,
    value: String? = nil,
    sub: String? = nil,
    gone: Bool = false,
    hovering: Bool = false,
    busy: Bool = false,
    @ViewBuilder leading: () -> Leading,
    @ViewBuilder trailing: () -> Trailing
  ) {
    self.shape = shape
    self.title = title
    self.value = value
    self.sub = sub
    self.gone = gone
    self.hovering = hovering
    self.busy = busy
    self.leading = leading()
    self.trailing = trailing()
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return HStack(alignment: .top, spacing: Grid.columnGap) {
      // The glyph column, 16 pt wide and as tall as the title's own line, so
      // a glyph beside a two-line row sits against the title and not
      // between the two lines.
      leading
        .frame(width: Grid.glyphSize, height: Typography.rowTitle.lineHeight)

      VStack(alignment: .leading, spacing: 0) {
        HStack(alignment: .firstTextBaseline, spacing: Grid.columnGap) {
          MiddleTitle(
            title, role: shape.titleRole, gone: gone, quietInk: shape.titleInk(in: palette))
          Spacer(minLength: 0)
          if let value {
            Text(value)
              .font(Typography.rowValue.font)
              .monospacedDigit()
              .lineLimit(1)
              .foregroundStyle(palette.textSecondary.color)
          }
        }
        .frame(minHeight: Typography.rowTitle.lineHeight)

        if let sub {
          Text(sub)
            .font(Typography.caption.font)
            .lineLimit(shape.subLines)
            .multilineTextAlignment(.leading)
            .fixedSize(horizontal: false, vertical: true)
            .foregroundStyle(palette.textTertiary.color)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
      }

      // `.row.is-busy .acts{display:none}`: while a write of this row's own
      // is in flight the row carries no action at all, rather than one that
      // is off.
      if !busy {
        trailing
          .frame(width: shape.trailingColumn, alignment: .trailing)
          .metricProbe("row.trailing")
          .frame(maxHeight: .infinity, alignment: .center)
      }
    }
    .padding(.horizontal, Grid.rowInset)
    .padding(.vertical, shape.verticalPadding)
    .frame(maxWidth: .infinity, minHeight: shape.minHeight, alignment: .leading)
    .background(
      RoundedRectangle(cornerRadius: Grid.controlRadius, style: .continuous)
        .fill(isFilled ? palette.hover.color : Color.clear)
    )
    .onHover { pointerInside = $0 }
    .metricProbe("row.\(shape.rawValue)")
  }

  /// `.row.hov{background:var(--s2)}`. The caller can state it, so a still
  /// of a hovered row renders, and the pointer states it in the app.
  var isFilled: Bool { hovering || pointerInside }
}

extension RowView where Leading == StatusGlyph, Trailing == EmptyView {
  /// The common row: a status glyph on the left and nothing on the right.
  public init(
    _ shape: RowShape = .plain,
    glyph: GlyphKind = .blank,
    tone: GlyphTone = .quiet,
    title: String,
    value: String? = nil,
    sub: String? = nil,
    gone: Bool = false,
    hovering: Bool = false
  ) {
    self.init(
      shape, title: title, value: value, sub: sub, gone: gone, hovering: hovering,
      leading: { StatusGlyph(glyph, tone: tone) }, trailing: { EmptyView() })
  }
}

extension RowView where Leading == StatusGlyph {
  /// A row with a status glyph and something on the right: actions, a
  /// switch, a button.
  public init(
    _ shape: RowShape = .plain,
    glyph: GlyphKind = .blank,
    tone: GlyphTone = .quiet,
    title: String,
    value: String? = nil,
    sub: String? = nil,
    gone: Bool = false,
    hovering: Bool = false,
    busy: Bool = false,
    @ViewBuilder trailing: () -> Trailing
  ) {
    self.init(
      shape, title: title, value: value, sub: sub, gone: gone, hovering: hovering, busy: busy,
      leading: { StatusGlyph(glyph, tone: tone) }, trailing: trailing)
  }
}
