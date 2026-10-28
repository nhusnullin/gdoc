// The stuck card: a write that stopped, and what the world looks like now.
//
// `.stuck` in docs/design/2026-10-17-presence-screens-v3.html is the hold's
// plate in red: the exclamation glyph, the title of what stopped, the reason
// in two lines, and the icons pulled up to the top right corner. There are no
// buttons on it, because the two dead ones a card like this used to carry
// (open a document that was never created, copy a link that does not exist)
// were what made it read as buttons again.
//
// Two rules the card holds and nothing else does. The reason is the cause
// with its place first and the state of the world second, in 80 characters
// or fewer, which is what two lines at this width come to. And the icon that
// closes the card is the last one, so a pointer on its way to Open note never
// crosses it.
//
// Dismissing is the app's own: the card and the menu bar's exclamation go,
// and the document keeps its row. Nothing is undone, because nothing was
// done.
//
// CardTests states the ceiling, measures the two lines of the reason and
// the two of the title, and reads the order of the icons.

import SwiftUI

/// One action a card or a row carries: which icon, the words a pointer gets,
/// what state it is in, and what pressing it does.
///
/// It is a plain description with a closure, which is the whole of this
/// task's rule: a piece of the panel is handed values and functions, and
/// never a process, a file or a machine.
public struct IconAction: Identifiable {
  public let icon: Icon
  public let tip: String
  public let state: ActionState
  /// Held open with no pointer, which a still of the panel needs.
  public let forcedTooltip: Bool
  public let perform: () -> Void

  public var id: String { icon.rawValue }

  public init(
    _ icon: Icon, tip: String, state: ActionState = .live, forcedTooltip: Bool = false,
    perform: @escaping () -> Void = {}
  ) {
    self.icon = icon
    self.tip = tip
    self.state = state
    self.forcedTooltip = forcedTooltip
    self.perform = perform
  }
}

/// What a stuck card draws.
public struct StuckCardModel: Equatable, Sendable {
  /// `Publish stopped · Chapter 4`.
  public let title: String
  /// The cause with its place, then the state of the world.
  public let reason: String

  public init(title: String, reason: String) {
    self.title = title
    self.reason = reason
  }
}

/// One stopped write, as a card.
public struct StuckCard: View {
  /// How long a reason may be. Scene r2's rule, which the words file obeys
  /// and this card measures.
  public static let reasonCeiling = 80
  /// `-webkit-line-clamp:2`.
  public static let reasonLines = 2
  /// The reason's own width: the panel's 360 less its 8 either side, less
  /// the card's 12 either side, less the glyph and the gap after it. It is
  /// `grid-column:2/4`, so the icons above do not narrow it.
  public static let reasonColumn: CGFloat =
    Grid.panelWidth - 2 * Grid.unit - 2 * Grid.rowInset - Grid.glyphSize - Grid.columnGap
  /// `.stuck .row-sub{margin-top:2px}`.
  public static let reasonTop: CGFloat = 2
  /// Where the reason starts: under the title, past the glyph's column, which
  /// is what `grid-column:2/4` puts it at.
  public static let reasonInset: CGFloat = Grid.glyphSize + Grid.columnGap
  /// `.stuck .acts{margin:-4px -4px 0 0}`: the icons sit against the card's
  /// corner rather than beside the title.
  public static let actionsOutset: CGFloat = 4
  /// The words on the icon that closes the card.
  public static let dismissTip = ButtonWord.dismiss.rawValue

  /// The room one line of the title has: the column beside the icons for
  /// the first, and the card's full body width for the second, which is
  /// drawn under them. The row's own rule cuts each of them at that width,
  /// so a card and a row never show one title two ways:
  /// testACardCutsATitleByTheRowsOwnRule.
  public static func column(line: Int, actions: Int) -> CGFloat {
    line == 0 ? titleColumn(actions: actions) : reasonColumn
  }

  /// What the title's first line is drawn in: the reason's own width less
  /// the gap and the icons beside it, which is as little as 112 pt under
  /// five of them. A card with no icon has the whole width.
  public static func titleColumn(actions: Int) -> CGFloat {
    guard actions > 0 else { return reasonColumn }
    let icons =
      CGFloat(actions) * Grid.actionSize + CGFloat(actions - 1) * Grid.actionGap
      - actionsOutset
    return reasonColumn - Grid.columnGap - icons
  }

