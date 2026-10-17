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
// of three says there is more below when there is not.
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

/// Whether this tree is being drawn into an image rather than shown.
private struct RenderingKey: EnvironmentKey {
  static let defaultValue = false
}

extension EnvironmentValues {
  /// True while a render is in progress, which `ScrollBox` reads because the
  /// renderer draws no `ScrollView`.
  public var isRendering: Bool {
    get { self[RenderingKey.self] }
    set { self[RenderingKey.self] = newValue }
  }
}

extension View {
  /// Draw this tree for an image: the scroll boxes in it clip a stack.
  public func rendering(_ on: Bool = true) -> some View {
    environment(\.isRendering, on)
  }
}

/// How tall the content inside a box turned out, reported upwards so the box
/// can decide whether it overflows in the same layout pass.
private struct ScrollContentHeight: PreferenceKey {
  static let defaultValue: CGFloat = 0

  static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
    value = max(value, nextValue())
  }
}

/// A list capped at the panel's scroll height, with the thumb and the fade.
public struct ScrollBox<Content: View>: View {
  /// `max-height:236px`, which the caller may lower and never raises.
  public let height: CGFloat
  public let content: Content

  @Environment(\.colorScheme) private var scheme
  @Environment(\.isRendering) private var isRendering

  public init(height: CGFloat = Grid.scrollBoxHeight, @ViewBuilder content: () -> Content) {
    self.height = height
    self.content = content()
  }

  /// `.scrollbox::after{right:3px;top:6px;width:4px;height:88px;
  /// border-radius:2px}`.
  public static var thumbWidth: CGFloat { 4 }
  public static var thumbHeight: CGFloat { 88 }
  public static var thumbInset: CGFloat { 3 }
  public static var thumbTop: CGFloat { 6 }
  public static var thumbRadius: CGFloat { 2 }

  public var body: some View {
    let palette = Palette.of(scheme)
    return
      box
      .frame(maxHeight: height, alignment: .top)
      .clipped()
      .overlayPreferenceValue(ScrollContentHeight.self) { measured in
        if measured > height {
          decoration(palette)
        }
      }
      .metricProbe("scrollbox")
  }

  /// The rows, shown in a scroller or clipped in a stack.
  @ViewBuilder private var box: some View {
    if isRendering {
      measured.fixedSize(horizontal: false, vertical: true)
    } else {
      ScrollView(.vertical) { measured }
        .scrollIndicators(.hidden)
    }
  }

  private var measured: some View {
    content
      .frame(maxWidth: .infinity, alignment: .leading)
      .background(
        GeometryReader { geometry in
          Color.clear.preference(key: ScrollContentHeight.self, value: geometry.size.height)
        }
      )
  }

  /// The thumb and the fade, over the rows and under no pointer.
  private func decoration(_ palette: Palette) -> some View {
    ZStack(alignment: .topTrailing) {
      LinearGradient(
        colors: [palette.surface.color.opacity(0), palette.surface.color],
        startPoint: .top, endPoint: .bottom
      )
      .frame(height: Grid.fadeHeight)
      .metricProbe("scrollbox.fade")
      .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottom)
      RoundedRectangle(cornerRadius: Self.thumbRadius)
        .fill(palette.scrollThumb.color)
        .frame(width: Self.thumbWidth, height: Self.thumbHeight)
        .metricProbe("scrollbox.thumb")
        .padding(.top, Self.thumbTop)
        .padding(.trailing, Self.thumbInset)
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
