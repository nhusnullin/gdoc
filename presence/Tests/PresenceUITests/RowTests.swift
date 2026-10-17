// The rows, the sections, the header and the titles, measured off the
// drawing.
//
// Every number below is a literal read out of
// docs/design/2026-10-17-presence-screens-v3.html by hand, and every one of
// them is then read back out of the render through MetricProbes, because a
// constant a layout ignores passes any test that reads the constant.
//
// The titles are invented and of the lengths the hub really holds, 20, 40,
// 60 and 80 characters, so no real document's name is in the tree.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceCore
@testable import PresenceUI

@MainActor
final class RowTests: XCTestCase {
  private let twenty = "Pricing one-pager v2"
  private let forty = "Partner onboarding checklists for Munich"
  private let sixty = "Treasury reconciliation runbook for the quarterly close call"
  private let eighty =
    "Second-quarter liquidity and settlement reviews for the Amsterdam trading office"

  // The title column is 140 pt in a document row, which is the narrowest the
  // panel has, and the head never falls under eight characters in it: a
  // title whose last word is wide enough to squeeze it drops the tail and
  // goes in whole instead.
  func testNoHeadUnderEightCharactersAtPanelWidth() {
    XCTAssertEqual(
      MiddleTitle.narrowestColumn, 140,
      "320 less the panel's 8 twice, the row's 12 twice, the 16 pt glyph, two 8 pt gaps "
        + "and the 108 pt action column")

    // The ellipsis is part of what the column carries. Scene r9's second
    // document is the title that proved it: a tail the counted rule admits,
    // a head the room admits, and seven characters of head once the row has
    // drawn the three dots after them.
    let ellipsisEater = "B2B2C payment card concept: the frozen vocabulary"

    for title in [twenty, forty, sixty, eighty, ellipsisEater] {
      let shown = MiddleTitle.headCharacters(title)
      XCTAssertGreaterThanOrEqual(
        shown, Titles.headFloor, "\"\(title)\" shows only \(shown) characters of head")
    }

    // A tail the counted rule admits but the column cannot carry: thirteen
    // characters and a space, after a head that starts wide. The measured
    // floor drops it, so the row draws the whole title with an ellipsis
    // rather than eight characters and a word.
    let wideTail = "Treasury reconciliation runbook for the announcements"
    XCTAssertEqual(Titles.split(wideTail).tail, " announcements", "the counted rule takes it")
    XCTAssertNil(
      MiddleTitle.fitted(wideTail).tail, "the measured floor gives it back")
    XCTAssertEqual(MiddleTitle.fitted(wideTail).head, wideTail)
    XCTAssertGreaterThanOrEqual(MiddleTitle.headCharacters(wideTail), Titles.headFloor)

    // The floor is measured and not counted: the same word count in narrow
    // letters keeps its tail.
    let narrowTail = "Illustrations of titles that are till it"
    XCTAssertEqual(MiddleTitle.fitted(narrowTail).tail, " it")
  }

  // `.row{min-height:32px}`, `.row.two{min-height:44px}`,
  // `.row.compact{min-height:28px}`, and the document row is a two-line row.
  func testRowHeightsAreTheGrid() throws {
    XCTAssertEqual(RowShape.allCases, [.plain, .two, .compact, .doc, .more])
    XCTAssertEqual(RowShape.plain.minHeight, 32)
    XCTAssertEqual(RowShape.two.minHeight, 44)
    XCTAssertEqual(RowShape.compact.minHeight, 28)
    XCTAssertEqual(RowShape.doc.minHeight, 44)
    XCTAssertEqual(RowShape.more.minHeight, 28)
    XCTAssertEqual(RowShape.two.verticalPadding, 6)
    XCTAssertEqual(RowShape.plain.verticalPadding, 0)

    let drawn: [RowShape: CGFloat] = [
      .plain: 32, .two: 44, .compact: 28, .doc: 44, .more: 28,
    ]
    for (shape, height) in drawn {
      let row = RowView(
        shape, glyph: .check, tone: .ok, title: twenty,
        value: shape == .doc ? nil : "09:05",
        sub: shape == .two || shape == .doc ? "Proposed · verified" : nil)
      let measured = try probes(row, scheme: .dark)
      XCTAssertEqual(
        measured["row.\(shape.rawValue)"]?.height, height,
        "\(shape) drew \(String(describing: measured["row.\(shape.rawValue)"]?.height))")
      XCTAssertEqual(
        measured["row.\(shape.rawValue)"]?.width, Grid.panelWidth, "\(shape) fills the panel")
    }
  }

