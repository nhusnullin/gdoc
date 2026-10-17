// The scroll box, measured off the drawing, and the word that opens a list.
//
// The box is `.scrollbox` in
// docs/design/2026-10-17-presence-screens-v3.html: a cap of 236 pt, a 4 by
// 88 pt thumb 3 pt in from the right and 6 pt down from the top, and a 28 pt
// fade at its foot. Every number below is that file's, read by hand, and read
// back out of the render through MetricProbes.
//
// The first test also pins the measurement this component rests on, which is
// in docs/v2/MEASURED.md: `ImageRenderer` draws a `ScrollView` as an empty
// box, so the box clips a plain stack while it is being drawn into an image.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class ScrollTests: XCTestCase {
  /// Forty rows of titles of the lengths the hub really holds, so no real
  /// document's name is in the tree.
  private func rows(_ count: Int) -> [String] {
    (0..<count).map { index in "Pricing one-pager v\(index + 1)" }
  }

  private func plainRows(_ titles: [String]) -> some View {
    VStack(spacing: 0) {
      ForEach(Array(titles.enumerated()), id: \.offset) { index, title in
        RowView(.plain, glyph: .check, tone: .ok, title: title, value: "09:05")
          .metricProbe("item-\(index)")
      }
    }
  }

  // `.scrollbox{max-height:236px;overflow:hidden}` with its thumb and its
  // fade, and nothing of either where the list already fits.
  func testTheBoxCapsItsHeightAndScrollsInsideWithTheThumbAndTheFade() throws {
    XCTAssertEqual(Grid.scrollBoxHeight, 236)
    XCTAssertEqual(Grid.fadeHeight, 28)
    XCTAssertEqual(ScrollBox<EmptyView>.thumbWidth, 4)
    XCTAssertEqual(ScrollBox<EmptyView>.thumbHeight, 88)
    XCTAssertEqual(ScrollBox<EmptyView>.thumbInset, 3)
    XCTAssertEqual(ScrollBox<EmptyView>.thumbTop, 6)
    XCTAssertEqual(ScrollBox<EmptyView>.thumbRadius, 2)
    XCTAssertEqual(ScrollBox<EmptyView>.thumbFloor, 24)

    // The thumb is as much of the track as the box is of the list. 88 over
    // a 236 pt box is the list the screens file drew, about 630 pt of it.
    XCTAssertEqual(ScrollBox<EmptyView>.thumbLength(box: 236, measured: 630), 84)
    XCTAssertEqual(ScrollBox<EmptyView>.thumbLength(box: 236, measured: 472), 112)
    XCTAssertEqual(
      ScrollBox<EmptyView>.thumbLength(box: 236, measured: 10_000), 24,
      "a very long list stops at the floor rather than becoming a dot")
    XCTAssertEqual(
      ScrollBox<EmptyView>.thumbLength(box: 236, measured: 100), 224,
      "a list that fits fills the track, where no thumb is drawn at all")

    // `rgba(255,255,255,.22)` on black and `rgba(0,0,0,.22)` on white.
    XCTAssertEqual(Palette.dark.scrollThumb, Swatch(0xFF_FF_FF, alpha: 0.22))
    XCTAssertEqual(Palette.light.scrollThumb, Swatch(0x00_00_00, alpha: 0.22))

    // The measurement: a `ScrollView` under the renderer draws nothing at
    // all, which is why the box has a drawn mode. Forty rows in a plain
    // stack, clipped to the cap, carry ink; the same rows in a `ScrollView`
    // carry none.
    let forty = rows(40)
    let scrolled = ScrollView { plainRows(forty) }
      .frame(width: Grid.panelWidth, height: Grid.scrollBoxHeight)
    XCTAssertEqual(
      try inkedPixels(scrolled, scheme: .dark, side: Grid.scrollBoxHeight), 0,
      "the renderer drew a ScrollView, so the drawn mode is no longer needed")
    let clipped = ScrollBox { plainRows(forty) }
      .rendering()
      .frame(width: Grid.panelWidth)
    XCTAssertGreaterThan(try inkedPixels(clipped, scheme: .dark, side: Grid.scrollBoxHeight), 0)

    for scheme in [ColorScheme.dark, .light] {
      let measured = try probes(ScrollBox { plainRows(forty) }.rendering(), scheme: scheme)
      XCTAssertEqual(measured["scrollbox"]?.height, 236, "the cap in \(scheme)")
      XCTAssertEqual(measured["scrollbox"]?.width, Grid.panelWidth)
      // Forty 32 pt rows under a 236 pt box: the thumb is that fraction of
      // the 224 pt track, which is 41, and not the screens file's 88. The
      // file drew one list; a thumb that is 88 whatever is under it says a
      // number that is not true.
      XCTAssertEqual(
        measured["scrollbox.thumb"], CGSize(width: 4, height: 41), "the thumb in \(scheme)")
      XCTAssertEqual(measured["scrollbox.fade"]?.height, 28, "the fade in \(scheme)")
      XCTAssertEqual(measured["scrollbox.fade"]?.width, Grid.panelWidth)

      // Forty rows are laid out inside it, scrolled rather than dropped: the
      // last one reports its box even though the box shows seven.
      XCTAssertNotNil(measured["item-0"])
      XCTAssertNotNil(measured["item-39"], "the fortieth row is inside the box in \(scheme)")
    }

    // A list that fits takes its own height and carries neither decoration:
    // a thumb on a list of three says there is more below when there is not.
    let three = try probes(ScrollBox { plainRows(rows(3)) }.rendering(), scheme: .dark)
    XCTAssertEqual(three["scrollbox"]?.height, 3 * Grid.rowHeight)
    XCTAssertNil(three["scrollbox.thumb"], "three rows need no thumb")
    XCTAssertNil(three["scrollbox.fade"], "three rows need no fade")

    // The fade reaches the canvas rather than being a field nobody drew.
    let bare = plainRows(rows(3)).frame(width: Grid.panelWidth)
    let faded = ScrollBox { plainRows(forty) }.rendering().frame(width: Grid.panelWidth)
    XCTAssertNotEqual(try png(bare, scheme: .dark), try png(faded, scheme: .dark))
  }

  // The panel's own body is a box, and an opened section inside it is a
  // second one: DOCUMENTS with forty rows is a box inside a box. What a box
  // measures is its own content and never the content of the box it holds,
  // or an outer box whose rows all fit draws a thumb and a fade that say
  // there is more below where there is nothing, and a box that really does
  // overflow draws a thumb the length of somebody else's list.
  func testABoxInsideABoxDecoratesOnlyItsOwnContent() throws {
    // Forty rows measure far past the outer box's own 548 pt cap, while the
    // outer box holds nothing but the inner box's 236 pt.
    let inner = ScrollBox { plainRows(rows(40)) }
    let outer = ScrollBox(height: 548) { VStack(spacing: 0) { inner } }

    let drawn = try probeList(outer.rendering(), scheme: .dark)

    XCTAssertEqual(drawn.filter { $0.name == "scrollbox" }.count, 2, "two boxes drew")
    XCTAssertEqual(
      drawn.filter { $0.name == "scrollbox.thumb" }.count, 1,
      "only the box whose own content overflows carries a thumb")
    XCTAssertEqual(
      drawn.filter { $0.name == "scrollbox.fade" }.count, 1,
      "and only that box fades at its foot")
    XCTAssertEqual(
      drawn.first { $0.name == "scrollbox.thumb" }?.size.height,
      ScrollBox<EmptyView>.thumbLength(
        box: Grid.scrollBoxHeight, measured: 40 * Grid.rowHeight),
      "the thumb is as much of the inner box's track as that box is of its own list")
  }

  // `MORE('Show 3 more')` closed, `MORE('Show less',true)` open: one row
  // changes how much of a section is on the screen, and nothing else moves.
  func testShowMoreAndShowLessToggleTheList() throws {
    // DOCUMENTS shows five of twelve.
    let closed = ShowMore(total: 12, collapsed: 5)
    XCTAssertEqual(closed.visible, 5)
    XCTAssertEqual(closed.hidden, 7)
    XCTAssertEqual(closed.title, "Show 7 more")

    let open = closed.toggled
    XCTAssertTrue(open.isOpen)
    XCTAssertEqual(open.visible, 12)
    XCTAssertEqual(open.hidden, 0)
    XCTAssertEqual(open.title, "Show less")
    XCTAssertEqual(open.toggled, closed, "the word takes the list back to five")

    // TODAY shows eight, and a day of four actions carries no row at all.
    XCTAssertEqual(ShowMore(total: 11, collapsed: 8).title, "Show 3 more")
    XCTAssertNil(ShowMore(total: 4, collapsed: 8).title, "nothing is hidden")
    XCTAssertEqual(ShowMore(total: 4, collapsed: 8).visible, 4)
    XCTAssertNil(
      ShowMore(total: 4, collapsed: 8, isOpen: true).title,
      "an opened list of four has nothing to close")

    // The counts are the drawing's: eight rows and a chevron closed, forty
    // rows inside the box open.
    let forty = rows(40)
    let shut = try probes(
      ShowMoreList(forty, collapsed: 8) { title in
        RowView(.plain, glyph: .check, tone: .ok, title: title)
          .metricProbe("item-\(forty.firstIndex(of: title) ?? -1)")
      }
      .rendering(), scheme: .dark)
    XCTAssertNotNil(shut["item-7"], "the eighth row is on the screen")
    XCTAssertNil(shut["item-8"], "the ninth is not")
    XCTAssertNil(shut["scrollbox"], "a closed list needs no box")
    XCTAssertNotNil(shut["chevron"], "the word is a chevron row")

    let opened = try probes(
      ShowMoreList(forty, collapsed: 8, isOpen: true) { title in
        RowView(.plain, glyph: .check, tone: .ok, title: title)
          .metricProbe("item-\(forty.firstIndex(of: title) ?? -1)")
      }
      .rendering(), scheme: .dark)
    XCTAssertNotNil(opened["item-39"], "every row is in the list")
    XCTAssertEqual(opened["scrollbox"]?.height, Grid.scrollBoxHeight, "and the list scrolls")
    XCTAssertNotNil(opened["chevron"])

    // The row is a button: pressing it is the only thing that changes the
    // count, and the list itself holds no state of its own.
    var pressed = 0
    let list = ShowMoreList(forty, collapsed: 8, toggle: { pressed += 1 }) { title in
      RowView(.plain, glyph: .check, tone: .ok, title: title)
    }
    XCTAssertEqual(list.state, ShowMore(total: 40, collapsed: 8))
    list.toggle()
    XCTAssertEqual(pressed, 1)
  }
}
