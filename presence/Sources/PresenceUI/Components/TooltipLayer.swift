// The one tooltip the panel draws, and where it goes.
//
// Every action in the panel is an icon, so the words that say what an icon
// does are the tooltip, and the tooltip is not a nicety: it is the label.
// The platform's own help tag is not used, because it is slow, grey and
// outside the panel's look; `[data-tip]::after` in
// docs/design/2026-10-17-presence-screens-v3.html is a plate of the panel's
// own, and this file is that rule.
//
// One layer draws it, over the whole panel, which is why a carrier reports
// its box upwards through `anchorPreference` rather than drawing a bubble of
// its own: a bubble inside a row would be clipped by the row, and two
// bubbles would overlap. The layer takes the last request, or the forced one
// where there is one.
//
// Nothing here waits for the pointer except `Tooltip.shows`, which is the
// half second written down as a function so a test can state it.
//
// ControlTests states the delay, the three alignments, the slide and the
// flip, each as a literal.

import SwiftUI

/// Where a bubble sits under the thing it describes.
public enum TooltipAlign: String, Sendable, CaseIterable {
  /// `[data-tip]::after{left:50%;transform:translateX(-50%)}`: the default,
  /// centred under the anchor.
  case centred
  /// `.acts [data-tip]::after{left:auto;right:0}`: an action's bubble hangs
  /// off its own right edge, which is how four icons in 108 pt keep their
  /// words inside the panel.
  case trailing
  /// `.row-title[data-tip]::after{left:0}`: a title's bubble starts where
  /// the title does.
  case leading

  /// `top:calc(100% + 6px)`, and 4 under a row title.
  public var gap: CGFloat {
    self == .leading ? 4 : 6
  }
}

/// Where the layer put a bubble, and whether it had to go over the anchor.
public struct TooltipPlacement: Equatable, Sendable {
  public let origin: CGPoint
  public let above: Bool

  public init(origin: CGPoint, above: Bool) {
    self.origin = origin
    self.above = above
  }
}

/// The tooltip's own rules: its plate, its delay and its placement.
public enum Tooltip {
  /// `padding:3px 7px` and `border-radius:4px`.
  public static let paddingH: CGFloat = 7
  public static let paddingV: CGFloat = 3
  public static let radius: CGFloat = 4
  /// How close to the panel's edge a bubble may sit, which is the panel's
  /// own padding: a tooltip that touches the edge reads as a clipped one.
  public static let margin: CGFloat = Grid.unit
  /// How long the pointer rests first.
  public static let delay: TimeInterval = Timing.tooltipDelay

  /// Whether the words are drawn: the pointer's rest, or a forced one.
  ///
  /// The rest and not the entry, because a pointer that crossed a button on
  /// its way somewhere else is not asking what the button does.
  public static func shows(hoverSince: Date?, now: Date, forced: Bool) -> Bool {
    if forced { return true }
    guard let hoverSince else { return false }
    return now.timeIntervalSince(hoverSince) >= delay
  }

  /// Where the bubble goes inside the panel.
  ///
  /// Under the anchor by preference, slid along until it is inside the
  /// margin, and over the anchor when there is no room under it. The panel
  /// is a window with nothing beneath it, so a bubble that fell off the foot
  /// would be a bubble nobody reads.
  public static func place(
    anchor: CGRect, size: CGSize, in panel: CGSize, align: TooltipAlign
  ) -> TooltipPlacement {
    var x: CGFloat
    switch align {
    case .centred: x = anchor.midX - size.width / 2
    case .trailing: x = anchor.maxX - size.width
    case .leading: x = anchor.minX
    }
    let rightmost = max(margin, panel.width - margin - size.width)
    x = min(max(x, margin), rightmost)

    var y = anchor.maxY + align.gap
    var above = false
    if y + size.height > panel.height - margin {
      above = true
      y = max(anchor.minY - align.gap - size.height, margin)
    }
    return TooltipPlacement(origin: CGPoint(x: x, y: y), above: above)
  }
}