  // A sub-line that does not fit its column wraps to a second line rather
  // than being cut, which is the rule a document row already followed and
  // every two-line row now follows. The hold's own sub-line is the one that
  // proved it: what is cut off the end of "reply held by rule Dictated ·
  // waiting in Desktop chat" is where the text is waiting, so the row said
  // a write was held and not where to go about it.
  func testASubLineWrapsToASecondLineRatherThanBeingCut() throws {
    XCTAssertEqual(RowShape.two.subLines, 2, "the same two lines a document row gets")
    XCTAssertEqual(RowShape.doc.subLines, 2)
    XCTAssertEqual(RowShape.plain.subLines, 1, "a one-line row has no second line")
    XCTAssertEqual(RowShape.compact.subLines, 1)
    XCTAssertEqual(RowShape.more.subLines, 1)

    // The hold's own sub-line, as scene e3 draws it.
    let held = "Proposal held by rule Unread target · waiting in Desktop chat"
    let row = RowView(
      .two, glyph: .pause, tone: .hold, title: "Hold · Group Channel Policy",
      value: "16:34", sub: held)
    let measured = try probes(row, scheme: .dark)

    // The column the sentence is drawn in, read off the drawing: the panel's
    // 320 less the row's 12 either side, the 16 pt glyph and the 8 pt gap
    // after it.
    let column = try XCTUnwrap(measured["row.sub"]?.width)
    XCTAssertEqual(column, 272)
    XCTAssertEqual(
      lines(held, role: Typography.caption, width: column), 2,
      "the sentence fits one line at this width, so this row proves nothing")

    // A sub-line that fits keeps the row at the grid's 44, so nothing grew
    // that did not have to.
    let short = try probes(
      RowView(.two, glyph: .dot, tone: .live, title: "Live · Chapter 5", sub: "since 13:10"),
      scheme: .dark)
    XCTAssertEqual(short["row.two"]?.height, Grid.twoLineRowHeight)

    // The wrapped one drew two lines of the same caption, and the row grew
    // past the grid's 44 to hold them: the 6 pt either side, the title's 16
    // and the two lines.
    let oneLine = try XCTUnwrap(short["row.sub"]?.height)
    XCTAssertEqual(measured["row.sub"]?.height, 2 * oneLine, "the sentence drew two lines")
    let height = try XCTUnwrap(measured["row.two"]?.height)
    XCTAssertGreaterThan(height, Grid.twoLineRowHeight, "the row grew rather than clipping")
    XCTAssertEqual(
      height,
      2 * RowShape.two.verticalPadding + Typography.rowTitle.lineHeight + 2 * oneLine)
    XCTAssertEqual(height, 56)
  }

