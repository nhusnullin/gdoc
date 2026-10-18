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

  // The title column is 112 pt in a document row, which is the narrowest the
  // panel has, and the head never falls under eight characters in it: a
  // title whose last word is wide enough to squeeze it drops the tail and
  // goes in whole instead. It was 140 pt until the pin became the row's
  // fifth icon and took 28 pt of the row for it.
  func testNoHeadUnderEightCharactersAtPanelWidth() {
    XCTAssertEqual(
      MiddleTitle.narrowestColumn, 112,
      "320 less the panel's 8 twice, the row's 12 twice, the 16 pt glyph, two 8 pt gaps "
        + "and the 136 pt action column")

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
    XCTAssertEqual(RowShape.allCases, [.plain, .two, .compact, .doc])
    XCTAssertEqual(RowShape.plain.minHeight, 32)
    XCTAssertEqual(RowShape.two.minHeight, 44)
    XCTAssertEqual(RowShape.compact.minHeight, 28)
    XCTAssertEqual(RowShape.doc.minHeight, 44)
    XCTAssertEqual(RowShape.two.verticalPadding, 6)
    XCTAssertEqual(RowShape.plain.verticalPadding, 0)

    let drawn: [RowShape: CGFloat] = [.plain: 32, .two: 44, .compact: 28, .doc: 44]
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

  // `.row.docrow{grid-template-columns:16px minmax(0,1fr) 136px}`: five
  // icons in 136 pt since the pin joined them, so every document row in a
  // list breaks its title at the same place.
  func testTheDocRowActionColumnIs136() throws {
    XCTAssertEqual(Grid.actionColumn, 136)
    XCTAssertEqual(RowShape.doc.trailingColumn, 136)
    XCTAssertNil(RowShape.plain.trailingColumn, "every other row is as wide as it asks for")

    // Five 24 pt icons and four 4 pt gaps are 136 pt, which is the number.
    XCTAssertEqual(5 * Grid.actionSize + 4 * Grid.actionGap, 136)

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
          ActionButton(.pin, tip: Tips.pin)
        }
      })
    let measured = try probes(row, scheme: .dark)
    XCTAssertEqual(measured["row.trailing"]?.width, 136)

    // `.row.is-busy .acts{display:none}`: a busy row carries no action at
    // all, so the column goes with them.
    let busy = RowView(
      .doc, glyph: .plain, tone: .quiet, title: drawn.title, sub: drawn.sub, busy: true,
      trailing: { ActionButton(.gdocs, tip: Tips.openDoc) })
    let busyMeasured = try probes(busy, scheme: .dark)
    XCTAssertNil(busyMeasured["row.trailing"], "a busy row draws no action column")
  }

  /// Every row in the panel, a document row included, starts on the panel's
  /// one left edge.
  ///
  /// A document row kept 8 pt in front of its glyph for the pin until the
  /// third look, and the rows under a pinned one kept it empty, so DOCUMENTS
  /// stood right of its own section label and of every other row in the
  /// panel. The pin is the row's fifth icon now, at the trailing edge where
  /// the other four are, and the leading column is the glyph's own 16 pt in
  /// every shape there is.
  func testEveryRowShapeKeepsTheGlyphsOwnLeadingColumn() throws {
    for shape in RowShape.allCases {
      XCTAssertEqual(shape.leadingColumn, 16, "\(shape) keeps the glyph's own column")
    }

    let drawn = try SheetPieces.documentRows()[0]
    let row = RowView(
      .doc, glyph: drawn.glyph, tone: drawn.tone, title: drawn.title, sub: drawn.sub,
      trailing: { ActionButton(.gdocs, tip: Tips.openDoc) })
    let measured = try probes(row, scheme: .dark)
    XCTAssertEqual(
      measured["row.leading"]?.width, Grid.glyphSize,
      "the document row's glyph column is every other row's")

    let left = try XCTUnwrap(
      try probeList(row, scheme: .dark).first { $0.name == "row.leading" })
    let plain = try XCTUnwrap(
      try probeList(
        RowView(.plain, glyph: drawn.glyph, tone: drawn.tone, title: drawn.title),
        scheme: .dark
      ).first { $0.name == "row.leading" })
    XCTAssertEqual(
      left.origin.x, plain.origin.x, "and it starts where every other row starts")
  }

  /// `.acts{margin-right:-4px}`: the icons and not their boxes meet the
  /// column every value in the panel ends on.
  ///
  /// An action's artwork is 16 pt centred in a 24 pt box, so a strip whose
  /// boxes ended on the row's content edge drew its icons 4 pt inside it:
  /// every time in TODAY, every value in SETTINGS and the DOCUMENTS count
  /// ended 4 pt right of every icon under them.
  func testAnActionStripMeetsTheColumnEveryValueEndsOn() throws {
    let drawn = try SheetPieces.documentRows()[0]
    let row = RowView(
      .doc, glyph: drawn.glyph, tone: drawn.tone, title: drawn.title, sub: drawn.sub,
      trailing: {
        HStack(spacing: Grid.actionGap) {
          ActionButton(.gdocs, tip: Tips.openDoc)
          ActionButton(.md, tip: Tips.openNote)
          ActionButton(.branch, tip: Tips.openChat)
          ActionButton(.link, tip: Tips.copyLink)
          ActionButton(.pin, tip: Tips.pin)
        }
      })
    let strip = try XCTUnwrap(
      try probeList(row, scheme: .dark).first { $0.name == "row.trailing" })
    XCTAssertEqual(
      strip.origin.x + strip.size.width, Grid.panelWidth - Grid.rowInset + Grid.actionOverhang,
      "the strip hangs 4 pt past the row's inset")
    XCTAssertEqual(
      strip.origin.x + strip.size.width - Grid.actionOverhang, Grid.panelWidth - Grid.rowInset,
      "so the artwork inside it ends on the row's own edge, where a value ends")
  }

  // Whatever is at the trailing edge of a one-line row sits in the middle
  // of it.
  //
  // A one-line row has nothing to hang its three columns from, so they are
  // centred: a 24 pt icon in a 28 pt row drawn a quarter of a row under its
  // words reads as a mark on an empty line. The SETTINGS row with an icon
  // on it is the one the panel draws today.
  func testAOneLineRowCentresWhatIsOnIt() throws {
    let drawn = try probeList(
      RowView(
        .compact, glyph: .blank, title: twenty,
        trailing: { ActionButton(.down, tip: Tips.installExtension) }),
      scheme: .dark)
    let row = try XCTUnwrap(drawn.first { $0.name == "row.compact" })
    let trailing = try XCTUnwrap(drawn.first { $0.name == "row.trailing" })

    XCTAssertEqual(trailing.size.height, Grid.actionSize, "the icon's own box")
    XCTAssertEqual(
      trailing.origin.y + trailing.size.height / 2,
      row.origin.y + row.size.height / 2, accuracy: 0.5,
      "a one-line row centres what is on its right")
  }

  // One right edge for every value in the panel.
  //
  // `.row.compact` puts its value at the content edge, the same edge as a
  // log row's time; only a row with something on its right pulls the value
  // in to make room for it. SETTINGS's rows with no icon were handing the
  // layout an empty action column, and 8 pt of nothing moved every mark and
  // keystroke left of every time in TODAY.
  func testAValueEndsWhereEveryOtherValueEnds() throws {
    let value = SettingsRows.quitShortcut
    for scheme in [ColorScheme.dark, .light] {
      let log = try inkEnds(
        RowView(.plain, glyph: .check, tone: .ok, title: "Reply", value: value), scheme: scheme)
      let settings = try inkEnds(
        SettingsRows(rows: [SettingsRow(.quit, value: value, glyph: .check, tone: .ok)]),
        scheme: scheme)
      XCTAssertEqual(settings, log, "the SETTINGS value is not on the log's edge in \(scheme)")

      // A row that does carry an icon keeps its value clear of it, which is
      // the Claude Desktop row.
      let withIcon = try probes(
        SettingsRows(
          rows: [
            SettingsRow(
              .desktop, value: "not installed", glyph: .empty,
              action: IconAction(.down, tip: Tips.installExtension))
          ]), scheme: scheme)
      XCTAssertNotNil(
        withIcon["row.trailing"], "a row with an icon draws the column in \(scheme)")

      let bare = try probes(
        SettingsRows(rows: [SettingsRow(.quit, value: value, glyph: .check, tone: .ok)]),
        scheme: scheme)
      XCTAssertNil(bare["row.trailing"], "a row with nothing on its right draws no column")
    }
  }

  // `.section-label{height:28px;text-transform:uppercase}`, and the name of
  // the section is the whole of what it says: the count on its right went
  // with the lists it was a number about.
  func testASectionLabelIs28AndUppercase() throws {
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
      let measured = try probes(SectionLabel("Documents"), scheme: scheme)
      XCTAssertEqual(measured["section-label"]?.height, 28, "the label in \(scheme)")
      XCTAssertEqual(measured["section-label"]?.width, Grid.panelWidth)
    }

    // `.section:first-of-type{margin-top:0;padding-top:0;border-top:0}`: the
    // first section sits against the header, and every other one carries
    // 8 pt, a hairline and 4 pt.
    let first = try probes(
      SectionView("Now", isFirst: true) { RowView(title: twenty) }, scheme: .dark)
    let later = try probes(
      SectionView("Documents") { RowView(title: twenty) }, scheme: .dark)
    let firstHeight = try XCTUnwrap(first["section"]?.height)
    let laterHeight = try XCTUnwrap(later["section"]?.height)
    XCTAssertEqual(laterHeight - firstHeight, Grid.unit + 4, "8 pt of air and 4 pt of padding")
  }

  // `.panel-header{height:36px}`: the mark and the name, the version with
  // its update, and the account with its glyph and its one action.
  //
  // Everything the header holds is on one 36 pt line, and the line is what
  // the panel gives it: 320 pt less the panel's own 8 twice. Seven pieces
  // at that width leave 4 pt over, so the test measures the address against
  // the width it wants. A stack that spent 8 of those pt on a gap drew
  // "nail@example…" on every screen.
  func testTheHeaderIs36WithTheMarkTheNameTheVersionAndTheAccount() throws {
    XCTAssertEqual(Grid.headerHeight, 36)

    let account = "nail@example.com"
    let signedIn = HeaderModel(
      mark: "⁘", version: "v2.16.0",
      update: ActionModel(.update, icon: .down, tip: Tips.checkUpdate, target: "gdoc"),
      account: account, glyph: .check, tone: .ok,
      actions: [ActionModel(.signOut, icon: .signout, tip: Tips.signOut, target: "auth")])
    let measured = try probes(PanelHeader(model: signedIn), scheme: .dark, width: line)
    XCTAssertEqual(measured["header"]?.height, 36 + 4, "the header and its 4 pt margin under it")
    XCTAssertEqual(measured["header.line"]?.height, 36, "the line itself")
    XCTAssertNotNil(measured["header.mark"], "the mark in effect")
    XCTAssertNotNil(measured["header.name"], "the word gdoc")
    XCTAssertNotNil(measured["header.version"], "the version after it")
    XCTAssertNotNil(measured["header.glyph"], "the check over a token that works")
    XCTAssertNotNil(measured["header.account"], "the address")
    XCTAssertNotNil(measured["header.signOut"], "and the one action")
    XCTAssertNotNil(measured["header.update"], "the update glyph, which is always on the line")
    XCTAssertNil(
      measured["header.update.dot"], "and nothing newer is known, so it carries no dot")

    // The account is the quietest grey in the panel, and the name is the
    // brightest: whose account it is decides nothing, so it is never loud.
    XCTAssertEqual(Typography.caption.size, 11)
    XCTAssertEqual(Typography.mark.size, 16)
    XCTAssertEqual(Typography.titleEmphasis.weight, .medium)

    // The address is drawn at the width it asks for, which is the whole of
    // it: the one thing on this line that is cut rather than wrapped.
    let wanted = try probes(
      Text(account).font(Typography.caption.font).fixedSize().metricProbe("alone"),
      scheme: .dark, width: line)
    XCTAssertEqual(
      measured["header.account"]?.width, wanted["alone"]?.width,
      "the address is squeezed on the line, so nobody can read the end of it")

    // A newer release is the same icon with the accent dot over it, and the
    // version that is published is the line under, not a second number on
    // this one: both of them here squeeze the address to half, which is the
    // one thing the morning of the same day was about.
    let behind = try probes(
      PanelHeader(
        model: HeaderModel(
          mark: "⁘", version: "v2.16.0",
          update: ActionModel(.update, icon: .down, tip: "Update to v2.17.0", target: "gdoc"),
          updateLook: .available,
          account: account, glyph: .check, tone: .ok,
          actions: [ActionModel(.signOut, icon: .signout, tip: Tips.signOut, target: "auth")],
          quiet: ["v2.17.0 is published"])),
      scheme: .dark, width: line)
    XCTAssertNotNil(behind["header.update"], "there is a release to take")
    XCTAssertEqual(
      behind["header.update.dot"]?.width, Grid.updateDotSize, "and the dot says so at a glance")
    XCTAssertEqual(behind["header.line"]?.height, 36, "the line itself is where it was")
    XCTAssertEqual(
      behind["header.account"]?.width, wanted["alone"]?.width,
      "and the address is whole, which is what the line has room for")

    // The measurement that decided it: both versions in the version slot
    // take the address down to half of itself.
    let crowded = try probes(
      PanelHeader(
        model: HeaderModel(
          mark: "⁘", version: "v2.16.0 → v2.17.0",
          update: ActionModel(.update, icon: .down, tip: "Update to v2.17.0", target: "gdoc"),
          updateLook: .available,
          account: account, glyph: .check, tone: .ok,
          actions: [ActionModel(.signOut, icon: .signout, tip: Tips.signOut, target: "auth")])),
      scheme: .dark, width: line)
    XCTAssertLessThan(
      try XCTUnwrap(crowded["header.account"]?.width),
      try XCTUnwrap(wanted["alone"]?.width) / 2 + 1,
      "which is nail@ex… on every screen, and why the number is under the line")

    // The mark is a setting and may be empty, and the empty mark draws
    // nothing rather than a gap.
    let noMark = try probes(
      PanelHeader(model: HeaderModel(account: account)), scheme: .dark, width: line)
    XCTAssertNil(noMark["header.mark"])
    XCTAssertEqual(noMark["header"]?.height, 40, "an empty mark keeps the header's height")

    // No token, no address: the header is the mark, the name and Sign in.
    let signedOut = try probes(
      PanelHeader(
        model: HeaderModel(
          mark: "⁘", version: "v2.16.0", account: AuthWord.signedOut.rawValue,
          glyph: .exclamation, tone: .bad,
          actions: [ActionModel(.signIn, icon: .signin, tip: Tips.signIn, target: "auth")])),
      scheme: .dark, width: line)
    XCTAssertNil(signedOut["header.signOut"], "there is nothing to sign out of")
    XCTAssertNotNil(signedOut["header.signIn"], "and one thing to do instead")
  }

  /// What the panel gives the header: its own width less its 8 pt padding
  /// on either side.
  private var line: CGFloat { Grid.panelWidth - 2 * Grid.unit }

  // "Sign out?" is asked in place of the header's line: the question where
  // the name was, the pair where the icon was, and the consequence under it.
  func testTheQuestionAsksInPlaceOfTheHeaderLine() throws {
    let asking = HeaderModel(
      mark: "⁘",
      question: QuestionModel(
        title: "Sign out?", consequence: "2 live watches will stop.", danger: .signOut),
      quiet: ["Account changed: first@example.com → second@example.com"])
    let measured = try probes(PanelHeader(model: asking), scheme: .dark)

    XCTAssertNil(measured["header.account"], "the address is not beside the question")
    XCTAssertNil(measured["header.version"], "and neither is the version")
    XCTAssertNotNil(measured["confirm.danger"], "Sign out is where the icon was")
    XCTAssertNotNil(measured["confirm.keep"], "and Keep beside it")
    XCTAssertGreaterThan(
      try XCTUnwrap(measured["header"]?.height), 36 + 4,
      "the consequence and the changed address are lines under it")
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

  /// The sheet: the four shapes, a header, a quiet line and the two labels,
  /// in both appearances, so the layout is looked at once.
  ///
  /// Every word on it is a scene's, through `SheetPieces`: the sheet is the
  /// one page with no record behind it, and a string written by hand here
  /// is a string Nail reads as the app's own.
  func testTheRowSheetRenders() throws {
    let shots = try shotsDirectory()
    let labels = try SheetPieces.sectionLabels()
    let live = try SheetPieces.live()
    let rows = try SheetPieces.documentRows()
    let compact = try XCTUnwrap(
      SheetPieces.settings().first { $0.kind == .quit }, "no scene drew the Quit row")
    let quiet = try SheetPieces.quiet()
    let header = try SheetPieces.header()

    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      // The four row shapes, and the two labels as components of their own.
      //
      // A label over rows is a section, and a section's header makes a claim
      // about what is under it: NOW over a document row and a SETTINGS row
      // is this page saying something no panel says. The pieces here are
      // pieces, in the order `RowShape` names them, and the labels stand at
      // the foot with nothing under them:
      // testASheetPutsNoLabelOverRowsFromAnotherScene.
      let sheet = VStack(alignment: .leading, spacing: 0) {
        PanelHeader(model: header)
        RowView(.two, glyph: .dot, tone: .live, title: live.title, sub: live.sub)
        RowView(.plain, glyph: .check, tone: .ok, title: rows[0].title)
        RowView(.compact, glyph: .blank, title: compact.kind.label, value: compact.value)
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
        QuietLine(quiet)
        // The two labels, each on its own and with air around it: stacked,
        // the first reads as a header over the second.
        Spacer(minLength: 2 * Grid.unit)
        SectionLabel(labels.now)
        Spacer(minLength: 2 * Grid.unit)
        SectionLabel(labels.documents)
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
