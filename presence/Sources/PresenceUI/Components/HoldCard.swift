// The hold card: the one thing in the panel that is a debt to the person.
//
// `.hold` in docs/design/2026-10-17-presence-screens-v3.html is a tinted
// plate rather than the amber block of the second version: a card inside the
// list, 10 pt by 12 pt on a seven-percent amber with a half-pixel edge of
// its own. It holds four things: the pause glyph, how long it has been
// waiting, what is held, and where the words are. Then the two buttons, which
// are the panel's only filled one and a plain one beside it.
//
// The second button says Dismiss. The screens file labels it "Release
// without writing", and the specification's decision table says the panel
// never says Release: dismissing is the app's own doing, the card and the
// glyph go, nothing is written either way, and the hold itself lives on in
// the chat until its own `until`. Releasing is a thing only the chat can do, because
// only the chat can say what it would write. So the card's quiet line names
// the chat, and the word Release is not in this file.
//
// The plate is in this file and the stuck card uses it, because the two are
// the same plate in two tints.
//
// CardTests states the words, the tints and every padding as literals.

import SwiftUI

/// What a hold card draws. Plain values: nothing here reads a record.
public struct HoldCardModel: Equatable, Sendable {
  /// `Waiting for you · 14 min`, which the words file builds.
  public let title: String
  /// What is held, and under which rule.
  public let sub: String
  /// Where the text is, which is the chat and never the panel.
  public let quiet: String
  /// The two lines the card shows instead while it asks.
  public let question: String
  public let consequence: String

  public init(
    title: String, sub: String, quiet: String, question: String = "", consequence: String = ""
  ) {
    self.title = title
    self.sub = sub
    self.quiet = quiet
    self.question = question
    self.consequence = consequence
  }
}

/// One hold, as a card.
public struct HoldCard: View {
  /// The two words on it, in the order they are drawn. Release is not one of
  /// them (the decision table, over the screens file's own label).
  public static let words: [ButtonWord] = [.openChat, .dismiss]

  /// `.row-actions{gap:6px}` and the hold's own inline `margin-top:10px`.
  public static let actionGap: CGFloat = 6
  public static let actionTop: CGFloat = 10
  /// The quiet line's own inline `padding:6px 0 0 24px`.
  public static let quietTop: CGFloat = 6
  /// Where the card's second and third lines start: the glyph's column and
  /// the row's gap, so everything under the title is aligned with it.
  public static let bodyInset: CGFloat = Grid.glyphSize + Grid.columnGap

  public let model: HoldCardModel
  /// Whether the card is asking rather than offering.
  public let asking: Bool
  public let onOpenChat: () -> Void
  public let onDismiss: () -> Void
  public let onConfirm: () -> Void
  public let onKeep: () -> Void

  @Environment(\.colorScheme) private var scheme

  public init(
    model: HoldCardModel,
    asking: Bool = false,
    onOpenChat: @escaping () -> Void = {},
    onDismiss: @escaping () -> Void = {},
    onConfirm: @escaping () -> Void = {},
    onKeep: @escaping () -> Void = {}
  ) {
    self.model = model
    self.asking = asking
    self.onOpenChat = onOpenChat
    self.onDismiss = onDismiss
    self.onConfirm = onConfirm
    self.onKeep = onKeep
  }

  /// `.hold{background:rgba(217,160,60,.07)}` and `#fff6e5` on white.
  public static func tint(in palette: Palette) -> Swatch { palette.holdTint }

  /// `box-shadow:inset 0 0 0 .5px rgba(217,160,60,.22)`, and the same ring
  /// outside the plate in the light block.
  public static func edge(in palette: Palette) -> Swatch { palette.holdTintEdge }

  /// `.hold .row-title`: the panel's brightest ink on black, and the warm
  /// plate's own on white, where `--t1` would be too dark to read as amber.
  public static func titleInk(in palette: Palette) -> Swatch { palette.holdInk }

  /// `.hold .row-sub`: tertiary on black, the plate's ink on white.
  public static func subInk(in palette: Palette) -> Swatch { palette.holdInkQuiet }

  /// The quiet line's own inline `color:var(--t2)`, and the plate's ink on
  /// white, which is the one place the three lines are the same colour.
  public static func quietInk(in scheme: ColorScheme) -> Swatch {
    let palette = Palette.of(scheme)
    return scheme == .dark ? palette.textSecondary : palette.holdInk
  }

