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

    for title in [twenty, forty, sixty, eighty] {
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

  // `.row.docrow{grid-template-columns:16px minmax(0,1fr) 108px}`: four
  // icons in 108 pt, so every document row in a list breaks its title at the
  // same place.
  func testTheDocRowActionColumnIs108() throws {
    XCTAssertEqual(Grid.actionColumn, 108)
    XCTAssertEqual(RowShape.doc.trailingColumn, 108)
    XCTAssertNil(RowShape.plain.trailingColumn, "every other row is as wide as it asks for")

    // Four 24 pt icons and three 4 pt gaps are 108 pt, which is the number.
    XCTAssertEqual(4 * Grid.actionSize + 3 * Grid.actionGap, 108)

    let row = RowView(
      .doc, glyph: .check, tone: .ok, title: sixty, sub: "Proposed 2 · verified",
      trailing: {
        HStack(spacing: Grid.actionGap) {
          ActionButton(.gdocs, tip: "Open in Google Docs")
          ActionButton(.md, tip: "Open md in Obsidian")
          ActionButton(.branch, tip: "Continue with Claude")
          ActionButton(.link, tip: "Copy link")
        }
      })
    let measured = try probes(row, scheme: .dark)
    XCTAssertEqual(measured["row.trailing"]?.width, 108)

    // `.row.is-busy .acts{display:none}`: a busy row carries no action at
    // all, so the column goes with them.
    let busy = RowView(
      .doc, glyph: .plain, tone: .quiet, title: sixty, sub: "Publishing", busy: true,
      trailing: { ActionButton(.gdocs, tip: "Open in Google Docs") })
    let busyMeasured = try probes(busy, scheme: .dark)
    XCTAssertNil(busyMeasured["row.trailing"], "a busy row draws no action column")
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
        SectionLabel("Documents", count: "14 days · 3 today"),
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
        SectionLabel("Documents", count: "14 days · 3 today").frame(width: Grid.panelWidth),
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
  func testTheRowSheetRenders() throws {
    let shots = try shotsDirectory()
    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      let sheet = VStack(alignment: .leading, spacing: 0) {
        PanelHeader(mark: "⁘", account: "nail@example.com")
        SectionView("Now", isFirst: true) {
          RowView(.two, glyph: .dot, tone: .live, title: forty, sub: "Watching · from Claude Code")
          RowView(.plain, glyph: .check, tone: .ok, title: twenty, value: "09:05")
          RowView(.compact, glyph: .blank, title: "Mark", value: "⁘")
        }
        SectionView("Documents", count: "14 days · 3 today") {
          RowView(
            .doc, glyph: .check, tone: .ok, title: eighty, sub: "Proposed 2 · verified",
            trailing: {
              HStack(spacing: Grid.actionGap) {
                ActionButton(.gdocs, tip: "Open in Google Docs")
                ActionButton(.md, tip: "Open md in Obsidian")
                ActionButton(.branch, tip: "Continue with Claude")
                ActionButton(.link, tip: "Copy link")
              }
            })
          RowView(.doc, glyph: .empty, tone: .dim, title: sixty, sub: "In the trash", gone: true)
          ChevronRow("Show 6 more", isOpen: false)
          QuietLine("Nothing since 09:05.")
        }
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
