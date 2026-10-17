// The nine pieces the panel's sections are made of, and the words in them.
//
// Every number below is a literal read out of
// docs/design/2026-10-17-presence-screens-v3.html by hand: `.hold`, `.stuck`,
// `.row-live`, `.now-job`, `.empty`, `.bigmail`, `.urlline`, `.strip` and the
// `HROW` rows of HEALTH. The words are literals too, because a sentence a
// person reads in the menu bar is a decision and not a string somebody
// improved.
//
// Three of these tests are about words rather than pixels. The hold card
// never says Release, the strip says four of them, and the health rows keep
// one order on every screen: each is a thing the panel would still draw
// wrongly with every size right.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class CardTests: XCTestCase {
  /// The width a card is drawn at inside the panel: 320 less the panel's own
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
      StuckCard.reasonColumn, 256,
      "320 less the panel's 8 either side, the card's 12 either side, the glyph and its gap")

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
    XCTAssertEqual(StuckCard.titleLines, 2)
    XCTAssertEqual(
      StuckCard.titleColumn(actions: 3), 172,
      "the reason's 256 less the 8 pt gap and three 24 pt icons with their two 4 pt gaps")
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
    let blind = JobCardModel(title: "Publishing", sub: nil, progress: nil, steps: [])
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
  // at 28 pt and one sentence that names the row to press. Nothing else in
  // the panel is centred.
  func testTheEmptyStateIsTheFirstRunText() throws {
    XCTAssertEqual(
      EmptyState.firstRun,
      "Not signed in. Sign in with the Google account that will post replies: "
        + "the Account row below.")
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
  // does not need a tour, he needs to not guess once.
  func testTheStripLabelsAreDocNoteChatLink() throws {
    XCTAssertEqual(StripLabel.allCases.map(\.rawValue), ["Doc", "Note", "Chat", "Link"])
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

  // HEALTH is the same list on every screen, with the rows that have nothing
  // to say left out: account, network, gdoc, Claude Desktop, the thread
  // mark, the switch, Quit. A section whose rows move is a section nobody
  // learns.
  func testHealthRowsKeepTheirOrderOnEveryScreen() throws {
    XCTAssertEqual(
      HealthRowKind.allCases,
      [.account, .network, .gdoc, .desktop, .mark, .launchAtLogin, .quit])
    XCTAssertEqual(
      HealthRowKind.allCases.map(\.label),
      [
        "Account", "Network", "gdoc", "Claude Desktop", "Thread mark",
        "Launch at login", "Quit",
      ])
    XCTAssertEqual(HealthRows.quitShortcut, "⌘Q")

    // Which rows each screen draws, out of that screen's own model. The
    // rows are the panel's and not a literal this test also sorts: a list
    // built here and put back in order by the same code says nothing about
    // what any screen holds.
    //
    // Network is drawn only where the record says Google did not answer,
    // and Claude Desktop only where no extension has been written, which is
    // why the two move screen to screen and the other five never do.
    let screens: [(String, [HealthRowKind])] = [
      ("r8", [.account, .network, .gdoc, .mark, .launchAtLogin, .quit]),
      ("e5", [.account, .gdoc, .desktop, .mark, .launchAtLogin, .quit]),
      ("e8", [.account, .gdoc, .mark, .launchAtLogin, .quit]),
      ("e9", [.account, .gdoc, .mark, .launchAtLogin, .quit]),
      ("e10", [.account, .gdoc, .mark, .launchAtLogin, .quit]),
    ]
    for (screen, expected) in screens {
      let scene = try XCTUnwrap(Scenes.all.first { $0.id == screen }, "\(screen) is not a scene")
      let drawn = try scene.model().health.map(\.kind)
      XCTAssertEqual(drawn, expected, "HEALTH on \(screen) is not the rows it should hold")
      XCTAssertEqual(
        HealthRows.ordered(drawn.map { HealthRow($0, value: $0.label) }.shuffled()).map(\.kind),
        drawn, "and the view puts them back in that order")
    }

    // The quiet line under a row is that row's, so it never floats to the
    // foot of the section.
    let withQuiet = [
      HealthRow(.gdoc, value: "v2.12.0"),
      HealthRow(
        .network, value: "Google not reachable", glyph: .exclamation, tone: .bad,
        quiet: "the internet is up, the handshake with Google fails"),
      HealthRow(.account, value: "nail@example.com", glyph: .check, tone: .ok),
    ]
    let ordered = HealthRows.ordered(withQuiet)
    XCTAssertEqual(ordered.map(\.kind), [.account, .network, .gdoc])
    XCTAssertEqual(ordered[1].quiet, "the internet is up, the handshake with Google fails")

    let measured = try probes(HealthRows(rows: withQuiet), scheme: .dark, width: cardWidth)
    XCTAssertEqual(measured["health.account"]?.height, Grid.compactRowHeight)
    XCTAssertNotNil(measured["health.network.quiet"])
    let switched = try probes(
      HealthRows(rows: [HealthRow(.launchAtLogin, value: "", isOn: true)]), scheme: .dark,
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
    let health = try SheetPieces.health().map { row in
      HealthRow(
        row.kind, value: row.value, glyph: row.glyph, tone: row.tone, waiting: row.waiting,
        quiet: row.quiet,
        actions: row.actions.map { IconAction($0.icon, tip: $0.tip, state: $0.state) },
        isOn: row.isOn)
    }
    let icons = stuck.actions.map { IconAction($0.icon, tip: $0.tip, state: $0.state) }
    let mark = try XCTUnwrap(
      SheetPieces.health().first { $0.kind == .mark }, "no scene drew the mark row"
    ).value

    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      let sheet = VStack(alignment: .leading, spacing: Grid.unit) {
        HoldCard(model: hold)
        HoldCard(model: hold, asking: true)
        StuckCard(model: stuck.card, actions: icons, onDismiss: {})
        LiveRow(model: live)
        LiveRow(model: live, pending: true)
        JobCard(model: job)
        IconStrip()
        URLLine(url: Scenes.signInURL)
        SignedInBanner(account: Scenes.account)
        HealthRows(rows: health)
        EmptyState(mark: mark, words: EmptyState.firstRun)
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