  // `.row.docrow{grid-template-columns:16px minmax(0,1fr) 108px}`: four
  // icons in 108 pt, so every document row in a list breaks its title at the
  // same place.
  func testTheDocRowActionColumnIs108() throws {
    XCTAssertEqual(Grid.actionColumn, 108)
    XCTAssertEqual(RowShape.doc.trailingColumn, 108)
    XCTAssertNil(RowShape.plain.trailingColumn, "every other row is as wide as it asks for")

    // Four 24 pt icons and three 4 pt gaps are 108 pt, which is the number.
    XCTAssertEqual(4 * Grid.actionSize + 3 * Grid.actionGap, 108)

    // The row is scene e6's own, so the words under the measurement are
    // words the panel draws: a count beside a verb is one of the things
    // gdoc never writes down.
    let drawn = try SheetPieces.documentRows()[0]
    let row = RowView(
      .doc, glyph: drawn.glyph, tone: drawn.tone, title: drawn.title, sub: drawn.sub,
      trailing: {
        HStack(spacing: Grid.actionGap) {
          ActionButton(.gdocs, tip: Tips.openDoc)
          ActionButton(.md, tip: Tips.openNote)
          ActionButton(.branch, tip: Tips.openChat)
          ActionButton(.link, tip: Tips.copyLink)
        }
      })
    let measured = try probes(row, scheme: .dark)
    XCTAssertEqual(measured["row.trailing"]?.width, 108)

    // `.row.is-busy .acts{display:none}`: a busy row carries no action at
    // all, so the column goes with them.
    let busy = RowView(
      .doc, glyph: .plain, tone: .quiet, title: drawn.title, sub: drawn.sub, busy: true,
      trailing: { ActionButton(.gdocs, tip: Tips.openDoc) })
    let busyMeasured = try probes(busy, scheme: .dark)
    XCTAssertNil(busyMeasured["row.trailing"], "a busy row draws no action column")
  }

  // A chevron sits in the middle of the row its words are in.
  //
  // `.row-chevron` is one 28 pt row with the words on the left and a 12 pt
  // chevron on the right, and a chevron a quarter of a row below the line it
  // belongs to reads as a mark on an empty line.
  func testTheChevronIsLevelWithItsWords() throws {
    for scheme in [ColorScheme.dark, .light] {
      let row = ChevronRow("Show 3 more")
      let words = try inkBand(row, scheme: scheme, from: 0, to: 160)
      let chevron = try inkBand(row, scheme: scheme, from: 280, to: Grid.panelWidth)
      let middle = { (band: ClosedRange<CGFloat>) in (band.lowerBound + band.upperBound) / 2 }
      XCTAssertEqual(
        middle(chevron), middle(words), accuracy: 1,
        "the chevron is \(middle(chevron) - middle(words)) pt off its words in \(scheme)")
    }
  }

  // One right edge for every value in the panel.
  //
  // `.row.compact` puts its value at the content edge, the same edge as a
  // log row's time; only a row with something on its right pulls the value
  // in to make room for it. HEALTH's rows with no icon were handing the
  // layout an empty action column, and 8 pt of nothing moved every version,
  // mark and keystroke left of every time in TODAY.
  func testAValueEndsWhereEveryOtherValueEnds() throws {
    let value = "v2.14.0"
    for scheme in [ColorScheme.dark, .light] {
      let log = try inkEnds(
        RowView(.plain, glyph: .check, tone: .ok, title: "Reply", value: value), scheme: scheme)
      let health = try inkEnds(
        HealthRows(rows: [HealthRow(.gdoc, value: value, glyph: .check, tone: .ok)]),
        scheme: scheme)
      XCTAssertEqual(health, log, "the HEALTH value is not on the log's edge in \(scheme)")

      // A row that does carry an icon keeps its value clear of it, which is
      // the Account row and the one in flight.
      let withIcon = try probes(
        HealthRows(
          rows: [
            HealthRow(
              .account, value: Scenes.account, glyph: .check, tone: .ok,
              action: IconAction(.signout, tip: SignedInBanner.signOutTip))
          ]), scheme: scheme)
      XCTAssertNotNil(
        withIcon["row.trailing"], "a row with an icon draws the column in \(scheme)")

      let bare = try probes(
        HealthRows(rows: [HealthRow(.gdoc, value: value, glyph: .check, tone: .ok)]),
        scheme: scheme)
      XCTAssertNil(bare["row.trailing"], "a row with nothing on its right draws no column")
    }
  }

