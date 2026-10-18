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
// Since 2026-10-18 the thumb is also a thing that comes and goes: on a live
// box it is drawn while the content moves and for one second after it
// stopped, and never otherwise, which is what every other list on this
// system does. The fade stays, because the fade is what says there is more
// below when nothing is moving. A still draws the thumb whatever happened,
// because a PNG cannot be scrolled and the screens file draws one.
//
// A box is capped in one of two ways. `.points` is its own number, which
// the opened sections and NOW use. `.given` is as tall as the stack above
// it has left, which is the panel's middle: `CappedHeight` is the one
// layout, so a box in a still and the same box on the screen are laid out
// by the same arithmetic. That matters because a plain frame cannot do
// either job: a frame around a fixed stack is rigid at its cap, and a frame
// around a scroll view is flexible from nothing to everything, so a stack
// of three would split the panel between them instead of giving the middle
// what is left.
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

/// A box as tall as its content allows and never taller than its cap, or as
/// tall as it is given where it has no cap of its own.
///
/// One layout for the box in a still and the box on the screen, because the
/// two have to be the same panel. A plain frame is not that: around a fixed
/// stack it is rigid at its cap, and around a scroll view it stretches from
/// nothing to everything, so a stack of three boxes split the panel evenly
/// between the two that could stretch. This reports its content's own height
/// where it is asked for nothing, and takes whatever height it is offered,
/// which is what makes the middle of the panel the one part that gives way.
struct CappedHeight: Layout {
  /// The most this box takes, or nothing where it takes what it is given.
  let cap: CGFloat?

  func sizeThatFits(
    proposal: ProposedViewSize, subviews: Subviews, cache: inout ()
  ) -> CGSize {
    let width = proposal.width ?? Grid.panelWidth
    let asked = ProposedViewSize(width: width, height: nil)
    let content = subviews.first?.sizeThatFits(asked).height ?? 0
    guard let cap else {
      return CGSize(width: width, height: proposal.height ?? content)
    }
    return CGSize(width: width, height: min(content, cap))
  }

  func placeSubviews(
    in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()
  ) {
    subviews.first?.place(
      at: CGPoint(x: bounds.minX, y: bounds.minY), anchor: .topLeading,
      proposal: ProposedViewSize(width: bounds.width, height: bounds.height))
  }
}

/// How tall a box is.
public enum ScrollCap: Equatable, Sendable {
  /// Its own number of points, which the content scrolls inside.
  case points(CGFloat)
  /// As tall as the stack above it has left for it, which the panel's
  /// middle is: the header, NOW and SETTINGS take theirs first.
  case given

  /// The cap in points, or nothing where the box takes what it is given.
  public var points: CGFloat? {
    if case .points(let height) = self { return height }
    return nil
  }
}

/// A list capped at the panel's scroll height, with the thumb and the fade.
public struct ScrollBox<Content: View>: View {
  /// `max-height:236px`, which the caller may lower and never raises, or
  /// the height the stack above hands over.
  public let cap: ScrollCap
  public let content: Content

  @Environment(\.colorScheme) private var scheme
  @Environment(\.renderMode) private var renderMode
  @Environment(\.scrollDepth) private var depth
  /// When this box's content last moved. Nothing but a scroll sets it, so a
  /// box nobody has touched carries no thumb.
  @State private var movedAt: Date?

  public init(height: CGFloat = Grid.scrollBoxHeight, @ViewBuilder content: () -> Content) {
    self.cap = .points(height)
    self.content = content()
  }

  public init(_ cap: ScrollCap, @ViewBuilder content: () -> Content) {
    self.cap = cap
    self.content = content()
  }

  /// The cap in points, where the box has one of its own.
  public var height: CGFloat? { cap.points }

  /// `.scrollbox::after{right:3px;top:6px;width:4px;height:88px;
  /// border-radius:2px}`. The height is the one the file drew, over the one
  /// list it drew: `thumbLength` is what any other list gets.
  public static var thumbWidth: CGFloat { 4 }
  public static var thumbHeight: CGFloat { 88 }
  public static var thumbInset: CGFloat { 3 }
  public static var thumbTop: CGFloat { 6 }
  public static var thumbRadius: CGFloat { 2 }

