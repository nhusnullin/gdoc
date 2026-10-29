// One row of the panel, in the four shapes the panel has.
//
// `.row` in docs/design/2026-10-17-presence-screens-v3.html is a grid of
// three columns, 16 pt for the glyph, the rest for the title, and as much as
// the right-hand side asks for; and of up to three rows, the title with its
// value, a sub-line under both, and a line of text buttons under that. The
// four shapes are that grid's variants: `.row`, `.row.two`,
// `.row.compact` and `.row.docrow`. The fifth was `.row-more`, the quiet
// row that opened a longer list, and it went on 2026-10-18 with the one
// list that was ever behind one.
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

/// The four grids a row is laid out on.
public enum RowShape: String, Sendable, CaseIterable {
  /// `.row`: one line, 32 pt.
  case plain
  /// `.row.two`: a title and a sub-line, 44 pt.
  case two
  /// `.row.compact`: one line, 28 pt, for a list of short facts.
  case compact
  /// `.row.docrow`: a title, a sub-line of two, and 108 pt of actions.
  case doc

  /// `min-height`.
  public var minHeight: CGFloat {
    switch self {
    case .plain: return Grid.rowHeight
    case .two, .doc: return Grid.twoLineRowHeight
    case .compact: return Grid.compactRowHeight
    }
  }

  /// The vertical half of `padding`. Every shape keeps the row's 12 pt
  /// either side; only the two-line ones pad above and below.
  public var verticalPadding: CGFloat {
    switch self {
    case .two, .doc: return 6
    case .plain, .compact: return 0
    }
  }

  /// `grid-template-columns`' first column: 16 pt for the glyph, in every
  /// shape there is.
  ///
  /// The document row kept 8 pt in front of its glyph for a pin until the
  /// third look, and every row of a list with a pin in it kept the slot
  /// whether it was pinned or not, which put DOCUMENTS right of its own
  /// label and of every other row in the panel. The pin is the row's fifth
  /// icon now, at the trailing edge with the other four, so the panel has
  /// one left edge again: testEveryRowShapeKeepsTheGlyphsOwnLeadingColumn.
  public var leadingColumn: CGFloat { Grid.glyphSize }

  /// `grid-template-columns`' third column, where the document row is the
  /// one shape that fixes it: four icons in 108 pt, so every document row
  /// in a list breaks its title at the same place.
  public var trailingColumn: CGFloat? {
    self == .doc ? Grid.actionColumn : nil
  }

  /// The title's type role, which is one role: every shape left draws its
  /// title at the row's own size.
  public var titleRole: TypeRole { Typography.rowTitle }

  /// Where the row's three columns meet.
  ///
  /// A two-line row hangs them from the top, so the glyph sits against the
  /// title and not between the two lines. A one-line row has nothing to
  /// hang from: its words, its glyph and whatever is at its trailing edge
  /// are one line, and a 24 pt icon beside a 16 pt line of words sat 6 pt
  /// under the words it belongs to: testAOneLineRowCentresWhatIsOnIt.
  public var crossAlignment: VerticalAlignment {
    switch self {
    case .two, .doc: return .top
    case .plain, .compact: return .center
    }
  }

  /// `-webkit-line-clamp`: a two-line row's sub-line wraps to a second
  /// line before it is cut, which is what the two-line shapes are for. The
  /// one-line shapes have no second line to wrap into.
  ///
  /// The row grows with it: a hold's sub-line names the rule and then where
  /// the text is waiting, and the half that was cut off the end was the half
  /// that said what to do about it.
  public var subLines: Int {
    switch self {
    case .two, .doc: return 2
    case .plain, .compact: return 1
    }
  }
}

