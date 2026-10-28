// The nine pieces the panel's sections are made of, and the words in them.
//
// Every number below is a literal read out of
// docs/design/2026-10-17-presence-screens-v3.html by hand: `.hold`, `.stuck`,
// `.row-live`, `.now-job`, `.empty`, `.bigmail`, `.urlline`, `.strip` and the
// `HROW` rows of SETTINGS. The words are literals too, because a sentence a
// person reads in the menu bar is a decision and not a string somebody
// improved.
//
// Three of these tests are about words rather than pixels. The hold card
// never says Release, the strip says four of them, and the settings rows keep
// one order on every screen: each is a thing the panel would still draw
// wrongly with every size right.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class CardTests: XCTestCase {
  /// The width a card is drawn at inside the panel: 360 less the panel's own
  /// 8 pt either side.
  private let cardWidth = Grid.panelWidth - 2 * Grid.unit

  // `.hold{padding:10px 12px;border-radius:8px}` on the amber tint, with the
  // two buttons the specification's decision table names. The screens file
  // labels the second one "Release without writing"; the decision table says
  // the panel never says Release and carries Dismiss instead, and a decision
  // wins over a label.
  func testTheHoldCardSaysOpenChatAndDismissAndNeverRelease() throws {
    XCTAssertEqual(HoldCard.words, [.openChat, .dismiss])
    XCTAssertEqual(HoldCard.words.map(\.rawValue), ["Open chat", "Dismiss"])
    for word in HoldCard.words {
      XCTAssertFalse(word.rawValue.contains("Release"))
    }
    // The word itself is nowhere in the file that would have drawn it.
    let source = try componentSource("HoldCard.swift")
    XCTAssertFalse(
      source.contains("Release without"),
      "HoldCard.swift carries the screens file's label instead of the decision's")

    // The plate is the amber tint and its half-pixel edge, and never the
    // amber block of the second version.
    for palette in [Palette.dark, Palette.light] {
      XCTAssertEqual(HoldCard.tint(in: palette), palette.holdTint)
      XCTAssertEqual(HoldCard.edge(in: palette), palette.holdTintEdge)
    }
    XCTAssertEqual(Grid.cardRadius, 8)
    XCTAssertEqual(Grid.cardInset, 10)
    XCTAssertEqual(HoldCard.quietTop, 6)
    XCTAssertEqual(HoldCard.actionTop, 10)
    XCTAssertEqual(HoldCard.actionGap, 6)
    XCTAssertEqual(HoldCard.bodyInset, 24, "the glyph's 16 pt and the row's 8 pt gap")

    let measured = try probes(HoldCard(model: try hold()), scheme: .dark, width: cardWidth)
    XCTAssertNotNil(measured["hold.card"])
    XCTAssertNotNil(measured["hold.quiet"], "the card says where the text is waiting")
    XCTAssertEqual(measured["hold.open-chat"]?.height, 24, "`.btn{height:24px}`")
    XCTAssertEqual(measured["hold.dismiss"]?.height, 24)

    // Pressed once, the card asks, and the pair that answers is the inline
    // confirm the panel already has.
    let asking = try probes(
      HoldCard(model: try hold(), asking: true), scheme: .dark, width: cardWidth)
    XCTAssertNotNil(asking["confirm.danger"])
    XCTAssertNotNil(asking["confirm.keep"])
    XCTAssertNil(asking["hold.open-chat"], "a card that is asking carries no other action")

    // The consequence is a line of its own, so the sentence is drawn whole
    // rather than cut after its first clause by the pair beside it. Two
    // lines of it at this width, which is what the words come to.
    let quiet = try XCTUnwrap(asking["quiet"], "the consequence has no line of its own")
    XCTAssertGreaterThan(
      quiet.height, Typography.rowValue.lineHeight + 4,
      "the consequence was squeezed onto one line and cut")
  }

  /// A card cuts a title by the row's own rule.
  ///
  /// Scene e3's export stopped on "Proposal: Expansion of Project Falcon to
  /// two further regions, draft v1". The second line was left to the
  /// system's middle truncation, which knows nothing about words and drew
  /// "Proposal: Expansio…her regions, draft v1": a reader sees the word
  /// "her regions", which is in no title of anybody's. The row's rule keeps
  /// the tail a whole word group, so the ellipsis is the only thing between
  /// the two halves.
  func testACardCutsATitleByTheRowsOwnRule() throws {
    let title =
      "Export stopped · Proposal: Expansion of Project Falcon to two further regions, draft v1"
    let lines = StuckCard.titleSplit(title, actions: 5)
    XCTAssertEqual(lines.count, 2, "the card takes two lines for it")
    XCTAssertEqual(lines[0], "Export stopped")

    let second = try XCTUnwrap(lines.last)
    let column = StuckCard.column(line: 1, actions: 5)
    XCTAssertEqual(column, StuckCard.reasonColumn, "the second line has the card's full body")

    let split = MiddleTitle.fitted(second, role: Typography.titleEmphasis, column: column)
    let tail = try XCTUnwrap(split.tail, "the rule kept no tail at all")
    XCTAssertTrue(
      second.hasSuffix(tail), "the tail is the title's own ending and nothing else")
    XCTAssertTrue(
      tail.hasPrefix(" ") || tail.hasPrefix(","),
      "the tail starts at a word: it is “\(tail)”")
    XCTAssertFalse(
      tail.contains("her regions"), "the tail starts inside a word of the title")

    // The first line is cut by the same rule, at the width the icons leave.
    XCTAssertEqual(StuckCard.column(line: 0, actions: 5), StuckCard.titleColumn(actions: 5))
  }

  /// A confirm draws no line it has no words for.
  ///
  /// The question is one line and its consequence is a line of the card's,
  /// so the row is the one-line shape: the two-line shape reserved a
  /// sub-line it never drew, and the card showed 21 pt of nothing between
  /// the question and the sentence under it, which reads as a line that
  /// failed to draw.
  func testTheConfirmDrawsNoLineItHasNoWordsFor() throws {
    let bare = try probes(
      ConfirmRow(title: ButtonWord.dismiss.rawValue + " this card?", danger: .dismiss),
      scheme: .dark, width: cardWidth)
    XCTAssertNil(bare["row.two"], "a question with no second line is not a two-line row")
    XCTAssertEqual(bare["row.plain"]?.height, Grid.rowHeight)

    // A confirm that does carry a sub-line keeps the two-line shape, which
    // is the header's Sign out with the count of what will stop.
    let withSub = try probes(
      ConfirmRow(title: "Sign out?", sub: "2 live watches and 1 hold will stop.", danger: .signOut),
      scheme: .dark, width: cardWidth)
    XCTAssertGreaterThanOrEqual(
      withSub["row.two"]?.height ?? 0, Grid.twoLineRowHeight,
      "a question with a sentence under it keeps the two-line shape")

    // The card is shorter than the two lines it was drawing, and nothing
    // between the question and the consequence is empty.
    let asking = try probes(
      HoldCard(model: try hold(), asking: true), scheme: .dark, width: cardWidth)
    let plate = try XCTUnwrap(asking["hold.card"], "the card drew no plate")
    let question = try XCTUnwrap(asking["row.plain"], "the question is not a one-line row")
    let quiet = try XCTUnwrap(asking["quiet"], "the consequence has no line of its own")
    XCTAssertEqual(
      plate.height, question.height + quiet.height + 2 * Grid.rowInset,
      "the plate is its padding, the question and the consequence, and nothing else")
  }

  /// A question inside the hold's tint is written in the plate's own ink.
  ///
  /// Principle 8 gives the light hold tint its own ink, and the offer card
  /// uses it for every line it draws. The question and its consequence sat
  /// on the same tint in the panel's neutral greys: two ink systems on one
  /// colour, one card apart.
  func testTheHoldConfirmUsesThePlatesOwnInk() throws {
    XCTAssertEqual(
      HoldCard.titleInk(in: .light), Palette.light.holdInk,
      "the plate's ink on white is the brown one")
    XCTAssertEqual(HoldCard.quietInk(in: .light), Palette.light.holdInk)

    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      let asking = HoldCard(model: try hold(), asking: true)
      let offer = HoldCard(model: try hold())
      // The same ink reaches both, which is the only way a reader of one
      // card after another sees one plate and not two.
      XCTAssertEqual(
        asking.questionInk(in: palette), HoldCard.titleInk(in: palette),
        "the question is not the plate's ink in \(scheme)")
      XCTAssertEqual(
        asking.consequenceInk(in: scheme), HoldCard.quietInk(in: scheme),
        "the consequence is not the plate's ink in \(scheme)")
      XCTAssertEqual(offer.questionInk(in: palette), HoldCard.titleInk(in: palette))
    }
  }

  // The consequence of the hold's question sits inside the card's padding
  // and starts where the question above it starts: the card negates its own
  // padding so the confirm row's 12 pt lines the question up with the
  // plate, and a line left at 0 started 12 pt outside the plate's padding.
  func testTheHoldConfirmConsequenceStartsUnderTheQuestion() throws {
    XCTAssertEqual(
      HoldCard.bodyInset, 24, "the glyph's 16 pt and the row's 8 pt gap")

    let drawn = try probeList(
      HoldCard(model: try hold(), asking: true), scheme: .dark, width: cardWidth)
    let plate = try XCTUnwrap(
      drawn.first { $0.name == "hold.card" }, "the card drew no plate")
    let line = try XCTUnwrap(
      drawn.first { $0.name == "quiet.text" }, "the consequence drew no line of its own")

    XCTAssertEqual(
      line.origin.x - plate.origin.x, Grid.rowInset + HoldCard.bodyInset,
      "the consequence starts under the question: the plate's 12 pt, the glyph and the gap")
    XCTAssertGreaterThanOrEqual(
      line.origin.x - plate.origin.x, Grid.rowInset, "and never outside the card's padding")
    XCTAssertLessThanOrEqual(
      line.origin.x + line.size.width, plate.origin.x + cardWidth - Grid.rowInset,
      "nor past its other side")
  }

  // `.stuck .row-sub{-webkit-line-clamp:2}`, and the reason rule of scene r2:
  // the cause with its place first, then the state of the world, in 80
  // characters or fewer. Two lines at the card's own width is what 80
  // characters of ordinary words come to, which is measured here rather than
  // assumed.
  func testTheStuckCardReasonFitsTwoLinesUnderEightyCharacters() throws {
    XCTAssertEqual(StuckCard.reasonCeiling, 80)
    XCTAssertEqual(StuckCard.reasonLines, 2)
    XCTAssertEqual(
      StuckCard.reasonColumn, 296,
      "360 less the panel's 8 either side, the card's 12 either side, the glyph and its gap")

    // The two reasons the screens file writes, and one at the ceiling.
    let reasons = [
      "SVG picture at line 41. Nothing was created; the note is unchanged.",
      "1 live watch stopped. Sign in and start it again.",
      "The second tab has no heading. Nothing was written; the document is unchanged.",
    ]
    for reason in reasons {
      XCTAssertTrue(StuckCard.fits(reason), "\(reason.count) characters is past the ceiling")
      XCTAssertLessThanOrEqual(
        lines(reason, role: Typography.caption, width: StuckCard.reasonColumn),
        StuckCard.reasonLines, "“\(reason)” does not fit two lines")
    }
    XCTAssertFalse(
      StuckCard.fits(String(repeating: "x", count: 81)), "81 characters is one too many")

    let measured = try probes(
      StuckCard(model: try stuck(), actions: [note(), branch()], onDismiss: {}),
      scheme: .dark, width: cardWidth)
    XCTAssertNotNil(measured["stuck.card"])
    XCTAssertEqual(
      measured["stuck.reason"]?.width, StuckCard.reasonColumn,
      "the reason spans the card past the glyph, which is `grid-column:2/4`")
    XCTAssertLessThanOrEqual(
      measured["stuck.reason"]?.height ?? 0, 2 * Typography.caption.lineHeight + 1,
      "a longer reason is clamped rather than drawn")
  }

  // A title too long for the column beside the icons goes in whole on two
  // lines, and the second of them uses the card's full width under the
  // icons. Scene e3's export is the one that proved it: on one line it read
  // "Export stoppe… v1", and over two lines of the narrow column it cut the
  // word Proposal out of the middle. The card's whole job is naming the
  // document that stopped.
  func testALongStuckTitleGoesInWholeOnTwoLinesUnderTheIcons() throws {
    XCTAssertEqual(
      StuckCard.titleColumn(actions: 3), 212,
      "the reason's 296 less the 8 pt gap and three 24 pt icons with their two 4 pt gaps")
    XCTAssertEqual(
      StuckCard.titleColumn(actions: 0), StuckCard.reasonColumn,
      "a card with no icon beside its title has the whole width for it")

    let title = "Export stopped · Proposal to the Falcon partners, draft v1"
    let drawn = StuckCard.titleSplit(title, actions: 3)
    XCTAssertEqual(drawn.count, 2, "one line of this column cannot hold it")
    XCTAssertEqual(
      drawn.joined(separator: Sentences.separator), title,
      "nothing of the title was dropped: the break is its own separator")
    XCTAssertEqual(drawn[0], "Export stopped", "what stopped is the first line")
    XCTAssertTrue(
      drawn[1].hasPrefix("Proposal"), "and the document it stopped on is the second")

    // A title with no separator in it is broken between its words instead,
    // and every word of it is still drawn.
    let run = StuckCard.titleSplit(
      "Publishing the quarterly settlement review for Amsterdam", actions: 3)
    XCTAssertEqual(run.count, 2)
    XCTAssertEqual(
      run.joined(separator: " "), "Publishing the quarterly settlement review for Amsterdam")

    // Each line fits the width it is drawn at, so neither is cut: the first
    // in the column beside the icons, the second across the card.
    XCTAssertLessThanOrEqual(
      MiddleTitle.width(drawn[0], role: Typography.titleEmphasis),
      StuckCard.titleColumn(actions: 3))
    XCTAssertLessThanOrEqual(
      MiddleTitle.width(drawn[1], role: Typography.titleEmphasis), StuckCard.reasonColumn)

    // A title that fits the column stays on one line, and the card draws no
    // second one.
    XCTAssertEqual(StuckCard.titleSplit("Publish stopped", actions: 3).count, 1)
    XCTAssertEqual(
      StuckCard.titleSplit("Publish stopped · Chapter 4", actions: 0).count, 1,
      "with no icon beside it the card has the width for the whole title")

    let long = StuckCardModel(
      title: title,
      reason: "The second tab has no heading. Nothing was written; the document is unchanged.")
    let measured = try probes(
      StuckCard(model: long, actions: [note(), branch()], onDismiss: {}),
      scheme: .dark, width: cardWidth)
    XCTAssertEqual(
      measured["stuck.title"]?.width, StuckCard.titleColumn(actions: 3),
      "the first line is drawn in the column the icons leave")
    XCTAssertEqual(
      measured["stuck.title.second"]?.width, StuckCard.reasonColumn,
      "and the second line in the full width under them")

    let short = try probes(
      StuckCard(model: StuckCardModel(title: "Publish stopped", reason: long.reason), actions: []),
      scheme: .dark, width: cardWidth)
    XCTAssertNil(short["stuck.title.second"], "a title that fits draws one line")
  }

  /// A publish that stopped in the render still names the note it was
  /// building.
  ///
  /// The envelope of such a run carries no title: the title is a field of a
  /// document that was never made. Since 2026-10-26 gdoc keeps the note's
  /// cover title on the run and writes it on the end, so the card reads the
  /// name the mockup draws instead of "Publish stopped" alone. Scene r2's
  /// fixture is that end line.
  func testThePublishThatStoppedInTheRenderNamesTheNote() throws {
    let stopped = try SheetPieces.stuck()
    XCTAssertEqual(
      stopped.card.title, "Publish stopped \u{00B7} Chapter 4",
      "the card of a publish that stopped in the render is nameless")
  }

  /// Principle 4's second sentence: a card never carries an icon that would
  /// do nothing.
  ///
  /// A row in a list dims an icon and says why, because the column is the
  /// same on every row of the list. A card is one plate about one write, and
  /// a dead icon on it is what made the card read as buttons again. A publish
  /// that created nothing has no document to open, no link to copy and no
  /// note either: no field of the record names the file it was built from.
  func testACardCarriesNoIconThatWouldDoNothing() throws {
    let stopped = try SheetPieces.stuck()
    XCTAssertEqual(
      stopped.actions.map(\.icon), [.branch, .xmark],
      "scene r2's publish made nothing, so the card is Continue with Claude and the cross")
    for action in stopped.actions {
      if case .off(let reason) = action.state {
        XCTFail("the card carries a dead \(action.icon.rawValue): \(reason)")
      }
    }

    // A write that did name a document carries the four of a row and the
    // cross, each of them live: scene e3's export.
    let named = try XCTUnwrap(
      try SheetPieces.scene("e3").model().now.compactMap { item -> [ActionModel]? in
        if case .stuck(_, _, let actions, _) = item { return actions }
        return nil
      }.first, "e3 drew no stuck card")
    XCTAssertEqual(named.map(\.icon), [.gdocs, .md, .branch, .link, .xmark])
    for action in named {
      if case .off(let reason) = action.state {
        XCTFail("e3's card carries a dead \(action.icon.rawValue): \(reason)")
      }
    }
  }

  /// A press on a live row opens its document, and the row's one control
  /// stops the watch. Neither does the other.
  ///
  /// The two closures the view is built with are read here through a
  /// `PanelActions` that writes down what it was handed, because this
  /// package has no offscreen click: what a test can hold is that the row
  /// and the icon were wired to two different things. Nail's hand check
  /// after the run holds the rest of it.
  ///
  /// Stop keeps everything it had at the third look: its own state, its
  /// ring while the signal is in flight, and no confirm. A click on the row
  /// never stops a watch, which is principle 3.
  func testTheRowOpensAndOnlyStopStops() throws {
    var pressed: [(kind: ActionKind, target: String?)] = []
    let actions = PanelActions { kind, target in pressed.append((kind, target)) }

    let item = try SheetPieces.liveItem()
    guard case .live(_, _, let stop, let pending, _) = item else {
      return XCTFail("scene r4's row is a watch")
    }
    let doc = try XCTUnwrap(item.opens, "scene r4's watch names the document it watches")

    let open = try XCTUnwrap(
      NowPress.open(item, actions: actions), "the row opens the document it names")
    open()
    XCTAssertEqual(pressed.count, 1, "one press, one thing done")
    XCTAssertEqual(pressed.first?.kind, .openDoc, "and it is the kind a document icon sends")
    XCTAssertEqual(pressed.first?.target, doc)

    NowPress.stop(stop, actions: actions)()
    XCTAssertEqual(pressed.count, 2)
    XCTAssertEqual(pressed.last?.kind, .stopLive, "Stop stops, and opens nothing")
    XCTAssertEqual(pressed.last?.target, doc)

    XCTAssertEqual(
      pressed.filter { $0.kind == .openDoc }.count, 1,
      "the control's press is its own and never the row's")
    XCTAssertEqual(NowPress.words, Tips.openDoc, "the title area carries the icon's own words")

    // Stop is what it was: the one icon, the ring while the signal is in
    // flight, and no confirm beside either.
    XCTAssertFalse(pending, "scene r4's watch has no signal in flight")
    XCTAssertEqual(LiveRow.icons(pending: false), [.stop])
    XCTAssertEqual(LiveRow.icons(pending: true), [], "the ring stands where the icon was")
    XCTAssertTrue(LiveRow.showsRing(pending: true))
  }

  /// A press on a document row opens the document it names, and the icons on
  /// the row keep their own presses.
  ///
  /// The row was opened by its first icon until the fifth look
  /// (docs/v2/DECISIONS.md, 2026-10-28): Nail asked for a click on the row,
  /// and one control per action, so the icon went. The kind is `.openRow`
  /// and not `.openDoc`, because a click on a row is not a click on an icon
  /// and the four labels wait for one.
  ///
  /// The press is read here through a `PanelActions` that writes down what
  /// it was handed, because this package has no offscreen click. Nail's hand
  /// check after the run holds the gesture itself.
  func testADocumentRowOpensAndItsIconsKeepTheirOwnPress() throws {
    var pressed: [(kind: ActionKind, target: String?)] = []
    let actions = PanelActions { kind, target in pressed.append((kind, target)) }

    let row = try SheetPieces.documentRows()[0]
    let doc = try XCTUnwrap(row.opens, "scene e6's first row names a document Drive wrote")
    XCTAssertEqual(doc, row.doc)

    let press = try XCTUnwrap(
      DocumentPress.open(row, actions: actions), "the row opens the document it names")
    press()
    XCTAssertEqual(pressed.count, 1, "one press, one thing done")
    XCTAssertEqual(pressed.first?.kind, .openRow, "and it is the row's own kind")
    XCTAssertEqual(pressed.first?.target, doc)

    // The icons are their own presses over the row's: four of them, and the
    // first is the note and never a second way to open the document.
    XCTAssertEqual(row.actions.map(\.kind), [.openNote, .openChat, .copyLink, .pinDoc])
    for action in row.actions {
      actions.perform(action.kind, action.target)
    }
    XCTAssertEqual(
      pressed.filter { $0.kind == .openRow }.count, 1,
      "an icon's press is its own and never the row's")

    // A row naming an id Drive could not have written builds no press at
    // all, which is the rule a NOW row follows: `Opener` refuses to build a
    // URL out of it, and a row that offers a press doing nothing is worse
    // than a row that offers none.
    let planted = DocumentRowModel(
      doc: "not-a-drive-id", title: row.title, sub: row.sub, glyph: row.glyph, tone: row.tone,
      gone: row.gone, isPinned: row.isPinned, opens: nil, actions: row.actions, menu: row.menu)
    XCTAssertNil(
      DocumentPress.open(planted, actions: actions),
      "a row with no document to open carries no press")
  }

  /// The held press of a document row opens the system's own menu, with the
  /// words the model put in it and nothing else.
  ///
  /// It was a SwiftUI popover until the fourth look, which is a window of
  /// its own, and the panel is a borderless `KeyPanel` that closes the
  /// moment it loses key: the menu could have taken the panel it belongs to
  /// with it. The press now hands the words to a presenter, which is an
  /// `NSMenu` popped up from the row's own view in the app and a recorder
  /// here. Opening the menu does nothing to the document, and a word of it
  /// does exactly what the same word of the secondary click's menu does.
  func testTheHeldPressOpensTheSystemMenuWithTheRowsOwnWords() throws {
    var offered: [[String]] = []
    var entries: [MenuEntry] = []
    var pressed: [(kind: ActionKind, target: String?)] = []
    let actions = PanelActions { kind, target in pressed.append((kind, target)) }
    let present = MenuPresenter { words, _ in
      offered.append(words.map(\.word))
      entries = words
    }

    let menu = RowMenu(
      items: [MenuItemModel(.hideDoc, word: "Hide from the list", target: SceneDocuments.chapter4)],
      actions: actions, present: present)
    let press = try XCTUnwrap(menu.held, "a row with a word in its menu has a held press")
    press()
    XCTAssertEqual(offered, [["Hide from the list"]], "the words are the model's")
    XCTAssertTrue(pressed.isEmpty, "opening a menu does nothing to the document")

    try XCTUnwrap(entries.first).press()
    XCTAssertEqual(pressed.count, 1, "one word, one press")
    XCTAssertEqual(pressed.first?.kind, .hideDoc)
    XCTAssertEqual(pressed.first?.target, SceneDocuments.chapter4)

    XCTAssertNil(
      RowMenu(items: [], actions: actions, present: present).held,
      "a pinned row offers nothing, so a press held on it opens nothing")
  }

  /// The gesture is the title's own, and the view the menu is measured from
  /// catches nothing.
  ///
  /// The press sat on the whole row, action strip included, so a slow click
  /// on an icon both did the icon's thing and opened the menu. It is the
  /// title area's now: the four icons are a column of their own and the
  /// gesture is not over them. The anchor is behind that title, and its
  /// `hitTest` is nil, so a click meant for the row or for one of its icons
  /// goes where it was going.
  func testTheHeldPressIsTheTitlesAndTheAnchorCatchesNothing() throws {
    let anchor = MenuAnchorView(frame: NSRect(x: 0, y: 0, width: 160, height: 44))
    XCTAssertNil(
      anchor.hitTest(NSPoint(x: 80, y: 22)), "the anchor is a place to measure from")

    var held = 0
    let row = RowView(
      .doc, glyph: .blank, title: "Ch 4. Pricing", titleHeld: { held += 1 },
      trailing: { EmptyView() })
    try XCTUnwrap(row.titleHeld, "the row carries the press on its title")()
    XCTAssertEqual(held, 1)

    let plain = RowView(.doc, glyph: .blank, title: "Ch 4. Pricing", trailing: { EmptyView() })
    XCTAssertNil(plain.titleHeld, "a row with no menu carries no gesture at all")
    XCTAssertEqual(Timing.menuHold, 0.5, "half a second, which is what the guide promises")
  }

  /// A render draws no anchor, so a row with a menu is the same picture as
  /// the row without one.
  ///
  /// The anchor is an `NSView`, and `ImageRenderer` cannot draw AppKit: it
  /// puts its yellow "cannot draw this" box where the view is. The first
  /// renders of the fourth look had that box over the title of every row
  /// with a menu, so the reviewer could not read the rows the look changed.
  func testARenderDrawsNoMenuAnchor() throws {
    let actions = PanelActions { _, _ in }
    let present = MenuPresenter { _, _ in }
    let items = [
      MenuItemModel(.hideDoc, word: "Hide from the list", target: SceneDocuments.chapter4)
    ]
    func row() -> some View {
      RowView(.doc, glyph: .blank, title: "Ch 4. Pricing", trailing: { EmptyView() })
        .frame(width: Grid.panelWidth, height: 44)
    }
    for scheme in [ColorScheme.dark, .light] {
      let withMenu = try png(
        row().rowMenu(RowMenu(items: items, actions: actions, present: present)).rendering(),
        scheme: scheme)
      let without = try png(row().rendering(), scheme: scheme)
      XCTAssertEqual(withMenu, without, "a render of a row with a menu draws no anchor, \(scheme)")
    }
  }

  // `.stuck .acts` holds Open md, Continue with Claude and the bare cross,
  // in that order: the one that closes the card is the last, so a pointer
  // moving right never crosses it on the way to something else. Scene e9's
  // card has one action and no cross at all, because a sign-in that expired
  // is not something to dismiss.
  func testTheStuckCardDismissIsTheLastIcon() throws {
    XCTAssertEqual(StuckCard.dismissTip, "Dismiss")
    XCTAssertEqual(
      StuckCard.icons([note(), branch()], dismissable: true), [.md, .branch, .xmark])
    XCTAssertEqual(StuckCard.icons([note(), branch()], dismissable: false), [.md, .branch])
    XCTAssertEqual(
      StuckCard.icons([IconAction(.signin, tip: "Sign in with Google")], dismissable: false),
      [.signin], "scene e9: one action, and nothing to dismiss")

    var dismissed = 0
    let card = StuckCard(
      model: try stuck(), actions: [note(), branch()], onDismiss: { dismissed += 1 })
    XCTAssertEqual(card.actions.map(\.icon), [.md, .branch, .xmark])
    card.actions.last?.perform()
    XCTAssertEqual(dismissed, 1, "the last icon is the one that closes the card")
  }

  // `LIVE()` in the screens file: one action, Stop watching. While the
  // SIGINT is in flight the icon becomes the small ring, because a stop that
  // can be pressed twice is a stop nobody trusts; the end arriving takes the
  // row away.
  func testTheLiveRowHasOneStopIconThatBecomesARingWhilePending() throws {
    XCTAssertEqual(LiveRow.stopTip, "Stop watching")
    XCTAssertEqual(LiveRow.icons(pending: false), [.stop])
    XCTAssertEqual(LiveRow.icons(pending: true), [], "the icon is gone while it is pending")
    XCTAssertFalse(LiveRow.showsRing(pending: false))
    XCTAssertTrue(LiveRow.showsRing(pending: true))

    let row = LiveRow(model: try live())
    let measured = try probes(row, scheme: .dark, width: cardWidth)
    XCTAssertEqual(measured["row.two"]?.height, 44)
    XCTAssertEqual(measured["live.stop"]?.height, Grid.actionSize)
    XCTAssertNil(measured["live.ring"])

    let pending = try probes(
      LiveRow(model: try live(), pending: true), scheme: .dark, width: cardWidth)
    XCTAssertNil(pending["live.stop"])
    XCTAssertEqual(pending["live.ring"], CGSize(width: Grid.glyphSize, height: Grid.glyphSize))

    // Off with its own words, which is the check in Task 18 refusing: the
    // row still draws one icon and never two.
    let off = LiveRow(model: try live(), stop: .off(Tips.cannotStop))
    XCTAssertEqual(off.state, .off(Tips.cannotStop))
    XCTAssertEqual(LiveRow.icons(pending: false).count, 1)
  }

  // `.now-job` is the large ring, the title and one quiet line. The step
  // rows under it are the record's `step` events, which M24b adds: with none
  // of them the card is the ring and the two lines, and never four grey rows
  // the app invented.
  func testTheJobCardShowsStepRowsOnlyWhenStepsAreKnown() throws {
    let blind = JobCardModel(name: "Publishing", sub: nil, progress: nil, steps: [])
    XCTAssertFalse(JobCard.showsSteps(blind))
    let measured = try probes(JobCard(model: blind), scheme: .dark, width: cardWidth)
    XCTAssertNotNil(measured["job.card"])
    XCTAssertEqual(measured["job.ring"], CGSize(width: Grid.ringLarge, height: Grid.ringLarge))
    XCTAssertTrue(
      measured.keys.filter { $0.hasPrefix("job.step.") }.isEmpty,
      "nothing is known about the steps, so the card draws none")

    // Scene r1, whose record names the second of the four parts as the one
    // that began last: one done, one running, two that have not happened.
    let known = try job()
    XCTAssertTrue(JobCard.showsSteps(known))
    let drawn = try probes(JobCard(model: known), scheme: .dark, width: cardWidth)
    XCTAssertEqual(drawn.keys.filter { $0.hasPrefix("job.step.") }.count, 4)
    XCTAssertEqual(drawn["job.step.0"]?.height, Grid.compactRowHeight)
    XCTAssertEqual(JobCard.ringProgress(known), 0.5, "the card's own fraction, not a step's")
    XCTAssertNil(JobCard.ringProgress(blind), "an indeterminate ring, which turns")
  }

  // `.empty` is the first open of a machine that never signed in: the mark
  // at 28 pt and one sentence that names the icon to press. Nothing else in
  // the panel is centred.
  func testTheEmptyStateIsTheFirstRunText() throws {
    XCTAssertEqual(
      EmptyState.firstRun,
      "Not signed in. Sign in with the Google account that will post replies: "
        + "the Sign in icon at the top.")
    XCTAssertEqual(Typography.markLarge.size, 28, "`.empty .mark{font-size:28px}`")
    XCTAssertEqual(EmptyState.topPadding, 28)
    XCTAssertEqual(EmptyState.bottomPadding, 20)
    XCTAssertEqual(EmptyState.markGap, 8)

    for scheme in [ColorScheme.dark, .light] {
      let measured = try probes(
        EmptyState(mark: "⁘", words: EmptyState.firstRun), scheme: scheme, width: cardWidth)
      XCTAssertNotNil(measured["empty"])
      XCTAssertNotNil(measured["empty.mark"])
      XCTAssertGreaterThan(
        try inkedPixels(
          EmptyState(mark: "⁘", words: EmptyState.firstRun), scheme: scheme,
          side: Grid.panelWidth), 500, "the empty state drew nothing in \(scheme)")
    }
  }

  // Scene e8: the address at 15 pt for five seconds after a sign-in, with
  // "Not you?" beside it, because one glance catching a personal account is
  // cheaper than a reply carrying its name into a document.
  func testTheBannerLivesFiveSeconds() throws {
    XCTAssertEqual(Timing.checkFlash, 5)
    XCTAssertEqual(SignedInBanner.life, Timing.checkFlash)
    XCTAssertEqual(
      SignedInBanner.words, "Signed in just now. Not you? Sign out and sign in again.")
    XCTAssertEqual(SignedInBanner.signOutTip, "Sign out")
    XCTAssertEqual(Typography.bigMail.size, 15, "`.bigmail{font:500 15px/20px}`")

    let signedIn = Date(timeIntervalSince1970: 1_000)
    XCTAssertTrue(SignedInBanner.isShowing(signedInAt: signedIn, now: signedIn))
    XCTAssertTrue(
      SignedInBanner.isShowing(signedInAt: signedIn, now: signedIn.addingTimeInterval(4.99)))
    XCTAssertFalse(
      SignedInBanner.isShowing(signedInAt: signedIn, now: signedIn.addingTimeInterval(5)))
    XCTAssertFalse(
      SignedInBanner.isShowing(signedInAt: nil, now: signedIn), "nobody signed in")

    let measured = try probes(
      SignedInBanner(account: "nail@example.com"), scheme: .dark, width: cardWidth)
    XCTAssertGreaterThanOrEqual(
      measured["banner"]?.height ?? 0, SignedInBanner.minHeight,
      "`min-height:48px`: the address is a line taller than a title")
    XCTAssertLessThanOrEqual(
      measured["banner"]?.height ?? 0, SignedInBanner.minHeight + Typography.caption.lineHeight,
      "and the sentence under it wraps to two lines at most")
    XCTAssertEqual(measured["banner.sign-out"]?.height, Grid.actionSize)
  }

  // `.strip` is shown once, under the first document row, and says what the
  // four icons above it are. It is the whole of the onboarding: one person
  // does not need a tour, he needs to not guess once. Doc left it at the
  // fifth look with the icon it named: the row is that press now.
  func testTheStripLabelsAreNoteChatLinkPin() throws {
    XCTAssertEqual(StripLabel.allCases.map(\.rawValue), ["Note", "Chat", "Link", "Pin"])
    XCTAssertEqual(StripLabel.allCases.count, 4, "one label per icon of the document row")
    XCTAssertEqual(IconStrip.cell, 24, "`.strip span{width:24px}`")
    XCTAssertEqual(IconStrip.gap, Grid.actionGap)
    XCTAssertEqual(IconStrip.leadingInset, Grid.quietInset, "`padding-left:44px`")
    XCTAssertEqual(Typography.strip.size, 10)

    let measured = try probes(IconStrip(), scheme: .dark, width: cardWidth)
    XCTAssertNotNil(measured["strip"])
    for label in StripLabel.allCases {
      XCTAssertEqual(
        measured["strip.\(label.rawValue)"]?.width, IconStrip.cell,
        "\(label.rawValue) sits over its own icon")
    }
  }

  /// Each label is centred on the icon it names, and not 4 pt to its left.
  ///
  /// The cells and the gaps were the icons' own from the first draft, but
  /// the trailing inset was a row's 12 pt where an action strip's is 8: the
  /// strip hangs 4 pt out over the content edge, because an action's 16 pt
  /// of artwork sits in a 24 pt box. So every word was drawn half a letter
  /// left of its icon, on the one element whose whole job is saying which
  /// word belongs to which icon. The third look made it worse by adding a
  /// fifth label: "Pin" under the gap beside the tack was the hint failing
  /// at the icon Nail could not find. The strip is four words now, and the
  /// pin is still the last of them.
  ///
  /// The two are measured in one tree here, because the claim is about one
  /// against the other and two separate renders agree on nothing.
  func testEachLabelIsCentredOnTheIconItNames() throws {
    let row = VStack(alignment: .leading, spacing: 0) {
      HStack(spacing: Grid.actionGap) {
        Spacer(minLength: 0)
        ForEach(StripLabel.allCases, id: \.rawValue) { label in
          ActionButton(.link, tip: label.rawValue).metricProbe("icon.\(label.rawValue)")
        }
      }
      .padding(.trailing, Grid.rowInset - Grid.actionOverhang)
      IconStrip()
    }
    let measured = try probeList(row, scheme: .dark, width: Grid.panelWidth)
    func centre(_ name: String) throws -> CGFloat {
      let probe = try XCTUnwrap(measured.first { $0.name == name }, "nothing reported \(name)")
      return probe.origin.x + probe.size.width / 2
    }

    for label in StripLabel.allCases {
      XCTAssertEqual(
        try centre("strip.\(label.rawValue)"), try centre("icon.\(label.rawValue)"),
        accuracy: 0.01, "\(label.rawValue) is not centred under its own icon")
    }
  }

  // SETTINGS is the same list on every screen, with the rows that have
  // nothing to say left out: network, Claude Desktop, the switch, Quit. A
  // section whose rows move is a section nobody learns.
  //
  // The account and the version are not on that list any more: the second
  // morning moved them into the header, which is where a person looks for
  // whose account a reply would go out under. The Thread mark row went with
  // the same morning, "a pointless thing": the mark is in the header, drawn
  // at the panel's top left, and a row repeating it is the same fact twice.
  // Hub went on the evening of the same day: the note is found rather than
  // configured, so there is no folder to choose.
  func testSettingsRowsKeepTheirOrderOnEveryScreen() throws {
    XCTAssertEqual(PanelSection.settings([]).label, "Settings")
    XCTAssertEqual(SectionLabel.text(PanelSection.settings([]).label), "SETTINGS")
    XCTAssertEqual(
      SettingsRowKind.allCases, [.network, .desktop, .launchAtLogin, .quit])
    XCTAssertEqual(
      SettingsRowKind.allCases.map(\.label),
      ["Network", "Claude Desktop", "Launch at login", "Quit"])
    XCTAssertFalse(
      SettingsRowKind.allCases.map(\.rawValue).contains("mark"),
      "the mark is the header's, and is said in one place")
    XCTAssertEqual(SettingsRows.quitShortcut, "⌘Q")

    // Which rows each screen draws, out of that screen's own model. The
    // rows are the panel's and not a literal this test also sorts: a list
    // built here and put back in order by the same code says nothing about
    // what any screen holds.
    //
    // Network is drawn only where the record says Google did not answer,
    // and Claude Desktop only where no extension has been written, which is
    // why the two move screen to screen and the other two never do.
    let screens: [(String, [SettingsRowKind])] = [
      ("r8", [.network, .launchAtLogin, .quit]),
      ("e5", [.desktop, .launchAtLogin, .quit]),
      ("e8", [.launchAtLogin, .quit]),
      ("e9", [.launchAtLogin, .quit]),
      ("e10", [.launchAtLogin, .quit]),
    ]
    for (screen, expected) in screens {
      let scene = try XCTUnwrap(Scenes.all.first { $0.id == screen }, "\(screen) is not a scene")
      let drawn = try scene.model().settings.map(\.kind)
      XCTAssertEqual(drawn, expected, "SETTINGS on \(screen) is not the rows it should hold")
      XCTAssertEqual(
        SettingsRows.ordered(drawn.map { SettingsRow($0, value: $0.label) }.shuffled()).map(\.kind),
        drawn, "and the view puts them back in that order")
    }

    // The quiet line under a row is that row's, so it never floats to the
    // foot of the section.
    let withQuiet = [
      SettingsRow(.quit, value: SettingsRows.quitShortcut),
      SettingsRow(
        .network, value: "Google not reachable", glyph: .exclamation, tone: .bad,
        quiet: "the internet is up, the handshake with Google fails"),
      SettingsRow(.desktop, value: "not installed", glyph: .empty),
    ]
    let ordered = SettingsRows.ordered(withQuiet)
    XCTAssertEqual(ordered.map(\.kind), [.network, .desktop, .quit])
    XCTAssertEqual(ordered[0].quiet, "the internet is up, the handshake with Google fails")

    let measured = try probes(SettingsRows(rows: withQuiet), scheme: .dark, width: cardWidth)
    XCTAssertEqual(measured["settings.quit"]?.height, Grid.compactRowHeight)
    XCTAssertNotNil(measured["settings.network.quiet"])
    let switched = try probes(
      SettingsRows(rows: [SettingsRow(.launchAtLogin, value: "", isOn: true)]), scheme: .dark,
      width: cardWidth)
    XCTAssertEqual(switched["switch"], CGSize(width: SwitchView.width, height: SwitchView.height))
  }

  // MARK: the render

  /// The sheet: the nine pieces in both appearances, so they are looked at
  /// once rather than trusted from a pixel count.
  ///
  /// Every word on it is a scene's, through `SheetPieces`: the sheet is one
  /// of the two pages with no record behind it, and a string written by
  /// hand here is a string Nail reads as the app's own.
  func testTheCardSheetRenders() throws {
    let shots = try shotsDirectory()
    let hold = try SheetPieces.hold()
    let stuck = try SheetPieces.stuck()
    let live = try SheetPieces.live()
    let job = try SheetPieces.job()
    let settings = try SheetPieces.settings().map { row in
      SettingsRow(
        row.kind, value: row.value, glyph: row.glyph, tone: row.tone, waiting: row.waiting,
        quiet: row.quiet,
        actions: row.actions.map { IconAction($0.icon, tip: $0.tip, state: $0.state) },
        isOn: row.isOn)
    }
    let icons = stuck.actions.map { IconAction($0.icon, tip: $0.tip, state: $0.state) }
    // The scene's own list ends with the cross, so the sheet hands over no
    // second one: a card with two crosses is the sheet inventing a control.
    XCTAssertEqual(
      StuckCard(model: stuck.card, actions: icons).actions.filter { $0.icon == .xmark }.count,
      1, "the sheet drew the dismiss twice")
    // SETTINGS is drawn with every row a fact reaches, which no one screen
    // has. The first-run sentence is not on this page: it names the Sign in
    // icon at the top, and there is no header on this sheet to find it on.
    // Scene e5 is where that sentence belongs, and the index page draws it
    // whole.
    XCTAssertTrue(
      settings.contains { $0.kind == .desktop }, "no scene drew the Claude Desktop row")

    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      let sheet = VStack(alignment: .leading, spacing: Grid.unit) {
        HoldCard(model: hold)
        HoldCard(model: hold, asking: true)
        StuckCard(model: stuck.card, actions: icons)
        LiveRow(model: live)
        LiveRow(model: live, pending: true)
        JobCard(model: job)
        IconStrip()
        URLLine(url: Scenes.signInURL)
        SignedInBanner(account: Scenes.account)
        SettingsRows(rows: settings)
      }
      .frame(width: cardWidth)
      .padding(Grid.unit)
      .background(palette.surface.color)

      let data = try png(sheet, scheme: scheme)
      let name = scheme == .dark ? "cards-dark.png" : "cards-light.png"
      try data.write(to: shots.appendingPathComponent(name))
      XCTAssertGreaterThan(data.count, 2_000, "\(name) is too small to be the sheet")
    }
  }

  // MARK: the fixtures
  //
  // Every value a piece draws is a scene's, through `SheetPieces`: the
  // records under Scenes/ are the one place a string in this package comes
  // from, so a measurement here is taken on the words the panel really
  // draws. The two icons are the panel's own tooltips.

  private func hold() throws -> HoldCardModel {
    try SheetPieces.hold()
  }

  private func stuck() throws -> StuckCardModel {
    try SheetPieces.stuck().card
  }

  private func live() throws -> LiveRowModel {
    try SheetPieces.live()
  }

  private func job() throws -> JobCardModel {
    try SheetPieces.job()
  }

  private func note() -> IconAction {
    IconAction(.md, tip: "Open the note in Obsidian")
  }

  private func branch() -> IconAction {
    IconAction(.branch, tip: "Continue with Claude")
  }

  // MARK: the measurements

  /// One component source by name, read from this file's own path.
  private func componentSource(_ name: String) throws -> String {
    let file =
      presencePackageRoot()
      .appendingPathComponent("Sources/PresenceUI/Components/\(name)")
    return try String(contentsOf: file, encoding: .utf8)
  }
}