  /// How long the thumb stays after the content stopped moving.
  public static var thumbLife: TimeInterval { Timing.thumbLife }

  /// Whether the thumb is on the screen: while the content is moving, and
  /// for one second after it stopped. A list nobody has moved has none.
  public static func thumbIsShowing(movedAt: Date?, now: Date) -> Bool {
    guard let movedAt else { return false }
    return now.timeIntervalSince(movedAt) < thumbLife
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    let capped = renderMode != .whole
    return
      laidOut(capped: capped)
      .overlayPreferenceValue(ScrollContent.self) { report in
        // The box's own height, read off the drawing: a `.given` box learns
        // it no other way, and a `.points` box has already been clamped to
        // the shorter of its cap and its content.
        GeometryReader { geometry in
          if capped, report.height > geometry.size.height {
            decoration(
              palette, box: geometry.size.height, measured: report.height,
              thumb: (report.overflowing ?? depth) <= depth && thumbShows)
          }
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

  /// The box at the height its cap allows, with the rest cut off.
  ///
  /// A whole render is the reviewer's copy, where nothing may be hidden, so
  /// there is no cap and no clip over it at all.
  @ViewBuilder private func laidOut(capped: Bool) -> some View {
    if capped {
      CappedHeight(cap: cap.points) { box }
        .clipped()
    } else {
      box
    }
  }

  /// The rows, shown in a scroller or clipped in a stack.
  @ViewBuilder private var box: some View {
    if renderMode == .live {
      ScrollView(.vertical) { tracked }
        .scrollIndicators(.hidden)
        .coordinateSpace(.named(Self.space))
        .task(id: movedAt) { await putTheThumbAway() }
    } else {
      measured.fixedSize(horizontal: false, vertical: true)
    }
  }

  /// The name of this box's own space, which the content's place inside it
  /// is read in. A box inside a box takes the nearest one, which is its own.
  private static var space: String { "scrollbox" }

  /// Whether the thumb is drawn at all: a still draws it, and a live box
  /// draws it while the content is moving and for a second after.
  private var thumbShows: Bool {
    renderMode != .live || Self.thumbIsShowing(movedAt: movedAt, now: Date())
  }

  /// The content, with the one thing that says it moved: where its top edge
  /// sits inside this box. Nothing but a scroll changes that.
  private var tracked: some View {
    measured.background(alignment: .top) {
      GeometryReader { geometry in
        let top = geometry.frame(in: .named(Self.space)).minY
        Color.clear.onChange(of: top) { _, _ in movedAt = Date() }
      }
    }
  }

  /// One second after the last move, the thumb goes. Each move starts this
  /// again, because `.task(id:)` is cancelled and run afresh when the
  /// moment changes.
  ///
  /// A cancelled sleep returns and clears nothing. The cancel is the next
  /// move arriving, and a `try?` that dropped it put the thumb away in the
  /// same frame the content moved in: while a hand scrolled, the moment and
  /// nil took turns and the thumb flickered.
  @MainActor private func putTheThumbAway() async {
    guard movedAt != nil else { return }
    do {
      try await Task.sleep(for: .seconds(Self.thumbLife))
    } catch {
      return
    }
    movedAt = nil
  }

  private var measured: some View {
    content
      .frame(maxWidth: .infinity, alignment: .leading)
      .environment(\.scrollDepth, depth + 1)
      .background(
        GeometryReader { geometry in
          Color.clear.preference(
            key: ScrollContent.self, value: report(geometry.size.height))
        }
      )
  }

  /// What the content came to, and whether it is past a cap of its own.
  ///
  /// A box with no cap of its own reports no overflow. It is the panel's
  /// middle, which no other box encloses, and whether it carries a thumb is
  /// decided against the height the stack above handed it rather than
  /// against a number of its own.
  private func report(_ content: CGFloat) -> ScrollReport {
    guard let height = cap.points else { return ScrollReport(height: content) }
    return ScrollReport(height: content, overflowing: content > height ? depth : nil)
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
  private func decoration(
    _ palette: Palette, box: CGFloat, measured: CGFloat, thumb: Bool
  ) -> some View {
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
            height: Self.thumbLength(box: box, measured: measured)
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