  /// The title over the card's two lines: what the column beside the icons
  /// holds, then the rest of it across the card.
  ///
  /// The whole title goes in, which is the one thing a card is for. A row
  /// cuts the middle out of a title and keeps its end, because a row is one
  /// line in a list of them; the card naming the write that stopped has a
  /// second line under the icons and uses it.
  ///
  /// The break is the title's own separator where there is one that fits,
  /// and the separator goes with it: a card's title is what stopped and the
  /// document it stopped on, so the two halves are the two lines and no line
  /// ends in a hanging dot. A title without one is broken between words.
  public static func titleSplit(_ title: String, actions: Int) -> [String] {
    let role = Typography.titleEmphasis
    let column = titleColumn(actions: actions)
    if MiddleTitle.width(title, role: role) <= column { return [title] }

    let halves = title.components(separatedBy: Sentences.separator)
    if halves.count > 1 {
      // The last separator whose first half still fits, so a title of three
      // parts keeps as much as it can on the line beside the icons.
      for cut in stride(from: halves.count - 1, through: 1, by: -1) {
        let head = halves.prefix(cut).joined(separator: Sentences.separator)
        if MiddleTitle.width(head, role: role) <= column {
          return [head, halves.dropFirst(cut).joined(separator: Sentences.separator)]
        }
      }
    }

    let words = title.split(separator: " ", omittingEmptySubsequences: false)
    var head: [Substring] = []
    for word in words {
      let next = (head + [word]).joined(separator: " ")
      if !head.isEmpty, MiddleTitle.width(next, role: role) > column { break }
      head.append(word)
    }
    // One word wider than the column: it is the first line anyway, cut by
    // the drawing, because a line of nothing is worse than a line that runs
    // out of room.
    if head.isEmpty { head = Array(words.prefix(1)) }
    let rest = words.dropFirst(head.count)
    guard !rest.isEmpty else { return [head.joined(separator: " ")] }
    return [head.joined(separator: " "), rest.joined(separator: " ")]
  }

  public let model: StuckCardModel
  /// The actions in the order they are drawn, the cross last where there is
  /// one. Built once so a test reads the same list the card draws.
  public let actions: [IconAction]
  /// The words the title carries where the card itself opens the document.
  public let titleTip: String?

  @Environment(\.colorScheme) private var scheme

  public init(
    model: StuckCardModel, actions: [IconAction], titleTip: String? = nil,
    onDismiss: (() -> Void)? = nil
  ) {
    self.model = model
    self.titleTip = titleTip
    var drawn = actions
    if let onDismiss {
      drawn.append(IconAction(.xmark, tip: Self.dismissTip, perform: onDismiss))
    }
    self.actions = drawn
  }

  /// Whether a reason is inside the ceiling.
  public static func fits(_ reason: String) -> Bool {
    reason.count <= reasonCeiling
  }

  /// The icons of a card, in order: what the caller handed over, and the
  /// cross after them when the card can be dismissed.
  public static func icons(_ actions: [IconAction], dismissable: Bool) -> [Icon] {
    actions.map(\.icon) + (dismissable ? [.xmark] : [])
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    // The title over two lines: the first beside the icons, the second
    // under them in the card's full width. With five live icons the column
    // beside them is about 112 pt, and two lines of that cut the middle out
    // of the title: the second line is where the rest of it goes.
    let title = Self.titleSplit(model.title, actions: actions.count)
    return VStack(alignment: .leading, spacing: 0) {
      HStack(alignment: .top, spacing: Grid.columnGap) {
        StatusGlyph(.exclamation, tone: .bad)
          .frame(width: Grid.glyphSize, height: Typography.rowTitle.lineHeight)
        line(title[0], column: Self.column(line: 0, actions: actions.count), in: palette)
          .frame(maxWidth: .infinity, alignment: .leading)
          .metricProbe("stuck.title")
          .tooltip(titleTip ?? "", align: .leading)
        HStack(spacing: Grid.actionGap) {
          ForEach(actions) { action in
            ActionButton(
              action.icon, tip: action.tip, state: action.state,
              forcedTooltip: action.forcedTooltip, action: action.perform)
          }
        }
        .padding(.top, -Self.actionsOutset)
        .padding(.trailing, -Self.actionsOutset)
        .metricProbe("stuck.acts")
      }

      if title.count > 1 {
        line(title[1], column: Self.column(line: 1, actions: actions.count), in: palette)
          .frame(maxWidth: .infinity, alignment: .leading)
          .metricProbe("stuck.title.second")
          .padding(.leading, Self.reasonInset)
      }

      Text(model.reason)
        .font(Typography.caption.font)
        .lineLimit(Self.reasonLines)
        .multilineTextAlignment(.leading)
        .fixedSize(horizontal: false, vertical: true)
        .foregroundStyle(palette.textTertiary.color)
        .frame(maxWidth: .infinity, alignment: .leading)
        .metricProbe("stuck.reason")
        .padding(.top, Self.reasonTop)
        .padding(.leading, Self.reasonInset)
    }
    .cardPlate(tint: palette.stuckTint, edge: palette.stuckTintEdge)
    .metricProbe("stuck.card")
  }

  /// One line of the title. The split already broke it between words, so
  /// what is cut here is a line that ran out of room on its own, and what
  /// is cut out of it is the middle: the end of a document's name is the
  /// part that tells two drafts apart.
  ///
  /// The cut is the row's own, through `MiddleTitle` at this line's width.
  /// The system's middle truncation does not know where a word ends, and on
  /// scene e3 it drew "Proposal: Expansio…her regions, draft v1", where
  /// "her regions" is a word that is not in the title.
  private func line(_ text: String, column: CGFloat, in palette: Palette) -> some View {
    MiddleTitle(text, role: Typography.titleEmphasis, column: column)
      .accessibilityLabel(model.title)
  }
}
