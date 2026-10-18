// The action icons, against the SVG they were ported from.
//
// Twelve of the thirteen are the `IC` object of
// docs/design/2026-10-17-presence-screens-v3.html, drawn as Path rather than
// as SF Symbols, because four of them sit side by side in one row and a
// system symbol would be whatever the platform ships this year. The numbers
// below are the `.act` and `.act svg` rules, read by hand and written as
// literals.
//
// The thirteenth is the pin, which that file has no shape for: the pins are
// the second morning's decision (docs/v2/DECISIONS.md, 2026-10-18) and the
// shape is written here in the same grammar.
//
// The sheet of all thirteen goes out as a PNG under PRESENCE_SHOTS, so the
// shapes are looked at once rather than trusted from a pixel count.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class IconTests: XCTestCase {
  // Every icon the blueprint names, each parsed out of its own SVG, each one
  // inside its box and each one putting ink on the canvas.
  func testTheIconSheetRenders() throws {
    XCTAssertEqual(
      Icon.allCases,
      [.gdocs, .md, .branch, .link, .done, .stop, .down, .signin, .signout, .x, .xmark, .pin],
      "the twelve the panel draws, and no spare from the IC object")

    // `.act svg{width:16px;height:16px;stroke-width:1.25;stroke-linecap:round;
    // stroke-linejoin:round}`. Every icon left is one of those: the 12 pt
    // one was the chevron of a Show more row.
    XCTAssertEqual(IconShape.stroke, 1.25)
    XCTAssertEqual(IconShape.style.lineWidth, 1.25)
    XCTAssertEqual(IconShape.style.lineCap, .round)
    XCTAssertEqual(IconShape.style.lineJoin, .round)
    for icon in Icon.allCases {
      XCTAssertEqual(icon.side, 16, "\(icon) is a 16 px icon")
    }

    for icon in Icon.allCases {
      let path = IconShape(icon).path(in: CGRect(x: 0, y: 0, width: icon.side, height: icon.side))
      XCTAssertFalse(path.isEmpty, "\(icon) parsed to nothing, so its SVG was not understood")
      let box = path.boundingRect
      XCTAssertGreaterThanOrEqual(box.minX, -0.01, "\(icon) draws left of its box")
      XCTAssertGreaterThanOrEqual(box.minY, -0.01, "\(icon) draws above its box")
      XCTAssertLessThanOrEqual(box.maxX, icon.side + 0.01, "\(icon) draws past its box")
      XCTAssertLessThanOrEqual(box.maxY, icon.side + 0.01, "\(icon) draws under its box")
      XCTAssertGreaterThan(box.width, 2, "\(icon) is too narrow to be the drawing")
      XCTAssertGreaterThan(box.height, 2, "\(icon) is too short to be the drawing")

      for scheme in [ColorScheme.dark, .light] {
        let drawn = try inkedPixels(
          IconView(icon, ink: Palette.of(scheme).actionIcon), scheme: scheme,
          side: Grid.actionSize)
        XCTAssertGreaterThan(drawn, 8, "\(icon) drew nothing in \(scheme)")
      }
    }

    try writeSheet()
  }

  // `.act.off{opacity:.38}`, and the off icon carries its own words: the
  // tooltip is why it cannot be pressed, never the action it is not doing.
  /// A path this does not understand draws nothing, and the one input that
  /// could draw nothing for ever is a number after a close: the close
  /// consumes no number, so repeating it would never move the scanner on.
  /// The pin's head is not a circle, and the pin is no action.
  ///
  /// It sits in a document row's leading column, which is a column of
  /// circles: the hollow ring is a status glyph of its own, so a ring on a
  /// stem beside one read as a magnifying glass standing next to a state.
  /// And Pin is a word in the row's menu, so the pin in an action strip said
  /// the panel had a pin to press, and a dimmed one that it had a dead one.
  func testThePinIsNoActionAndNoCircle() throws {
    for part in Icon.pin.parts {
      guard case .path = part else {
        return XCTFail("the pin's head is a circle, and its column is full of circles")
      }
    }
    let ink = IconShape(.pin).path(in: CGRect(x: 0, y: 0, width: 16, height: 16)).boundingRect
    XCTAssertGreaterThan(ink.height, ink.width, "a tack is taller than its head is wide")

    XCTAssertFalse(Icon.actions.contains(.pin), "the pin is not something to press")
    XCTAssertEqual(
      Icon.actions.count, Icon.allCases.count - 1, "and it is the only one that is not")
    XCTAssertEqual(
      Set(Icon.allCases).subtracting(Icon.actions), [.pin])
  }

  func testANumberAfterACloseDrawsNothing() throws {
    XCTAssertNil(SVGPath.parse("M0 0L4 4Z 2 2"), "a number after Z is a path, not a hang")
    XCTAssertNil(SVGPath.parse("M0 0L4 4z 2 2"))
    XCTAssertNotNil(SVGPath.parse("M0 0L4 4Z"), "and the same path without it still draws")
    XCTAssertNotNil(
      SVGPath.parse("M12.5 2.5 8 7"), "a second pair after a moveto is still a lineto")
    XCTAssertNil(SVGPath.parse("2 2"), "a number before any command is nothing to repeat")
  }

  func testAnOffIconIsDimmedWithItsOwnTooltip() throws {
    XCTAssertEqual(ActionButton.size, 24)
    XCTAssertEqual(ActionButton.radius, 6)
    XCTAssertEqual(ActionButton.offOpacity, 0.38)

    XCTAssertEqual(ActionButton.opacity(for: .live), 1)
    XCTAssertEqual(ActionButton.opacity(for: .off("No Google Doc yet")), 0.38)
    XCTAssertEqual(ActionButton.opacity(for: .done), 1)

    XCTAssertEqual(
      ActionButton.tooltip("Open in Google Docs", state: .live), "Open in Google Docs")
    XCTAssertEqual(
      ActionButton.tooltip("Open in Google Docs", state: .off("No Google Doc yet")),
      "No Google Doc yet", "an off icon says why, and not what it would have done")
    XCTAssertFalse(ActionButton.isPressable(.off("No Google Doc yet")))
    XCTAssertTrue(ActionButton.isPressable(.live))

    // The opacity reaches the drawing: the same icon, dimmed, carries less
    // ink than the live one.
    for scheme in [ColorScheme.dark, .light] {
      let live = try inkWeight(
        ActionButton(.gdocs, tip: "Open in Google Docs"), scheme: scheme, side: ActionButton.size)
      let off = try inkWeight(
        ActionButton(.gdocs, tip: "Open in Google Docs", state: .off("No Google Doc yet")),
        scheme: scheme, side: ActionButton.size)
      XCTAssertGreaterThan(live, 0)
      XCTAssertLessThan(off, live * 0.5, "an off icon is dimmed in \(scheme)")
    }
  }

  // `.act.is-done{color:var(--ok)}` with `show-tip`: Copy link swaps to the
  // check, goes green and holds its tooltip open, for one second.
  func testADoneIconIsGreenForOneSecondThenReturns() throws {
    XCTAssertEqual(Timing.doneFlash, 1, "the copied icon is green for one second")

    let copied = Date(timeIntervalSince1970: 1_000)
    XCTAssertTrue(ActionButton.isDone(copiedAt: copied, now: copied))
    XCTAssertTrue(ActionButton.isDone(copiedAt: copied, now: copied.addingTimeInterval(0.99)))
    XCTAssertFalse(ActionButton.isDone(copiedAt: copied, now: copied.addingTimeInterval(1)))
    XCTAssertFalse(ActionButton.isDone(copiedAt: copied, now: copied.addingTimeInterval(2)))
    XCTAssertFalse(
      ActionButton.isDone(copiedAt: nil, now: copied), "nothing was copied, so nothing is green")

    XCTAssertEqual(
      ActionButton.shown(icon: .link, state: .done), .done,
      "the link icon becomes the check while it is green")
    XCTAssertEqual(ActionButton.shown(icon: .link, state: .live), .link)
    XCTAssertEqual(
      ActionButton.shown(icon: .link, state: .off("No link yet")), .link,
      "an off icon keeps its own shape")

    for palette in [Palette.dark, Palette.light] {
      XCTAssertEqual(ActionButton.ink(state: .done, in: palette), palette.ok)
      XCTAssertEqual(ActionButton.ink(state: .live, in: palette), palette.actionIcon)
      XCTAssertEqual(ActionButton.ink(state: .off("No link yet"), in: palette), palette.actionIcon)
      XCTAssertNotEqual(
        palette.actionIcon, palette.ok, "the icon is only green while it is saying Copied")
    }

    XCTAssertTrue(
      ActionButton.showsTooltipAlways(.done), "`show-tip`: Copied is read without a hover")
    XCTAssertFalse(ActionButton.showsTooltipAlways(.live))
    XCTAssertFalse(ActionButton.showsTooltipAlways(.off("No link yet")))

    // Green really is on the canvas: the done button draws in --ok, which no
    // other state does.
    let done = try png(
      ActionButton(.link, tip: "Copied", state: .done).frame(
        width: ActionButton.size, height: ActionButton.size), scheme: .dark)
    let live = try png(
      ActionButton(.link, tip: "Copy link").frame(
        width: ActionButton.size, height: ActionButton.size), scheme: .dark)
    XCTAssertNotEqual(done, live, "the copied icon is not the link icon in the panel's grey")
  }

  // MARK: the renders

  /// The sheet: the thirteen in a row, live, off and done, in both
  /// appearances, so the shapes are looked at once.
  private func writeSheet() throws {
    let shots = try shotsDirectory()
    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      let sheet = VStack(alignment: .leading, spacing: 8) {
        HStack(spacing: 4) {
          ForEach(Icon.actions, id: \.self) { icon in
            ActionButton(icon, tip: icon.rawValue)
          }
        }
        HStack(spacing: 4) {
          ForEach(Icon.actions, id: \.self) { icon in
            ActionButton(icon, tip: icon.rawValue, state: .off("not here"))
          }
        }
        HStack(spacing: 4) {
          ActionButton(.link, tip: "Copied", state: .done)
        }
        // The pin is no action, so it is drawn where the panel draws it:
        // a document row's leading column, at its own size, beside the
        // state glyph it stands in front of and beside the empty slot the
        // rows under a pinned one keep.
        HStack(spacing: Grid.columnGap) {
          DocumentGlyph(.check, tone: .ok, pin: .pinned)
          DocumentGlyph(.check, tone: .ok, pin: .slot)
          DocumentGlyph(.check, tone: .ok)
        }
      }
      .padding(16)
      .background(palette.surface.color)

      let data = try png(sheet, scheme: scheme)
      let name = scheme == .dark ? "icons-dark.png" : "icons-light.png"
      try data.write(to: shots.appendingPathComponent(name))
      XCTAssertGreaterThan(data.count, 1_000, "\(name) is too small to be the sheet")
    }
  }
}