/// One row.
public struct RowView<Leading: View, Trailing: View>: View {
  public let shape: RowShape
  public let title: String
  /// The verb in front of the title, where the row has one: NOW's watch
  /// says "Live stopped" over the name it is watching. It is drawn whole
  /// and the name is measured in what it leaves.
  public let titlePrefix: String?
  /// The room the title is measured in. `nil` is the narrowest column in
  /// the panel, the document row's, so one name reads the same in every
  /// section; a row that opens with a verb hands its own.
  public let titleColumn: CGFloat?
  /// The words the title area carries for a pointer that rests on it, where
  /// the row itself is a press. A row whose press opens a document says so
  /// in the press's own words, which an icon carried once: NowPress.tip.
  public let titleTip: String?
  /// What a press held on the title area does, where the row has a menu.
  /// It is the title's and not the row's: the four icons are a column of
  /// their own, and a slow click on one of them is that icon's press alone.
  public let titleHeld: (() -> Void)?
  /// A click on the title, which opens the document the row names.
  ///
  /// It sits on the title line with the held press, because a long press
  /// swallows the click under it: a tap on one view and a hold on another
  /// would make a click on the name of a row with a menu do nothing. The
  /// quiet parts of the row press through the row's own box, which is
  /// `opensDocument` in the list: DocumentPress.open.
  public let titleTap: (() -> Void)?
  /// The row uncut, which a pointer resting on the middle column brings up.
  /// A row that has none carries none: NOW's rows say their words in a
  /// bubble, and the card is DOCUMENTS' own.
  public let card: DetailCardModel?
  /// The card held open with no pointer, which only a still ever asks for.
  public let forcedCard: Bool
  public let value: String?
  public let sub: String?
  /// The title's colour, where the row is not drawn on the panel's own
  /// surface: a question inside the hold card's tint is the plate's ink and
  /// not the panel's. `nil` is the shape's own.
  public let titleInk: Swatch?
  public let gone: Bool
  public let hovering: Bool
  public let busy: Bool
  public let leading: Leading
  public let trailing: Trailing

  @Environment(\.colorScheme) private var scheme
  @State private var pointerInside = false
  /// Whether this row's card is up, which three gestures on the same words
  /// decide between them: Components/CardRest.
  @State private var rest = CardRest()

