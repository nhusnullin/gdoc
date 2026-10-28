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

  // The panel is 360 points wide since the fifth look, the width Nail chose
  // of the four he was offered (docs/v2/DECISIONS.md, 2026-10-28). It was
  // 320 from M23 to v2.24.0, and the number is read back out of a row rather
  // than out of the constant, because a window that opens at another width
  // passes any test that reads the constant.
  func testThePanelIs360PointsWide() throws {
    XCTAssertEqual(Grid.panelWidth, 360)

    let row = RowView(
      .doc, glyph: .check, tone: .ok, title: twenty, sub: "Proposed · verified",
      trailing: { EmptyView() })
    XCTAssertEqual(
      try probes(row, scheme: .dark)["row.doc"]?.width, 360,
      "a document row is drawn at a width other than the panel's")
  }

  // The title column is 180 pt in a document row, which is the narrowest the
  // panel has, and the head never falls under eight characters in it: a
  // title whose last word is wide enough to squeeze it drops the tail and
  // goes in whole instead. It was 112 pt until the fifth look, which took
  // the Google Docs icon off the row for 28 pt of it and widened the panel
  // for the other 40.
  func testNoHeadUnderEightCharactersAtPanelWidth() {
    XCTAssertEqual(
      MiddleTitle.narrowestColumn, 180,
      "360 less the panel's 8 twice, the row's 12 twice, the 16 pt glyph, two 8 pt gaps "
        + "and the 108 pt action column")

    // The ellipsis is part of what the column carries. Scene r9's second
    // document is the title that proved it: a tail the counted rule admits,
    // a head the room admits, and seven characters of head in the 112 pt
    // column once the row had drawn the three dots after them. It shows
    // fourteen in the 180 the fifth look left it.
    let ellipsisEater = "B2B2C payment card concept: the frozen vocabulary"

    for title in [twenty, forty, sixty, eighty, ellipsisEater] {
      let shown = MiddleTitle.headCharacters(title)
      XCTAssertGreaterThanOrEqual(
        shown, Titles.headFloor, "\"\(title)\" shows only \(shown) characters of head")
    }

    // A tail the counted rule admits but the column cannot carry: thirteen
    // characters and a space, after a head that starts wide. The measured
    // floor drops it, so the row draws the whole title with an ellipsis
    // rather than nine characters and a word.
    //
    // The head has to start wide for the 180 pt column to refuse the tail:
    // " announcements" is 99 pt of it, "AMS/MUC a…" is 82, and 181 is one pt
    // more than the column has. The same tail after "Treasury " was refused
    // at 140 pt and is carried at 180.
    let wideTail = "AMS/MUC and the Munich announcements"
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

  // A verb in front of a name takes its own room, and the floor counts the
  // name.
  //
  // Scene r9's watch was drawn `Live stopped by you · Ch 5.… v0.1`, the same
  // document DOCUMENTS showed as `Ch 5. Commerc… v0.1`: the floor was
  // measured on the verb and the name together, so it passed at 27
  // characters while the name inside it was cut to five. The verb and the
  // dot after it are one piece now, drawn whole, and the name is measured in
  // what is left of the row.
  func testAPrefixTakesItsRoomAndTheFloorCountsTheName() throws {
    // The name is longer than the one the backlog item was written about,
    // because the 268 pt a NOW row has at 360 carries "Ch 5. Commercial
    // Model v0.1" whole with the verb in front of it, and a line that fits
    // proves nothing about the room a verb takes. This one is cut in both
    // columns, so the two measurements can be told apart.
    let name = "Ch 5. Commercial Model and its appendices v0.1"
    let verb = "Live stopped"

    // The verb is drawn whole, separator and all, and the two pieces put
    // back together are the line a reader sees.
    XCTAssertEqual(MiddleTitle.lede(verb), "Live stopped · ")
    XCTAssertEqual(
      MiddleTitle.whole(prefix: verb, name: name),
      "Live stopped · Ch 5. Commercial Model and its appendices v0.1")
    XCTAssertEqual(MiddleTitle.whole(prefix: nil, name: name), name, "a name is a line of its own")

    // A NOW row's own room: the panel's 360 less its 8 either side, the
    // row's 12 either side, the 16 pt glyph, the two 8 pt gaps and the one
    // 24 pt action, that action's 4 pt of overhang given back.
    XCTAssertEqual(MiddleTitle.nowColumn, 268)
    let row = LiveRow(model: LiveRowModel(prefix: verb, name: name, sub: "since 13:10"))
    let inPanel = Grid.panelWidth - 2 * Grid.unit
    XCTAssertEqual(
      try probes(row, scheme: .dark, width: inPanel)["row.title"]?.width,
      MiddleTitle.nowColumn,
      "the row drew a title column other than the one the name is measured in")

    // The name keeps its last word and at least the floor's characters of
    // its head, which is what DOCUMENTS gives it.
    XCTAssertEqual(
      MiddleTitle.fitted(name, prefix: verb, column: MiddleTitle.nowColumn).tail, " v0.1",
      "the name lost the one word that tells it from the draft beside it")
    let shown = MiddleTitle.headCharacters(name, prefix: verb, column: MiddleTitle.nowColumn)
    XCTAssertGreaterThanOrEqual(
      shown, Titles.headFloor, "the name shows only \(shown) characters of head")

    // The verb takes room and nothing hides that: the same name shows more
    // of itself with no verb in front of it, in the same column.
    XCTAssertGreaterThan(
      MiddleTitle.headCharacters(name, column: MiddleTitle.nowColumn), shown,
      "the verb was measured as though it took no room")

    // And it is why a row that opens with a verb is measured in its own
    // room rather than the document row's 180: there, the verb takes the
    // name's one telling word away.
    //
    // At 320 the claim was the floor itself, six characters of head in the
    // 140 pt column against a floor of eight. 180 pt holds eleven, so the
    // floor is no longer what breaks; what breaks is the tail. The verb
    // leaves 92 pt, the tail " v0.1" wants 27 of it and the floor after it
    // wants 73, so the tail is dropped and the name loses the v0.1 that
    // tells it from the draft beside it, which the same column gives it
    // whole with no verb in front.
    XCTAssertNil(
      MiddleTitle.fitted(name, prefix: verb).tail,
      "the document row's column has room for the verb and the tail after all")
    XCTAssertEqual(MiddleTitle.fitted(name).tail, " v0.1", "which it keeps with no verb")
    XCTAssertEqual(MiddleTitle.headCharacters(name, prefix: verb), 11)
    XCTAssertEqual(
      MiddleTitle.headCharacters(name), 20,
      "the verb costs the name nine characters of its head in this column")

    // The job card is the same rule at the card's own width and weight, so
    // a publish's name is cut in the middle like every other name in the
    // panel and not at its tail: 360 less the panel's 8 either side, the
    // card's 12 either side, the 28 pt ring and the 12 pt gap after it.
    XCTAssertEqual(JobCard.titleColumn, 280)
    let card = JobCardModel(prefix: "Publishing", name: name, sub: "started 09:03")
    XCTAssertEqual(
      card.title, "Publishing · Ch 5. Commercial Model and its appendices v0.1")
    let job = MiddleTitle.fitted(
      name, prefix: "Publishing", role: Typography.titleEmphasis, column: JobCard.titleColumn)
    XCTAssertEqual(job.tail, " v0.1", "the card cut the name at its tail")
    let onCard = MiddleTitle.headCharacters(
      name, prefix: "Publishing", role: Typography.titleEmphasis, column: JobCard.titleColumn)
    XCTAssertGreaterThanOrEqual(
      onCard, Titles.headFloor, "the card shows only \(onCard) characters of head")
    XCTAssertNotNil(
      try probes(JobCard(model: card), scheme: .dark, width: inPanel)["job.title"],
      "the card draws its title somewhere other than through MiddleTitle")

    // A name with no verb measures in the document row's column, by the
    // counted split and the head the ellipsis leaves room for: the head is
    // the whole of the name less its last word, and twenty characters of it
    // are drawn in the 180 pt column.
    XCTAssertEqual(MiddleTitle.fitted(name).tail, " v0.1")
    XCTAssertEqual(
      MiddleTitle.fitted(name).head, "Ch 5. Commercial Model and its appendices")
    XCTAssertEqual(MiddleTitle.headCharacters(name), 20)
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
    // 360 less the row's 12 either side, the 16 pt glyph and the 8 pt gap
    // after it.
    let column = try XCTUnwrap(measured["row.sub"]?.width)
    XCTAssertEqual(column, 312)
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
  // icons in 108 pt since the fifth look took the Google Docs icon off the
  // row, so every document row in a list breaks its title at the same place.
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
          ActionButton(.md, tip: Tips.openNote)
          ActionButton(.branch, tip: Tips.openChat)
          ActionButton(.link, tip: Tips.copyLink)
          ActionButton(.pin, tip: Tips.pin)
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

  /// Every row in the panel, a document row included, starts on the panel's
  /// one left edge.
  ///
  /// A document row kept 8 pt in front of its glyph for the pin until the
  /// third look, and the rows under a pinned one kept it empty, so DOCUMENTS
  /// stood right of its own section label and of every other row in the
  /// panel. The pin is the row's last icon, at the trailing edge where
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

  /// The switch stops where every other right edge stops.
  ///
  /// A row pulls its trailing column 4 pt out over the content edge,
  /// because an action's 16 pt of artwork sits in a 24 pt box and it is the
  /// artwork that meets the edge. The switch is a 26 pt pill with no such
  /// margin inside it, so the same pull put the one control in the panel
  /// 4 pt right of every icon, every time and the ⌘Q directly under it. It
  /// is drawn on nearly every screen since Launch at login went off by
  /// default, which is what made it the thing an eye lands on.
  func testTheSwitchStopsWhereTheKeystrokeUnderItStops() throws {
    for scheme in [ColorScheme.dark, .light] {
      let keystroke = try inkEnds(
        SettingsRows(
          rows: [SettingsRow(.quit, value: SettingsRows.quitShortcut, glyph: .check, tone: .ok)]),
        scheme: scheme)
      for isOn in [true, false] {
        let settings = try inkEnds(
          SettingsRows(rows: [SettingsRow(.launchAtLogin, value: "", glyph: .blank, isOn: isOn)]),
          scheme: scheme)
        XCTAssertEqual(
          settings, keystroke, accuracy: 1,
          "the switch is off the panel's one right edge in \(scheme), on: \(isOn)")
      }
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
  // the panel gives it: 360 pt less the panel's own 8 twice. The seven
  // pieces take 252 pt of that at the widths they ask for, so the test
  // measures the address against the width it wants and never against what
  // is left over. A stack that spent 8 of those pt on a gap drew
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

    // A newer release is the same icon with the accent dot over it, and no
    // second number on this line: both of them here cut the address, which
    // is the one thing the morning of the same day was about. The number is
    // on the tooltip.
    let behind = try probes(
      PanelHeader(
        model: HeaderModel(
          mark: "⁘", version: "v2.16.0",
          update: ActionModel(.update, icon: .down, tip: "Update to v2.17.0", target: "gdoc"),
          updateLook: .available,
          account: account, glyph: .check, tone: .ok,
          actions: [ActionModel(.signOut, icon: .signout, tip: Tips.signOut, target: "auth")])),
      scheme: .dark, width: line)
    XCTAssertNotNil(behind["header.update"], "there is a release to take")
    XCTAssertEqual(
      behind["header.update.dot"]?.width, Grid.updateDotSize, "and the dot says so at a glance")
    XCTAssertEqual(behind["header.line"]?.height, 36, "the line itself is where it was")
    XCTAssertEqual(
      behind["header.account"]?.width, wanted["alone"]?.width,
      "and the address is whole, which is what the line has room for")

    // The measurement that decided it: both versions in the version slot
    // take the address down from the whole 98 pt to 90.5, which is an
    // address with its end cut. At 320 it went to 49, and the number it
    // would take is on the tooltip either way.
    let crowded = try probes(
      PanelHeader(
        model: HeaderModel(
          mark: "⁘", version: "v2.16.0 → v2.17.0",
          update: ActionModel(.update, icon: .down, tip: "Update to v2.17.0", target: "gdoc"),
          updateLook: .available,
          account: account, glyph: .check, tone: .ok,
          actions: [ActionModel(.signOut, icon: .signout, tip: Tips.signOut, target: "auth")])),
      scheme: .dark, width: line)
    XCTAssertEqual(try XCTUnwrap(crowded["header.account"]?.width), 90.5)
    XCTAssertLessThan(
      try XCTUnwrap(crowded["header.account"]?.width),
      try XCTUnwrap(wanted["alone"]?.width),
      "which is an address with its end cut, and why the number is on the tooltip")

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
    let glyph = try SheetPieces.headerStates()

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
        // The header four times, because the update glyph is drawn four
        // ways and a panel shows one of them at a time: the accent ink with
        // its dot, the caption grey of a machine told of nothing newer, the
        // second after a press that installed nothing, and the ring while
        // the child is up. The last two are on no scene of a day, which is
        // the whole reason they are here:
        // testTheRowSheetDrawsTheUpdateGlyphInEveryDrawingItHas.
        PanelHeader(model: header)
        PanelHeader(model: glyph.quiet)
        PanelHeader(model: glyph.checked)
        PanelHeader(model: glyph.running)
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

  /// Every drawing of the update glyph is on the page.
  ///
  /// The rule is the second morning's: every state a milestone adds is on a
  /// screen, or it is new surface nobody has looked at. The third look gave
  /// the glyph the accent ink with a dot, the caption grey, the check a
  /// second long after a press that installed nothing, and the ring while
  /// the child is up. The first two are on scenes of a day, r8 and e1. The
  /// other two last as long as a press does and belong to no moment of a
  /// day, so the sheet is the only page they can be on: both were drawn
  /// nowhere until this pass.
  ///
  /// The weights are read off the drawn pixels rather than off the model,
  /// because a header the panel hands the right look and draws the same way
  /// twice is the thing this is for.
  func testTheRowSheetDrawsTheUpdateGlyphInEveryDrawingItHas() throws {
    let available = try SheetPieces.header()
    let states = try SheetPieces.headerStates()

    XCTAssertEqual(available.updateLook, .available, "r8 is the screen with a release")
    XCTAssertEqual(available.update?.state, .live, "and nothing has been pressed on it")
    XCTAssertEqual(states.quiet.updateLook, .quiet)
    XCTAssertEqual(states.quiet.update?.state, .live)
    XCTAssertEqual(states.checked.update?.state, .done, "the second after a press")
    XCTAssertEqual(
      states.checked.updateLook, .quiet,
      "a check that found nothing is a machine told of nothing newer")
    XCTAssertTrue(states.running.versionWaiting, "the ring while the child is up")
    XCTAssertNil(states.running.update, "and nothing to press beside it")

    // The available header's own line, and nobody else's: a machine told of
    // nothing newer has nothing to say under the header. On r8 that line is
    // the press of 17:04, which is the one thing the line is for since the
    // fourth look: a release nobody pressed for writes no line at all.
    XCTAssertEqual(available.quiet, ["GitHub did not answer at 17:04. Nothing changed."])
    XCTAssertEqual(states.quiet.quiet, [])
    XCTAssertEqual(states.checked.quiet, [])
    XCTAssertEqual(states.running.quiet, [])

    // Four headers, four images. Two of them drawing the same weight is two
    // states a person cannot tell apart.
    for scheme in [ColorScheme.dark, .light] {
      let drawn = [available, states.quiet, states.checked, states.running]
      let weights = try drawn.map {
        try inkWeight(PanelHeader(model: $0), scheme: scheme, side: Grid.panelWidth)
      }
      XCTAssertGreaterThan(
        weights[0], weights[1],
        "the available header carries the dot and the published line in \(scheme)")
      for (first, second) in [(1, 2), (1, 3), (2, 3)] {
        XCTAssertNotEqual(
          weights[first], weights[second], accuracy: 0.5,
          "headers \(first) and \(second) are one image in \(scheme)")
      }
    }
  }
}
