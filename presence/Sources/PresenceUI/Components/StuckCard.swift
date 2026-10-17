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
// the TODAY row stays. Nothing is undone, because nothing was done.
//
// CardTests states the ceiling, measures the two lines, and reads the order
// of the icons.

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
  /// The reason's own width: the panel's 320 less its 8 either side, less
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
  /// How many lines the title may take on a card.
  public static let titleLines = 2

  public let model: StuckCardModel
  /// The actions in the order they are drawn, the cross last where there is
  /// one. Built once so a test reads the same list the card draws.
  public let actions: [IconAction]

  @Environment(\.colorScheme) private var scheme

  public init(
    model: StuckCardModel, actions: [IconAction], onDismiss: (() -> Void)? = nil
  ) {
    self.model = model
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
    return VStack(alignment: .leading, spacing: 0) {
      HStack(alignment: .top, spacing: Grid.columnGap) {
        StatusGlyph(.exclamation, tone: .bad)
          .frame(width: Grid.glyphSize, height: Typography.rowTitle.lineHeight)
        // Two lines: with five live icons beside it the title has about
        // 95 pt, and on one line that is "Export stoppe… v1", a card that
        // no longer says which document stopped.
        MiddleTitle(model.title, role: Typography.titleEmphasis, lines: Self.titleLines)
          .frame(maxWidth: .infinity, alignment: .leading)
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
}
