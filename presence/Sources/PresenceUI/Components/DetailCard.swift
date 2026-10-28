// The card itself: the row uncut, on the tooltip's own plate.
//
// It is drawn by the one layer every tooltip is drawn by, for the reason
// that layer exists: a plate inside a row would be clipped by the row, and
// two plates would overlap. So this file is a plate and a carrier, and
// where it goes is `Tooltip.place`'s, as a bubble's is.
//
// The plate is the tooltip's surface, edge and shadow, so the panel has one
// kind of thing that floats and not two. What it is not is a bubble: a
// bubble is one line of words about a control, and this is three or four
// lines about a document, each of them wrapped whole. That is the whole
// difference, and it is why `DetailCard.width` is stated rather than
// measured off the longest line: a plate whose width followed its words
// would be a different shape on every row of one list.
//
// The width is the two edges the fifth look's design note names: the left
// is the title's own, 44 pt from the panel's edge, and the right is the
// column every value, every time and every icon in the panel ends on. On a
// 360 pt panel that is 296, which ControlTests states as a literal and
// reads back off the drawing.
//
// It takes no click. The layer refuses hits for everything it draws, so the
// row under the card keeps its own press: a card that swallowed the click
// it was telling you about would be the panel's worst joke.

import SwiftUI

/// The row uncut, as a plate.
public struct DetailCard: View {
  /// From the title's left edge to the column every value ends on.
  public static let width: CGFloat =
    Grid.panelWidth - Grid.quietInset - Grid.unit - Grid.rowInset
  /// The plate's own padding, which is the 8 pt grid's unit.
  public static let padding: CGFloat = Grid.unit
  /// The gap between two of its lines.
  public static let lineGap: CGFloat = 2

  public let model: DetailCardModel

  @Environment(\.colorScheme) private var scheme

  public init(_ model: DetailCardModel) {
    self.model = model
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    let shadow = palette.tooltipShadow
    return VStack(alignment: .leading, spacing: Self.lineGap) {
      line(model.title, role: Typography.rowTitle, ink: palette.text)
      if let when = model.when {
        line(when, role: Typography.caption, ink: palette.textSecondary)
      }
      line(model.note, role: Typography.caption, ink: palette.textSecondary)
      // The one line that is not about the document, in the quietest ink
      // there is: it is an instruction, and it goes for good after the
      // first row a person opens.
      if let hint = model.hint {
        line(hint, role: Typography.caption, ink: palette.textTertiary)
      }
    }
    .padding(Self.padding)
    .frame(width: Self.width, alignment: .leading)
    .background(
      RoundedRectangle(cornerRadius: Grid.controlRadius, style: .continuous)
        .fill(palette.tooltipSurface.color)
    )
    .overlay(
      RoundedRectangle(cornerRadius: Grid.controlRadius, style: .continuous)
        .strokeBorder(palette.tooltipEdge.color, lineWidth: Grid.hairline)
    )
    .shadow(color: shadow.swatch.color, radius: shadow.radius, x: shadow.x, y: shadow.y)
    .metricProbe("card")
  }

  /// One line of the card, wrapped whole: nothing on this plate is cut,
  /// because being uncut is the only thing it is for.
  private func line(_ words: String, role: TypeRole, ink: Swatch) -> some View {
    Text(words)
      .font(role.font)
      .foregroundStyle(ink.color)
      .multilineTextAlignment(.leading)
      .fixedSize(horizontal: false, vertical: true)
      .frame(maxWidth: .infinity, alignment: .leading)
  }
}

extension View {
  /// Carry this row's card for the layer to draw once the pointer rests.
  ///
  /// It sits on the row's middle column, the title and the sub-line
  /// together, because those two lines are what the card says whole. The
  /// icon column is its own thing and carries its own words.
  ///
  /// The rest is the caller's, so the two presses on the same words can end
  /// it: `RowView` owns it and hands a binding down.
  func detailCard(_ model: DetailCardModel?, rest: Binding<CardRest>, forced: Bool)
    -> some View
  {
    modifier(DetailCardCarrier(model: model, rest: rest, forced: forced))
  }
}

/// The hover half: this row's own rest, and its box reported upwards.
///
/// It is `TooltipCarrier` with the rest kept outside it, which is the one
/// difference between the two: a bubble's rest is nobody's business but the
/// bubble's, and a card's is ended by presses the carrier never sees.
///
/// A row with no card is handed back untouched, which is not a saving: the
/// preference is set and not added to, so a carrier reporting no request
/// over a title that was asking for its bubble would answer for it. NOW's
/// live rows are that row:
/// ControlTests.testACardlessRowKeepsItsTitlesOwnTooltip.
struct DetailCardCarrier: ViewModifier {
  let model: DetailCardModel?
  @Binding var rest: CardRest
  let forced: Bool

  @State private var hovers = 0

  @ViewBuilder
  func body(content: Content) -> some View {
    if let model {
      carrying(content, model)
    } else {
      content
    }
  }

  private func carrying(_ content: Content, _ model: DetailCardModel) -> some View {
    content
      .onHover { inside in
        hovers += 1
        guard inside else {
          rest.after(.leave)
          return
        }
        rest.after(.enter)
        let entered = hovers
        Task { @MainActor in
          try? await Task.sleep(nanoseconds: UInt64(Tooltip.delay * 1_000_000_000))
          // The same hover: a pointer that left and came back is a new rest.
          if entered == hovers { rest.after(.rested) }
        }
      }
      .anchorPreference(key: TooltipAnchors.self, value: .bounds) { anchor in
        guard forced || rest.isShowing else { return [] }
        return [
          TooltipRequest(
            content: .card(model), align: .leading, forced: forced, anchor: anchor)
        ]
      }
  }
}
