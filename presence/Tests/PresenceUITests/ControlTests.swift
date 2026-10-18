// The four controls: the tooltip layer, the switch, the inline confirm and
// the one text button.
//
// Every number below is a literal read out of
// docs/design/2026-10-17-presence-screens-v3.html by hand: `[data-tip]::after`
// and its two variants, `.switch`, `.confirm` and `.btn`. The delays are the
// decisions of the specification, stated again here as literals.
//
// The last test is a scan rather than a reading, because a word a person can
// press is the panel's rarest thing: the blueprint has one text button, Open
// chat, and one pair of them in a confirm row, and the next person to add a
// third will not have read the specification.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class ControlTests: XCTestCase {
  // `[data-tip]::after{top:calc(100% + 6px);left:50%;transform:translateX(-50%)}`,
  // `.acts [data-tip]::after{left:auto;right:0}`,
  // `.row-title[data-tip]::after{left:0;top:calc(100% + 4px)}`, and the half
  // second a pointer rests before any of it is drawn.
  func testATooltipAppearsAfterHalfASecondAndFlipsInsideThePanel() throws {
    XCTAssertEqual(Timing.tooltipDelay, 0.5, "the pointer rests half a second")
    XCTAssertEqual(Tooltip.delay, Timing.tooltipDelay)
    XCTAssertEqual(Tooltip.radius, 4)
    XCTAssertEqual(Tooltip.paddingH, 7)
    XCTAssertEqual(Tooltip.paddingV, 3)
    XCTAssertEqual(TooltipAlign.allCases, [.centred, .trailing, .leading])
    XCTAssertEqual(TooltipAlign.centred.gap, 6)
    XCTAssertEqual(TooltipAlign.trailing.gap, 6)
    XCTAssertEqual(
      TooltipAlign.leading.gap, 4, "a title's tooltip sits 4 pt under it, not 6")

    // The rest, not the entry: a pointer that crossed a button is not asking
    // what it does.
    let entered = Date(timeIntervalSince1970: 1_000)
    XCTAssertFalse(Tooltip.shows(hoverSince: entered, now: entered, forced: false))
    XCTAssertFalse(
      Tooltip.shows(hoverSince: entered, now: entered.addingTimeInterval(0.49), forced: false))
    XCTAssertTrue(
      Tooltip.shows(hoverSince: entered, now: entered.addingTimeInterval(0.5), forced: false))
    XCTAssertFalse(
      Tooltip.shows(hoverSince: nil, now: entered.addingTimeInterval(9), forced: false),
      "nothing is hovered, so nothing is asked")
    XCTAssertTrue(
      Tooltip.shows(hoverSince: nil, now: entered, forced: true),
      "`show-tip`: a forced tooltip needs no pointer at all")

    let panel = CGSize(width: Grid.panelWidth, height: 240)
    let bubble = CGSize(width: 96, height: 20)

    // The plain case: centred under the anchor, 6 pt below it.
    let middle = CGRect(x: 140, y: 100, width: 24, height: 24)
    let under = Tooltip.place(anchor: middle, size: bubble, in: panel, align: .centred)
    XCTAssertEqual(under.origin.x, middle.midX - bubble.width / 2)
    XCTAssertEqual(under.origin.y, middle.maxY + 6)
    XCTAssertFalse(under.above)

    // `.acts [data-tip]::after{right:0}`: an action's tooltip ends on the
    // panel's own right inset, which is how four icons in 108 pt keep their
    // words inside the panel whichever of them the pointer is on.
    let action = CGRect(x: 280, y: 100, width: 24, height: 24)
    let trailing = Tooltip.place(anchor: action, size: bubble, in: panel, align: .trailing)
    XCTAssertEqual(trailing.origin.x + bubble.width, panel.width - Tooltip.margin)

    // And when a centred bubble would hang out of the panel, it slides in:
    // the margin is the panel's own padding, so no tooltip touches its edge.
    XCTAssertEqual(Tooltip.margin, Grid.unit)
    let squeezed = Tooltip.place(anchor: action, size: bubble, in: panel, align: .centred)
    XCTAssertEqual(
      squeezed.origin.x + bubble.width, panel.width - Tooltip.margin,
      "the bubble's right edge rests on the panel's margin")
    let leftEdge = CGRect(x: 8, y: 100, width: 24, height: 24)
    let slid = Tooltip.place(anchor: leftEdge, size: bubble, in: panel, align: .centred)
    XCTAssertEqual(slid.origin.x, Tooltip.margin, "and never left of it either")

    // The flip: a row at the foot of the panel carries its tooltip above,
    // because the panel is a window and there is nothing under it to draw on.
    let bottom = CGRect(x: 140, y: 206, width: 24, height: 24)
    let flipped = Tooltip.place(anchor: bottom, size: bubble, in: panel, align: .centred)
    XCTAssertTrue(flipped.above, "a tooltip at the foot of the panel flips over the anchor")
    XCTAssertEqual(flipped.origin.y, bottom.minY - 6 - bubble.height)
    XCTAssertGreaterThanOrEqual(flipped.origin.y, Tooltip.margin)

    // The bubble is ink in both appearances, and the two appearances are not
    // the same drawing: `.screen.light [data-tip]::after` is its own plate.
    for scheme in [ColorScheme.dark, .light] {
      let drawn = try inkedPixels(
        TooltipBubble("Open in Google Docs"), scheme: scheme, side: Grid.panelWidth)
      XCTAssertGreaterThan(drawn, 100, "the bubble drew nothing in \(scheme)")
    }
    XCTAssertNotEqual(
      try png(TooltipBubble("Open in Google Docs"), scheme: .dark),
      try png(TooltipBubble("Open in Google Docs"), scheme: .light))
  }

  // The third of four icons in a document row. Its words are drawn under
  // the icon the pointer is on and end on the panel's right inset, so the
  // bubble of the third icon is the bubble of the fourth one place up: a
  // plate that hung off each icon's own edge stepped left across the row
  // and read as four different things.
  func testAnActionTooltipSitsUnderItsIconAndEndsOnThePanelsInset() throws {
    // The document row's icons: 108 pt of column against the panel's right
    // inset, four 24 pt icons with 4 pt between them.
    let panel = CGSize(width: Grid.panelWidth, height: 420)
    let columnEnd = Grid.panelWidth - Grid.unit - Grid.rowInset
    let third = CGRect(
      x: columnEnd - 2 * Grid.actionSize - Grid.actionGap, y: 180,
      width: Grid.actionSize, height: Grid.actionSize)
    let bubble = CGSize(width: 132, height: 20)

    let placed = Tooltip.place(anchor: third, size: bubble, in: panel, align: .trailing)
    XCTAssertEqual(
      placed.origin.y, third.maxY + TooltipAlign.trailing.gap,
      "the bubble hangs 6 pt under the icon the pointer is on")
    XCTAssertFalse(placed.above, "there is room under it, so it is not flipped over the row")
    XCTAssertEqual(
      placed.origin.x + bubble.width, panel.width - Tooltip.margin,
      "and ends on the panel's own inset rather than on the icon's edge")
    XCTAssertGreaterThanOrEqual(placed.origin.x, Tooltip.margin)
    XCTAssertGreaterThanOrEqual(
      placed.origin.y, third.maxY, "the icon's own row is never covered")

    // The fourth icon's bubble is the same plate in the same place, which is
    // the whole of this rule: the words change and the frame does not.
    let fourth = CGRect(
      x: columnEnd - Grid.actionSize, y: 180, width: Grid.actionSize, height: Grid.actionSize)
    let last = Tooltip.place(anchor: fourth, size: bubble, in: panel, align: .trailing)
    XCTAssertEqual(last.origin, placed.origin)
  }

  // `.show-tip::after{opacity:1}`: the mock holds a tooltip open to show one,
  // and so does a still of the panel. Nothing in the app forces one except
  // the second after Copy link.
  func testATooltipCanBeForcedForARender() throws {
    let words = "Open the note in Obsidian"
    let plain = ActionButton(.md, tip: words)
      .tooltipLayer()
      .frame(width: Grid.panelWidth, height: 80)
    let forced = ActionButton(.md, tip: words, forcedTooltip: true)
      .tooltipLayer()
      .frame(width: Grid.panelWidth, height: 80)

    for scheme in [ColorScheme.dark, .light] {
      let quiet = try inkedPixels(plain, scheme: scheme, side: Grid.panelWidth)
      let shown = try inkedPixels(forced, scheme: scheme, side: Grid.panelWidth)
      XCTAssertGreaterThan(
        shown, quiet + 100, "the forced tooltip reached the canvas in \(scheme)")
    }

    // A forced tooltip on a pressed Copy link says Copied, which is the
    // button's own rule and not the layer's.
    XCTAssertTrue(ActionButton.showsTooltipAlways(.done))
    XCTAssertFalse(ActionButton.showsTooltipAlways(.live))
  }

  // `.switch{width:26px;height:16px;border-radius:8px}` with a 12 px knob
  // inset 2 px, drawn with shapes rather than handed to the platform: a
  // native toggle is 51 pt wide and blue, which is the complaint that
  // started this milestone.
  func testTheSwitchIsDrawnNotToggle() throws {
    XCTAssertEqual(SwitchView.width, 26)
    XCTAssertEqual(SwitchView.height, 16)
    XCTAssertEqual(SwitchView.radius, 8)
    XCTAssertEqual(SwitchView.knob, 12)
    XCTAssertEqual(SwitchView.inset, 2)
    XCTAssertEqual(SwitchView.knobTravel, 12, "`.switch.is-on::after{left:12px}`")
    XCTAssertEqual(SwitchView.knobOffset(isOn: false), 2)
    XCTAssertEqual(SwitchView.knobOffset(isOn: true), 12)

    // `.switch.is-on{background:var(--t1)}` and
    // `.switch.is-on::after{background:var(--s0)}`: the on switch is the
    // panel's brightest grey with a hole in it, and never a colour.
    for palette in [Palette.dark, Palette.light] {
      XCTAssertEqual(SwitchView.track(isOn: false, in: palette), palette.switchTrack)
      XCTAssertEqual(SwitchView.track(isOn: true, in: palette), palette.text)
      XCTAssertEqual(SwitchView.knobInk(isOn: false, in: palette), palette.switchKnob)
      XCTAssertEqual(SwitchView.knobInk(isOn: true, in: palette), palette.surface)
      XCTAssertNotEqual(SwitchView.track(isOn: true, in: palette), palette.ok)
    }

    for scheme in [ColorScheme.dark, .light] {
      let measured = try probes(
        SwitchView(isOn: true), scheme: scheme, width: SwitchView.width)
      XCTAssertEqual(measured["switch"], CGSize(width: 26, height: 16), "the switch in \(scheme)")
      XCTAssertNotEqual(
        try png(SwitchView(isOn: true), scheme: scheme),
        try png(SwitchView(isOn: false), scheme: scheme),
        "the knob moved and the track changed in \(scheme)")
    }

    // The scan: no file in this package hands a switch to the platform.
    // Spelled in pieces so this test is not its own failure.
    let native = "Tog" + "gle("
    for (name, text) in try componentSources() {
      XCTAssertFalse(text.contains(native), "\(name) draws a native switch")
    }
  }

  // `.confirm .btn{height:20px;padding:0 8px}` with a 2 px gap. The question
  // waits for its answer: it goes with Keep, with the confirm, or with the
  // panel closing, and never with the clock. Three seconds was the rule
  // until 2026-10-18, when a person reading the sentence under "Sign out?"
  // found the question gone and the icon back under the pointer.
  func testAConfirmRowWaitsForItsAnswer() throws {
    XCTAssertEqual(ConfirmRow.gap, 2)

    let asked = Date(timeIntervalSince1970: 1_000)
    XCTAssertTrue(ConfirmRow.isShowing(askedAt: asked, now: asked))
    XCTAssertTrue(ConfirmRow.isShowing(askedAt: asked, now: asked.addingTimeInterval(3)))
    XCTAssertTrue(ConfirmRow.isShowing(askedAt: asked, now: asked.addingTimeInterval(30)))
    XCTAssertTrue(ConfirmRow.isShowing(askedAt: asked, now: asked.addingTimeInterval(3600)))
    XCTAssertFalse(ConfirmRow.isShowing(askedAt: nil, now: asked), "no question, nothing shown")
    XCTAssertFalse(
      ConfirmRow.isShowing(askedAt: asked, now: asked.addingTimeInterval(-1)),
      "a question asked in the future is no question on the screen")
    XCTAssertFalse(ConfirmRow.isShowing(askedAt: nil, now: asked), "nobody asked anything")

    // The row is a two-line row with the pair where the actions would be,
    // and the pair's buttons are the short ones.
    let row = ConfirmRow(
      title: "Sign out?", sub: "2 live watches and 1 hold will stop.", danger: .signOut)
    let measured = try probes(row, scheme: .dark)
    XCTAssertEqual(
      measured["row.two"]?.height, 56,
      "the sentence beside the pair wraps to a second line rather than being cut, "
        + "which is the grid's 44 and one 12 pt line of caption under it")
    XCTAssertEqual(measured["confirm.danger"]?.height, 20)
    XCTAssertEqual(measured["confirm.keep"]?.height, 20)
    XCTAssertEqual(TextButton.height(.compact), 20)
    XCTAssertEqual(TextButton.height(.regular), 24, "`.btn{height:24px}`")
    XCTAssertEqual(TextButton.paddingH(.compact), 8)
    XCTAssertEqual(TextButton.paddingH(.regular), 10)

    // The dangerous half is the panel's red and the other is its plain ink,
    // so the two are told apart without reading them.
    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      XCTAssertEqual(TextButton.ink(.danger, in: palette), palette.bad)
      XCTAssertEqual(TextButton.ink(.plain, in: palette), palette.text)
      XCTAssertEqual(TextButton.ink(.primary, in: palette), palette.primaryInk)
      XCTAssertEqual(TextButton.fill(.primary, in: palette, scheme: scheme), palette.primaryFill)
      XCTAssertNil(
        TextButton.edge(.danger, in: palette), "`.confirm .btn{border-color:transparent}`")
      XCTAssertEqual(TextButton.edge(.plain, in: palette), palette.hairlineStrong)

      // Inside a confirm neither half carries one, so the pair reads as one
      // pair rather than as a button beside a word.
      XCTAssertNil(TextButton.edge(.plain, size: .compact, in: palette))
      XCTAssertNil(TextButton.edge(.danger, size: .compact, in: palette))
    }

    // `.btn{background:transparent}` on black, so a plain button on the hold
    // card's warm plate lets the plate through; `.screen.light .btn
    // {background:var(--s0)}` on white, where a transparent button would
    // vanish into the tint.
    XCTAssertNil(TextButton.fill(.plain, in: .dark, scheme: .dark))
    XCTAssertEqual(TextButton.fill(.plain, in: .light, scheme: .light), Palette.light.surface)

    // `.screen.light .btn-primary{box-shadow:inset 0 .5px 0 #ffffff26}`: the
    // one filled button carries a half-pixel highlight on white only.
    XCTAssertEqual(TextButton.primaryHighlight.hex, 0xFF_FF_FF)
    XCTAssertEqual(TextButton.primaryHighlight.alpha, Double(0x26) / 255)
  }

  // The blueprint's only pressable words: Open chat on the hold card, and
  // the pair of an inline confirm. Everything else a person can press is an
  // icon with a tooltip, which is the vocabulary rule of the design
  // principles. The words are an enum so the scan has something to count,
  // and Release is not among them: the panel never says it (the
  // specification's decision table, over the screens file's own label).
  func testTheOnlyTextButtonsAreOpenChatAndTheConfirmPair() throws {
    XCTAssertEqual(
      ButtonWord.allCases.map(\.rawValue), ["Open chat", "Dismiss", "Keep", "Sign out"])
    XCTAssertFalse(ButtonWord.allCases.map(\.rawValue).contains("Release"))

    // The scan: a word in a button comes from the enum, so no component
    // writes one of its own.
    let literal = "Text(\""
    let maker = "But" + "ton"
    var makers: [String] = []
    for (name, text) in try componentSources() {
      XCTAssertFalse(
        text.contains(literal), "\(name) writes a word of its own instead of a ButtonWord")
      XCTAssertFalse(
        text.contains("TextButton(\""),
        "\(name) hands TextButton a string instead of a ButtonWord")
      // The word on its own, so TextButton's own callers are not counted as
      // makers of a button.
      if makes(maker + "(", in: text) || makes(maker + " {", in: text) { makers.append(name) }
    }
    XCTAssertEqual(
      makers.sorted(),
      ["ActionButton.swift", "ButtonStyles.swift", "ChevronRow.swift", "SwitchView.swift"],
      "a fifth component makes a button of its own")
  }

  // MARK: the renders

  /// The sheet: the four controls in both appearances, so they are looked at
  /// once rather than trusted from a pixel count.
  ///
  /// Every word on it is a scene's, through `SheetPieces`. This page had
  /// four strings typed into it by hand, which is the one way a sentence the
  /// app cannot print reaches a screen Nail reads: the question is scene
  /// e10's own and the two tooltips are scene e6's row icons.
  ///
  /// A tooltip is drawn under the icon it belongs to, so the sheet draws it
  /// that way: a bubble with no control under it says nothing about where
  /// the panel puts one, and a bubble over its own icon hides the dead
  /// control it is explaining.
  func testTheControlSheetRenders() throws {
    let shots = try shotsDirectory()
    let question = try SheetPieces.signOut()
    let tips = try SheetPieces.tips()

    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      let sheet = VStack(alignment: .leading, spacing: Grid.unit) {
        HStack(spacing: Grid.unit) {
          SwitchView(isOn: false)
          SwitchView(isOn: true)
          TextButton(.openChat, kind: .primary)
          TextButton(.dismiss, kind: .plain)
        }
        ConfirmRow(
          title: question.title, sub: question.consequence, danger: question.danger)
        // One icon per band, each with its own layer: the panel draws one
        // bubble at a time, so two icons under one layer is one bubble.
        // The band is tall enough for the bubble under the icon, because
        // a layer with no room below flips the bubble over the icon it is
        // explaining. The icon is at the trailing edge, which is where a
        // row carries its actions and the only place a bubble that hangs
        // off its own right edge lands under it:
        // testTheSheetDrawsEachTooltipUnderItsOwnIcon.
        ActionButton(.gdocs, tip: tips.live, forcedTooltip: true)
          .metricProbe("sheet.icon")
          .padding(.trailing, Grid.unit)
          .frame(width: Grid.panelWidth, height: Self.tooltipBand, alignment: .topTrailing)
          .tooltipLayer()
        ActionButton(.md, tip: tips.dead.words, state: tips.dead.state, forcedTooltip: true)
          .metricProbe("sheet.icon")
          .padding(.trailing, Grid.unit)
          .frame(width: Grid.panelWidth, height: Self.tooltipBand, alignment: .topTrailing)
          .tooltipLayer()
      }
      .frame(width: Grid.panelWidth)
      .padding(Grid.unit)
      .background(palette.surface.color)

      let data = try png(sheet, scheme: scheme)
      let name = scheme == .dark ? "controls-dark.png" : "controls-light.png"
      try data.write(to: shots.appendingPathComponent(name))
      XCTAssertGreaterThan(data.count, 1_000, "\(name) is too small to be the sheet")
    }
  }

  /// How much room one icon and the bubble under it take on the sheet: the
  /// icon's own box, the gap the bubble keeps, and a line of words with its
  /// padding either side.
  static let tooltipBand =
    Grid.actionSize + TooltipAlign.trailing.gap + 3 * Grid.unit
    + Typography.caption.lineHeight

  /// A sheet of a tooltip has one thing to show: where the panel puts it.
  ///
  /// The page drew one bubble over two icons, which is both halves of the
  /// thing going wrong: the layer draws one bubble at a time, and a layer
  /// with no room below the anchor flips the bubble on top of it. So the
  /// dead control the words were explaining was the one thing hidden.
  func testTheSheetDrawsEachTooltipUnderItsOwnIcon() throws {
    let tips = try SheetPieces.tips()
    for (words, state) in [(tips.live, ActionState.live), (tips.dead.words, tips.dead.state)] {
      let band =
        ActionButton(.gdocs, tip: words, state: state, forcedTooltip: true)
        .metricProbe("sheet.icon")
        .padding(.trailing, Grid.unit)
        .frame(width: Grid.panelWidth, height: Self.tooltipBand, alignment: .topTrailing)
        .tooltipLayer()
      let drawn = try probeList(band, scheme: .dark)
      let icon = try XCTUnwrap(drawn.first { $0.name == "sheet.icon" }, "the icon drew nothing")
      let bubble = try XCTUnwrap(
        drawn.first { $0.name == "tooltip" }, "the words were not drawn at all")
      XCTAssertGreaterThanOrEqual(
        bubble.origin.y, icon.origin.y + icon.size.height,
        "the bubble covers the icon it belongs to")
      XCTAssertLessThanOrEqual(
        bubble.origin.x + bubble.size.width, Grid.panelWidth,
        "and hangs off the panel")
      // The bubble hangs off its icon's own right edge and stops at the
      // panel's margin, which is the rule that keeps the last of four
      // icons' words inside the panel. On the sheet that only reads as
      // pointing at the icon if the icon is where a row keeps it, so the
      // two have to overlap: a bubble 173 pt to the right of its icon
      // points at nothing.
      XCTAssertLessThanOrEqual(
        bubble.origin.x, icon.origin.x + icon.size.width,
        "the bubble starts right of the icon it explains")
      XCTAssertGreaterThanOrEqual(
        bubble.origin.x + bubble.size.width, icon.origin.x,
        "and ends left of it")
    }
  }

  // MARK: the scan

  /// Whether the text makes one, with the needle's first word standing on
  /// its own: `TextButton(` is a caller and not a maker.
  private func makes(_ needle: String, in text: String) -> Bool {
    var search = text.startIndex..<text.endIndex
    while let found = text.range(of: needle, range: search) {
      if found.lowerBound == text.startIndex { return true }
      let before = text[text.index(before: found.lowerBound)]
      if !(before.isLetter || before.isNumber || before == "_") { return true }
      search = found.upperBound..<text.endIndex
    }
    return false
  }

  /// Every component source, by file name. The path is this test file's own
  /// and never the working directory, which swift test does not promise.
  private func componentSources() throws -> [(String, String)] {
    let base =
      presencePackageRoot()
      .appendingPathComponent("Sources/PresenceUI/Components")
    let names = try FileManager.default.contentsOfDirectory(atPath: base.path)
      .filter { $0.hasSuffix(".swift") }
    XCTAssertGreaterThan(names.count, 5, "no component was read, so this scan guards nothing")
    return try names.map {
      ($0, try String(contentsOf: base.appendingPathComponent($0), encoding: .utf8))
    }
  }
}