  // `.section-label{height:28px;text-transform:uppercase}` with its count
  // on the right in ordinary letters.
  func testASectionLabelIs28AndUppercaseWithTheCount() throws {
    XCTAssertEqual(Grid.sectionLabelHeight, 28)
    XCTAssertEqual(SectionLabel.text("Documents"), "DOCUMENTS")
    XCTAssertEqual(Typography.sectionLabel.size, 11)
    XCTAssertEqual(Typography.sectionLabel.tracking, 0.06)
    XCTAssertTrue(Typography.sectionLabel.uppercase)

    // `--t3` on black and `--t2` on white, because a tertiary grey on white
    // is too faint to read at 11 pt.
    XCTAssertEqual(SectionLabel.ink(in: .dark), Palette.dark.textTertiary)
    XCTAssertEqual(SectionLabel.ink(in: .light), Palette.light.textSecondary)

    for scheme in [ColorScheme.dark, .light] {
      let measured = try probes(
        SectionLabel("Documents", count: "14 days · 11"),
        scheme: scheme)
      XCTAssertEqual(measured["section-label"]?.height, 28, "the label in \(scheme)")
      XCTAssertEqual(measured["section-label"]?.width, Grid.panelWidth)
    }

    // The count is ink on the screen and not a field nobody drew: the label
    // with one carries more than the label without.
    for scheme in [ColorScheme.dark, .light] {
      let bare = try inkWeight(
        SectionLabel("Documents").frame(width: Grid.panelWidth), scheme: scheme,
        side: Grid.panelWidth)
      let counted = try inkWeight(
        SectionLabel("Documents", count: "14 days · 11").frame(width: Grid.panelWidth),
        scheme: scheme, side: Grid.panelWidth)
      XCTAssertGreaterThan(counted, bare, "the count reached the screen in \(scheme)")
    }

    // `.section:first-of-type{margin-top:0;padding-top:0;border-top:0}`: the
    // first section sits against the header, and every other one carries
    // 8 pt, a hairline and 4 pt.
    let first = try probes(
      SectionView("Now", isFirst: true) { RowView(title: twenty) }, scheme: .dark)
    let later = try probes(
      SectionView("Today") { RowView(title: twenty) }, scheme: .dark)
    let firstHeight = try XCTUnwrap(first["section"]?.height)
    let laterHeight = try XCTUnwrap(later["section"]?.height)
    XCTAssertEqual(laterHeight - firstHeight, Grid.unit + 4, "8 pt of air and 4 pt of padding")
  }

  // `.panel-header{height:36px}` with the mark, the name and the account.
  func testTheHeaderIs36WithTheMarkTheNameAndTheAccount() throws {
    XCTAssertEqual(Grid.headerHeight, 36)

    let measured = try probes(
      PanelHeader(mark: "⁘", account: "nail@example.com"), scheme: .dark)
    XCTAssertEqual(measured["header"]?.height, 36 + 4, "the header and its 4 pt margin under it")
    XCTAssertNotNil(measured["header.mark"], "the mark in effect")
    XCTAssertNotNil(measured["header.name"], "the word gdoc")
    XCTAssertNotNil(measured["header.account"], "the address, pushed right")

    // The account is the quietest grey in the panel, and the name is the
    // brightest: whose account it is decides nothing, so it is never loud.
    XCTAssertEqual(Typography.caption.size, 11)
    XCTAssertEqual(Typography.mark.size, 16)
    XCTAssertEqual(Typography.titleEmphasis.weight, .medium)

    // The mark is a setting and may be empty, and the empty mark draws
    // nothing rather than a gap.
    let noMark = try probes(PanelHeader(mark: "", account: "nail@example.com"), scheme: .dark)
    XCTAssertNil(noMark["header.mark"])
    XCTAssertEqual(noMark["header"]?.height, 40, "an empty mark keeps the header's height")

    // No token, no address: the header is the mark and the name.
    let signedOut = try probes(PanelHeader(mark: "⁘"), scheme: .dark)
    XCTAssertNil(signedOut["header.account"])
  }