  /// What the question is drawn in, which is the plate's own ink and the
  /// same ink the offer's title uses. The body draws through it rather than
  /// beside it, so a test reading this reads what the card handed the
  /// confirm row: testTheHoldConfirmUsesThePlatesOwnInk.
  func questionInk(in palette: Palette) -> Swatch { Self.titleInk(in: palette) }

  /// And the consequence's, which is the quiet line's.
  func consequenceInk(in scheme: ColorScheme) -> Swatch { Self.quietInk(in: scheme) }

  public var body: some View {
    let palette = Palette.of(scheme)
    return Group {
      if asking {
        // `.row:has(.confirm)`: the question takes the card over, so the
        // card carries no other action while it is asking.
        //
        // The consequence is a line of its own and not the row's sub-line,
        // which is what the header's Sign out does and for the same reason: the
        // pair of buttons takes the right-hand column, and a sentence
        // squeezed into what is left is cut after its first clause.
        //
        // The row negates the plate's own padding, so its 12 pt line the
        // question up with the plate's edge; the consequence keeps the
        // plate's padding and takes the body's inset on top of it, which is
        // where the question above it starts.
        VStack(alignment: .leading, spacing: 0) {
          ConfirmRow(
            title: model.question, glyph: .pause, tone: .hold,
            danger: .dismiss, titleInk: questionInk(in: palette),
            onConfirm: onConfirm, onKeep: onKeep
          )
          .padding(.horizontal, -Grid.rowInset)
          QuietLine(
            model.consequence, inset: Self.bodyInset, ink: consequenceInk(in: scheme))
        }
      } else {
        offer(in: palette)
      }
    }
    .cardPlate(tint: Self.tint(in: palette), edge: Self.edge(in: palette))
    .metricProbe("hold.card")
  }

  @ViewBuilder
  private func offer(in palette: Palette) -> some View {
    VStack(alignment: .leading, spacing: 0) {
      // `.hold .row{padding:0;min-height:0}`: the plate is the inset, so the
      // line is drawn here rather than through RowView, which would add the
      // row's own 12 pt and the panel's own ink.
      HStack(alignment: .top, spacing: Grid.columnGap) {
        StatusGlyph(.pause, tone: .hold)
          .frame(width: Grid.glyphSize, height: Typography.rowTitle.lineHeight)
        VStack(alignment: .leading, spacing: 2) {
          Text(model.title)
            .font(Typography.titleEmphasis.font)
            .foregroundStyle(Self.titleInk(in: palette).color)
            .metricProbe("hold.title")
          Text(model.sub)
            .font(Typography.caption.font)
            .fixedSize(horizontal: false, vertical: true)
            .foregroundStyle(Self.subInk(in: palette).color)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
      }

      Text(model.quiet)
        .font(Typography.rowValue.font)
        .fixedSize(horizontal: false, vertical: true)
        .foregroundStyle(Self.quietInk(in: scheme).color)
        .padding(.top, Self.quietTop)
        .padding(.leading, Self.bodyInset)
        .frame(maxWidth: .infinity, alignment: .leading)
        .metricProbe("hold.quiet")

      HStack(spacing: Self.actionGap) {
        TextButton(.openChat, kind: .primary, action: onOpenChat)
          .metricProbe("hold.open-chat")
        TextButton(.dismiss, kind: .plain, action: onDismiss)
          .metricProbe("hold.dismiss")
      }
      .padding(.top, Self.actionTop)
      .padding(.leading, Self.bodyInset)
    }
  }
}

/// The plate a card is drawn on: `.hold` and `.stuck`, which differ in their
/// tint and in nothing else.
public struct CardPlate: ViewModifier {
  public let tint: Swatch
  public let edge: Swatch

  public init(tint: Swatch, edge: Swatch) {
    self.tint = tint
    self.edge = edge
  }

  public func body(content: Content) -> some View {
    content
      .padding(.vertical, Grid.cardInset)
      .padding(.horizontal, Grid.rowInset)
      .frame(maxWidth: .infinity, alignment: .leading)
      .background(
        RoundedRectangle(cornerRadius: Grid.cardRadius, style: .continuous)
          .fill(tint.color)
      )
      .overlay(
        RoundedRectangle(cornerRadius: Grid.cardRadius, style: .continuous)
          .strokeBorder(edge.color, lineWidth: Grid.hairline)
      )
      // `margin:2px 0`: a card in a list of rows needs the gap a row does
      // not, so two cards are two cards.
      .padding(.vertical, 2)
  }
}

extension View {
  /// Draw this on a card's plate.
  public func cardPlate(tint: Swatch, edge: Swatch) -> some View {
    modifier(CardPlate(tint: tint, edge: edge))
  }
}