  public init(
    _ shape: RowShape = .plain,
    title: String,
    titlePrefix: String? = nil,
    titleColumn: CGFloat? = nil,
    titleTip: String? = nil,
    titleHeld: (() -> Void)? = nil,
    titleTap: (() -> Void)? = nil,
    card: DetailCardModel? = nil,
    forcedCard: Bool = false,
    value: String? = nil,
    sub: String? = nil,
    titleInk: Swatch? = nil,
    gone: Bool = false,
    hovering: Bool = false,
    busy: Bool = false,
    @ViewBuilder leading: () -> Leading,
    @ViewBuilder trailing: () -> Trailing
  ) {
    self.shape = shape
    self.title = title
    self.titlePrefix = titlePrefix
    self.titleColumn = titleColumn
    self.titleTip = titleTip
    self.titleHeld = titleHeld
    self.titleTap = titleTap
    self.card = card
    self.forcedCard = forcedCard
    self.value = value
    self.sub = sub
    self.titleInk = titleInk
    self.gone = gone
    self.hovering = hovering
    self.busy = busy
    self.leading = leading()
    self.trailing = trailing()
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return HStack(alignment: shape.crossAlignment, spacing: Grid.columnGap) {
      // The glyph column, as tall as the title's own line, so a glyph beside
      // a two-line row sits against the title and not between the two
      // lines. It is the glyph's own 16 pt in every shape, which is the
      // panel's one left edge.
      leading
        .frame(
          width: shape.leadingColumn, height: Typography.rowTitle.lineHeight,
          alignment: .trailing
        )
        .metricProbe("row.leading")

      VStack(alignment: .leading, spacing: 0) {
        HStack(alignment: .firstTextBaseline, spacing: Grid.columnGap) {
          MiddleTitle(
            title, prefix: titlePrefix, role: shape.titleRole,
            column: titleColumn ?? MiddleTitle.narrowestColumn,
            gone: gone, quietInk: titleInk)
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
        .metricProbe("row.title")
        .tooltip(titleTip ?? "", align: .leading)
        .opensDocument(tapTellingTheCard)
        .heldPress(heldTellingTheCard)

        if let sub {
          Text(sub)
            .font(Typography.caption.font)
            .lineLimit(shape.subLines)
            .multilineTextAlignment(.leading)
            .fixedSize(horizontal: false, vertical: true)
            .foregroundStyle((subInk ?? palette.textTertiary).color)
            .frame(maxWidth: .infinity, alignment: .leading)
            .metricProbe("row.sub")
        }
      }
      // The two lines the card says whole are the two lines it rests over.
      // The icon column is outside it, so a pointer that moved on to an
      // icon has left the card and the icon's own words take the layer.
      .detailCard(card, rest: $rest, forced: forcedCard)

      // `.row.is-busy .acts{display:none}`: while a write of this row's own
      // is in flight the row carries no action at all, rather than one that
      // is off.
      if !busy {
        trailing
          .frame(width: shape.trailingColumn, alignment: .trailing)
          .metricProbe("row.trailing")
          .frame(maxHeight: .infinity, alignment: .center)
          // `.acts{margin-right:-4px}`: the icons and not their boxes meet
          // the column every time and every value in the panel ends on.
          .padding(.trailing, -Grid.actionOverhang)
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

  /// The click, with the card told about it first.
  ///
  /// A card is an answer to a question the click has just stopped asking:
  /// the browser is coming up, and a plate left behind over the row would
  /// be the panel still explaining itself.
  var tapTellingTheCard: (() -> Void)? {
    guard let titleTap else { return nil }
    let press: @MainActor () -> Void = {
      rest.after(.click)
      titleTap()
    }
    return press
  }

  /// The held press, with the card told first and the menu opened on the
  /// next turn of the main queue.
  ///
  /// The turn is the whole point: `NSMenu.popUp` tracks the menu itself and
  /// does not return until it closes, so a menu opened in the same turn as
  /// the card's removal would stand over a card nothing had redrawn. It is
  /// here and not in `RowMenu.held`, because the card is the row's and the
  /// menu knows nothing about it.
  var heldTellingTheCard: (() -> Void)? {
    guard let titleHeld else { return nil }
    let press: @MainActor () -> Void = {
      rest.after(.heldPress)
      Task { titleHeld() }
    }
    return press
  }

  /// The sub-line's colour on a tinted row, which is the plate's quiet ink.
  var subInk: Swatch? {
    guard titleInk != nil else { return nil }
    return Palette.of(scheme).holdInkQuiet
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
    titlePrefix: String? = nil,
    titleColumn: CGFloat? = nil,
    titleTip: String? = nil,
    titleHeld: (() -> Void)? = nil,
    titleTap: (() -> Void)? = nil,
    card: DetailCardModel? = nil,
    forcedCard: Bool = false,
    value: String? = nil,
    sub: String? = nil,
    gone: Bool = false,
    hovering: Bool = false
  ) {
    self.init(
      shape, title: title, titlePrefix: titlePrefix, titleColumn: titleColumn,
      titleTip: titleTip, titleHeld: titleHeld, titleTap: titleTap, card: card,
      forcedCard: forcedCard, value: value, sub: sub, gone: gone, hovering: hovering,
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
    titlePrefix: String? = nil,
    titleColumn: CGFloat? = nil,
    titleTip: String? = nil,
    titleHeld: (() -> Void)? = nil,
    titleTap: (() -> Void)? = nil,
    card: DetailCardModel? = nil,
    forcedCard: Bool = false,
    value: String? = nil,
    sub: String? = nil,
    titleInk: Swatch? = nil,
    gone: Bool = false,
    hovering: Bool = false,
    busy: Bool = false,
    @ViewBuilder trailing: () -> Trailing
  ) {
    self.init(
      shape, title: title, titlePrefix: titlePrefix, titleColumn: titleColumn,
      titleTip: titleTip, titleHeld: titleHeld, titleTap: titleTap, card: card,
      forcedCard: forcedCard, value: value, sub: sub, titleInk: titleInk, gone: gone,
      hovering: hovering, busy: busy,
      leading: { StatusGlyph(glyph, tone: tone) }, trailing: trailing)
  }
}