  // `.row-title.gone{text-decoration:line-through;color:var(--t2)}`: a
  // document in the trash keeps its row, and its title is struck through.
  func testAGoneTitleIsStruckThrough() throws {
    for palette in [Palette.dark, Palette.light] {
      XCTAssertEqual(MiddleTitle.ink(gone: true, in: palette), palette.textSecondary)
      XCTAssertEqual(MiddleTitle.ink(gone: false, in: palette), palette.text)
      XCTAssertNotEqual(palette.textSecondary, palette.text, "a gone title is quieter too")
    }

    // The line reaches the canvas: the same title struck through is not the
    // same drawing, and carries more ink than the plain one.
    for scheme in [ColorScheme.dark, .light] {
      let plain = MiddleTitle(twenty).frame(width: MiddleTitle.narrowestColumn)
      let gone = MiddleTitle(twenty, gone: true).frame(width: MiddleTitle.narrowestColumn)
      XCTAssertNotEqual(
        try png(plain, scheme: scheme), try png(gone, scheme: scheme),
        "the strike-through drew nothing in \(scheme)")
    }
  }

  // MARK: the renders

  /// The sheet: the five shapes, a section, a header and a quiet line, in
  /// both appearances, so the layout is looked at once.
  ///
  /// Every word on it is a scene's, through `SheetPieces`: the sheet is the
  /// one page with no record behind it, and a string written by hand here
  /// is a string Nail reads as the app's own.
  func testTheRowSheetRenders() throws {
    let shots = try shotsDirectory()
    let labels = try SheetPieces.sectionLabels()
    let live = try SheetPieces.live()
    let log = try SheetPieces.logRow()
    let documents = try SheetPieces.documents()
    let rows = try SheetPieces.documentRows()
    let mark = try XCTUnwrap(
      SheetPieces.health().first { $0.kind == .mark }, "no scene drew the mark row")
    let quiet = try SheetPieces.quiet()

    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      // The five row shapes, and the two labels as components of their own.
      //
      // A label over rows is a section, and a section's header makes a claim
      // about what is under it: NOW over a log row and a HEALTH row, or
      // "14 days · 12" over two rows and a Show more, is this page saying
      // something no panel says. The pieces here are pieces, in the order
      // `RowShape` names them, and the labels stand at the foot with nothing
      // under them: testASheetPutsNoLabelOverRowsFromAnotherScene.
      let sheet = VStack(alignment: .leading, spacing: 0) {
        PanelHeader(mark: mark.value, account: Scenes.account)
        RowView(.two, glyph: .dot, tone: .live, title: live.title, sub: live.sub)
        RowView(.plain, glyph: .check, tone: .ok, title: log.title, value: log.value)
        RowView(.compact, glyph: .blank, title: mark.kind.label, value: mark.value)
        ForEach(rows) { row in
          RowView(
            .doc, glyph: row.glyph, tone: row.tone, title: row.title, sub: row.sub,
            gone: row.gone,
            trailing: {
              HStack(spacing: Grid.actionGap) {
                ForEach(row.actions) { action in
                  ActionButton(action.icon, tip: action.tip, state: action.state)
                }
              }
            })
        }
        ChevronRow(documents.more, isOpen: false)
        QuietLine(quiet)
        // The two labels, each on its own and with air around it: stacked,
        // the first reads as a header over the second.
        Spacer(minLength: 2 * Grid.unit)
        SectionLabel(labels.now)
        Spacer(minLength: 2 * Grid.unit)
        SectionLabel(labels.documents, count: documents.count)
      }
      .frame(width: Grid.panelWidth)
      .padding(Grid.unit)
      .background(palette.surface.color)

      let data = try png(sheet, scheme: scheme)
      let name = scheme == .dark ? "rows-dark.png" : "rows-light.png"
      try data.write(to: shots.appendingPathComponent(name))
      XCTAssertGreaterThan(data.count, 1_000, "\(name) is too small to be the sheet")
    }
  }
}
