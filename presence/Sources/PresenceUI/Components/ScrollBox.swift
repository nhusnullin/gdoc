// The box an opened section scrolls inside, and the word that opens it.
//
// `.scrollbox` in docs/design/2026-10-17-presence-screens-v3.html is a cap
// and two decorations: 236 pt of height with the rest hidden, a 4 by 88 pt
// thumb 3 pt in from the right and 6 pt down from the top, and a 28 pt fade
// from the surface's colour at its foot. The panel is capped at 600 pt, so a
// section that opens scrolls inside itself rather than growing the panel past
// the screen.
//
// The thumb is a decoration and not a scroller: it says the list goes on,
// at the one size and the one place the screens file draws it. The platform's
// own scroller is hidden, because it is grey, wide and outside the panel's
// look.
//
// Neither decoration appears where the list already fits: a thumb on a list
// of three says there is more below when there is not. And one thumb per
// edge: where a box inside this one is scrolling, the inner list keeps the
// thumb and the outer box keeps only its fade.
//
// One measurement decides the inside of this file, and it is in
// docs/v2/MEASURED.md: `ImageRenderer` lays a `ScrollView` out and draws it
// empty, every time, even with one row inside. So the box carries a drawn
// mode, turned on by `rendering()`, which clips a plain stack instead. The
// mode is an environment value rather than an argument, because the scenes
// render a whole panel and cannot reach a box inside it.
//
// ScrollTests states the cap, the thumb, the fade and the two words, each as
// a literal, and pins the measurement.

import SwiftUI

/// How a tree is being drawn.
public enum RenderMode: Sendable {
  /// On the screen, where a box scrolls and the person reaches the rest.
  case live
  /// Into an image, as the person first sees it: the box clips at its cap.
  case clipped
  /// Into an image for somebody reviewing it, where nothing may be hidden:
  /// no cap, no clip, every row drawn. A reviewer cannot scroll a PNG, and
  /// a screen whose SETTINGS is below the cut is a screen nobody checked.
  case whole
}

private struct RenderModeKey: EnvironmentKey {
  static let defaultValue = RenderMode.live
}

extension EnvironmentValues {
  /// How this tree is drawn. `ScrollBox` reads it because the renderer
  /// draws no `ScrollView`, and `PanelView` because a whole render is not
  /// capped.
  public var renderMode: RenderMode {
    get { self[RenderModeKey.self] }
    set { self[RenderModeKey.self] = newValue }
  }

  /// True while a render is in progress, in either of its two modes.
  public var isRendering: Bool { renderMode != .live }
}

extension View {
  /// Draw this tree for an image: the scroll boxes in it clip a stack.
  public func rendering(_ mode: RenderMode = .clipped) -> some View {
    environment(\.renderMode, mode)
  }
}

/// What the content inside a box came to: how tall it is, and how deep the
/// deepest box that overflows under it sits.
///
/// Both are reported upwards so a box can decide, in the same layout pass,
/// whether it overflows and whether a box inside it is already saying so.
struct ScrollReport: Equatable, Sendable {
  var height: CGFloat = 0
  /// The depth of the deepest box whose own list is past its cap, or `nil`
  /// where nothing under here overflows at all.
  var overflowing: Int?
}

private struct ScrollContent: PreferenceKey {
  static let defaultValue = ScrollReport()

  static func reduce(value: inout ScrollReport, nextValue: () -> ScrollReport) {
    let next = nextValue()
    value.height = max(value.height, next.height)
    value.overflowing = [value.overflowing, next.overflowing].compactMap { $0 }.max()
  }
}

/// How deep in the boxes a view sits. The panel's own body is the first box
/// and an opened section inside it is the second, so a box knows whether a
/// box under it is reporting its own overflow or somebody else's.
private struct ScrollDepthKey: EnvironmentKey {
  static let defaultValue = 0
}

extension EnvironmentValues {
  var scrollDepth: Int {
    get { self[ScrollDepthKey.self] }
    set { self[ScrollDepthKey.self] = newValue }
  }
}

/// A list capped at the panel's scroll height, with the thumb and the fade.
public struct ScrollBox<Content: View>: View {
  /// `max-height:236px`, which the caller may lower and never raises.
  public let height: CGFloat
  public let content: Content

  @Environment(\.colorScheme) private var scheme
  @Environment(\.renderMode) private var renderMode
  @Environment(\.scrollDepth) private var depth

  public init(height: CGFloat = Grid.scrollBoxHeight, @ViewBuilder content: () -> Content) {
    self.height = height
    self.content = content()
  }

  /// `.scrollbox::after{right:3px;top:6px;width:4px;height:88px;
  /// border-radius:2px}`. The height is the one the file drew, over the one
  /// list it drew: `thumbLength` is what any other list gets.
  public static var thumbWidth: CGFloat { 4 }
  public static var thumbHeight: CGFloat { 88 }
  public static var thumbInset: CGFloat { 3 }
  public static var thumbTop: CGFloat { 6 }
  public static var thumbRadius: CGFloat { 2 }

  public var body: some View {
    let palette = Palette.of(scheme)
    let capped = renderMode != .whole
    return
      box
      .frame(maxHeight: capped ? height : .infinity, alignment: .top)
      .clipped()
      .overlayPreferenceValue(ScrollContent.self) { report in
        if capped, report.height > height {
          decoration(
            palette, measured: report.height,
            thumb: (report.overflowing ?? depth) <= depth)
        }
      }
      // The height stops here, so a box inside this one reports its list to
      // itself and not upwards: the reduce is a max, and the panel's own
      // body would otherwise take the length of an opened section's list
      // for its own and draw a thumb and a fade over rows that all fit:
      // testABoxInsideABoxDecoratesOnlyItsOwnContent. Which depth overflows
      // carries on up, because that is the one thing the box above has to
      // know: testOnlyTheInnerBoxCarriesAThumbOnE1.
      .transformPreference(ScrollContent.self) { $0.height = 0 }
      .metricProbe("scrollbox")
  }