/// One view's request for the layer: the words, the alignment and its box.
public struct TooltipRequest: Sendable {
  public let words: String
  public let align: TooltipAlign
  public let forced: Bool
  public let anchor: Anchor<CGRect>
}

/// Every request under a view. The layer reads them at the panel's level.
public struct TooltipAnchors: PreferenceKey {
  public static let defaultValue: [TooltipRequest] = []

  public static func reduce(value: inout [TooltipRequest], nextValue: () -> [TooltipRequest]) {
    value.append(contentsOf: nextValue())
  }
}

/// The plate itself.
public struct TooltipBubble: View {
  public let words: String

  @Environment(\.colorScheme) private var scheme

  public init(_ words: String) {
    self.words = words
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    let shadow = palette.tooltipShadow
    return Text(words)
      .font(Typography.caption.font)
      .foregroundStyle(palette.text.color)
      .lineLimit(1)
      .padding(.horizontal, Tooltip.paddingH)
      .padding(.vertical, Tooltip.paddingV)
      .background(
        RoundedRectangle(cornerRadius: Tooltip.radius, style: .continuous)
          .fill(palette.tooltipSurface.color)
      )
      .overlay(
        RoundedRectangle(cornerRadius: Tooltip.radius, style: .continuous)
          .strokeBorder(palette.tooltipEdge.color, lineWidth: Grid.hairline)
      )
      .shadow(
        color: shadow.swatch.color, radius: shadow.radius, x: shadow.x, y: shadow.y
      )
      .metricProbe("tooltip")
  }
}

extension View {
  /// Carry words for the layer to draw after the pointer rests on this view.
  public func tooltip(_ words: String, align: TooltipAlign = .centred, forced: Bool = false)
    -> some View
  {
    modifier(TooltipCarrier(words: words, align: align, forced: forced))
  }

  /// Draw the panel's one tooltip over everything under it.
  public func tooltipLayer() -> some View {
    overlayPreferenceValue(TooltipAnchors.self) { requests in
      GeometryReader { geometry in
        if let request = requests.first(where: \.forced) ?? requests.last {
          TooltipPlacementLayout(
            anchor: geometry[request.anchor], align: request.align
          ) {
            TooltipBubble(request.words)
          }
        }
      }
      .allowsHitTesting(false)
    }
  }
}

/// The hover half: this view's own rest, and its box reported upwards.
struct TooltipCarrier: ViewModifier {
  let words: String
  let align: TooltipAlign
  let forced: Bool

  @State private var resting = false
  @State private var hovers = 0

  func body(content: Content) -> some View {
    content
      .onHover { inside in
        hovers += 1
        guard inside else {
          resting = false
          return
        }
        let entered = hovers
        Task { @MainActor in
          try? await Task.sleep(nanoseconds: UInt64(Tooltip.delay * 1_000_000_000))
          // The same hover: a pointer that left and came back is a new rest.
          if entered == hovers { resting = true }
        }
      }
      .anchorPreference(key: TooltipAnchors.self, value: .bounds) { anchor in
        guard !words.isEmpty, forced || resting else { return [] }
        return [
          TooltipRequest(words: words, align: align, forced: forced, anchor: anchor)
        ]
      }
  }
}

/// The placement, as a layout rather than as a measured offset: a layout
/// knows the bubble's size and the panel's box in the same pass, so a still
/// of the panel draws the tooltip where the app would.
struct TooltipPlacementLayout: Layout {
  let anchor: CGRect
  let align: TooltipAlign

  func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
    proposal.replacingUnspecifiedDimensions()
  }

  func placeSubviews(
    in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()
  ) {
    guard let bubble = subviews.first else { return }
    let size = bubble.sizeThatFits(.unspecified)
    let placed = Tooltip.place(anchor: anchor, size: size, in: bounds.size, align: align)
    bubble.place(
      at: CGPoint(x: bounds.minX + placed.origin.x, y: bounds.minY + placed.origin.y),
      proposal: ProposedViewSize(size))
  }
}