  /// The rows, shown in a scroller or clipped in a stack.
  @ViewBuilder private var box: some View {
    if renderMode == .live {
      ScrollView(.vertical) { measured }
        .scrollIndicators(.hidden)
    } else {
      measured.fixedSize(horizontal: false, vertical: true)
    }
  }

  private var measured: some View {
    content
      .frame(maxWidth: .infinity, alignment: .leading)
      .environment(\.scrollDepth, depth + 1)
      .background(
        GeometryReader { geometry in
          Color.clear.preference(
            key: ScrollContent.self,
            value: ScrollReport(
              height: geometry.size.height,
              overflowing: geometry.size.height > height ? depth : nil))
        }
      )
  }

  /// How long the thumb is for a list of the height named.
  ///
  /// As much of the track as the box is of the list, which is the one thing
  /// a thumb says. The screens file draws 88 pt over its 236 pt box, which
  /// is a list of about 630 pt; a fixed 88 pt under a list of any other
  /// length says a number that is not true, and principle 5 is that the
  /// panel states facts. The floor keeps it a thumb rather than a dot.
  public static func thumbLength(box: CGFloat, measured: CGFloat) -> CGFloat {
    let track = box - 2 * thumbTop
    guard measured > box else { return track }
    return max(thumbFloor, (track * box / measured).rounded())
  }

  /// The shortest a thumb is drawn, whatever is under it.
  public static var thumbFloor: CGFloat { 24 }

  /// The thumb and the fade, over the rows and under no pointer.
  ///
  /// One thumb per edge: a box with a scrollable box inside it keeps its
  /// fade, which is what says there is more below, and leaves the thumb to
  /// the list the pointer is over. Two thumbs 3 pt apart on one edge are
  /// two thumbs nobody can tell apart.
  private func decoration(_ palette: Palette, measured: CGFloat, thumb: Bool) -> some View {
    ZStack(alignment: .topTrailing) {
      LinearGradient(
        colors: [palette.surface.color.opacity(0), palette.surface.color],
        startPoint: .top, endPoint: .bottom
      )
      .frame(height: Grid.fadeHeight)
      .metricProbe("scrollbox.fade")
      .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottom)
      if thumb {
        RoundedRectangle(cornerRadius: Self.thumbRadius)
          .fill(palette.scrollThumb.color)
          .frame(
            width: Self.thumbWidth,
            height: Self.thumbLength(box: height, measured: measured)
          )
          .metricProbe("scrollbox.thumb")
          .padding(.top, Self.thumbTop)
          .padding(.trailing, Self.thumbInset)
      }
    }
    .allowsHitTesting(false)
  }
}

/// How much of a list is on the screen, and the word that changes it.
///
/// `MORE('Show 3 more')` closed and `MORE('Show less',true)` open. The
/// numbers are the section's own: DOCUMENTS shows five, TODAY shows eight.
/// Where nothing is hidden there is no word, because a row that promises
/// nothing is a row.
public struct ShowMore: Equatable, Sendable {
  public let total: Int
  public let collapsed: Int
  public let isOpen: Bool

  public init(total: Int, collapsed: Int, isOpen: Bool = false) {
    self.total = total
    self.collapsed = collapsed
    self.isOpen = isOpen
  }

  /// How many rows are drawn.
  public var visible: Int {
    isOpen ? total : min(collapsed, total)
  }

  /// How many are not.
  public var hidden: Int {
    max(0, total - visible)
  }

  /// The same list, the other way round.
  public var toggled: ShowMore {
    ShowMore(total: total, collapsed: collapsed, isOpen: !isOpen)
  }

  /// The chevron row's words, or nothing where the whole list is on the
  /// screen already.
  public var title: String? {
    if isOpen { return total > collapsed ? "Show less" : nil }
    return hidden > 0 ? "Show \(hidden) more" : nil
  }
}

/// A list of rows under its own Show more, scrolling inside the box once it
/// is open.
///
/// The view holds no state: how much is on the screen is `isOpen`, which the
/// panel's model owns, because an expanded section survives a refold.
public struct ShowMoreList<Item, Row: View>: View {
  public let items: [Item]
  public let collapsed: Int
  public let isOpen: Bool
  public let toggle: () -> Void
  private let row: (Item) -> Row

  public init(
    _ items: [Item], collapsed: Int, isOpen: Bool = false,
    toggle: @escaping () -> Void = {}, @ViewBuilder row: @escaping (Item) -> Row
  ) {
    self.items = items
    self.collapsed = collapsed
    self.isOpen = isOpen
    self.toggle = toggle
    self.row = row
  }

  /// What this list shows and what its word says.
  public var state: ShowMore {
    ShowMore(total: items.count, collapsed: collapsed, isOpen: isOpen)
  }

  public var body: some View {
    VStack(spacing: 0) {
      if isOpen {
        ScrollBox { rows }
      } else {
        rows
      }
      if let title = state.title {
        ChevronRow(title, isOpen: isOpen, action: toggle)
      }
    }
    .metricProbe("showmore")
  }

  private var rows: some View {
    VStack(spacing: 0) {
      ForEach(Array(items.prefix(state.visible).enumerated()), id: \.offset) { _, item in
        row(item)
      }
    }
  }
}
